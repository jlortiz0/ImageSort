package bitbench

import (
	"unsafe"
)

// #include "libpopcnt.h"
// #include "bitcnt.h"
import "C"

const HASH_SIZE = 0x0C
const HASH_DIFF = 10

type Hash [HASH_SIZE * HASH_SIZE / 8]byte

var bitsTable [16]uint16 = [16]uint16{
	0, 1, 1, 2, 1, 2, 2, 3,
	1, 2, 2, 3, 2, 3, 3, 4,
}

func TableDiff(h1 Hash, h2 Hash) bool {
	var c uint16
	for k := 0; k < len(h1); k++ {
		temp := h1[k] ^ h2[k]
		c += bitsTable[temp&0xf]
		c += bitsTable[temp>>4]
		if c > HASH_DIFF {
			break
		}
	}
	return c <= HASH_DIFF
}

func XorAll(h1 Hash, h2 Hash) bool {
	var temp Hash
	for i := 0; i < len(temp); i++ {
		temp[i] = h1[i] ^ h2[i]
	}
	return C.popcnt(unsafe.Pointer(&temp[0]), C.ulonglong(len(temp))) <= HASH_DIFF
}

func XorIncr(h1 Hash, h2 Hash) bool {
	return bool(C.blob_similar((*C.uchar)(&h1[0]), (*C.uchar)(&h2[0]), C.ulonglong(len(h1))))
}

func XorIncr2(h1 Hash, h2 Hash) bool {
	return bool(C.blob_similar_alt((*C.uchar)(&h1[0]), (*C.uchar)(&h2[0]), C.ulonglong(len(h1))))
}

func XorIncr3(h1 Hash, h2 Hash) bool {
	return bool(C.blob_similar_alt2((*C.uchar)(&h1[0]), (*C.uchar)(&h2[0]), C.ulonglong(len(h1))))
}

func main() {

}
