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
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path"
	"sort"
	"strconv"
	"strings"
	"syscall"
	"unsafe"

	"github.com/veandco/go-sdl2/img"
	"github.com/veandco/go-sdl2/sdl"
)

type ImageBrowser interface {
	Menu
	getHeight() int32
	getY() int32
	modY(int32)
	stopAnim()
	imageLoader() int
}

type ImageMenu struct {
	ChoiceMenu
	prevMoveDir  bool
	ffmpeg       *ffmpegReader
	fldr         string
	shouldReload bool
}

var flingOffsets = []int32{36, 43, 51, 62, 77, 95, 120, 152, 196, 255, 336, 449, 610, 840}

func makeImageMenu(fldr string) *ImageMenu {
	f, err := os.Open(fldr)
	if err != nil {
		panic(err)
	}
	entries, err := f.ReadDir(0)
	if err != nil {
		panic(err)
	}
	ls := make([]string, 0, len(entries))
	for _, v := range entries {
		if !v.IsDir() {
			ind := strings.LastIndexByte(v.Name(), '.')
			if ind == -1 {
				continue
			}
			switch strings.ToLower(v.Name()[ind+1:]) {
			case "mp4":
				fallthrough
			case "webm":
				fallthrough
			case "gif":
				fallthrough
			case "mov":
				fallthrough
			case "bmp":
				fallthrough
			case "jpg":
				fallthrough
			case "png":
				fallthrough
			case "jpeg":
				ls = append(ls, v.Name())
			}
		}
	}
	sort.Strings(ls)
	menu := new(ImageMenu)
	menu.fldr = fldr
	menu.itemList = ls
	if len(ls) == 0 {
		if len(entries) == 0 {
			displayMessage("Folder " + fldr + "\nis empty.")
		} else {
			displayMessage("Folder " + fldr + "\nhas no supported images.")
		}
		return nil
	}
	display.SetDrawColor(64, 64, 64, 0)
	return menu
}

func (menu *ImageMenu) destroy() {
	menu.image.Destroy()
	if menu.ffmpeg != nil {
		menu.ffmpeg.Destroy()
		menu.ffmpeg = nil
	}
}

func (menu *ImageMenu) getHeight() int32 {
	return menu.pos.H
}

func (menu *ImageMenu) getY() int32 {
	return menu.pos.Y
}

func (menu *ImageMenu) modY(y int32) {
	menu.pos.Y += y
}

func (menu *ImageMenu) stopAnim() {
	if menu.animated {
		menu.ffmpeg.Destroy()
		menu.ffmpeg = nil
	}
}

func moveFile(menu ImageBrowser, from, target string) int {
	moveFactor := 0
	for -menu.getHeight() < menu.getY() && menu.getY() < display.GetViewport().H {
		if target != "Trash" {
			menu.modY(-flingOffsets[moveFactor])
		} else {
			menu.modY(flingOffsets[moveFactor])
		}
		if moveFactor < len(flingOffsets)-1 {
			moveFactor++
		}
		menu.renderer()
		display.Present()
		delay()
	}
	menu.stopAnim()
	newName := path.Base(from)
	if _, err := os.Stat(path.Join(target, newName)); err == nil {
		x := -1
		dLoc := strings.IndexByte(newName, '.')
		before := newName
		var after string
		if dLoc != -1 {
			before = newName[:dLoc]
			after = newName[dLoc+1:]
		}
		for ; err == nil; _, err = os.Stat(path.Join(target, fmt.Sprintf("%s_%d.%s", before, x, after))) {
			x++
		}
		newName = fmt.Sprintf("%s_%d.%s", before, x, after)
	}
	os.Rename(from, path.Join(target, newName))
	if target != "Trash" {
		hashes[path.Join(target, newName)] = hashes[from]
	}
	delete(hashes, from)
	ret := menu.imageLoader()
	menu.renderer()
	display.Present()
	return ret
}

