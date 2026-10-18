package main

import (
	"image"
	"image/color"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/wicanr2/fd2_re/remake/internal/battle"
)

const nativeShopEquipPanelMode = "equip_panel"
const nativeItemPortraitDirectory = "portraits"

// 與成功發布的像素面板共用輸入快照，選取值為原始八格索引。
type nativeItemPanelTextData struct {
	data            nativeUnitStatusData
	selectedRawSlot int
}

func prepareNativeItemPanelText(record []byte, assets battle.NativeItemPanelDataAssets, rows []byte) (*nativeItemPanelTextData, bool) {
	if len(record) != 80 {
		return nil, false
	}
	art := make([]byte, 64000)
	if err := battle.RenderNativeItemPanelBaseResources(separatedAssetPath(""), separatedAssetPath(nativeItemPortraitDirectory), int(record[7]), art); err != nil {
		return nil, false
	}
	return &nativeItemPanelTextData{data: nativeUnitStatusData{
		record: append([]byte(nil), record...), base: art, assets: assets, itemRows: rows,
	}, selectedRawSlot: -1}, true
}

func (g *Game) publishNativeItemPanelPixels(pixels []byte, selected int) bool {
	if !g.setNativeItemPanelPixels(pixels) {
		return false
	}
	if g.nativeItemPanelText != nil {
		copy := *g.nativeItemPanelText
		copy.selectedRawSlot = selected
		g.nativeItemPanelText = &copy
	}
	return true
}

func (g *Game) composeModernHDItemPanel() (*ebiten.Image, []modernHDTextRun, bool) {
	d := g.nativeItemPanelText
	if g.modernHDText == nil || g.font == nil || d == nil || len(d.data.base) != 64000 {
		return nil, nil, false
	}
	capture := &modernHDIndexedText{palette: g.nativeUIPalette}
	previous := g.font.indexedHD
	g.font.indexedHD = capture
	defer func() { g.font.indexedHD = previous }()
	assets := g.modernHDChurchItemAssets(d.data.assets)
	pixels := append([]byte(nil), d.data.base...)
	if err := battle.RenderNativeItemPanelData(assets, d.data.record, pixels); err != nil {
		return nil, nil, false
	}
	if err := battle.RenderNativeItemPanelRows(assets, d.data.record, d.selectedRawSlot, d.data.itemRows, pixels); err != nil {
		return nil, nil, false
	}
	palette := append(color.Palette(nil), g.nativeUIPalette...)
	if len(palette) < 256 {
		return nil, nil, false
	}
	palette[0] = color.NRGBA{A: 255}
	frame := image.NewPaletted(image.Rect(0, 0, 320, 200), palette)
	copy(frame.Pix, pixels)
	return ebiten.NewImageFromImage(frame), capture.runs, true
}

// 拆分裁切矩形，保留不透明面板外部的文字，不以整句丟棄代替遮蔽。
func (g *Game) clearModernHDTextRegion(covered image.Rectangle) {
	if g.modernHDText == nil || covered.Empty() {
		return
	}
	var kept []modernHDTextRun
	for _, run := range g.modernHDText.runs {
		clip := run.clip
		if clip.Empty() {
			clip = image.Rect(0, 0, logicalW, logicalH)
		}
		overlap := clip.Intersect(covered)
		if overlap.Empty() {
			kept = append(kept, run)
			continue
		}
		for _, outside := range []image.Rectangle{
			image.Rect(clip.Min.X, clip.Min.Y, clip.Max.X, overlap.Min.Y),
			image.Rect(clip.Min.X, overlap.Max.Y, clip.Max.X, clip.Max.Y),
			image.Rect(clip.Min.X, overlap.Min.Y, overlap.Min.X, overlap.Max.Y),
			image.Rect(overlap.Max.X, overlap.Min.Y, clip.Max.X, overlap.Max.Y),
		} {
			if !outside.Empty() {
				copy := run
				copy.clip = outside
				kept = append(kept, copy)
			}
		}
	}
	g.modernHDText.runs = kept
}
