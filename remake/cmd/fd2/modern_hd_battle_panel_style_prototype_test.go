package main

// Historical fixed-color proposal beside the current production panel.
// Nothing in the production renderer calls the proposal painter.
import (
	"crypto/sha256"
	"encoding/binary"
	"encoding/json"
	"fmt"
	"image"
	"image/color"
	"image/draw"
	"image/png"
	"os"
	"path/filepath"
	"testing"

	"github.com/wicanr2/fd2_re/remake/internal/battle"
)

func proposedBattlePanelStyle(p *modernBattlePanelText, palette color.Palette) (*image.RGBA, error) {
	canvas := image.NewRGBA(image.Rect(0, 0, 894, 252))
	fill := func(r image.Rectangle, c color.RGBA) {
		draw.Draw(canvas, r, image.NewUniform(c), image.Point{}, draw.Src)
	}
	fill(canvas.Bounds(), color.RGBA{53, 81, 126, 255})
	border := color.RGBA{175, 195, 214, 255}
	for _, r := range []image.Rectangle{image.Rect(0, 0, 894, 3), image.Rect(0, 249, 894, 252), image.Rect(0, 0, 3, 252), image.Rect(891, 0, 894, 252), image.Rect(24, 120, 870, 122)} {
		fill(r, border)
	}
	for _, bar := range []struct {
		y, current, maximum int
		ink                 color.RGBA
	}{
		{22, p.values.HP, p.values.MaxHP, color.RGBA{237, 148, 63, 255}},
		{31, p.values.MP, p.values.MaxMP, color.RGBA{155, 203, 60, 255}},
	} {
		r := image.Rect(21*6, bar.y*6, 124*6, (bar.y+6)*6)
		fill(r, color.RGBA{85, 24, 35, 255})
		width := 0
		if bar.maximum > 0 && bar.current > 0 {
			width = (101*bar.current/bar.maximum + 1) * 6
		}
		if width > r.Dx() {
			width = r.Dx()
		}
		if width > 0 {
			fill(image.Rect(r.Min.X, r.Min.Y, r.Min.X+width, r.Max.Y), bar.ink)
		}
		for _, edge := range []image.Rectangle{image.Rect(r.Min.X, r.Min.Y, r.Max.X, r.Min.Y+1), image.Rect(r.Min.X, r.Max.Y-1, r.Max.X, r.Max.Y), image.Rect(r.Min.X, r.Min.Y, r.Min.X+1, r.Max.Y), image.Rect(r.Max.X-1, r.Min.Y, r.Max.X, r.Max.Y)} {
			fill(edge, border)
		}
	}
	for i, f := range []struct {
		value string
		rect  image.Rectangle
		size  int
	}{
		{p.name, image.Rect(30, 24, 684, 120), 96}, {"LV·", image.Rect(690, 24, 786, 78), 48},
		{fmt.Sprintf("%02d", p.values.Level), image.Rect(792, 24, 882, 78), 48},
		{"HP", image.Rect(18, 126, 120, 180), 48}, {"MP", image.Rect(18, 180, 120, 234), 48},
		{fmt.Sprintf("%03d", p.values.HP), image.Rect(756, 126, 870, 180), 48},
		{fmt.Sprintf("%03d", p.values.MP), image.Rect(756, 180, 870, 234), 48},
	} {
		ink := color.RGBA{242, 240, 226, 255}
		if i == 5 && p.values.HP != p.values.MaxHP || i == 6 && p.values.MP != p.values.MaxMP {
			ink = color.RGBAModel.Convert(palette[p.digitInk[1]]).(color.RGBA)
		}
		if err := drawModernBattlePanelInk(canvas, p.font, f.value, f.rect, f.size, ink); err != nil {
			return nil, err
		}
	}
	return canvas, nil
}

