package main

import (
	"errors"
	"image"
	"image/color"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/wicanr2/fd2_re/remake/internal/battle"
	"github.com/wicanr2/fd2_re/remake/internal/fdother"
	"github.com/wicanr2/fd2_re/remake/internal/indexedmap"
)

type nativeCommandModifierPresentationJob struct {
	commandID   int
	baseline    []byte
	palettes    []color.Palette
	phase       int
	drawn       bool
	transaction func() (battle.NativeCommandModifierResult, error)
	then        func(battle.NativeCommandModifierResult)
}

// startNativeCommandModifierPresentation owns the recovered player-only
// 0x1D6C8 boundary: FDOTHER #88 sub0, then four command-color/black DAC entry
// zero cycles. The state transaction cannot run until all eight phases were
// acknowledged by Draw.
func (g *Game) startNativeCommandModifierPresentation(
	commandID int,
	actor *battle.Unit,
	targets []*battle.Unit,
	transaction func() (battle.NativeCommandModifierResult, error),
	then func(battle.NativeCommandModifierResult),
) error {
	if g == nil || g.st == nil || transaction == nil || commandID < 17 || commandID > 19 {
		return errors.New("native command modifier presentation context unavailable")
	}
	if g.nativeModifierPresentation != nil || g.nativeHealPresentation != nil || g.indexedTransition != nil {
		return errors.New("native command modifier presentation already active")
	}
	if len(g.nativeMapVGA) != indexedmap.NativeMapVGASize || len(g.nativeMapDAC) != 256*3 ||
		g.nativeCommandPaletteFlash == nil {
		return errors.New("native command modifier framebuffer/DAC table unavailable")
	}
	if g.nativeMapDAC[0] != 0 || g.nativeMapDAC[1] != 0 || g.nativeMapDAC[2] != 0 {
		return errors.New("native command modifier baseline DAC entry zero is not black")
	}
	if err := g.st.ValidateNativeCommandModifierTargets(actor, targets, commandID); err != nil {
		return err
	}
	if !osMuteOrShot(g) && len(g.sfxCommandModifier) == 0 {
		return errors.New("native command modifier FDOTHER #88 sub0 unavailable")
	}
	phases, err := g.nativeCommandPaletteFlash.NativeCommandPaletteFlashPhases(commandID)
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
	g.nativeModifierPresentation = &nativeCommandModifierPresentationJob{
		commandID:   commandID,
		baseline:    append([]byte(nil), g.nativeMapVGA...),
		palettes:    palettes,
		transaction: transaction,
		then:        then,
	}
	g.playRaw(g.sfxCommandModifier)
	return nil
}

func (g *Game) stepNativeCommandModifierPresentation() {
	j := g.nativeModifierPresentation
	if j == nil || !j.drawn {
		return
	}
	j.drawn = false
	j.phase++
	if j.phase < len(j.palettes) {
		return
	}
	result, err := j.transaction()
	then := j.then
	g.nativeModifierPresentation = nil
	if err != nil {
		g.msg = "原始指令 modifier transaction 失敗: " + err.Error()
		return
	}
	if then != nil {
		then(result)
	}
}

func (g *Game) drawNativeCommandModifierPresentation(screen *ebiten.Image) bool {
	j := g.nativeModifierPresentation
	if j == nil || j.phase < 0 || j.phase >= len(j.palettes) ||
		len(j.baseline) != indexedmap.NativeMapVGASize || len(j.palettes[j.phase]) != 256 {
		return false
	}
	img := image.NewPaletted(image.Rect(0, 0, 320, 200), j.palettes[j.phase])
	copy(img.Pix, j.baseline)
	op := &ebiten.DrawImageOptions{}
	op.GeoM.Scale(2, 2)
	screen.DrawImage(ebiten.NewImageFromImage(img), op)
	j.drawn = true
	return true
}
