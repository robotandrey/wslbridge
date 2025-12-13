package main

import (
	"fmt"
	"os"

	"wslbridge/internal/command"
	"wslbridge/internal/commands"
	"wslbridge/internal/execx"
	"wslbridge/internal/platform"
	"wslbridge/internal/runtime"
)

func main() {
	platformInfo, err := platform.Detect()
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}

	rt, err := runtime.New(execx.OSRunner{}, platformInfo)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}

	reg := command.New(commands.All()...)

	args := os.Args[1:]
	if len(args) == 0 || isHelp(args[0]) {
		printHelp(reg)
		os.Exit(0)
	}

	cmdName := args[0]
	cmdArgs := args[1:]

	cmd, ok := reg.Get(cmdName)
	if !ok {
		fmt.Fprintln(os.Stderr, "unknown command:", cmdName)
		fmt.Fprintln(os.Stderr)
		printHelp(reg)
		os.Exit(2)
	}

	if err := cmd.Run(rt, cmdArgs); err != nil {
		fmt.Fprintln(os.Stderr, "error:", err)
		os.Exit(1)
	}
}

func isHelp(s string) bool {
	return s == "help" || s == "-h" || s == "--help"
}

func printHelp(reg command.Registry) {
	fmt.Println("Usage: wslbridge <command> [args]")
	fmt.Println()
	fmt.Println("Commands:")
	for _, line := range reg.HelpLines() {
		fmt.Println(line)
	}
}
