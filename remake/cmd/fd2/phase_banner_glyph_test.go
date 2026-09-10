package main

import (
	"path/filepath"
	"testing"

	"github.com/wicanr2/fd2_re/remake/internal/fdother"
	"github.com/wicanr2/fd2_re/remake/internal/indexedmap"
)

// 三塊字樣在 FDOTHER #5 的位置與尺寸。`sub_1F1CC` 敵方傳 0x52、玩家傳 0x50，
// 第二塊固定 0x51。尺寸由分離資產的 bank 雜湊背書，這裡再釘一次，是為了讓
// 「換了 bank 導致 index 位移」直接在這支測試上炸開，而不是在畫面上默默畫錯字。
func TestPhaseBannerGlyphIndexesPointAtTheOriginalCells(t *testing.T) {
	bank, err := fdother.LoadSeparatedLMI1Bank(
		filepath.Clean("../../generated-assets/fd2-original-b97caf22/ui/fdother_005_lmi1_opaque"),
		fdother.NativeCommandHealTailDigitResource,
	)
	if err != nil {
		t.Fatal(err)
	}
	for _, tc := range []struct {
		name          string
		index         int
		width, height int
	}{
		{"PLAYER", phaseBannerGlyphPlayer, 75, 14},
		{"PHASE", phaseBannerGlyphPhase, 63, 14},
		{"ENEMY", phaseBannerGlyphEnemy, 72, 14},
	} {
		if tc.index >= len(bank) {
			t.Fatalf("%s 的 index 0x%02X 超出 bank（%d 格）", tc.name, tc.index, len(bank))
		}
		cell := bank[tc.index]
		if cell.Width != tc.width || cell.Height != tc.height {
			t.Fatalf("%s（0x%02X）是 %d×%d，原版是 %d×%d",
				tc.name, tc.index, cell.Width, cell.Height, tc.width, tc.height)
		}
		// 索引 0 是透明，字身落在 DAC 暗化不動的 0x00..0x0F。整格都不透明或
		// 整格都透明都代表抓錯格。
		clear, opaque := 0, 0
		for _, v := range cell.Pixels {
			if v == 0 {
				clear++
			} else {
				opaque++
			}
		}
		if clear == 0 || opaque == 0 {
			t.Fatalf("%s 的透明像素 %d、不透明 %d，不像字樣", tc.name, clear, opaque)
		}
	}
}

// phaseBannerGlyphFixture 造一組看得出邊界的字樣：整格填同一個非零值，blit
// 之後就能用「哪些像素被改了」反推落點。用假格子而不是真資產，是為了讓落點
// 斷言不依賴資產快取，也不會被真字模的透明區干擾。
func phaseBannerGlyphFixture(banner string) (*Game, []byte) {
	bank := make([]fdother.LMI1Entry, phaseBannerGlyphEnemy+1)
	for i := range bank {
		bank[i] = fdother.LMI1Entry{Width: 1, Height: 1, Pixels: []byte{1}}
	}
	fill := func(index, w, h int, v byte) {
		pixels := make([]byte, w*h)
		for i := range pixels {
			pixels[i] = v
		}
		bank[index] = fdother.LMI1Entry{Width: w, Height: h, Pixels: pixels}
	}
	fill(phaseBannerGlyphPlayer, 75, 14, 0x21)
	fill(phaseBannerGlyphEnemy, 72, 14, 0x21)
	fill(phaseBannerGlyphPhase, 63, 14, 0x22)
	g := &Game{banner: banner, nativeMapAssets: &nativeMapAssets{CommandHealDigits: bank}}
	return g, make([]byte, 320*200)
}

// spanOf 回傳 dst 第 y 列上值為 v 的 x 區間（左閉右閉），沒有就回 (-1,-1)。
func spanOf(dst []byte, y int, v byte) (int, int) {
	first, last := -1, -1
	for x := 0; x < 320; x++ {
		if dst[y*320+x] == v {
			if first < 0 {
				first = x
			}
			last = x
		}
	}
	return first, last
}

