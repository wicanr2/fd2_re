package main

import (
	"fmt"
	"image"
	"os"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/wicanr2/fd2_re/remake/internal/battle"
	"github.com/wicanr2/fd2_re/remake/internal/fdicon"
	"github.com/wicanr2/fd2_re/remake/internal/fdother"
	"github.com/wicanr2/fd2_re/remake/internal/indexedmap"
)

const (
	nativeAIIdleRecoveryWorkBase  = 0x8088
	nativeAIIdleRecoveryVGAOffset = 0x504
	nativeAIIdleRecoveryWidth     = 312
	nativeAIIdleRecoveryHeight    = 192
	nativeAIIdleRecoveryStride    = 0x1c8
	nativeAIIdleRecoveryVGAStride = 320
)

// nativeAIIdleRecoveryJob owns the asynchronous presentation portion of
// 0x13FD4.  The job presents three indexed frames (the two raw decode/copy
// boundaries plus the final acknowledged frame), one frame per verified
// 0x17AA9(1) wait.  HP is committed only after the third present acknowledgement
// and a fresh raw-record preflight.
type nativeAIIdleRecoveryJob struct {
	actor       *battle.Unit
	plan        battle.NativeAIIdleRecoveryPresentation
	decision    battle.NativeAIIdleRecoveryDecision
	frames      [][]byte
	frame       int
	drawn       bool
	beforeRange int
	beforeHas   bool
	after       func()
}

// beginNativeAIIdleRecovery is the concrete indexed/audio owner for the
// 0x14121→0x13FD4 edge.  It deliberately uses the already verified native map
// compositor, FDICON 24×24 RLE path and FDOTHER#31 index-4 sample.  It does not
// assign a high-level name to either 0x1DA16 decode mode.
func (g *Game) beginNativeAIIdleRecovery(
	actor *battle.Unit, decision battle.NativeAIIdleRecoveryDecision, after func(),
) error {
	if g == nil || g.st == nil || actor == nil {
		return fmt.Errorf("native AI 0x13fd4: runtime state is unavailable")
	}
	if g.nativeAIIdleRecovery != nil {
		return fmt.Errorf("native AI 0x13fd4: presentation is already active")
	}
	if !decision.Accepted {
		return fmt.Errorf("native AI 0x13fd4: caller supplied a rejected raw decision")
	}
	beforeRange, beforeHas := g.st.NativeMapRangeMode, g.st.HasNativeMapRangeModeState
	restoreRange := func() {
		g.st.NativeMapRangeMode = beforeRange
		g.st.HasNativeMapRangeModeState = beforeHas
	}
	records, err := battle.NativeAIScoringRecords(g.st.Units)
	if err != nil {
		return fmt.Errorf("native AI 0x13fd4: raw records: %w", err)
	}
	current, err := battle.PlanNativeAIIdleRecovery(records, len(g.st.Units), unitIndex(g.st, actor))
	if err != nil || current != decision {
		if err != nil {
			return fmt.Errorf("native AI 0x13fd4: raw preflight: %w", err)
		}
		return fmt.Errorf("native AI 0x13fd4: raw decision changed before presentation")
	}
	presentation, err := battle.BuildNativeAIIdleRecoveryPresentation(decision)
	if err != nil {
		return err
	}
	if presentation.CoordinateCall.Address != 0x12D7B ||
		presentation.CoordinateCall.Unit != unitIndex(g.st, actor) ||
		presentation.SampleHandleExpr != "[0x53EEC]" ||
		presentation.SampleIndex != 4 || presentation.SampleLoopCount != 1 {
		return fmt.Errorf("native AI 0x13fd4: raw presentation tuple changed")
	}
	if !osMuteOrShot(g) && (g.sfx == nil || int(presentation.SampleIndex) < 0 ||
		int(presentation.SampleIndex) >= len(g.sfx) || len(g.sfx[int(presentation.SampleIndex)]) == 0) {
		return fmt.Errorf(
			"native AI 0x13fd4: raw sample unavailable handle=%s index=%d",
			presentation.SampleHandleExpr, presentation.SampleIndex,
		)
	}
	if len(g.nativeMapDAC) != 256*3 || !nativeMapAssetsAvailable(g.nativeMapAssets) {
		return fmt.Errorf("native AI 0x13fd4: indexed palette/assets are unavailable")
	}
	// The first proven write is [0x51A83]=0. Apply it before the compositor
	// snapshot so 0x12D7B/0x1DA16 consume the same raw range state; every
	// pre-job failure restores the caller's state.
	g.st.NativeMapRangeMode = 0
	g.st.HasNativeMapRangeModeState = true
	if err := g.composeNativeMapFrame(); err != nil {
		restoreRange()
		return fmt.Errorf("native AI 0x13fd4: source frame: %w", err)
	}
	frames, err := g.buildNativeAIIdleRecoveryFrames(actor, presentation)
	if err != nil {
		restoreRange()
		return err
	}
	if len(frames) != 3 {
		restoreRange()
		return fmt.Errorf("native AI 0x13fd4: frame owner returned %d frames, want 3", len(frames))
	}
	job := &nativeAIIdleRecoveryJob{
		actor: actor, plan: presentation, decision: decision,
		frames: frames, after: after,
		beforeRange: beforeRange, beforeHas: beforeHas,
	}
	// Keep a restoration snapshot so a renderer/runtime failure cannot leak a
	// partially consumed range state.
	g.nativeAIIdleRecovery = job
	if !osMuteOrShot(g) {
		g.playSFX(int(presentation.SampleIndex))
	}
	return nil
}

