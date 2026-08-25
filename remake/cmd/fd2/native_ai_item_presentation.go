package main

import (
	"errors"
	"fmt"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/wicanr2/fd2_re/remake/internal/battle"
	"github.com/wicanr2/fd2_re/remake/internal/fdother"
	"github.com/wicanr2/fd2_re/remake/internal/indexedmap"
)

type nativeAIItemPresentationJob struct {
	transaction                  *nativeAIItemTransaction
	frames                       []nativeCompoundPresentedFrame
	frame, repeat                int
	publishAt                    int
	published, drawn, holding    bool
	hold                         int
	baselineWork, baselineVGA    []byte
	postTransactionWork, postVGA []byte
	then                         func()
}

func nativeAIItemClonedState(st *battle.State, tx *nativeAIItemTransaction) (*battle.State, error) {
	if st == nil || tx == nil || len(tx.after) != len(st.Units)*80 {
		return nil, errors.New("native AI item cloned state unavailable")
	}
	clone := *st
	clone.Units = make([]*battle.Unit, len(st.Units))
	for index, unit := range st.Units {
		if unit == nil {
			return nil, fmt.Errorf("native AI item cloned unit %d unavailable", index)
		}
		copied := *unit
		syncNativeItemRuntimeRecord(&copied, tx.after[index*80:(index+1)*80])
		clone.Units[index] = &copied
	}
	return &clone, nil
}