// TestPhaseBannerGlyphsLandOnTheOriginalCoordinates 釘住停住時兩塊的落點：
// VGA 的 (0x59,0x56) 與 (0xA9,0x56)。
func TestPhaseBannerGlyphsLandOnTheOriginalCoordinates(t *testing.T) {
	for _, banner := range []string{phaseBannerEnemyText, phaseBannerPlayerText} {
		g, dst := phaseBannerGlyphFixture(banner)
		if !g.blitPhaseBannerGlyphs(dst, 0) {
			t.Fatalf("%s 沒有畫成", banner)
		}
		first, last := spanOf(dst, phaseBannerGlyphY, 0x21)
		wantLast := phaseBannerGlyphFirstX + 71
		if banner == phaseBannerPlayerText {
			wantLast = phaseBannerGlyphFirstX + 74
		}
		if first != phaseBannerGlyphFirstX || last != wantLast {
			t.Fatalf("%s 第一塊落在 x %d..%d，應為 %d..%d",
				banner, first, last, phaseBannerGlyphFirstX, wantLast)
		}
		if f, l := spanOf(dst, phaseBannerGlyphY, 0x22); f != phaseBannerGlyphSecondX ||
			l != phaseBannerGlyphSecondX+62 {
			t.Fatalf("%s 第二塊落在 x %d..%d，應為 %d..%d",
				banner, f, l, phaseBannerGlyphSecondX, phaseBannerGlyphSecondX+62)
		}
		// 上下界：14 列高，超出的列必須乾淨。
		if f, _ := spanOf(dst, phaseBannerGlyphY-1, 0x21); f >= 0 {
			t.Fatalf("%s 畫到了字樣上緣之上", banner)
		}
		if f, _ := spanOf(dst, phaseBannerGlyphY+14, 0x21); f >= 0 {
			t.Fatalf("%s 畫到了字樣下緣之下", banner)
		}
	}
}

// TestPhaseBannerGlyphsSlideTowardEachOther 釘住兩塊是相向移動的：第一塊由左
// 往右、第二塊由右往左，最後在停住位置會合。`sub_1F42D` 的兩個 x 一個是
// `0x55 − 參數`、一個是 `參數 + 0xA5`，套同一個方向就會整排一起平移，那是錯的。
func TestPhaseBannerGlyphsSlideTowardEachOther(t *testing.T) {
	previousFirst, previousSecond := -1, 320
	for step := 0; step < 5; step++ { // 參數 100、75、50、25、0 這五步是單調的
		g, dst := phaseBannerGlyphFixture(phaseBannerEnemyText)
		offset := indexedmap.PhaseBannerSlideInOffset(step)
		if !g.blitPhaseBannerGlyphs(dst, offset) {
			t.Fatalf("第 %d 步沒有畫成", step)
		}
		first, _ := spanOf(dst, phaseBannerGlyphY, 0x21)
		_, secondLast := spanOf(dst, phaseBannerGlyphY, 0x22)
		if first < 0 || secondLast < 0 {
			t.Fatalf("第 %d 步有一塊完全在畫面外：first=%d secondLast=%d",
				step, first, secondLast)
		}
		if first <= previousFirst {
			t.Fatalf("第 %d 步第一塊左緣 %d 沒有往右（前一步 %d）",
				step, first, previousFirst)
		}
		if secondLast >= previousSecond {
			t.Fatalf("第 %d 步第二塊右緣 %d 沒有往左（前一步 %d）",
				step, secondLast, previousSecond)
		}
		previousFirst, previousSecond = first, secondLast
	}
	if previousFirst != phaseBannerGlyphFirstX {
		t.Fatalf("滑入結束時第一塊在 %d，應停在 %d", previousFirst, phaseBannerGlyphFirstX)
	}
	if previousSecond != phaseBannerGlyphSecondX+62 {
		t.Fatalf("滑入結束時第二塊右緣在 %d，應停在 %d",
			previousSecond, phaseBannerGlyphSecondX+62)
	}
}

// TestPhaseBannerFallsBackWithoutTheOriginalBank 釘住反向：拿不到分離素材時
// 不畫原版字樣，改由重製端字型頂著（drawPhaseBanner 那條路）。
func TestPhaseBannerFallsBackWithoutTheOriginalBank(t *testing.T) {
	g, dst := phaseBannerGlyphFixture(phaseBannerEnemyText)
	g.nativeMapAssets.CommandHealDigits = g.nativeMapAssets.CommandHealDigits[:phaseBannerGlyphEnemy]
	if g.phaseBannerGlyphsAvailable() {
		t.Fatal("bank 少一格時仍宣稱有原版字樣")
	}
	if g.blitPhaseBannerGlyphs(dst, 0) {
		t.Fatal("bank 少一格時仍宣稱畫成了")
	}
	for _, v := range dst {
		if v != 0 {
			t.Fatal("失敗即關閉之後畫面仍被動過")
		}
	}
	g.nativeMapAssets = nil
	if g.phaseBannerGlyphsAvailable() || g.blitPhaseBannerGlyphs(dst, 0) {
		t.Fatal("沒有資產時仍宣稱畫得出原版字樣")
	}
}
