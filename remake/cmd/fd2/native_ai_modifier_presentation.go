package main

import (
	"errors"
	"fmt"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/wicanr2/fd2_re/remake/internal/battle"
	"github.com/wicanr2/fd2_re/remake/internal/fdother"
	"github.com/wicanr2/fd2_re/remake/internal/indexedmap"
)

type nativeAICommandModifierPresentationJob struct {
	actor                     *battle.Unit
	plan                      *battle.NativeAICommandModifierPlan
	frames                    []nativeCompoundPresentedFrame
	frame, repeat             int
	publishAt, endAt          int
	drawn, holding, published bool
	hold                      int
	rngBefore                 uint16
	baselineWork, baselineVGA []byte
	stageWork, stageVGA       []byte
	then                      func(battle.NativeCommandModifierResult)
}

func (g *Game) startNativeAICommandModifierPresentation(actor *battle.Unit, commandID int, then func(battle.NativeCommandModifierResult)) error {
	if !g.nativeFullPresentationEnabled() || g == nil || g.st == nil || actor == nil {
		return errors.New("native AI command modifier presentation context unavailable")
	}
	if g.nativeAICommandModifier != nil || g.nativeModifierPresentation != nil || g.nativeHealPresentation != nil ||
		g.nativeCmd0Presentation != nil || g.nativeCmd1Presentation != nil || g.nativeCmd2Presentation != nil ||
		g.nativeCmd3Presentation != nil || g.nativeCmd5Presentation != nil || g.nativeCmd6Presentation != nil ||
		g.nativeCmd7Presentation != nil || g.nativeCmd8Presentation != nil || g.nativeCmd9Player != nil ||
		g.nativeCmd9AIPresentation != nil || g.nativeCmd1012 != nil || g.nativeCmd24Presentation != nil ||
		g.nativeCmd29Presentation != nil || g.nativeCmd32Presentation != nil || g.nativeCmd33Presentation != nil ||
		g.nativeCmd34Presentation != nil || g.nativeCmd35Presentation != nil || g.indexedTransition != nil || g.atk != nil {
		return errors.New("native AI command modifier presentation already active")
	}
	if !g.st.HasNativeMapViewState || len(g.nativeMapWork) != indexedmap.NativeUnitPresentWorkSize ||
		len(g.nativeMapVGA) != indexedmap.NativeMapVGASize || !nativeMapAssetsAvailable(g.nativeMapAssets) {
		return errors.New("native AI command modifier indexed map state unavailable")
	}
	plan, err := g.st.PlanNativeAICommandModifier(actor, commandID, g.nativeRNGState)
	if err != nil {
		return err
	}
	schedules, err := fdother.BuildNativeCommand34TailSchedule()
	if err != nil {
		return err
	}
	var schedule *fdother.NativeCommand34StageSchedule
	for index := range schedules {
		if schedules[index].CommandID == commandID {
			schedule = &schedules[index]
			break
		}
	}
	if schedule == nil || len(g.nativeMapAssets.FDOTHER6) < schedule.EffectStart+schedule.EffectFrames ||
		len(g.nativeMapAssets.CommandHealDigits) <= schedule.DigitBias+9 {
		return errors.New("native AI command modifier tail descriptors unavailable")
	}

	view := g.st.NativeMapViewState
	tailTargets, err := nativeCommandHealTailTargets(g.st, plan.Targets)
	if err != nil {
		return err
	}
	effectTargets := make([]indexedmap.NativeCommandHealTailTarget, 0, len(tailTargets))
	for _, target := range tailTargets {
		if target.X >= view.CameraX-1 && target.X <= view.CameraX+12 && target.Y >= view.CameraY-1 && target.Y <= view.CameraY+8 {
			effectTargets = append(effectTargets, target)
		}
	}
	baselineWork, baselineVGA := append([]byte(nil), g.nativeMapWork...), append([]byte(nil), g.nativeMapVGA...)
	roster, err := g.st.NativeMapFrameRoster()
	if err != nil {
		return err
	}
	hud, ok := g.nativeMapHUDInput()
	if !ok {
		return errors.New("native AI command modifier HUD unavailable")
	}
	mapPalette, err := fdother.VGAPaletteFromDAC(g.nativeMapDAC)
	if err != nil {
		return err
	}

	effectPixels := make([][]byte, 0, schedule.EffectFrames)
	for frame := 0; frame < schedule.EffectFrames; frame++ {
		work, vga := append([]byte(nil), baselineWork...), append([]byte(nil), baselineVGA...)
		if err := indexedmap.ComposeNativeCommandHealEffectFrame(work, vga, baselineWork, g.nativeMapAssets.FDOTHER6[schedule.EffectStart+frame], effectTargets, view.CameraX, view.CameraY); err != nil {
			return err
		}
		effectPixels = append(effectPixels, vga)
	}
	maskWork, maskVGA := append([]byte(nil), baselineWork...), append([]byte(nil), baselineVGA...)
	if err := indexedmap.ComposeNativeCommandHealMaskFrame(maskWork, maskVGA, baselineWork, g.nativeMapAssets.Units, g.st.NativeMapSelectorCache, effectTargets, view.CameraX, view.CameraY, roster.Cycles.Idle, byte(schedule.MaskIndex)); err != nil {
		return err
	}
	clonedState, err := nativeCommand34ClonedState(g.st, plan.After)
	if err != nil {
		return err
	}
	postInput, err := buildNativeMapFrameInput(g.nativeMapAssets, g.m, clonedState, nativeMapFrameRuntime{HUD: hud})
	if err != nil {
		return err
	}
	postWork, postVGA := append([]byte(nil), baselineWork...), append([]byte(nil), baselineVGA...)
	if err := indexedmap.ComposeNativeFrame(postWork, postVGA, postInput); err != nil {
		return err
	}
	queue := make([]battle.NativePresentationDigit, 0, len(tailTargets)*4)
	for targetIndex, target := range tailTargets {
		inCamera := target.X >= view.CameraX && target.X < view.CameraX+12 && target.Y >= view.CameraY-1 && target.Y <= view.CameraY+7
		processed, value := false, 0
		if commandID < 19 {
			processed, value = plan.Result.WordSteps[targetIndex].Processed, int(plan.Result.WordSteps[targetIndex].Delta)
		} else {
			processed, value = plan.Result.PairSteps[targetIndex].Processed, 0x0f
		}
		if processed {
			queue, err = battle.AppendNativePresentationDigits(queue, value, schedule.DigitBias, target.RecordIndex, inCamera)
			if err != nil {
				return err
			}
		} else if inCamera {
			for slot, glyph := range [...]int{74, 75, 76, 76} {
				queue = append(queue, battle.NativePresentationDigit{PositionCode: [...]int{2, 8, 12, 17}[slot], Target: target.RecordIndex, Digit: glyph})
			}
		}
	}
	digitPixels := make([][]byte, 0, schedule.DigitFrames)
	for frame := 0; frame < schedule.DigitFrames; frame++ {
		work, vga := append([]byte(nil), postWork...), append([]byte(nil), postVGA...)
		if err := indexedmap.ComposeNativeCommandHealDigitFrame(work, vga, postWork, g.nativeMapAssets.CommandHealDigits, queue, tailTargets, view.CameraX, view.CameraY, schedule.DigitVertical, frame); err != nil {
			return err
		}
		digitPixels = append(digitPixels, vga)
	}
	effectImages, err := nativeCommand24IndexedImages(effectPixels, mapPalette)
	if err != nil {
		return err
	}
	maskImages, err := nativeCommand24IndexedImages([][]byte{baselineVGA, maskVGA}, mapPalette)
	if err != nil {
		return err
	}
	digitImages, err := nativeCommand24IndexedImages(digitPixels, mapPalette)
	if err != nil {
		return err
	}
	frames := make([]nativeCompoundPresentedFrame, 0, len(effectImages)+schedule.MaskPairs*2+1+len(digitImages))
	if frames, err = appendNativeCompoundFrames(frames, effectImages, 1); err != nil {
		return err
	}
	effectSample := loadWav(assetPath(fmt.Sprintf("assets/sfx/battle_80_%02d.wav", schedule.EffectSample)))
	frames[0].sound = effectSample
	for _, rawFrame := range schedule.ExtraSampleFrameIndices {
		if rawFrame <= 0 || rawFrame >= schedule.EffectFrames {
			return errors.New("native AI command modifier extra sample frame unavailable")
		}
		frames[rawFrame].sound = effectSample
	}
	maskStart := len(frames)
	for index := 0; index < schedule.MaskPairs*2+1; index++ {
		frames = append(frames, nativeCompoundPresentedFrame{image: maskImages[index%2], delay: 1})
	}
	frames[maskStart].sound = loadWav(assetPath("assets/sfx/battle_80_01.wav"))
	publishAt := len(frames)
	if frames, err = appendNativeCompoundFrames(frames, digitImages, 1); err != nil {
		return err
	}
	if !osMuteOrShot(g) && (len(effectSample) == 0 || len(frames[maskStart].sound) == 0) {
		return errors.New("native AI command modifier raw sample unavailable")
	}
	g.nativeAICommandModifier = &nativeAICommandModifierPresentationJob{
		actor: actor, plan: plan, frames: frames, publishAt: publishAt, endAt: len(frames), rngBefore: g.nativeRNGState,
		baselineWork: baselineWork, baselineVGA: baselineVGA, stageWork: postWork, stageVGA: postVGA, then: then,
	}
	return nil
}

