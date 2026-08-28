package main

import (
	"errors"
	"fmt"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/wicanr2/fd2_re/remake/internal/battle"
	"github.com/wicanr2/fd2_re/remake/internal/battlepresent"
	"github.com/wicanr2/fd2_re/remake/internal/fdother"
	"github.com/wicanr2/fd2_re/remake/internal/figani"
	"github.com/wicanr2/fd2_re/remake/internal/indexedmap"
)

type nativeCompoundPresentedFrame struct {
	image       *ebiten.Image
	delay       int
	sound       []byte
	soundPlayed bool
}

type nativeCommand32PresentationJob struct {
	actor                        *battle.Unit
	plan                         *battle.NativeCompoundCommand32Plan
	frames                       []nativeCompoundPresentedFrame
	frame, repeat                int
	publishAt                    int
	published, drawn, holding    bool
	hold                         int
	rngBefore                    uint16
	baselineWork, baselineVGA    []byte
	postTransactionWork, postVGA []byte
	then                         func(battle.NativeCompoundCommand32Result)
}

func nativeCommand32ClonedState(st *battle.State, result battle.NativeCompoundCommand32Result) (*battle.State, error) {
	if st == nil || len(st.Units) == 0 {
		return nil, errors.New("native command32 cloned state unavailable")
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
	for index, target := range result.Targets {
		copied := byOriginal[target.Target]
		if copied == nil || target.HPAfter < 0 || target.HPAfter > copied.MaxHP {
			return nil, fmt.Errorf("native command32 cloned target %d unavailable", index)
		}
		copied.HP = target.HPAfter
	}
	return &clone, nil
}

func nativeCompoundImagesWithDAC(frames [][]byte, baselineDAC []byte, deltas []int, raw [3]byte) ([]*ebiten.Image, error) {
	if len(frames) != len(deltas) {
		return nil, errors.New("native command32 frame/DAC schedule mismatch")
	}
	out := make([]*ebiten.Image, 0, len(frames))
	for index, pixels := range frames {
		dac, err := fdother.InterpolateNativeCompoundDAC(baselineDAC, 0, 0xff, deltas[index], raw)
		if err != nil {
			return nil, err
		}
		palette, err := fdother.VGAPaletteFromDAC(dac)
		if err != nil {
			return nil, err
		}
		images, err := nativeCommand24IndexedImages([][]byte{pixels}, palette)
		if err != nil {
			return nil, err
		}
		out = append(out, images[0])
	}
	return out, nil
}

func appendNativeCompoundFrames(dst []nativeCompoundPresentedFrame, images []*ebiten.Image, delay int) ([]nativeCompoundPresentedFrame, error) {
	if len(images) == 0 || delay <= 0 {
		return nil, errors.New("native command32 presented frames unavailable")
	}
	for _, image := range images {
		if image == nil {
			return nil, errors.New("native command32 nil presented frame")
		}
		dst = append(dst, nativeCompoundPresentedFrame{image: image, delay: delay})
	}
	return dst, nil
}

func (g *Game) startNativeCommand32Presentation(actor, confirmed *battle.Unit, then func(battle.NativeCompoundCommand32Result)) error {
	if !g.nativeFullPresentationEnabled() || g == nil || g.st == nil || actor == nil || confirmed == nil {
		return errors.New("native command32 presentation context unavailable")
	}
	if g.nativeCmd32Presentation != nil || g.nativeHealPresentation != nil || g.nativeModifierPresentation != nil ||
		g.nativeCmd0Presentation != nil || g.nativeCmd1Presentation != nil || g.nativeCmd2Presentation != nil ||
		g.nativeCmd3Presentation != nil || g.nativeCmd5Presentation != nil || g.nativeCmd6Presentation != nil ||
		g.nativeCmd7Presentation != nil || g.nativeCmd8Presentation != nil || g.nativeCmd9Player != nil ||
		g.nativeCmd9AIPresentation != nil || g.nativeCmd1012 != nil || g.nativeCmd24Presentation != nil ||
		g.nativeCmd29Presentation != nil || g.indexedTransition != nil || g.atk != nil {
		return errors.New("native command32 presentation already active")
	}
	if !g.st.HasNativeMapViewState || len(g.nativeMapWork) != indexedmap.NativeUnitPresentWorkSize ||
		len(g.nativeMapVGA) != indexedmap.NativeMapVGASize || len(g.nativeMapDAC) != 256*3 ||
		!nativeMapAssetsAvailable(g.nativeMapAssets) {
		return errors.New("native command32 indexed map state unavailable")
	}
	plan, err := g.st.PlanNativeCompoundCommand32(actor, confirmed, g.nativeRNGState)
	if err != nil {
		return err
	}
	if len(plan.Result.Targets) == 0 {
		return errors.New("native command32 final target array is empty")
	}

	fdotherArchive := nativeFDOTHERPath()
	if fdotherArchive == "" {
		return errors.New("native command32 player-provided archives unavailable")
	}
	actorIdle, err := figani.LoadSeparatedResource(separatedAssetPath("animations"), actor.BattleFig*3)
	if err != nil || len(actorIdle.Frames) == 0 {
		return errors.New("native command32 actor idle FIGANI unavailable")
	}
	actorEffect, err := figani.LoadSeparatedResource(separatedAssetPath("animations"), actor.BattleFig*3+1)
	if err != nil {
		return err
	}
	firstTarget := plan.Result.Targets[0].Target
	if firstTarget == nil || !firstTarget.HasBattleFig {
		return errors.New("native command32 first target BattleFig unavailable")
	}
	targetIdle, err := figani.LoadSeparatedResource(separatedAssetPath("animations"), firstTarget.BattleFig*3)
	if err != nil || len(targetIdle.Frames) == 0 {
		return errors.New("native command32 target idle FIGANI unavailable")
	}
	commonEffect, err := figani.LoadSeparatedArchiveResource(separatedAssetPath("animations"), "FDOTHER.DAT", 65)
	if err != nil {
		return err
	}
	commonSchedule, err := figani.BuildNativeCompoundPresentationSchedule(32, commonEffect)
	if err != nil {
		return err
	}
	tailSchedule, err := fdother.BuildNativeCommand32TailSchedule(g.nativeMapAssets.FDOTHER6, g.nativeMapAssets.CommandHealDigits)
	if err != nil {
		return err
	}
	paletteDAC, battlePalette, err := loadNativeBattlePalette()
	if err != nil || len(paletteDAC) != 256*3 {
		return errors.New("native command32 battle DAC unavailable")
	}

	actorSelector, err := nativeCommand24BGSelector(g.m, actor)
	if err != nil {
		return err
	}
	background, err := fdother.LoadSeparatedSingleFrame(separatedAssetPath("surfaces"), "BG.DAT", actorSelector)
	if err != nil {
		return err
	}
	panelAssets, err := battle.LoadNativeItemPanelDataAssets(separatedAssetPath(""))
	if err != nil {
		return err
	}
	actorRecord, err := battle.NativeBattlePanelRecordForUnit(actor)
	if err != nil {
		return err
	}
	actorIndex, err := nativeCommand24RuntimeUnitIndex(g.st, actor)
	if err != nil {
		return err
	}
	var platform *fdother.Frame
	if actor.NativeRecordByte6 != 0 {
		frame, decodeErr := fdother.LoadSeparatedSingleFrame(separatedAssetPath("surfaces"), "TAI.DAT", actorSelector)
		if decodeErr != nil {
			return decodeErr
		}
		platform = &frame
	}
	actorBase, err := nativeCommand24BackgroundBase(background, panelAssets, actorRecord, actorIndex, g.handlerChapter, platform)
	if err != nil {
		return err
	}
	preludeBase, err := nativeCommand24BackgroundBase(background, panelAssets, actorRecord, actorIndex, g.handlerChapter, nil)
	if err != nil {
		return err
	}
	prelude, err := battlepresent.BuildNativeCommandPreludeFrames(battlepresent.NativeCommandPreludeInput{
		Base: preludeBase, ActorIdle: actorIdle.Frames[0], Platform: platform,
		RawSide: actor.NativeRecordByte6, Mode: 1, BaselineDAC: paletteDAC,
	})
	if err != nil {
		return err
	}
	preludeImages, err := nativeCommand24PreludeImages(prelude)
	if err != nil {
		return err
	}
	actorPixels, err := battlepresent.BuildNativeCompoundActorFrames(actorBase, actorEffect, targetIdle, actor.NativeRecordByte6)
	if err != nil {
		return err
	}
	actorImages, err := nativeCommand24IndexedImages(actorPixels, battlePalette)
	if err != nil {
		return err
	}
	common, err := battlepresent.BuildNativeCompoundCommonFrames(actorBase, actorEffect.Frames[0], commonEffect, commonSchedule)
	if err != nil {
		return err
	}
	actorSlide, err := nativeCommand24IndexedImages(common.ActorSlide, battlePalette)
	if err != nil {
		return err
	}
	effectSlide, err := nativeCommand24IndexedImages(common.EffectSlide, battlePalette)
	if err != nil {
		return err
	}
	mainImages, err := nativeCommand24IndexedImages(common.Main, battlePalette)
	if err != nil {
		return err
	}
	tailDeltas := make([]int, len(common.Tail))
	for index := range tailDeltas {
		tailDeltas[index] = 40 - 4*index
	}
	commonTailImages, err := nativeCompoundImagesWithDAC(common.Tail, paletteDAC, tailDeltas, commonSchedule.PaletteRGB)
	if err != nil {
		return err
	}

	targetUnits := make([]*battle.Unit, len(plan.Result.Targets))
	results := make([]battlepresent.NativeCommand32TailResult, len(plan.Result.Targets))
	for index, target := range plan.Result.Targets {
		targetUnits[index] = target.Target
		recordIndex, indexErr := nativeCommand24RuntimeUnitIndex(g.st, target.Target)
		if indexErr != nil {
			return indexErr
		}
		results[index] = battlepresent.NativeCommand32TailResult{TargetRecord: recordIndex, Hit: target.Damage.Hit, Damage: target.Damage.Damage}
	}
	tailTargets, err := nativeCommandHealTailTargets(g.st, targetUnits)
	if err != nil {
		return err
	}
	clonedState, err := nativeCommand32ClonedState(g.st, plan.Result)
	if err != nil {
		return err
	}
	hud, ok := g.nativeMapHUDInput()
	if !ok {
		return errors.New("native command32 post-transaction HUD unavailable")
	}
	postInput, err := buildNativeMapFrameInput(g.nativeMapAssets, g.m, clonedState, nativeMapFrameRuntime{HUD: hud})
	if err != nil {
		return err
	}
	postWork := append([]byte(nil), g.nativeMapWork...)
	postVGA := append([]byte(nil), g.nativeMapVGA...)
	if err := indexedmap.ComposeNativeFrame(postWork, postVGA, postInput); err != nil {
		return err
	}
	view := g.st.NativeMapViewState
	tail, err := battlepresent.BuildNativeCommand32TailFrames(
		g.nativeMapWork, g.nativeMapVGA, postWork, postVGA,
		g.nativeMapAssets.FDOTHER6, g.nativeMapAssets.CommandHealDigits,
		tailTargets, results, view.CameraX, view.CameraY, tailSchedule,
	)
	if err != nil {
		return err
	}
	mapPalette, err := fdother.VGAPaletteFromDAC(g.nativeMapDAC)
	if err != nil {
		return err
	}
	tailEffectImages, err := nativeCommand24IndexedImages(tail.Effect, mapPalette)
	if err != nil {
		return err
	}
	toggleImages, err := nativeCommand24IndexedImages(tail.Toggle, mapPalette)
	if err != nil {
		return err
	}
	resultImages, err := nativeCommand24IndexedImages(tail.Result, mapPalette)
	if err != nil {
		return err
	}
	rampPixels, rampDeltas := make([][]byte, 0, 41), make([]int, 0, 41)
	for delta := 0; delta <= 40; delta++ {
		rampPixels = append(rampPixels, append([]byte(nil), g.nativeMapVGA...))
		rampDeltas = append(rampDeltas, delta)
	}
	rampImages, err := nativeCompoundImagesWithDAC(rampPixels, g.nativeMapDAC, rampDeltas, commonSchedule.PaletteRGB)
	if err != nil {
		return err
	}

	frames := make([]nativeCompoundPresentedFrame, 0)
	if frames, err = appendNativeCompoundFrames(frames, preludeImages, 1); err != nil {
		return err
	}
	if frames, err = appendNativeCompoundFrames(frames, actorImages, 1); err != nil {
		return err
	}
	frames[len(frames)-1].delay += commonSchedule.PreludeHoldTicks
	if frames, err = appendNativeCompoundFrames(frames, actorSlide, 1); err != nil {
		return err
	}
	if frames, err = appendNativeCompoundFrames(frames, effectSlide, 1); err != nil {
		return err
	}
	mainStart := len(frames)
	if frames, err = appendNativeCompoundFrames(frames, mainImages, commonSchedule.MainDelayTicks); err != nil {
		return err
	}
	frames[mainStart].sound = loadWav(assetPath("assets/sfx/battle_91_02.wav"))
	tailStart := len(frames)
	if frames, err = appendNativeCompoundFrames(frames, commonTailImages, commonSchedule.MainDelayTicks); err != nil {
		return err
	}
	commonSample1 := loadWav(assetPath("assets/sfx/battle_91_01.wav"))
	for index := 0; index < len(commonTailImages); index += 2 {
		frames[tailStart+index].sound = commonSample1
	}
	if frames, err = appendNativeCompoundFrames(frames, rampImages, 6); err != nil {
		return err
	}
	effectStart := len(frames)
	if frames, err = appendNativeCompoundFrames(frames, tailEffectImages, tailSchedule.EffectDelayTicks); err != nil {
		return err
	}
	frames[effectStart].sound = loadWav(assetPath("assets/sfx/battle_80_09.wav"))
	if frames, err = appendNativeCompoundFrames(frames, toggleImages, nativeDelayTicks(tailSchedule.ToggleDelayMS)); err != nil {
		return err
	}
	publishAt := len(frames)
	if frames, err = appendNativeCompoundFrames(frames, resultImages, 1); err != nil {
		return err
	}
	if !osMuteOrShot(g) {
		for _, required := range [][]byte{frames[mainStart].sound, commonSample1, frames[effectStart].sound} {
			if len(required) == 0 {
				return errors.New("native command32 required raw sample unavailable")
			}
		}
	}
	g.nativeCmd32Presentation = &nativeCommand32PresentationJob{
		actor: actor, plan: plan, frames: frames, publishAt: publishAt,
		rngBefore:    g.nativeRNGState,
		baselineWork: append([]byte(nil), g.nativeMapWork...), baselineVGA: append([]byte(nil), g.nativeMapVGA...),
		postTransactionWork: postWork, postVGA: postVGA, then: then,
	}
	return nil
}

func (g *Game) cancelNativeCommand32Presentation() {
	j := g.nativeCmd32Presentation
	if j == nil {
		return
	}
	_ = battle.AbortNativeCompoundCommand32(j.plan)
	g.nativeRNGState = j.rngBefore
	g.nativeMapWork = append(g.nativeMapWork[:0], j.baselineWork...)
	g.nativeMapVGA = append(g.nativeMapVGA[:0], j.baselineVGA...)
	g.nativeCmd32Presentation = nil
}

func (g *Game) failNativeCommand32Presentation(err error) {
	g.cancelNativeCommand32Presentation()
	g.loadErr = "native command32 presentation: " + err.Error()
}

func (g *Game) stepNativeCommand32Presentation() {
	j := g.nativeCmd32Presentation
	if j == nil {
		return
	}
	if j.holding {
		if j.hold > 0 {
			j.hold--
			return
		}
		if err := battle.CompleteNativeCompoundCommand32(j.plan); err != nil {
			g.failNativeCommand32Presentation(err)
			return
		}
		then, result := j.then, j.plan.Result
		g.nativeCmd32Presentation = nil
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
		for index := range j.plan.Result.Targets {
			if err := battle.ApplyNativeCompoundCommand32Target(j.plan, index); err != nil {
				g.failNativeCommand32Presentation(err)
				return
			}
		}
		g.nativeRNGState = j.plan.Result.RNGState
		g.nativeMapWork = append(g.nativeMapWork[:0], j.postTransactionWork...)
		g.nativeMapVGA = append(g.nativeMapVGA[:0], j.postVGA...)
		j.published = true
	}
	j.frame = next
	if j.frame >= len(j.frames) {
		if !j.published {
			g.failNativeCommand32Presentation(errors.New("required damage boundary was not presented"))
			return
		}
		j.frame = len(j.frames) - 1
		j.holding, j.hold = true, nativeDelayTicks(500)
	}
}

func (g *Game) drawNativeCommand32Presentation(screen *ebiten.Image) bool {
	j := g.nativeCmd32Presentation
	if j == nil || j.frame < 0 || j.frame >= len(j.frames) || j.frames[j.frame].image == nil {
		return false
	}
	op := &ebiten.DrawImageOptions{}
	op.GeoM.Scale(2, 2)
	screen.DrawImage(j.frames[j.frame].image, op)
	j.drawn = true
	return true
}
