package main

import "github.com/xRiErOS/beans/internal/commands"

func main() {
	root := commands.NewRootCmd()
	commands.RegisterCoreCommands(root)
	commands.Execute(root)
}
