package prompts

import (
	"errors"
	"fmt"
	"io"
	"math"
	"os"
	"sort"
	"strings"
	"unicode"

	"github.com/malcolmston/chalk"
	"golang.org/x/term"
)

// ErrNoMatch is returned by [Fuzzy] when input ends while the filter matches
// nothing, so there is no candidate the prompt could honestly return. It is
// distinct from [ErrCanceled]: the user did not back out, the query simply
// selected the empty set.
var ErrNoMatch = errors.New("prompts: no candidate matched the filter")

// FuzzyOptions tunes the matcher used by [FuzzyScore], [FuzzyRank] and the
// [Fuzzy] prompt.
//
// The zero value means "case-insensitive", which is what a filter line wants
// almost always: the user is typing fast and does not want Shift to change the
// result set. SmartCase is the fzf behaviour and CaseSensitive is the override
// for callers that never want folding.
type FuzzyOptions struct {
	// SmartCase makes the match case-sensitive only when the query itself
	// contains an uppercase rune. A lowercase query keeps matching either case.
	SmartCase bool
	// CaseSensitive forces a case-sensitive match regardless of the query. It
	// wins over SmartCase.
	CaseSensitive bool
}

// caseSensitive resolves the two flags against a concrete query.
func (o FuzzyOptions) caseSensitive(query []rune) bool {
	if o.CaseSensitive {
		return true
	}
	if !o.SmartCase {
		return false
	}
	for _, r := range query {
		if unicode.IsUpper(r) {
			return true
		}
	}
	return false
}

// FuzzyMatch is one candidate that matched a query.
type FuzzyMatch struct {
	// Index is the candidate's position in the slice passed to [FuzzyRank].
	Index int
	// Candidate is the candidate string itself, copied for convenience.
	Candidate string
	// Score is the ranking score; higher is a better match. Scores are only
	// comparable between candidates matched against the same query.
	Score int
	// Positions holds the matched offsets, as *rune* indexes into Candidate,
	// ascending. Rune indexes rather than byte offsets because a renderer walks
	// the candidate rune by rune to decide what to highlight; byte offsets would
	// force every caller to redo that conversion. It is nil for an empty query,
	// which matches everything without highlighting anything.
	Positions []int
}

// Scoring weights. These are absolute numbers rather than ratios because the
// only thing that matters is their order of magnitude relative to each other:
// one matched rune is worth more than any single bonus, a word-boundary hit is
// worth more than a camelCase hump, and a gap costs less than a match earns so
// that a spread-out match still beats no match at all.
const (
	fuzzyScoreMatch       = 16 // per matched rune
	fuzzyBonusBoundary    = 8  // match at the start, or after a separator
	fuzzyBonusCamel       = 6  // lower-to-upper transition inside a word
	fuzzyBonusConsecutive = 8  // match directly after the previous match
	fuzzyBonusFirstMult   = 2  // the first query rune's bonus counts double
	fuzzyGapStart         = -3 // opening a gap of skipped runes
	fuzzyGapExtension     = -1 // each further skipped rune in that gap
)

// fuzzyNegInf is the "unreachable" sentinel for the DP table. Halving MinInt
// keeps the sentinel from overflowing when a penalty is added to it.
const fuzzyNegInf = math.MinInt / 2

// fuzzyIsSeparator reports whether r ends a word, so the rune after it starts a
// new one.
func fuzzyIsSeparator(r rune) bool {
	switch r {
	case ' ', '\t', '/', '\\', '-', '_', '.', ',', ':', ';', '(', ')', '[', ']', '{', '}', '<', '>', '\'', '"', '|', '+', '=', '@', '#':
		return true
	}
	return unicode.IsSpace(r)
}

// fuzzyBonusAt scores the *position* of a match inside cand, independent of the
// query: the start of the string and the first rune of a word are the offsets a
// human means when they type a few letters, so they earn the most.
func fuzzyBonusAt(cand []rune, i int) int {
	if i == 0 {
		return fuzzyBonusBoundary
	}
	prev, cur := cand[i-1], cand[i]
	if fuzzyIsSeparator(prev) {
		return fuzzyBonusBoundary
	}
	if !unicode.IsLetter(prev) && !unicode.IsDigit(prev) {
		return fuzzyBonusBoundary
	}
	if unicode.IsLower(prev) && unicode.IsUpper(cur) {
		return fuzzyBonusCamel
	}
	if unicode.IsDigit(cur) && !unicode.IsDigit(prev) {
		return fuzzyBonusCamel
	}
	return 0
}

