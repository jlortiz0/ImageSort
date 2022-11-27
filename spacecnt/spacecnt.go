package main

import (
	"bufio"
	"encoding/binary"
	"flag"
	"fmt"
	"io"
	"os"
	"path"
	"sort"
)

func main() {
	csv := flag.Bool("c", false, "output data in csv format")
	alpha := flag.Bool("a", false, "sort alphabetically instead of by value")
	rev := flag.Bool("r", false, "sort descending instead of ascending")
	pad := flag.Bool("x", false, "include padding total")
	per := flag.Bool("p", false, "show as percent of file per folder")
	nterm := flag.Bool("n", false, "include null terminator as part of folder counts")
	fPath := flag.String("i", "imgSort.cache", "path to cache file")
	flag.Parse()
	f, err := os.Open(*fPath)
	if err != nil {
		panic(err)
	}
	defer f.Close()
	stat, err := f.Stat()
	if err != nil {
		panic(err)
	}
	fSize := float32(stat.Size())
	reader := bufio.NewReader(f)
	hashSize, _ := reader.ReadByte()
	size := int(hashSize)
	size *= size
	size /= 8
	size += 4
	temp := make([]byte, 4)
	_, err = reader.Read(temp)
	if err != nil {
		panic(err)
	}
	entries := binary.BigEndian.Uint32(temp)
	folders := make(map[string]int, entries/128)
	folders["(padding)"] = 5
	var s string
	for {
		s, err = reader.ReadString(0)
		if err != nil {
			break
		}
		fldr := path.Dir(s)
		if *nterm {
			folders[fldr] += len(s)
		} else {
			folders[fldr] += len(s) - 1
			folders["(padding)"]++
		}
		_, err = reader.Discard(size)
		if err != nil {
			panic(err)
		}
		folders[fldr] += size
	}
	if err == io.EOF {
	} else if err != nil {
		panic(err)
	}
	f.Close()
	if !(*pad) {
		delete(folders, "(padding)")
	}
	names := make([]string, 0, len(folders))
	for k := range folders {
		names = append(names, k)
	}
	if *alpha {
		if *rev {
			sort.Sort(sort.Reverse(sort.StringSlice(names)))
		} else {
			sort.Strings(names)
		}
	} else {
		if *rev {
			sort.Slice(names, func(i, j int) bool {
				return folders[names[i]] > folders[names[j]]
			})
		} else {
			sort.Slice(names, func(i, j int) bool {
				return folders[names[i]] < folders[names[j]]
			})
		}
	}
	s = "%-24s"
	if *csv {
		s = "%s,"
	}
	if *per {
		s += "%.2f%%\n"
	} else {
		s += "%d\n"
	}
	for _, v := range names {
		if *per {
			fmt.Printf(s, v, float32(folders[v])/fSize*100)
		} else {
			fmt.Printf(s, v, folders[v])
		}
	}
	if shouldPause() {
		fmt.Println("Press any key to continue...")
		io.CopyN(io.Discard, os.Stdin, 1)
	}
}
