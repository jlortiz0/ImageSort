/*
Copyright (C) 2026 jlortiz

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
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/veandco/go-sdl2/sdl"
)

const albumSuffix = ":ALBUM"
const albumsSubdir = "albums"

type AlbumMenu struct {
	ImageMenu
}

func makeAlbumMenu(fldr string, items []string) *AlbumMenu {
	return &AlbumMenu{
		ImageMenu: ImageMenu{
			fldr: fldr,
			ChoiceMenu: ChoiceMenu{
				itemList: items,
			},
		},
	}
}

func (menu *AlbumMenu) keyHandler(key sdl.Keycode) int {
	switch key {
	case sdl.K_LEFTBRACKET:
		if menu.Selected > 0 && len(menu.itemList) > 1 {
			menu.itemList[menu.Selected], menu.itemList[menu.Selected-1] = menu.itemList[menu.Selected-1], menu.itemList[menu.Selected]
			menu.Selected--
		}
	case sdl.K_RIGHTBRACKET:
		if menu.Selected < len(menu.itemList)-1 && len(menu.itemList) > 1 {
			menu.itemList[menu.Selected], menu.itemList[menu.Selected+1] = menu.itemList[menu.Selected+1], menu.itemList[menu.Selected]
			menu.Selected++
		}
	case sdl.K_x:
		if sdl.GetModState()&sdl.KMOD_SHIFT == 0 {
			return LOOP_CONT
		}
	case sdl.K_c:
		if sdl.GetModState()&sdl.KMOD_SHIFT != 0 {
			return menu.ImageMenu.keyHandler(key)
		}
		return moveFile(menu, filepath.Join(menu.fldr, menu.itemList[menu.Selected]), filepath.Dir(filepath.Dir(menu.fldr)))
	case sdl.K_RETURN, sdl.K_RETURN2, sdl.K_KP_ENTER:
		return LOOP_CONT
	case sdl.K_EQUALS, sdl.K_PLUS, sdl.K_KP_PLUS:
		return LOOP_CONT
	}
	return menu.ImageMenu.keyHandler(key)
}

type AlbumSelMenu struct {
	original *sdl.Texture
	ImageMenu
}

func makeAlbumSelMenu(fldr string, albums map[string][]string, original *sdl.Texture) *AlbumSelMenu {
	items := make([]string, 0, len(albums)+1)
	for k := range albums {
		items = append(items, k+albumSuffix)
	}
	sort.Strings(items)
	items = append(items, "")

	return &AlbumSelMenu{
		original: original,
		ImageMenu: ImageMenu{
			fldr: fldr,
			ChoiceMenu: ChoiceMenu{
				itemList: items,
			},
			albums: albums,
		},
	}
}

func (menu *AlbumSelMenu) renderer() {
	if menu.shouldReload {
		menu.shouldReload = false
		menu.imageLoader()
	}
	menu.ImageMenu.renderer()
}

func (menu *AlbumSelMenu) imageLoader() int {
	if menu.image != nil {
		menu.image.Destroy()
	}
	if menu.ffmpeg != nil {
		menu.ffmpeg.Destroy()
		menu.ffmpeg = nil
	}
	if menu.Selected == len(menu.itemList)-1 {
		menu.image, menu.pos = drawMessage("New...")
		return LOOP_CONT
	}
	return menu.ImageMenu.imageLoader()
}

func (menu *AlbumSelMenu) keyHandler(key sdl.Keycode) int {
	switch key {
	case sdl.K_x, sdl.K_c:
		return LOOP_CONT
	case sdl.K_RETURN, sdl.K_RETURN2, sdl.K_KP_ENTER:
		var sy, sx int32
		wW, wH := window.GetSize()
		_, _, fw, fh, _ := menu.original.Query()
		if fh*wW >= fw*wH {
			sy = wH
			sx = wH * fw / fh
		} else {
			sx = wW
			sy = wW * fh / fw
		}
		origPos := sdl.Rect{X: (wW - sx) / 2, Y: wH, H: sy, W: sx}
		const scaleAmount = 8
		targetPos := sdl.Rect{X: (wW - sx/scaleAmount) / 2, Y: (wH - sy/scaleAmount) / 2, H: sy / scaleAmount, W: sx / scaleAmount}

		const maxFactor = 336
		factor := int32(0)
		for x := int32(6); x <= 36; x += 2 {
			factor += x
			menu.renderer()
			pos := sdl.Rect{
				X: int32(float32(factor*(targetPos.X-origPos.X))/maxFactor) + origPos.X,
				Y: int32(float32(factor*(targetPos.Y-origPos.Y))/maxFactor) + origPos.Y,
				W: int32(float32(factor*(targetPos.W-origPos.W))/maxFactor) + origPos.W,
				H: int32(float32(factor*(targetPos.H-origPos.H))/maxFactor) + origPos.H,
			}
			display.Copy(menu.original, nil, &pos)
			display.Present()
			delay()
		}
		return LOOP_REDO
	}
	return menu.ImageMenu.keyHandler(key)
}

func (menu *ImageMenu) moveAlbum(from, to string) {
	// from will have src/albumID
	albumID := strings.TrimSuffix(filepath.Base(from), albumSuffix)
	from = filepath.Dir(from)
	if from != menu.fldr {
		panic(fmt.Sprintf("from not equal, expect %s got %s", menu.fldr, from))
	}
	albumData, ok := menu.albums[albumID]
	if !ok {
		panic(fmt.Sprintf("album %s not found in source %s", albumID, menu.fldr))
	}
	os.Mkdir(filepath.Join(to, albumsSubdir), 0600)

	var destAlbumData map[string]json.RawMessage
	f, err := os.OpenFile(filepath.Join(to, albumsSubdir, "albums.json"), os.O_RDWR|os.O_CREATE, 0600)
	if err != nil {
		panic(fmt.Errorf("failed to open album data for dest %s: %w", to, err))
	}
	fi, _ := f.Stat()
	if fi.Size() > 0 {
		err = json.NewDecoder(f).Decode(&destAlbumData)
		if err != nil {
			panic(fmt.Errorf("failed to decode album data for dest %s: %w", to, err))
		}
	}
	if destAlbumData == nil {
		destAlbumData = make(map[string]json.RawMessage)
	}

	newAlbumID := albumID
	if _, ok := destAlbumData[newAlbumID]; ok {
		x := -1
		for ; ok; _, ok = destAlbumData[fmt.Sprintf("%s_%d", newAlbumID, x)] {
			x++
		}
		newAlbumID = fmt.Sprintf("%s_%d", newAlbumID, x)
	}
	encoded, err := json.Marshal(albumData)
	if err != nil {
		panic(fmt.Errorf("failed to encode album data for dest %s: %w", to, err))
	}
	destAlbumData[newAlbumID] = encoded
	err = os.Rename(filepath.Join(menu.fldr, albumsSubdir, albumID), filepath.Join(to, albumsSubdir, newAlbumID))
	if err != nil {
		panic(fmt.Errorf("failed to rename album folder from %s/%s to %s/%s: %w", menu.fldr, albumID, to, newAlbumID, err))
	}

	f.Seek(0, 0)
	err = json.NewEncoder(f).Encode(destAlbumData)
	if err != nil {
		panic(fmt.Errorf("failed to encode album data for dest %s: %w", to, err))
	}
	err = f.Close()
	if err != nil {
		panic(fmt.Errorf("failed to sync album data for dest %s: %w", to, err))
	}
	delete(menu.albums, albumID)
	menu.albumsDirty = true
}
