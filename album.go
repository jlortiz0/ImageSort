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
	"path/filepath"

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
	}
	return menu.ImageMenu.keyHandler(key)
}
