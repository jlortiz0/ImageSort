package streamy_test

import (
	"bytes"
	"image"
	"image/png"
	"net/http"
	"os"
	"testing"

	"github.com/jlortiz0/ImageSort/streamy"
)

const GET_FRAME_MAX = 5
const FILE_NAME = "iwonb.webm"

func TestNormalOps(t *testing.T) {
	rd, err := streamy.NewAvVideoReader(FILE_NAME, "", true)
	if err != nil {
		t.Fatal(err)
	}
	sx, sy := rd.GetDimensions()
	t.Logf("Dimensions: %dx%d", sx, sy)
	t.Logf("FPS: %f", rd.GetFPS())
	data := make([]byte, sx*sy*4)
	rd.Read(nil)
	err = rd.Read(data)
	if err != nil {
		t.Fatal(err)
	}
	img := &image.RGBA{Pix: data, Stride: int(sx) * 4, Rect: image.Rectangle{Max: image.Pt(int(sx), int(sy))}}
	f, err := os.OpenFile("out.png", os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0600)
	if err != nil {
		t.Fatal(err)
	}
	err = png.Encode(f, img)
	if err != nil {
		t.Fatal(err)
	}
	f.Close()
	rd.Destroy()
}

func TestErrNotExist(t *testing.T) {
	_, err := streamy.NewAvVideoReader("nonexist.ouch", "", true)
	if err == nil {
		t.FailNow()
	}
	t.Log(err)
}

type SpecificFileHandler string

const USER_AGENT = "interesting string"

func (s SpecificFileHandler) ServeHTTP(w http.ResponseWriter, req *http.Request) {
	if len(req.URL.Path) == 0 {
		http.NotFound(w, req)
		return
	}
	if req.URL.Path[0] == '/' {
		req.URL.Path = req.URL.Path[1:]
	}
	if req.URL.Path != string(s) {
		http.NotFound(w, req)
		return
	}
	if req.Header.Get("User-Agent") != USER_AGENT {
		w.WriteHeader(http.StatusForbidden)
		w.Write([]byte(http.StatusText(http.StatusForbidden)))
		return
	}
	http.ServeFile(w, req, string(s))
}

func TestUserAgent(t *testing.T) {
	srv := &http.Server{Handler: SpecificFileHandler(FILE_NAME), Addr: ":8000"}
	go srv.ListenAndServe()
	rd, err := streamy.NewAvVideoReader("http://localhost:8000/"+FILE_NAME, USER_AGENT, true)
	if err != nil {
		t.Fatal(err)
	}
	err = rd.Read(nil)
	if err != nil {
		t.Fatal(err)
	}
	rd.Destroy()
	srv.Close()
}

func TestGetFrame(t *testing.T) {
	rd, err := streamy.NewAvVideoReader(FILE_NAME, "", true)
	if err != nil {
		t.Fatal(err)
	}
	sx, sy := rd.GetDimensions()
	t.Logf("Dimensions: %dx%d", sx, sy)
	t.Logf("FPS: %f", rd.GetFPS())
	data := make([]byte, sx*sy*4)
	for i := 0; i < GET_FRAME_MAX; i++ {
		err = rd.Read(data)
		if err != nil {
			t.Fatal(err)
		}
		img, err := streamy.GetVideoFrame(FILE_NAME, i)
		if err != nil {
			t.Fatal(err)
		}
		if !bytes.Equal(data, img.Pix) {
			t.Fatalf("frame %d not equal", i)
		}
	}
}
