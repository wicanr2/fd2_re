package main

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"image"
	"os"
	"path/filepath"
	"testing"

	"github.com/wicanr2/fd2_re/remake/internal/fdother"
)

var moneyPrototypeRect = image.Rect(5, 95, 68, 110)
var moneyPrototypeDigits = image.Rect(16, 98, 64, 107)

func moneyPrototypeCell(g *Game, owner string) (fdother.RawCell, bool) {
	if owner == "shop" {
		a, _, _, ok := g.nativeShopState()
		if !ok || a == nil {
			return fdother.RawCell{}, false
		}
		c := a.Decoration
		return fdother.RawCell{Width: c.Width, Height: c.Height, Pixels: c.Pixels}, true
	}
	if g.nativeClassUI == nil || len(g.nativeClassUI.entries) <= 1 {
		return fdother.RawCell{}, false
	}
	c := g.nativeClassUI.entries[1]
	return fdother.RawCell{Width: c.Width, Height: c.Height, Pixels: c.Pixels}, true
}
func moneyPrototypeMatch(frame []byte, c fdother.RawCell) bool {
	if len(frame) != 64000 || c.Width != 63 || c.Height != 15 || len(c.Pixels) != 945 {
		return false
	}
	sum := sha256.Sum256(c.Pixels)
	if hex.EncodeToString(sum[:]) != "3ae75118a0206611cf5e3b8bc90821fffad3ace38c99407e33cef7c238816837" {
		return false
	}
	for y := 0; y < 15; y++ {
		for x := 0; x < 63; x++ {
			at := image.Pt(5+x, 95+y)
			if !at.In(moneyPrototypeDigits) && frame[at.Y*320+at.X] != c.Pixels[y*63+x] {
				return false
			}
		}
	}
	return true
}
func TestModernHDFacilityMoneyPrototype(t *testing.T) {
	g := facilityDialogueTestGame(t, nil)
	facilityRosterTestEightParty(g)
	states, masked, negative, jobs, frames := 0, 0, 0, 0, 0
	for _, owner := range []string{"shop", "church"} {
		variants := []int{1, 3, 5}
		if owner == "church" {
			variants = []int{0}
		}
		for _, variant := range variants {
			modes := facilityDialogueChurchModes
			if owner == "shop" {
				modes = nil
				for _, m := range shopDialogueModes {
					modes = append(modes, m.mode)
				}
			}
			for _, mode := range modes {
				if owner == "shop" {
					shopDialogueTestState(g, variant, mode, 0, 0)
				} else {
					facilityDialogueTestOwner(g, owner, mode, 0, 0)
				}
				c, ok := moneyPrototypeCell(g, owner)
				if !ok {
					t.Fatal("decoration source missing")
				}
				var raw []byte
				if owner == "shop" {
					raw, ok = shopDialogueCompose(g)
				} else {
					raw, ok = facilityDialogueCompose(g)
				}
				if !ok || !moneyPrototypeMatch(raw, c) {
					t.Fatal("money source differs", owner, variant, mode)
				}
				if mode == "menu" {
					for y := 95; y < 110; y++ {
						for x := 5; x < 68; x++ {
							at := image.Pt(x, y)
							index := y*320 + x
							copyFrame := append([]byte(nil), raw...)
							copyFrame[index] ^= 1
							if moneyPrototypeMatch(copyFrame, c) == !at.In(moneyPrototypeDigits) {
								t.Fatal("incorrect protected cell", at)
							}
							if at.In(moneyPrototypeDigits) {
								masked++
							} else {
								negative++
							}
						}
					}
				}
				states++
			}
			for _, closing := range []bool{false, true} {
				serviceMenuTestState(g, owner, variant)
				c, _ := moneyPrototypeCell(g, owner)
				if !serviceMenuTestBegin(g, owner, closing, nil) {
					t.Fatal("menu job absent")
				}
				cache, restore := serviceMenuTestCache(g, owner)
				for _, raw := range cache {
					if !moneyPrototypeMatch(raw, c) {
						t.Fatal("moving menu obscures money")
					}
					frames++
				}
				if closing && !moneyPrototypeMatch(restore, c) {
					t.Fatal("menu restore differs")
				}
				jobs++
			}
		}
	}
	// A raw source is read-only; the proposed holes include exactly the native
	// rolling digit window and never the currency prefix or frame.
	source := append([]byte(nil), g.nativeClassUI.entries[1].Pixels...)
	c, _ := moneyPrototypeCell(g, "church")
	if !bytes.Equal(source, c.Pixels) {
		t.Fatal("source mutated")
	}
	result := map[string]any{"status": "PASS", "stable_states": states, "menu_jobs": jobs, "menu_frames": frames, "protected_byte_corruptions_rejected": negative, "digit_window_bytes_admitted": masked, "source_rect": []int{5, 95, 68, 110}, "digit_window": []int{16, 98, 64, 107}, "scope": "four existing facility decorations, no new original oracle"}
	if out := os.Getenv("FD2_HD_FACILITY_MONEY_PROTOTYPE_OUT"); out != "" {
		raw, _ := json.MarshalIndent(result, "", "  ")
		if err := os.WriteFile(filepath.Join(out, "prototype-report.json"), append(raw, '\n'), 0644); err != nil {
			t.Fatal(err)
		}
	}
	t.Log(result)
}

func TestModernHDFacilityMoneyPaletteEvidence(t *testing.T) {
	shared, err := loadNativeClassUIAssets()
	if err != nil {
		t.Fatal(err)
	}
	colors := map[int][]uint32{}
	for _, index := range []int{73, 75, 169, 173, 177, 189, 254} {
		r, g, b, a := shared.palette[index].RGBA()
		colors[index] = []uint32{r >> 8, g >> 8, b >> 8, a >> 8}
	}
	if colors[169][0] != 211 || colors[169][1] != 227 || colors[169][2] != 251 || colors[75][0] != 28 {
		t.Fatal("reference currency palette differs")
	}
	raw, _ := json.MarshalIndent(colors, "", "  ")
	if out := os.Getenv("FD2_HD_FACILITY_MONEY_PROTOTYPE_OUT"); out != "" {
		if err := os.WriteFile(filepath.Join(out, "palette-evidence.json"), append(raw, '\n'), 0644); err != nil {
			t.Fatal(err)
		}
	}
	t.Log(colors)
}
