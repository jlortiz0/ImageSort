package main

import (
	"bufio"
	"encoding/binary"
	"fmt"
	"io"
	"os"
	"strings"
)

type hashEntry struct {
	hash    []byte
	modTime int64
}

var hashes map[string]hashEntry
var hashSize byte

func main() {
	err := loadHashes()
	if err != nil {
		panic(err)
	}
	dirty := false
	for k, v := range hashes {
		if os.PathSeparator == '\\' && strings.ContainsRune(k, os.PathSeparator) {
			delete(hashes, k)
			k = strings.ReplaceAll(k, string(os.PathSeparator), "/")
			hashes[k] = v
			dirty = true
		}
		info, err := os.Stat(k)
		if err != nil && os.IsNotExist(err) {
			delete(hashes, k)
			fmt.Println(k)
			dirty = true
		} else if err == nil && info.ModTime().Unix() != v.modTime {
			delete(hashes, k)
			fmt.Println(k)
			dirty = true
		}
	}
	if dirty {
		err = saveHashes()
		if err != nil {
			panic(err)
		}
		fmt.Println("Press any key to continue...")
		io.CopyN(io.Discard, os.Stdin, 1)
	}
}

func loadHashes() error {
	f, err := os.Open("imgSort.cache")
	if err != nil {
		return err
	}
	defer f.Close()
	reader := bufio.NewReader(f)
	hashSize, _ = reader.ReadByte()
	size := uint16(hashSize)
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
	writer.WriteByte(hashSize)
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
