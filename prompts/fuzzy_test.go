package prompts

import (
	"bytes"
	"errors"
	"fmt"
	"reflect"
	"strings"
	"sync"
	"testing"

	"github.com/malcolmston/chalk"
)

// names is shorthand for the candidate order a rank produced.
func rankNames(ms []FuzzyMatch) []string {
	out := make([]string, len(ms))
	for i, m := range ms {
		out[i] = m.Candidate
	}
	return out
}

func TestFuzzyRankOrder(t *testing.T) {
	tests := []struct {
		name       string
		query      string
		candidates []string
		want       []string
	}{
		{
			name:       "consecutive run beats a scattered match",
			query:      "ab",
			candidates: []string{"axxxb", "a-x-ab"},
			want:       []string{"a-x-ab", "axxxb"},
		},
		{
			name:       "word boundary beats mid-word",
			query:      "fb",
			candidates: []string{"foobar", "foo bar"},
			want:       []string{"foo bar", "foobar"},
		},
		{
			name:       "start of candidate beats a later position",
			query:      "co",
			candidates: []string{"the config", "config"},
			want:       []string{"config", "the config"},
		},
		{
			name:       "camelCase hump beats a flat word",
			query:      "gp",
			candidates: []string{"getpath", "getPath"},
			want:       []string{"getPath", "getpath"},
		},
		{
			name:       "shorter gap beats a longer one",
			query:      "ac",
			candidates: []string{"a-------c", "a-c"},
			want:       []string{"a-c", "a-------c"},
		},
		{
			name:       "shorter candidate wins a tie",
			query:      "x",
			candidates: []string{"xyz", "xy"},
			want:       []string{"xy", "xyz"},
		},
		{
			name:       "separator kinds all count as boundaries",
			query:      "ab",
			candidates: []string{"xxa_b", "a_b"},
			want:       []string{"a_b", "xxa_b"},
		},
		{
			name:       "empty query returns everything in input order",
			query:      "",
			candidates: []string{"zeta", "alpha", "mu"},
			want:       []string{"zeta", "alpha", "mu"},
		},
		{
			name:       "non-subsequence is dropped",
			query:      "ba",
			candidates: []string{"abc", "bca", "cab"},
			want:       []string{"bca"},
		},
		{
			name:       "no match at all yields nothing",
			query:      "zzz",
			candidates: []string{"abc", "def"},
			want:       []string{},
		},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got := rankNames(FuzzyRank(tc.query, tc.candidates, FuzzyOptions{}))
			if len(got) == 0 && len(tc.want) == 0 {
				return
			}
			if !reflect.DeepEqual(got, tc.want) {
				t.Fatalf("rank(%q) = %v, want %v", tc.query, got, tc.want)
			}
		})
	}
}

func TestFuzzyRankTieBreakByIndex(t *testing.T) {
	// Identical candidates can only be ordered by input index; that key is unique,
	// so the output order is a total order and cannot jitter.
	got := FuzzyRank("ab", []string{"ab", "ab", "ab"}, FuzzyOptions{})
	if len(got) != 3 {
		t.Fatalf("len = %d", len(got))
	}
	for i, m := range got {
		if m.Index != i {
			t.Fatalf("match %d has Index %d", i, m.Index)
		}
	}
}

func TestFuzzyRankDeterministic(t *testing.T) {
	cands := []string{"src/main.go", "src/main_test.go", "cmd/main.go", "MAIN.md", "domain.txt"}
	first := FuzzyRank("main", cands, FuzzyOptions{})
	for i := 0; i < 20; i++ {
		if !reflect.DeepEqual(FuzzyRank("main", cands, FuzzyOptions{}), first) {
			t.Fatalf("rank is not deterministic on run %d", i)
		}
	}
}

func TestFuzzyScorePositions(t *testing.T) {
	tests := []struct {
		query, candidate string
		want             []int
		ok               bool
	}{
		{"fb", "foo bar", []int{0, 4}, true},
		{"ab", "a-x-ab", []int{4, 5}, true}, // the consecutive pair, not the leading a
		{"", "anything", nil, true},
		{"abc", "ab", nil, false},
		{"ba", "abc", nil, false},
		{"go", "Go", []int{0, 1}, true},
	}
	for _, tc := range tests {
		gotScore, gotPos, ok := FuzzyScore(tc.query, tc.candidate, FuzzyOptions{})
		if ok != tc.ok {
			t.Fatalf("FuzzyScore(%q, %q) ok = %v", tc.query, tc.candidate, ok)
		}
		if !ok {
			continue
		}
		if !reflect.DeepEqual(gotPos, tc.want) {
			t.Fatalf("FuzzyScore(%q, %q) positions = %v, want %v", tc.query, tc.candidate, gotPos, tc.want)
		}
		if tc.query != "" && gotScore <= 0 {
			t.Fatalf("FuzzyScore(%q, %q) score = %d, want positive", tc.query, tc.candidate, gotScore)
		}
	}
}

