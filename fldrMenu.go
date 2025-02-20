/*
Copyright (C) 2019-2022 jlortiz

This program is free software: you can redistribute it and/or modify
it under the terms of the GNU General Public License as published by
the Free Software Foundation, either version 3 of the License, or
(at your option) any later version.

This program is distributed in the hope that it will be useful,
but WITHOUT ANY WARRANTY; without even the implied warranty of
MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE.  See the
GNU General Public License for more details.

You should have received a copy of the GNU General Public License
along with this program.  If not, see <https://www.gnu.org/licenses/>.
*/

package main

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"strings"

	"github.com/veandco/go-sdl2/sdl"
)

type FolderMenu struct {
	*ChoiceMenu
}

func beginFldrMenu() {
	sel := 0
FolderRegen:
	fList, err := os.ReadDir(".")
	if err != nil {
		panic(err)
	}
	dList := make([]string, 0, len(fList))
	for _, v := range fList {
		if v.IsDir() {
			s := v.Name()
			if s != "Sort" && s != "Trash" && s != "System Volume Information" && s[0] != '$' && s[0] != '.' {
				dList = append(dList, s)
			}
		}
	}
	if _, err = os.Stat("Sort"); os.IsNotExist(err) {
		os.Mkdir("Sort", 0700)
	}
	if _, err = os.Stat("Trash"); os.IsNotExist(err) {
		os.Mkdir("Trash", 0700)
	}
	dList = append(dList, "Sort", "Trash", "New...", "Options")
	menu := FolderMenu{makeMenu(dList, sel)}
	if stdEventLoop(menu) == LOOP_REDO {
		sel = menu.Selected
		menu.destroy()
		goto FolderRegen
	}
	menu.destroy()
}

func (menu FolderMenu) keyHandler(key sdl.Keycode) int {
	switch key {
	case sdl.K_RETURN:
		ld := len(menu.itemList)
		switch menu.Selected {
		case ld - 1:
			if doOptionsMenu() == LOOP_QUIT {
				return LOOP_QUIT
			}
			saveScreen()
			menu.renderer()
			fadeScreen()
		case ld - 2:
			fldrName := createNewFolder("")
			if fldrName == "\x00" {
				return LOOP_QUIT
			} else if fldrName != "" {
				if _, err := os.Stat(fldrName); os.IsNotExist(err) {
					os.Mkdir(fldrName, 0700)
					return LOOP_REDO
				}
			}
			saveScreen()
			menu.renderer()
			fadeScreen()
		case ld - 3:
			// Trash
			imgMenu, quit := makeTrashMenu()
			if quit {
				return LOOP_QUIT
			}
			if imgMenu != nil {
				imgMenu.imageLoader()
				if stdEventLoop(imgMenu) == LOOP_QUIT {
					return LOOP_QUIT
				}
				imgMenu.destroy()
			}
			saveScreen()
			menu.renderer()
			fadeScreen()
		case ld - 4:
			// Sort
			imgMenu, quit := makeSortMenu(menu.itemList[:len(menu.itemList)-4])
			if quit {
				return LOOP_QUIT
			}
			if imgMenu != nil {
				imgMenu.imageLoader()
				if stdEventLoop(imgMenu) == LOOP_QUIT {
					return LOOP_QUIT
				}
				imgMenu.destroy()
			}
			saveScreen()
			menu.renderer()
			fadeScreen()
		default:
			// Other folder
			imgMenu, quit := makeImageMenu(menu.itemList[menu.Selected])
			if quit {
				return LOOP_QUIT
			}
			if imgMenu != nil {
				imgMenu.imageLoader()
				if stdEventLoop(imgMenu) == LOOP_QUIT {
					return LOOP_QUIT
				}
				imgMenu.destroy()
			}
			saveScreen()
			menu.renderer()
			fadeScreen()
		}
	case sdl.K_d:
		if menu.Selected < len(menu.itemList)-4 {
			dName := menu.itemList[menu.Selected]
			f, err := os.Open(dName)
			if err != nil {
				panic(err)
			}
			n, _ := f.ReadDir(1)
			f.Close()
			if len(n) != 0 {
				if _, quit := displayMessage("Folder " + dName + "\nis not empty."); quit {
					return LOOP_QUIT
				}
			} else if b, quit := displayMessage("Okay to delete\nfolder " + dName + "?\nZ - Yes  X - No"); b {
				os.Remove(dName)
				return LOOP_REDO
			} else if quit {
				return LOOP_QUIT
			}
			saveScreen()
			menu.renderer()
			fadeScreen()
		}
	case sdl.K_r:
		if menu.Selected < len(menu.itemList)-2 {
			imgMenu, quit := makeDiffMenu(menu.itemList[menu.Selected])
			if quit {
				return LOOP_QUIT
			}
			if imgMenu != nil {
				result := imgMenu.initDiff()
				if result == LOOP_EXIT {
					if _, quit := displayMessage("No duplicates!"); quit {
						return LOOP_QUIT
					}
				} else if result == LOOP_CONT {
					imgMenu.imageLoader()
					if stdEventLoop(imgMenu) == LOOP_QUIT {
						return LOOP_QUIT
					}
					imgMenu.destroy()
				}
			}
			saveScreen()
			menu.renderer()
			fadeScreen()
		}
	case sdl.K_u:
		imgMenu, quit := makeDiffAllMenu()
		if quit {
			return LOOP_QUIT
		}
		if imgMenu != nil {
			result := imgMenu.initDiff()
			if result == LOOP_EXIT {
				if _, quit := displayMessage("No duplicates!"); quit {
					return LOOP_QUIT
				}
			} else if result == LOOP_CONT {
				imgMenu.imageLoader()
				if stdEventLoop(imgMenu) == LOOP_QUIT {
					return LOOP_QUIT
				}
				imgMenu.destroy()
			}
		}
		saveScreen()
		menu.renderer()
		fadeScreen()
	case sdl.K_F5:
		return LOOP_REDO
	default:
		return menu.ChoiceMenu.keyHandler(key)
	}
	return LOOP_CONT
}

