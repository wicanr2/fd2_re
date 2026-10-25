package main

import (
	"bytes"
	"image/color"
	"strings"
	"testing"

	"github.com/wicanr2/fd2_re/remake/internal/campaign"
	"github.com/wicanr2/fd2_re/remake/internal/fdother"
)

func TestModernHDConfirmationNativePlansAndText(t *testing.T) {
	ui, err := loadNativePreparationUIAssets()
	if err != nil {
		t.Fatal(err)
	}
	class, err := loadNativeClassUIAssets()
	if err != nil {
		t.Fatal(err)
	}
	c, err := freezeModernHDConfirmation(ui.choices, class.palette, loadFont())
	if err != nil {
		t.Fatal(err)
	}
	for locale, labels := range c.labels {
		catalog, err := loadOfficialLocale(locale)
		if err != nil {
			t.Fatal(err)
		}
		for option, key := range []string{"common.yes", "common.no"} {
			want, err := catalog.Format(key)
			if err != nil || labels[option] != want || c.bounds[locale][option].Empty() {
				t.Fatal("official label or glyph bounds differ", locale, option, err)
			}
		}
	}
	background := bytes.Repeat([]byte{255}, 64000)
	check := func(kind fullRewardPhase, index, choice, pulse int, want []byte) {
		t.Helper()
		placements, err := c.placements(kind, index, choice, pulse)
		if err != nil || len(placements) != 2 {
			t.Fatal("missing native choice positions", err)
		}
		actual := append([]byte(nil), background...)
		for _, p := range placements {
			cellIndex := -1
			for index, cell := range c.cells {
				if cell == p.cell {
					cellIndex = index
				}
			}
			if cellIndex < 0 || ui.choices[cellIndex].BlitAt(actual, 320, p.rect.Min.X, p.rect.Min.Y) != nil {
				t.Fatal("unknown native cell")
			}
		}
		if !bytes.Equal(actual, want) {
			t.Fatal("typed positions/order differ from actual native producer", kind, index, choice, pulse)
		}
	}
	opening, err := campaign.NativeClassConfirmationOpeningFrames(background, ui.choices)
	if err != nil {
		t.Fatal(err)
	}
	closing, err := campaign.NativeClassConfirmationClosingFrames(background, ui.choices)
	if err != nil {
		t.Fatal(err)
	}
	for i := 0; i < 4; i++ {
		check(fullRewardOpening, i+6, 0, 0, opening[i])
		check(fullRewardChoiceClosing, i, 0, 0, closing[i])
		choice, pulse := i/2, i%2
		stable, err := campaign.ComposeNativeConfirmationChoices(background, ui.choices, choice, pulse)
		if err != nil {
			t.Fatal(err)
		}
		check(fullRewardChoice, -1, choice, pulse, stable)
	}
	for _, kind := range []fullRewardPhase{fullRewardBlankClosing, fullRewardResponse, fullRewardCancelOpening, fullRewardCancelClosing} {
		p, err := c.placements(kind, 0, 0, 0)
		if err != nil || len(p) != 0 {
			t.Fatal("buttons survived inactive phase", kind, err)
		}
	}
}

func TestModernHDConfirmationRejectsAndFreezesSource(t *testing.T) {
	ui, err := loadNativePreparationUIAssets()
	if err != nil {
		t.Fatal(err)
	}
	class, err := loadNativeClassUIAssets()
	if err != nil {
		t.Fatal(err)
	}
	font := loadFont()
	palette := append(color.Palette(nil), class.palette...)
	c, err := freezeModernHDConfirmation(ui.choices, palette, font)
	if err != nil {
		t.Fatal(err)
	}
	before := append([]byte(nil), c.cells[48].patch().Pix...)
	palette[199] = color.RGBA{255, 0, 0, 255}
	if !bytes.Equal(before, c.cells[48].patch().Pix) {
		t.Fatal("frozen colors followed mutable palette")
	}
	palette[199] = nil
	if _, err := freezeModernHDConfirmation(ui.choices, palette, font); err == nil || !strings.Contains(err.Error(), "palette unavailable") {
		t.Fatal("missing source color accepted", err)
	}
	cells := append([]fdother.RawCell(nil), ui.choices...)
	cells[52].Pixels = nil
	if _, err := freezeModernHDConfirmation(cells, class.palette, font); err == nil || !strings.Contains(err.Error(), "cell52 shape differs") {
		t.Fatal("missing native cell accepted", err)
	}
	if _, err := freezeModernHDConfirmation(ui.choices, class.palette, nil); err == nil || !strings.Contains(err.Error(), "font unavailable") {
		t.Fatal("missing HD font accepted", err)
	}
}
