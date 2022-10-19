package bitbench

import (
	"bufio"
	"errors"
	"fmt"
	"io"
	"os"
	"unsafe"
)

// #include "libpopcnt.h"
// #include "bitcnt.h"
import "C"

const HASH_SIZE = 0x10
const HASH_DIFF = 64

type Hash [HASH_SIZE * HASH_SIZE / 8]byte

func XorAll(hashes []Hash, log func(...interface{})) {
	for i, v := range hashes {
		j := i + 1
		for j < len(hashes) {
			var temp Hash
			for k := 0; k < len(v); k++ {
				temp[k] = v[k] ^ hashes[j][k]
			}
			if C.popcnt(unsafe.Pointer(&temp[0]), C.ulonglong(len(v))) <= HASH_DIFF {
				log(i, j)
			}
			j++
		}
	}
}

func XorIncr(hashes []Hash, log func(...interface{})) {
	for i, v := range hashes {
		j := i + 1
		for j < len(hashes) {
			if C.blob_similar((*C.uchar)(&v[0]), (*C.uchar)(&hashes[j][0]), C.ulonglong(len(v))) {
				log(i, j)
			}
			j++
		}
	}
}

func XorIncr2(hashes []Hash, log func(...interface{})) {
	for i, v := range hashes {
		j := i + 1
		for j < len(hashes) {
			if C.blob_similar_alt((*C.uchar)(&v[0]), (*C.uchar)(&hashes[j][0]), C.ulonglong(len(v))) {
				log(i, j)
			}
			j++
		}
	}
}

func XorIncr3(hashes []Hash, log func(...interface{})) {
	for i, v := range hashes {
		j := i + 1
		for j < len(hashes) {
			if C.blob_similar_alt2((*C.uchar)(&v[0]), (*C.uchar)(&hashes[j][0]), C.ulonglong(len(v))) {
				log(i, j)
			}
			j++
		}
	}
}

func main() {
	f, err := os.Open("imgSort.cache")
	if err != nil {
		panic(err)
	}
	defer f.Close()
	reader := bufio.NewReader(f)
	sz, _ := reader.ReadByte()
	if sz != HASH_SIZE {
		panic(errors.New("unexpected hash size"))
	}
	temp := make([]byte, 4)
	_, err = reader.Read(temp)
	if err != nil {
		panic(err)
	}
	hashes := make([]Hash, 10)
	for i := uint32(0); i < 10; i++ {
		_, err = reader.ReadString(0)
		if err != nil {
			panic(err)
		}
		_, err = io.ReadFull(reader, temp)
		if err != nil {
			panic(err)
		}
		_, err = io.ReadFull(reader, hashes[i][:])
		if err != nil {
			panic(err)
		}
	}
	XorIncr(hashes, func(args ...interface{}) { fmt.Println(args...) })
}
