package main

import (
	"github.com/TheTitanrain/w32"
	"golang.org/x/sys/windows"
)

func shouldPause() bool {
	console := w32.GetConsoleWindow()
	if console != 0 {
		_, consoleId := w32.GetWindowThreadProcessId(console)
		return int(windows.GetCurrentProcessId()) == consoleId
	}
	return false
}