// startNativeAIItemRestorePresentation owns only the 0x211A4 tail reached
// directly by item types 5/13. It intentionally excludes command33's
// 0x27FC9 prelude and keeps the detached transaction private through the
// final 0x1C2DA mask Draw boundary.
func (g *Game) startNativeAIItemRestorePresentation(plan *battle.AIPlan, then func()) (bool, error) {
	if g == nil || g.st == nil || plan == nil || plan.U == nil {
		return false, errors.New("native AI item presentation context unavailable")
	}
	tx, err := g.planNativeAITargetItem(plan)
	if err != nil {
		return false, err
	}
	if tx.restore == nil {
		if tx.damageRoute != nil && tx.damageRoute.Presentation == 0x1cd17 {
			return g.startNativeAIItemDamagePresentation(tx, then)
		}
		return false, nil
	}
	if !g.nativeFullPresentationEnabled() {
		return false, errors.New("native AI item indexed presentation unavailable")
	}
	if g.nativeAIItemPresentation != nil || g.nativeHealPresentation != nil ||
		g.nativeModifierPresentation != nil || g.nativeAICommandModifier != nil ||
		g.nativeCmd0Presentation != nil || g.nativeCmd1Presentation != nil ||
		g.nativeCmd2Presentation != nil || g.nativeCmd3Presentation != nil ||
		g.nativeCmd5Presentation != nil || g.nativeCmd6Presentation != nil ||
		g.nativeCmd7Presentation != nil || g.nativeCmd8Presentation != nil ||
		g.nativeCmd9Player != nil || g.nativeCmd9AIPresentation != nil ||
		g.nativeCmd1012 != nil || g.nativeCmd24Presentation != nil ||
		g.nativeCmd29Presentation != nil || g.nativeCmd32Presentation != nil ||
		g.nativeCmd33Presentation != nil || g.nativeCmd34Presentation != nil ||
		g.nativeCmd35Presentation != nil || g.indexedTransition != nil || g.atk != nil {
		return false, errors.New("native AI item presentation already active")
	}
	if !g.st.HasNativeMapViewState || len(g.nativeMapWork) != indexedmap.NativeUnitPresentWorkSize ||
		len(g.nativeMapVGA) != indexedmap.NativeMapVGASize || len(g.nativeMapDAC) != 256*3 ||
		!nativeMapAssetsAvailable(g.nativeMapAssets) {
		return false, errors.New("native AI item indexed map state unavailable")
	}
	schedule, err := fdother.BuildNativeCommand33TailSchedule()
	if err != nil {
		return false, err
	}
	a := g.nativeMapAssets
	if len(a.FDOTHER6) < schedule.EffectStart+schedule.EffectFrames ||
		len(a.CommandHealDigits) <= schedule.DigitBias+9 {
		return false, errors.New("native AI item tail descriptors unavailable")
	}
	view := g.st.NativeMapViewState
	tailTargets, err := nativeCommandHealTailTargets(g.st, tx.targets)
	if err != nil {
		return false, err
	}
	effectTargets := make([]indexedmap.NativeCommandHealTailTarget, 0, len(tailTargets))
	for _, target := range tailTargets {
		if target.X >= view.CameraX-1 && target.X <= view.CameraX+12 &&
			target.Y >= view.CameraY-1 && target.Y <= view.CameraY+8 {
			effectTargets = append(effectTargets, target)
		}
	}
	baselineWork := append([]byte(nil), g.nativeMapWork...)
	baselineVGA := append([]byte(nil), g.nativeMapVGA...)
	effectPixels := make([][]byte, 0, schedule.EffectFrames)
	for frame := 0; frame < schedule.EffectFrames; frame++ {
		work := append([]byte(nil), baselineWork...)
		vga := append([]byte(nil), baselineVGA...)
		if err := indexedmap.ComposeNativeCommandHealEffectFrame(
			work, vga, baselineWork, a.FDOTHER6[schedule.EffectStart+frame],
			effectTargets, view.CameraX, view.CameraY,
		); err != nil {
			return false, err
		}
		effectPixels = append(effectPixels, vga)
	}
	roster, err := g.st.NativeMapFrameRoster()
	if err != nil {
		return false, err
	}
	maskWork := append([]byte(nil), baselineWork...)
	maskVGA := append([]byte(nil), baselineVGA...)
	if err := indexedmap.ComposeNativeCommandHealMaskFrame(
		maskWork, maskVGA, baselineWork, a.Units, g.st.NativeMapSelectorCache,
		effectTargets, view.CameraX, view.CameraY, roster.Cycles.Idle, byte(schedule.MaskIndex),
	); err != nil {
		return false, err
	}
	clonedState, err := nativeAIItemClonedState(g.st, tx)
	if err != nil {
		return false, err
	}
	hud, ok := g.nativeMapHUDInput()
	if !ok {
		return false, errors.New("native AI item post-transaction HUD unavailable")
	}
	postInput, err := buildNativeMapFrameInput(a, g.m, clonedState, nativeMapFrameRuntime{HUD: hud})
	if err != nil {
		return false, err
	}
	postWork := append([]byte(nil), baselineWork...)
	postVGA := append([]byte(nil), baselineVGA...)
	if err := indexedmap.ComposeNativeFrame(postWork, postVGA, postInput); err != nil {
		return false, err
	}
	queue := make([]battle.NativePresentationDigit, 0, len(tx.restore.Results)*4)
	for index, result := range tx.restore.Results {
		target := tailTargets[index]
		inCamera := target.X >= view.CameraX && target.X < view.CameraX+12 &&
			target.Y >= view.CameraY-1 && target.Y <= view.CameraY+7
		queue, err = battle.AppendNativePresentationDigits(
			queue, result.Actual, schedule.DigitBias, target.RecordIndex, inCamera,
		)
		if err != nil {
			return false, err
		}
	}
	digitPixels := make([][]byte, 0, schedule.DigitFrames)
	for frame := 0; frame < schedule.DigitFrames; frame++ {
		work := append([]byte(nil), postWork...)
		vga := append([]byte(nil), postVGA...)
		if err := indexedmap.ComposeNativeCommandHealDigitFrame(
			work, vga, postWork, a.CommandHealDigits, queue, tailTargets,
			view.CameraX, view.CameraY, schedule.DigitVertical, frame,
		); err != nil {
			return false, err
		}
		digitPixels = append(digitPixels, vga)
	}
	palette, err := fdother.VGAPaletteFromDAC(g.nativeMapDAC)
	if err != nil {
		return false, err
	}
	effectImages, err := nativeCommand24IndexedImages(effectPixels, palette)
	if err != nil {
		return false, err
	}
	maskImages, err := nativeCommand24IndexedImages([][]byte{baselineVGA, maskVGA}, palette)
	if err != nil {
		return false, err
	}
	digitImages, err := nativeCommand24IndexedImages(digitPixels, palette)
	if err != nil {
		return false, err
	}
	frames := make([]nativeCompoundPresentedFrame, 0, len(effectImages)+schedule.MaskPairs*2+1+len(digitImages))
	if frames, err = appendNativeCompoundFrames(frames, effectImages, schedule.EffectFrameDelayTicks); err != nil {
		return false, err
	}
	frames[0].sound = loadWav(assetPath("assets/sfx/battle_80_12.wav"))
	maskStart := len(frames)
	for index := 0; index < schedule.MaskPairs*2+1; index++ {
		frames = append(frames, nativeCompoundPresentedFrame{image: maskImages[index%2], delay: 1})
	}
	frames[maskStart].sound = loadWav(assetPath("assets/sfx/battle_80_01.wav"))
	publishAt := len(frames)
	if frames, err = appendNativeCompoundFrames(frames, digitImages, 1); err != nil {
		return false, err
	}
	if !osMuteOrShot(g) && (len(frames[0].sound) == 0 || len(frames[maskStart].sound) == 0) {
		return false, errors.New("native AI item required raw sample unavailable")
	}
	g.nativeAIItemPresentation = &nativeAIItemPresentationJob{
		transaction: tx, frames: frames, publishAt: publishAt,
		baselineWork: baselineWork, baselineVGA: baselineVGA,
		postTransactionWork: postWork, postVGA: postVGA, then: then,
	}
	return true, nil
}

