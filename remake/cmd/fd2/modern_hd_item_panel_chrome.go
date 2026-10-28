package main

import (
	"crypto/sha256"
	"encoding/json"
	"errors"
	"image"
	"image/color"

	"github.com/wicanr2/fd2_re/remake/internal/battle"
	"github.com/wicanr2/fd2_re/remake/internal/fdother"
)

type modernHDItemPanelChrome struct {
	seal   [32]byte
	frames [3]modernHDDialogueChrome
	art    [3]image.Rectangle
}

func modernHDItemPanelSourceSeal(d *nativeUnitStatusData, palette color.Palette) [32]byte {
	if d == nil {
		return [32]byte{}
	}
	var cells []fdother.RawCell
	for i := 1; i <= 17; i++ {
		cells = append(cells, d.assets.RawCells[i])
	}
	b, _ := json.Marshal([]any{d.base, cells, d.assets.ItemPanelUpper, d.assets.ItemPanelBottom, palette})
	return sha256.Sum256(b)
}

func prepareModernHDItemPanelChrome(d *nativeUnitStatusData, palette color.Palette) (*modernHDItemPanelChrome, error) {
	if d == nil || len(d.base) != 64000 {
		return nil, errors.New("modern HD item panel: source unavailable")
	}
	for _, source := range []struct {
		entry fdother.LMI1Entry
		size  image.Point
	}{{d.assets.ItemPanelUpper, image.Pt(223, 86)}, {d.assets.ItemPanelBottom, image.Pt(310, 99)}} {
		if source.entry.Width != source.size.X || source.entry.Height != source.size.Y || len(source.entry.Pixels) != source.size.X*source.size.Y {
			return nil, errors.New("modern HD item panel: artwork source shape differs")
		}
	}
	seal := modernHDItemPanelSourceSeal(d, palette)
	if d.itemChrome != nil {
		if d.itemChrome.seal != seal {
			return nil, errors.New("modern HD item panel: changed frozen source")
		}
		return d.itemChrome, nil
	}
	cells := make([]fdother.RawCell, 18)
	for i := 1; i <= 17; i++ {
		cells[i] = d.assets.RawCells[i]
	}
	frame, err := freezeModernHDDialogueChrome(cells, palette)
	if err != nil {
		return nil, err
	}
	result := &modernHDItemPanelChrome{seal: seal}
	result.art = [3]image.Rectangle{image.Rect(5, 7, 91, 93), image.Rect(92, 7, 315, 93), image.Rect(5, 94, 315, 193)}
	for i := range result.frames {
		result.frames[i] = *frame
	}
	return result, nil
}

func (g *Game) ensureModernHDItemPanelChrome(d *nativeUnitStatusData, palette color.Palette) bool {
	if g.modernHDText == nil {
		return true
	}
	frame, err := prepareModernHDItemPanelChrome(d, palette)
	if err != nil {
		g.loadErr = err.Error()
		return false
	}
	values, err := prepareModernHDItemPanelValues(d, palette, g.font)
	if err != nil {
		g.loadErr = err.Error()
		return false
	}
	rows, err := prepareModernHDItemRowLabels(d, palette, g.font)
	if err != nil {
		g.loadErr = err.Error()
		return false
	}
	icons, err := prepareModernHDItemPanelIcons(d, palette, g.interfaceAssetTheme(), values, rows)
	if err != nil {
		g.loadErr = err.Error()
		return false
	}
	d.itemChrome = frame
	d.itemValues = values
	d.rowLabels = rows
	d.itemIcons = icons
	return true
}

func (g *Game) preflightModernHDItemPanel(d *nativeUnitStatusData, palette color.Palette) bool {
	if !g.modernHDInterfaceEnabled() {
		return true
	}
	art, err := prepareModernHDItemPanelChrome(d, palette)
	if err != nil {
		g.loadErr = err.Error()
		return false
	}
	var values *modernHDItemPanelValues
	var rows *modernHDItemRowLabels
	var icons *modernHDItemPanelIcons
	if g.font != nil {
		values, err = prepareModernHDItemPanelValues(d, palette, g.font)
		if err != nil {
			g.loadErr = err.Error()
			return false
		}
		rows, err = prepareModernHDItemRowLabels(d, palette, g.font)
		if err != nil {
			g.loadErr = err.Error()
			return false
		}
		icons, err = prepareModernHDItemPanelIcons(d, palette, g.interfaceAssetTheme(), values, rows)
		if err != nil {
			g.loadErr = err.Error()
			return false
		}
	}
	d.itemChrome = art
	d.itemValues = values
	d.rowLabels = rows
	d.itemIcons = icons
	return true
}

