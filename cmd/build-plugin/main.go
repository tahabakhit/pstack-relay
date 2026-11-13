// Command build-plugin generates marketplace artifacts for development and CI.
package main

import (
	"flag"
	"fmt"
	"os"
	"path/filepath"

	"github.com/tahabakhit/pstack-rex/internal/plugin"
)

func main() {
	check := flag.Bool("check", false, "check generated files without writing")
	flag.Parse()
	if flag.NArg() != 0 {
		fmt.Fprintln(os.Stderr, "unexpected positional arguments")
		os.Exit(1)
	}
	root, err := checkoutRoot()
	if err == nil {
		err = plugin.Build(root, *check)
	}
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	if *check {
		fmt.Println("plugin tree is up to date")
	} else {
		fmt.Println("built plugins/pstack-rex and marketplace manifests")
	}
}
func checkoutRoot() (string, error) {
	dir, err := os.Getwd()
	if err != nil {
		return "", err
	}
	for {
		if _, err := os.Stat(filepath.Join(dir, "upstream", "REVISION")); err == nil {
			if _, err := os.Stat(filepath.Join(dir, "internal", "plugin", "adaptations.json")); err == nil {
				return dir, nil
			}
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			return "", fmt.Errorf("build-plugin must run inside the checkout")
		}
		dir = parent
	}
}