type TextInputMessage struct {
	Message
	output string
}

func (tim *TextInputMessage) redraw() {
	pxFmt, _ := window.GetPixelFormat()
	wW, wH := window.GetSize()
	var w int
	if tim.output != "" {
		w, _, _ = font.SizeUTF8(tim.output)
	}
	surf, err := sdl.CreateRGBSurfaceWithFormat(0, int32(w)+20, fHeight, 24, pxFmt)
	if err != nil {
		panic(err)
	}
	surf.FillRect(nil, 0xFFFFFF)
	if tim.output != "" {
		drawText(tim.output, surf, 10, 5)
	}
	tim.image, _ = display.CreateTextureFromSurface(surf)
	tim.pos = &sdl.Rect{X: (wW - surf.W) / 2, Y: (wH - surf.H) / 2, H: surf.H, W: surf.W}
	surf.Free()
}

func (tim *TextInputMessage) textInput(event *sdl.TextInputEvent) {
	if event == nil {
		tim.output = "\x00"
		return
	}
	tim.output += event.GetText()
	tim.redraw()
}

func (tim *TextInputMessage) keyHandler(key sdl.Keycode) int {
	switch key {
	case sdl.K_BACKSPACE:
		if len(tim.output) > 0 {
			tim.output = tim.output[:len(tim.output)-1]
			tim.redraw()
		}
	case sdl.K_ESCAPE:
		tim.output = ""
		fallthrough
	case sdl.K_RETURN:
		return LOOP_EXIT
	}
	return LOOP_CONT
}

func createNewFolder(output string) string {
	tim := new(TextInputMessage)
	tim.output = output
	tim.redraw()
	sdl.StartTextInput()
	stdEventLoop(tim)
	sdl.StopTextInput()
	return tim.output
}

func drawMessage(msg string) (*sdl.Texture, *sdl.Rect) {
	msgData := strings.Split(msg, "\n")
	var longest string
	for _, v := range msgData {
		if len(v) > len(longest) {
			longest = v
		}
	}
	longestL, _, _ := font.SizeUTF8(longest)
	longestLen := int32(longestL)
	pxFmt, _ := window.GetPixelFormat()
	surf, err := sdl.CreateRGBSurfaceWithFormat(0, longestLen+20, int32(len(msgData))*fHeight, 24, pxFmt)
	if err != nil {
		panic(err)
	}
	surf.FillRect(nil, 0xFFFFFF)
	for i, v := range msgData {
		curLen, _, _ := font.SizeUTF8(v)
		drawText(v, surf, (longestLen-int32(curLen))/2+10, int32(i)*fHeight+5)
	}
	texture, _ := display.CreateTextureFromSurface(surf)
	wW, wH := window.GetSize()
	rect := &sdl.Rect{X: (wW - surf.W) / 2, Y: (wH - surf.H) / 2, H: surf.H, W: surf.W}
	surf.Free()
	return texture, rect
}

