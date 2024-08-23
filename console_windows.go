package main

import (
	"os"
	"os/exec"
	"path/filepath"
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
	cmd := exec.Command("explorer", "/select,", filepath.Join(cwd, p1, p2))
	cwd = "explorer /select," + cmd.Args[2]
	cmd.SysProcAttr = &syscall.SysProcAttr{CmdLine: cwd}
	cmd.Run()
}

func viewFile(p string) {
	exec.Command("rundll32.exe", "url.dll,FileProtocolHandler", p).Run()
}
