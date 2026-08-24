package main

import (
	"errors"
	"fmt"
	"image"
	"image/color"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/wicanr2/fd2_re/remake/internal/battle"
	"github.com/wicanr2/fd2_re/remake/internal/fdother"
	"github.com/wicanr2/fd2_re/remake/internal/indexedmap"
)

type nativeCommand9PlayerPhase uint8

const (
	nativeCommand9PlayerPalette nativeCommand9PlayerPhase = iota
	nativeCommand9PlayerEffect
	nativeCommand9PlayerResult
	nativeCommand9PlayerHold
)

type nativeCommand9PlayerJob struct {
	actor, target                 *battle.Unit
	plan                          *battle.NativeCommandDamagePlan
	palettes                      []color.Palette
	effectFrames, resultFrames    [][]byte
	baselineWork, baselineVGA     []byte
	phase                         nativeCommand9PlayerPhase
	frame, hold                   int
	drawn                         bool
	actorMPBefore, targetHPBefore int
	actorActedBefore              bool
	rngBefore                     uint16
	then                          func([]battle.NativeCommandDamageResult)
}

func (g *Game) startNativeCommand9PlayerPresentation(actor, confirmed *battle.Unit, then func([]battle.NativeCommandDamageResult)) error {
	if !g.nativeFullPresentationEnabled() || g == nil || g.st == nil || actor == nil || confirmed == nil {
		return errors.New("native command9 player presentation context unavailable")
	}
	if g.nativeCmd9Player != nil || g.nativeHealPresentation != nil || g.nativeModifierPresentation != nil ||
		g.nativeCmd0Presentation != nil || g.nativeCmd1Presentation != nil || g.nativeCmd2Presentation != nil ||
		g.nativeCmd3Presentation != nil || g.nativeCmd5Presentation != nil || g.nativeCmd6Presentation != nil ||
		g.nativeCmd7Presentation != nil || g.nativeCmd8Presentation != nil || g.nativeCmd9AIPresentation != nil ||
		g.nativeCmd24Presentation != nil || g.nativeCmd29Presentation != nil || g.indexedTransition != nil {
		return errors.New("native command9 player presentation already active")
	}
	if !g.st.HasNativeMapViewState || len(g.nativeMapWork) != indexedmap.NativeUnitPresentWorkSize ||
		len(g.nativeMapVGA) != indexedmap.NativeMapVGASize || len(g.nativeMapDAC) != 256*3 ||
		g.nativeCommandPaletteFlash == nil || !nativeMapAssetsAvailable(g.nativeMapAssets) {
		return errors.New("native command9 player map/palette assets unavailable")
	}
	a := g.nativeMapAssets
	schedule, err := fdother.BuildNativeCommand9PlayerSchedule(a.FDOTHER6, a.CommandHealDigits)
	if err != nil {
		return err
	}
	if schedule.SoundResource != 80 || schedule.InitialSample != 14 || schedule.RepeatSample != 15 ||
		(!osMuteOrShot(g) && (len(g.sfxCommand9PlayerPalette) == 0 || len(g.sfxCommand9PlayerInitial) == 0 || len(g.sfxCommand9PlayerRepeat) == 0)) {
		return errors.New("native command9 player FDOTHER #80 selectors unavailable")
	}
	plan, err := g.st.PlanNativeCommandDamage(actor, confirmed, 9, g.st.NativeCommandResistances, g.nativeRNGState)
	if err != nil {
		return err
	}
	if len(plan.Results) != 1 || plan.Results[0].Target != confirmed {
		return errors.New("native command9 player single target contract unavailable")
	}
	targets, err := nativeCommandHealTailTargets(g.st, []*battle.Unit{confirmed})
	if err != nil {
		return err
	}
	view := g.st.NativeMapViewState
	if targets[0].X < view.CameraX-1 || targets[0].X > view.CameraX+12 || targets[0].Y < view.CameraY-1 || targets[0].Y > view.CameraY+8 {
		return errors.New("native command9 player target outside recovered compositor bounds")
	}
	baselineWork, baselineVGA := append([]byte(nil), g.nativeMapWork...), append([]byte(nil), g.nativeMapVGA...)
	effects := make([][]byte, 0, schedule.EffectFrames)
	for frame := 0; frame < schedule.EffectFrames; frame++ {
		work, vga := append([]byte(nil), baselineWork...), append([]byte(nil), baselineVGA...)
		if err := indexedmap.ComposeNativeCommandHealEffectFrame(work, vga, baselineWork, a.FDOTHER6[schedule.EffectStart+frame], targets, view.CameraX, view.CameraY); err != nil {
			return fmt.Errorf("native command9 player effect %d: %w", frame, err)
		}
		effects = append(effects, vga)
	}
	record := targets[0].RecordIndex
	inCamera := targets[0].X >= view.CameraX && targets[0].X < view.CameraX+12 && targets[0].Y >= view.CameraY-1 && targets[0].Y <= view.CameraY+7
	queue := make([]battle.NativePresentationDigit, 0, 4)
	if plan.Results[0].Hit {
		queue, err = battle.AppendNativePresentationDigits(queue, plan.Results[0].Damage, schedule.ResultDigitBias, record, inCamera)
	} else if inCamera {
		positions := [...]int{2, 8, 12, 17}
		for i, descriptor := range schedule.ResultMissDescriptors {
			queue = append(queue, battle.NativePresentationDigit{PositionCode: positions[i], Target: record, Digit: descriptor})
		}
	}
	if err != nil {
		return err
	}
	results := make([][]byte, 0, schedule.ResultFrames)
	for frame := 0; frame < schedule.ResultFrames; frame++ {
		work, vga := append([]byte(nil), baselineWork...), append([]byte(nil), baselineVGA...)
		if err := indexedmap.ComposeNativeCommandHealDigitFrame(work, vga, baselineWork, a.CommandHealDigits, queue, targets, view.CameraX, view.CameraY, schedule.ResultVertical, frame); err != nil {
			return fmt.Errorf("native command9 player result %d: %w", frame, err)
		}
		results = append(results, vga)
	}
	phases, err := g.nativeCommandPaletteFlash.NativeCommandPaletteFlashPhases(9)
	if err != nil {
		return err
	}
	palettes := make([]color.Palette, 0, len(phases))
	for _, rgb := range phases {
		dac := append([]byte(nil), g.nativeMapDAC...)
		copy(dac[:3], rgb[:])
		palette, err := fdother.VGAPaletteFromDAC(dac)
		if err != nil {
			return err
		}
		palettes = append(palettes, palette)
	}
	g.nativeCmd9Player = &nativeCommand9PlayerJob{
		actor: actor, target: confirmed, plan: plan, palettes: palettes, effectFrames: effects, resultFrames: results,
		baselineWork: baselineWork, baselineVGA: baselineVGA, actorMPBefore: actor.MP,
		actorActedBefore: actor.Acted, targetHPBefore: confirmed.HP, rngBefore: g.nativeRNGState, then: then,
	}
	g.playRaw(g.sfxCommand9PlayerPalette)
	return nil
}

