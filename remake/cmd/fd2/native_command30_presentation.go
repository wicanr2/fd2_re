package main

import (
	"fmt"

	"github.com/wicanr2/fd2_re/remake/internal/battle"
	"github.com/wicanr2/fd2_re/remake/internal/fdother"
	"github.com/wicanr2/fd2_re/remake/internal/figani"
)

func (g *Game) startNativeCommand30PresentationAtCursor(actor *battle.Unit, saved, confirmed battle.Cell, then func([]battle.NativeCommand24Damage)) error {
	if g == nil || g.st == nil {
		return fmt.Errorf("native command30 context unavailable")
	}
	target := g.st.UnitAt(confirmed.X, confirmed.Y)
	if target == nil {
		return fmt.Errorf("native command30 confirmed cursor has no unit")
	}
	return g.startNativeDerivedStrikePresentation(actor, target, 30, saved, confirmed, then)
}

// Complete every static source check before the existing Go damage RNG is
// consumed. The command29 owner then reuses the same typed primitive path.
func (g *Game) preflightNativeCommand30Sources(actor *battle.Unit, targets []*battle.Unit, effect *figani.Animation) error {
	if _, _, err := loadNativeBattlePalette(); err != nil {
		return err
	}
	if effect == nil || len(effect.Frames) != 14 || effect.HeaderByte2 != 9 {
		return fmt.Errorf("native command30 resource56 shape differs")
	}
	if effect.Frames[2].RawByte5 != 2 || effect.Frames[2].RawByte4 != 0 || effect.Frames[4].RawByte4 != 1 || effect.Frames[11].RawByte4 != 1 {
		return fmt.Errorf("native command30 effect events differ")
	}
	for i, frame := range effect.Frames {
		if err := frame.BlitAt(make([]byte, 64000), 320); err != nil {
			return fmt.Errorf("native command30 effect%d: %w", i, err)
		}
	}
	assets, err := battle.LoadNativeItemPanelDataAssets(separatedAssetPath(""))
	if err != nil {
		return err
	}
	for i := 0; i < 3; i++ {
		if _, err := fdother.LoadSeparatedSingleFrame(separatedAssetPath("surfaces"), "BG.DAT", i); err != nil {
			return err
		}
	}
	for _, unit := range append([]*battle.Unit{actor}, targets...) {
		if unit == nil || !unit.HasBattleFig {
			return fmt.Errorf("native command30 unit source unavailable")
		}
		idle, err := figani.LoadSeparatedResource(separatedAssetPath("animations"), unit.BattleFig*3)
		if err != nil || len(idle.Frames) == 0 {
			return fmt.Errorf("native command30 idle unavailable: %v", err)
		}
		for _, f := range idle.Frames {
			if f.Delay <= 0 {
				return fmt.Errorf("native command30 idle delay invalid")
			}
			if err := f.BlitAt(make([]byte, 64000), 320); err != nil {
				return err
			}
		}
		selector, err := nativeCommand24BGSelector(g.m, unit)
		if err != nil {
			return err
		}
		if _, err := fdother.LoadSeparatedSingleFrame(separatedAssetPath("surfaces"), "BG.DAT", selector); err != nil {
			return err
		}
		if unit == actor && actor.NativeRecordByte6 != 0 {
			if _, err := fdother.LoadSeparatedSingleFrame(separatedAssetPath("surfaces"), "TAI.DAT", selector); err != nil {
				return err
			}
		}
		record, err := battle.NativeBattlePanelRecordForUnit(unit)
		if err != nil {
			return err
		}
		index, err := nativeCommand24RuntimeUnitIndex(g.st, unit)
		if err != nil {
			return err
		}
		var text []*modernBattlePanelText
		if err := g.renderBattlePanelText(assets, record, make([]byte, 64000), index, g.handlerChapter, &text); err != nil {
			return err
		}
	}
	return nil
}
