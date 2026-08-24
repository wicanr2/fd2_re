package main

import (
	"errors"
	"fmt"
	"image"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/wicanr2/fd2_re/remake/internal/battle"
	"github.com/wicanr2/fd2_re/remake/internal/fdother"
	"github.com/wicanr2/fd2_re/remake/internal/indexedmap"
)

type nativeAICommandModifierPresentationJob struct {
	actor                     *battle.Unit
	plan                      *battle.NativeAICommandModifierPlan
	plan2022                  *battle.NativeAICommand2022Plan
	frames                    []nativeCompoundPresentedFrame
	frame, repeat             int
	publishAt, endAt          int
	drawn, holding, published bool
	hold                      int
	rngBefore                 uint16
	baselineWork, baselineVGA []byte
	stageWork, stageVGA       []byte
	then                      func(battle.NativeCommandModifierResult)
	then2022                  func(*battle.NativeAICommand2022Plan)
}

func nativeCommand2022ClonedState(st *battle.State, plan *battle.NativeAICommand2022Plan) (*battle.State, error) {
	if st == nil || plan == nil || len(st.Units) == 0 || len(plan.Results) == 0 {
		return nil, errors.New("native AI command 20-22 cloned state unavailable")
	}
	clone := *st
	clone.Units = make([]*battle.Unit, len(st.Units))
	byOriginal := make(map[*battle.Unit]*battle.Unit, len(st.Units))
	for index, unit := range st.Units {
		if unit == nil {
			continue
		}
		copied := *unit
		clone.Units[index] = &copied
		byOriginal[unit] = &copied
	}
	for index, result := range plan.Results {
		copied := byOriginal[result.Target]
		if copied == nil || result.Offset < 0x22 || result.Offset > 0x27 {
			return nil, fmt.Errorf("native AI command 20-22 cloned target %d unavailable", index)
		}
		if result.Restore != nil && result.Restore.Applied {
			copied.HP += result.Restore.Restore.Actual
			copied.SetNativeTransientDuration(result.Offset, 0)
		}
		if result.Apply != nil && result.Apply.Applied {
			copied.HP -= result.Apply.Damage.Actual
			copied.SetNativeTransientDuration(result.Offset, result.Apply.Marker)
		}
	}
	return &clone, nil
}

// startNativeAICommand2022Presentation owns the same handler tail reached by
// both funcs_1541F and the player table. Enemy callers intentionally omit the
// player-only 0x1D6C8 palette prelude.
func (g *Game) startNativeAICommand2022Presentation(actor *battle.Unit, commandID int, then func(*battle.NativeAICommand2022Plan)) error {
	return g.startNativeCommand2022Presentation(actor, nil, commandID, false, then)
}

