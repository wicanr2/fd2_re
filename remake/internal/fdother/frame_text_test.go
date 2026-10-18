package fdother

import (
	"bytes"
	"testing"
)

func TestTextPlaceholderValidatesAndPreservesDestination(t *testing.T) {
	source := Frame{X: 1, Y: 2, Width: 2, Height: 2, Indexed: []byte{8, 9, 10, 11}, Mask: []byte{255, 255, 255, 255}}
	x, y := -1, -1
	placeholder, err := source.TextPlaceholder(func(px, py int) { x, y = px, py })
	if err != nil {
		t.Fatal(err)
	}
	dst := bytes.Repeat([]byte{77}, 160)
	if err := placeholder.BlitAt(dst, 10, 23, -1); err != nil {
		t.Fatal(err)
	}
	if x != 4 || y != 4 || !bytes.Equal(dst, bytes.Repeat([]byte{77}, 160)) {
		t.Fatal("placeholder changed source coordinates or background")
	}
	if !bytes.Equal(source.Indexed, []byte{8, 9, 10, 11}) {
		t.Fatal("placeholder mutated source")
	}
	source.Mask[0] = 33
	if _, err := source.TextPlaceholder(func(int, int) {}); err == nil {
		t.Fatal("invalid source mask accepted")
	}
}