func TestBattlePanelStyleDisposablePrototype(t *testing.T) {
	out := os.Getenv("FD2_PANEL_STYLE_PROTOTYPE_OUT")
	if out == "" {
		t.Skip("explicit disposable prototype output required")
	}
	if _, err := os.Stat(out); !os.IsNotExist(err) {
		t.Fatal("do not overwrite prototype evidence")
	}
	if err := os.Mkdir(out, 0755); err != nil {
		t.Fatal(err)
	}
	g, actor, _ := physicalSceneTestGame(t)
	attachOfficialLocale(t, g)
	g.font = loadFont()
	assets, err := battle.LoadNativeItemPanelDataAssets(separatedAssetPath(""))
	if err != nil {
		t.Fatal(err)
	}
	record, err := battle.NativeBattlePanelRecordForUnit(actor)
	if err != nil {
		t.Fatal(err)
	}
	record[6], record[8], record[0x21] = 0, 0, 12
	binary.LittleEndian.PutUint16(record[0x42:], 80)
	binary.LittleEndian.PutUint16(record[0x46:], 30)
	canvas := image.NewRGBA(image.Rect(0, 0, 1848, 1050))
	draw.Draw(canvas, canvas.Bounds(), image.NewUniform(color.RGBA{17, 24, 37, 255}), image.Point{}, draw.Src)
	for _, label := range []struct {
		value string
		rect  image.Rectangle
	}{
		{"目前遊戲的HD面板", image.Rect(24, 18, 918, 80)},
		{"早期固定配色樣式原型", image.Rect(942, 18, 1836, 80)},
	} {
		if err := drawModernBattlePanelInk(canvas, g.font, label.value, label.rect, 42, color.RGBA{242, 240, 226, 255}); err != nil {
			t.Fatal(err)
		}
	}
	var samples []map[string]any
	for i, values := range [][2]int{{80, 30}, {73, 23}, {0, 0}} {
		binary.LittleEndian.PutUint16(record[0x40:], uint16(values[0]))
		binary.LittleEndian.PutUint16(record[0x44:], uint16(values[1]))
		scene := &nativePhysicalScene{panelAssets: assets}
		base := make([]byte, 64000)
		if err := g.renderPhysicalBattlePanel(scene, record, base, 0); err != nil {
			t.Fatal(err)
		}
		p := scene.hdPanelText[0]
		old, _, err := p.raster(g.nativeUIPalette)
		if err != nil {
			t.Fatal(err)
		}
		proposal, err := proposedBattlePanelStyle(p, g.nativeUIPalette)
		if err != nil {
			t.Fatal(err)
		}
		y := 104 + i*300
		draw.Draw(canvas, image.Rect(24, y, 918, y+252), old, image.Point{}, draw.Src)
		draw.Draw(canvas, image.Rect(942, y, 1836, y+252), proposal, image.Point{}, draw.Src)
		samples = append(samples, map[string]any{"values": p.values, "name": p.name, "native_origin": []int{p.native.X, p.native.Y}, "current_rgba_sha256": fmt.Sprintf("%x", sha256.Sum256(old.(*image.RGBA).Pix)), "proposal_rgba_sha256": fmt.Sprintf("%x", sha256.Sum256(proposal.Pix))})
	}
	if err := drawModernBattlePanelInk(canvas, g.font, "同一份具型別面板資料。左側為現行HD；右側保留早期固定配色原型。", image.Rect(24, 1010, 1836, 1045), 32, color.RGBA{199, 210, 226, 255}); err != nil {
		t.Fatal(err)
	}
	file, err := os.Create(filepath.Join(out, "comparison.png"))
	if err != nil {
		t.Fatal(err)
	}
	if err := png.Encode(file, canvas); err != nil {
		t.Fatal(err)
	}
	if err := file.Close(); err != nil {
		t.Fatal(err)
	}
	raw, err := json.MarshalIndent(map[string]any{"status": "PASS-DISPOSABLE-PANEL-PROTOTYPE", "production_changed": false, "scope": "typed panel raster comparison; no Game.Draw, original oracle, normal player or platform claim", "samples": samples, "style_reference": "tools/fd2_toolbar_mm3_prototype.html HD interface SVG blue, border, track, HP and MP colors", "size": []int{1848, 1050}}, "", "  ")
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(out, "receipt.json"), append(raw, '\n'), 0644); err != nil {
		t.Fatal(err)
	}
}