// fuzzyLeadPenalty charges for the runes skipped before the first match, so an
// early match outranks a late one in an otherwise identical candidate.
func fuzzyLeadPenalty(i int) int {
	if i <= 0 {
		return 0
	}
	return fuzzyGapStart + fuzzyGapExtension*(i-1)
}

// fuzzyIsSubsequence is the cheap reject: if the query is not a subsequence of
// the candidate at all, the DP below cannot find anything and is not worth
// allocating for. Interactive filtering runs this over every candidate on every
// keystroke, so the common "no match" case has to be cheap.
func fuzzyIsSubsequence(cand, query []rune, sensitive bool) bool {
	j := 0
	for i := 0; i < len(cand) && j < len(query); i++ {
		if fuzzyEq(cand[i], query[j], sensitive) {
			j++
		}
	}
	return j == len(query)
}

func fuzzyEq(a, b rune, sensitive bool) bool {
	if sensitive {
		return a == b
	}
	return unicode.ToLower(a) == unicode.ToLower(b)
}

// fuzzyScoreRunes is the matcher proper: an optimal (not greedy) alignment of
// query against cand, returning the best score and the offsets it matched.
//
// It is a dynamic program over "query rune j matched at candidate rune i". Two
// ways to reach that cell are considered — directly after the previous match
// (earning fuzzyBonusConsecutive) or after a gap (paying fuzzyGapStart plus
// fuzzyGapExtension per extra skipped rune) — and the better one wins. A greedy
// left-to-right scan gets this wrong on the cases users hit first: for "ab"
// against "a-x-ab" greedy locks onto the leading "a" and reports a gap of four,
// where the DP finds the consecutive "ab" at the end.
//
// Ties are resolved toward the *earliest* end position, which is what makes the
// result stable: the same query and candidate always produce the same offsets,
// so a rendered list never jitters between frames.
func fuzzyScoreRunes(cand, query []rune, sensitive bool) (int, []int, bool) {
	n, m := len(cand), len(query)
	if m == 0 {
		return 0, nil, true
	}
	if m > n || !fuzzyIsSubsequence(cand, query, sensitive) {
		return 0, nil, false
	}

	bonus := make([]int, n)
	for i := range cand {
		bonus[i] = fuzzyBonusAt(cand, i)
	}

	// dp[j*n+i] is the best score for aligning query[:j+1] with cand[:i+1] where
	// query[j] lands exactly on cand[i]; parent[j*n+i] is the i of query[j-1].
	dp := make([]int, m*n)
	parent := make([]int, m*n)
	for i := range dp {
		dp[i] = fuzzyNegInf
	}

	for i := 0; i < n; i++ {
		if fuzzyEq(cand[i], query[0], sensitive) {
			dp[i] = fuzzyScoreMatch + fuzzyBonusFirstMult*bonus[i] + fuzzyLeadPenalty(i)
			parent[i] = -1
		}
	}

	for j := 1; j < m; j++ {
		row, prevRow := j*n, (j-1)*n
		// gapBest tracks max over k <= i-2 of dp[prevRow+k] plus the gap penalty
		// for the (i-k-1) runes skipped, carried forward instead of rescanned so
		// the whole matcher stays O(len(cand) * len(query)).
		gapBest, gapArg := fuzzyNegInf, -1
		for i := 0; i < n; i++ {
			if i >= 2 {
				extended, opened := fuzzyNegInf, dp[prevRow+i-2]
				if gapBest > fuzzyNegInf {
					extended = gapBest + fuzzyGapExtension
				}
				if opened > fuzzyNegInf {
					opened += fuzzyGapStart
				}
				// >= would let a later, equally good predecessor replace an
				// earlier one; > keeps the leftmost, which is what makes
				// backtracking deterministic.
				if opened > extended {
					gapBest, gapArg = opened, i-2
				} else {
					gapBest = extended
				}
			}
			if !fuzzyEq(cand[i], query[j], sensitive) {
				continue
			}
			best, bestK := fuzzyNegInf, -1
			if i >= 1 && dp[prevRow+i-1] > fuzzyNegInf {
				b := bonus[i]
				if fuzzyBonusConsecutive > b {
					b = fuzzyBonusConsecutive
				}
				best, bestK = dp[prevRow+i-1]+fuzzyScoreMatch+b, i-1
			}
			if gapBest > fuzzyNegInf {
				if v := gapBest + fuzzyScoreMatch + bonus[i]; v > best {
					best, bestK = v, gapArg
				}
			}
			if bestK >= 0 {
				dp[row+i], parent[row+i] = best, bestK
			}
		}
	}

	last, end := m-1, -1
	score := fuzzyNegInf
	for i := 0; i < n; i++ {
		if dp[last*n+i] > score {
			score, end = dp[last*n+i], i
		}
	}
	if end < 0 {
		return 0, nil, false
	}

	positions := make([]int, m)
	for j := last; j >= 0; j-- {
		positions[j] = end
		end = parent[j*n+end]
	}
	return score, positions, true
}

