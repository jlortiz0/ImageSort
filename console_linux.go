package main

import (
	"os"
	"os/exec"
	"path/filepath"
)

func hideConsole() {}

// TODO: There's probably not a good cross-de way to do this, but if there is, it would be nice to fix this.
func highlightFile(p1, _ string) {
	cwd, _ := os.Getwd()
	// xdg-open can block depending on what program is launched, so wait in a goroutine.
	go exec.Command("xdg-open", filepath.Join(cwd, p1)).Run()
}

func viewFile(p string) {
	go exec.Command("xdg-open", p).Run()
}
