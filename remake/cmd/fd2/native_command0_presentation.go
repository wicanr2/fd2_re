package main

import (
	"encoding/binary"
	"errors"
	"fmt"
	"image"
	"image/color"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/wicanr2/fd2_re/remake/internal/battle"
	"github.com/wicanr2/fd2_re/remake/internal/battlepresent"
	"github.com/wicanr2/fd2_re/remake/internal/fdicon"
	"github.com/wicanr2/fd2_re/remake/internal/fdother"
	"github.com/wicanr2/fd2_re/remake/internal/figani"
)

type nativeCommand0PresentationPhase uint8

const (
	nativeCommand0Prelude nativeCommand0PresentationPhase = iota
	nativeCommand0Actor
	nativeCommand0Target
	nativeCommand0Tail
)

type nativeCommand0PresentationJob struct {
	actor, target    *battle.Unit
	plan             *battle.NativeCommandDamagePlan
	prelude          []*ebiten.Image
	actorBlack       []*ebiten.Image
	actorPulse       []*ebiten.Image
	actorSpecs       []battlepresent.NativeCommand0ActorFrame
	targetFrames     []*ebiten.Image
	targetImpact     []bool
	tail             []*ebiten.Image
	phase            nativeCommand0PresentationPhase
	frame            int
	pulseBlack       bool
	drawn            bool
	mpPublished      bool
	damageStages     int
	actorMPBefore    int
	actorActedBefore bool
	targetHPBefore   int
	then             func([]battle.NativeCommandDamageResult)
}

func nativeCommand0Control(m *MapData, unit *battle.Unit) ([4]byte, error) {
	var out [4]byte
	if m == nil || unit == nil || m.W <= 0 || m.H <= 0 || len(m.Tiles) != m.W*m.H ||
		len(m.NativeTerrainControl) == 0 || len(m.NativeTerrainControl)%4 != 0 ||
		unit.X < 0 || unit.Y < 0 || unit.X >= m.W || unit.Y >= m.H {
		return out, errors.New("native command0 terrain control unavailable")
	}
	tile := m.Tiles[unit.Y*m.W+unit.X]
	if tile < 0 || tile > 0x3ff || tile*4+4 > len(m.NativeTerrainControl) {
		return out, errors.New("native command0 terrain control out of range")
	}
	copy(out[:], m.NativeTerrainControl[tile*4:tile*4+4])
	return out, nil
}

func nativeCommand0ActorEffect(path string, selector int) (*figani.Animation, error) {
	if selector < 0 || selector > 0xff {
		return nil, errors.New("native command0 actor FIGANI selector unavailable")
	}
	resource := selector*3 + 2
	raw, err := fdother.ReadResource(path, resource)
	if err != nil {
		return nil, err
	}
	if len(raw) < 2 {
		return nil, errors.New("native command0 actor effect header unavailable")
	}
	if binary.LittleEndian.Uint16(raw[:2]) == 0 {
		resource--
		raw, err = fdother.ReadResource(path, resource)
		if err != nil {
			return nil, err
		}
	}
	return figani.Parse(raw)
}

func nativeCommand0Base(
	background fdother.Frame,
	panelAssets battle.NativeItemPanelDataAssets,
	actorRecord, targetRecord []byte,
	actorIndex, targetIndex, chapter int,
	platform *fdother.Frame,
) ([]byte, error) {
	base := make([]byte, 320*200)
	if err := battle.RenderNativeBattlePanel(panelAssets, actorRecord, base, actorIndex, chapter); err != nil {
		return nil, err
	}
	if err := battle.RenderNativeBattlePanel(panelAssets, targetRecord, base, targetIndex, chapter); err != nil {
		return nil, err
	}
	background.X, background.Y = 0, 50
	if err := background.Blit(base, 320, -1); err != nil {
		return nil, err
	}
	if platform != nil {
		frame := *platform
		frame.X, frame.Y = 164, 157
		if err := frame.Blit(base, 320, -1); err != nil {
			return nil, err
		}
	}
	return base, nil
}

func nativeCommand0Image(pixels []byte, palette color.Palette) (*ebiten.Image, error) {
	if len(pixels) != 320*200 || len(palette) != 256 {
		return nil, errors.New("native command0 indexed image unavailable")
	}
	img := image.NewPaletted(image.Rect(0, 0, 320, 200), palette)
	copy(img.Pix, pixels)
	return ebiten.NewImageFromImage(img), nil
}

