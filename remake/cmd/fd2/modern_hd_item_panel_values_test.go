package main

import (
	"bytes"
	"encoding/binary"
	"image"
	"image/color"
	"testing"
)

func itemPanelValuesTestSource(t *testing.T) (*nativeUnitStatusData, color.Palette, *Font) {
	t.Helper()
	d, palette, _ := itemPanelChromeTestSource(t)
	d.record = make([]byte, 80)
	binary.LittleEndian.PutUint16(d.record[64:], 50)
	binary.LittleEndian.PutUint16(d.record[66:], 100)
	binary.LittleEndian.PutUint16(d.record[68:], 25)
	binary.LittleEndian.PutUint16(d.record[70:], 50)
	f := loadFont()
	if f == nil {
		t.Fatal("Noto font required")
	}
	return d, palette, f
}

func TestModernHDItemPanelValuesSourcesAndBounds(t *testing.T) {
	d, palette, f := itemPanelValuesTestSource(t)
	before := append([]byte(nil), d.base...)
	v, err := prepareModernHDItemPanelValues(d, palette, f)
	if err != nil {
		t.Fatal(err)
	}
	// Independent source rectangles include the old bevel, but never numeric ink.
	allowed := []image.Rectangle{image.Rect(139, 33, 157, 41), image.Rect(139, 44, 157, 52), image.Rect(138, 55, 157, 63), image.Rect(99, 55, 117, 63), image.Rect(96, 67, 117, 75), image.Rect(99, 79, 117, 87), image.Rect(139, 67, 157, 75), image.Rect(139, 79, 157, 87), image.Rect(177, 32, 197, 40), image.Rect(177, 51, 197, 59), image.Rect(286, 41, 293, 49), image.Rect(286, 59, 293, 67), image.Rect(197, 32, 302, 39), image.Rect(197, 51, 302, 58)}
	counts := make([]int, len(allowed))
	for y := 0; y < 86*6; y++ {
		for x := 0; x < 223*6; x++ {
			point := image.Pt(x/6+92, y/6+7)
			inside := false
			for i, r := range allowed {
				if point.In(r) {
					inside = true
					counts[i]++
				}
			}
			if (v.picture.RGBAAt(x, y).A != 0) != inside {
				t.Fatal("values escape their measured rectangles", x, y)
			}
		}
	}
	for i, count := range counts {
		if count == 0 {
			t.Fatal("empty field", i)
		}
	}
	if !bytes.Equal(before, d.base) || binary.LittleEndian.Uint16(d.record[64:]) != 50 {
		t.Fatal("native source changed")
	}
	background := color.RGBAModel.Convert(palette[d.assets.ItemPanelUpper.Pixels[70*223+100]]).(color.RGBA)
	if got := v.picture.RGBAAt((157-92)*6-1, (41-7)*6-1); got != background {
		t.Fatal("dialogue shade used for status label", got, background)
	}
	for _, c := range []struct{ current, maximum, width int }{{0, 100, 0}, {100, 100, 618}, {50, 100, 312}, {1, 100, 18}, {-1, 100, 12}, {10, 0, 0}, {32767, 1, 618}, {-10, -100, 72}} {
		if got := modernHDItemPanelBarWidth(c.current, c.maximum); got != c.width {
			t.Fatal("signed native endpoint", c, got)
		}
	}
}

func TestModernHDItemPanelValuesFreezeAndAtomicTheme(t *testing.T) {
	d, palette, f := itemPanelValuesTestSource(t)
	v, err := prepareModernHDItemPanelValues(d, palette, f)
	if err != nil {
		t.Fatal(err)
	}
	d.itemValues = v
	d.record[64]++
	if _, err := prepareModernHDItemPanelValues(d, palette, f); err == nil {
		t.Fatal("changed HP accepted")
	}
	d.record[64]--
	if restored, err := prepareModernHDItemPanelValues(d, palette, f); err != nil || restored != v {
		t.Fatal("restored frozen input", err)
	}
	bad := *d
	bad.itemValues = nil
	bad.record = nil
	g := &Game{font: f, nativeUIPalette: palette, nativeItemPanelText: &nativeItemPanelTextData{data: *d}, nativePlayerStatus: &nativePlayerStatusState{data: &bad}}
	if publish, err := g.stageModernHDItemPanels(&modernStoryPortraitSet{hasHD: true}); err == nil || publish != nil || g.nativeItemPanelText.data.itemValues != v {
		t.Fatal("partial values theme published", err)
	}
	if publish, err := g.stageModernHDItemPanels(nil); err != nil || publish == nil {
		t.Fatal("pixel mode blocked", err)
	}
	for _, kind := range []string{"record", "upper", "bar", "palette", "font"} {
		d, palette, f := itemPanelValuesTestSource(t)
		switch kind {
		case "record":
			d.record = d.record[:79]
		case "upper":
			d.assets.ItemPanelUpper.Height++
		case "bar":
			delete(d.assets.RawCells, 24)
		case "palette":
			palette[d.assets.ItemPanelUpper.Pixels[70*223+100]] = color.RGBA{255, 255, 255, 0}
		case "font":
			f = nil
		}
		if _, err := prepareModernHDItemPanelValues(d, palette, f); err == nil {
			t.Fatal("malformed source accepted", kind)
		}
	}
}
