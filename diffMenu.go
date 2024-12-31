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
	"bufio"
	"encoding/binary"
	"errors"
	"fmt"
	"image"
	"io"
	"math/bits"
	"os"
	"path"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"github.com/devedge/imagehash"
	"github.com/veandco/go-sdl2/sdl"
)

// Use path for keys to hashes, not filepath
// This allows them to be portable across OSes
var hashes map[string]hashEntry

type hashEntry struct {
	hash    []byte
	modTime int64
}

type DiffMenu struct {
	image2   *sdl.Texture
	pos2     *sdl.Rect
	ffmpeg2  *ffmpegReader
	diffList [][2]string
	ImageMenu
	imageSel int
}

func makeDiffMenu(fldr string) (*DiffMenu, bool) {
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
	f.Close()
	sort.Strings(ls)
	menu := new(DiffMenu)
	if fldr != "." && len(ls) == 0 {
		var quit bool
		if len(entries) == 0 {
			_, quit = displayMessage("Folder " + fldr + "\nis empty.")
		} else {
			_, quit = displayMessage("Folder " + fldr + "\nhas no supported images.")
		}
		return nil, quit
	}
	menu.itemList = ls
	menu.fldr = fldr
	display.SetDrawColor(64, 64, 64, 0)
	return menu, false
}

type hashErr struct {
	err  error
	path string
}

func (menu *DiffMenu) initDiff() int {
	var ops float32
	texture, rect := drawMessage("Finding duplicates...\nPreparing...")
	display.Clear()
	display.Copy(texture, nil, rect)
	fadeScreen()
	lastUpdate := time.Now()
	lastPump := time.Now()
	diffLs := make([][]byte, len(menu.itemList))
	failed := make([]hashErr, 0, 10)
	var err error
	for k, v := range menu.itemList {
		if menu.fldr == "." {
			diffLs[k], err = getHash(v)
			if os.PathSeparator != '/' {
				menu.itemList[k] = filepath.Join(path.Split(v))
			}
		} else {
			diffLs[k], err = getHash(path.Join(menu.fldr, v))
		}
		if err != nil {
			failed = append(failed, hashErr{err, menu.itemList[k]})
		}
		ops++
		if time.Since(lastUpdate) > time.Second/4 {
			texture.Destroy()
			texture, rect = drawMessage(fmt.Sprintf("Finding duplicates...\nHashing %.1f%%", ops/float32(len(menu.itemList))*100))
			display.Clear()
			display.Copy(texture, nil, rect)
			display.Present()
			lastUpdate = time.Now()
		}
		if time.Since(lastPump) > time.Second/16 {
			event := sdl.PollEvent()
			for event != nil {
				keyEvent, ok := event.(*sdl.KeyboardEvent)
				if ok && keyEvent.Keysym.Sym == sdl.K_ESCAPE {
					return LOOP_EXIT
				}
				event = sdl.PollEvent()
			}
			lastPump = time.Now()
		}
	}
	for i, v := range diffLs {
		j := i + 1
		for j < len(diffLs) {
			if compareBits(v, diffLs[j]) {
				menu.diffList = append(menu.diffList, [2]string{menu.itemList[i], menu.itemList[j]})
			}
			j++
		}
	}
	menu.itemList = make([]string, len(menu.diffList))
	texture.Destroy()
	saveScreen()
	if len(failed) > 0 {
		f, err := os.Create("failed.txt")
		if err != nil {
			_, quit := displayMessage("While writing failed hashes, recieved:\n" + err.Error())
			if quit {
				return LOOP_QUIT
			}
			if len(menu.diffList) == 0 {
				return LOOP_EXIT
			}
			return LOOP_CONT
		}
		buf := bufio.NewWriter(f)
		for _, v := range failed {
			buf.WriteString(v.path)
			buf.WriteByte('\t')
			buf.WriteString(v.err.Error())
			buf.WriteByte('\n')
		}
		buf.Flush()
		f.Close()
		_, quit := displayMessage("Some files failed to hash.\nA list has been written to failed.txt")
		if quit {
			return LOOP_QUIT
		}
	}
	if len(menu.diffList) == 0 {
		return LOOP_EXIT
	}
	return LOOP_CONT
}

