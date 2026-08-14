package streamy

// #cgo pkg-config: libavformat libavcodec libavutil libswscale
// #include <libavutil/error.h>
// #include "stream.h"
// #include <stdlib.h>
import "C"

import (
	"errors"
	"image"
	"strings"
	"unsafe"
)

type AvVideoReader C.LibavReader

func NewAvVideoReader(file string, userAgent string, loop bool) (*AvVideoReader, error) {
	fName := C.CString(file)
	defer C.free(unsafe.Pointer(fName))
	var uAgent *C.char
	if userAgent != "" {
		uAgent = C.CString(userAgent)
		defer C.free(unsafe.Pointer(uAgent))
	}
	var v *C.LibavReader
	code := C.libavreader_new(fName, &v, uAgent, C.bool(loop))
	if code != 0 {
		return nil, errHelper(code)
	}
	return (*AvVideoReader)(v), nil
}

func (v *AvVideoReader) GetDimensions() (int32, int32) {
	out := C.libavreader_dimensions((*C.LibavReader)(v))
	return int32(out.x), int32(out.y)
}

func (v *AvVideoReader) GetFPS() float32 {
	return float32(C.libavreader_fps((*C.LibavReader)(v)))
}

func (v *AvVideoReader) Destroy() error {
	if v != nil {
		C.libavreader_destroy((*C.LibavReader)(v))
	}
	return nil
}

func (v *AvVideoReader) Read(b []uint8) error {
	var code C.int
	if b == nil {
		code = C.libavreader_next((*C.LibavReader)(v), nil)
	} else {
		code = C.libavreader_next((*C.LibavReader)(v), (*C.uint8_t)(&b[0]))
	}
	if code != 0 {
		return errHelper(code)
	}
	return nil
}

func errHelper(code C.int) error {
	var errbuf [C.AV_ERROR_MAX_STRING_SIZE]byte
	C.av_strerror(code, (*C.char)(unsafe.Pointer(&errbuf[0])), C.AV_ERROR_MAX_STRING_SIZE)
	text := string(errbuf[:])
	ind := strings.IndexByte(text, 0)
	if ind != -1 {
		text = text[:ind]
	}
	return errors.New(text)
}

func GetVideoFrame(file string, i int) (*image.RGBA, error) {
	fName := C.CString(file)
	defer C.free(unsafe.Pointer(fName))
	var v *C.LibavReader
	code := C.libavreader_new(fName, &v, nil, false)
	if code != 0 {
		return nil, errHelper(code)
	}
	defer C.libavreader_destroy(v)
	p := C.libavreader_dimensions(v)
	img := image.NewRGBA(image.Rect(0, 0, int(p.x), int(p.y)))
	for j := 0; j < i; j++ {
		code = C.libavreader_next(v, nil)
		if code != 0 {
			return nil, errHelper(code)
		}
	}
	code = C.libavreader_next(v, (*C.uint8_t)(unsafe.Pointer(&img.Pix[0])))
	if code != 0 {
		return nil, errHelper(code)
	}
	return img, nil
}
