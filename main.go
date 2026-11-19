package main

import (
	"os"

	"github.com/ForceCLI/force/command"
	"github.com/ForceCLI/force/sf"
)

func main() {
	if sf.InvokedAsSf(os.Args[0]) {
		sf.Execute()
		return
	}
	command.Execute()
}