func (menu *DiffMenu) keyHandler(key sdl.Keycode) int {
	switch key {
	case sdl.K_u:
		posBak := menu.pos.X
		moveFactor := int32(0)
		for menu.pos.X < display.GetViewport().W {
			menu.pos.X += flingOffsets[moveFactor]
			moveFactor = minInt32(moveFactor+1, int32(len(flingOffsets)-1))
			menu.renderer()
			display.Present()
			delay()
		}
		menu.pos.X = posBak
		menu.image, menu.image2 = menu.image2, menu.image
		menu.pos, menu.pos2 = menu.pos2, menu.pos
		menu.ffmpeg, menu.ffmpeg2 = menu.ffmpeg2, menu.ffmpeg
		moveFactor = int32(0)
		for menu.pos.X > -menu.pos.W {
			menu.pos.X -= flingOffsets[minInt32(moveFactor, int32(len(flingOffsets)-1))]
			moveFactor++
		}
		for moveFactor > 0 {
			moveFactor--
			menu.pos.X += flingOffsets[minInt32(moveFactor, int32(len(flingOffsets)-1))]
			menu.renderer()
			display.Present()
			delay()
		}

		if menu.ffmpeg != nil {
			menu.ffmpeg.Destroy()
			menu.ffmpeg = nil
			menu.animated = true
		}
		if menu.ffmpeg2 != nil {
			menu.ffmpeg2.Destroy()
			menu.ffmpeg2 = nil
		}
		temp := filepath.Join(menu.fldr, fmt.Sprintf("%d.tmp", time.Now().Unix()))
		a := menu.diffList[menu.Selected]
		err := os.Rename(filepath.Join(menu.fldr, a[menu.imageSel]), temp)
		if err != nil {
			break
		}
		err = os.Rename(filepath.Join(menu.fldr, a[menu.imageSel^1]), filepath.Join(menu.fldr, a[menu.imageSel]))
		if err != nil {
			err = os.Rename(temp, filepath.Join(menu.fldr, a[menu.imageSel]))
			if err != nil {
				panic(err)
			}
		}
		err = os.Rename(temp, filepath.Join(menu.fldr, a[menu.imageSel^1]))
		if err != nil {
			panic(err)
		}
		temp2 := hashes[path.Join(menu.fldr, a[menu.imageSel])]
		hashes[path.Join(menu.fldr, a[menu.imageSel])] = hashes[path.Join(menu.fldr, a[menu.imageSel^1])]
		hashes[path.Join(menu.fldr, a[menu.imageSel^1])] = temp2
		if menu.animated {
			menu.shouldReload = true
		}
	case sdl.K_q:
		menu.imageSel ^= 1
		menu.image, menu.image2 = menu.image2, menu.image
		menu.pos, menu.pos2 = menu.pos2, menu.pos
		menu.ffmpeg, menu.ffmpeg2 = menu.ffmpeg2, menu.ffmpeg
		menu.animated = menu.ffmpeg != nil
		menu.itemList[menu.Selected] = menu.diffList[menu.Selected][menu.imageSel]
		if menu.imageSel == 0 {
			display.SetDrawColor(64, 64, 64, 0)
		} else {
			display.SetDrawColor(40, 40, 40, 0)
		}
	case sdl.K_x:
		return moveFile(menu, filepath.Join(menu.fldr, menu.diffList[menu.Selected][menu.imageSel]), "Sort")
	case sdl.K_c:
		return moveFile(menu, filepath.Join(menu.fldr, menu.diffList[menu.Selected][menu.imageSel]), "Trash")
	case sdl.K_g:
		sel := menu.Selected
		ret := menu.ImageMenu.keyHandler(sdl.K_g)
		if menu.Selected != sel {
			menu.imageLoader()
		}
		return ret
	default:
		return menu.ImageMenu.keyHandler(key)
	}
	return LOOP_CONT
}

func (menu *DiffMenu) renderer() {
	if menu.shouldReload {
		menu.imageLoader()
		menu.shouldReload = false
	}
	menu.ImageMenu.renderer()
}

func (menu *DiffMenu) imageLoader() int {
	if len(menu.diffList) == 0 {
		menu.animated = false
		return LOOP_EXIT
	}
	_, err := os.Stat(filepath.Join(menu.fldr, menu.diffList[menu.Selected][0]))
	_, err2 := os.Stat(filepath.Join(menu.fldr, menu.diffList[menu.Selected][1]))
	if os.IsNotExist(err) || os.IsNotExist(err2) {
		if menu.Selected == len(menu.diffList)-1 {
			menu.Selected--
		} else {
			copy(menu.diffList[menu.Selected:], menu.diffList[menu.Selected+1:])
			if menu.prevMoveDir && menu.Selected > 0 {
				menu.Selected--
			}
		}
		menu.diffList = menu.diffList[:len(menu.diffList)-1]
		menu.itemList = menu.itemList[:len(menu.itemList)-1]
		return menu.imageLoader()
	}
	menu.itemList[menu.Selected] = menu.diffList[menu.Selected][menu.imageSel^1]
	menu.ImageMenu.imageLoader()
	menu.image, menu.image2 = menu.image2, menu.image
	menu.ffmpeg, menu.ffmpeg2 = menu.ffmpeg2, menu.ffmpeg
	menu.pos2 = menu.pos
	menu.itemList[menu.Selected] = menu.diffList[menu.Selected][menu.imageSel]
	menu.ImageMenu.imageLoader()
	return LOOP_CONT
}

