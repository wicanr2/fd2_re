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

// TestPhaseBannerDimFollowsTheBlockCurve 釘住 `sub_11D40` 的減量：level 就是
// 方塊邊長減一，進場 0..15、退場 16..0。
func TestPhaseBannerDimFollowsTheBlockCurve(t *testing.T) {
	if PhaseBannerEnterSteps+PhaseBannerExitSteps != PhaseBannerSteps {
		t.Fatalf("進場 %d ＋ 退場 %d ≠ %d 步",
			PhaseBannerEnterSteps, PhaseBannerExitSteps, PhaseBannerSteps)
	}
	for step := 0; step < PhaseBannerSteps; step++ {
		if got, want := PhaseBannerDim(step), PhaseBannerBlock(step)-1; got != want {
			t.Fatalf("第 %d 步減量 %d，應為邊長減一 %d", step, got, want)
		}
	}
	// 進場最後一步是 sub_1F1CC 的 ebp=16／edi=15；退場第一步是 sub_1F30A 的
	// ebp=0x11／edi=0x10。
	if got := PhaseBannerDim(PhaseBannerEnterSteps - 1); got != 15 {
		t.Fatalf("進場最後一步減量 %d，收據是 15", got)
	}
	if got := PhaseBannerDim(PhaseBannerEnterSteps); got != 16 {
		t.Fatalf("退場第一步減量 %d，收據是 16", got)
	}
	for _, step := range []int{-1, PhaseBannerSteps} {
		if got := PhaseBannerDim(step); got != 0 {
			t.Fatalf("界外第 %d 步減量 %d，應為 0", step, got)
		}
	}
	// 一個 BIOS tick：18.2065097 Hz。
	if PhaseBannerStepMillis < 54.9 || PhaseBannerStepMillis > 54.95 {
		t.Fatalf("每步停留 %.4f 毫秒，應為一個 BIOS tick", PhaseBannerStepMillis)
	}
}

// TestPhaseBannerSlideOffsetsMatchTheCallArguments 釘住 `sub_1F42D` 的七次與
// 五次呼叫參數：x 相對停住位置的偏移就是參數的負值。
func TestPhaseBannerSlideOffsetsMatchTheCallArguments(t *testing.T) {
	wantIn := []int{-100, -75, -50, -25, 0, -1, 0}
	if len(wantIn) != PhaseBannerSlideInSteps {
		t.Fatalf("滑入 %d 步，收據是 %d", PhaseBannerSlideInSteps, len(wantIn))
	}
	for step, want := range wantIn {
		if got := PhaseBannerSlideInOffset(step); got != want {
			t.Fatalf("滑入第 %d 步偏移 %d，收據是 %d", step, got, want)
		}
	}
	wantOut := []int{0, -25, -50, -75, -100}
	if len(wantOut) != PhaseBannerSlideOutSteps {
		t.Fatalf("滑出 %d 步，收據是 %d", PhaseBannerSlideOutSteps, len(wantOut))
	}
	for step, want := range wantOut {
		if got := PhaseBannerSlideOutOffset(step); got != want {
			t.Fatalf("滑出第 %d 步偏移 %d，收據是 %d", step, got, want)
		}
	}
	// 界外回停住的位置，不是憑空的偏移。
	for _, step := range []int{-1, PhaseBannerSlideInSteps} {
		if got := PhaseBannerSlideInOffset(step); got != 0 {
			t.Fatalf("滑入界外第 %d 步回 %d", step, got)
		}
	}
	if PhaseBannerSlideOriginX != 0x55 {
		t.Fatalf("停住的 x 是 %d，原版是 0x55", PhaseBannerSlideOriginX)
	}
}
