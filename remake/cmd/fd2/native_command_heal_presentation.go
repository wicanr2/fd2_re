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

type nativeCommandHealPresentationPhase uint8

const (
	nativeCommandHealFrames nativeCommandHealPresentationPhase = iota
	nativeCommandHealMidHold
	nativeCommandHealFrontHold
	nativeCommandHealEffectFrames
	nativeCommandHealMaskFrames
	nativeCommandHealDigitFrames
	nativeCommandHealDigitHold
)

type nativeCommandHealPresentationJob struct {
	frontSchedule fdother.NativeCommandHealPresentationSchedule
	tailSchedule  fdother.NativeCommandHealTailSchedule
	frontFrames   [][]byte
	effectFrames  [][]byte
	maskFrame     []byte
	digitFrames   [][]byte
	baselineVGA   []byte
	baselineWork  []byte
	digits        []fdother.LMI1Entry
	targets       []indexedmap.NativeCommandHealTailTarget
	palette       color.Palette
	frame         int
	phase         nativeCommandHealPresentationPhase
	hold          int
	drawn         bool
	transaction   func() ([]battle.NativeCommandHealResult, error)
	results       []battle.NativeCommandHealResult
	then          func([]battle.NativeCommandHealResult)
}

func nativeCommandHealTailTargets(st *battle.State, targets []*battle.Unit) ([]indexedmap.NativeCommandHealTailTarget, error) {
	if st == nil || len(targets) == 0 {
		return nil, errors.New("native command heal tail target array unavailable")
	}
	out := make([]indexedmap.NativeCommandHealTailTarget, 0, len(targets))
	for _, target := range targets {
		index := -1
		for i, candidate := range st.Units {
			if candidate == target {
				index = i
				break
			}
		}
		if index < 0 || target == nil || !target.HasNativeMapPresentation || !target.HasMapSelectorSlot {
			return nil, errors.New("native command heal tail target lacks raw record presentation")
		}
		out = append(out, indexedmap.NativeCommandHealTailTarget{
			RecordIndex: index,
			X:           int(target.NativeMapPresentation.X), Y: int(target.NativeMapPresentation.Y),
			SelectorSlot: target.MapSelectorSlot,
		})
	}
	return out, nil
}

