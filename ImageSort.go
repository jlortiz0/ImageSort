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
	"os"
	"runtime/debug"
	"strings"
	"time"
	"unsafe"

	"github.com/adrg/sysfont"
	"github.com/veandco/go-sdl2/img"
	"github.com/veandco/go-sdl2/sdl"
	"github.com/veandco/go-sdl2/ttf"
)

var COLOR_BLACK = sdl.Color{A: 255}
var COLOR_WHITE = sdl.Color{R: 255, G: 255, B: 255, A: 255}
var COLOR_BLUE = sdl.Color{R: 193, G: 221, B: 243, A: 255}

var window *sdl.Window
var display *sdl.Renderer
var font *ttf.Font
var loading *sdl.Texture

var fHeight int32

var config struct {
	FadeSpeed uint8
	HashDiff  uint16
	HashSize  byte
	AnimFrame byte
}

func main() {
	var err error
	if _, err = os.Stat("jlortiz_TEST"); err == nil {
		os.Chdir("jlortiz_TEST")
	}
	f, err := os.Open("ImgSort.cfg")
	if err != nil {
		config.HashDiff = 12
		config.HashSize = 8
		config.FadeSpeed = 56
	} else {
		data := make([]byte, 512)
		n, err := f.Read(data)
		if err != nil {
			panic(err)
		}
		err = json.Unmarshal(data[:n], &config)
		if err != nil {
			panic(err)
		}
	}
	f.Close()
	err = loadHashes()
	if err != nil {
		panic(err)
	}
	err = sdl.Init(sdl.INIT_TIMER | sdl.INIT_VIDEO)
	if err != nil {
		panic(err)
	}
	defer sdl.Quit()
	sdl.EventState(sdl.MOUSEMOTION, sdl.DISABLE)
	sdl.EventState(sdl.KEYUP, sdl.DISABLE)
	sdl.SetHint(sdl.HINT_RENDER_SCALE_QUALITY, "best")

	initWindow()
	defer window.Destroy()
	defer display.Destroy()
	hideConsole()

	var fontName string
	for _, v := range sysfont.NewFinder(nil).List() {
		if v.Name == "Times New Roman" {
			fontName = v.Filename
		} else if v.Name == "Ubuntu Mono" || strings.HasSuffix(v.Filename, "UbuntuMono-Regular.ttf") {
			fontName = v.Filename
			break
		}
	}
	err = ttf.Init()
	if err != nil {
		panic(err)
	}
	font, err = ttf.OpenFont(fontName, 24)
	if err != nil {
		panic(err)
	}
	fHeight = int32(font.Height()) + 10
	defer font.Close()

	txtSurf, err := font.RenderUTF8Shaded("Loading...", COLOR_BLACK, COLOR_WHITE)
	if err == nil {
		loading, _ = display.CreateTextureFromSurface(txtSurf)
	}
	defer func() {
		tmp := recover()
		err, ok := tmp.(error)
		if !ok {
			return
		}
		if _, err2 := os.Stat("Px437_IBM_VGA_9x16.ttf"); err2 == nil {
			font.Close()
			font, _ = ttf.OpenFont("Px437_IBM_VGA_9x16.ttf", 16)
			fHeight = 20
		}
		display.SetDrawColor(0, 0, 170, 0)
		display.Clear()
		wW, wH := window.GetSize()
		txtSurf, _ := font.RenderUTF8Shaded(" ImageSort ", sdl.Color{B: 170}, sdl.Color{R: 171, G: 169, B: 168})
		pos := sdl.Rect{H: txtSurf.H, W: txtSurf.W, X: wW/2 - txtSurf.W/2, Y: wH / 4}
		txtText, _ := display.CreateTextureFromSurface(txtSurf)
		txtSurf.Free()
		display.Copy(txtText, nil, &pos)
		txtSurf, _ = font.RenderUTF8Shaded("An error has occurred. Press any key to quit.", sdl.Color{R: 171, G: 169, B: 168}, sdl.Color{B: 170})
		pos = sdl.Rect{H: txtSurf.H, W: txtSurf.W, X: wW / 8, Y: wH/4 + 2*fHeight}
		txtText, _ = display.CreateTextureFromSurface(txtSurf)
		txtSurf.Free()
		display.Copy(txtText, nil, &pos)
		txtSurf, _ = font.RenderUTF8Shaded("This information will be saved in crash.log. Attach this file to any bug report.", sdl.Color{R: 171, G: 169, B: 168}, sdl.Color{B: 170})
		pos = sdl.Rect{H: txtSurf.H, W: txtSurf.W, X: wW / 8, Y: wH/4 + 2*fHeight}
		txtText, _ = display.CreateTextureFromSurface(txtSurf)
		txtSurf.Free()
		display.Copy(txtText, nil, &pos)
		txtSurf, _ = font.RenderUTF8Shaded(err.Error(), sdl.Color{R: 171, G: 169, B: 168}, sdl.Color{B: 170})
		pos = sdl.Rect{H: txtSurf.H, W: txtSurf.W, X: wW / 8, Y: wH/4 + 5*fHeight}
		txtText, _ = display.CreateTextureFromSurface(txtSurf)
		txtSurf.Free()
		display.Copy(txtText, nil, &pos)
		stack := string(debug.Stack())
		for i := 0; i < 5; i++ {
			stack = stack[strings.IndexByte(stack, '\n')+1:]
		}
		f, err2 := os.OpenFile("crash.log", os.O_CREATE|os.O_TRUNC|os.O_WRONLY, 0644)
		if err2 == nil {
			f.Write([]byte(err.Error()))
			f.Write([]byte("\n"))
			f.Write([]byte(stack))
		}
		i := int32(6)
		x := strings.IndexByte(stack, '\n')
		for x != -1 {
			if stack[0] == '\t' {
				stack = "    " + stack[1:]
				x += 3
			}
			txtSurf, _ = font.RenderUTF8Solid(stack[:x], sdl.Color{R: 171, G: 169, B: 168})
			pos = sdl.Rect{H: txtSurf.H, W: txtSurf.W, X: wW / 8, Y: wH/4 + i*fHeight}
			txtText, _ = display.CreateTextureFromSurface(txtSurf)
			txtSurf.Free()
			display.Copy(txtText, nil, &pos)
			stack = stack[x+1:]
			i++
			x = strings.IndexByte(stack, '\n')
		}
		if err2 != nil {
			txtSurf, _ = font.RenderUTF8Shaded("Bugger me! Encountered an error generating this screen:", sdl.Color{R: 171, G: 169, B: 168}, sdl.Color{B: 170})
			pos = sdl.Rect{H: txtSurf.H, W: txtSurf.W, X: wW / 8, Y: wH/4 + i*fHeight}
			txtText, _ = display.CreateTextureFromSurface(txtSurf)
			txtSurf.Free()
			display.Copy(txtText, nil, &pos)
			txtSurf, _ = font.RenderUTF8Shaded(err2.Error(), sdl.Color{R: 171, G: 169, B: 168}, sdl.Color{B: 170})
			pos = sdl.Rect{H: txtSurf.H, W: txtSurf.W, X: wW / 8, Y: wH/4 + (i+1)*fHeight}
			txtText, _ = display.CreateTextureFromSurface(txtSurf)
			txtSurf.Free()
			display.Copy(txtText, nil, &pos)
		}
		display.Present()
		for {
			event := sdl.WaitEvent()
			if _, ok := event.(*sdl.KeyboardEvent); ok {
				break
			}
			if _, ok := event.(*sdl.QuitEvent); ok {
				break
			}
		}
	}()
	prevDelay = time.Now()
	beginFldrMenu()
	saveScreen()
	display.SetDrawColor(0, 0, 0, 0)
	display.Clear()
	fadeScreen()
	err = saveHashes()
	if err != nil {
		panic(err)
	}
}

