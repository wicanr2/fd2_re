package figani

import (
	"bytes"
	"crypto/sha256"
	"fmt"
	"os"
	"reflect"
	"testing"
)

func TestBlitAtBaseLastRowSpan(t *testing.T) {
	for _, tc := range []struct {
		name       string
		frame      Frame
		length     int
		base       int
		wantWrites map[int]byte
	}{
		{"right-pane", Frame{X: 2, Y: 1, Width: 2, Height: 1, Pixels: []byte{7, 8}, Mask: []byte{1, 0}}, 16, 4, map[int]byte{14: 7}},
		{"left-pane", Frame{X: 2, Y: 1, Width: 2, Height: 1, Pixels: []byte{7, 8}, Mask: []byte{1, 0}}, 16, 0, map[int]byte{10: 7}},
		{"multiple-rows", Frame{X: 1, Width: 2, Height: 2, Pixels: []byte{2, 3, 4, 5}, Mask: []byte{1, 0, 1, 1}}, 16, 4, map[int]byte{5: 2, 13: 4, 14: 5}},
		{"partial-final-row", Frame{X: 1, Width: 2, Height: 2, Pixels: []byte{2, 3, 4, 5}, Mask: []byte{1, 0, 1, 1}}, 15, 4, map[int]byte{5: 2, 13: 4, 14: 5}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			dst := bytes.Repeat([]byte{9}, tc.length)
			want := append([]byte(nil), dst...)
			for offset, value := range tc.wantWrites {
				want[offset] = value
			}
			if err := tc.frame.BlitAtBase(dst, 8, tc.base); err != nil {
				t.Fatal(err)
			}
			if !bytes.Equal(dst, want) {
				t.Fatalf("非opaque來源或鄰格被改寫：got=%v want=%v", dst, want)
			}
		})
	}
}

func TestBlitAtBaseInvalidSpanPreservesDestination(t *testing.T) {
	valid := Frame{X: 1, Width: 2, Height: 2, Pixels: []byte{2, 3, 4, 5}, Mask: []byte{1, 1, 1, 1}}
	maxInt := int(^uint(0) >> 1)
	for _, tc := range []struct {
		name   string
		length int
		stride int
		base   int
		change func(*Frame)
	}{
		{"one-byte-short", 14, 8, 4, func(*Frame) {}},
		{"negative-base", 16, 8, -1, func(*Frame) {}},
		{"empty-at-base", 16, 8, 16, func(*Frame) {}},
		{"base-outside", 16, 8, 17, func(*Frame) {}},
		{"zero-stride", 16, 0, 4, func(*Frame) {}},
		{"negative-x", 16, 8, 4, func(f *Frame) { f.X = -1 }},
		{"negative-y", 16, 8, 4, func(f *Frame) { f.Y = -1 }},
		{"row-overflow", 16, 8, 4, func(f *Frame) { f.X = 7 }},
		{"missing-mask", 16, 8, 4, func(f *Frame) { f.Mask = nil }},
		{"wrong-pixels", 16, 8, 4, func(f *Frame) { f.Pixels = f.Pixels[:3] }},
		{"huge-y", 16, 8, 4, func(f *Frame) { f.Y = maxInt }},
		{"huge-width", 16, 8, 4, func(f *Frame) { f.Width = maxInt }},
		{"huge-height", 16, 8, 4, func(f *Frame) { f.Height = maxInt }},
		{"huge-stride", 16, maxInt, 4, func(*Frame) {}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			frame := valid
			tc.change(&frame)
			dst := bytes.Repeat([]byte{9}, tc.length)
			before := append([]byte(nil), dst...)
			if err := frame.BlitAtBase(dst, tc.stride, tc.base); err == nil || !bytes.Equal(dst, before) {
				t.Fatalf("非法span未拒收或發布部分像素：err=%v dst=%v", err, dst)
			}
		})
	}
}

func TestOriginalFIGANI51FitsRightWorkPane(t *testing.T) {
	const archive = "../../../org_game/炎龍騎士團/FLAME2/FIGANI.DAT"
	raw, err := os.ReadFile(archive)
	if os.IsNotExist(err) {
		t.Skip("player-provided FIGANI.DAT is absent")
	}
	if err != nil {
		t.Fatal(err)
	}
	if fmt.Sprintf("%x", sha256.Sum256(raw)) != "e909314ce37b70701238fd24febc8f61603bbe8d81a9ecbf8f289c97a5b67a1f" {
		t.Fatal("unknown FIGANI.DAT source")
	}
	original, err := DecodeResource(archive, 51)
	if err != nil {
		t.Fatal(err)
	}
	separated, err := LoadSeparatedResource("../../generated-assets/fd2-original-b97caf22/animations", 51)
	if err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(original, separated) {
		t.Fatal("分離FANI51與固定原版不相同")
	}
	first := original.Frames[0]
	if first.X != 184 || first.Y != 35 || first.Width != 109 || first.Height != 165 {
		t.Fatal("FANI51 frame0 geometry與原始header不相同")
	}
	for index, frame := range separated.Frames {
		dst := bytes.Repeat([]byte{9}, 640*200)
		want := append([]byte(nil), dst...)
		// 先在320-wide右半原生畫面形成預期，再核對640-stride staging的全部bytes。
		right := bytes.Repeat([]byte{9}, 320*200)
		if err := frame.BlitAt(right, 320); err != nil {
			t.Fatal(err)
		}
		for y := 0; y < 200; y++ {
			copy(want[y*640+320:y*640+640], right[y*320:(y+1)*320])
		}
		if err := frame.BlitAtBase(dst, 640, 320); err != nil {
			t.Fatalf("frame%d: %v", index, err)
		}
		if !bytes.Equal(dst, want) {
			t.Fatalf("frame%d右半staging與原生畫面不相同", index)
		}
	}
}
