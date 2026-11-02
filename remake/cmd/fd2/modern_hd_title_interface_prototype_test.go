package main

import (
	"bytes"
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"image"
	"image/color"
	"image/png"
	"os"
	"path/filepath"
	"testing"

	"github.com/hajimehoshi/ebiten/v2/text"
	"github.com/wicanr2/fd2_re/remake/internal/campaign"
	"github.com/wicanr2/fd2_re/remake/internal/fdother"
	xdraw "golang.org/x/image/draw"
)

// Disposable source/typography prototype. No Game input or production Draw hook.
func titlePrototypeRaster(raw []byte, palette color.Palette) *image.RGBA {
	src := image.NewPaletted(image.Rect(0, 0, 320, 200), palette)
	copy(src.Pix, raw)
	dst := image.NewRGBA(image.Rect(0, 0, 1920, 1200))
	xdraw.NearestNeighbor.Scale(dst, dst.Bounds(), src, src.Bounds(), xdraw.Src, nil)
	return dst
}

func titlePrototypeItem(t *testing.T, source fdother.Frame, palette color.Palette, f *Font, item int) (fdother.Frame, []modernHDTextRun, map[string]any) {
	t.Helper()
	if len(source.Indexed) != source.Width*source.Height || len(source.Mask) != len(source.Indexed) {
		t.Fatal("incomplete indexed title source")
	}
	result := source
	result.Indexed = append([]byte(nil), source.Indexed...)
	result.Mask = append([]byte(nil), source.Mask...)
	brightest := -1
	ink := byte(0)
	replaced := 0
	for y := 0; y < source.Height; y++ {
		bg := source.Indexed[y*source.Width+7]
		r, _, b, _ := palette[bg].RGBA()
		if b > r {
			t.Fatal("background sample overlaps title ink")
		}
		for x := 7; x < source.Width-7; x++ {
			at := y*source.Width + x
			index := source.Indexed[at]
			r, g, b, _ := palette[index].RGBA()
			if index == bg {
				continue
			}
			if b <= r {
				t.Fatal("unexpected non-letter source inside title text region", x, y, index)
			}
			if brightness := int(r + g + b); brightness > brightest {
				brightest = brightness
				ink = index
			}
			result.Indexed[at] = bg
			replaced++
		}
	}
	if replaced == 0 {
		t.Fatal("empty title ink")
	}
	left := (320 - source.Width) / 2
	y := titleMenuY(item)
	rect := image.Rect((left+7)*2, y*2, (left+source.Width-7)*2, (y+source.Height)*2)
	face, asc := f.faceFor(54)
	label := []string{"START", "LOAD", "CONTINUE"}[item]
	bounds := text.BoundString(face, label)
	if bounds.Dx() > rect.Dx()*3 || bounds.Dy() > rect.Dy()*3 {
		t.Fatal("HD title does not fit original text region", label, bounds, rect)
	}
	run := modernHDTextRun{font: f, text: label, pixels: 18, clip: rect,
		x:     float64(rect.Min.X*3+(rect.Dx()*3-bounds.Dx())/2-bounds.Min.X) / 3,
		y:     (float64(rect.Min.Y*3+(rect.Dy()*3-bounds.Dy())/2-bounds.Min.Y) - asc) / 3,
		color: modernHDNativeTextColor(palette, ink)}
	shadow := run
	shadow.x += 1.0 / 3
	shadow.y += 1.0 / 3
	shadow.color = modernHDNativeTextColor(palette, 253)
	return result, []modernHDTextRun{shadow, run}, map[string]any{"width": source.Width, "height": source.Height, "indexed_sha256": fmt.Sprintf("%x", sha256.Sum256(source.Indexed)), "mask_sha256": fmt.Sprintf("%x", sha256.Sum256(source.Mask)), "text_region": rect, "replaced_ink": replaced, "foreground_index": ink, "label": label}
}

