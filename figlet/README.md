# figlet — renders text as ASCII-art banners using FIGfont

[![Go Reference](https://pkg.go.dev/badge/github.com/malcolmston/chalk/figlet.svg)](https://pkg.go.dev/github.com/malcolmston/chalk/figlet)
[![Parity](https://img.shields.io/badge/parity-100%25-brightgreen)](https://github.com/malcolmston/go/tree/main/parity/chalk/nested/figlet)

Package figlet renders text as ASCII-art banners using FIGfont, a Go port of
the classic figlet program and the Node figlet library. It ships a built-in
block font plus a large registry of bundled variants, and it can load any
standard .flf FIGfont from a file, directory or reader.

Use figlet to draw large banner text for CLI splash screens, section headers
in logs, or generated README art. The simplest entry point is `Render`, which
renders a string with the built-in font; `RenderFont` renders with a named
font from the registry (see `Fonts` and `GetFont`); and a `Font` value
obtained from `LoadFont`, `LoadFontFile` or `ParseFont` can render directly
with `Font.Render`. The companion helpers `RenderRainbow` and `RenderGradient`
(and the lower-level `Rainbow` and `Gradient`) colorize a finished banner with
the sibling chalk package.

## Install

```sh
go get github.com/malcolmston/chalk@v0.5.0
```

```go
import "github.com/malcolmston/chalk/figlet"
```

## Usage

This is the package's own `ExampleFont_Render`, so it compiles and its output is
asserted on every `go test ./figlet/`.

```go
f := figlet.BuiltinFont()
	narrow := f.Render("go chalk", figlet.Options{Width: 30})
	fmt.Println(len(strings.Split(narrow, "\n")) / f.Height())

	wide := f.Render("go chalk")
	fmt.Println(len(strings.Split(wide, "\n")) / f.Height())
```

```
2
1
```

## Exported surface

### Functions

| Function | What it does |
| --- | --- |
| `func Fonts() []string` | Fonts returns the sorted names of all registered fonts. |
| `func Gradient(banner, startHex, endHex string) string` | Gradient colors a rendered banner with a left-to-right color gradient from startHex to endHex (e.g. |
| `func LoadFontDir(dir string) ([]string, error)` | LoadFontDir parses every .flf FIGfont in dir and registers each under its base file name (without extension). |
| `func Rainbow(banner string) string` | Rainbow colors a rendered banner across the full hue spectrum by column. |
| `func Register(name string, f *Font)` | Register adds a named font to the registry, making it available to RenderFont and GetFont. |
| `func Render(text string, opts ...Options) string` | Render renders text with the built-in font. |
| `func RenderFont(name, text string, opts ...Options) (string, error)` | RenderFont renders text with a named registered font. |
| `func RenderGradient(text, startHex, endHex string) string` | RenderGradient renders text with the built-in font and applies a gradient. |
| `func RenderRainbow(text string) string` | RenderRainbow renders text with the built-in font and applies rainbow colors. |

### Types

| Type | What it is |
| --- | --- |
| `FittingRules` | FittingRules reports the layout modes and smushing-rule switches a font's header resolves to, the equivalent of the `fittingRules` object the… |
| `Font` | Font is a parsed FIGfont. |
| `Layout` | Layout selects how adjacent characters are combined, horizontally by Options.Layout and vertically by Options.VerticalLayout. |
| `Metadata` | Metadata reports the FIGfont header fields, matching the object the reference implementation's figlet.metadata returns. |
| `Options` | Options configures a render. |

<details>
<summary><code>Font</code> — constructors and methods</summary>

| Signature | What it does |
| --- | --- |
| `func BuiltinFont() *Font` | BuiltinFont returns the built-in block font. |
| `func FontFromGlyphs(height, layout int, glyphs map[rune][]string) *Font` | FontFromGlyphs builds a *Font from a glyph map of the given row height. |
| `func GetFont(name string) (*Font, bool)` | GetFont returns a registered font by name (case-insensitive). |
| `func LoadFont(r io.Reader) (*Font, error)` | LoadFont parses a FIGfont from a reader. |
| `func LoadFontFile(path string) (*Font, error)` | LoadFontFile parses a FIGfont from a .flf file on disk. |
| `func ParseFont(r io.Reader) (*Font, error)` | ParseFont reads a FIGfont from r. |
| `func (f *Font) FittingRules() FittingRules` | FittingRules returns the font's resolved layout rules. |
| `func (f *Font) Height() int` | Height returns the number of rows in a rendered line. |
| `func (f *Font) Metadata() Metadata` | Metadata returns the font's header fields. |
| `func (f *Font) Render(text string, opts ...Options) string` | Render renders text using the font. |

</details>

### Constants

`LayoutKerning`, `LayoutSmush`, `MaxFontBytes`, `MaxFontHeight`

Full signatures, doc comments and every runnable example are on
[pkg.go.dev](https://pkg.go.dev/github.com/malcolmston/chalk/figlet).

## Measured parity

Compared case-for-case against **`figlet@1.11.4`** by the harness in
[`parity/chalk/nested/figlet`](https://github.com/malcolmston/go/tree/main/parity/chalk/nested/figlet):

| | |
| --- | --- |
| Parity | **100%** |
| Cases | 198 |
| Matching | 191 |
| Mismatching | 0 |
| Declared deviations | 7 |

Regenerate from the aggregator repo with `go test ./parity/chalk/nested/figlet/`.
Declared deviations are documented differences, excluded from the denominator
and listed in the harness report.

## Deviations from upstream

Deliberate differences for this package, where any exist, are recorded in the
module-wide [`API-DEVIATIONS.md`](../API-DEVIATIONS.md).

## License

MIT, as part of [`github.com/malcolmston/chalk`](..).
An independent re-implementation, not affiliated with or endorsed by the
original project.
