package main

import (
	"bytes"
	"crypto/sha256"
	"encoding/json"
	"image"
	"image/draw"

	"github.com/wicanr2/fd2_re/remake/internal/battle"
)

type modernHDPreparationStatus struct {
	key    [32]byte
	font   *Font
	theme  *modernStoryPortraitSet
	chrome *modernHDItemPanelChrome
	values *modernHDItemPanelValues
}

// Use the same native record as RenderNativeItemPanelData, without manufacturing
// an item list or treating its base icon as a character portrait.
func (g *Game) prepareModernHDPreparationStatus(ui *nativePreparationUIAssets) *modernHDPreparationStatus {
	if g.modernHDText == nil || g.font == nil || g.font.sf == nil || ui == nil || ui.roster == nil || g.nativeClassUI == nil ||
		!g.prepSelecting || g.prepConfirm || g.prepRecordSlots || g.nativeClassUIJob != nil ||
		g.camp == nil || g.camp.Node() == nil || g.camp.Node().Type != "preparation" ||
		g.prepSel < 0 || g.prepSel >= len(g.prepIDs) || len(g.prepPromptSource) != 64000 ||
		ui.roster.UpperRight.Width != 223 || ui.roster.UpperRight.Height != 86 ||
		!bytes.Equal(ui.roster.UpperRight.Pixels, ui.status.ItemPanelUpper.Pixels) {
		return nil
	}
	unit, ok := g.partyRoster[g.prepIDs[g.prepSel]]
	if !ok {
		return nil
	}
	record, err := battle.NativeItemPanelRecordForUnit(&unit)
	if err != nil {
		return nil
	}
	d := &nativeUnitStatusData{record: record, base: g.prepPromptSource, assets: ui.status}
	set := g.interfaceAssetTheme()
	sources, err := modernHDItemPanelIconSources(d, set)
	if err != nil {
		return nil
	}
	palette := g.nativeClassUI.palette
	raw, err := json.Marshal([]any{modernHDItemPanelSourceSeal(d, palette), modernHDItemPanelValuesSeal(d, palette), sources})
	if err != nil {
		return nil
	}
	key := sha256.Sum256(raw)
	if h := g.prepStatusHD; h != nil && h.key == key && h.font == g.font && h.theme == set {
		return h
	}
	chrome, err := prepareModernHDItemPanelChrome(d, palette)
	if err != nil {
		return nil
	}
	values, err := prepareModernHDItemPanelValues(d, palette, g.font)
	if err != nil {
		return nil
	}
	picture := image.NewRGBA(values.picture.Bounds())
	draw.Draw(picture, picture.Bounds(), values.picture, image.Point{}, draw.Src)
	if err := paintModernHDItemPanelUpperIcons(picture, d, set); err != nil {
		return nil
	}
	h := &modernHDPreparationStatus{key: key, font: g.font, theme: set, chrome: chrome,
		values: &modernHDItemPanelValues{seal: key, picture: picture}}
	g.prepStatusHD = h
	return h
}

func (g *Game) appendModernHDPreparationStableLayers(ui *nativePreparationUIAssets) {
	g.appendModernHDPreparationDialogue(ui, nil)
	if h := g.prepareModernHDPreparationStatus(ui); h != nil {
		g.modernHDText.panels = append(g.modernHDText.panels, modernHDDialoguePanel{
			chrome: &h.chrome.frames[1], values: h.values, rect: h.chrome.art[1], status: true})
	}
}
