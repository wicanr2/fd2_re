package main

import (
	"errors"
	"fmt"
	"image"
	"image/color"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/wicanr2/fd2_re/remake/internal/battle"
	"github.com/wicanr2/fd2_re/remake/internal/campaign"
	"github.com/wicanr2/fd2_re/remake/internal/fdother"
	"github.com/wicanr2/fd2_re/remake/internal/indexedmap"
)

type nativeCommand1012Phase uint8

const (
	nativeCommand1012Prelude nativeCommand1012Phase = iota
	nativeCommand1012Main
	nativeCommand1012Result
	nativeCommand1012Hold
)

type nativeCommand1012Job struct {
	actor                     *battle.Unit
	plan                      *battle.NativeCommandDamagePlan
	schedule                  fdother.NativeCommand10To12Schedule
	mainFrames, resultFrames  [][]byte
	baselineWork, baselineVGA []byte
	palette                   color.Palette
	phase                     nativeCommand1012Phase
	frame, hold               int
	drawn, preludeDone        bool
	actorMPBefore             int
	actorActedBefore          bool
	targetHPBefore            []int
	rngBefore                 uint16
	then                      func([]battle.NativeCommandDamageResult)
}

func (g *Game) startNativeCommand1012Presentation(actor, confirmed *battle.Unit, commandID int, then func([]battle.NativeCommandDamageResult)) error {
	if !g.nativeFullPresentationEnabled() || g == nil || g.st == nil || actor == nil || confirmed == nil || g.nativeCmd1012 != nil || g.native2189A != nil {
		return errors.New("native command10-12 presentation context unavailable")
	}
	if g.nativeHealPresentation != nil || g.nativeModifierPresentation != nil || g.nativeCmd0Presentation != nil ||
		g.nativeCmd1Presentation != nil || g.nativeCmd2Presentation != nil || g.nativeCmd3Presentation != nil ||
		g.nativeCmd5Presentation != nil || g.nativeCmd6Presentation != nil || g.nativeCmd7Presentation != nil ||
		g.nativeCmd8Presentation != nil || g.nativeCmd9Player != nil || g.nativeCmd9AIPresentation != nil ||
		g.nativeCmd24Presentation != nil || g.nativeCmd29Presentation != nil || g.indexedTransition != nil {
		return errors.New("native command10-12 presentation already active")
	}
	schedule, err := fdother.BuildNativeCommand10To12Schedule(commandID)
	if err != nil {
		return err
	}
	if g.st == nil || g.m == nil || !nativeMapAssetsAvailable(g.nativeMapAssets) || !g.st.HasNativeMapViewState ||
		len(g.nativeMapWork) != indexedmap.NativeUnitPresentWorkSize || len(g.nativeMapVGA) != indexedmap.NativeMapVGASize ||
		len(g.nativeMapAssets.CommandHealDigits) <= 118 ||
		(!osMuteOrShot(g) && (len(g.sfxCommand1012Main) == 0 || schedule.Prelude.Enabled && len(g.sfxCommand1012Prelude) == 0)) {
		return errors.New("native command10-12 map/result/audio assets unavailable")
	}
	plan, err := g.st.PlanNativeCommandDamage(actor, confirmed, commandID, g.st.NativeCommandResistances, g.nativeRNGState)
	if err != nil {
		return err
	}
	if len(plan.Results) == 0 || plan.DamageStages != 1 {
		return errors.New("native command10-12 final targets unavailable")
	}
	targetUnits := make([]*battle.Unit, 0, len(plan.Results))
	for _, result := range plan.Results {
		targetUnits = append(targetUnits, result.Target)
	}
	targets, err := nativeCommandHealTailTargets(g.st, targetUnits)
	if err != nil {
		return err
	}
	hud, ok := g.nativeMapHUDInput()
	if !ok {
		return errors.New("native command10-12 HUD provenance unavailable")
	}
	in, err := buildNativeMapFrameInput(g.nativeMapAssets, g.m, g.st, nativeMapFrameRuntime{HUD: hud})
	if err != nil {
		return err
	}
	transition := nativeTransitionInput(in)
	baselineWork, baselineVGA := append([]byte(nil), g.nativeMapWork...), append([]byte(nil), g.nativeMapVGA...)
	surfaces := make([][]byte, 4)
	surfaces[0] = append([]byte(nil), baselineVGA...)
	for phase := 0; phase < 3; phase++ {
		work, vga := append([]byte(nil), baselineWork...), append([]byte(nil), baselineVGA...)
		if err := indexedmap.ComposeNativeCommand10To12Surface(work, vga, transition, schedule.XOrigins[phase], schedule.YOrigins[phase], schedule.SamplingIncrements[phase]); err != nil {
			return fmt.Errorf("native command10-12 sampled surface %d: %w", phase, err)
		}
		surfaces[phase+1] = vga
	}
	mainFrames := make([][]byte, schedule.Frames)
	for frame := range mainFrames {
		mainFrames[frame] = surfaces[schedule.SurfaceCycle[frame%len(schedule.SurfaceCycle)]]
	}
	queue := make([]battle.NativePresentationDigit, 0, len(plan.Results)*4)
	view := g.st.NativeMapViewState
	for index, result := range plan.Results {
		target := targets[index]
		inCamera := target.X >= view.CameraX && target.X < view.CameraX+12 && target.Y >= view.CameraY-1 && target.Y <= view.CameraY+7
		if result.Hit {
			queue, err = battle.AppendNativePresentationDigits(queue, result.Damage, schedule.ResultDigitBias, target.RecordIndex, inCamera)
		} else if inCamera {
			positions := [...]int{2, 8, 12, 17}
			for slot, descriptor := range schedule.ResultMissDescriptors {
				queue = append(queue, battle.NativePresentationDigit{PositionCode: positions[slot], Target: target.RecordIndex, Digit: descriptor})
			}
		}
		if err != nil {
			return err
		}
	}
	resultFrames := make([][]byte, 0, schedule.ResultFrames)
	for frame := 0; frame < schedule.ResultFrames; frame++ {
		work, vga := append([]byte(nil), baselineWork...), append([]byte(nil), baselineVGA...)
		if err := indexedmap.ComposeNativeCommandHealDigitFrame(work, vga, baselineWork, g.nativeMapAssets.CommandHealDigits, queue, targets, view.CameraX, view.CameraY, schedule.ResultVertical, frame); err != nil {
			return fmt.Errorf("native command10-12 result frame %d: %w", frame, err)
		}
		resultFrames = append(resultFrames, vga)
	}
	palette := append(color.Palette(nil), g.nativeMapAssets.Palette...)
	if len(g.nativeMapDAC) == 256*3 {
		palette, err = fdother.VGAPaletteFromDAC(g.nativeMapDAC)
		if err != nil {
			return err
		}
	}
	job := &nativeCommand1012Job{actor: actor, plan: plan, schedule: schedule, mainFrames: mainFrames,
		resultFrames: resultFrames, baselineWork: baselineWork, baselineVGA: baselineVGA, palette: palette,
		phase: nativeCommand1012Main, actorMPBefore: actor.MP, actorActedBefore: actor.Acted,
		targetHPBefore: make([]int, len(plan.Results)), rngBefore: g.nativeRNGState, then: then}
	for i, result := range plan.Results {
		job.targetHPBefore[i] = result.Target.HP
	}
	g.nativeCmd1012 = job
	if schedule.Prelude.Enabled {
		actorSlot := -1
		for i, unit := range g.st.Units {
			if unit == actor {
				actorSlot = i
				break
			}
		}
		if actorSlot < 0 {
			g.nativeCmd1012 = nil
			return errors.New("native command10-12 actor raw slot unavailable")
		}
		job.phase = nativeCommand1012Prelude
		loop := campaign.Native2189ALoop{Slot: actorSlot, InitialRadius: schedule.Prelude.InitialRadius, RadiusStep: schedule.Prelude.RadiusStep,
			Repeat: schedule.Prelude.Repeat, WorkOffset: 0x8088, WorkStride: 456, MapRows: 8, MapColumns: 13, ClipWidth: 312, ClipHeight: 192, PresentStride: 320}
		if err := g.startNative2189A(loop, func() {
			job.preludeDone = true
			job.phase, job.frame = nativeCommand1012Main, 0
			g.playRaw(g.sfxCommand1012Main)
		}); err != nil {
			g.nativeCmd1012 = nil
			return err
		}
		g.playRaw(g.sfxCommand1012Prelude)
	} else {
		g.playRaw(g.sfxCommand1012Main)
	}
	return nil
}

