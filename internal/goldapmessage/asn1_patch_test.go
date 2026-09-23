package message

import (
	"bytes"
	"encoding/hex"
	"testing"
)

func TestWriteInt64Encoding(t *testing.T) {
	cases := []struct {
		in   int64
		want string // 期望 BER 内容字节（不含 tag/length）
	}{
		{0, "00"},
		{1, "01"},
		{127, "7f"},
		{128, "0080"},
		{255, "00ff"},
		{256, "0100"},
		{32768, "008000"},
		{-1, "ff"},
		{-128, "80"},
		{-129, "ff7f"},
		{100000, "0186a0"},
	}
	for _, c := range cases {
		n := sizeInt64(c.in)
		buf := make([]byte, n)
		b := NewBytes(n, buf)
		wn := writeInt64(b, c.in)
		got := hex.EncodeToString(buf)
		if got != c.want {
			t.Errorf("writeInt64(%d) = %s（%d 字节），应为 %s", c.in, got, n, c.want)
		}
		if wn != sizeInt64(c.in) {
			t.Errorf("size/write 不一致: %d", c.in)
		}
	}
	// 端到端：MessageID 128 写出的完整 INTEGER TLV 应为 02 02 00 80
	buf2 := make([]byte, 8)
	b2 := NewBytes(8, buf2)
	w := MessageID(128)
	s := w.writeTagged(b2, classUniversal, tagInteger)
	got2 := hex.EncodeToString(buf2[8-s:])
	if got2 != "02020080" {
		t.Errorf("MessageID(128) TLV = %s，应为 02020080", got2)
	}
	_ = bytes.MinRead

}