// FuzzyScore scores one candidate against one query and reports the matched
// rune offsets.
//
// Matching is subsequence matching: every rune of query must appear in
// candidate, in order, but not necessarily adjacently. The score prefers, in
// this order of influence:
//
//   - a match at the start of the candidate, or at the start of a word (after a
//     separator such as space, "-", "_", "/" or "."), or at a camelCase hump;
//     the first query rune's position bonus counts double, because that is the
//     one the user aimed at,
//   - consecutive matched runes over scattered ones,
//   - shorter gaps between matched runes, and earlier matches over later ones.
//
// An empty query matches every candidate with score 0 and no positions. A
// candidate the query is not a subsequence of returns ok == false. The
// alignment is optimal rather than greedy, and ties resolve toward the earliest
// offsets, so repeated calls are byte-for-byte identical — a rendered list must
// not reshuffle just because it was redrawn.
//
// Candidate length is deliberately *not* part of the score: [FuzzyRank] uses it
// as a tie-breaker instead, so that "prefer the shorter candidate" cannot
// outweigh a genuinely better alignment.
func FuzzyScore(query, candidate string, opts FuzzyOptions) (score int, positions []int, ok bool) {
	q := []rune(query)
	return fuzzyScoreRunes([]rune(candidate), q, opts.caseSensitive(q))
}

// FuzzyRank filters candidates by query and returns the matches best first.
//
// Scoring is [FuzzyScore]. Equal scores are broken deterministically, in order:
// the shorter candidate (in runes) first, then the one whose first match is
// earlier, then the lower input index. Because the last key is unique, the
// output order is a total order — the same inputs always produce the same
// slice, which is what keeps an interactive list from jittering.
//
// An empty query returns every candidate, in input order, with score 0 and nil
// Positions. When nothing matches the result is empty (nil), never a partial
// list.
func FuzzyRank(query string, candidates []string, opts FuzzyOptions) []FuzzyMatch {
	q := []rune(query)
	sensitive := opts.caseSensitive(q)
	out := make([]FuzzyMatch, 0, len(candidates))
	for i, c := range candidates {
		score, pos, ok := fuzzyScoreRunes([]rune(c), q, sensitive)
		if !ok {
			continue
		}
		out = append(out, FuzzyMatch{Index: i, Candidate: c, Score: score, Positions: pos})
	}
	if len(q) == 0 {
		// Input order already; sorting would be a no-op with the tie-breakers
		// below, but skipping it documents the guarantee.
		if len(out) == 0 {
			return nil
		}
		return out
	}
	sort.Slice(out, func(a, b int) bool {
		x, y := out[a], out[b]
		if x.Score != y.Score {
			return x.Score > y.Score
		}
		lx, ly := len([]rune(x.Candidate)), len([]rune(y.Candidate))
		if lx != ly {
			return lx < ly
		}
		if len(x.Positions) > 0 && len(y.Positions) > 0 && x.Positions[0] != y.Positions[0] {
			return x.Positions[0] < y.Positions[0]
		}
		return x.Index < y.Index
	})
	if len(out) == 0 {
		return nil
	}
	return out
}

