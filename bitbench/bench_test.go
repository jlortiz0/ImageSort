package bitbench_test

import (
	"bufio"
	"encoding/binary"
	"errors"
	"io"
	"os"
	"testing"

	imgsort "github.com/jlortiz0/ImageSort/bitbench"
)

var hashes []imgsort.Hash

const HashBytes = (imgsort.HASH_SIZE * imgsort.HASH_SIZE) / 8

func loadHashes(b *testing.B) error {
	b.Helper()
	f, err := os.Open("imgSort.cache")
	if err != nil {
		return err
	}
	defer f.Close()
	reader := bufio.NewReader(f)
	sz, _ := reader.ReadByte()
	if sz != imgsort.HASH_SIZE {
		return errors.New("unexpected hash size")
	}
	temp := make([]byte, 4)
	_, err = reader.Read(temp)
	if err != nil {
		return err
	}
	entries := binary.BigEndian.Uint32(temp)
	hashes = make([]imgsort.Hash, entries)
	for i := uint32(0); i < entries; i++ {
		_, err = reader.ReadString(0)
		if err != nil {
			break
		}
		_, err = io.ReadFull(reader, temp)
		if err != nil {
			break
		}
		_, err = io.ReadFull(reader, hashes[i][:])
		if err != nil {
			break
		}
	}
	if err == io.EOF {
		return nil
	}
	return err
}

func loopHelper(b *testing.B, fn func(*imgsort.Hash, *imgsort.Hash) bool) {
	err := loadHashes(b)
	if err != nil {
		b.Fatalf("hash load error: %s", err.Error())
	}
	b.SetBytes(2 * HashBytes)
	b.ResetTimer()
	i := 0
	j := 1
	var _dontOptimizeMe bool
	for f := 0; f < b.N; f++ {
		v := &hashes[i]
		v2 := &hashes[j]
		_dontOptimizeMe = fn(v, v2)
		j += 1
		if j >= len(hashes) {
			i += 1
			if i >= len(hashes)-1 {
				i = 0
			}
			j = i + 1
		}
	}
	b.Log(_dontOptimizeMe)
}

func BenchmarkTable(b *testing.B) {
	loopHelper(b, imgsort.TableDiff)
}

func BenchmarkTable2(b *testing.B) {
	loopHelper(b, imgsort.TableDiff2)
}

func BenchmarkXorAll(b *testing.B) {
	loopHelper(b, imgsort.XorAll)
}

func BenchmarkXorIncr(b *testing.B) {
	loopHelper(b, imgsort.XorIncr)
}

func BenchmarkXorIncr2(b *testing.B) {
	loopHelper(b, imgsort.XorIncr2)
}

func BenchmarkXorIncr3(b *testing.B) {
	loopHelper(b, imgsort.XorIncr3)
}
