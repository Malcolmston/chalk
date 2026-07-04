// Library content for the chalk documentation site. The chalk entry is copied
// verbatim from the malcolmston/go landing (web/src/data.ts) so the two sites
// stay in sync.
export interface Lib {
  id: string; name: string; icon: string; accent: string; pkg: string; node: string;
  repo: string; docs: string; tagline: string; blurb: string; tags: string[];
  features: string[]; node_code: string; go_code: string; integrate: string;
}
export const NODE_ACCENT = '#8cc84b';
export const LIBS: Lib[] = [
  {
    id:"chalk", name:"chalk", icon:'<i class="fa-solid fa-palette"></i>', accent:"#ffa657",
    pkg:"github.com/malcolmston/chalk", node:"chalk/chalk",
    repo:"https://github.com/malcolmston/chalk", docs:"https://malcolmston.github.io/chalk/",
    tagline:"Terminal string styling done right — plus prompts &amp; figlet.",
    blurb:"Chainable, immutable ANSI styling (16 / 256 / truecolor) with automatic capability degradation and "+
      "NO_COLOR / FORCE_COLOR support. Includes chalk/prompts (inquirer-style interactive prompts) and "+
      "chalk/figlet (ASCII-art banners with bundled fonts, gradients and rainbow).",
    tags:["ANSI colors","truecolor","NO_COLOR","prompts","figlet","gradients"],
    features:[
      "Chainable styles: <code>chalk.New().Red().Bold().Sprint(\"x\")</code>",
      "Modifiers (bold, dim, italic, underline, inverse, strikethrough …) and bg colors",
      "Truecolor / 256-color: <code>RGB</code>, <code>Hex</code>, <code>Ansi256</code>, auto-degrading",
      "<code>chalk.Strip</code> and <code>chalk.VisibleLength</code> helpers",
      "<b>chalk/prompts</b> — input, confirm, select, multiselect, password (raw-mode TTY)",
      "<b>chalk/figlet</b> — FIGfont rendering, bundled distinct fonts, gradient &amp; rainbow coloring"
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
`<span class="tok-c">// Prompts and figlet banners</span>
import (
    "github.com/malcolmston/chalk/prompts"
    "github.com/malcolmston/chalk/figlet"
)

name, _ := prompts.Input(prompts.InputConfig{Message: "Your name?"})
ok, _   := prompts.Confirm(prompts.ConfirmConfig{Message: "Proceed?"})

fmt.Println(figlet.Render("Hello"))                 <span class="tok-c">// default font</span>
banner, _ := figlet.RenderFont("banner", "GO")      <span class="tok-c">// bundled outline font</span>
fmt.Println(figlet.RenderRainbow(banner))`
  }
];