func nativeCommand0Images(pixels [][]byte, palette color.Palette) ([]*ebiten.Image, error) {
	out := make([]*ebiten.Image, len(pixels))
	for i := range pixels {
		var err error
		out[i], err = nativeCommand0Image(pixels[i], palette)
		if err != nil {
			return nil, fmt.Errorf("native command0 image %d: %w", i, err)
		}
	}
	return out, nil
}

func nativeCommand0TargetAndTailPixels(
	plan *battle.NativeCommandDamagePlan,
	schedule figani.NativeCommand0PresentationSchedule,
	effect, actorEffect, targetIdle *figani.Animation,
	bases [][]byte,
	background, platform fdother.Frame,
	luts [][]byte,
) ([][]byte, []bool, [][]byte, error) {
	if plan == nil || len(plan.Results) != 1 || len(bases) != 8 || effect == nil ||
		actorEffect == nil || len(actorEffect.Frames) == 0 || targetIdle == nil || len(targetIdle.Frames) == 0 || len(luts) <= 14 {
		return nil, nil, nil, errors.New("native command0 target/tail inputs unavailable")
	}
	for i, frame := range targetIdle.Frames {
		if frame.Delay <= 0 {
			return nil, nil, nil, fmt.Errorf("native command0 target idle delay %d", i)
		}
	}
	targetFrame, targetRepeat, stage := 0, 0, 0
	frames := make([][]byte, 0, schedule.Frames)
	impacts := make([]bool, 0, schedule.Frames)
	for step := 0; step < schedule.Frames; step++ {
		pixels, err := battlepresent.ComposeNativeCommand0TargetFrame(
			bases[stage], targetIdle.Frames[targetFrame], effect, schedule, step,
		)
		if err != nil {
			return nil, nil, nil, err
		}
		impact := false
		for layer := 0; layer < 7; layer++ {
			impact = impact || figani.NativeCommand0Impact(step, layer)
		}
		frames, impacts = append(frames, pixels), append(impacts, impact)
		if impact {
			stage++
		}
		targetRepeat++
		if targetRepeat >= targetIdle.Frames[targetFrame].Delay {
			targetRepeat = 0
			targetFrame = (targetFrame + 1) % len(targetIdle.Frames)
		}
	}
	if stage != figani.NativeCommand0DamageStages {
		return nil, nil, nil, fmt.Errorf("native command0 target impact count=%d", stage)
	}
	tail := make([][]byte, 0, 4)
	for pass := 0; pass < 4; pass++ {
		pixels := append([]byte(nil), bases[7]...)
		bg, tai := background, platform
		bg.X, bg.Y, tai.X, tai.Y = 0, 50, 164, 157
		var err error
		if pass < 3 {
			err = bg.BlitLUTAt(pixels, 320, 0, luts[12+pass])
			if err == nil {
				err = tai.BlitLUTAt(pixels, 320, 0, luts[12+pass])
			}
		} else {
			err = bg.Blit(pixels, 320, -1)
			if err == nil {
				err = tai.Blit(pixels, 320, -1)
			}
		}
		if err != nil {
			return nil, nil, nil, err
		}
		if err := actorEffect.Frames[0].BlitAt(pixels, 320); err != nil {
			return nil, nil, nil, err
		}
		if err := targetIdle.Frames[targetFrame].BlitAt(pixels, 320); err != nil {
			return nil, nil, nil, err
		}
		tail = append(tail, pixels)
		targetRepeat++
		if targetRepeat >= targetIdle.Frames[targetFrame].Delay {
			targetRepeat = 0
			targetFrame = (targetFrame + 1) % len(targetIdle.Frames)
		}
	}
	return frames, impacts, tail, nil
}

