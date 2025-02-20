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
	"path"
	"path/filepath"
	"slices"
	"sort"
	"strconv"
	"strings"

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
	ffmpeg *StreamyWrapper
	fldr   string
	ChoiceMenu
	shouldReload bool
	prevMoveDir  bool
}

var flingOffsets = []int32{36, 43, 51, 62, 77, 95, 120, 152, 196, 255, 336, 449, 610, 840}

func makeImageMenu(fldr string) (*ImageMenu, bool) {
	var entries []os.DirEntry
	var err error
	if config.SizeSort == 0 {
		entries, err = os.ReadDir(fldr)
	} else {
		// Avoid sorting by name if we are going to sort by size anyway.
		// I love pointless microoptimizations!
		var f *os.File
		f, err = os.Open(fldr)
		if err != nil {
			panic(err)
		}
		entries, err = f.ReadDir(0)
		f.Close()
	}
	if err != nil {
		panic(err)
	}
	ls := make([]string, 0, len(entries))
	var srtMap map[string]int64
	if config.SizeSort != 0 {
		srtMap = make(map[string]int64, len(entries))
	}
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
				if config.SizeSort != 0 {
					info, _ := v.Info()
					srtMap[v.Name()] = info.Size()
				}
			}
		}
	}
	if config.SizeSort != 0 {
		sort.Slice(ls, func(i, j int) bool { return srtMap[ls[i]] > srtMap[ls[j]] })
	}
	if config.ReverseSort != 0 {
		slices.Reverse(ls)
	}
	menu := new(ImageMenu)
	menu.fldr = fldr
	menu.itemList = ls
	if len(ls) == 0 {
		var quit bool
		if len(entries) == 0 {
			_, quit = displayMessage("Folder " + fldr + "\nis empty.")
		} else {
			_, quit = displayMessage("Folder " + fldr + "\nhas no supported images.")
		}
		return nil, quit
	}
	display.SetDrawColor(64, 64, 64, 0)
	return menu, false
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
	newName := filepath.Base(from)
	if _, err := os.Stat(filepath.Join(target, newName)); err == nil {
		x := -1
		dLoc := strings.IndexByte(newName, '.')
		before := newName
		var after string
		if dLoc != -1 {
			before = newName[:dLoc]
			after = newName[dLoc+1:]
		}
		for ; err == nil; _, err = os.Stat(filepath.Join(target, fmt.Sprintf("%s_%d.%s", before, x, after))) {
			x++
		}
		newName = fmt.Sprintf("%s_%d.%s", before, x, after)
	}
	os.Rename(from, filepath.Join(target, newName))
	if target != "Trash" {
		hashes[path.Join(target, newName)] = hashes[from]
	}
	delete(hashes, filepath.ToSlash(from))
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
			menu.shouldReload = true
		}
	case sdl.K_RIGHT:
		if menu.Selected < len(menu.itemList)-1 {
			menu.Selected++
			menu.prevMoveDir = false
			menu.shouldReload = true
		}
	case sdl.K_HOME:
		menu.Selected = 0
		menu.prevMoveDir = false
		menu.shouldReload = true
	case sdl.K_END:
		menu.Selected = len(menu.itemList) - 1
		menu.shouldReload = true
	case sdl.K_z:
		stat, _ := os.Stat(filepath.Join(menu.fldr, menu.itemList[menu.Selected]))
		if stat == nil {
			break
		}
		sz := float64(stat.Size()) / 1024
		var quit bool
		if sz > 1024 {
			_, quit = displayMessage(fmt.Sprintf("File: %s\nScale Height: %d\nScale Width: %d\nStorage: %.1f MiB", menu.itemList[menu.Selected], menu.pos.H, menu.pos.W, sz/1024))
		} else {
			_, quit = displayMessage(fmt.Sprintf("File: %s\nScale Height: %d\nScale Width: %d\nStorage: %.1f KiB", menu.itemList[menu.Selected], menu.pos.H, menu.pos.W, sz))
		}
		if quit {
			return LOOP_QUIT
		}
		saveScreen()
		menu.renderer()
		fadeScreen()
	case sdl.K_x:
		return moveFile(menu, filepath.Join(menu.fldr, menu.itemList[menu.Selected]), "Sort")
	case sdl.K_c:
		return moveFile(menu, filepath.Join(menu.fldr, menu.itemList[menu.Selected]), "Trash")
	case sdl.K_F3:
		var sy, sx int32
		wW, wH := window.GetSize()
		_, _, fw, fh, _ := menu.image.Query()
		if fh*wW >= fw*wH {
			sy = wH
			sx = wH * fw / fh
		} else {
			sx = wW
			sy = wW * fh / fw
		}
		menu.pos = &sdl.Rect{X: (wW - sx) / 2, Y: (wH - sy) / 2, H: sy, W: sx}
	case sdl.K_g:
		str := createNewFolder(strconv.Itoa(menu.Selected + 1))
		if str == "\x00" {
			return LOOP_QUIT
		} else if str != "" {
			i, err := strconv.Atoi(str)
			if err == nil && i < len(menu.itemList)+1 && i > 0 {
				menu.Selected = i - 1
				menu.imageLoader()
			}
		}
		// imageLoader will overwrite saveScreen if we call it before
		saveScreen()
		display.SetDrawColor(64, 64, 64, 0)
		menu.renderer()
		fadeScreen()
	case sdl.K_v:
		viewFile(filepath.Join(menu.fldr, menu.itemList[menu.Selected]))
	case sdl.K_h:
		highlightFile(menu.fldr, menu.itemList[menu.Selected])
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
	// On some systems, just trying to blit loading will cause a black screen or flash a previous frame
	// Need to copy fadeFg under it
	saveScreen()
	display.SetRenderTarget(nil)
	wW, wH := window.GetSize()
	display.Copy(fadeFg, nil, &sdl.Rect{W: wW, H: wH})
	display.Copy(loading, nil, &sdl.Rect{W: sx, H: sy, X: wW - sx, Y: wH - sy})
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
		if _, err2 := os.Stat(filepath.Join(menu.fldr, menu.itemList[menu.Selected])); os.IsNotExist(err2) {
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
		menu.image, menu.pos = drawMessage(wordWrapper(err.Error(), []string{"Error loading ", menu.itemList[menu.Selected], ""}))
		return LOOP_CONT
	}
	ind := strings.LastIndexByte(menu.itemList[menu.Selected], '.')
	ext := strings.ToLower(menu.itemList[menu.Selected][ind+1:])
	if ext == "mp4" || ext == "webm" || ext == "mov" || ext == "gif" {
		menu.ffmpeg, err = NewStreamyWrapper(filepath.Join(menu.fldr, menu.itemList[menu.Selected]), 30)
		if err != nil {
			goto Error
		}
		fw, fh := menu.ffmpeg.GetDimensions()
		if fh < 1 || fw < 1 {
			menu.ffmpeg.Destroy()
			err = strconv.ErrRange
			goto Error
		}
		menu.image, err = display.CreateTexture(uint32(sdl.PIXELFORMAT_RGBA32), sdl.TEXTUREACCESS_STREAMING, fw, fh)
		if err != nil {
			menu.image.Destroy()
			menu.ffmpeg.Destroy()
			goto Error
		}
		menu.image.SetBlendMode(sdl.BLENDMODE_BLEND)
		if fh*wW >= fw*wH {
			sy = wH
			sx = wH * fw / fh
		} else {
			sx = wW
			sy = wW * fh / fw
		}
		menu.pos = &sdl.Rect{X: (wW - sx) / 2, Y: (wH - sy) / 2, H: sy, W: sx}
		menu.animated = true
		return LOOP_CONT
	}
	rawImg, err := img.Load(filepath.Join(menu.fldr, menu.itemList[menu.Selected]))
	if err != nil {
		goto Error
	}
	menu.image, _ = display.CreateTextureFromSurface(rawImg)
	if rawImg.H*wW >= rawImg.W*wH {
		sy = wH
		sx = wH * rawImg.W / rawImg.H
	} else {
		sx = wW
		sy = wW * rawImg.H / rawImg.W
	}
	menu.pos = &sdl.Rect{X: (wW - sx) / 2, Y: (wH - sy) / 2, H: sy, W: sx}
	menu.animated = false
	rawImg.Free()
	return LOOP_CONT
}