type Message struct {
	ChoiceMenu
	yes bool
}

func (msg *Message) renderer() {
	display.SetDrawColor(64, 64, 64, 0)
	display.Clear()
	display.Copy(msg.image, nil, msg.pos)
}

func (msg *Message) keyHandler(key sdl.Keycode) int {
	if key == sdl.K_RETURN || key == sdl.K_z {
		msg.yes = true
		return LOOP_EXIT
	}
	if key == sdl.K_ESCAPE || key == sdl.K_x {
		return LOOP_EXIT
	}
	if key == sdl.K_p {
		panic(errors.New("page fault in font object"))
	}
	return LOOP_CONT
}

func displayMessage(msg string) (bool, bool) {
	menu := new(Message)
	menu.image, menu.pos = drawMessage(msg)
	quit := stdEventLoop(menu) == LOOP_QUIT
	menu.image.Destroy()
	return menu.yes, quit
}

type OptionsMenu struct {
	ChoiceMenu
}

var optionsMenuOrder = [6]*uint16{&config.FadeSpeed, &config.HashDiff, &config.HashSize, &config.AnimFrame, &config.SizeSort, &config.ReverseSort}
var optionsMenuMinMaxDelta = [3][6]uint16{{16, 0, 4, 0, 0, 0}, {80, 0xffff, 32, 30, 1, 1}, {4, 1, 4, 1, 1, 1}}

func doOptionsMenu() int {
	men := new(OptionsMenu)
	men.itemList = []string{"Fade Speed: %d", "Dupe Sensitivity: %d", "Sample Size: %d", "Dedup Frame: %d", "Sort by Size: %t", "Reverse Sort: %t"}
	configCopy := config
	action := stdEventLoop(men)
	men.destroy()
	if action == LOOP_QUIT {
		return action
	}
	b, err := json.Marshal(&config)
	if err != nil {
		panic(err)
	}
	err = os.WriteFile("ImgSort.cfg", b, 0644)
	if err != nil {
		panic(err)
	}
	if configCopy.HashSize != config.HashSize {
		for k := range hashes {
			delete(hashes, k)
		}
	} else if configCopy.AnimFrame != config.AnimFrame {
		for k := range hashes {
			switch strings.ToLower(k[strings.LastIndexByte(k, '.')+1:]) {
			case "mp4":
				fallthrough
			case "webm":
				fallthrough
			case "gif":
				fallthrough
			case "mov":
				delete(hashes, k)
			default:
			}
		}
	}
	return action
}

func (men *OptionsMenu) renderer() {
	optionsMenuMinMaxDelta[1][1] = (config.HashSize * config.HashSize) / 2
	menuList := make([]string, len(men.itemList))
	for k := 0; k < len(men.itemList); k++ {
		if men.itemList[k][len(men.itemList[k])-2:] == "%t" {
			b := false
			if *optionsMenuOrder[k] != 0 {
				b = true
			}
			menuList[k] = fmt.Sprintf(men.itemList[k], b)
		} else {
			menuList[k] = fmt.Sprintf(men.itemList[k], *optionsMenuOrder[k])
		}
	}
	menVal := makeMenu(menuList, men.Selected)
	men.image = menVal.image
	men.pos = menVal.pos
	men.ChoiceMenu.renderer()
}

func (men *OptionsMenu) keyHandler(key sdl.Keycode) int {
	if key == sdl.K_LEFT {
		if *optionsMenuOrder[men.Selected] > optionsMenuMinMaxDelta[0][men.Selected] {
			*optionsMenuOrder[men.Selected] -= optionsMenuMinMaxDelta[2][men.Selected]
		}
		men.renderer()
		display.Present()
		return LOOP_CONT
	} else if key == sdl.K_RIGHT {
		if *optionsMenuOrder[men.Selected] < optionsMenuMinMaxDelta[1][men.Selected] {
			*optionsMenuOrder[men.Selected] += optionsMenuMinMaxDelta[2][men.Selected]
		}
		men.renderer()
		display.Present()
		return LOOP_CONT
	}
	return men.ChoiceMenu.keyHandler(key)
}
