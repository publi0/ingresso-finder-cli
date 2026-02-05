package main

import (
	"fmt"
	"os"

	tea "github.com/charmbracelet/bubbletea"
	"ingresso-finder-cli/tui"
)

const appName = "ingresso-finder-cli"

var (
	version = "dev"
	commit  = "none"
)

func printUsage(out *os.File) {
	fmt.Fprintf(out, "Usage: %s [--version]\n", appName)
}

func printVersion() {
	fmt.Printf("%s %s", appName, version)
	if commit != "none" && commit != "" {
		fmt.Printf(" (%s)", commit)
	}
	fmt.Println()
}

func handleArgs(args []string) bool {
	if len(args) == 0 {
		return true
	}

	for _, arg := range args {
		switch arg {
		case "-h", "--help", "help":
			printUsage(os.Stdout)
			return false
		case "-v", "--version", "version":
			printVersion()
			return false
		default:
			fmt.Fprintf(os.Stderr, "Unknown argument: %s\n", arg)
			printUsage(os.Stderr)
			os.Exit(2)
		}
	}

	return false
}

func main() {
	if !handleArgs(os.Args[1:]) {
		return
	}

	if _, err := tea.NewProgram(tui.New(), tea.WithAltScreen()).Run(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
