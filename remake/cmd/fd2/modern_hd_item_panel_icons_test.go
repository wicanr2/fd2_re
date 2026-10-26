package main

import (
	"bytes"
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"image"
	"image/draw"
	"image/png"
	"os"
	"path/filepath"
	"testing"

	"github.com/wicanr2/fd2_re/remake/internal/battle"
)

func addPanelIconFrameMetadata(row map[string]any, d *nativeUnitStatusData, commands bool) error {
	fields := []map[string]any{}
	if d != nil {
		index := 53
		if d.record[6] == 0 {
			index = 54
		}
		fields = append(fields, map[string]any{"source": index, "rect": []int{101, 30, 130, 54}})
		for i := 0; i < 3; i++ {
			if d.record[37+i] != 0 {
				x := 194 + 35*i
				fields = append(fields, map[string]any{"source": 55 + i, "rect": []int{x, 68, x + 24, 88}})
			}
		}
		if !commands {
			cells, err := battle.NativeItemSelectorCells(d.record, 0, -1, d.itemRows)
			if err != nil {
				return err
			}
			for _, c := range cells {
				if modernUIPanelIconSize(c.CategoryIcon) == (image.Point{}) {
					continue
				}
				x, y := c.LabelX-29, c.LabelY-2
				fields = append(fields, map[string]any{"source": c.CategoryIcon, "rect": []int{x, y, x + 24, y + 20}, "raw_slot": c.RawSlot, "item_id": int(c.ItemID)})
			}
		}
	}
	row["panel_icon_fields"] = fields
	return nil
}

func TestModernHDPanelIconsSourcesAndLayout(t *testing.T) {
	base, palette, f := itemPanelValuesTestSource(t)
	set := itemHDSet(t)
	if len(set.uiPanelIcons) != 10 {
		t.Fatal("complete registered icon group required")
	}
	for variant := 0; variant < 2; variant++ {
		d := mixedHDRowLabelData(t, base)
		d.record[6] = byte(variant)
		d.record[37], d.record[38], d.record[39] = 1, 1, 1
		values, err := prepareModernHDItemPanelValues(d, palette, f)
		if err != nil {
			t.Fatal(err)
		}
		rows, err := prepareModernHDItemRowLabels(d, palette, f)
		if err != nil {
			t.Fatal(err)
		}
		art, err := prepareModernHDItemPanelIcons(d, palette, set, values, rows)
		if err != nil {
			t.Fatal(err)
		}
		head := 54 - variant
		upper := []struct{ index, x, y int }{{head, 101, 30}, {55, 194, 68}, {56, 229, 68}, {57, 264, 68}}
		for mode, pair := range [][2]*modernHDItemPanelValues{{values, art.upper}, {rows.items, art.items}} {
			old, pic := pair[0].picture, pair[1].picture
			allowed := map[image.Point]bool{}
			place := func(index, x, y, ox, oy int) {
				t.Helper()
				source := set.uiPanelIcons[index].picture
				at := image.Pt((x-ox)*6, (y-oy)*6)
				for y := 0; y < source.Bounds().Dy(); y++ {
					for x := 0; x < source.Bounds().Dx(); x++ {
						p := at.Add(image.Pt(x, y))
						allowed[p] = true
						if pic.RGBAAt(p.X, p.Y) != source.RGBAAt(x, y) {
							t.Fatal("registered icon did not occupy native rectangle", mode, index, p)
						}
					}
				}
			}
			if mode == 0 {
				for _, field := range upper {
					place(field.index, field.x, field.y, 92, 7)
				}
			} else {
				for slot, index := range []int{62, 63, 61, 61, 59, 60, 61, 61} {
					place(index, 13+150*(slot/4), 101+22*(slot%4), 5, 94)
				}
			}
			for y := 0; y < pic.Bounds().Dy(); y++ {
				for x := 0; x < pic.Bounds().Dx(); x++ {
					if !allowed[image.Pt(x, y)] && old.RGBAAt(x, y) != pic.RGBAAt(x, y) {
						t.Fatal("icons changed existing labels or pixels outside native rectangles", mode, x, y)
					}
				}
			}
		}
		if got, err := prepareModernHDItemPanelIcons(d, palette, &modernStoryPortraitSet{}, values, rows); err != nil || got != nil {
			t.Fatal("legacy theme did not retain native icons", err)
		}
	}
}

