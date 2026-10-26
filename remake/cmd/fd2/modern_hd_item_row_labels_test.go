package main

import (
	"image"
	"image/color"
	"testing"

	"github.com/wicanr2/fd2_re/remake/internal/battle"
)

func realHDRowLabelItems(t *testing.T, rows []byte) []int {
	t.Helper()
	ids := map[int]int{}
	for id := 1; id < len(rows)/battle.NativeItemEffectRowSize; id++ {
		r := rows[id*battle.NativeItemEffectRowSize:]
		kind := 0
		switch {
		case r[0] < 0x15:
			kind = 64
		case r[0] < 0x20:
			kind = 65
		case r[0] == 0x20 && r[13] == 5:
			kind = 66
		case r[0] == 0x20 && r[13] == 11:
			kind = 67
		}
		if kind != 0 && ids[kind] == 0 {
			ids[kind] = id
		}
	}
	result := []int{ids[64], ids[65], ids[66], ids[67]}
	for _, id := range result {
		if id == 0 {
			t.Fatal("real item row kind unavailable", result)
		}
	}
	return result
}

func mixedHDRowLabelData(t *testing.T, source *nativeUnitStatusData) *nativeUnitStatusData {
	t.Helper()
	d := *source
	d.record = append([]byte(nil), source.record...)
	d.itemChrome, d.itemValues, d.rowLabels = nil, nil, nil
	d.itemIcons = nil
	var err error
	d.itemRows, err = battle.LoadNativeItemEffectRowPrefix(assetPath("assets/data/native_item_effect_rows.json"))
	if err != nil {
		t.Fatal(err)
	}
	ids := realHDRowLabelItems(t, d.itemRows)
	for i := 0; i < 8; i++ {
		flag := byte(0)
		if i < 2 { // Equip only the first weapon and armor, never consumables.
			flag = 0x40
		}
		d.record[10+2*i], d.record[11+2*i] = flag, byte(ids[i%4])
	}
	d.commandBook, err = battle.LoadNativeCommandRecords(assetPath("assets/spells.json"))
	if err != nil {
		t.Fatal(err)
	}
	d.commandIDs = []int{0, 1, 2, 3, 4, 5, 6, 7, 8, 9, 10, 11}
	return &d
}

func TestModernHDRowLabelSourcesAndLayout(t *testing.T) {
	base, palette, f := itemPanelValuesTestSource(t)
	d := mixedHDRowLabelData(t, base)
	art, err := prepareModernHDItemRowLabels(d, palette, f)
	if err != nil {
		t.Fatal(err)
	}
	for mode, pic := range []*modernHDItemPanelValues{art.items, art.commands} {
		if pic == nil || pic.picture.Bounds() != image.Rect(0, 0, 1860, 594) {
			t.Fatal("missing full bottom artwork", mode)
		}
		var fields []image.Rectangle
		count := 8
		if mode == 1 {
			count = 12
		}
		for i := 0; i < count; i++ {
			x, y, width := 110+150*(i/4), 107+22*(i%4), 21
			if mode == 1 {
				x, y, width = 68+100*(i/4), 108+22*(i%4), 20
			}
			fields = append(fields, image.Rect(x, y, x+width, y+8))
		}
		for y := 0; y < 594; y++ {
			for x := 0; x < 1860; x++ {
				point := image.Pt(x/6+5, y/6+94)
				inside := false
				for _, field := range fields {
					inside = inside || point.In(field)
				}
				if (pic.picture.RGBAAt(x, y).A != 0) != inside {
					t.Fatal("row labels escape source geometry", mode, point)
				}
			}
		}
	}
	// Raw inventory holes compact only the display, not the stored slots.
	d.rowLabels = nil
	for _, slot := range []int{0, 2, 3, 5, 6} {
		d.record[10+2*slot] = 0x80
	}
	d.commandIDs = nil
	art, err = prepareModernHDItemRowLabels(d, palette, f)
	if err != nil || art.commands != nil || art.items == nil {
		t.Fatal("compacted row source", err)
	}
	opaque := 0
	for at := 3; at < len(art.items.picture.Pix); at += 4 {
		if art.items.picture.Pix[at] == 255 {
			opaque++
		}
	}
	if opaque != 3*21*8*36 {
		t.Fatal("raw holes did not compact to exactly three labels", opaque)
	}
	for i := 0; i < 8; i++ {
		d.record[10+2*i] = 0x80
	}
	art, err = prepareModernHDItemRowLabels(d, palette, f)
	if err != nil || art.items != nil || art.commands != nil {
		t.Fatal("empty panel retains prior row labels", err)
	}
}

