// Command banner prints a figlet banner from its arguments.
package main

import (
	"fmt"
	"os"
	"strings"

	"github.com/malcolmston/chalk/figlet"
)

func main() {
	text := "Hello"
	if len(os.Args) > 1 {
		text = strings.Join(os.Args[1:], " ")
	}
	fmt.Println(figlet.Render(text))
}