func (g *Game) startNativeCommandHealPresentation(commandID int, targetUnits []*battle.Unit, transaction func() ([]battle.NativeCommandHealResult, error), then func([]battle.NativeCommandHealResult)) error {
	if !g.nativeFullPresentationEnabled() {
		return errors.New("native abbreviated presentation owner unavailable")
	}
	if g == nil || transaction == nil {
		return errors.New("native command heal presentation game/transaction unavailable")
	}
	if g.nativeHealPresentation != nil || g.nativeCmd0Presentation != nil || g.nativeCmd1Presentation != nil || g.nativeCmd6Presentation != nil || g.nativeCmd24Presentation != nil || g.nativeCmd29Presentation != nil || g.indexedTransition != nil {
		return errors.New("native command heal presentation already active")
	}
	if g.st == nil || !g.st.HasNativeMapViewState || len(g.nativeMapVGA) != indexedmap.NativeMapVGASize || len(g.nativeMapWork) != indexedmap.NativeUnitPresentWorkSize {
		return errors.New("native command heal presentation raw view/frame unavailable")
	}
	front, err := fdother.BuildNativeCommandHealPresentationSchedule(commandID)
	if err != nil {
		return err
	}
	tail, err := fdother.BuildNativeCommandHealTailSchedule(commandID)
	if err != nil {
		return err
	}
	a := g.nativeMapAssets
	if !nativeMapAssetsAvailable(a) || len(a.FDOTHER6) < tail.EffectStart+tail.EffectFrames || len(a.CommandHealDigits) <= tail.DigitBias+9 {
		return errors.New("native command heal tail FDOTHER #5/#6 descriptors unavailable")
	}
	targets, err := nativeCommandHealTailTargets(g.st, targetUnits)
	if err != nil {
		return err
	}
	in, err := g.buildNativeIndexedTransitionInputForState(g.st)
	if err != nil {
		return err
	}
	view := g.st.NativeMapViewState
	centerX, centerY := 24*view.VisibleCursorX+12, 24*view.VisibleCursorY+16
	frontFrames := make([][]byte, 0, len(front.Frames))
	for index, spec := range front.Frames {
		lut, err := fdother.NativeIndexedTransitionLUT(a.LUTs, spec.LUTIndex)
		if err != nil {
			return fmt.Errorf("native command heal presentation frame %d LUT: %w", index, err)
		}
		pass, err := fdother.BuildNativeIndexedTransitionPass(centerX, centerY, spec.Radius, 0, fdother.NativeTransitionStageHeight)
		if err != nil {
			return fmt.Errorf("native command heal presentation frame %d geometry: %w", index, err)
		}
		work := make([]byte, indexedmap.NativeUnitPresentWorkSize)
		vga := make([]byte, indexedmap.NativeMapVGASize)
		if err := indexedmap.ComposeNativeTransitionFrame(work, vga, in, pass, lut); err != nil {
			return fmt.Errorf("native command heal presentation frame %d: %w", index, err)
		}
		frontFrames = append(frontFrames, vga)
	}
	baselineWork, baselineVGA := append([]byte(nil), g.nativeMapWork...), append([]byte(nil), g.nativeMapVGA...)
	effectTargets := make([]indexedmap.NativeCommandHealTailTarget, 0, len(targets))
	for _, target := range targets {
		if target.X >= in.CameraX-1 && target.X <= in.CameraX+12 && target.Y >= in.CameraY-1 && target.Y <= in.CameraY+8 {
			effectTargets = append(effectTargets, target)
		}
	}
	effectFrames := make([][]byte, 0, tail.EffectFrames)
	for frame := 0; frame < tail.EffectFrames; frame++ {
		work, vga := append([]byte(nil), baselineWork...), append([]byte(nil), baselineVGA...)
		if err := indexedmap.ComposeNativeCommandHealEffectFrame(work, vga, baselineWork, a.FDOTHER6[tail.EffectStart+frame], effectTargets, in.CameraX, in.CameraY); err != nil {
			return fmt.Errorf("native command heal tail effect %d: %w", frame, err)
		}
		effectFrames = append(effectFrames, vga)
	}
	roster, err := g.st.NativeMapFrameRoster()
	if err != nil {
		return err
	}
	maskWork, maskVGA := append([]byte(nil), baselineWork...), append([]byte(nil), baselineVGA...)
	if err := indexedmap.ComposeNativeCommandHealMaskFrame(maskWork, maskVGA, baselineWork, a.Units, g.st.NativeMapSelectorCache, effectTargets, in.CameraX, in.CameraY, roster.Cycles.Idle, byte(tail.MaskIndex)); err != nil {
		return err
	}
	hud, ok := g.nativeMapHUDInput()
	if !ok {
		return errors.New("native command heal post-transaction redraw HUD unavailable")
	}
	steady, err := buildNativeMapFrameInput(a, g.m, g.st, nativeMapFrameRuntime{HUD: hud})
	if err != nil {
		return err
	}
	preflightWork, preflightVGA := append([]byte(nil), baselineWork...), append([]byte(nil), baselineVGA...)
	if err := indexedmap.ComposeNativeFrame(preflightWork, preflightVGA, steady); err != nil {
		return fmt.Errorf("native command heal post-transaction redraw preflight: %w", err)
	}
	for _, target := range targets {
		if target.X < view.CameraX || target.X >= view.CameraX+12 || target.Y < view.CameraY-1 || target.Y > view.CameraY+7 {
			continue
		}
		for digit := 0; digit < 10; digit++ {
			queue := make([]battle.NativePresentationDigit, 4)
			for slot := range queue {
				queue[slot] = battle.NativePresentationDigit{PositionCode: 5*slot + 2, Target: target.RecordIndex, Digit: tail.DigitBias + digit}
			}
			for frame := 0; frame < tail.DigitFrames; frame++ {
				work, vga := append([]byte(nil), baselineWork...), append([]byte(nil), baselineVGA...)
				if err := indexedmap.ComposeNativeCommandHealDigitFrame(work, vga, baselineWork, a.CommandHealDigits, queue, targets, view.CameraX, view.CameraY, tail.DigitVertical, frame); err != nil {
					return fmt.Errorf("native command heal digit preflight target %d digit %d frame %d: %w", target.RecordIndex, digit, frame, err)
				}
			}
		}
	}
	palette := a.Palette
	if len(g.nativeMapDAC) == 256*3 {
		palette, err = fdother.VGAPaletteFromDAC(g.nativeMapDAC)
		if err != nil {
			return err
		}
	}
	if len(palette) != 256 {
		return errors.New("native command heal presentation palette unavailable")
	}
	g.nativeHealPresentation = &nativeCommandHealPresentationJob{
		frontSchedule: front, tailSchedule: tail,
		frontFrames: frontFrames, effectFrames: effectFrames, maskFrame: maskVGA,
		baselineVGA: baselineVGA, baselineWork: baselineWork,
		digits: a.CommandHealDigits, targets: targets,
		palette: append(color.Palette(nil), palette...), phase: nativeCommandHealFrames,
		transaction: transaction, then: then,
	}
	g.playSFX(front.SampleIndex)
	return nil
}

