package main

import (
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
