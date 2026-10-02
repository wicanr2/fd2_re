package fdother

import (
	"bytes"
	"os"
	"path/filepath"
	"testing"
)

func TestNativeMapParallaxMatchesBothFixedRawSegments(t *testing.T) {
	root := filepath.Join("..", "..", "generated-assets", "fd2-original-b97caf22")
	var frames [2]Frame
	for i, name := range []string{"FDOTHER_016.bin", "FDOTHER_017.bin"} {
		raw, err := os.ReadFile(filepath.Join(root, "raw", "FDOTHER", name))
		if err != nil {
			t.Skip("使用者原版素材未提供")
		}
		frames[i], err = ParseSingleFrame(raw)
		if err != nil {
			t.Fatal(err)
		}
	}
	want, err := BuildNativeMapParallax(frames[0], frames[1])
	if err != nil {
		t.Fatal(err)
	}
	got, err := LoadSeparatedNativeMapParallax(filepath.Join(root, "surfaces"))
	if err != nil || !bytes.Equal(got.Pixels, want.Pixels) {
		t.Fatalf("分離山景與固定原版解碼不同：%v", err)
	}
	if _, err := LoadSeparatedNativeMapParallax(t.TempDir()); err == nil {
		t.Fatal("缺山景資產卻成功")
	}
	frames[1].Height--
	if _, err := BuildNativeMapParallax(frames[0], frames[1]); err == nil {
		t.Fatal("錯誤雙段高度卻成功")
	}
}

func TestNativeMapParallaxViewportEdgesScrollAndAtomicFailure(t *testing.T) {
	surface := &NativeMapParallaxSurface{Pixels: make([]byte, 462*226)}
	for i := range surface.Pixels {
		surface.Pixels[i] = byte(i*7 + i/462)
	}
	for _, input := range [][4]int{{5, 6, 0, 0}, {37, 17, 0, 0}, {4, 5, -5, -5}, {4, 5, 5, 5}} {
		cx, cy, sx, sy := input[0], input[1], input[2], input[3]
		dst := bytes.Repeat([]byte{253}, 456*191+312)
		if err := BlitNativeMapParallaxViewport(dst, 456, surface, cx, cy, sx, sy); err != nil {
			t.Fatal(err)
		}
		// 原始 signed division 向零截斷，最後一列只需312 bytes。
		for _, row := range []int{0, 112, 113, 191} {
			start := (2*cy+sy/3+row)*462 + 3*cx + sx/2
			if !bytes.Equal(dst[row*456:row*456+312], surface.Pixels[start:start+312]) {
				t.Fatalf("鏡頭／捲動來源錯誤：%v row%d", input, row)
			}
		}
		if dst[312] != 253 {
			t.Fatal("鋪底污染456 stride padding")
		}
	}
	dst := bytes.Repeat([]byte{77}, 456*192)
	before := append([]byte(nil), dst...)
	for _, input := range [][4]int{{0, 0, -6, 0}, {37, 17, 0, 6}, {51, 0, 0, 0}, {0, 18, 0, 0}} {
		if err := BlitNativeMapParallaxViewport(dst, 456, surface, input[0], input[1], input[2], input[3]); err == nil || !bytes.Equal(dst, before) {
			t.Fatalf("越界輸入未原子拒收：%v", input)
		}
	}
}