func (g *Game) startNativeCommand2022Presentation(actor, confirmed *battle.Unit, commandID int, player bool, then func(*battle.NativeAICommand2022Plan)) error {
	if g == nil || !g.nativeFullPresentationEnabled() || g.st == nil || actor == nil || commandID < 20 || commandID > 22 {
		return errors.New("native command 20-22 presentation context unavailable")
	}
	if player && confirmed == nil {
		return errors.New("native player command 20-22 confirmed target unavailable")
	}
	if g.nativeAICommandModifier != nil || g.nativeModifierPresentation != nil || g.nativeHealPresentation != nil ||
		g.nativeCmd0Presentation != nil || g.nativeCmd1Presentation != nil || g.nativeCmd2Presentation != nil ||
		g.nativeCmd3Presentation != nil || g.nativeCmd5Presentation != nil || g.nativeCmd6Presentation != nil ||
		g.nativeCmd7Presentation != nil || g.nativeCmd8Presentation != nil || g.nativeCmd9Player != nil ||
		g.nativeCmd9AIPresentation != nil || g.nativeCmd1012 != nil || g.nativeCmd24Presentation != nil ||
		g.nativeCmd29Presentation != nil || g.nativeCmd32Presentation != nil || g.nativeCmd33Presentation != nil ||
		g.nativeCmd34Presentation != nil || g.nativeCmd35Presentation != nil || g.indexedTransition != nil || g.atk != nil {
		return errors.New("native AI command 20-22 presentation already active")
	}
	if !g.st.HasNativeMapViewState || len(g.nativeMapWork) != indexedmap.NativeUnitPresentWorkSize ||
		len(g.nativeMapVGA) != indexedmap.NativeMapVGASize || !nativeMapAssetsAvailable(g.nativeMapAssets) {
		return errors.New("native AI command 20-22 indexed map state unavailable")
	}
	var plan *battle.NativeAICommand2022Plan
	var err error
	if player {
		plan, err = g.st.PlanNativePlayerCommand2022(actor, confirmed, commandID, g.nativeRNGState)
	} else {
		plan, err = g.st.PlanNativeAICommand2022(actor, commandID, g.nativeRNGState)
	}
	if err != nil {
		return err
	}
	rows, err := fdother.BuildNativeCommand2022TailSchedule()
	if err != nil {
		return err
	}
	var schedule *fdother.NativeCommand34StageSchedule
	for index := range rows {
		if rows[index].CommandID == commandID {
			schedule = &rows[index]
			break
		}
	}
	if schedule == nil || len(g.nativeMapAssets.FDOTHER6) < schedule.EffectStart+schedule.EffectFrames ||
		len(g.nativeMapAssets.CommandHealDigits) <= schedule.DigitBias+9 {
		return errors.New("native AI command 20-22 tail descriptors unavailable")
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
		return errors.New("native AI command 20-22 HUD unavailable")
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
	clonedState, err := nativeCommand2022ClonedState(g.st, plan)
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
	queue := make([]battle.NativePresentationDigit, 0, len(plan.Results)*4)
	for _, result := range plan.Results {
		record := -1
		for unitIndex, unit := range g.st.Units {
			if result.Target == unit {
				record = unitIndex
				break
			}
		}
		if record < 0 {
			return errors.New("native AI command 20-22 result target unavailable")
		}
		inCamera := result.Target.X >= view.CameraX && result.Target.X < view.CameraX+12 && result.Target.Y >= view.CameraY-1 && result.Target.Y <= view.CameraY+7
		processed, value := false, 0
		if result.Restore != nil {
			processed, value = result.Restore.Applied, result.Restore.Restore.Actual
		} else if result.Apply != nil {
			processed, value = result.Apply.Applied, result.Apply.Damage.Actual
		}
		if processed {
			queue, err = battle.AppendNativePresentationDigits(queue, value, schedule.DigitBias, record, inCamera)
			if err != nil {
				return err
			}
		} else if inCamera {
			for slot, glyph := range [...]int{74, 75, 76, 76} {
				queue = append(queue, battle.NativePresentationDigit{PositionCode: [...]int{2, 8, 12, 17}[slot], Target: record, Digit: glyph})
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
	frames := make([]nativeCompoundPresentedFrame, 0, 8+len(effectImages)+schedule.MaskPairs*2+1+len(digitImages))
	if player {
		if g.nativeCommandPaletteFlash == nil || (!osMuteOrShot(g) && len(g.sfxCommandModifier) == 0) {
			return errors.New("native player command 20-22 palette assets unavailable")
		}
		phases, phaseErr := g.nativeCommandPaletteFlash.NativeCommandPaletteFlashPhases(commandID)
		if phaseErr != nil || len(phases) != 8 {
			return errors.New("native player command 20-22 palette phases unavailable")
		}
		for _, rgb := range phases {
			dac := append([]byte(nil), g.nativeMapDAC...)
			copy(dac[:3], rgb[:])
			palette, paletteErr := fdother.VGAPaletteFromDAC(dac)
			if paletteErr != nil {
				return paletteErr
			}
			indexed := image.NewPaletted(image.Rect(0, 0, 320, 200), palette)
			copy(indexed.Pix, baselineVGA)
			frames = append(frames, nativeCompoundPresentedFrame{image: ebiten.NewImageFromImage(indexed), delay: 1})
		}
		frames[0].sound = g.sfxCommandModifier
	}
	if frames, err = appendNativeCompoundFrames(frames, effectImages, 1); err != nil {
		return err
	}
	effectSample := loadWav(assetPath(fmt.Sprintf("assets/sfx/battle_80_%02d.wav", schedule.EffectSample)))
	effectStart := len(frames) - len(effectImages)
	frames[effectStart].sound = effectSample
	for _, rawFrame := range schedule.ExtraSampleFrameIndices {
		if rawFrame <= 0 || rawFrame >= schedule.EffectFrames {
			return errors.New("native AI command 20-22 extra sample frame unavailable")
		}
		frames[effectStart+rawFrame].sound = effectSample
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
		return errors.New("native AI command 20-22 raw sample unavailable")
	}
	g.nativeAICommandModifier = &nativeAICommandModifierPresentationJob{actor: actor, plan2022: plan, frames: frames, publishAt: publishAt, endAt: len(frames), rngBefore: g.nativeRNGState, baselineWork: baselineWork, baselineVGA: baselineVGA, stageWork: postWork, stageVGA: postVGA, then2022: then}
	return nil
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
	if j.plan2022 != nil {
		_ = battle.AbortNativeAICommand2022(j.plan2022)
	} else {
		_ = battle.AbortNativeAICommandModifier(j.plan)
	}
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
		var err error
		if j.plan2022 != nil {
			err = battle.CompleteNativeAICommand2022(j.plan2022)
		} else {
			err = battle.CompleteNativeAICommandModifier(j.plan)
		}
		if err != nil {
			g.failNativeAICommandModifierPresentation(err)
			return
		}
		then, then2022 := j.then, j.then2022
		var result battle.NativeCommandModifierResult
		if j.plan != nil {
			result = j.plan.Result
		}
		plan2022 := j.plan2022
		g.nativeAICommandModifier = nil
		if then2022 != nil {
			then2022(plan2022)
		} else if then != nil {
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
		var err error
		if j.plan2022 != nil {
			err = battle.PublishNativeAICommand2022(j.plan2022)
		} else {
			err = battle.PublishNativeAICommandModifier(j.plan)
		}
		if err != nil {
			g.failNativeAICommandModifierPresentation(err)
			return
		}
		if j.plan2022 != nil {
			g.nativeRNGState = j.plan2022.RNGState
		} else {
			g.nativeRNGState = j.plan.Result.RNGState
		}
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
