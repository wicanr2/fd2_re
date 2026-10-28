package main

import (
	"image"
	"image/color"
	"slices"

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
	d.data.commandIDs = slices.Clone(data.commandIDs)
	d.data.commandBook = slices.Clone(data.commandBook)
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
			run = run.freezeLayout()
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
	if !g.ensureModernHDItemPanelChrome(data, palette) {
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
		if !g.modernHDInterfaceEnabled() && g.modernStoryPortraits != nil && data != nil {
			g.addModernHDPanelPortrait(data.record, regions)
		}
		return false
	}
	frame := append([]byte(nil), source...)
	for _, region := range regions {
		copyNativeChurchPanelRegion(frame, panel, region)
	}
	g.presentNativeClassFrameWithPalette(screen, frame, palette)
	g.modernHDText.runs = append(g.modernHDText.runs, modernHDPanelRegionText(runs, regions)...)
	g.appendModernHDItemPanelChrome(data, regions, commands)
	g.addModernHDPanelPortrait(data.record, regions)
	return true
}

func (g *Game) drawModernHDStatusTransition(screen *ebiten.Image, d *nativeStatusTransitionText, index int, palette color.Palette) bool {
	if d == nil {
		return false
	}
	regions, commands, ok := nativeStatusTransitionPass(d.kind, index, d.commands)
	return ok && g.drawModernHDStatusRegions(screen, d.source, &d.data, commands, palette, regions)
}

// The modern battlefield has already been drawn. Only the opaque panel regions
// may cover it; the indexed source contains the faithful map, not that field.
func (g *Game) drawModernBattlePlayerStatus(screen *ebiten.Image, s *nativePlayerStatusState) bool {
	if (g.modernStoryPortraits == nil && !g.modernHDInterfaceEnabled()) || s == nil || len(g.nativeUIPalette) != 256 {
		return false
	}
	regions, ok := nativeChurchPanelRegions(0)
	commands := s.phase == nativePlayerStatusCommands
	data := s.data
	if len(s.frames) > 0 {
		if s.text == nil {
			return false
		}
		regions, commands, ok = nativeStatusTransitionPass(s.text.kind, s.frame, s.text.commands)
		data = &s.text.data
	}
	if !ok {
		return false
	}
	panel := s.status
	if commands {
		panel = s.commands
	}
	var runs []modernHDTextRun
	if hd, text, valid := g.composeModernHDUnitStatus(data, commands, g.nativeUIPalette); valid {
		panel, runs = hd, text
	} else if g.loadErr != "" {
		return false
	}
	if len(panel) != 64000 {
		return false
	}
	paletted := image.NewPaletted(image.Rect(0, 0, 320, 200), g.nativeUIPalette)
	copy(paletted.Pix, panel)
	texture := ebiten.NewImageFromImage(paletted)
	for _, region := range regions {
		if !region.Enabled || region.Width <= 0 || region.Height <= 0 {
			continue
		}
		dest := image.Rect(region.DestX*2, region.DestY*2, (region.DestX+region.Width)*2, (region.DestY+region.Height)*2)
		g.clearModernHDTextRegion(dest)
		part := texture.SubImage(image.Rect(region.SourceX, region.SourceY, region.SourceX+region.Width, region.SourceY+region.Height)).(*ebiten.Image)
		op := &ebiten.DrawImageOptions{}
		op.GeoM.Scale(2, 2)
		op.GeoM.Translate(float64(dest.Min.X), float64(dest.Min.Y))
		screen.DrawImage(part, op)
	}
	if g.modernHDText != nil {
		g.modernHDText.runs = append(g.modernHDText.runs, modernHDPanelRegionText(runs, regions)...)
	}
	if data != nil {
		g.appendModernHDItemPanelChrome(data, regions, commands)
		g.addModernHDPanelPortrait(data.record, regions)
	}
	return true
}
