package main

import (
	"fmt"
	"os"
	"os/exec"
	"syscall"

	"github.com/TheTitanrain/w32"
	"golang.org/x/sys/windows"
)

func hideConsole() {
	console := w32.GetConsoleWindow()
	if console != 0 {
		_, consoleId := w32.GetWindowThreadProcessId(console)
		if int(windows.GetCurrentProcessId()) == consoleId {
			w32.ShowWindow(console, w32.SW_HIDE)
		}
	}
}

func highlightFile(p1, p2 string) {
	cwd, _ := os.Getwd()
	cmd := exec.Command("explorer", "/select,", fmt.Sprintf("\"%s%c%s%c%s\"", cwd, os.PathSeparator, p1, os.PathSeparator, p2))
	cwd = fmt.Sprintf("explorer /select,%s", cmd.Args[2])
	cmd.SysProcAttr = &syscall.SysProcAttr{CmdLine: cwd}
	cmd.Run()
}