func (menu *DiffMenu) destroy() {
	menu.image.Destroy()
	menu.image2.Destroy()
}

func makeDiffAllMenu() (*DiffMenu, bool) {
	menu, quit := makeDiffMenu(".")
	if quit {
		return nil, true
	}
	f, err := os.Open(".")
	if err != nil {
		panic(err)
	}
	entries, err := f.ReadDir(0)
	if err != nil {
		panic(err)
	}
	ls := make([]string, 0, len(entries)<<7)
	for _, fldr := range entries {
		if fldr.IsDir() && fldr.Name() != "Trash" && fldr.Name()[0] != '.' && fldr.Name()[0] != '$' {
			f, err := os.Open(fldr.Name())
			if err != nil {
				panic(err)
			}
			entries, err := f.ReadDir(0)
			if err != nil {
				panic(err)
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
						// Have to use path here because filepath will confuse getHash
						ls = append(ls, path.Join(fldr.Name(), v.Name()))
					}
				}
			}
			f.Close()
		}
	}
	f.Close()
	if len(ls) == 0 {
		_, quit = displayMessage("No supported images.")
		return nil, quit
	}
	sort.Strings(ls)
	menu.itemList = ls
	return menu, false
}

func loadHashes() error {
	f, err := os.Open("imgSort.cache")
	if err != nil && errors.Is(err, os.ErrNotExist) {
		hashes = make(map[string]hashEntry, 128)
		return nil
	} else if err != nil {
		return err
	}
	defer f.Close()
	reader := bufio.NewReader(f)
	sz, _ := reader.ReadByte()
	if sz&128 != 0 || sz != config.HashSize {
		hashes = make(map[string]hashEntry, 128)
		return nil
	}
	size := uint16(sz)
	size *= size
	size /= 8
	temp := make([]byte, 4)
	_, err = reader.Read(temp)
	if err != nil {
		return err
	}
	entries := binary.BigEndian.Uint32(temp)
	hashes = make(map[string]hashEntry, entries)
	var s string
	for {
		s, err = reader.ReadString(0)
		if err != nil {
			break
		}
		s = s[:len(s)-1]
		_, err = io.ReadFull(reader, temp)
		if err != nil {
			break
		}
		lModify := int64(binary.BigEndian.Uint32(temp))
		temp2 := make([]byte, size)
		_, err = io.ReadFull(reader, temp2)
		if err != nil {
			break
		}
		hashes[s] = hashEntry{temp2, lModify}
	}
	if err == io.EOF {
		return nil
	}
	return err
}

func saveHashes() error {
	f, err := os.OpenFile("imgSort.cache", os.O_WRONLY|os.O_CREATE|os.O_TRUNC, 0600)
	if err != nil {
		return err
	}
	defer f.Close()
	writer := bufio.NewWriter(f)
	writer.WriteByte(byte(config.HashSize))
	temp := make([]byte, 4)
	binary.BigEndian.PutUint32(temp, uint32(len(hashes)))
	_, err = writer.Write(temp)
	if err != nil {
		return err
	}
	for k, v := range hashes {
		if v.hash == nil {
			continue
		}
		_, err = writer.WriteString(k)
		if err != nil {
			return err
		}
		writer.WriteByte(0)
		binary.BigEndian.PutUint32(temp, uint32(v.modTime))
		_, err = writer.Write(temp)
		if err != nil {
			return err
		}
		_, err = writer.Write(v.hash)
		if err != nil {
			return err
		}
	}
	writer.Flush()
	return nil
}

func getHash(path string) ([]byte, error) {
	hash, ok := hashes[path]
	if ok {
		info, err := os.Stat(path)
		if err == nil && info.ModTime().Unix() == hash.modTime {
			return hash.hash, nil
		}
	}
	var err error
	var img image.Image
	switch strings.ToLower(path[strings.LastIndexByte(path, '.')+1:]) {
	case "mp4":
		fallthrough
	case "webm":
		fallthrough
	case "gif":
		fallthrough
	case "mov":
		img, err = getVideoFrame(path)
	default:
		img, err = imagehash.OpenImg(path)
	}
	if err != nil {
		return nil, err
	}
	hsh, err := imagehash.DhashHorizontal(img, int(config.HashSize))
	if err != nil {
		return nil, err
	}
	info, err := os.Stat(path)
	if err != nil {
		return nil, err
	}
	hashes[path] = hashEntry{hsh, info.ModTime().Unix()}
	return hsh, nil
}

func compareBits(x, y []byte) bool {
	if len(x) != len(y) {
		return false
	}
	var c uint16
	for i := 0; i < len(x); i++ {
		c += uint16(bits.OnesCount8(x[i] ^ y[i]))
		if c > config.HashDiff {
			return false
		}
	}
	return true
}