func (g *Game) startNativeCommand0Presentation(actor, target *battle.Unit, then func([]battle.NativeCommandDamageResult)) error {
	if !g.nativeFullPresentationEnabled() {
		return errors.New("native abbreviated presentation owner unavailable")
	}
	if g == nil || g.st == nil || actor == nil || target == nil || g.nativeCommandScene == nil ||
		g.nativeCommandPaletteFlash == nil || g.nativeCmd0Presentation != nil ||
		g.nativeCmd1Presentation != nil || g.nativeCmd2Presentation != nil || g.nativeCmd5Presentation != nil || g.nativeCmd6Presentation != nil ||
		g.nativeCmd24Presentation != nil || g.nativeCmd29Presentation != nil || g.nativeHealPresentation != nil ||
		g.nativeModifierPresentation != nil || g.atk != nil {
		return errors.New("native command0 presentation context unavailable")
	}
	if !actor.HasBattleFig || !actor.HasNativeRecordByte6 || !target.HasBattleFig ||
		!target.HasNativeRecordByte6 || len(g.nativeUIPalette) != 256 || len(g.nativeMapAssets.LUTs) <= 14 {
		return errors.New("native command0 raw scene provenance unavailable")
	}
	plan, err := g.st.PlanBoundNativeCommand0(actor, target, g.nativeRNGState)
	if err != nil {
		return err
	}
	if len(plan.Results) != 1 || plan.Results[0].Target != target ||
		len(g.st.NativeCommandBook) != battle.NativeCommandRecordCount || g.st.NativeCommandBook[0].EffectMode != 0 {
		return errors.New("native command0 fixed single-target scene unavailable")
	}
	initial, err := g.nativeCommandScene.InitialBackground(g.handlerChapter)
	if err != nil {
		return err
	}
	actorControl, err := nativeCommand0Control(g.m, actor)
	if err != nil {
		return err
	}
	targetControl, err := nativeCommand0Control(g.m, target)
	if err != nil {
		return err
	}
	actorGate, err := battle.NativeCommandBackgroundGate(actor)
	if err != nil {
		return err
	}
	targetGate, err := battle.NativeCommandBackgroundGate(target)
	if err != nil {
		return err
	}
	actorSelector := initial
	if !actorGate || actorSelector == 0 {
		actorSelector = actorControl[2]
	}
	targetSelector := fdicon.NativeCommandBackgroundSelector(initial, []fdicon.NativeCommandBackgroundTarget{{Gate: targetGate, Control: targetControl}})
	bgSelector, taiSelector := targetSelector, actorSelector
	if actor.NativeRecordByte6 == 0 {
		bgSelector, taiSelector = actorSelector, targetSelector
	}
	figaniPath, bgPath, taiPath, fdotherPath, fdtxtPath := nativeFIGANIPath(), nativeBGPath(), nativeTAIPath(), nativeFDOTHERPath(), nativeFDTXTPath()
	if figaniPath == "" || bgPath == "" || taiPath == "" || fdotherPath == "" || fdtxtPath == "" {
		return errors.New("native command0 player-provided archives unavailable")
	}
	background, err := fdother.DecodeArchiveSingleFrame(bgPath, int(bgSelector))
	if err != nil {
		return err
	}
	platform, err := fdother.DecodeArchiveSingleFrame(taiPath, int(taiSelector))
	if err != nil {
		return err
	}
	actorIdle, err := figani.DecodeResource(figaniPath, actor.BattleFig*3)
	if err != nil {
		return err
	}
	actorEffect, err := nativeCommand0ActorEffect(figaniPath, actor.BattleFig)
	if err != nil {
		return err
	}
	targetIdle, err := figani.DecodeResource(figaniPath, target.BattleFig*3)
	if err != nil {
		return err
	}
	effectResource := 18
	if actor.NativeRecordByte6 == 0 {
		effectResource = 20
	}
	effect, err := figani.DecodeResource(fdotherPath, effectResource)
	if err != nil {
		return err
	}
	schedule, err := figani.BuildNativeCommand0PresentationSchedule(actor.NativeRecordByte6, effect)
	if err != nil {
		return err
	}
	if sample0, e := fdother.ReadNestedResource(fdotherPath, schedule.SoundResource, 0); e != nil || len(sample0) == 0 {
		return errors.New("native command0 FDOTHER #82 sub0 unavailable")
	}
	if sample1, e := fdother.ReadNestedResource(fdotherPath, schedule.SoundResource, schedule.SoundIndex); e != nil || len(sample1) == 0 {
		return errors.New("native command0 FDOTHER #82 sub1 unavailable")
	}
	if !osMuteOrShot(g) && (len(g.sfxCommand0Actor) == 0 || len(g.sfxCommand0Target) == 0) {
		return errors.New("native command0 converted #82 samples unavailable")
	}
	panelAssets, err := battle.LoadNativeItemPanelDataAssets(fdotherPath, fdtxtPath)
	if err != nil {
		return err
	}
	dac, err := fdother.ReadResource(fdotherPath, 0)
	if err != nil {
		return err
	}
	actorRecord, err := battle.NativeBattlePanelRecordForUnit(actor)
	if err != nil {
		return err
	}
	actorAfter := *actor
	actorAfter.MP = plan.MPAfter
	actorAfterRecord, err := battle.NativeBattlePanelRecordForUnit(&actorAfter)
	if err != nil {
		return err
	}
	targetIndex, err := nativeCommand24RuntimeUnitIndex(g.st, target)
	if err != nil {
		return err
	}
	actorIndex, err := nativeCommand24RuntimeUnitIndex(g.st, actor)
	if err != nil {
		return err
	}
	targetRecords := make([][]byte, 8)
	bases := make([][]byte, 8)
	for stage := 0; stage <= 7; stage++ {
		staged := *target
		staged.HP = plan.Results[0].HPBefore - (plan.Results[0].HPBefore-plan.Results[0].HPAfter)*stage/7
		targetRecords[stage], err = battle.NativeBattlePanelRecordForUnit(&staged)
		if err != nil {
			return err
		}
		bases[stage], err = nativeCommand0Base(background, panelAssets, actorAfterRecord, targetRecords[stage], actorIndex, targetIndex, g.handlerChapter, &platform)
		if err != nil {
			return err
		}
	}
	preludeBase, err := nativeCommand0Base(background, panelAssets, actorRecord, targetRecords[0], actorIndex, targetIndex, g.handlerChapter, nil)
	if err != nil {
		return err
	}
	actorBaseBefore, err := nativeCommand0Base(background, panelAssets, actorRecord, targetRecords[0], actorIndex, targetIndex, g.handlerChapter, &platform)
	if err != nil {
		return err
	}
	prelude, err := battlepresent.BuildNativeCommandPreludeFrames(battlepresent.NativeCommandPreludeInput{
		Base: preludeBase, ActorIdle: actorIdle.Frames[0], FirstTargetIdle: targetIdle.Frames[0],
		Platform: &platform, RawSide: actor.NativeRecordByte6, Mode: 0, BaselineDAC: dac,
	})
	if err != nil {
		return err
	}
	preludeImages, err := nativeCommand24PreludeImages(prelude)
	if err != nil {
		return err
	}
	actorSpecs, err := battlepresent.BuildNativeCommand0ActorFrames(battlepresent.NativeCommand0ActorInput{
		BaseBefore: actorBaseBefore, BaseAfter: bases[0], ActorEffect: actorEffect,
		FirstTargetIdle: targetIdle, RawSide: actor.NativeRecordByte6,
		Background: background, Platform: platform, LUT: g.nativeMapAssets.LUTs[11],
	})
	if err != nil {
		return err
	}
	actorPixels := make([][]byte, len(actorSpecs))
	for i := range actorSpecs {
		actorPixels[i] = actorSpecs[i].Pixels
	}
	actorBlack, err := nativeCommand0Images(actorPixels, g.nativeUIPalette)
	if err != nil {
		return err
	}
	pulseDAC := append([]byte(nil), dac...)
	if len(pulseDAC) != 256*3 || len(g.nativeCommandPaletteFlash.Entries) != battle.NativeCommandRecordCount {
		return errors.New("native command0 DAC inputs unavailable")
	}
	copy(pulseDAC[:3], g.nativeCommandPaletteFlash.Entries[0][:])
	pulsePalette, err := fdother.VGAPaletteFromDAC(pulseDAC)
	if err != nil {
		return err
	}
	actorPulse, err := nativeCommand0Images(actorPixels, pulsePalette)
	if err != nil {
		return err
	}
	targetPixels, impacts, tailPixels, err := nativeCommand0TargetAndTailPixels(plan, schedule, effect, actorEffect, targetIdle, bases, background, platform, g.nativeMapAssets.LUTs)
	if err != nil {
		return err
	}
	targetImages, err := nativeCommand0Images(targetPixels, g.nativeUIPalette)
	if err != nil {
		return err
	}
	tailImages, err := nativeCommand0Images(tailPixels, g.nativeUIPalette)
	if err != nil {
		return err
	}
	g.nativeCmd0Presentation = &nativeCommand0PresentationJob{
		actor: actor, target: target, plan: plan, prelude: preludeImages,
		actorBlack: actorBlack, actorPulse: actorPulse, actorSpecs: actorSpecs,
		targetFrames: targetImages, targetImpact: impacts, tail: tailImages,
		actorMPBefore: actor.MP, actorActedBefore: actor.Acted, targetHPBefore: target.HP,
		then: then,
	}
	return nil
}