func TestFuzzyScorePositionsAreAscendingAndInRange(t *testing.T) {
	cand := "internal/prompts/fuzzy_test.go"
	_, pos, ok := FuzzyScore("prfz", cand, FuzzyOptions{})
	if !ok {
		t.Fatal("expected a match")
	}
	n := len([]rune(cand))
	for i, p := range pos {
		if p < 0 || p >= n {
			t.Fatalf("position %d out of range: %d", i, p)
		}
		if i > 0 && p <= pos[i-1] {
			t.Fatalf("positions not ascending: %v", pos)
		}
	}
}

func TestFuzzyCaseModes(t *testing.T) {
	tests := []struct {
		name    string
		query   string
		cand    string
		opts    FuzzyOptions
		wantHit bool
	}{
		{"insensitive by default", "AB", "ab", FuzzyOptions{}, true},
		{"smart case: uppercase query is sensitive", "AB", "ab", FuzzyOptions{SmartCase: true}, false},
		{"smart case: lowercase query still folds", "ab", "AB", FuzzyOptions{SmartCase: true}, true},
		{"smart case: uppercase query matches uppercase", "AB", "xxABx", FuzzyOptions{SmartCase: true}, true},
		{"forced sensitive ignores smart case", "ab", "AB", FuzzyOptions{CaseSensitive: true}, false},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			if _, _, ok := FuzzyScore(tc.query, tc.cand, tc.opts); ok != tc.wantHit {
				t.Fatalf("ok = %v, want %v", ok, tc.wantHit)
			}
		})
	}
}

func TestFuzzyHighlightWrapsOnlyMatchedRunes(t *testing.T) {
	// Styling is globally disabled for the rest of the suite (see init in
	// prompts_test.go); pin a level here so the escape codes are observable.
	old := chalk.GetLevel()
	chalk.SetLevel(chalk.LevelBasic)
	defer chalk.SetLevel(old)

	got := highlight("Order", []int{0, 3})
	if strings.Count(got, "\x1b[") == 0 {
		t.Fatalf("no styling emitted: %q", got)
	}
	if chalk.Strip(got) != "Order" {
		t.Fatalf("stripped = %q, want %q", chalk.Strip(got), "Order")
	}
	// Positions 0 and 3 are "O" and "e": each must be wrapped on its own, so the
	// styled runs enclose exactly those two runes.
	styled := chalk.New().Yellow().Bold()
	if want := styled.Sprint("O") + "rd" + styled.Sprint("e") + "r"; got != want {
		t.Fatalf("highlight = %q, want %q", got, want)
	}
	if highlight("abc", nil) != "abc" {
		t.Fatal("empty positions must not style anything")
	}
}

// keys builds a scripted keystroke reader. "\x1b[B" is Down, "\x7f" Backspace,
// "\r" Enter, "\x03" Ctrl-C, "\x1b" Esc, "\t" Tab.
func keys(s string) *strings.Reader { return strings.NewReader(s) }

var fruit = []Choice{{Name: "apple"}, {Name: "banana"}, {Name: "cherry"}}

func TestFuzzySelectsTypedMatch(t *testing.T) {
	out := &bytes.Buffer{}
	idx, err := Fuzzy(FuzzyConfig{
		Message: "Pick",
		Choices: fruit,
		In:      keys("ban\r"),
		Out:     out,
	})
	if err != nil || idx != 1 {
		t.Fatalf("idx = %d err = %v", idx, err)
	}
	if !strings.Contains(out.String(), "banana") {
		t.Fatalf("summary missing the answer: %q", out.String())
	}
}

func TestFuzzyBackspaceWidensTheFilter(t *testing.T) {
	repaint := true
	out := &bytes.Buffer{}
	// Type "chz" (no match), delete "z", then pick: the filter is "ch" again.
	idx, err := Fuzzy(FuzzyConfig{
		Message: "Pick",
		Choices: fruit,
		Repaint: &repaint,
		In:      keys("chz\x7f\r"),
		Out:     out,
	})
	if err != nil || idx != 2 {
		t.Fatalf("idx = %d err = %v", idx, err)
	}
	if !strings.Contains(out.String(), "no match") {
		t.Fatalf("expected an intermediate no-match frame: %q", out.String())
	}
}

