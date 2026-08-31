package main

import (
	"errors"
	"fmt"

	"github.com/wicanr2/fd2_re/remake/internal/battle"
	"github.com/wicanr2/fd2_re/remake/internal/fdother"
	"github.com/wicanr2/fd2_re/remake/internal/figani"
)

const (
	localizedBattlePanelNameWidth  = 120
	localizedBattlePanelNameHeight = 16
	localizedBattlePanelNameScale  = 0.65
)

func (g *Game) renderLocalizedNativeBattlePanel(
	assets battle.NativeItemPanelDataAssets,
	record, dst []byte,
	unitIndex, rawChapter int,
) error {
	if g == nil || len(record) <= 8 {
		return errors.New("localized native battle panel inputs are invalid")
	}
	if g.localeID == "zh-Hant" {
		return battle.RenderNativeBattlePanel(assets, record, dst, unitIndex, rawChapter)
	}
	if g.localeEntities == nil || g.font == nil {
		return errors.New("localized native battle panel catalog or font is unavailable")
	}
	name, err := g.localeEntities.BattleName(int(record[8]))
	if err != nil {
		return err
	}
	if g.font.Width(name, localizedBattlePanelNameScale) > localizedBattlePanelNameWidth {
		return fmt.Errorf("localized battle name %d %q exceeds panel rectangle", record[8], name)
	}
	staged := append([]byte(nil), dst...)
	if err := battle.RenderNativeBattlePanelWithoutName(assets, record, staged, unitIndex, rawChapter); err != nil {
		return err
	}
	x, y, err := battle.NativeBattlePanelOrigin(record, unitIndex, rawChapter)
	if err != nil {
		return err
	}
	if err := drawIndexedLocalizedText(
		staged, g.font, name, x+5, y+4,
		localizedBattlePanelNameWidth, localizedBattlePanelNameHeight,
		localizedBattlePanelNameScale, 0xcd, 0x4c,
	); err != nil {
		return err
	}
	copy(dst, staged)
	return nil
}

func (g *Game) localizedNativeCommand0Base(
	background fdother.Frame,
	panelAssets battle.NativeItemPanelDataAssets,
	actorRecord, targetRecord []byte,
	actorIndex, targetIndex, chapter int,
	platform *fdother.Frame,
) ([]byte, error) {
	base := make([]byte, 320*200)
	if err := g.renderLocalizedNativeBattlePanel(panelAssets, actorRecord, base, actorIndex, chapter); err != nil {
		return nil, err
	}
	if err := g.renderLocalizedNativeBattlePanel(panelAssets, targetRecord, base, targetIndex, chapter); err != nil {
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

func (g *Game) localizedNativeCommand24BackgroundBase(
	background fdother.Frame,
	panelAssets battle.NativeItemPanelDataAssets,
	panelRecord []byte,
	unitIndex, rawChapter int,
	platform *fdother.Frame,
	overlays ...figani.Frame,
) ([]byte, error) {
	base := make([]byte, 320*200)
	background.X, background.Y = 0, 50
	if err := background.Blit(base, 320, -1); err != nil {
		return nil, err
	}
	if err := g.renderLocalizedNativeBattlePanel(panelAssets, panelRecord, base, unitIndex, rawChapter); err != nil {
		return nil, err
	}
	if platform != nil {
		frame := *platform
		frame.X, frame.Y = 164, 157
		if err := frame.Blit(base, 320, -1); err != nil {
			return nil, err
		}
	}
	for _, overlay := range overlays {
		if err := overlay.BlitAt(base, 320); err != nil {
			return nil, err
		}
	}
	return base, nil
}