func (g *Game) failNativeCommand0Presentation(err error) {
	j := g.nativeCmd0Presentation
	if j == nil {
		return
	}
	j.actor.MP, j.actor.Acted, j.target.HP = j.actorMPBefore, j.actorActedBefore, j.targetHPBefore
	g.nativeCmd0Presentation = nil
	g.loadErr = "native command0 presentation: " + err.Error()
}

func (g *Game) stepNativeCommand0Presentation() {
	j := g.nativeCmd0Presentation
	if j == nil || !j.drawn {
		return
	}
	j.drawn = false
	switch j.phase {
	case nativeCommand0Prelude:
		j.frame++
		if j.frame >= len(j.prelude) {
			j.phase, j.frame = nativeCommand0Actor, 0
		}
	case nativeCommand0Actor:
		spec := j.actorSpecs[j.frame]
		if spec.Pulse && !j.pulseBlack {
			if spec.PublishMP && !j.mpPublished {
				if err := battle.ApplyNativeCommandDamageMP(j.plan); err != nil {
					g.failNativeCommand0Presentation(err)
					return
				}
				j.mpPublished = true
				g.playRaw(g.sfxCommand0Actor)
			}
			j.pulseBlack = true
			return
		}
		if spec.PublishMP && !j.mpPublished {
			if err := battle.ApplyNativeCommandDamageMP(j.plan); err != nil {
				g.failNativeCommand0Presentation(err)
				return
			}
			j.mpPublished = true
			g.playRaw(g.sfxCommand0Actor)
		}
		j.pulseBlack = false
		j.frame++
		if j.frame >= len(j.actorBlack) {
			if !j.mpPublished {
				g.failNativeCommand0Presentation(errors.New("actor MP marker was not presented"))
				return
			}
			j.phase, j.frame = nativeCommand0Target, 0
		}
	case nativeCommand0Target:
		if j.targetImpact[j.frame] {
			j.damageStages++
			if err := battle.ApplyNativeCommandDamageStage(j.plan, 0, j.damageStages); err != nil {
				g.failNativeCommand0Presentation(err)
				return
			}
			g.playRaw(g.sfxCommand0Target)
		}
		j.frame++
		if j.frame >= len(j.targetFrames) {
			if j.damageStages != figani.NativeCommand0DamageStages {
				g.failNativeCommand0Presentation(errors.New("target impact stages incomplete"))
				return
			}
			j.phase, j.frame = nativeCommand0Tail, 0
		}
	case nativeCommand0Tail:
		j.frame++
		if j.frame < len(j.tail) {
			return
		}
		if err := battle.CompleteNativeCommandDamage(j.plan); err != nil {
			g.failNativeCommand0Presentation(err)
			return
		}
		g.nativeRNGState = j.plan.RNGAfter
		then, results := j.then, append([]battle.NativeCommandDamageResult(nil), j.plan.Results...)
		g.nativeCmd0Presentation = nil
		if then != nil {
			then(results)
		}
	}
}