func TestFuzzyArrowMovesWithinMatches(t *testing.T) {
	// "an" matches banana only, so widen to "a": apple and banana both match;
	// Down then Enter takes the second.
	idx, err := Fuzzy(FuzzyConfig{
		Message: "Pick",
		Choices: fruit,
		In:      keys("a\x1b[B\r"),
		Out:     &bytes.Buffer{},
	})
	if err != nil {
		t.Fatalf("err = %v", err)
	}
	ranked := FuzzyRank("a", []string{"apple", "banana", "cherry"}, FuzzyOptions{})
	if len(ranked) < 2 {
		t.Fatalf("fixture assumption broken: %v", rankNames(ranked))
	}
	if want := ranked[1].Index; idx != want {
		t.Fatalf("idx = %d, want %d (second match)", idx, want)
	}
}

func TestFuzzyEditingResetsHighlightToBestMatch(t *testing.T) {
	// Move down, then type: the highlight must return to the top match, because
	// the row it pointed at is gone.
	idx, err := Fuzzy(FuzzyConfig{
		Message: "Pick",
		Choices: fruit,
		In:      keys("\x1b[Bch\r"),
		Out:     &bytes.Buffer{},
	})
	if err != nil || idx != 2 {
		t.Fatalf("idx = %d err = %v", idx, err)
	}
}

func TestFuzzyInitialQuery(t *testing.T) {
	idx, err := Fuzzy(FuzzyConfig{
		Message: "Pick",
		Choices: fruit,
		Query:   "che",
		In:      keys("\r"),
		Out:     &bytes.Buffer{},
	})
	if err != nil || idx != 2 {
		t.Fatalf("idx = %d err = %v", idx, err)
	}
}

func TestFuzzyCancel(t *testing.T) {
	for _, script := range []string{"\x03", "\x1b", "ba\x03"} {
		if _, err := Fuzzy(FuzzyConfig{
			Message: "Pick",
			Choices: fruit,
			In:      keys(script),
			Out:     &bytes.Buffer{},
		}); !errors.Is(err, ErrCanceled) {
			t.Fatalf("script %q: err = %v, want ErrCanceled", script, err)
		}
	}
}

func TestFuzzyEnterOnEmptyResultIsIgnored(t *testing.T) {
	// Enter with no match must not resolve; the following backspace + Enter does.
	idx, err := Fuzzy(FuzzyConfig{
		Message: "Pick",
		Choices: fruit,
		In:      keys("chz\r\x7f\r"),
		Out:     &bytes.Buffer{},
	})
	if err != nil || idx != 2 {
		t.Fatalf("idx = %d err = %v", idx, err)
	}
}

func TestFuzzyEOFWithNoMatch(t *testing.T) {
	if _, err := Fuzzy(FuzzyConfig{
		Message: "Pick",
		Choices: fruit,
		In:      keys("zzz"),
		Out:     &bytes.Buffer{},
	}); !errors.Is(err, ErrNoMatch) {
		t.Fatalf("err = %v, want ErrNoMatch", err)
	}
}

func TestFuzzyEOFSelectsHighlighted(t *testing.T) {
	// End of input behaves like Enter, so a piped filter resolves rather than
	// hanging.
	idx, err := Fuzzy(FuzzyConfig{
		Message: "Pick",
		Choices: fruit,
		In:      keys("ban"),
		Out:     &bytes.Buffer{},
	})
	if err != nil || idx != 1 {
		t.Fatalf("idx = %d err = %v", idx, err)
	}
}

func TestFuzzySkipsDisabledChoices(t *testing.T) {
	choices := []Choice{{Name: "apple", Disabled: true}, {Name: "apricot"}}
	idx, err := Fuzzy(FuzzyConfig{
		Message: "Pick",
		Choices: choices,
		In:      keys("ap\r"),
		Out:     &bytes.Buffer{},
	})
	if err != nil || idx != 1 {
		t.Fatalf("idx = %d err = %v", idx, err)
	}
	if _, err := Fuzzy(FuzzyConfig{
		Message: "Pick",
		Choices: []Choice{{Name: "only", Disabled: true}},
		In:      keys("\r"),
		Out:     &bytes.Buffer{},
	}); err == nil {
		t.Fatal("an all-disabled list must be an error")
	}
	if _, err := Fuzzy(FuzzyConfig{Message: "Pick", In: keys("\r"), Out: &bytes.Buffer{}}); err == nil {
		t.Fatal("an empty list must be an error")
	}
}