func (menu *ImageMenu) keyHandler(key sdl.Keycode) int {
	switch key {
	case sdl.K_LEFT:
		if menu.Selected > 0 {
			menu.Selected--
			menu.prevMoveDir = true
			menu.drawNext = true
			menu.shouldReload = true
		}
	case sdl.K_RIGHT:
		if menu.Selected < len(menu.itemList)-1 {
			menu.Selected++
			menu.prevMoveDir = false
			menu.drawNext = true
			menu.shouldReload = true
		}
	case sdl.K_HOME:
		if menu.Selected != 0 {
			menu.Selected = 0
			menu.prevMoveDir = false
			menu.drawNext = true
			menu.shouldReload = true
		}
	case sdl.K_END:
		if menu.Selected < len(menu.itemList) {
			menu.Selected = len(menu.itemList) - 1
			menu.drawNext = true
			menu.shouldReload = true
		}
	case sdl.K_z:
		stat, _ := os.Stat(path.Join(menu.fldr, menu.itemList[menu.Selected]))
		sz := float64(stat.Size()) / 1024
		if sz > 1024 {
			displayMessage(fmt.Sprintf("File: %s\nScale Height: %d\nScale Width: %d\nStorage: %.1f MiB", menu.itemList[menu.Selected], menu.pos.H, menu.pos.W, sz/1024))
		} else {
			displayMessage(fmt.Sprintf("File: %s\nScale Height: %d\nScale Width: %d\nStorage: %.1f KiB", menu.itemList[menu.Selected], menu.pos.H, menu.pos.W, sz))
		}
		// displayMessage(fmt.Sprintf("File: %s\nScale Height: %d\nScale Width: %d", menu.itemList[menu.Selected], menu.pos.H, menu.pos.W))
		saveScreen()
		menu.renderer()
		fadeScreen()
	case sdl.K_x:
		fallthrough
	case sdl.K_c:
		target := "Trash"
		if key == sdl.K_x {
			target = "Sort"
		}
		return moveFile(menu, path.Join(menu.fldr, menu.itemList[menu.Selected]), target)
	case sdl.K_DOWN:
		if menu.pos.W > 64 && menu.pos.H > 64 {
			menu.pos.W = menu.pos.W * 4 / 5
			menu.pos.H = menu.pos.H * 4 / 5
			menu.pos.X += menu.pos.W / 8
			menu.pos.Y += menu.pos.H / 8
			if menu.pos.W < display.GetViewport().W {
				menu.pos.X = (display.GetViewport().W - menu.pos.W) / 2
			} else if menu.pos.X > 0 {
				menu.pos.X = 0
			} else if menu.pos.X < display.GetViewport().W-menu.pos.W {
				menu.pos.X = display.GetViewport().W - menu.pos.W
			}
			if menu.pos.H < display.GetViewport().H {
				menu.pos.Y = (display.GetViewport().H - menu.pos.H) / 2
			} else if menu.pos.Y > 0 {
				menu.pos.Y = 0
			} else if menu.pos.Y < display.GetViewport().H-menu.pos.H {
				menu.pos.Y = display.GetViewport().H - menu.pos.H
			}
			menu.drawNext = true
		}
	case sdl.K_UP:
		if menu.pos.W < 10000 && menu.pos.H < 10000 {
			menu.pos.X -= menu.pos.W / 8
			menu.pos.Y -= menu.pos.H / 8
			menu.pos.W = menu.pos.W * 5 / 4
			menu.pos.H = menu.pos.H * 5 / 4
			menu.drawNext = true
		}
	case sdl.K_w:
		if menu.pos.Y < 0 {
			menu.pos.Y += 20
			if menu.pos.Y > 0 {
				menu.pos.Y = 0
			}
			menu.drawNext = true
		}
	case sdl.K_a:
		if menu.pos.X < 0 {
			menu.pos.X += 20
			if menu.pos.X > 0 {
				menu.pos.X = 0
			}
			menu.drawNext = true
		}
	case sdl.K_s:
		if menu.pos.H > display.GetViewport().H {
			menu.pos.Y -= 20
			if menu.pos.Y < display.GetViewport().H-menu.pos.H {
				menu.pos.Y = display.GetViewport().H - menu.pos.H
			}
			menu.drawNext = true
		}
	case sdl.K_d:
		if menu.pos.W > display.GetViewport().W {
			menu.pos.X -= 20
			if menu.pos.X < display.GetViewport().W-menu.pos.W {
				menu.pos.X = display.GetViewport().W - menu.pos.W
			}
			menu.drawNext = true
		}
	case sdl.K_g:
		str := createNewFolder(strconv.Itoa(menu.Selected + 1))
		if str == "CANCEL" {
			return LOOP_QUIT
		}
		saveScreen()
		if str != "" {
			i, err := strconv.Atoi(str)
			if err == nil && i < len(menu.itemList) && i > 0 {
				menu.Selected = i - 1
				menu.imageLoader()
			}
		}
		display.SetDrawColor(64, 64, 64, 0)
		menu.renderer()
		fadeScreen()
	case sdl.K_v:
		if os.PathSeparator == '\\' {
			exec.Command("rundll32.exe", "url.dll,FileProtocolHandler", menu.fldr+string(os.PathSeparator)+menu.itemList[menu.Selected]).Run()
		}
	case sdl.K_h:
		if os.PathSeparator == '\\' {
			cwd, _ := os.Getwd()
			cmd := exec.Command("explorer", "/select,", fmt.Sprintf("\"%s%c%s%c%s\"", cwd, os.PathSeparator, menu.fldr, os.PathSeparator, menu.itemList[menu.Selected]))
			cwd = fmt.Sprintf("explorer /select, %s", cmd.Args[2])
			cmd.SysProcAttr = &syscall.SysProcAttr{CmdLine: cwd}
			cmd.Run()
		}
	case sdl.K_p:
		panic(errors.New("no windows available for re-popping"))
	}
	return LOOP_CONT
}