func TestModernHDRowLabelFreezeAndAtomicTheme(t *testing.T) {
	base, palette, f := itemPanelValuesTestSource(t)
	d := mixedHDRowLabelData(t, base)
	art, err := prepareModernHDItemRowLabels(d, palette, f)
	if err != nil {
		t.Fatal(err)
	}
	d.rowLabels = art
	d.record[11]++
	if _, err := prepareModernHDItemRowLabels(d, palette, f); err == nil {
		t.Fatal("changed row source accepted")
	}
	d.record[11]--
	if got, err := prepareModernHDItemRowLabels(d, palette, f); err != nil || got != art {
		t.Fatal("restored frozen row source", err)
	}
	bad := *d
	bad.record = nil
	g := &Game{font: f, nativeUIPalette: palette, nativeItemPanelText: &nativeItemPanelTextData{data: *d}, nativePlayerStatus: &nativePlayerStatusState{data: &bad}}
	if publish, err := g.stageModernHDItemPanels(&modernStoryPortraitSet{hasHD: true}); err == nil || publish != nil || g.nativeItemPanelText.data.rowLabels != art || g.nativeItemPanelText.data.itemValues != nil {
		t.Fatal("failed theme partly published row art", err)
	}
	for _, kind := range []string{"shape", "record", "book", "overflow", "palette"} {
		d := mixedHDRowLabelData(t, base)
		colors := append(color.Palette(nil), palette...)
		switch kind {
		case "shape":
			cell := d.assets.RawCells[92]
			cell.Width++
			d.assets.RawCells[92] = cell
		case "record":
			d.record = d.record[:79]
		case "book":
			d.commandBook = nil
		case "overflow":
			d.commandIDs = append(d.commandIDs, 12)
		case "palette":
			colors[73] = color.RGBA{A: 0}
		}
		if _, err := prepareModernHDItemRowLabels(d, colors, f); err == nil {
			t.Fatal("invalid row source accepted", kind)
		}
		// Each malformed case gets independent asset maps next time.
		base, palette, f = itemPanelValuesTestSource(t)
	}
}

func TestModernHDStatusSnapshotKeepsEmptyCommandCache(t *testing.T) {
	for _, emptyBook := range []bool{false, true} {
		t.Run(map[bool]string{false: "loaded-book", true: "empty-book"}[emptyBook], func(t *testing.T) {
			base, palette, f := itemPanelValuesTestSource(t)
			d := mixedHDRowLabelData(t, base)
			d.commandIDs = (&battle.Unit{}).NativeCommandIDs()
			if emptyBook {
				d.commandBook = make([]battle.NativeCommandRecord, 0)
			}
			g := &Game{font: f, nativeUIPalette: palette, modernStoryPortraits: &modernStoryPortraitSet{hasHD: true}}
			if !g.preflightModernHDItemPanel(d, palette) {
				t.Fatal("empty command panel preflight", g.loadErr)
			}
			cached := d.rowLabels
			transition := snapshotNativeStatusTransition(make([]byte, 64000), d, nativeStatusOpening, false)
			if !g.ensureModernHDItemPanelChrome(&transition.data, palette) {
				t.Fatal("unchanged empty command snapshot rejected", g.loadErr)
			}
			if transition.data.rowLabels != cached || cached.commands != nil || len(transition.data.commandIDs) != 0 {
				t.Fatal("snapshot changed empty commands or discarded frozen art")
			}
			g.nativePlayerStatus = &nativePlayerStatusState{data: d, text: transition}
			publish, err := g.stageModernHDItemPanels(&modernStoryPortraitSet{hasHD: true})
			if err != nil || publish == nil {
				t.Fatal("theme staging rejected empty command owner", err)
			}
			publish()
			if transition.data.rowLabels != cached {
				t.Fatal("theme staging replaced unchanged row cache")
			}
		})
	}
}