func (g *Game) buildNativeCommandHealDigitFrames(j *nativeCommandHealPresentationJob, results []battle.NativeCommandHealResult) error {
	hud, ok := g.nativeMapHUDInput()
	if !ok {
		return errors.New("native command heal post-transaction HUD unavailable")
	}
	steady, err := buildNativeMapFrameInput(g.nativeMapAssets, g.m, g.st, nativeMapFrameRuntime{HUD: hud})
	if err != nil {
		return err
	}
	redrawnWork := append([]byte(nil), j.baselineWork...)
	redrawnVGA := append([]byte(nil), j.baselineVGA...)
	if err := indexedmap.ComposeNativeFrame(redrawnWork, redrawnVGA, steady); err != nil {
		return fmt.Errorf("native command heal post-transaction redraw: %w", err)
	}
	j.baselineWork, j.baselineVGA = redrawnWork, redrawnVGA
	queue := make([]battle.NativePresentationDigit, 0, len(results)*4)
	view := g.st.NativeMapViewState
	for _, result := range results {
		record := -1
		for i, unit := range g.st.Units {
			if unit == result.Target {
				record = i
				break
			}
		}
		var target indexedmap.NativeCommandHealTailTarget
		found := false
		for _, candidate := range j.targets {
			if candidate.RecordIndex == record {
				target, found = candidate, true
				break
			}
		}
		if !found {
			return errors.New("native command heal digit result target unavailable")
		}
		inCamera := target.X >= view.CameraX && target.X < view.CameraX+12 && target.Y >= view.CameraY-1 && target.Y <= view.CameraY+7
		var err error
		queue, err = battle.AppendNativePresentationDigits(queue, result.Restore.Actual, j.tailSchedule.DigitBias, record, inCamera)
		if err != nil {
			return err
		}
	}
	frames := make([][]byte, 0, j.tailSchedule.DigitFrames)
	for frame := 0; frame < j.tailSchedule.DigitFrames; frame++ {
		work, vga := append([]byte(nil), j.baselineWork...), append([]byte(nil), j.baselineVGA...)
		if err := indexedmap.ComposeNativeCommandHealDigitFrame(work, vga, j.baselineWork, j.digits, queue, j.targets, view.CameraX, view.CameraY, j.tailSchedule.DigitVertical, frame); err != nil {
			return fmt.Errorf("native command heal digit frame %d: %w", frame, err)
		}
		frames = append(frames, vga)
	}
	j.digitFrames = frames
	return nil
}