func TestModernHDPanelIconsFreezeAndAtomicTheme(t *testing.T) {
	base, palette, f := itemPanelValuesTestSource(t)
	set := itemHDSet(t)
	d := mixedHDRowLabelData(t, base)
	values, _ := prepareModernHDItemPanelValues(d, palette, f)
	rows, _ := prepareModernHDItemRowLabels(d, palette, f)
	art, err := prepareModernHDItemPanelIcons(d, palette, set, values, rows)
	if err != nil {
		t.Fatal(err)
	}
	d.itemIcons = art
	d.record[6] ^= 1
	if _, err := prepareModernHDItemPanelIcons(d, palette, set, values, rows); err == nil {
		t.Fatal("changed frozen record accepted")
	}
	d.record[6] ^= 1
	icon := set.uiPanelIcons[53]
	icon.picture.Pix[0] ^= 1
	if _, err := prepareModernHDItemPanelIcons(d, palette, set, values, rows); err == nil {
		t.Fatal("changed HD image accepted")
	}
	icon.picture.Pix[0] ^= 1
	if got, err := prepareModernHDItemPanelIcons(d, palette, set, values, rows); err != nil || got != art {
		t.Fatal("restored immutable icon source rejected", err)
	}
	d.itemIcons = nil
	bad := *d
	bad.record = nil
	g := &Game{font: f, nativeUIPalette: palette, nativeItemPanelText: &nativeItemPanelTextData{data: *d}, nativePlayerStatus: &nativePlayerStatusState{data: &bad}}
	if publish, err := g.stageModernHDItemPanels(set); err == nil || publish != nil || g.nativeItemPanelText.data.itemIcons != nil {
		t.Fatal("failed staging partly published icon group", err)
	}
}

func TestModernHDPanelIconsAssetLoader(t *testing.T) {
	catalog, pack := os.Getenv("FD2_TEST_UNIT_PRESENT_CATALOG"), os.Getenv("FD2_TEST_UNIT_PRESENT_PACK")
	if catalog == "" || pack == "" {
		t.Skip("explicit full HD catalog and pack required")
	}
	raw, err := os.ReadFile(catalog)
	if err != nil {
		t.Fatal(err)
	}
	var all modernThemeCatalog
	if err := json.Unmarshal(raw, &all); err != nil {
		t.Fatal(err)
	}
	minimal := modernThemeCatalog{SchemaVersion: 1, ThemeID: modernHandpaintedThemeID}
	for _, asset := range all.Assets {
		if asset.Role == "native_ui_icon_hd" || asset.Role == "story_portrait_frame" && len(minimal.Assets) == 0 {
			minimal.Assets = append(minimal.Assets, asset)
		}
	}
	if len(minimal.Assets) != 11 {
		t.Fatal("real portrait and ten original icon entries required")
	}
	for _, kind := range []string{"valid", "missing", "duplicate", "shape", "native", "path", "sha", "transparent"} {
		var c modernThemeCatalog
		encoded, _ := json.Marshal(minimal)
		json.Unmarshal(encoded, &c)
		index := len(c.Assets) - 1
		testPack := pack
		switch kind {
		case "missing":
			c.Assets = c.Assets[:index]
		case "duplicate":
			c.Assets = append(c.Assets, c.Assets[index])
		case "shape":
			c.Assets[index].Width++
		case "native":
			c.Assets[index].NativeRawCell.PixelsSHA256 = "invalid"
		case "path":
			c.Assets[index].File = "../outside.png"
		case "sha":
			c.Assets[index].SHA256 = "invalid"
		case "transparent":
			// Keep native dimensions and rebind SHA, so only alpha is invalid.
			testPack = t.TempDir()
			for _, asset := range c.Assets {
				if err := os.Symlink(filepath.Join(pack, asset.File), filepath.Join(testPack, asset.File)); err != nil {
					t.Fatal(err)
				}
			}
			file := c.Assets[index].File
			raw, err := os.ReadFile(filepath.Join(pack, file))
			if err != nil {
				t.Fatal(err)
			}
			original, _, err := image.Decode(bytes.NewReader(raw))
			if err != nil {
				t.Fatal(err)
			}
			picture := image.NewRGBA(original.Bounds())
			draw.Draw(picture, picture.Bounds(), original, original.Bounds().Min, draw.Src)
			picture.Pix[3] = 0
			var buffer bytes.Buffer
			if err := png.Encode(&buffer, picture); err != nil {
				t.Fatal(err)
			}
			path := filepath.Join(testPack, file)
			if err := os.Remove(path); err != nil {
				t.Fatal(err)
			}
			if err := os.WriteFile(path, buffer.Bytes(), 0600); err != nil {
				t.Fatal(err)
			}
			c.Assets[index].SHA256 = fmt.Sprintf("%x", sha256.Sum256(buffer.Bytes()))
		}
		encoded, _ = json.Marshal(c)
		path := filepath.Join(t.TempDir(), "catalog.json")
		if err := os.WriteFile(path, encoded, 0600); err != nil {
			t.Fatal(err)
		}
		set, err := loadModernStoryPortraitSet(path, testPack)
		if kind == "valid" {
			if err != nil || len(set.uiPanelIcons) != 10 {
				t.Fatal("valid original-bound icon group rejected", err)
			}
		} else if err == nil {
			t.Fatal("invalid icon group accepted", kind)
		}
	}
}