func TestFuzzyRepaintFrames(t *testing.T) {
	repaint := true
	out := &bytes.Buffer{}
	if _, err := Fuzzy(FuzzyConfig{
		Message: "Pick",
		Choices: fruit,
		Repaint: &repaint,
		In:      keys("b\r"),
		Out:     out,
	}); err != nil {
		t.Fatalf("err = %v", err)
	}
	got := out.String()
	for _, want := range []string{"\x1b[J", "❯ ", "(1/3)", "(3/3)", "> b"} {
		if !strings.Contains(got, want) {
			t.Fatalf("frame output missing %q:\n%q", want, got)
		}
	}
}

func TestFuzzyWindowsLongLists(t *testing.T) {
	repaint := true
	var choices []Choice
	for i := 0; i < 20; i++ {
		choices = append(choices, Choice{Name: fmt.Sprintf("item-%02d", i)})
	}
	out := &bytes.Buffer{}
	if _, err := Fuzzy(FuzzyConfig{
		Message:    "Pick",
		Choices:    choices,
		MaxVisible: 4,
		Repaint:    &repaint,
		In:         keys("\r"),
		Out:        out,
	}); err != nil {
		t.Fatalf("err = %v", err)
	}
	got := out.String()
	if !strings.Contains(got, "↓") {
		t.Fatalf("expected a scroll hint:\n%q", got)
	}
	if strings.Contains(got, "item-19") {
		t.Fatalf("MaxVisible not honored, whole list drawn:\n%q", got)
	}
}

// TestFuzzyNonTerminalDegrades is the important one: a bytes.Buffer is not a
// terminal, so nothing that moves a cursor may reach it.
func TestFuzzyNonTerminalDegrades(t *testing.T) {
	out := &bytes.Buffer{}
	idx, err := Fuzzy(FuzzyConfig{
		Message: "Pick",
		Choices: fruit,
		In:      keys("ban\r"),
		Out:     out, // Repaint left nil: auto-detected as not a terminal
	})
	if err != nil || idx != 1 {
		t.Fatalf("idx = %d err = %v", idx, err)
	}
	got := out.String()
	if strings.Contains(got, "\x1b") {
		t.Fatalf("escape sequence written to a non-terminal: %q", got)
	}
	if strings.Contains(got, "\r") {
		t.Fatalf("carriage return written to a non-terminal: %q", got)
	}
	if want := "? Pick > \n? Pick banana\n"; got != want {
		t.Fatalf("degraded output = %q, want %q", got, want)
	}
}

func TestFuzzyNonTerminalCancelIsPlain(t *testing.T) {
	out := &bytes.Buffer{}
	if _, err := Fuzzy(FuzzyConfig{
		Message: "Pick",
		Choices: fruit,
		In:      keys("\x03"),
		Out:     out,
	}); !errors.Is(err, ErrCanceled) {
		t.Fatalf("err = %v", err)
	}
	if strings.ContainsAny(out.String(), "\x1b\r") {
		t.Fatalf("cancel path wrote control bytes: %q", out.String())
	}
}

func TestFuzzyMulti(t *testing.T) {
	// Filter to "an" (banana), toggle it, widen to "" and toggle cherry as well.
	idxs, chosen, err := FuzzyMulti(FuzzyMultiConfig{
		Message: "Pick many",
		Choices: fruit,
		In:      keys("ban\t\x7f\x7f\x7fch\t\r"),
		Out:     &bytes.Buffer{},
	})
	if err != nil {
		t.Fatalf("err = %v", err)
	}
	if !reflect.DeepEqual(idxs, []int{1, 2}) {
		t.Fatalf("idxs = %v, want [1 2]", idxs)
	}
	if len(chosen) != 2 || chosen[0].Name != "banana" || chosen[1].Name != "cherry" {
		t.Fatalf("chosen = %v", chosen)
	}
}

