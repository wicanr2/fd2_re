package indexedmap

import (
	"bytes"
	"testing"
)

// TestPhaseBannerBlockIsTheMeasuredPalindrome 釘住 33 步的對稱曲線。
func TestPhaseBannerBlockIsTheMeasuredPalindrome(t *testing.T) {
	want := []int{1, 2, 3, 4, 5, 6, 7, 8, 9, 10, 11, 12, 13, 14, 15, 16, 17,
		16, 15, 14, 13, 12, 11, 10, 9, 8, 7, 6, 5, 4, 3, 2, 1}
	if len(want) != PhaseBannerSteps {
		t.Fatalf("步數 %d，收據是 %d", PhaseBannerSteps, len(want))
	}
	for step, w := range want {
		if got := PhaseBannerBlock(step); got != w {
			t.Fatalf("第 %d 步邊長 %d，收據是 %d", step, got, w)
		}
	}
	for step := 0; step < PhaseBannerSteps; step++ {
		if PhaseBannerBlock(step) != PhaseBannerBlock(PhaseBannerSteps-1-step) {
			t.Fatalf("第 %d 步與其鏡像不對稱", step)
		}
	}
}

// TestMosaicNativeMapViewportTakesBlockTopLeftAndKeepsTheBorder 釘住取樣規則。
func TestMosaicNativeMapViewportTakesBlockTopLeftAndKeepsTheBorder(t *testing.T) {
	src := make([]byte, NativeMapVGASize)
	for i := range src {
		src[i] = byte(i%251 + 1) // 每個像素都不同，取樣規則才有鑑別力
	}
	dst := make([]byte, NativeMapVGASize)
	if err := MosaicNativeMapViewport(dst, src, 8); err != nil {
		t.Fatal(err)
	}
	const originX, originY = 4, 4
	for y := 0; y < viewHeight; y++ {
		for x := 0; x < steadyViewportWidth; x++ {
			want := src[(originY+y/8*8)*viewWidth+originX+x/8*8]
			if got := dst[(originY+y)*viewWidth+originX+x]; got != want {
				t.Fatalf("(%d,%d) = %d，應取方塊左上角 %d", x, y, got, want)
			}
		}
	}
	// 四邊黑邊不得被動到。
	for y := 0; y < 200; y++ {
		for x := 0; x < 320; x++ {
			inside := x >= originX && x < originX+steadyViewportWidth &&
				y >= originY && y < originY+viewHeight
			if inside {
				continue
			}
			if dst[y*viewWidth+x] != src[y*viewWidth+x] {
				t.Fatalf("窗格外 (%d,%d) 被改寫", x, y)
			}
		}
	}
	identity := make([]byte, NativeMapVGASize)
	if err := MosaicNativeMapViewport(identity, src, 1); err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(identity, src) {
		t.Fatal("邊長 1 應等同整幀複製")
	}
}