// FuzzyConfig configures Fuzzy.
type FuzzyConfig struct {
	// Message is the question shown to the user.
	Message string
	// Choices are the candidates the filter narrows. Disabled choices are
	// excluded from matching entirely: an entry the user cannot pick has no
	// business occupying a row in a list whose whole point is narrowing.
	Choices []Choice
	// Query is the initial filter text.
	Query string
	// MaxVisible caps how many matches are shown at once, scrolling the list
	// around the highlighted entry. Zero shows every match.
	MaxVisible int
	// Match tunes the matcher (case sensitivity). The zero value is
	// case-insensitive.
	Match FuzzyOptions
	// Repaint controls in-place redrawing with ANSI cursor-movement codes. nil
	// (the default) decides automatically: repaint when Out is a terminal, and
	// degrade to a single prompt line plus a summary when it is not, so a pipe,
	// a log or a file never receives cursor escapes. Set it explicitly to force
	// either mode — tests use *true to assert on rendered frames without a TTY.
	Repaint *bool
	// In is the input source (defaults to os.Stdin).
	In io.Reader
	// Out is the output destination (defaults to os.Stdout).
	Out io.Writer
}

// FuzzyMultiConfig configures FuzzyMulti. The fields mean what they do in
// [FuzzyConfig].
type FuzzyMultiConfig struct {
	Message    string
	Choices    []Choice
	Query      string
	MaxVisible int
	Match      FuzzyOptions
	Repaint    *bool
	In         io.Reader
	Out        io.Writer
}

// Fuzzy-specific styles. Kept here rather than in the shared theme so this file
// owns its own look.
var (
	styleFuzzyMatched = chalk.New().Yellow().Bold() // matched runes inside a candidate
	styleFuzzyCount   = chalk.New().Gray()          // "3/12" match counter
)

// outIsTerminal reports whether out is a real terminal. Anything that is not an
// *os.File (a bytes.Buffer in a test, a pipe wrapper) is by definition not one.
func outIsTerminal(out io.Writer) bool {
	f, ok := out.(*os.File)
	return ok && term.IsTerminal(int(f.Fd()))
}

// fuzzyRenderer writes frames, or refuses to. In repaint mode it clears and
// redraws in place; otherwise it swallows intermediate frames entirely and only
// emits the opening line and the final summary, with LF line endings, because
// CR and cursor movement are meaningless in a file.
type fuzzyRenderer struct {
	out     io.Writer
	repaint bool
	lines   int
}

func newFuzzyRenderer(out io.Writer, repaint *bool, header string) *fuzzyRenderer {
	on := outIsTerminal(out)
	if repaint != nil {
		on = *repaint
	}
	r := &fuzzyRenderer{out: out, repaint: on}
	if !on {
		writeString(out, plainLines(header))
	}
	return r
}

// plainLines converts a frame's CRLF endings to LF for non-terminal output.
func plainLines(s string) string { return strings.ReplaceAll(s, "\r\n", "\n") }

func (r *fuzzyRenderer) frame(content string) {
	if !r.repaint {
		return
	}
	r.lines = renderFrame(r.out, r.lines, content)
}

// finish writes the closing summary and leaves the cursor on a fresh line.
func (r *fuzzyRenderer) finish(summary string) {
	if r.repaint {
		renderFrame(r.out, r.lines, summary)
		return
	}
	writeString(r.out, plainLines(summary))
}

// cancel reports cancellation in whichever mode is active.
func (r *fuzzyRenderer) cancel() {
	if r.repaint {
		writeString(r.out, "\r\n")
		return
	}
	writeString(r.out, "\n")
}

// fuzzyCandidates returns the selectable choices as strings plus their indexes
// in the caller's slice, so a match can be mapped back to what was passed in.
func fuzzyCandidates(choices []Choice) (names []string, origin []int) {
	for i, c := range choices {
		if c.Disabled {
			continue
		}
		names = append(names, c.Name)
		origin = append(origin, i)
	}
	return names, origin
}

// highlight renders a candidate with its matched runes styled.
func highlight(s string, positions []int) string {
	if len(positions) == 0 {
		return s
	}
	set := make(map[int]bool, len(positions))
	for _, p := range positions {
		set[p] = true
	}
	var b strings.Builder
	for i, r := range []rune(s) {
		if set[i] {
			b.WriteString(styleFuzzyMatched.Sprint(string(r)))
			continue
		}
		b.WriteRune(r)
	}
	return b.String()
}