// buildNativeAIIdleRecoveryFrames consumes the proven 24×24 FDICON raw
// decoder and the corrected 312×192 viewport-copy ABI.  The first frame is a
// raw selector redraw; the second restores the compositor snapshot, matching
// the observed decode/reset call order without naming either visual mode.
func (g *Game) buildNativeAIIdleRecoveryFrames(
	actor *battle.Unit, presentation battle.NativeAIIdleRecoveryPresentation,
) ([][]byte, error) {
	if g == nil || g.st == nil || g.nativeMapAssets == nil || actor == nil {
		return nil, fmt.Errorf("native AI 0x13fd4: frame inputs are unavailable")
	}
	if presentation.FirstDecode.Mode != 2 || presentation.FirstDecode.Tail != 0xfd ||
		presentation.SecondDecode.Mode != 0 || presentation.SecondDecode.Tail != 0 {
		return nil, fmt.Errorf("native AI 0x13fd4: raw decode tuple changed")
	}
	if presentation.CoordinateCall.Address != 0x12D7B ||
		presentation.SampleHandleExpr != "[0x53EEC]" || presentation.SampleIndex != 4 ||
		presentation.SampleLoopCount != 1 || presentation.WaitTicks != [3]uint32{1, 1, 1} {
		return nil, fmt.Errorf("native AI 0x13fd4: raw presentation tuple changed")
	}
	entry, ok := actor.NativeUnitLayerEntry()
	if !ok || g.st.NativeMapSelectorCache == nil {
		return nil, fmt.Errorf("native AI 0x13fd4: actor lacks FDICON selector provenance")
	}
	roster, err := g.st.NativeMapFrameRoster()
	if err != nil {
		return nil, fmt.Errorf("native AI 0x13fd4: frame roster: %w", err)
	}
	cycle, err := fdicon.NativeFrameIndex(
		entry.MotionOffset, entry.ForceBase, roster.Cycles.Idle, roster.Cycles.Moving,
	)
	if err != nil {
		return nil, fmt.Errorf("native AI 0x13fd4: FDICON cycle: %w", err)
	}
	sprite, err := g.nativeMapAssets.Units.SpriteForNativeSlot(
		g.st.NativeMapSelectorCache, entry.Slot, entry.Pose, cycle,
	)
	if err != nil {
		return nil, fmt.Errorf("native AI 0x13fd4: FDICON selector: %w", err)
	}
	view := g.st.NativeMapViewState
	offset, err := fdicon.NativePlacementOffset(
		entry.X, entry.Y, view.CameraX, view.CameraY, entry.Pose,
		entry.MotionOffset, roster.UnitPixelShift, entry.ForceBase,
	)
	if err != nil || offset < 0 {
		if err != nil {
			return nil, fmt.Errorf("native AI 0x13fd4: FDICON placement: %w", err)
		}
		return nil, fmt.Errorf("native AI 0x13fd4: FDICON placement is outside work buffer")
	}
	baseWork := append([]byte(nil), g.nativeMapWork...)
	baseVGA := append([]byte(nil), g.nativeMapVGA...)
	if len(baseWork) != indexedmap.NativeUnitPresentWorkSize || len(baseVGA) != indexedmap.NativeMapVGASize {
		return nil, fmt.Errorf("native AI 0x13fd4: indexed buffers are incomplete")
	}
	firstWork := append([]byte(nil), baseWork...)
	if err := sprite.BlitForNativeFlagsAtOffset(firstWork, nativeAIIdleRecoveryStride, offset, entry.Flags); err != nil {
		return nil, fmt.Errorf("native AI 0x13fd4: first 24x24 decode: %w", err)
	}
	firstVGA, err := nativeAIIdleRecoveryViewport(firstWork, baseVGA)
	if err != nil {
		return nil, err
	}
	secondVGA, err := nativeAIIdleRecoveryViewport(baseWork, baseVGA)
	if err != nil {
		return nil, err
	}
	// The third frame is the acknowledged post-reset copy. Keeping it as a
	// separate buffer makes the three raw wait boundaries observable and avoids
	// committing HP on the second copy's draw callback.
	thirdVGA := append([]byte(nil), secondVGA...)
	return [][]byte{firstVGA, secondVGA, thirdVGA}, nil
}

