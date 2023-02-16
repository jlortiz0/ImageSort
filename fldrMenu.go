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
	"sort"
	"strings"

	"github.com/veandco/go-sdl2/sdl"
)

type FolderMenu struct {
	*ChoiceMenu
}

func beginFldrMenu() {
	sel := 0
FolderRegen:
	f, err := os.Open(".")
	if err != nil {
		panic(err)
	}
	fList, err := f.ReadDir(0)
	if err != nil {
		panic(err)
	}
	f.Close()
	dList := make([]string, 0, len(fList))
	for _, v := range fList {
		if v.IsDir() {
			s := v.Name()
			if s != "Sort" && s != "Trash" && s != "System Volume Information" && s[0] != '$' && s[0] != '.' {
				dList = append(dList, s)
			}
		}
	}
	sort.Strings(dList)
	if _, err = os.Stat("Sort"); os.IsNotExist(err) {
		os.Mkdir("Sort", 0600)
	}
	if _, err = os.Stat("Trash"); os.IsNotExist(err) {
		os.Mkdir("Trash", 0600)
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
			if fldrName == "CANCEL" {
				return LOOP_QUIT
			} else if fldrName != "" {
				if _, err := os.Stat(fldrName); os.IsNotExist(err) {
					os.Mkdir(fldrName, 0600)
					return LOOP_REDO
				}
			}
			saveScreen()
			menu.renderer()
			fadeScreen()
		case ld - 3:
			// Trash
			imgMenu := makeTrashMenu()
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
			imgMenu := makeSortMenu(menu.itemList[:len(menu.itemList)-4])
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
			imgMenu := makeImageMenu(menu.itemList[menu.Selected])
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
				displayMessage("Folder " + dName + "\nis not empty.")
			} else if displayMessage("Sure to delete\nfolder " + dName + "?\nZ - Yes  X - No") {
				os.Remove(dName)
				return LOOP_REDO
			}
			saveScreen()
			menu.renderer()
			fadeScreen()
		}
	case sdl.K_r:
		if menu.Selected < len(menu.itemList)-2 {
			imgMenu := makeDiffMenu(menu.itemList[menu.Selected])
			if imgMenu != nil {
				saveScreen()
				result := imgMenu.initDiff()
				if result == LOOP_EXIT {
					displayMessage("No duplicates!")
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
		imgMenu := makeDiffAllMenu()
		if imgMenu != nil {
			saveScreen()
			result := imgMenu.initDiff()
			if result == LOOP_EXIT {
				displayMessage("No duplicates!")
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

func createNewFolder(output string) string {
	wW, wH := window.GetSize()
	rerender := func() {
		txtSurf, err := font.RenderUTF8Shaded(output, COLOR_BLACK, COLOR_WHITE)
		if err != nil {
			if output == "" {
				txtSurf, _ = sdl.CreateRGBSurfaceWithFormat(0, 0, int32(font.Height()), 0, 0)
			} else {
				panic(err)
			}
		}
		rect := txtSurf.ClipRect
		rect.H += 10
		rect.W += 20
		rect.X = (wW - txtSurf.W - 20) / 2
		rect.Y = (wH - txtSurf.H - 10) / 2
		display.SetDrawColor(64, 64, 64, 0)
		display.Clear()
		display.SetDrawColor(255, 255, 255, 0)
		display.FillRect(&rect)
		if output != "" {
			rect.X += 10
			rect.Y += 5
			texture, _ := display.CreateTextureFromSurface(txtSurf)
			display.Copy(texture, nil, &sdl.Rect{X: rect.X, Y: rect.Y, H: txtSurf.H, W: txtSurf.W})
			texture.Destroy()
		}
		txtSurf.Free()
	}
	sdl.StartTextInput()
	saveScreen()
	rerender()
	fadeScreen()
Outer:
	for {
		delay()
		for event := sdl.PollEvent(); event != nil; event = sdl.PollEvent() {
			switch event := event.(type) {
			case *sdl.QuitEvent:
				output = "CANCEL"
				break Outer
			case *sdl.TextInputEvent:
				output += event.GetText()
				rerender()
				display.Present()
			case *sdl.KeyboardEvent:
				switch event.Keysym.Sym {
				case sdl.K_BACKSPACE:
					if len(output) > 0 {
						output = output[:len(output)-1]
						rerender()
						display.Present()
					}
				case sdl.K_ESCAPE:
					output = ""
					fallthrough
				case sdl.K_RETURN:
					if output == "CANCEL" {
						output = ""
					}
					break Outer
				}
			case *sdl.WindowEvent:
				if event.Event == sdl.WINDOWEVENT_SIZE_CHANGED {
					pxFmt, _ := window.GetPixelFormat()
					fadeBg.Destroy()
					fadeFg.Destroy()
					fadeFg, _ = display.CreateTexture(pxFmt, sdl.TEXTUREACCESS_TARGET, event.Data1, event.Data2)
					fadeBg, _ = display.CreateTexture(pxFmt, sdl.TEXTUREACCESS_TARGET, event.Data1, event.Data2)
					rerender()
					display.Present()
				}
			}
		}
	}
	sdl.StopTextInput()
	return output
}

func drawMessage(msg string) (*sdl.Texture, *sdl.Rect) {
	msgData := strings.Split(msg, "\n")
	fHeight := int32(font.Height()) + 10
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

func displayMessage(msg string) bool {
	menu := new(Message)
	menu.image, menu.pos = drawMessage(msg)
	if stdEventLoop(menu) == LOOP_QUIT {
		sdl.PushEvent(&sdl.QuitEvent{})
	}
	menu.image.Destroy()
	return menu.yes
}

type OptionsMenu struct {
	ChoiceMenu
}

func doOptionsMenu() int {
	men := new(OptionsMenu)
	men.itemList = []string{"Fade Speed: %d", "Dupe Sensitivity: %d", "Sample Size: %d"}
	action := stdEventLoop(men)
	men.destroy()
	f, err := os.OpenFile("ImgSort.cfg", os.O_WRONLY|os.O_CREATE|os.O_TRUNC, 0644)
	if err != nil {
		panic(err)
	}
	b, err := json.Marshal(&config)
	if err != nil {
		panic(err)
	}
	f.Write(b)
	f.Close()
	return action
}

func (men *OptionsMenu) renderer() {
	menuValues := []interface{}{config.FadeSpeed, config.HashDiff, config.HashSize}
	menuList := make([]string, len(men.itemList))
	for k := 0; k < len(men.itemList); k++ {
		menuList[k] = fmt.Sprintf(men.itemList[k], menuValues[k])
	}
	menVal := makeMenu(menuList, men.Selected)
	men.image = menVal.image
	men.pos = menVal.pos
	men.ChoiceMenu.renderer()
}

func (men *OptionsMenu) keyHandler(key sdl.Keycode) int {
	if key == sdl.K_LEFT || key == sdl.K_RIGHT {
		switch men.Selected {
		case 0: // FadeSpeed
			if key == sdl.K_LEFT {
				if config.FadeSpeed > 16 {
					config.FadeSpeed -= 4
				}
			} else {
				if config.FadeSpeed < 80 {
					config.FadeSpeed += 4
				}
			}
		case 1: // HashDiff
			if key == sdl.K_LEFT {
				if config.HashDiff != 0 {
					config.HashDiff--
				}
			} else if config.HashDiff < (uint16(config.HashSize)*uint16(config.HashSize))/2 {
				config.HashDiff++
			}
		case 2: // HashSize
			if key == sdl.K_LEFT {
				if config.HashSize > 4 {
					config.HashSize -= 4
				}
			} else if config.HashSize < 32 {
				config.HashSize += 4
			}
		}
		if config.HashDiff > (uint16(config.HashSize)*uint16(config.HashSize))/2 {
			config.HashDiff = (uint16(config.HashSize) * uint16(config.HashSize)) / 2
		}
		men.renderer()
		display.Present()
		return LOOP_CONT
	}
	return men.ChoiceMenu.keyHandler(key)
}