// fuzzyHeader is the "? message > query" line, also the whole of the degraded
// non-terminal output.
func fuzzyHeader(message, query string) string {
	return stylePrefix.Sprint("?") + " " + styleMessage.Sprint(message) + " " +
		styleDim.Sprint(">") + " " + query + "\r\n"
}

// fuzzyFrame draws the filter line, the match counter and the visible window of
// matches.
func fuzzyFrame(message, query string, matches []FuzzyMatch, total, cur, maxVisible int, checked map[int]bool, hint string) string {
	var b strings.Builder
	b.WriteString(stylePrefix.Sprint("?") + " " + styleMessage.Sprint(message) + " " +
		styleDim.Sprint(">") + " " + query + " " +
		styleFuzzyCount.Sprint(fmt.Sprintf("(%d/%d)", len(matches), total)) + " " +
		styleHelp.Sprint(hint) + "\r\n")
	if len(matches) == 0 {
		b.WriteString(styleDim.Sprint("  no match") + "\r\n")
		return b.String()
	}
	start, end, more, less := visibleWindow(cur, len(matches), maxVisible)
	if more {
		b.WriteString(overflowHint("↑"))
	}
	for i := start; i < end; i++ {
		m := matches[i]
		pointer := "  "
		if i == cur {
			pointer = stylePointer.Sprint("❯ ")
		}
		box := ""
		if checked != nil {
			box = "◯ "
			if checked[m.Index] {
				box = styleCheckOn.Sprint("◉ ")
			}
		}
		// The highlight of matched runes is applied whether or not the row is the
		// current one: nesting the "selected" style around it would have to be
		// re-opened after every inner reset, and the pointer already says which
		// row is current.
		b.WriteString(pointer + box + highlight(m.Candidate, m.Positions) + "\r\n")
	}
	if less {
		b.WriteString(overflowHint("↓"))
	}
	return b.String()
}

// Fuzzy presents an fzf-style fuzzy finder: the user types a filter, the
// candidate list narrows live, and the runes that matched are highlighted
// inside each candidate. It returns the index of the chosen entry in
// cfg.Choices.
//
// Keys: printable runes and space extend the filter, Backspace shortens it,
// Up/Down (or Tab) move the highlight, Enter picks the highlighted match, and
// Ctrl-C or Esc returns [ErrCanceled]. Editing the filter resets the highlight
// to the best match, which is the fzf behaviour and the only one that makes
// sense when the list under the cursor has just been replaced. Reaching end of
// input behaves like Enter, so a scripted or piped filter resolves instead of
// hanging; if nothing matches at that point the result is [ErrNoMatch].
//
// Ranking and highlighting come from [FuzzyRank], so they can be tested as a
// pure function. Matching is case-insensitive unless cfg.Match says otherwise.
//
// When cfg.Out is not a terminal the prompt does not repaint: it writes the
// filter line once and the answer summary at the end, with LF endings and no
// cursor-movement escapes, which keeps a piped run or a CI log readable. See
// FuzzyConfig.Repaint to force either mode.
func Fuzzy(cfg FuzzyConfig) (int, error) {
	in, out := resolveIO(cfg.In, cfg.Out)
	names, origin := fuzzyCandidates(cfg.Choices)
	if len(names) == 0 {
		return -1, fmt.Errorf("prompts: Fuzzy requires at least one choice that is not disabled")
	}
	restore := enterRaw(in)
	defer restore()
	kr := newKeyReader(in)

	query := []rune(cfg.Query)
	matches := FuzzyRank(string(query), names, cfg.Match)
	cur := 0
	const hint = "(type to filter, ↑/↓ to move, enter to select)"

	r := newFuzzyRenderer(out, cfg.Repaint, fuzzyHeader(cfg.Message, string(query)))
	r.frame(fuzzyFrame(cfg.Message, string(query), matches, len(names), cur, cfg.MaxVisible, nil, hint))

	for {
		k := kr.read()
		switch k.typ {
		case keyUp, keyLeft:
			if len(matches) > 0 {
				cur = (cur - 1 + len(matches)) % len(matches)
			}
		case keyDown, keyRight, keyTab:
			if len(matches) > 0 {
				cur = (cur + 1) % len(matches)
			}
		case keyRune, keySpace:
			query = append(query, k.r)
			matches, cur = FuzzyRank(string(query), names, cfg.Match), 0
		case keyBackspace:
			if len(query) > 0 {
				query = query[:len(query)-1]
				matches, cur = FuzzyRank(string(query), names, cfg.Match), 0
			}
		case keyEnter, keyEOF:
			if len(matches) == 0 {
				if k.typ == keyEOF {
					r.cancel()
					return -1, ErrNoMatch
				}
				break // Enter on an empty result set is a no-op: nothing to pick.
			}
			idx := origin[matches[cur].Index]
			r.finish(stylePrefix.Sprint("?") + " " + styleMessage.Sprint(cfg.Message) + " " +
				styleAnswer.Sprint(cfg.Choices[idx].Name) + "\r\n")
			return idx, nil
		case keyCtrlC, keyEsc:
			r.cancel()
			return -1, ErrCanceled
		}
		r.frame(fuzzyFrame(cfg.Message, string(query), matches, len(names), cur, cfg.MaxVisible, nil, hint))
	}
}