func TestFuzzyMultiPrechecked(t *testing.T) {
	choices := []Choice{{Name: "apple", Checked: true}, {Name: "banana"}, {Name: "cherry", Disabled: true}}
	idxs, _, err := FuzzyMulti(FuzzyMultiConfig{
		Message: "Pick many",
		Choices: choices,
		In:      keys("\r"),
		Out:     &bytes.Buffer{},
	})
	if err != nil || !reflect.DeepEqual(idxs, []int{0}) {
		t.Fatalf("idxs = %v err = %v", idxs, err)
	}
}

func TestFuzzyMultiEmptyConfirmAndCancel(t *testing.T) {
	idxs, chosen, err := FuzzyMulti(FuzzyMultiConfig{
		Message: "Pick many",
		Choices: fruit,
		In:      keys("\r"),
		Out:     &bytes.Buffer{},
	})
	if err != nil || len(idxs) != 0 || len(chosen) != 0 {
		t.Fatalf("idxs = %v chosen = %v err = %v", idxs, chosen, err)
	}
	if _, _, err := FuzzyMulti(FuzzyMultiConfig{
		Message: "Pick many",
		Choices: fruit,
		In:      keys("\x1b"),
		Out:     &bytes.Buffer{},
	}); !errors.Is(err, ErrCanceled) {
		t.Fatalf("err = %v, want ErrCanceled", err)
	}
	if _, _, err := FuzzyMulti(FuzzyMultiConfig{Message: "x", In: keys("\r"), Out: &bytes.Buffer{}}); err == nil {
		t.Fatal("an empty list must be an error")
	}
}

func TestFuzzyMultiNonTerminalDegrades(t *testing.T) {
	out := &bytes.Buffer{}
	if _, _, err := FuzzyMulti(FuzzyMultiConfig{
		Message: "Pick many",
		Choices: fruit,
		In:      keys("ban\t\r"),
		Out:     out,
	}); err != nil {
		t.Fatalf("err = %v", err)
	}
	if strings.ContainsAny(out.String(), "\x1b\r") {
		t.Fatalf("control bytes in degraded output: %q", out.String())
	}
}

// TestFuzzyConcurrent runs independent prompts from several goroutines. There is
// no animation goroutine here, but the prompts do share package-level styles and
// the matcher must hold no mutable state; -race proves both.
func TestFuzzyConcurrent(t *testing.T) {
	const n = 16
	var wg sync.WaitGroup
	errs := make([]error, n)
	got := make([]int, n)
	for i := 0; i < n; i++ {
		wg.Add(1)
		go func(i int) {
			defer wg.Done()
			repaint := i%2 == 0
			idx, err := Fuzzy(FuzzyConfig{
				Message: fmt.Sprintf("Pick %d", i),
				Choices: fruit,
				Repaint: &repaint,
				In:      keys("ban\r"),
				Out:     &bytes.Buffer{},
			})
			got[i], errs[i] = idx, err
		}(i)
	}
	wg.Wait()
	for i := range errs {
		if errs[i] != nil || got[i] != 1 {
			t.Fatalf("goroutine %d: idx = %d err = %v", i, got[i], errs[i])
		}
	}
}

func TestFuzzyRankLargeInputIsSane(t *testing.T) {
	// A guard against the DP degenerating on a long candidate: this must finish
	// and must find the trailing consecutive run.
	cand := strings.Repeat("ab", 500) + "-zq"
	score, pos, ok := FuzzyScore("zq", cand, FuzzyOptions{})
	if !ok || len(pos) != 2 || pos[1] != pos[0]+1 {
		t.Fatalf("score = %d pos = %v ok = %v", score, pos, ok)
	}
}

func ExampleFuzzy() {
	// Scripted keystrokes stand in for a keyboard: type "kub", press Enter.
	idx, err := Fuzzy(FuzzyConfig{
		Message: "Deploy where?",
		Choices: []Choice{
			{Name: "docker-compose"},
			{Name: "kubernetes-staging"},
			{Name: "kubernetes-production"},
		},
		In:  strings.NewReader("kub\r"),
		Out: new(bytes.Buffer), // not a terminal: no cursor escapes are emitted
	})
	fmt.Println(idx, err)
	// Output: 1 <nil>
}

func ExampleFuzzyRank() {
	for _, m := range FuzzyRank("mn", []string{"main.go", "domain.txt", "man"}, FuzzyOptions{}) {
		fmt.Printf("%-11s score=%3d at=%v\n", m.Candidate, m.Score, m.Positions)
	}
	// Output:
	// man         score= 45 at=[0 2]
	// main.go     score= 44 at=[0 3]
	// domain.txt  score= 24 at=[2 5]
}
