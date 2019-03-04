package utf

import (
	"testing"
	"unicode/utf8"
)

func Test_UTF16_To_UTF8(t *testing.T) {
	s16 := string([]byte{
		0xff, // BOM
		0xfe, // BOM
		'T',
		0x00,
		'E',
		0x00,
		'S',
		0x00,
		'T',
		0x00,
		0x6C,
		0x34,
		'\n',
		0x00,
	})
	s8 := EncodeUTF8(s16)
	if !utf8.ValidString(s8) {
		t.Errorf("invalid string: %s", s8)
	}
}