func (g *Game) stepNativeCommandHealPresentation() {
	j := g.nativeHealPresentation
	if j == nil {
		return
	}
	switch j.phase {
	case nativeCommandHealFrames:
		if !j.drawn {
			return
		}
		if j.frame+1 == j.frontSchedule.MidFrame {
			j.phase, j.hold = nativeCommandHealMidHold, nativeDelayTicks(j.frontSchedule.MidDelayMs)
			return
		}
		if j.frame+1 >= len(j.frontFrames) {
			j.phase, j.hold, j.drawn = nativeCommandHealFrontHold, nativeDelayTicks(j.frontSchedule.TailDelayMs), false
			return
		}
		j.frame++
		j.drawn = false
	case nativeCommandHealMidHold:
		if j.hold > 0 {
			j.hold--
			return
		}
		j.frame, j.phase, j.drawn = j.frontSchedule.MidFrame, nativeCommandHealFrames, false
	case nativeCommandHealFrontHold:
		if j.hold > 0 {
			j.hold--
			return
		}
		j.frame, j.phase, j.drawn = 0, nativeCommandHealEffectFrames, false
		g.playSFX(j.tailSchedule.EffectSample)
	case nativeCommandHealEffectFrames:
		if !j.drawn {
			return
		}
		if j.frame+1 < len(j.effectFrames) {
			j.frame++
			j.drawn = false
			return
		}
		j.frame, j.phase, j.drawn = 0, nativeCommandHealMaskFrames, false
		g.playSFX(j.tailSchedule.MaskSample)
	case nativeCommandHealMaskFrames:
		if !j.drawn {
			return
		}
		if j.frame < j.tailSchedule.MaskPairs*2 {
			j.frame++
			j.drawn = false
			return
		}
		results, err := j.transaction()
		if err != nil {
			g.loadErr = "native command heal post-presentation transaction: " + err.Error()
			g.nativeHealPresentation = nil
			return
		}
		if err := g.buildNativeCommandHealDigitFrames(j, results); err != nil {
			g.loadErr = err.Error()
			g.nativeHealPresentation = nil
			return
		}
		j.results = results
		j.frame, j.phase, j.drawn = 0, nativeCommandHealDigitFrames, false
	case nativeCommandHealDigitFrames:
		if !j.drawn {
			return
		}
		if j.frame+1 < len(j.digitFrames) {
			j.frame++
			j.drawn = false
			return
		}
		j.phase, j.hold, j.drawn = nativeCommandHealDigitHold, nativeDelayTicks(j.tailSchedule.DigitHoldMs), false
	case nativeCommandHealDigitHold:
		if j.hold > 0 {
			j.hold--
			return
		}
		then, results := j.then, j.results
		g.nativeMapVGA = append(g.nativeMapVGA[:0], j.baselineVGA...)
		g.nativeMapWork = append(g.nativeMapWork[:0], j.baselineWork...)
		g.nativeHealPresentation = nil
		if then != nil {
			then(results)
		}
	}
}

func (g *Game) drawNativeCommandHealPresentation(screen *ebiten.Image) bool {
	j := g.nativeHealPresentation
	if j == nil || len(j.palette) != 256 {
		return false
	}
	pixels := j.baselineVGA
	switch j.phase {
	case nativeCommandHealFrames:
		if j.frame < 0 || j.frame >= len(j.frontFrames) {
			return false
		}
		pixels, j.drawn = j.frontFrames[j.frame], true
	case nativeCommandHealEffectFrames:
		if j.frame < 0 || j.frame >= len(j.effectFrames) {
			return false
		}
		pixels, j.drawn = j.effectFrames[j.frame], true
	case nativeCommandHealMaskFrames:
		if j.frame%2 == 1 {
			pixels = j.maskFrame
		}
		j.drawn = true
	case nativeCommandHealDigitFrames:
		if j.frame < 0 || j.frame >= len(j.digitFrames) {
			return false
		}
		pixels, j.drawn = j.digitFrames[j.frame], true
	}
	if len(pixels) != indexedmap.NativeMapVGASize {
		return false
	}
	img := image.NewPaletted(image.Rect(0, 0, 320, 200), j.palette)
	copy(img.Pix, pixels)
	op := &ebiten.DrawImageOptions{}
	op.GeoM.Scale(2, 2)
	screen.DrawImage(ebiten.NewImageFromImage(img), op)
	return true
}