func (g *Game) drawNativeCommand0Presentation(screen *ebiten.Image) bool {
	j := g.nativeCmd0Presentation
	if j == nil || screen == nil {
		return false
	}
	var frame *ebiten.Image
	switch j.phase {
	case nativeCommand0Prelude:
		if j.frame < 0 || j.frame >= len(j.prelude) {
			return false
		}
		frame = j.prelude[j.frame]
	case nativeCommand0Actor:
		if j.frame < 0 || j.frame >= len(j.actorBlack) || j.frame >= len(j.actorPulse) || j.frame >= len(j.actorSpecs) {
			return false
		}
		frame = j.actorBlack[j.frame]
		if j.actorSpecs[j.frame].Pulse && !j.pulseBlack {
			frame = j.actorPulse[j.frame]
		}
	case nativeCommand0Target:
		if j.frame < 0 || j.frame >= len(j.targetFrames) {
			return false
		}
		frame = j.targetFrames[j.frame]
	case nativeCommand0Tail:
		if j.frame < 0 || j.frame >= len(j.tail) {
			return false
		}
		frame = j.tail[j.frame]
	}
	if frame == nil {
		return false
	}
	op := &ebiten.DrawImageOptions{}
	op.GeoM.Scale(2, 2)
	screen.DrawImage(frame, op)
	j.drawn = true
	return true
}