// startNativeAIItemDamagePresentation owns type 20/24's caller-specific
// 0x1c4cc -> 0x1cd17 -> result queue path. The transaction remains detached
// until the final blend frame has crossed a Draw boundary.
func (g *Game) startNativeAIItemDamagePresentation(tx *nativeAIItemTransaction, then func()) (bool, error) {
	if g == nil || g.st == nil || tx == nil || tx.damageRoute == nil || len(tx.damage) != len(tx.targets) {
		return false, errors.New("native AI item damage presentation context unavailable")
	}
	if !g.nativeFullPresentationEnabled() {
		return false, errors.New("native AI item indexed presentation unavailable")
	}
	if g.nativeAIItemPresentation != nil || g.nativeHealPresentation != nil ||
		g.nativeModifierPresentation != nil || g.nativeAICommandModifier != nil ||
		g.nativeCmd0Presentation != nil || g.nativeCmd1Presentation != nil ||
		g.nativeCmd2Presentation != nil || g.nativeCmd3Presentation != nil ||
		g.nativeCmd5Presentation != nil || g.nativeCmd6Presentation != nil ||
		g.nativeCmd7Presentation != nil || g.nativeCmd8Presentation != nil ||
		g.nativeCmd9Player != nil || g.nativeCmd9AIPresentation != nil ||
		g.nativeCmd1012 != nil || g.nativeCmd24Presentation != nil ||
		g.nativeCmd29Presentation != nil || g.nativeCmd32Presentation != nil ||
		g.nativeCmd33Presentation != nil || g.nativeCmd34Presentation != nil ||
		g.nativeCmd35Presentation != nil || g.indexedTransition != nil || g.atk != nil {
		return false, errors.New("native AI item presentation already active")
	}
	if !g.st.HasNativeMapViewState || !g.st.HasNativeMapCycleState ||
		len(g.nativeMapWork) != indexedmap.NativeUnitPresentWorkSize ||
		len(g.nativeMapVGA) != indexedmap.NativeMapVGASize || len(g.nativeMapDAC) != 256*3 ||
		!nativeMapAssetsAvailable(g.nativeMapAssets) {
		return false, errors.New("native AI item indexed map state unavailable")
	}
	schedule, err := fdother.BuildNativeAIItemDamageTailSchedule(tx.damageRoute.CommandID)
	if err != nil {
		return false, err
	}
	a := g.nativeMapAssets
	if len(a.FDOTHER6) < schedule.EffectStart+schedule.EffectFrames ||
		len(a.CommandHealDigits) <= schedule.DigitBias+9 {
		return false, errors.New("native AI item damage descriptors unavailable")
	}
	view := g.st.NativeMapViewState
	tailTargets, err := nativeCommandHealTailTargets(g.st, tx.targets)
	if err != nil {
		return false, err
	}
	visible := make([]indexedmap.NativeCommandHealTailTarget, 0, len(tailTargets))
	for _, target := range tailTargets {
		if target.X >= view.CameraX-1 && target.X <= view.CameraX+12 &&
			target.Y >= view.CameraY-1 && target.Y <= view.CameraY+8 {
			visible = append(visible, target)
		}
	}
	baselineWork := append([]byte(nil), g.nativeMapWork...)
	baselineVGA := append([]byte(nil), g.nativeMapVGA...)
	effectPixels := make([][]byte, 0, schedule.EffectFrames)
	for frame := 0; frame < schedule.EffectFrames; frame++ {
		work, vga := append([]byte(nil), baselineWork...), append([]byte(nil), baselineVGA...)
		if err := indexedmap.ComposeNativeCommandHealEffectFrame(work, vga, baselineWork,
			a.FDOTHER6[schedule.EffectStart+frame], visible, view.CameraX, view.CameraY); err != nil {
			return false, err
		}
		effectPixels = append(effectPixels, vga)
	}
	blendPixels := make([][]byte, 0, schedule.BlendFrames)
	for frame := 0; frame < schedule.BlendFrames; frame++ {
		work, vga := append([]byte(nil), baselineWork...), append([]byte(nil), baselineVGA...)
		if err := indexedmap.ComposeNativeAIItemDamageBlendFrame(work, vga, baselineWork,
			a.Units, g.st.NativeMapSelectorCache, visible, view.CameraX, view.CameraY,
			g.st.NativeMapCycleState.Idle, schedule.Blend[frame], byte(schedule.RawBase)); err != nil {
			return false, err
		}
		blendPixels = append(blendPixels, vga)
	}
	clonedState, err := nativeAIItemClonedState(g.st, tx)
	if err != nil {
		return false, err
	}
	hud, ok := g.nativeMapHUDInput()
	if !ok {
		return false, errors.New("native AI item post-transaction HUD unavailable")
	}
	postInput, err := buildNativeMapFrameInput(a, g.m, clonedState, nativeMapFrameRuntime{HUD: hud})
	if err != nil {
		return false, err
	}
	postWork, postVGA := append([]byte(nil), baselineWork...), append([]byte(nil), baselineVGA...)
	if err := indexedmap.ComposeNativeFrame(postWork, postVGA, postInput); err != nil {
		return false, err
	}
	queue := make([]battle.NativePresentationDigit, 0, len(tx.damage)*4)
	for index, result := range tx.damage {
		target := tailTargets[index]
		inCamera := target.X >= view.CameraX && target.X < view.CameraX+12 &&
			target.Y >= view.CameraY-1 && target.Y <= view.CameraY+7
		if result.Hit {
			queue, err = battle.AppendNativePresentationDigits(queue, result.Damage,
				schedule.DigitBias, target.RecordIndex, inCamera)
			if err != nil {
				return false, err
			}
		} else if inCamera {
			for slot, glyph := range [...]int{74, 75, 76, 76} {
				queue = append(queue, battle.NativePresentationDigit{
					PositionCode: [...]int{2, 8, 12, 17}[slot], Target: target.RecordIndex, Digit: glyph,
				})
			}
		}
	}
	digitPixels := make([][]byte, 0, schedule.DigitFrames)
	for frame := 0; frame < schedule.DigitFrames; frame++ {
		work, vga := append([]byte(nil), postWork...), append([]byte(nil), postVGA...)
		if err := indexedmap.ComposeNativeCommandHealDigitFrame(work, vga, postWork,
			a.CommandHealDigits, queue, tailTargets, view.CameraX, view.CameraY,
			schedule.DigitVertical, frame); err != nil {
			return false, err
		}
		digitPixels = append(digitPixels, vga)
	}
	palette, err := fdother.VGAPaletteFromDAC(g.nativeMapDAC)
	if err != nil {
		return false, err
	}
	effectImages, err := nativeCommand24IndexedImages(effectPixels, palette)
	if err != nil {
		return false, err
	}
	blendImages, err := nativeCommand24IndexedImages(blendPixels, palette)
	if err != nil {
		return false, err
	}
	digitImages, err := nativeCommand24IndexedImages(digitPixels, palette)
	if err != nil {
		return false, err
	}
	frames := make([]nativeCompoundPresentedFrame, 0, len(effectImages)+len(blendImages)+len(digitImages))
	if frames, err = appendNativeCompoundFrames(frames, effectImages, schedule.EffectFrameDelayTicks); err != nil {
		return false, err
	}
	frames[0].sound = loadWav(assetPath("assets/sfx/battle_80_06.wav"))
	if frames, err = appendNativeCompoundFrames(frames, blendImages, schedule.BlendDelayTicks); err != nil {
		return false, err
	}
	publishAt := len(frames)
	if frames, err = appendNativeCompoundFrames(frames, digitImages, schedule.DigitFrameDelayTicks); err != nil {
		return false, err
	}
	if !osMuteOrShot(g) && len(frames[0].sound) == 0 {
		return false, errors.New("native AI item required raw sample unavailable")
	}
	g.nativeAIItemPresentation = &nativeAIItemPresentationJob{
		transaction: tx, frames: frames, publishAt: publishAt,
		baselineWork: baselineWork, baselineVGA: baselineVGA,
		postTransactionWork: postWork, postVGA: postVGA, then: then,
	}
	return true, nil
}

