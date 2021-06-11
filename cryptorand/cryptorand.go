package cryptorand

import (
	"crypto/rand"
)

func NewSource() Source {
	return Source{}
}

type Source struct{}

func (Source) Int63() int64 {
	var buf [8]byte
	_, err := rand.Read(buf[:])
	if err != nil {
		panic(err)
	}
	return int64(buf[0]) |
		int64(buf[1])<<8 |
		int64(buf[2])<<16 |
		int64(buf[3])<<24 |
		int64(buf[4])<<32 |
		int64(buf[5])<<40 |
		int64(buf[6])<<48 |
		int64(buf[7]&0x7f)<<56
}

func (Source) Seed(int64) {}