func (g *Game) failNativeCommand1012Presentation(err error) {
	j := g.nativeCmd1012
	if j == nil {
		return
	}
	j.actor.MP, j.actor.Acted = j.actorMPBefore, j.actorActedBefore
	for i, result := range j.plan.Results {
		result.Target.HP = j.targetHPBefore[i]
	}
	g.nativeRNGState = j.rngBefore
	g.nativeMapWork, g.nativeMapVGA = append(g.nativeMapWork[:0], j.baselineWork...), append(g.nativeMapVGA[:0], j.baselineVGA...)
	g.nativeCmd1012 = nil
	g.loadErr = "native command10-12 presentation: " + err.Error()
}

func (g *Game) stepNativeCommand1012Presentation() {
	j := g.nativeCmd1012
	if j == nil {
		return
	}
	if j.phase == nativeCommand1012Prelude {
		if g.native2189A == nil && !j.preludeDone {
			g.failNativeCommand1012Presentation(errors.New("prelude ended without completion"))
		}
		return
	}
	if j.phase == nativeCommand1012Main {
		if !j.drawn {
			return
		}
		j.drawn = false
		if j.frame == 0 {
			if err := battle.ApplyNativeCommandDamageMP(j.plan); err != nil {
				g.failNativeCommand1012Presentation(err)
				return
			}
		}
		j.frame++
		for _, marker := range j.schedule.MainSampleFrames {
			if j.frame == marker {
				g.playRaw(g.sfxCommand1012Main)
			}
		}
		if j.frame < len(j.mainFrames) {
			return
		}
		for index := range j.plan.Results {
			if err := battle.ApplyNativeCommandDamageStage(j.plan, index, j.plan.DamageStages); err != nil {
				g.failNativeCommand1012Presentation(err)
				return
			}
		}
		g.nativeRNGState = j.plan.RNGAfter
		j.phase, j.frame = nativeCommand1012Result, 0
		return
	}
	if j.phase == nativeCommand1012Result {
		if !j.drawn {
			return
		}
		j.drawn = false
		j.frame++
		if j.frame >= len(j.resultFrames) {
			j.phase, j.hold = nativeCommand1012Hold, nativeDelayTicks(j.schedule.ResultHoldMS)
		}
		return
	}
	if j.hold > 0 {
		j.hold--
		return
	}
	if err := battle.CompleteNativeCommandDamage(j.plan); err != nil {
		g.failNativeCommand1012Presentation(err)
		return
	}
	then, results := j.then, append([]battle.NativeCommandDamageResult(nil), j.plan.Results...)
	g.nativeMapWork, g.nativeMapVGA = append(g.nativeMapWork[:0], j.baselineWork...), append(g.nativeMapVGA[:0], j.baselineVGA...)
	g.nativeCmd1012 = nil
	if then != nil {
		then(results)
	}
}

func (g *Game) drawNativeCommand1012Presentation(screen *ebiten.Image) bool {
	j := g.nativeCmd1012
	if j == nil || len(j.palette) != 256 || j.phase == nativeCommand1012Prelude {
		return false
	}
	pixels := j.baselineVGA
	if j.phase == nativeCommand1012Main {
		if j.frame < 0 || j.frame >= len(j.mainFrames) {
			return false
		}
		pixels = j.mainFrames[j.frame]
	} else if j.phase == nativeCommand1012Result {
		if j.frame < 0 || j.frame >= len(j.resultFrames) {
			return false
		}
		pixels = j.resultFrames[j.frame]
	}
	if len(pixels) != indexedmap.NativeMapVGASize {
		return false
	}
	img := image.NewPaletted(image.Rect(0, 0, 320, 200), j.palette)
	copy(img.Pix, pixels)
	op := &ebiten.DrawImageOptions{}
	op.GeoM.Scale(2, 2)
	screen.DrawImage(ebiten.NewImageFromImage(img), op)
	j.drawn = true
	return true
}