func (g *Game) failNativeCommand9PlayerPresentation(err error) {
	j := g.nativeCmd9Player
	if j == nil {
		return
	}
	j.actor.MP, j.actor.Acted, j.target.HP = j.actorMPBefore, j.actorActedBefore, j.targetHPBefore
	g.nativeRNGState = j.rngBefore
	g.nativeMapWork, g.nativeMapVGA = append(g.nativeMapWork[:0], j.baselineWork...), append(g.nativeMapVGA[:0], j.baselineVGA...)
	g.nativeCmd9Player = nil
	g.loadErr = "native command9 player presentation: " + err.Error()
}

func (g *Game) stepNativeCommand9PlayerPresentation() {
	j := g.nativeCmd9Player
	if j == nil {
		return
	}
	switch j.phase {
	case nativeCommand9PlayerPalette:
		if !j.drawn {
			return
		}
		j.drawn = false
		j.frame++
		if j.frame >= len(j.palettes) {
			j.phase, j.frame = nativeCommand9PlayerEffect, 0
			g.playRaw(g.sfxCommand9PlayerInitial)
		}
	case nativeCommand9PlayerEffect:
		if !j.drawn {
			return
		}
		j.drawn = false
		j.frame++
		if j.frame == 15 || j.frame == 19 {
			g.playRaw(g.sfxCommand9PlayerRepeat)
		}
		if j.frame >= len(j.effectFrames) {
			if err := battle.ApplyNativeCommandDamageMP(j.plan); err != nil {
				g.failNativeCommand9PlayerPresentation(err)
				return
			}
			if err := battle.ApplyNativeCommandDamageStage(j.plan, 0, j.plan.DamageStages); err != nil {
				g.failNativeCommand9PlayerPresentation(err)
				return
			}
			g.nativeRNGState = j.plan.RNGAfter
			j.phase, j.frame = nativeCommand9PlayerResult, 0
		}
	case nativeCommand9PlayerResult:
		if !j.drawn {
			return
		}
		j.drawn = false
		j.frame++
		if j.frame >= len(j.resultFrames) {
			j.phase, j.hold = nativeCommand9PlayerHold, nativeDelayTicks(500)
		}
	case nativeCommand9PlayerHold:
		if j.hold > 0 {
			j.hold--
			return
		}
		if err := battle.CompleteNativeCommandDamage(j.plan); err != nil {
			g.failNativeCommand9PlayerPresentation(err)
			return
		}
		then, results := j.then, append([]battle.NativeCommandDamageResult(nil), j.plan.Results...)
		g.nativeMapWork, g.nativeMapVGA = append(g.nativeMapWork[:0], j.baselineWork...), append(g.nativeMapVGA[:0], j.baselineVGA...)
		g.nativeCmd9Player = nil
		if then != nil {
			then(results)
		}
	}
}

func (g *Game) drawNativeCommand9PlayerPresentation(screen *ebiten.Image) bool {
	j := g.nativeCmd9Player
	if j == nil {
		return false
	}
	pixels, palette := j.baselineVGA, color.Palette(nil)
	switch j.phase {
	case nativeCommand9PlayerPalette:
		if j.frame < 0 || j.frame >= len(j.palettes) {
			return false
		}
		palette = j.palettes[j.frame]
	case nativeCommand9PlayerEffect:
		if j.frame < 0 || j.frame >= len(j.effectFrames) {
			return false
		}
		pixels = j.effectFrames[j.frame]
	case nativeCommand9PlayerResult:
		if j.frame < 0 || j.frame >= len(j.resultFrames) {
			return false
		}
		pixels = j.resultFrames[j.frame]
	}
	if palette == nil {
		var err error
		palette, err = fdother.VGAPaletteFromDAC(g.nativeMapDAC)
		if err != nil {
			return false
		}
	}
	if len(pixels) != indexedmap.NativeMapVGASize || len(palette) != 256 {
		return false
	}
	img := image.NewPaletted(image.Rect(0, 0, 320, 200), palette)
	copy(img.Pix, pixels)
	op := &ebiten.DrawImageOptions{}
	op.GeoM.Scale(2, 2)
	screen.DrawImage(ebiten.NewImageFromImage(img), op)
	j.drawn = true
	return true
}