func (g *Game) cancelNativeAIItemPresentation() {
	if g == nil {
		return
	}
	j := g.nativeAIItemPresentation
	if j == nil {
		return
	}
	if j.published && j.transaction != nil {
		if g.st == nil || len(j.transaction.before) != len(g.st.Units)*80 {
			if g.loadErr == "" {
				g.loadErr = "native AI item rollback roster changed"
			}
		} else {
			valid := true
			for _, unit := range g.st.Units {
				if unit == nil {
					valid = false
					if g.loadErr == "" {
						g.loadErr = "native AI item rollback unit unavailable"
					}
					break
				}
			}
			if valid {
				for index, unit := range g.st.Units {
					syncNativeItemRuntimeRecord(unit, j.transaction.before[index*80:(index+1)*80])
				}
				g.nativeRNGState = j.transaction.rngBefore
			}
		}
	}
	g.nativeMapWork = append(g.nativeMapWork[:0], j.baselineWork...)
	g.nativeMapVGA = append(g.nativeMapVGA[:0], j.baselineVGA...)
	g.nativeAIItemPresentation = nil
}

func (g *Game) failNativeAIItemPresentation(err error) {
	g.cancelNativeAIItemPresentation()
	g.loadErr = "native AI item presentation: " + err.Error()
}

