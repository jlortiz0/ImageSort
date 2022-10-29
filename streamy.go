package main

import (
	"github.com/jlortiz0/multisav/streamy"
)

type StreamyWrapper struct {
	*streamy.AvVideoReader
	count, target float32
}

func NewStreamyWrapper(name string, fps float32) (*StreamyWrapper, error) {
	rd, err := streamy.NewAvVideoReader(name)
	if err != nil {
		return nil, err
	}
	target := rd.GetFPS() / fps
	return &StreamyWrapper{rd, 1 - target, target}, nil
}

func (v *StreamyWrapper) Read(b []byte) error {
	v.count += v.target
	for v.count >= 2 {
		err := v.AvVideoReader.Read(nil)
		if err != nil {
			return err
		}
		v.count--
	}
	if v.count >= 1 {
		v.count--
		return v.AvVideoReader.Read(b)
	}
	return nil
}
