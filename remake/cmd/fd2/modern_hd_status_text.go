package main

import (
	"image/color"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/wicanr2/fd2_re/remake/internal/battle"
)

const nativeChurchModeStatusCommands = "status_commands"
const nativePlayerStatusCommands = "commands"

// This snapshot belongs to the open panel. Live unit state, display themes,
// and the indexed animation buffers have independent lifetimes.
type nativeUnitStatusData struct {
	record, base, itemRows []byte
	assets                 battle.NativeItemPanelDataAssets
	commandIDs             []int
	commandBook            []battle.NativeCommandRecord
	itemChrome             *modernHDItemPanelChrome
	itemValues             *modernHDItemPanelValues
	rowLabels              *modernHDItemRowLabels
}

func (d *nativeUnitStatusData) compose(assets battle.NativeItemPanelDataAssets, commands bool) ([]byte, bool) {
	if d == nil || len(d.base) != 320*200 {
		return nil, false
	}
	frame := append([]byte(nil), d.base...)
	if err := battle.RenderNativeItemPanelData(assets, d.record, frame); err != nil {
		return nil, false
	}
	if commands {
		if len(d.commandIDs) == 0 {
			return nil, false
		}
		if err := battle.RenderNativeCommandOverlay(assets, d.commandIDs, d.commandBook, -1, frame); err != nil {
			return nil, false
		}
	} else if err := battle.RenderNativeItemPanelRows(assets, d.record, -1, d.itemRows, frame); err != nil {
		return nil, false
	}
	return frame, true
}

func (g *Game) drawModernHDUnitStatus(screen *ebiten.Image, source []byte, data *nativeUnitStatusData, commands bool, palette color.Palette) bool {
	regions, ok := nativeChurchPanelRegions(0)
	return ok && g.drawModernHDStatusRegions(screen, source, data, commands, palette, regions)
}