func minInt32(x, y int32) int32 {
	if x < y {
		return x
	}
	return y
}

func maxInt32(x, y int32) int32 {
	if x < y {
		return y
	}
	return x
}

func clampInt32(x, lower, upper int32) int32 {
	return minInt32(upper, maxInt32(lower, x))
}

const imageMenuMoveAmount = 16
const imageMenuZoomBase = 8

func (menu *ImageMenu) renderer() {
	if menu.shouldReload {
		menu.shouldReload = false
		menu.imageLoader()
	}
	keys := sdl.GetKeyboardState()
	if keys[sdl.SCANCODE_W] != 0 && menu.pos.Y < 0 {
		menu.pos.Y = minInt32(0, menu.pos.Y+imageMenuMoveAmount)
	} else if keys[sdl.SCANCODE_S] != 0 && menu.pos.H > display.GetViewport().H {
		menu.pos.Y = maxInt32(display.GetViewport().H-menu.pos.H, menu.pos.Y-imageMenuMoveAmount)
	}
	if keys[sdl.SCANCODE_A] != 0 && menu.pos.X < 0 {
		menu.pos.X = minInt32(0, menu.pos.X+imageMenuMoveAmount)
	} else if keys[sdl.SCANCODE_D] != 0 && menu.pos.W > display.GetViewport().W {
		menu.pos.X = maxInt32(display.GetViewport().W-menu.pos.W, menu.pos.X-imageMenuMoveAmount)
	}
	if keys[sdl.SCANCODE_UP] != 0 && menu.pos.W < 10000 && menu.pos.H < 10000 {
		vp := display.GetViewport()
		menu.pos.X -= (vp.W/2 - menu.pos.X) / imageMenuZoomBase
		menu.pos.Y -= (vp.H/2 - menu.pos.Y) / imageMenuZoomBase
		menu.pos.W += menu.pos.W / imageMenuZoomBase
		menu.pos.H += menu.pos.H / imageMenuZoomBase
	} else if keys[sdl.SCANCODE_DOWN] != 0 && menu.pos.W > 64 && menu.pos.H > 64 {
		menu.pos.W -= menu.pos.W / (imageMenuZoomBase + 1)
		menu.pos.H -= menu.pos.H / (imageMenuZoomBase + 1)
		vp := display.GetViewport()
		if menu.pos.W < vp.W {
			menu.pos.X = (vp.W - menu.pos.W) / 2
		} else {
			menu.pos.X = clampInt32(menu.pos.X+(vp.W/2-menu.pos.X)/(imageMenuZoomBase+1), vp.W-menu.pos.W, 0)
		}
		if menu.pos.H < vp.H {
			menu.pos.Y = (vp.H - menu.pos.H) / 2
		} else {
			menu.pos.Y = clampInt32(menu.pos.Y+(vp.H/2-menu.pos.Y)/(imageMenuZoomBase+1), vp.H-menu.pos.H, 0)
		}
	}
	display.Clear()
	if menu.animated {
		b, _, err := menu.image.Lock(nil)
		if err == nil {
			// TODO: Error handling
			menu.ffmpeg.Read(b)
			menu.image.Unlock()
		}
	}
	display.Copy(menu.image, nil, menu.pos)
	wW, wH := window.GetSize()
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

func makeTrashMenu() (*TrashMenu, bool) {
	men, quit := makeImageMenu("Trash")
	if men == nil || quit {
		return nil, quit
	}
	return &TrashMenu{ImageMenu: *men}, quit
}

func (men *TrashMenu) keyHandler(key sdl.Keycode) int {
	if key == sdl.K_c {
		return LOOP_CONT
	} else if key == sdl.K_l {
		if b, quit := displayMessage("Okay to empty trash?\nZ - Yes X - No"); b {
			if men.animated {
				men.ffmpeg.Destroy()
				men.ffmpeg = nil
			}
			err := os.RemoveAll("Trash")
			if err == nil {
				os.Mkdir("Trash", 0644)
				if _, quit := displayMessage("Trash emptied."); quit {
					return LOOP_QUIT
				}
				return LOOP_EXIT
			}
			// TODO: Word wrap this error and any others that are directly displayed to the user
			if _, quit := displayMessage(wordWrapper(err.Error(), nil)); quit {
				return LOOP_QUIT
			}
		} else if quit {
			return LOOP_QUIT
		}
		saveScreen()
		men.renderer()
		fadeScreen()
		return LOOP_CONT
	}
	return men.ImageMenu.keyHandler(key)
}

type SortMenu struct {
	*ImageMenu
	folders      []string
	folderBar    *sdl.Texture
	folderBarPos []int
	folderBarInd int
	showBar      bool
}

func makeSortMenu(folders []string) (*SortMenu, bool) {
	innerMenu, quit := makeImageMenu("Sort")
	if innerMenu == nil || quit {
		return nil, quit
	}
	men := &SortMenu{ImageMenu: innerMenu, folders: folders, showBar: len(folders) > 0}
	if men.showBar {
		men.folderBarPos = make([]int, 1, (len(folders)+4)/5+1)
		keys := []byte{'1', '2', '3', '4', '5', '6', '7', '8', '9', '0', '-', '='}
		curPos := 0
		spaces, _, _ := font.SizeUTF8(" ")
		spaces /= 2
		totalLen := int32(spaces)
		for k, v := range folders {
			v = fmt.Sprintf(" %c %s ", keys[curPos], v)
			fW, _, _ := font.SizeUTF8(v)
			fW32 := int32(fW)
			if fW32+totalLen > display.GetViewport().W || curPos+1 == len(keys) {
				if curPos+1 == len(keys) {
					k++
				}
				men.folderBarPos = append(men.folderBarPos, k)
				totalLen = int32(spaces)
				curPos = 0
			}
			curPos++
			totalLen += fW32
		}
		if curPos > 1 || len(folders) == 1 {
			men.folderBarPos = append(men.folderBarPos, len(folders))
		}
	}
	return men, false
}

func (men *SortMenu) imageLoader() int {
	if men.showBar && men.folderBar == nil {
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
	barS := men.folderBarPos[men.folderBarInd]
	barE := men.folderBarPos[men.folderBarInd+1]
	for k, v := range men.folders[barS:barE] {
		v = fmt.Sprintf(" %c %s ", keys[k], v)
		fW, _, _ := font.SizeUTF8(v)
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
		totalLen += int32(fW)
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
}

func (men *SortMenu) keyHandler(key sdl.Keycode) int {
	if key == sdl.K_MINUS || key == sdl.K_EQUALS || (key >= sdl.K_0 && key <= sdl.K_9) {
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
		barS := men.folderBarPos[men.folderBarInd]
		barE := men.folderBarPos[men.folderBarInd+1]
		if barS+pos >= barE {
			return LOOP_CONT
		}
		targetFldr := men.folders[barS+pos]
		men.loadFolderBar(pos)
		ret := moveFile(men, filepath.Join(men.fldr, men.itemList[men.Selected]), targetFldr)
		men.loadFolderBar(-1)
		return ret
	}
	switch key {
	case sdl.K_x:
	case sdl.K_q:
		if !men.showBar {
			men.showBar = true
		} else {
			if sdl.GetModState()&sdl.KMOD_SHIFT != 0 {
				men.folderBarInd--
				if men.folderBarInd < 0 {
					men.folderBarInd = len(men.folderBarPos) - 2
				}
			} else {
				men.folderBarInd++
				if men.folderBarInd >= len(men.folderBarPos)-1 {
					men.folderBarInd = 0
				}
			}
			men.loadFolderBar(-1)
		}
	case sdl.K_i:
		men.showBar = !men.showBar
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
	menu.ImageMenu.destroy()
	menu.folderBar.Destroy()
}