var prevDelay time.Time

func delay() {
	target := time.Since(prevDelay).Milliseconds()
	target = 33 - target
	if target < 0 {
		target = 0
	}
	sdl.Delay(uint32(target))
	prevDelay = time.Now()
}

func drawText(text string, dest *sdl.Surface, x, y int32) {
	txtSurf, err := font.RenderUTF8Shaded(text, COLOR_BLACK, COLOR_WHITE)
	if err != nil {
		panic(err)
	}
	txtSurf.Blit(nil, dest, &sdl.Rect{X: x, Y: y})
	txtSurf.Free()
}

func initWindow() {
	var err error
	window, display, err = sdl.CreateWindowAndRenderer(1024, 768, sdl.WINDOW_SHOWN|sdl.WINDOW_RESIZABLE)
	if err != nil {
		panic(err)
	}
	window.SetTitle("Image Sorter")
	ico := sdl.RWFromFile("photostack.ico", "rb")
	if ico != nil {
		var ico2 *sdl.Surface
		ico2, err = img.LoadICORW(ico)
		if err != nil {
			panic(err)
		}
		ico2.SetColorKey(true, 0xFFFFFF)
		window.SetIcon(ico2)
		ico2.Free()
		ico.Close()
	}
	pxFmt, _ := window.GetPixelFormat()
	fadeFg, _ = display.CreateTexture(pxFmt, sdl.TEXTUREACCESS_STATIC, 1024, 768)
	fadeBg, _ = display.CreateTexture(pxFmt, sdl.TEXTUREACCESS_STATIC, 1024, 768)
}

