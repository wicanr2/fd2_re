package main

import (
	"bytes"
	"testing"
	"time"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/wicanr2/fd2_re/remake/internal/battle"
	"github.com/wicanr2/fd2_re/remake/internal/fdicon"
	"github.com/wicanr2/fd2_re/remake/internal/indexedmap"
)

func aiIdleRecoveryTestGame(t *testing.T) (*Game, battle.NativeAIIdleRecoveryDecision) {
	t.Helper()
	g := endRecoveryTestGame(t)
	u := g.st.Units[0]
	u.NativeRecordByte6, u.Camp = 0, battle.Enemy
	u.HasNativeRecordByte34, u.HasNativeRecordByte35, u.HasNativeRecordByte36 = true, true, true
	u.HasNativeRecordWord46 = true
	u.InventorySlots, u.NativeInventoryFlags = make([]int, 8), make([]int, 8)
	u.HasNativeRecordByte8 = true
	records, err := battle.NativeAIScoringRecords(g.st.Units)
	if err != nil {
		t.Fatal(err)
	}
	decision, err := battle.PlanNativeAIIdleRecovery(records, len(g.st.Units), 0)
	if err != nil || !decision.Accepted {
		t.Fatal("AI 回復測試來源", err, decision)
	}
	return g, decision
}

func TestNativeAIIdleRecoveryInitialMaskAndRawRestore(t *testing.T) {
	g, decision := aiIdleRecoveryTestGame(t)
	for i := range g.nativeMapAssets.Units.Sprites {
		g.nativeMapAssets.Units.Sprites[i].Pixels[0] = 0
		g.nativeMapAssets.Units.Sprites[i].Mask[1] = 0
	}
	u := g.st.Units[0]
	u.NativeRecordByte5 = 0x80 // mode0 must bypass the steady-frame gray LUT.
	if err := g.composeNativeMapFrameAt(time.Unix(100, 0)); err != nil {
		t.Fatal(err)
	}
	entry, _ := u.NativeUnitLayerEntry()
	view := g.st.NativeMapViewState
	offset, err := fdicon.NativePlacementOffset(entry.X, entry.Y, view.CameraX, view.CameraY, entry.Pose, entry.MotionOffset, 0, false)
	if err != nil {
		t.Fatal(err)
	}
	const pixel = (4*24-2)*320 + 4 + 4*24
	// A later layer in the initial work snapshot must not replace raw mode0.
	g.nativeMapWork[offset+2], g.nativeMapVGA[pixel+2] = 0x44, 0x44
	before := append([]byte(nil), g.nativeMapVGA...)
	presentation, err := battle.BuildNativeAIIdleRecoveryPresentation(decision)
	if err != nil {
		t.Fatal(err)
	}
	frames, err := g.buildNativeAIIdleRecoveryFrames(u, presentation)
	if err != nil || len(frames) != 3 {
		t.Fatal(err, len(frames))
	}
	if !bytes.Equal(frames[0], before) {
		t.Fatal("第一次 wait 應呈現初始畫面")
	}
	if frames[1][pixel] != 0xc8 || frames[1][pixel+1] != before[pixel+1] || frames[1][pixel+2] != 0xc8 {
		t.Fatal("mode2 未保留 source0 寫入或透明 span")
	}
	if frames[2][pixel] != 0 || frames[2][pixel+2] != 3 {
		t.Fatal("mode0 必須恢復 raw 像素，不用已行動 LUT 或 composer 快照")
	}
}

func TestNativeAIIdleRecoveryThreeDrawsBeforeHPCommit(t *testing.T) {
	g, decision := aiIdleRecoveryTestGame(t)
	u := g.st.Units[0]
	finished := false
	if err := g.beginNativeAIIdleRecovery(u, decision, func() { finished = true }); err != nil {
		t.Fatal(err)
	}
	job := g.nativeAIIdleRecovery
	screen := ebiten.NewImage(640, 400)
	for stage := 0; stage < 3; stage++ {
		g.stepNativeAIIdleRecovery()
		if job.frame != stage || u.HP != 5 || finished {
			t.Fatal("未 Draw 就越過畫面或提交 HP")
		}
		g.Draw(screen)
		if err := g.Update(); err != nil {
			t.Fatal(err)
		}
		if stage < 2 && (u.HP != 5 || finished) {
			t.Fatal("第三張發布前已提交 HP")
		}
	}
	if u.HP != 7 || !finished || g.nativeAIIdleRecovery != nil || g.loadErr != "" {
		t.Fatal("第三張發布後的回復／continuation", u.HP, finished, g.loadErr)
	}
}

func TestNativeAIIdleRecoveryMissingAssetsAndInvalidTupleLeaveHP(t *testing.T) {
	for _, kind := range []string{"asset", "tuple"} {
		t.Run(kind, func(t *testing.T) {
			g, decision := aiIdleRecoveryTestGame(t)
			u := g.st.Units[0]
			beforeRange := g.st.NativeMapRangeMode
			if kind == "asset" {
				g.nativeMapAssets.Units.Sprites = nil
				if err := g.beginNativeAIIdleRecovery(u, decision, nil); err == nil {
					t.Fatal("缺素材仍開始回復")
				}
			} else {
				presentation, _ := battle.BuildNativeAIIdleRecoveryPresentation(decision)
				presentation.FirstDecode.Mode = 0
				if _, err := g.buildNativeAIIdleRecoveryFrames(u, presentation); err == nil {
					t.Fatal("非法 tuple 仍產生畫面")
				}
			}
			if u.HP != 5 || u.Acted || g.nativeAIIdleRecovery != nil || g.st.NativeMapRangeMode != beforeRange {
				t.Fatal("失敗留下 HP／行動／range 交易")
			}
		})
	}
}

func TestNativeAIIdleRecoveryViewportCopiesVerified312x192Region(t *testing.T) {
	work := make([]byte, indexedmap.NativeUnitPresentWorkSize)
	vga := make([]byte, indexedmap.NativeMapVGASize)
	for y := 0; y < nativeAIIdleRecoveryHeight; y++ {
		work[nativeAIIdleRecoveryWorkBase+y*nativeAIIdleRecoveryStride] = byte(y + 1)
	}
	got, err := nativeAIIdleRecoveryViewport(work, vga)
	if err != nil {
		t.Fatal(err)
	}
	for y := 0; y < nativeAIIdleRecoveryHeight; y++ {
		if got[nativeAIIdleRecoveryVGAOffset+y*nativeAIIdleRecoveryVGAStride] != byte(y+1) {
			t.Fatalf("row %d copied byte=%d, want %d", y,
				got[nativeAIIdleRecoveryVGAOffset+y*nativeAIIdleRecoveryVGAStride], y+1)
		}
	}
	if got[nativeAIIdleRecoveryVGAOffset+nativeAIIdleRecoveryWidth-1] != 0 {
		t.Fatalf("copy unexpectedly wrote beyond 312-byte row")
	}
}

func TestNativeAIIdleRecoveryViewportRejectsShortBuffersBeforeWrite(t *testing.T) {
	work := make([]byte, indexedmap.NativeUnitPresentWorkSize-1)
	vga := make([]byte, indexedmap.NativeMapVGASize)
	if _, err := nativeAIIdleRecoveryViewport(work, vga); err == nil {
		t.Fatal("short native work buffer unexpectedly accepted")
	}
}