func nativeAIIdleRecoveryViewport(work, baseVGA []byte) ([]byte, error) {
	if len(work) != indexedmap.NativeUnitPresentWorkSize || len(baseVGA) != indexedmap.NativeMapVGASize {
		return nil, fmt.Errorf("native AI 0x13fd4: viewport buffers are malformed")
	}
	vga := append([]byte(nil), baseVGA...)
	srcEnd := nativeAIIdleRecoveryWorkBase + (nativeAIIdleRecoveryHeight-1)*nativeAIIdleRecoveryStride + nativeAIIdleRecoveryWidth
	dstEnd := nativeAIIdleRecoveryVGAOffset + (nativeAIIdleRecoveryHeight-1)*nativeAIIdleRecoveryVGAStride + nativeAIIdleRecoveryWidth
	if srcEnd > len(work) || dstEnd > len(vga) {
		return nil, fmt.Errorf("native AI 0x13fd4: viewport copy exceeds indexed buffers")
	}
	if err := fdicon.CopyNativeIndexedRegion(
		vga[nativeAIIdleRecoveryVGAOffset:], nativeAIIdleRecoveryVGAStride,
		work[nativeAIIdleRecoveryWorkBase:], nativeAIIdleRecoveryStride,
		nativeAIIdleRecoveryWidth, nativeAIIdleRecoveryHeight,
	); err != nil {
		return nil, fmt.Errorf("native AI 0x13fd4: viewport copy: %w", err)
	}
	return vga, nil
}

func osMuteOrShot(g *Game) bool {
	return g == nil || os.Getenv("FD2_MUTE") != "" || g.shotPath != ""
}

func unitIndex(st *battle.State, actor *battle.Unit) int {
	if st == nil || actor == nil {
		return -1
	}
	for i, unit := range st.Units {
		if unit == actor {
			return i
		}
	}
	return -1
}

func (g *Game) stepNativeAIIdleRecovery() {
	job := g.nativeAIIdleRecovery
	if job == nil || !job.drawn {
		return
	}
	job.drawn = false
	job.frame++
	if job.frame < len(job.frames) {
		return
	}
	idx := unitIndex(g.st, job.actor)
	records, err := battle.NativeAIScoringRecords(g.st.Units)
	if err != nil {
		g.restoreNativeAIIdleRecoveryRange(job)
		g.nativeAIIdleRecovery = nil
		g.loadErr = "native AI 0x13fd4 commit: " + err.Error()
		g.aiBusy = false
		return
	}
	current, err := battle.PlanNativeAIIdleRecovery(records, len(g.st.Units), idx)
	if err != nil || current != job.decision || job.actor.HP != int(job.plan.BeforeHP) {
		if err == nil {
			err = fmt.Errorf("raw record changed during presentation")
		}
		g.restoreNativeAIIdleRecoveryRange(job)
		g.nativeAIIdleRecovery = nil
		g.loadErr = "native AI 0x13fd4 commit: " + err.Error()
		g.aiBusy = false
		return
	}
	job.actor.HP = int(job.plan.AfterHP)
	g.st.NativeMapRangeMode = int(job.plan.AfterGlobalWrite.Value)
	g.st.HasNativeMapRangeModeState = true
	after := job.after
	g.nativeAIIdleRecovery = nil
	if after != nil {
		after()
	}
}

func (g *Game) restoreNativeAIIdleRecoveryRange(job *nativeAIIdleRecoveryJob) {
	if g == nil || job == nil {
		return
	}
	g.st.NativeMapRangeMode = job.beforeRange
	g.st.HasNativeMapRangeModeState = job.beforeHas
}

func (g *Game) drawNativeAIIdleRecovery(screen *ebiten.Image) bool {
	job := g.nativeAIIdleRecovery
	if job == nil || job.frame < 0 || job.frame >= len(job.frames) ||
		g.nativeMapAssets == nil || len(g.nativeMapAssets.Palette) != 256 {
		return false
	}
	palette := g.nativeMapAssets.Palette
	if len(g.nativeMapDAC) == 256*3 {
		if current, err := fdother.VGAPaletteFromDAC(g.nativeMapDAC); err == nil {
			palette = current
		}
	}
	if len(job.frames[job.frame]) != indexedmap.NativeMapVGASize {
		return false
	}
	img := image.NewPaletted(image.Rect(0, 0, 320, 200), palette)
	copy(img.Pix, job.frames[job.frame])
	op := &ebiten.DrawImageOptions{}
	op.GeoM.Scale(2, 2)
	screen.DrawImage(ebiten.NewImageFromImage(img), op)
	job.drawn = true
	return true
}