func (g *Game) stepNativeAIItemPresentation() {
	j := g.nativeAIItemPresentation
	if j == nil {
		return
	}
	if j.holding {
		if j.hold > 0 {
			j.hold--
			return
		}
		then := j.then
		g.nativeAIItemPresentation = nil
		if then != nil {
			then()
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
		if err := g.commitNativeAIItemTransaction(j.transaction); err != nil {
			g.failNativeAIItemPresentation(err)
			return
		}
		g.nativeMapWork = append(g.nativeMapWork[:0], j.postTransactionWork...)
		g.nativeMapVGA = append(g.nativeMapVGA[:0], j.postVGA...)
		j.published = true
	}
	j.frame = next
	if j.frame >= len(j.frames) {
		if !j.published {
			g.failNativeAIItemPresentation(errors.New("required item publication boundary was not presented"))
			return
		}
		j.frame = len(j.frames) - 1
		j.holding, j.hold = true, nativeDelayTicks(500)
	}
}

func (g *Game) drawNativeAIItemPresentation(screen *ebiten.Image) bool {
	j := g.nativeAIItemPresentation
	if j == nil || j.frame < 0 || j.frame >= len(j.frames) || j.frames[j.frame].image == nil {
		return false
	}
	op := &ebiten.DrawImageOptions{}
	op.GeoM.Scale(2, 2)
	screen.DrawImage(j.frames[j.frame].image, op)
	j.drawn = true
	return true
}