func (menu *ImageMenu) imageLoader() int {
	if len(menu.itemList) == 0 {
		menu.animated = false
		return LOOP_EXIT
	}
	_, _, sx, sy, _ := loading.Query()
	display.Copy(loading, nil, &sdl.Rect{W: sx, H: sy, X: display.GetViewport().W - sx, Y: display.GetViewport().H - sy})
	display.Present()
	if menu.image != nil {
		menu.image.Destroy()
	}
	if menu.ffmpeg != nil {
		menu.ffmpeg.Destroy()
		menu.ffmpeg = nil
	}
	var err error
Error:
	if err != nil {
		if _, err2 := os.Stat(path.Join(menu.fldr, menu.itemList[menu.Selected])); os.IsNotExist(err2) {
			if menu.Selected == len(menu.itemList)-1 {
				menu.Selected--
			} else {
				copy(menu.itemList[menu.Selected:], menu.itemList[menu.Selected+1:])
				if menu.prevMoveDir && menu.Selected > 0 {
					menu.Selected--
				}
			}
			menu.itemList = menu.itemList[:len(menu.itemList)-1]
			return menu.imageLoader()
		}
		menu.animated = false
		menu.image, menu.pos = drawMessage("Error loading " + menu.itemList[menu.Selected] + ":\n" + err.Error())
		return LOOP_CONT
	}
	wW, wH := window.GetSize()
	ind := strings.LastIndexByte(menu.itemList[menu.Selected], '.')
	ext := strings.ToLower(menu.itemList[menu.Selected][ind+1:])
	if ext == "mp4" || ext == "webm" || ext == "mov" || ext == "gif" {
		menu.ffmpeg = newFfmpegReader(path.Join(menu.fldr, menu.itemList[menu.Selected]))
		if menu.ffmpeg.h < 1 || menu.ffmpeg.w < 1 {
			menu.ffmpeg.Destroy()
			err = strconv.ErrRange
			goto Error
		}
		menu.image, err = display.CreateTexture(sdl.PIXELFORMAT_RGB24, sdl.TEXTUREACCESS_STREAMING, menu.ffmpeg.w, menu.ffmpeg.h)
		if err != nil {
			menu.image.Destroy()
			menu.ffmpeg.Destroy()
			goto Error
		}
		if menu.ffmpeg.h*wW >= menu.ffmpeg.w*wH {
			sy = wH
			sx = wH * menu.ffmpeg.w / menu.ffmpeg.h
		} else {
			sx = wW
			sy = wW * menu.ffmpeg.h / menu.ffmpeg.w
		}
		menu.pos = &sdl.Rect{X: (wW - sx) / 2, Y: (wH - sy) / 2, H: sy, W: sx}
		menu.animated = true
		return LOOP_CONT
	}
	rawImg, err := img.Load(path.Join(menu.fldr, menu.itemList[menu.Selected]))
	if err != nil {
		goto Error
	} else {
		menu.image, _ = display.CreateTextureFromSurface(rawImg)
		// var sx, sy int32
		if rawImg.H*wW >= rawImg.W*wH {
			sy = wH
			sx = wH * rawImg.W / rawImg.H
		} else {
			sx = wW
			sy = wW * rawImg.H / rawImg.W
		}
		menu.pos = &sdl.Rect{X: (wW - sx) / 2, Y: (wH - sy) / 2, H: sy, W: sx}
	}
	menu.animated = false
	rawImg.Free()
	return LOOP_CONT
}

