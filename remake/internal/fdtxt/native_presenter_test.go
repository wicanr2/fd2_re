package fdtxt

import (
	"bytes"
	"testing"
)

func TestNativePresenterValidatesPreservesBackgroundAndDoesNotMutateFont(t *testing.T) {
	f, err := ParseFont(bytes.Repeat([]byte{255}, GlyphBytes))
	if err != nil {
		t.Fatal(err)
	}
	style := NativeGlyphStyle{Foreground: 205, Shadow: 76, Background: 74}
	calls := 0
	adapter := f.WithNativePresenter(func(stride, base, index int, actual NativeGlyphStyle) bool {
		calls++
		if stride != 32 || base != 33 || index != 0 || actual != style {
			t.Fatal("lost original glyph arguments")
		}
		return true
	})
	dst := bytes.Repeat([]byte{9}, 32*20)
	if err := adapter.BlitNativeGlyph(dst, 32, 33, 0, style); err != nil {
		t.Fatal(err)
	}
	for y := 0; y < 16; y++ {
		for x := 0; x < 16; x++ {
			if dst[33+y*32+x] != 74 {
				t.Fatal("background was suppressed")
			}
		}
	}
	if dst[33+16*32] != 9 || dst[32] != 9 {
		t.Fatal("replacement leaked native shadow")
	}
	if err := adapter.BlitNativeGlyph(dst, 32, 33, 1, style); err == nil {
		t.Fatal("invalid glyph accepted")
	}
	if err := adapter.BlitNativeGlyph(dst[:50], 32, 33, 0, style); err == nil {
		t.Fatal("invalid destination accepted")
	}
	if calls != 1 {
		t.Fatal("presenter ran before validation")
	}
	original := make([]byte, len(dst))
	if err := f.BlitNativeGlyph(original, 32, 33, 0, style); err != nil {
		t.Fatal(err)
	}
	if original[33] != 205 || original[33+16*32] != 76 {
		t.Fatal("adapter mutated original font")
	}
	fallback := f.WithNativePresenter(func(int, int, int, NativeGlyphStyle) bool { return false })
	actual := make([]byte, len(dst))
	if err := fallback.BlitNativeGlyph(actual, 32, 33, 0, style); err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(original, actual) {
		t.Fatal("fallback changed original bytes")
	}
}
