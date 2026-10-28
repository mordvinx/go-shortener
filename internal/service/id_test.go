package service

import (
	"encoding/base64"
	"encoding/binary"
	"testing"
)

func TestGenerateID(t *testing.T) {
	for _, number := range []uint64{0, 1, 42, 63, 64, 255, 256, 65535, 65536, ^uint64(0)} {
		id, err := GenerateID(number)
		if err != nil {
			t.Fatal(err)
		}
		if len(id) < 6 {
			t.Fatalf("number %d: ID too short: %q", number, id)
		}
		prefix, err := base64.RawURLEncoding.DecodeString(id[:4])
		if err != nil || len(prefix) != 3 {
			t.Fatalf("number %d: invalid random prefix %q", number, id[:4])
		}
		suffix, err := base64.RawURLEncoding.DecodeString(id[4:])
		if err != nil || len(suffix) == 0 || len(suffix) > 8 {
			t.Fatalf("number %d: invalid counter suffix %q", number, id[4:])
		}
		if len(suffix) > 1 && suffix[0] == 0 {
			t.Fatalf("number %d: counter contains a leading zero byte", number)
		}
		var padded [8]byte
		copy(padded[8-len(suffix):], suffix)
		if got := binary.BigEndian.Uint64(padded[:]); got != number {
			t.Fatalf("counter round trip: got %d, want %d", got, number)
		}
	}
}
