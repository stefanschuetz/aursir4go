package aurarath

import "bytes"

const (
	BIN uint8 = iota
	JSON
)

func decode(s *bytes.Buffer, t interface {}, codec string){}
func encode(interface {}) (b *bytes.Buffer) {
	return
}
