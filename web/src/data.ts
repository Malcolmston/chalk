// Library content for the chalk documentation site. LIBS[0] (chalk) drives the
// landing page and is kept in sync with the malcolmston/go landing; the trailing
// entries document the sibling subpackages shipped in the same module —
// chalk/prompts and chalk/figlet.
export interface Lib {
  id: string; name: string; icon: string; accent: string; pkg: string; node: string;
  repo: string; docs: string; tagline: string; blurb: string; tags: string[];
  features: string[]; node_code: string; go_code: string; integrate: string;
}
export const NODE_ACCENT = '#8cc84b';
export const LIBS: Lib[] = [
  {
    id:"chalk", name:"chalk", icon:'<i class="fa-solid fa-palette"></i>', accent:"#ffa657",
    pkg:"github.com/malcolmston/chalk", node:"chalk",
    repo:"https://github.com/malcolmston/chalk", docs:"https://malcolmston.github.io/chalk/",
    tagline:"Terminal string styling done right — chainable, immutable ANSI colour.",
    blurb:"A Go port of Node's chalk: expressive, chainable, immutable terminal styling across 16 / 256 / "+
      "truecolor, degrading automatically to whatever the terminal supports. Capability detection honours "+
      "NO_COLOR, FORCE_COLOR, COLORTERM and TERM. The same module also ships chalk/prompts "+
      "(inquirer-style interactive prompts) and chalk/figlet (FIGfont ASCII-art banners).",
    tags:["ANSI colors","truecolor","256-color","NO_COLOR","FORCE_COLOR","chainable","terminal","chalk-compatible"],
    features:[
      "Chainable, immutable styles: <code>chalk.New().Red().Bold().Underline().Sprint(\"x\")</code>",
      "Modifiers: <code>Bold</code>, <code>Dim</code>, <code>Italic</code>, <code>Underline</code>, <code>Inverse</code>, <code>Hidden</code>, <code>Strikethrough</code>, <code>Overline</code>",
      "16 colors with bright variants and backgrounds: <code>Red</code> … <code>BrightCyan</code>, <code>BgBlue</code> … <code>BgBrightWhite</code>",
      "256-color &amp; truecolor: <code>Ansi256</code>, <code>RGB</code>, <code>Hex</code> (plus <code>Bg</code> forms), auto-degrading to the detected level",
      "Capability detection honouring <code>NO_COLOR</code>, <code>FORCE_COLOR</code>, <code>COLORTERM</code> and <code>TERM</code>",
      "Manual level control: <code>SetLevel</code>, <code>GetLevel</code>, <code>Enabled</code>, <code>SetEnabled</code>, <code>ResetDetection</code>",
      "Package-level shortcuts: <code>chalk.Red(\"err\")</code>, <code>chalk.Hex(\"#ff8800\", \"x\")</code>",
      "ANSI-aware helpers: <code>chalk.Strip</code> and <code>chalk.VisibleLength</code>",
      "Also in the module: <b>chalk/prompts</b> (interactive prompts) and <b>chalk/figlet</b> (ASCII-art banners)"
    ],
    node_code:
`const chalk = require('chalk')

console.log(chalk.red.bold('error!'))
console.log(chalk.green('ok'))
console.log(chalk.hex('#ff8800').underline('orange'))`,
    go_code:
`import "github.com/malcolmston/chalk"

fmt.Println(chalk.New().Red().Bold().Sprint("error!"))
fmt.Println(chalk.Green("ok"))
fmt.Println(chalk.New().Hex("#ff8800").Underline().Sprint("orange"))`,
    integrate:
`<span class="tok-c">// Styles are immutable — build once, reuse everywhere</span>
warn := chalk.New().Yellow().Bold()
fmt.Println(warn.Sprint("careful!"))

<span class="tok-c">// Truecolor foreground over a background, auto-degrading</span>
fmt.Println(chalk.New().Hex("#ff8800").BgBlack().Sprint("orange on black"))

<span class="tok-c">// Pin a level, then measure visible width (ANSI stripped)</span>
chalk.SetLevel(chalk.LevelTrueColor)
fmt.Println(chalk.VisibleLength(warn.Sprint("careful!")))   <span class="tok-c">// 8</span>`
  },
  {
    id:"prompts", name:"chalk/prompts", icon:'<i class="fa-solid fa-terminal"></i>', accent:"#7ee787",
    pkg:"github.com/malcolmston/chalk/prompts", node:"inquirer",
    repo:"https://github.com/malcolmston/chalk", docs:"https://malcolmston.github.io/chalk/",
    tagline:"Interactive terminal prompts — inquirer, ported to Go.",
    blurb:"Interactive command-line prompts in the spirit of Node's inquirer / @inquirer/prompts, styled with "+
      "chalk. Offers Input, Password, Confirm, Number, Select and MultiSelect, with raw-mode TTY key handling, "+
      "validation and transform hooks, and clean cancellation via ErrCanceled.",
    tags:["prompts","inquirer","interactive","CLI","raw-mode","TTY","select","validation"],
    features:[
      "<code>Input</code> — line editing with <code>Default</code>, <code>Validate</code> and <code>Transform</code> hooks",
      "<code>Password</code> — hidden or masked entry via the <code>Mask</code> rune",
      "<code>Confirm</code> — yes/no with a default (<code>Y/n</code>)",
      "<code>Number</code> — parsed floats with <code>Min</code> / <code>Max</code> / <code>Integer</code> bounds",
      "<code>Select</code> — arrow-key single choice, returns the index &amp; <code>Choice</code>",
      "<code>MultiSelect</code> — space-to-toggle checkboxes, returns <code>[]int</code> &amp; <code>[]Choice</code>",
      "<code>Choice</code> options: <code>Name</code>, <code>Value</code>, <code>Disabled</code>, <code>Checked</code>",
      "Raw-mode TTY input via <code>golang.org/x/term</code>; scriptable through <code>In</code> / <code>Out</code>",
      "Cancellation with <code>ErrCanceled</code> on Ctrl-C / Esc; answers echoed with chalk styling"
    ],
    node_code:
`const { input, confirm, select } = require('@inquirer/prompts')

const name  = await input({ message: 'Your name?' })
const ok    = await confirm({ message: 'Proceed?' })
const color = await select({
  message: 'Pick one',
  choices: [{ name: 'Red' }, { name: 'Green' }, { name: 'Blue' }],
})`,
    go_code:
`import "github.com/malcolmston/chalk/prompts"

name, _ := prompts.Input(prompts.InputConfig{Message: "Your name?"})
ok, _ := prompts.Confirm(prompts.ConfirmConfig{Message: "Proceed?"})
i, color, _ := prompts.Select(prompts.SelectConfig{
    Message: "Pick one",
    Choices: []prompts.Choice{{Name: "Red"}, {Name: "Green"}, {Name: "Blue"}},
})`,
    integrate:
`<span class="tok-c">// Validate input and mask a secret</span>
email, _ := prompts.Input(prompts.InputConfig{
    Message:  "Email?",
    Validate: func(s string) error {
        if !strings.Contains(s, "@") {
            return errors.New("invalid email")
        }
        return nil
    },
})
pw, _ := prompts.Password(prompts.PasswordConfig{Message: "Password?", Mask: '*'})

<span class="tok-c">// MultiSelect returns the chosen indexes and choices</span>
idx, chosen, err := prompts.MultiSelect(prompts.MultiSelectConfig{
    Message: "Toppings",
    Choices: []prompts.Choice{{Name: "Cheese", Checked: true}, {Name: "Basil"}},
})
if errors.Is(err, prompts.ErrCanceled) {
    return
}`
  },
  {
    id:"figlet", name:"chalk/figlet", icon:'<i class="fa-solid fa-font"></i>', accent:"#d2a8ff",
    pkg:"github.com/malcolmston/chalk/figlet", node:"figlet",
    repo:"https://github.com/malcolmston/chalk", docs:"https://malcolmston.github.io/chalk/",
    tagline:"ASCII-art banners from FIGfont — 1,027 fonts bundled in.",
    blurb:"A Go port of the classic figlet / Node figlet library: render text as FIGfont ASCII art. It bundles "+
      "1,027 named fonts in the registry, parses any standard .flf font from disk, supports full-width / kerning "+
      "/ smushing layouts, and can tint banners with chalk-powered gradient and rainbow coloring.",
    tags:["figlet","FIGfont","ASCII art","banners","fonts","gradient","rainbow"],
    features:[
      "<code>Render</code> / <code>RenderFont</code> — turn text into FIGfont ASCII-art",
      "1,027 bundled fonts in the registry — enumerate with <code>Fonts()</code>, fetch via <code>GetFont</code>",
      "Load real FIGfonts: <code>LoadFontFile</code>, <code>LoadFont</code>, <code>ParseFont</code>, <code>LoadFontDir</code> (.flf)",
      "Layout control: <code>Options{Layout, Width}</code> with <code>LayoutFull</code>, <code>LayoutKerning</code>, <code>LayoutSmush</code>",
      "Gradient coloring: <code>Gradient</code> / <code>RenderGradient</code> between two hex colors",
      "Rainbow coloring: <code>Rainbow</code> / <code>RenderRainbow</code> across the hue spectrum",
      "Build fonts programmatically: <code>BuiltinFont</code>, <code>FontFromGlyphs</code>, <code>Register</code>"
    ],
    node_code:
`const figlet = require('figlet')

console.log(figlet.textSync('Hello'))
console.log(figlet.textSync('GO', { font: 'Standard' }))`,
    go_code:
`import "github.com/malcolmston/chalk/figlet"

fmt.Println(figlet.Render("Hello"))
banner, _ := figlet.RenderFont("standard", "GO")
fmt.Println(banner)`,
    integrate:
`<span class="tok-c">// 1,027 bundled fonts — count them or pick one by name</span>
fmt.Println(len(figlet.Fonts()))               <span class="tok-c">// 1027</span>
banner, _ := figlet.RenderFont("banner", "GO")

<span class="tok-c">// Colorize a rendered banner (chalk under the hood)</span>
fmt.Println(figlet.Rainbow(banner))
fmt.Println(figlet.RenderGradient("Hi", "#ff0080", "#00d7ff"))

<span class="tok-c">// Or load a real .flf FIGfont from disk</span>
f, _ := figlet.LoadFontFile("slant.flf")
fmt.Println(f.Render("Hello"))`
  }
];