func (menu *ImageMenu) renderer() {
	if menu.shouldReload {
		menu.imageLoader()
		menu.shouldReload = false
	}
	menu.drawNext = menu.animated
	wW, wH := window.GetSize()
	display.Clear()
	if menu.animated {
		data, err := menu.ffmpeg.Read()
		if err == nil {
			menu.image.Update(nil, unsafe.Pointer(&data[0]), int(menu.ffmpeg.w)*3)
		}
	}
	display.Copy(menu.image, nil, menu.pos)
	posIndic, err := font.RenderUTF8Shaded(fmt.Sprintf("%d/%d", menu.Selected+1, len(menu.itemList)), COLOR_BLACK, COLOR_WHITE)
	if err != nil {
		panic(err)
	}
	posInTxt, _ := display.CreateTextureFromSurface(posIndic)
	display.Copy(posInTxt, nil, &sdl.Rect{X: wW - posIndic.W, Y: wH - posIndic.H, H: posIndic.H, W: posIndic.W})
	posIndic.Free()
	posInTxt.Destroy()
	_, _, iW, iH, _ := menu.image.Query()
	posIndic, err = font.RenderUTF8Shaded(fmt.Sprintf("%dx%d", iW, iH), COLOR_BLACK, COLOR_WHITE)
	if err != nil {
		panic(err)
	}
	posInTxt, _ = display.CreateTextureFromSurface(posIndic)
	display.Copy(posInTxt, nil, &sdl.Rect{Y: wH - posIndic.H, H: posIndic.H, W: posIndic.W})
	posIndic.Free()
	posInTxt.Destroy()
}

type TrashMenu struct {
	ImageMenu
}

func makeTrashMenu() *TrashMenu {
	men := makeImageMenu("Trash")
	if men == nil {
		return nil
	}
	return &TrashMenu{ImageMenu: *men}
}

func (men *TrashMenu) keyHandler(key sdl.Keycode) int {
	if key == sdl.K_c {
		return LOOP_CONT
	} else if key == sdl.K_l {
		if displayMessage("Sure to empty trash?\nZ - Yes X - No") {
			if men.animated {
				men.ffmpeg.Destroy()
				men.ffmpeg = nil
			}
			err := os.RemoveAll("Trash")
			if err != nil {
				displayMessage(err.Error())
				return LOOP_CONT
			}
			os.Mkdir("Trash", 0644)
			displayMessage("Trash emptied.")
			return LOOP_EXIT
		}
		men.drawNext = true
		return LOOP_CONT
	}
	return men.ImageMenu.keyHandler(key)
}

type SortMenu struct {
	*ImageMenu
	folders    []string
	folderBar  *sdl.Texture
	folderBarS int
	folderBarE int
	showBar    bool
}

func makeSortMenu(folders []string) *SortMenu {
	men := &SortMenu{ImageMenu: makeImageMenu("Sort"), folders: folders, showBar: len(folders) > 0}
	if men.ImageMenu == nil {
		return nil
	}
	return men
}

func (men *SortMenu) imageLoader() int {
	if len(men.folders) > 0 && men.folderBarE == men.folderBarS {
		men.loadFolderBar(-1)
	}
	return men.ImageMenu.imageLoader()
}

