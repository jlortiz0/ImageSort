package main

import (
	"bufio"
	"encoding/binary"
	"fmt"
	"image"
	"io"
	"os"
	"sort"
	"strconv"
	"strings"
	"time"
	"unsafe"

	"github.com/devedge/imagehash"
	"github.com/veandco/go-sdl2/sdl"
)

// #include "libpopcnt.h"
import "C"

var hashes map[string]hashEntry

type hashEntry struct {
	hash    []byte
	modTime int64
}

type DiffMenu struct {
	ImageMenu
	image2   *sdl.Texture
	pos2     *sdl.Rect
	diffList [][2]string
	imageSel int
	ffmpeg2  *ffmpegReader
}

func makeDiffMenu(fldr string) *DiffMenu {
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
		if len(entries) == 0 {
			displayMessage("Folder " + fldr + "\nis empty.")
		} else {
			displayMessage("Folder " + fldr + "\nhas no supported images.")
		}
		return menu
	}
	menu.itemList = ls
	menu.fldr = fldr
	display.SetDrawColor(64, 64, 64, 0)
	return menu
}

func (menu *DiffMenu) initDiff() bool {
	var ops float32
	texture, rect := drawMessage("Finding duplicates...\nPreparing...")
	display.Clear()
	display.Copy(texture, nil, rect)
	fadeScreen()
	lastUpdate := time.Now()
	diffLs := make([][]byte, len(menu.itemList))
	for k, v := range menu.itemList {
		if menu.fldr == "." {
			diffLs[k] = getHash(v)
		} else {
			diffLs[k] = getHash(menu.fldr + string(os.PathSeparator) + v)
		}
		ops++
		if time.Since(lastUpdate) > time.Second/2 {
			texture.Destroy()
			texture, rect = drawMessage(fmt.Sprintf("Finding duplicates...\nHashing %.1f%%", ops/float32(len(menu.itemList))*100))
			display.Clear()
			display.Copy(texture, nil, rect)
			display.Present()
			sdl.PumpEvents()
			lastUpdate = time.Now()
		}
	}
	for i, v := range diffLs {
		j := i + 1
		for j < len(diffLs) {
			if compareBits(v, diffLs[j]) <= config.HashDiff {
				menu.diffList = append(menu.diffList, [2]string{menu.itemList[i], menu.itemList[j]})
			}
			j++
		}
	}
	menu.itemList = make([]string, len(menu.diffList))
	texture.Destroy()
	saveScreen()
	return len(menu.diffList) == 0
}