var fadeFg *sdl.Texture
var fadeBg *sdl.Texture

func saveScreen() {
	pxFmt, _ := window.GetPixelFormat()
	wW, wH := window.GetSize()
	pitch := sdl.BytesPerPixel(pxFmt) * int(wW)
	buffer := make([]byte, pitch*int(wH))
	err := display.ReadPixels(nil, pxFmt, unsafe.Pointer(&buffer[0]), pitch)
	if err != nil {
		panic(err)
	}
	fadeFg.Update(nil, unsafe.Pointer(&buffer[0]), pitch)
}

func fadeScreen() {
	pxFmt, _ := window.GetPixelFormat()
	wW, wH := window.GetSize()
	pitch := sdl.BytesPerPixel(pxFmt) * int(wW)
	buffer := make([]byte, pitch*int(wH))
	err := display.ReadPixels(nil, pxFmt, unsafe.Pointer(&buffer[0]), pitch)
	if err != nil {
		panic(err)
	}
	fadeBg.Update(nil, unsafe.Pointer(&buffer[0]), pitch)
	rect := &sdl.Rect{W: wW, H: wH}
	fadeFg.SetBlendMode(sdl.BLENDMODE_BLEND)
	var i uint8 = 255
	for i > 0 {
		fadeFg.SetAlphaMod(i)
		display.Copy(fadeBg, nil, rect)
		display.Copy(fadeFg, nil, rect)
		display.Present()
		delay()
		if i > config.FadeSpeed {
			i -= config.FadeSpeed
		} else {
			i = 0
		}
	}
	display.Clear()
	display.Copy(fadeBg, nil, rect)
	display.Present()
	sdl.PumpEvents()
	sdl.PeepEvents(make([]sdl.Event, 16), sdl.GETEVENT, sdl.KEYDOWN, sdl.KEYDOWN)
}

const (
	LOOP_CONT = iota
	LOOP_QUIT
	LOOP_EXIT
	LOOP_REDO
)