// The complete source rectangle moves first; only then does the native
// memmove region clip it. A clipped edge is never a newly drawn border.
func modernHDItemPanelPlacements(c *modernHDItemPanelChrome, regions []battle.NativeItemPanelRegion) []modernHDDialoguePanel {
	if c == nil {
		return nil
	}
	full, _ := battle.NativeItemPanelFrameFor(0)
	originals := []battle.NativeItemPanelRegion{full.Left, full.Upper, full.Bottom}
	var result []modernHDDialoguePanel
	for _, region := range regions {
		if !region.Enabled || region.Width <= 0 || region.Height <= 0 {
			continue
		}
		source := image.Rect(region.SourceX, region.SourceY, region.SourceX+region.Width, region.SourceY+region.Height)
		for i, original := range originals {
			base := image.Rect(original.SourceX, original.SourceY, original.SourceX+original.Width, original.SourceY+original.Height)
			if !source.In(base) {
				continue
			}
			offset := image.Pt(region.DestX-region.SourceX, region.DestY-region.SourceY)
			result = append(result, modernHDDialoguePanel{chrome: &c.frames[i], rect: c.art[i].Add(offset), clip: source.Add(offset), status: true})
			break
		}
	}
	return result
}

func (g *Game) appendModernHDItemPanelChrome(d *nativeUnitStatusData, regions []battle.NativeItemPanelRegion, commands bool) {
	if g.modernHDText != nil && d != nil {
		panels := modernHDItemPanelPlacements(d.itemChrome, regions)
		for i := range panels {
			if panels[i].chrome == &d.itemChrome.frames[1] {
				panels[i].values = d.itemValues
				if d.itemIcons != nil {
					panels[i].values = d.itemIcons.upper
				}
			}
			if panels[i].chrome == &d.itemChrome.frames[2] && d.rowLabels != nil {
				panels[i].values = d.rowLabels.items
				if commands {
					panels[i].values = d.rowLabels.commands
				} else if d.itemIcons != nil {
					panels[i].values = d.itemIcons.items
				}
			}
		}
		g.modernHDText.panels = append(g.modernHDText.panels, panels...)
	}
}

func (g *Game) stageModernHDItemPanels(set *modernStoryPortraitSet) (func(), error) {
	if set == nil || !set.hasHD {
		return func() {}, nil
	}
	type pending struct {
		data   *nativeUnitStatusData
		art    *modernHDItemPanelChrome
		values *modernHDItemPanelValues
		rows   *modernHDItemRowLabels
		icons  *modernHDItemPanelIcons
	}
	var staged []pending
	add := func(data *nativeUnitStatusData, palette color.Palette) error {
		if data == nil {
			return nil
		}
		art, err := prepareModernHDItemPanelChrome(data, palette)
		var values *modernHDItemPanelValues
		var rows *modernHDItemRowLabels
		var icons *modernHDItemPanelIcons
		if err == nil && g.font != nil {
			values, err = prepareModernHDItemPanelValues(data, palette, g.font)
			if err == nil {
				rows, err = prepareModernHDItemRowLabels(data, palette, g.font)
			}
			if err == nil {
				icons, err = prepareModernHDItemPanelIcons(data, palette, set, values, rows)
			}
		}
		if err == nil {
			staged = append(staged, pending{data, art, values, rows, icons})
		}
		return err
	}
	if g.nativeItemPanelText != nil {
		if err := add(&g.nativeItemPanelText.data, g.nativeUIPalette); err != nil {
			return nil, err
		}
	}
	if p := g.nativePlayerStatus; p != nil {
		if err := add(p.data, g.nativeUIPalette); err != nil {
			return nil, err
		}
		if p.text != nil {
			if err := add(&p.text.data, g.nativeUIPalette); err != nil {
				return nil, err
			}
		}
	}
	if g.nativeClassUI != nil {
		if err := add(g.churchStatusData, g.nativeClassUI.palette); err != nil {
			return nil, err
		}
		if job := g.nativeClassUIJob; job != nil && job.statusText != nil {
			if err := add(&job.statusText.data, g.nativeClassUI.palette); err != nil {
				return nil, err
			}
		}
	}
	return func() {
		for _, item := range staged {
			item.data.itemChrome = item.art
			item.data.itemValues = item.values
			item.data.rowLabels = item.rows
			item.data.itemIcons = item.icons
		}
	}, nil
}
