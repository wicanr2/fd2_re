package main

import (
	"bytes"
	"image"
	"testing"
	"time"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/wicanr2/fd2_re/remake/internal/battle"
	"github.com/wicanr2/fd2_re/remake/internal/campaign"
)

func TestModernHDGoldRollSnapshotsAndThemeSwitchKeepNativeSchedule(t *testing.T) {
	shared, err := loadNativeClassUIAssets()
	if err != nil {
		t.Fatal(err)
	}
	shops, err := loadNativeShopUIAssets(shared)
	if err != nil {
		t.Fatal(err)
	}
	g := &Game{nativeClassUI: shared, font: loadFont()}
	screen := ebiten.NewImage(logicalW, logicalH)
	for variant, assets := range shops.shops {
		for _, credit := range []bool{false, true} {
			oldGold, amount := 1000, 10
			if credit {
				oldGold, amount = 990, 7
			}
			base, err := campaign.ComposeNativeShopBareScene(assets, shared.digits, oldGold)
			if err != nil {
				t.Fatal(err)
			}
			compose := campaign.ComposeNativeGoldDebitFrames
			if credit {
				compose = campaign.ComposeNativeGoldCreditFrames
			}
			frames, next, err := compose(base, assets.GoldRollStrip, oldGold, amount)
			if err != nil {
				t.Fatal(err)
			}
			d := g.snapshotNativeGoldRoll(assets, oldGold, next, credit)
			if d == nil || len(d.windows) != len(frames) {
				t.Fatalf("variant %d has no window snapshot: strip=%dx%d/%d digit=%dx%d offset=%d,%d indexed=%d mask=%d", variant,
					assets.GoldRollStrip.Width, assets.GoldRollStrip.Height, len(assets.GoldRollStrip.Pixels),
					shared.digits[0].Width, shared.digits[0].Height, shared.digits[0].X, shared.digits[0].Y,
					len(shared.digits[0].Indexed), len(shared.digits[0].Mask))
			}
			called := 0
			job := &nativeClassUIJob{goldText: d, after: func() { called++ }}
			for i, frame := range frames {
				before := append([]byte(nil), frame...)
				hd, runs, ok := d.compose(frame, i, g.font, shared.palette)
				if !ok || len(runs) < 8 || !bytes.Equal(frame, before) {
					t.Fatal("HD altered the indexed source")
				}
				for offset := range frame {
					x, y := offset%320, offset/320
					if (x < 16 || x >= 64 || y < 98 || y >= 107) && hd[offset] != frame[offset] {
						t.Fatal("HD changed pixels outside the gold window")
					}
				}
				for _, run := range runs {
					if run.clip.Dx() != 12 || run.clip.Dy() != 18 || !run.clip.In(image.Rect(32, 196, 128, 214)) {
						t.Fatal("unbounded gold glyph")
					}
				}
				job.timeline = append(job.timeline, nativeClassUITimelineStep{frame: frame, palette: shared.palette, duration: 10 * time.Millisecond})
			}
			g.gold, g.nativeShopUIJob = 12345678, job
			for i := range frames {
				job.elapsed = time.Duration(i)*10*time.Millisecond + time.Millisecond
				for _, hd := range []bool{true, false, true} {
					g.modernHDText = nil
					if hd {
						g.modernHDText = &modernHDTextFrame{}
					}
					if !g.drawNativeShopUIJob(screen) || job.frame != 0 || !job.drawn || called != 0 || g.gold != 12345678 {
						t.Fatal("theme draw changed the transaction or timeline")
					}
					if hd && len(g.modernHDText.runs) < 8 {
						t.Fatal("roll has no HD digits")
					}
				}
			}
			job.elapsed = time.Duration(len(frames)) * 10 * time.Millisecond
			g.drawNativeShopUIJob(screen)
			g.stepNativeShopUILifecycle(time.Now())
			g.stepNativeShopUILifecycle(time.Now())
			if called != 1 || g.nativeShopUIJob != nil {
				t.Fatal("continuation did not execute exactly once")
			}
		}
	}
}

func TestModernHDShopSaleQuestionReplacesCoveredGreeting(t *testing.T) {
	shared, err := loadNativeClassUIAssets()
	if err != nil {
		t.Fatal(err)
	}
	shops, err := loadNativeShopUIAssets(shared)
	if err != nil {
		t.Fatal(err)
	}
	g := &Game{nativeClassUI: shared, nativeShopUI: shops, font: loadFont(), gold: 950,
		nativeShopVariant: 1, nativeShopMode: "sell_confirm", shopSellSlotSel: 3,
		partyJoinOrder: []int{0}, partyRoster: map[int]battle.Unit{0: {
			Inventory: []int{0, 132, 192, 128}, Equipped: []bool{true, true, false, false},
			InventorySlots:       []int{0, 132, 192, 128, 255, 255, 0, 0},
			NativeInventoryFlags: []int{64, 64, 0, 0, 128, 128, 128, 128},
		}}, nativeShopSellItemIDs: []int{0, 132, 192, 128}}
	attachOfficialLocale(t, g)
	_, g.nativeBattleGlyphs, err = loadNativeBattleNameAssets()
	if err != nil {
		t.Fatal(err)
	}
	legacy, ok := g.composeNativeShopSellQuestionBase()
	if !ok {
		t.Fatal("native sale question unavailable")
	}
	g.modernHDText = &modernHDTextFrame{}
	capture := &modernHDIndexedText{palette: shared.palette}
	g.font.indexedHD = capture
	_, ok = g.composeNativeShopSellQuestionBase()
	if !ok || len(capture.runs) == 0 {
		t.Fatal("sale question missing HD glyphs")
	}
	for _, run := range capture.runs {
		if run.text == "歡迎光臨，需要什麼嗎？" || run.text == "歡" {
			t.Fatal("covered greeting leaked over sale question")
		}
	}
	g.modernHDText, g.font.indexedHD = nil, nil
	after, ok := g.composeNativeShopSellQuestionBase()
	if !ok || !bytes.Equal(legacy, after) {
		t.Fatal("HD changed faithful sale question")
	}
}

func TestModernHDGoldRollRejectsBadRecipeWithoutBlockingNativeDraw(t *testing.T) {
	shared, err := loadNativeClassUIAssets()
	if err != nil {
		t.Fatal(err)
	}
	d := &nativeGoldRollText{windows: []campaign.NativeGoldRollWindow{{Rows: [8]int{91}}}}
	g := &Game{nativeClassUI: shared, font: loadFont(), modernHDText: &modernHDTextFrame{}}
	job := &nativeClassUIJob{goldText: d, timeline: []nativeClassUITimelineStep{{frame: make([]byte, 64000), palette: shared.palette, duration: time.Second}}}
	g.nativeShopUIJob = job
	if !g.drawNativeShopUIJob(ebiten.NewImage(logicalW, logicalH)) || !job.drawn || len(g.modernHDText.runs) != 0 {
		t.Fatal("bad recipe did not preserve native fallback")
	}
}
