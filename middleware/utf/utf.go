package utf

import (
	"unicode/utf8"
)

// EncodeUTF8 is an endian aware UTF16 to UTF8 converter.
func EncodeUTF8(s string) string {
	rs := []rune(s)
	size := 0
	for _, r := range rs {
		size += utf8.RuneLen(r)
	}

	bs := make([]byte, size)

	count := 0
	for _, r := range rs {
		count += utf8.EncodeRune(bs[count:], r)
	}

	return string(bs)
}