func stdEventLoop(men Menu) int {
	saveScreen()
	men.renderer()
	fadeScreen()
	// prevDelay = time.Now()
	for {
		delay()
		for event := sdl.PollEvent(); event != nil; event = sdl.PollEvent() {
			switch event := event.(type) {
			case *sdl.QuitEvent:
				return LOOP_QUIT
			case *sdl.KeyboardEvent:
				key := event.Keysym.Sym
				if key == sdl.K_ESCAPE {
					return LOOP_EXIT
				}
				kResp := men.keyHandler(key)
				if kResp != LOOP_CONT {
					return kResp
				}
			case *sdl.WindowEvent:
				if event.Event == sdl.WINDOWEVENT_SIZE_CHANGED {
					pxFmt, _ := window.GetPixelFormat()
					fadeBg.Destroy()
					fadeFg.Destroy()
					fadeFg, _ = display.CreateTexture(pxFmt, sdl.TEXTUREACCESS_TARGET, event.Data1, event.Data2)
					fadeBg, _ = display.CreateTexture(pxFmt, sdl.TEXTUREACCESS_TARGET, event.Data1, event.Data2)
					men.renderer()
					display.Present()
				}
			}
		}
		if men.shouldDraw() {
			men.renderer()
			display.Present()
		}
	}
}

type Menu interface {
	renderer()
	// destroy()
	keyHandler(sdl.Keycode) int
	shouldDraw() bool
}

type ChoiceMenu struct {
	Selected int
	animated bool
	pos      *sdl.Rect
	image    *sdl.Texture
	itemList []string
	drawNext bool
}

func makeMenu(list []string, selected int) *ChoiceMenu {
	men := &ChoiceMenu{Selected: selected, itemList: list}
	var longest string
	for _, v := range list {
		if len(v) > len(longest) {
			longest = v
		}
	}
	longestLen, _, _ := font.SizeUTF8(longest)
	pixFmt, _ := window.GetPixelFormat()
	surf, err := sdl.CreateRGBSurfaceWithFormat(0, int32(longestLen+60), int32(len(list))*fHeight, 24, pixFmt)
	if err != nil {
		panic(err)
	}
	wW, wH := window.GetSize()
	xpos := (wW - surf.W - 60) / 2
	surf.FillRect(nil, 0xFFFFFF)
	canFit := int(wH / fHeight)
	var ypos int32
	if len(list) > canFit {
		vFactor := 1
		if selected+1 > canFit {
			vFactor = canFit - selected - 1
		}
		ypos = int32(vFactor) * fHeight
	} else {
		ypos = (wH - surf.H) / 2
	}
	for i, v := range list {
		drawText(v, surf, 50, int32(i)*fHeight+5)
	}
	men.image, err = display.CreateTextureFromSurface(surf)
	if err != nil {
		panic(err)
	}
	sH, sW := surf.H, surf.W
	men.pos = &sdl.Rect{X: xpos, Y: ypos, H: sH, W: sW}
	surf.Free()
	return men
}

func (men *ChoiceMenu) keyHandler(key sdl.Keycode) int {
	switch key {
	case sdl.K_UP:
		men.Selected--
		men.drawNext = true
		if men.Selected < 0 {
			men.Selected = len(men.itemList) - 1
		}
	case sdl.K_DOWN:
		men.Selected++
		men.drawNext = true
		if men.Selected == len(men.itemList) {
			men.Selected = 0
		}
	case sdl.K_p:
		panic(errors.New("unable to render texture normals"))
	}
	_, wH := window.GetSize()
	canFit := int(wH / fHeight)
	if int32(men.Selected)*fHeight+men.pos.Y < fHeight {
		men.pos.Y = int32(1-men.Selected) * fHeight
	} else if int32(men.Selected+1)*fHeight+men.pos.Y > wH {
		men.pos.Y = int32(canFit-men.Selected-1) * fHeight
	}
	return 0
}

func (menu *ChoiceMenu) renderer() {
	menu.drawNext = false
	display.SetDrawColor(64, 64, 64, 0)
	display.Clear()
	display.Copy(menu.image, nil, menu.pos)
	display.SetDrawColor(0xC1, 0xDD, 0xF3, 0)
	display.FillRect(&sdl.Rect{X: menu.pos.X, Y: int32(menu.Selected)*fHeight + menu.pos.Y, H: fHeight, W: 40})
}

func (menu *ChoiceMenu) destroy() {
	menu.image.Destroy()
}

func (menu *ChoiceMenu) shouldDraw() bool {
	return menu.drawNext
}