func TestModernHDTitleInterfacePrototype(t *testing.T) {
	out := os.Getenv("FD2_HD_TITLE_INTERFACE_PROTOTYPE_OUT")
	if out == "" {
		t.Skip("explicit prototype output required")
	}
	if err := os.Mkdir(out, 0755); err != nil {
		t.Fatal(err)
	}
	f := loadFont()
	if f == nil {
		t.Fatal("HD font unavailable")
	}
	class, err := loadNativeClassUIAssets()
	if err != nil {
		t.Fatal(err)
	}
	g := &Game{font: f, nativeClassUI: class, nativePreparationUI: &nativePreparationUIAssets{dialogue: class.dialogue}, localeID: "zh-Hant", fontScale: 1}
	_, palette, err := fdother.LoadSeparatedFDOTHERPalette(separatedAssetPath("palette"), 8)
	if err != nil {
		t.Fatal(err)
	}
	base, err := fdother.LoadSeparatedNestedSingleFrame(separatedAssetPath("surfaces"), 7, 0)
	if err != nil {
		t.Fatal(err)
	}
	rows := []map[string]any{}
	files := []map[string]any{}
	save := func(name string, pic image.Image) {
		path := filepath.Join(out, name)
		file, err := os.Create(path)
		if err != nil {
			t.Fatal(err)
		}
		err = png.Encode(file, pic)
		file.Close()
		if err != nil {
			t.Fatal(err)
		}
		raw, err := os.ReadFile(path)
		if err != nil {
			t.Fatal(err)
		}
		files = append(files, map[string]any{"file": name, "sha256": fmt.Sprintf("%x", sha256.Sum256(raw))})
	}
	for selection := 0; selection < 3; selection++ {
		normal := make([]byte, 64000)
		if err := base.Blit(normal, 320, -1); err != nil {
			t.Fatal(err)
		}
		hd := append([]byte(nil), normal...)
		runs := []modernHDTextRun{}
		for item := 0; item < 3; item++ {
			state := 0
			if item == selection {
				state = 1
			}
			source, err := fdother.LoadSeparatedNestedSingleFrame(separatedAssetPath("surfaces"), 7, 1+item*2+state)
			if err != nil {
				t.Fatal(err)
			}
			blank, newRuns, row := titlePrototypeItem(t, source, palette, f, item)
			row["nested"] = 1 + item*2 + state
			rows = append(rows, row)
			source.X, source.Y = (320-source.Width)/2, titleMenuY(item)
			blank.X, blank.Y = source.X, source.Y
			if err := source.Blit(normal, 320, -1); err != nil {
				t.Fatal(err)
			}
			if err := blank.Blit(hd, 320, -1); err != nil {
				t.Fatal(err)
			}
			runs = append(runs, newRuns...)
		}
		save(fmt.Sprintf("menu-selection%d-original.png", selection), titlePrototypeRaster(normal, palette))
		pic := titlePrototypeRaster(hd, palette)
		save(fmt.Sprintf("menu-selection%d-hd.png", selection), systemGPUCPU(g, pic.Pix, &modernHDTextFrame{runs: runs}, nil))
	}
	assets, err := loadNativeLoadSlotsUIAssets()
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(assets.dialogueBox.Pixels, class.panel.Pixels) {
		t.Fatal("title slots and registered class panel differ")
	}
	if fmt.Sprintf("%x", sha256.Sum256(assets.dialogueBox.Pixels)) != "c009ad5ed72466295e3c60d4cb5117c3331444bcbe23d5d1befd2f413b15ac00" {
		t.Fatal("panel hash differs")
	}
	_, glyphs, err := loadNativeBattleNameAssets()
	if err != nil {
		t.Fatal(err)
	}
	g.nativeBattleGlyphs = glyphs
	chrome, err := freezeModernHDDialogueChrome(class.dialogue, assets.palette)
	if err != nil {
		t.Fatal(err)
	}
	slots := [4]campaign.NativeLoadSlot{{Empty: true}, {Chapter: 2}, {Chapter: 24}, {Empty: true}}
	for selection := 0; selection < 4; selection++ {
		native, err := campaign.ComposeNativeLoadSlotsFrame(make([]byte, 64000), assets.dialogueBox, assets.strings, assets.font, slots, selection)
		if err != nil {
			t.Fatal(err)
		}
		capture := &modernHDIndexedText{palette: assets.palette}
		g.modernHDText = &modernHDTextFrame{}
		hd, err := campaign.ComposeNativeLoadSlotsFrame(make([]byte, 64000), assets.dialogueBox, assets.strings, g.modernHDFacilityFont(assets.font, capture), slots, selection)
		if err != nil {
			t.Fatal(err)
		}
		save(fmt.Sprintf("slots-selection%d-original.png", selection), titlePrototypeRaster(native, assets.palette))
		pic := titlePrototypeRaster(hd, assets.palette)
		frame := &modernHDTextFrame{runs: capture.runs, panels: []modernHDDialoguePanel{{chrome: chrome, rect: image.Rect(5, 112, 315, 198)}}}
		save(fmt.Sprintf("slots-selection%d-hd.png", selection), systemGPUCPU(g, pic.Pix, frame, nil))
	}
	report := map[string]any{"status": "PROTOTYPE-SOURCE-ONLY", "menu_rows": rows, "files": files, "slot_panel_sha256": fmt.Sprintf("%x", sha256.Sum256(assets.dialogueBox.Pixels)), "slot_source": "FDOTHER13 LMI entry16", "palette_source": "title FDOTHER8; slots existing native battle palette", "scope": "disposable source and existing Noto/original-color border prototype; no production hook, original oracle or normal-player reachability"}
	raw, _ := json.MarshalIndent(report, "", "  ")
	if err := os.WriteFile(filepath.Join(out, "prototype.json"), append(raw, '\n'), 0644); err != nil {
		t.Fatal(err)
	}
	t.Log("three original menu selections and four mixed empty/populated slot selections exported")
}
