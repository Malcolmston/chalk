// Command demo shows chalk color styling, a figlet banner, and prompts.
package main

import (
	"fmt"

	"github.com/malcolmston/chalk"
	"github.com/malcolmston/chalk/figlet"
	"github.com/malcolmston/chalk/prompts"
)

func main() {
	fmt.Println(chalk.New().Hex("#00d7ff").Bold().Sprint(figlet.Render("Go Chalk")))
	fmt.Println(chalk.Green("✔ ") + chalk.Bold("styled ") + chalk.New().Yellow().Underline().Sprint("terminal") + " output")

	name, _ := prompts.Input(prompts.InputConfig{Message: "What's your name?", Default: "friend"})
	fmt.Printf("Hello, %s!\n", chalk.Cyan(name))

	i, choice, _ := prompts.Select(prompts.SelectConfig{
		Message: "Pick a color",
		Choices: []prompts.Choice{{Name: "Red"}, {Name: "Green"}, {Name: "Blue"}},
	})
	fmt.Printf("You picked #%d: %s\n", i, chalk.Bold(choice.Name))
}
