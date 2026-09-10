package main

import (
	"math"
	"testing"
)

// TestNormalizeFontScaleSnapsToOffered 釘住設定值會收斂到提供的那幾格：舊設定檔
// 沒有這個欄位（讀出來是 0）要落在預設值，別的值取最接近的一格。
func TestNormalizeFontScaleSnapsToOffered(t *testing.T) {
	for _, tc := range []struct {
		in, want float64
	}{
		{0, defaultFontScale},  // 舊設定檔沒有這個欄位
		{-1, defaultFontScale}, // 手改壞的值
		{1.0, 1.0},
		{0.83, 0.8},
		{1.19, 1.25},
		{5, 1.25}, // 超出範圍取最接近的一格
	} {
		if got := normalizeFontScale(tc.in); got != tc.want {
			t.Fatalf("normalizeFontScale(%.2f) = %.2f，應為 %.2f", tc.in, got, tc.want)
		}
	}
	for _, scale := range fontScales {
		if got := normalizeFontScale(scale); got != scale {
			t.Fatalf("提供的 %.2f 被改成 %.2f", scale, got)
		}
	}
}

// TestFontUserScaleMultipliesAndClamps 釘住字級倍率會乘在呼叫端要的 scale 上，
// 而且夾在可讀範圍內——太小讀不到、太大反而更容易溢出版面。
func TestFontUserScaleMultipliesAndClamps(t *testing.T) {
	f := &Font{base: 18}
	if got := f.effectiveScale(1.2); got != 1.2 {
		t.Fatalf("沒設倍率時 effectiveScale(1.2) = %.3f，應原樣回傳", got)
	}
	f.SetUserScale(0.8)
	if got := f.effectiveScale(1.0); math.Abs(got-0.8) > 1e-9 {
		t.Fatalf("倍率 0.8 時 effectiveScale(1.0) = %.3f", got)
	}
	if got := f.effectiveScale(1.25); math.Abs(got-1.0) > 1e-9 {
		t.Fatalf("倍率要乘在呼叫端的 scale 上，得到 %.3f", got)
	}
	f.SetUserScale(0.1)
	if f.userScale != 0.6 {
		t.Fatalf("過小的倍率沒有被夾住：%.2f", f.userScale)
	}
	f.SetUserScale(9)
	if f.userScale != 1.6 {
		t.Fatalf("過大的倍率沒有被夾住：%.2f", f.userScale)
	}
	// nil 接收者要安全——載入字型失敗時 g.font 就是 nil。
	var missing *Font
	missing.SetUserScale(1.0)
	if got := missing.effectiveScale(1.0); got != 1.0 {
		t.Fatalf("nil 字型的 effectiveScale 回 %.3f", got)
	}
}

// TestCycleFontScaleWrapsAndPersists 釘住 F7 會依序走完那幾格再繞回來，而且套用到
// 兩支字型上。
func TestCycleFontScaleWrapsAndPersists(t *testing.T) {
	t.Setenv("XDG_DATA_HOME", t.TempDir())
	userDataDirCached = ""
	defer func() { userDataDirCached = "" }()

	g := &Game{font: &Font{base: 18}, fontNm: &Font{base: 12}}
	g.applyFontScale(defaultFontScale)
	seen := []float64{}
	for i := 0; i < len(fontScales)+1; i++ {
		g.cycleFontScale()
		seen = append(seen, g.fontScale)
		if g.font.userScale != g.fontScale || g.fontNm.userScale != g.fontScale {
			t.Fatalf("第 %d 次切換後兩支字型的倍率不一致：%.2f／%.2f／%.2f",
				i, g.fontScale, g.font.userScale, g.fontNm.userScale)
		}
	}
	// 切換 len(fontScales) 次會走完一輪，第 len+1 次應該回到第一次切到的那一格。
	if seen[len(seen)-1] != seen[0] {
		t.Fatalf("切換 %d 次之後沒有繞回來：%v", len(fontScales)+1, seen)
	}
	distinct := map[float64]bool{}
	for _, v := range seen[:len(fontScales)] {
		distinct[v] = true
	}
	if len(distinct) != len(fontScales) {
		t.Fatalf("一輪裡沒有走過每一格：%v", seen)
	}
	if got := loadSettings().FontScale; got != g.fontScale {
		t.Fatalf("設定沒有存下來：檔案是 %.2f，目前是 %.2f", got, g.fontScale)
	}
}

// TestFontUserScaleChangesMeasuredWidth 用真字型驗字級確實改變排版寬度——
// effectiveScale 算對了不代表 rasterizer 那端跟著變。英文與日文譯文之所以斷不完，
// 就是因為同一個框裡的可用寬度是固定的，所以這裡量的正是那個會決定斷行的數字。
func TestFontUserScaleChangesMeasuredWidth(t *testing.T) {
	f := loadFont()
	if f == nil {
		t.Skip("這個環境沒有可用的 CJK 字型")
	}
	const sample = "The bandits split up and head for the village."
	base := f.Width(sample, 1.0)
	if base <= 0 {
		t.Fatalf("量到的寬度是 %.1f", base)
	}
	f.SetUserScale(0.8)
	smaller := f.Width(sample, 1.0)
	f.SetUserScale(1.25)
	larger := f.Width(sample, 1.0)
	if !(smaller < base && base < larger) {
		t.Fatalf("字級沒有改變寬度：0.8→%.1f、1.0→%.1f、1.25→%.1f", smaller, base, larger)
	}
	// 縮到 0.8 至少要省下一成寬度，否則對「多塞一行」沒有實質幫助。
	if smaller > base*0.92 {
		t.Fatalf("0.8 只省了 %.1f%%，太少", (1-smaller/base)*100)
	}
}
