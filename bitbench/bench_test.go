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

func loadHashes(B testing.TB) error {
	B.Helper()
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

var bitsTable [16]uint16 = [16]uint16{
	0, 1, 1, 2, 1, 2, 2, 3,
	1, 2, 2, 3, 2, 3, 3, 4,
}

func BenchmarkTable(B *testing.B) {
	err := loadHashes(B)
	if err != nil {
		B.Fatalf("hash load error: %s", err.Error())
	}
	B.ResetTimer()
	for i, v := range hashes {
		j := i + 1
		for j < len(hashes) {
			var c uint16
			for k := 0; k < len(v); k++ {
				temp := v[k] ^ hashes[j][k]
				c += bitsTable[temp&0xf]
				c += bitsTable[temp>>4]
				if c > imgsort.HASH_DIFF {
					break
				}
			}
			if c < imgsort.HASH_DIFF {
				B.Log(i, j)
			}
			j++
		}
	}
}

func BenchmarkXorAll(B *testing.B) {
	err := loadHashes(B)
	if err != nil {
		B.Fatalf("hash load error: %s", err.Error())
	}
	B.ResetTimer()
	imgsort.XorAll(hashes, B.Log)
}

func BenchmarkXorIncr(B *testing.B) {
	err := loadHashes(B)
	if err != nil {
		B.Fatalf("hash load error: %s", err.Error())
	}
	B.ResetTimer()
	imgsort.XorIncr(hashes, B.Log)
}

func BenchmarkXorIncr2(B *testing.B) {
	err := loadHashes(B)
	if err != nil {
		B.Fatalf("hash load error: %s", err.Error())
	}
	B.ResetTimer()
	imgsort.XorIncr2(hashes, B.Log)
}

func BenchmarkXorIncr3(B *testing.B) {
	err := loadHashes(B)
	if err != nil {
		B.Fatalf("hash load error: %s", err.Error())
	}
	B.ResetTimer()
	imgsort.XorIncr3(hashes, B.Log)
}

func TestCount(T *testing.T) {
	err := loadHashes(T)
	if err != nil {
		T.Fatalf("hash load error: %s", err.Error())
	}
	v := hashes[0]
	j := 1
	T.Log(v)
	T.Log(hashes[1])
	var c uint16
	for k := 0; k < len(v); k++ {
		temp := v[k] ^ hashes[j][k]
		c += bitsTable[temp&0xf]
		c += bitsTable[temp>>4]
		// if c > imgsort.HASH_DIFF {
		// 	break
		// }
	}
	T.Log(c)
	T.Fail()
}
