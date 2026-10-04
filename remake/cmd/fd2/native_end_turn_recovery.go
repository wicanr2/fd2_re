package main

import (
	"fmt"
	"image"
	"time"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/wicanr2/fd2_re/remake/internal/battle"
	"github.com/wicanr2/fd2_re/remake/internal/fdicon"
	"github.com/wicanr2/fd2_re/remake/internal/fdother"
	"github.com/wicanr2/fd2_re/remake/internal/indexedmap"
)

// Evidence/spec: docs/data/ida/fd2_end_turn_recovery_20261004.json (READY).
// 0x1A30B publishes all mode2 candidates together, then all mode0 restores.
// The two zero-tick waits retain Draw boundaries without claiming DOS duration.
type nativeEndTurnRecoveryJob struct {
	plan        []battle.NativeEndTurnRecovery
	entries     []fdicon.NativeUnitLayerEntry
	frames      [3][]byte // initial, mask, raw restore
	restoreWork []byte
	frame       int
	drawn       bool
	readyAt     time.Time
	beforeGate  byte
	cueCount    int
	committed   bool
	after       func()
}

func (g *Game) beginNativeEndTurnRecovery(after func()) error {
	if g == nil || g.st == nil || g.nativeEndTurnRecovery != nil ||
		!g.st.HasNativeMapViewState || g.st.NativeMapSelectorCache == nil ||
		!nativeMapAssetsAvailable(g.nativeMapAssets) || len(g.nativeMapDAC) != 256*3 ||
		len(g.nativeMapWork) != indexedmap.NativeUnitPresentWorkSize ||
		len(g.nativeMapVGA) != indexedmap.NativeMapVGASize {
		return fmt.Errorf("native END recovery: source frame/assets unavailable")
	}
	plan := g.st.PlanNativeEndTurnRecovery()
	if len(plan) > 0 && !osMuteOrShot(g) && (len(g.sfx) <= 4 || len(g.sfx[4]) == 0) {
		return fmt.Errorf("native END recovery: sample4 unavailable")
	}
	job := &nativeEndTurnRecoveryJob{plan: plan, after: after,
		readyAt: time.Now().Add(nativeBIOSTickPeriod)}
	job.frames[0] = append([]byte(nil), g.nativeMapVGA...)
	maskWork := append([]byte(nil), g.nativeMapWork...)
	restoreWork := append([]byte(nil), g.nativeMapWork...)
	roster, err := g.st.NativeMapFrameRoster()
	if err != nil {
		return fmt.Errorf("native END recovery: roster: %w", err)
	}
	view := g.st.NativeMapViewState
	for _, rec := range plan {
		entry, ok := rec.Unit.NativeUnitLayerEntry()
		if !ok || entry.ForceBase || entry.Inactive {
			return fmt.Errorf("native END recovery: candidate lacks raw selector provenance")
		}
		job.entries = append(job.entries, entry)
		// 0x1DA53..0x1DA90 returns before sprite lookup outside this window.
		if entry.X < view.CameraX-1 || entry.X > view.CameraX+12 ||
			entry.Y < view.CameraY-1 || entry.Y > view.CameraY+8 {
			continue
		}
		cycle, err := fdicon.NativeFrameIndex(entry.MotionOffset, false, roster.Cycles.Idle, roster.Cycles.Moving)
		if err != nil {
			return err
		}
		sprite, err := g.nativeMapAssets.Units.SpriteForNativeSlot(g.st.NativeMapSelectorCache, entry.Slot, entry.Pose, cycle)
		if err != nil {
			return fmt.Errorf("native END recovery: selector: %w", err)
		}
		offset, err := fdicon.NativePlacementOffset(entry.X, entry.Y, view.CameraX, view.CameraY,
			entry.Pose, entry.MotionOffset, 0, false)
		if err != nil || offset < 0 {
			return fmt.Errorf("native END recovery: invalid placement %d: %v", offset, err)
		}
		// 4DDD7 reads arg8 both as stride and opaque mask color. argC=FD is
		// not read. Mask includes opaque source zero, unlike color-key blits.
		if err := sprite.BlitConstantMaskAt(maskWork, 456, offset%456, offset/456, 0xc8); err != nil {
			return fmt.Errorf("native END recovery: mode2: %w", err)
		}
		// 4DEDA does not consume +5 bit7. Restore raw pixels before gray redraw.
		if err := sprite.BlitForNativeFlagsAtOffset(restoreWork, 456, offset, 0); err != nil {
			return fmt.Errorf("native END recovery: mode0: %w", err)
		}
	}
	job.frames[1], err = nativeAIIdleRecoveryViewport(maskWork, job.frames[0])
	if err != nil {
		return err
	}
	job.frames[2], err = nativeAIIdleRecoveryViewport(restoreWork, job.frames[0])
	if err != nil {
		return err
	}
	job.restoreWork = restoreWork
	if g.st.HasNativeMapHUDState {
		job.beforeGate = g.st.NativeMapHUDState.DisplayGateB
		g.st.NativeMapHUDState.DisplayGateB = 0
	}
	g.nativeEndTurnRecovery = job
	return nil
}

func (g *Game) cancelNativeEndTurnRecovery(err error) {
	if job := g.nativeEndTurnRecovery; job != nil && !job.committed && g.st.HasNativeMapHUDState {
		g.st.NativeMapHUDState.DisplayGateB = job.beforeGate
	}
	g.nativeEndTurnRecovery = nil
	g.loadErr = err.Error()
}

func (g *Game) stepNativeEndTurnRecovery(now time.Time) {
	job := g.nativeEndTurnRecovery
	if job == nil || !job.drawn {
		return
	}
	if job.frame == 0 && now.Before(job.readyAt) {
		return
	}
	if job.frame == 1 {
		for i, rec := range job.plan {
			entry, ok := rec.Unit.NativeUnitLayerEntry()
			if !ok || entry != job.entries[i] {
				g.cancelNativeEndTurnRecovery(fmt.Errorf("native END recovery: raw selector changed before commit"))
				return
			}
		}
		if err := g.st.ValidateNativeEndTurnRecovery(job.plan); err != nil {
			g.cancelNativeEndTurnRecovery(err)
			return
		}
		if len(job.plan) > 0 {
			job.cueCount++
			if !osMuteOrShot(g) {
				g.playSFX(4)
			}
		}
		if err := g.st.CommitNativeEndTurnRecovery(job.plan); err != nil {
			g.cancelNativeEndTurnRecovery(err)
			return
		}
		job.committed = true
		g.nativeMapWork = append([]byte(nil), job.restoreWork...)
	}
	job.frame++
	job.drawn = false
	if job.frame < len(job.frames) {
		return
	}
	if err := g.composeNativeMapFrame(); err != nil {
		g.cancelNativeEndTurnRecovery(fmt.Errorf("native END recovery: final 11CAC: %w", err))
		return
	}
	after := job.after
	g.nativeEndTurnRecovery = nil
	if after != nil {
		after()
	}
}

func (g *Game) drawNativeEndTurnRecovery(screen *ebiten.Image) bool {
	job := g.nativeEndTurnRecovery
	if job == nil || job.frame < 0 || job.frame >= len(job.frames) ||
		len(job.frames[job.frame]) != indexedmap.NativeMapVGASize || len(g.nativeMapDAC) != 256*3 {
		return false
	}
	palette, err := fdother.VGAPaletteFromDAC(g.nativeMapDAC)
	if err != nil {
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