func (menu *DiffMenu) keyHandler(key sdl.Keycode) int {
	if key == sdl.K_q {
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
		menu.drawNext = true
	} else if key == sdl.K_LEFT && menu.Selected > 0 {
		menu.Selected--
		menu.prevMoveDir = false
		menu.shouldReload = true
		menu.drawNext = true
		return LOOP_CONT
	} else if key == sdl.K_RIGHT && menu.Selected < len(menu.itemList)-1 {
		menu.Selected++
		menu.prevMoveDir = true
		menu.drawNext = true
		menu.shouldReload = true
		return LOOP_CONT
	} else if key == sdl.K_HOME && menu.Selected > 0 {
		menu.Selected = 0
		menu.prevMoveDir = false
		menu.drawNext = true
		menu.shouldReload = true
		return LOOP_CONT
	} else if key == sdl.K_END && menu.Selected < len(menu.itemList) {
		menu.Selected = len(menu.itemList) - 1
		menu.prevMoveDir = true
		menu.drawNext = true
		menu.shouldReload = true
		return LOOP_CONT
	} else if key == sdl.K_x || key == sdl.K_c {
		targetFldr := "Sort" + string(os.PathSeparator)
		if key == sdl.K_c {
			targetFldr = "Trash" + string(os.PathSeparator)
		}
		moveFactor := 0
		for -menu.pos.H < menu.pos.Y && menu.pos.Y < display.GetViewport().H {
			if key == sdl.K_x {
				menu.pos.Y -= flingOffsets[moveFactor]
			} else {
				menu.pos.Y += flingOffsets[moveFactor]
			}
			if moveFactor < len(flingOffsets)-1 {
				moveFactor++
			}
			menu.renderer()
			display.Present()
			delay()
		}
		if menu.ffmpeg != nil {
			menu.ffmpeg.Destroy()
		}
		newName := menu.diffList[menu.Selected][menu.imageSel]
		if menu.fldr == "." {
			newName = newName[strings.LastIndexByte(newName, os.PathSeparator)+1:]
		}
		if _, err := os.Stat(targetFldr + newName); err == nil {
			x := -1
			dLoc := strings.IndexByte(newName, '.')
			before := newName
			var after string
			if dLoc != -1 {
				before = newName[:dLoc]
				after = newName[dLoc+1:]
			}
			for ; err == nil; _, err = os.Stat(fmt.Sprintf("%s%s_%d.%s", targetFldr, before, x, after)) {
				x++
			}
			newName = fmt.Sprintf("%s_%d.%s", before, x, after)
		}
		os.Rename(menu.fldr+string(os.PathSeparator)+menu.diffList[menu.Selected][menu.imageSel], targetFldr+newName)
		if menu.fldr == "." {
			if targetFldr != "Trash"+string(os.PathSeparator) {
				hashes[targetFldr+newName] = hashes[menu.diffList[menu.Selected][menu.imageSel]]
			}
			delete(hashes, menu.diffList[menu.Selected][menu.imageSel])
		} else {
			if targetFldr != "Trash"+string(os.PathSeparator) {
				hashes[targetFldr+newName] = hashes[menu.fldr+string(os.PathSeparator)+menu.diffList[menu.Selected][menu.imageSel]]
			}
			delete(hashes, menu.fldr+string(os.PathSeparator)+menu.diffList[menu.Selected][menu.imageSel])
		}
		ret := menu.imageLoader()
		menu.renderer()
		display.Present()
		return ret
	} else if key == sdl.K_g {
		str := createNewFolder(strconv.Itoa(menu.Selected + 1))
		if str == "CANCEL" {
			return LOOP_QUIT
		} else if str != "" {
			saveScreen()
			i, err := strconv.Atoi(str)
			if err == nil {
				menu.Selected = i - 1
				menu.imageLoader()
			}
			menu.renderer()
			fadeScreen()
		}

	}
	return menu.ImageMenu.keyHandler(key)
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
		return LOOP_EXIT
	}
	_, err := os.Stat(menu.fldr + string(os.PathSeparator) + menu.diffList[menu.Selected][0])
	_, err2 := os.Stat(menu.fldr + string(os.PathSeparator) + menu.diffList[menu.Selected][1])
	if os.IsNotExist(err) || os.IsNotExist(err2) {
		if menu.Selected == len(menu.diffList)-1 {
			menu.Selected--
		} else {
			copy(menu.diffList[menu.Selected:], menu.diffList[menu.Selected+1:])
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

func makeDiffAllMenu() *DiffMenu {
	menu := makeDiffMenu(".")
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
						ls = append(ls, fldr.Name()+string(os.PathSeparator)+v.Name())
					}
				}
			}
			f.Close()
		}
	}
	f.Close()
	if len(ls) == 0 {
		displayMessage("No supported images.")
		return nil
	}
	sort.Strings(ls)
	menu.itemList = ls
	return menu
}

func loadHashes() error {
	f, err := os.Open("imgSort.cache")
	if err != nil && os.IsExist(err) {
		return err
	} else if os.IsNotExist(err) {
		hashes = make(map[string]hashEntry, 128)
		return nil
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

func getHash(path string) []byte {
	hash, ok := hashes[path]
	if ok {
		info, err := os.Stat(path)
		if err == nil && info.ModTime().Unix() == hash.modTime {
			return hash.hash
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
		fmt.Printf("Could not open %s: %s\n", path, err.Error())
		return nil
	}
	hsh, err := imagehash.DhashHorizontal(img, int(config.HashSize))
	if err != nil {
		fmt.Printf("Could not hash %s: %s\n", path, err.Error())
		return nil
	}
	info, err := os.Stat(path)
	if err != nil {
		return nil
	}
	hashes[path] = hashEntry{hsh, info.ModTime().Unix()}
	return hsh
}

// var bitsTable [16]uint16 = [16]uint16{
// 	0, 1, 1, 2, 1, 2, 2, 3,
// 	1, 2, 2, 3, 2, 3, 3, 4,
// }

func compareBits(x, y []byte) uint16 {
	if len(x) != len(y) {
		return 0xFFFF
	}
	// var c uint16
	// for i := 0; i < len(x); i++ {
	// temp := x[i] ^ y[i]
	// c += bitsTable[temp&0xf]
	// c += bitsTable[temp>>4]
	// if c > config.HashDiff {
	// break
	// }
	// }
	// return c
	temp := make([]byte, len(x))
	for i := 0; i < len(x); i++ {
		temp[i] = x[i] ^ y[i]
	}
	return uint16(C.popcnt(unsafe.Pointer(&temp[0]), C.ulonglong(len(x))))
}