func (g *Game) cancelNativeAICommandModifierPresentation() {
	j := g.nativeAICommandModifier
	if j == nil {
		return
	}
	_ = battle.AbortNativeAICommandModifier(j.plan)
	g.nativeRNGState = j.rngBefore
	g.nativeMapWork = append(g.nativeMapWork[:0], j.baselineWork...)
	g.nativeMapVGA = append(g.nativeMapVGA[:0], j.baselineVGA...)
	g.nativeAICommandModifier = nil
}

func (g *Game) failNativeAICommandModifierPresentation(err error) {
	g.cancelNativeAICommandModifierPresentation()
	g.loadErr = "native AI command modifier presentation: " + err.Error()
}

func (g *Game) stepNativeAICommandModifierPresentation() {
	j := g.nativeAICommandModifier
	if j == nil {
		return
	}
	if j.holding {
		if j.hold > 0 {
			j.hold--
			return
		}
		if err := battle.CompleteNativeAICommandModifier(j.plan); err != nil {
			g.failNativeAICommandModifierPresentation(err)
			return
		}
		then, result := j.then, j.plan.Result
		g.nativeAICommandModifier = nil
		if then != nil {
			then(result)
		}
		return
	}
	if !j.drawn || j.frame < 0 || j.frame >= len(j.frames) {
		return
	}
	j.drawn = false
	frame := &j.frames[j.frame]
	if !frame.soundPlayed && len(frame.sound) > 0 {
		g.playRaw(frame.sound)
		frame.soundPlayed = true
	}
	j.repeat++
	if j.repeat < frame.delay {
		return
	}
	j.repeat = 0
	next := j.frame + 1
	if next == j.publishAt && !j.published {
		if err := battle.PublishNativeAICommandModifier(j.plan); err != nil {
			g.failNativeAICommandModifierPresentation(err)
			return
		}
		g.nativeRNGState = j.plan.Result.RNGState
		g.nativeMapWork = append(g.nativeMapWork[:0], j.stageWork...)
		g.nativeMapVGA = append(g.nativeMapVGA[:0], j.stageVGA...)
		j.published = true
	}
	if next == j.endAt {
		if !j.published {
			g.failNativeAICommandModifierPresentation(errors.New("required modifier boundary was not presented"))
			return
		}
		j.frame = next - 1
		j.holding, j.hold = true, nativeDelayTicks(500)
		return
	}
	j.frame = next
}

func (g *Game) drawNativeAICommandModifierPresentation(screen *ebiten.Image) bool {
	j := g.nativeAICommandModifier
	if j == nil || j.frame < 0 || j.frame >= len(j.frames) || j.frames[j.frame].image == nil {
		return false
	}
	op := &ebiten.DrawImageOptions{}
	op.GeoM.Scale(2, 2)
	screen.DrawImage(j.frames[j.frame].image, op)
	j.drawn = true
	return true
}
