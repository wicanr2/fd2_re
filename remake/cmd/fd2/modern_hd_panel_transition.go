package main

import (
	"image"
	"image/color"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/wicanr2/fd2_re/remake/internal/battle"
)

type nativeStatusTransitionKind uint8

const (
	nativeStatusOpening nativeStatusTransitionKind = iota
	nativeStatusClosing
	nativeStatusToCommands
)

type nativeStatusTransitionText struct {
	source   []byte
	data     nativeUnitStatusData
	kind     nativeStatusTransitionKind
	commands bool
}

func snapshotNativeStatusTransition(source []byte, data *nativeUnitStatusData, kind nativeStatusTransitionKind, commands bool) *nativeStatusTransitionText {
	if len(source) != 64000 || data == nil {
		return nil
	}
	d := &nativeStatusTransitionText{source: append([]byte(nil), source...), data: *data, kind: kind, commands: commands}
	d.data.record = append([]byte(nil), data.record...)
	d.data.base = append([]byte(nil), data.base...)
	d.data.itemRows = append([]byte(nil), data.itemRows...)
	d.data.commandIDs = append([]int(nil), data.commandIDs...)
	d.data.commandBook = append([]battle.NativeCommandRecord(nil), data.commandBook...)
	return d
}

func nativeStatusTransitionPass(kind nativeStatusTransitionKind, index int, commands bool) ([]battle.NativeItemPanelRegion, bool, bool) {
	if index < 0 {
		return nil, false, false
	}
	switch kind {
	case nativeStatusOpening, nativeStatusClosing:
		if index >= 12 {
			return nil, false, false
		}
		frame := index
		if kind == nativeStatusOpening {
			frame = 11 - index
		}
		regions, ok := nativeChurchPanelRegions(frame)
		return regions, commands, ok
	case nativeStatusToCommands:
		if index >= 14 {
			return nil, false, false
		}
		step := index
		commands = index >= 7
		if commands {
			step = 13 - index
		}
		regions, ok := nativeChurchBottomPanelRegions(step)
		return regions, commands, ok
	default:
		return nil, false, false
	}
}

// Both renderers use the actual memmove regions. A source clip moves with its
// glyph baseline, then intersects the destination; stationary clipping is not
// sufficient when a panel enters from above, left or below the framebuffer.
func modernHDPanelRegionText(runs []modernHDTextRun, regions []battle.NativeItemPanelRegion) []modernHDTextRun {
	var result []modernHDTextRun
	for _, region := range regions {
		if !region.Enabled || region.Width <= 0 || region.Height <= 0 {
			continue
		}
		source := image.Rect(region.SourceX*2, region.SourceY*2, (region.SourceX+region.Width)*2, (region.SourceY+region.Height)*2)
		dest := image.Rect(region.DestX*2, region.DestY*2, (region.DestX+region.Width)*2, (region.DestY+region.Height)*2)
		offset := dest.Min.Sub(source.Min)
		for _, run := range runs {
			clip := run.clip
			if clip.Empty() {
				clip = image.Rect(0, 0, logicalW, logicalH)
			}
			clip = clip.Intersect(source)
			if clip.Empty() {
				continue
			}
			copy := run
			copy.x += float64(offset.X)
			copy.y += float64(offset.Y)
			copy.clip = clip.Add(offset).Intersect(dest)
			result = append(result, copy)
		}
	}
	return result
}

func (g *Game) composeModernHDUnitStatus(data *nativeUnitStatusData, commands bool, palette color.Palette) ([]byte, []modernHDTextRun, bool) {
	if g.modernHDText == nil || g.font == nil || data == nil {
		return nil, nil, false
	}
	capture := &modernHDIndexedText{palette: palette}
	previous := g.font.indexedHD
	g.font.indexedHD = capture
	defer func() { g.font.indexedHD = previous }()
	panel, ok := data.compose(g.modernHDChurchItemAssets(data.assets), commands)
	return panel, capture.runs, ok
}

func (g *Game) drawModernHDStatusRegions(screen *ebiten.Image, source []byte, data *nativeUnitStatusData, commands bool, palette color.Palette, regions []battle.NativeItemPanelRegion) bool {
	if len(source) != 64000 {
		return false
	}
	for _, region := range regions {
		if region.Enabled && (region.Width < 0 || region.Height < 0 || region.SourceX < 0 || region.SourceY < 0 || region.DestX < 0 || region.DestY < 0 ||
			region.SourceX+region.Width > 320 || region.SourceY+region.Height > 200 || region.DestX+region.Width > 320 || region.DestY+region.Height > 200) {
			return false
		}
	}
	panel, runs, ok := g.composeModernHDUnitStatus(data, commands, palette)
	if !ok {
		return false
	}
	frame := append([]byte(nil), source...)
	for _, region := range regions {
		copyNativeChurchPanelRegion(frame, panel, region)
	}
	g.presentNativeClassFrameWithPalette(screen, frame, palette)
	g.modernHDText.runs = append(g.modernHDText.runs, modernHDPanelRegionText(runs, regions)...)
	return true
}

func (g *Game) drawModernHDStatusTransition(screen *ebiten.Image, d *nativeStatusTransitionText, index int, palette color.Palette) bool {
	if d == nil {
		return false
	}
	regions, commands, ok := nativeStatusTransitionPass(d.kind, index, d.commands)
	return ok && g.drawModernHDStatusRegions(screen, d.source, &d.data, commands, palette, regions)
}