func (men *SortMenu) loadFolderBar(highlight int) {
	keys := []byte{'1', '2', '3', '4', '5', '6', '7', '8', '9', '0', '-', '='}
	var totalLen int32
	pxFmt, _ := window.GetPixelFormat()
	barSurf, err := sdl.CreateRGBSurfaceWithFormat(0, display.GetViewport().W, int32(font.Height())*6/5, 32, pxFmt)
	if err != nil {
		panic(err)
	}
	barSurf.FillRect(nil, 0xFFFFFF)
	lim := men.folderBarS + 12
	men.folderBarE = men.folderBarS
	if lim > len(men.folders) {
		lim = len(men.folders)
	}
	for k, v := range men.folders[men.folderBarS:lim] {
		v = fmt.Sprintf(" %c %s ", keys[k], v)
		fW, _, _ := font.SizeUTF8(v)
		fW32 := int32(fW)
		if fW32+totalLen > display.GetViewport().W {
			break
		}
		if highlight == k {
			// barSurf.FillRect(&sdl.Rect{H: int32(font.Height()), W: fW32, X: totalLen, Y: int32(font.Height()) / 10}, 0xC1DDF3)
			txtSurf, err := font.RenderUTF8Shaded(v, COLOR_BLACK, COLOR_BLUE)
			if err != nil {
				panic(err)
			}
			txtSurf.Blit(nil, barSurf, &sdl.Rect{X: totalLen, Y: int32(font.Height()) / 10})
			txtSurf.Free()
			// drawText(v, barSurf, totalLen, int32(font.Height())/10)
		} else {
			drawText(v, barSurf, totalLen, 0)
		}
		men.folderBarE++
		totalLen += fW32
	}
	barSurf2, err := sdl.CreateRGBSurfaceWithFormat(0, display.GetViewport().W, int32(font.Height())*6/5, 32, pxFmt)
	if err != nil {
		panic(err)
	}
	barSurf2.FillRect(nil, 0xFFFFFF)
	spaces, _, _ := font.SizeUTF8(" ")
	barSurf.Blit(nil, barSurf2, &sdl.Rect{H: int32(font.Height()) * 6 / 5, W: display.GetViewport().W, X: (display.GetViewport().W - totalLen - int32(spaces)) / 2})
	barSurf.Free()
	men.folderBar, err = display.CreateTextureFromSurface(barSurf2)
	if err != nil {
		panic(err)
	}
	barSurf2.Free()
	if highlight != -1 {
		men.folderBarE = men.folderBarS
	}
}

func (men *SortMenu) keyHandler(key sdl.Keycode) int {
	switch key {
	case sdl.K_x:
	case sdl.K_q:
		if !men.showBar {
			men.showBar = true
		} else {
			men.folderBarS = men.folderBarE
			if men.folderBarS >= len(men.folders) {
				men.folderBarS = 0
			}
			men.loadFolderBar(-1)
		}
		men.drawNext = true
	case sdl.K_i:
		men.showBar = !men.showBar
		men.drawNext = true
	case sdl.K_MINUS:
		fallthrough
	case sdl.K_EQUALS:
		fallthrough
	case sdl.K_0:
		fallthrough
	case sdl.K_1:
		fallthrough
	case sdl.K_2:
		fallthrough
	case sdl.K_3:
		fallthrough
	case sdl.K_4:
		fallthrough
	case sdl.K_5:
		fallthrough
	case sdl.K_6:
		fallthrough
	case sdl.K_7:
		fallthrough
	case sdl.K_8:
		fallthrough
	case sdl.K_9:
		if !men.showBar {
			return LOOP_CONT
		}
		var pos int
		switch key {
		case sdl.K_MINUS:
			pos = 10
		case sdl.K_EQUALS:
			pos = 11
		case sdl.K_0:
			pos = 9
		default:
			pos = int(key) - 49
		}
		if men.folderBarS+pos >= men.folderBarE {
			return LOOP_CONT
		}
		targetFldr := men.folders[men.folderBarS+pos]
		men.loadFolderBar(pos)
		return moveFile(men, path.Join(men.fldr, men.itemList[men.Selected]), targetFldr)
	default:
		ret := men.ImageMenu.keyHandler(key)
		if ret == LOOP_CONT && men.showBar {
			display.Copy(men.folderBar, nil, &sdl.Rect{H: int32(font.Height()) * 6 / 5, W: display.GetViewport().W})
			display.Present()
		}
		return ret
	}
	return LOOP_CONT
}

func (menu *SortMenu) renderer() {
	menu.ImageMenu.renderer()
	if menu.showBar {
		display.Copy(menu.folderBar, nil, &sdl.Rect{H: int32(font.Height()) * 6 / 5, W: display.GetViewport().W})
	}
}

func (menu *SortMenu) destroy() {
	menu.image.Destroy()
	menu.folderBar.Destroy()
}
