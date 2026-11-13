package main

import (
	"fmt"
	"os"
	"strings"

	pstackrex "github.com/tahabakhit/pstack-rex"
)

func main() {
	if len(os.Args) == 2 && os.Args[1] == "version" {
		fmt.Printf("pstack-rex 1.0.0-dev\nupstream %s\n", strings.TrimSpace(pstackrex.UpstreamRevision))
		return
	}
	fmt.Fprintln(os.Stderr, "Usage: pstack-rex <command>\n\nCommands:\n  version  Print the version and upstream revision")
	if len(os.Args) > 1 && os.Args[1] != "--help" && os.Args[1] != "help" {
		os.Exit(1)
	}
}
