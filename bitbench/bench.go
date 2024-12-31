package bitbench

// #include "libpopcnt.h"
// #include "bitcnt.h"
import "C"
import "math/bits"

const HASH_SIZE = 0x0C
const HASH_DIFF = 10

type Hash [HASH_SIZE * HASH_SIZE / 8]byte

var bitsTable [16]uint16 = [16]uint16{
	0, 1, 1, 2, 1, 2, 2, 3,
	1, 2, 2, 3, 2, 3, 3, 4,
}

func TableDiff(h1 *Hash, h2 *Hash) bool {
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

func TableDiff2(h1 *Hash, h2 *Hash) bool {
	var c int
	for k := 0; k < len(h1); k++ {
		c += bits.OnesCount8(h1[k] ^ h2[k])
		if c > HASH_DIFF {
			break
		}
	}
	return c <= HASH_DIFF
}

func XorAll(h1 *Hash, h2 *Hash) bool {
	return int(C.xor_all((*C.uchar)(&h1[0]), (*C.uchar)(&h2[0]), HASH_SIZE)) <= HASH_DIFF
}

func XorIncr(h1 *Hash, h2 *Hash) bool {
	return bool(C.blob_similar((*C.uchar)(&h1[0]), (*C.uchar)(&h2[0]), HASH_SIZE))
}

func XorIncr2(h1 *Hash, h2 *Hash) bool {
	return bool(C.blob_similar_alt((*C.uchar)(&h1[0]), (*C.uchar)(&h2[0]), HASH_SIZE))
}

func XorIncr3(h1 *Hash, h2 *Hash) bool {
	return bool(C.blob_similar_alt2((*C.uchar)(&h1[0]), (*C.uchar)(&h2[0]), HASH_SIZE))
}