// FuzzyMulti is [Fuzzy] with checkboxes: Tab toggles the highlighted match and
// Enter confirms the whole selection. Tab is the toggle rather than Space
// because Space is a filter character here — an fzf-style prompt cannot spend it
// on navigation.
//
// It returns the chosen indexes into cfg.Choices, ascending, and the matching
// choices. Toggled entries stay selected while the filter changes, so the user
// can narrow, pick, re-narrow and pick again. Confirming with nothing selected
// returns an empty selection and a nil error; Ctrl-C or Esc returns
// [ErrCanceled]. Non-terminal output degrades exactly as [Fuzzy] describes.
func FuzzyMulti(cfg FuzzyMultiConfig) ([]int, []Choice, error) {
	in, out := resolveIO(cfg.In, cfg.Out)
	names, origin := fuzzyCandidates(cfg.Choices)
	if len(names) == 0 {
		return nil, nil, fmt.Errorf("prompts: FuzzyMulti requires at least one choice that is not disabled")
	}
	restore := enterRaw(in)
	defer restore()
	kr := newKeyReader(in)

	// checked is keyed by candidate index (the position within names), which is
	// what a FuzzyMatch reports, and is stable across re-filtering.
	checked := make(map[int]bool, len(names))
	for i, o := range origin {
		checked[i] = cfg.Choices[o].Checked
	}
	query := []rune(cfg.Query)
	matches := FuzzyRank(string(query), names, cfg.Match)
	cur := 0
	const hint = "(type to filter, tab to toggle, enter to confirm)"

	r := newFuzzyRenderer(out, cfg.Repaint, fuzzyHeader(cfg.Message, string(query)))
	r.frame(fuzzyFrame(cfg.Message, string(query), matches, len(names), cur, cfg.MaxVisible, checked, hint))

	for {
		k := kr.read()
		switch k.typ {
		case keyUp, keyLeft:
			if len(matches) > 0 {
				cur = (cur - 1 + len(matches)) % len(matches)
			}
		case keyDown, keyRight:
			if len(matches) > 0 {
				cur = (cur + 1) % len(matches)
			}
		case keyTab:
			if len(matches) > 0 {
				ci := matches[cur].Index
				checked[ci] = !checked[ci]
			}
		case keyRune, keySpace:
			query = append(query, k.r)
			matches, cur = FuzzyRank(string(query), names, cfg.Match), 0
		case keyBackspace:
			if len(query) > 0 {
				query = query[:len(query)-1]
				matches, cur = FuzzyRank(string(query), names, cfg.Match), 0
			}
		case keyEnter, keyEOF:
			var idxs []int
			var chosen []Choice
			var labels []string
			for ci := range names {
				if !checked[ci] {
					continue
				}
				o := origin[ci]
				idxs = append(idxs, o)
				chosen = append(chosen, cfg.Choices[o])
				labels = append(labels, cfg.Choices[o].Name)
			}
			r.finish(stylePrefix.Sprint("?") + " " + styleMessage.Sprint(cfg.Message) + " " +
				styleAnswer.Sprint(strings.Join(labels, ", ")) + "\r\n")
			return idxs, chosen, nil
		case keyCtrlC, keyEsc:
			r.cancel()
			return nil, nil, ErrCanceled
		}
		r.frame(fuzzyFrame(cfg.Message, string(query), matches, len(names), cur, cfg.MaxVisible, checked, hint))
	}
}
