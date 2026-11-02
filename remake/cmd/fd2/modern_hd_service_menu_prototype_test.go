package main

import (
	"bytes"
	"encoding/json"
	"image"
	"os"
	"path/filepath"
	"testing"

	"github.com/wicanr2/fd2_re/remake/internal/fdother"
)

func serviceMenuTestState(g *Game, owner string, variant int) {
	if owner == "shop" {
		shopDialogueTestState(g, variant, "menu", 0, 0)
	} else {
		facilityDialogueTestOwner(g, "church", "menu", 0, 0)
	}
}
func serviceMenuTestSource(g *Game, owner string) func() ([]byte, bool) {
	if owner == "shop" {
		return g.composeNativeShopStable
	}
	return g.composeNativeChurchSourceFrame
}
func serviceMenuTestCells(t *testing.T, g *Game, owner string) (int, [4]fdother.RawCell) {
	t.Helper()
	var cells [4]fdother.RawCell
	if owner == "shop" {
		a, _, _, ok := g.nativeShopState()
		if !ok {
			t.Fatal("shop state absent")
		}
		for i := range cells {
			cells[i] = a.ServiceCells[i][0]
		}
		return a.ResourceID, cells
	}
	for i := range cells {
		c := g.nativeClassUI.entries[3+2*i]
		cells[i] = fdother.RawCell{Width: c.Width, Height: c.Height, Pixels: c.Pixels}
	}
	return 14, cells
}
func serviceMenuTestBegin(g *Game, owner string, closing bool, after func()) bool {
	if owner == "shop" {
		if closing {
			return g.beginNativeShopServiceClosing(after)
		}
		return g.beginNativeShopServiceOpening()
	}
	if closing {
		return g.beginNativeChurchMenuClosing(after)
	}
	return g.beginNativeChurchMenuOpening()
}
func serviceMenuTestCache(g *Game, owner string) ([][]byte, []byte) {
	if owner == "shop" {
		return g.nativeShopUIJob.frames, g.nativeShopUIJob.restore
	}
	return g.nativeChurchUIJob.frames, g.nativeChurchUIJob.restore
}
func TestModernHDServiceMenuPrototype(t *testing.T) {
	set := facilityHDTestSet(t)
	g := facilityDialogueTestGame(t, set)
	states, jobs, frames, restores, zeroPixels, writerBytes := 0, 0, 0, 0, 0, 0
	for _, locale := range []string{"zh-Hant", "zh-Hans", "en", "ja"} {
		var err error
		g.localeCatalog, err = loadOfficialLocale(locale)
		if err != nil {
			t.Fatal(err)
		}
		g.localeContent, err = loadOfficialLocaleContent(locale)
		if err != nil {
			t.Fatal(err)
		}
		g.localeEntities, err = loadOfficialLocaleEntities(locale)
		if err != nil {
			t.Fatal(err)
		}
		g.localeID = locale
		for _, owner := range []string{"shop", "church"} {
			variants := []int{1, 3, 5}
			if owner == "church" {
				variants = []int{0}
			}
			for _, variant := range variants {
				serviceMenuTestState(g, owner, variant)
				producer := serviceMenuTestSource(g, owner)
				source, ok := producer()
				if !ok {
					t.Fatal("source absent")
				}
				resource, cells := serviceMenuTestCells(t, g, owner)
				var a, b, hd []byte
				var runs []modernHDTextRun
				var oka, okb, okhd bool
				if owner == "shop" {
					a, oka = prototypeShopTransitionMarker(g, 0, producer)
					b, okb = prototypeShopTransitionMarker(g, 255, producer)
					hd, runs, okhd = prototypeShopTransitionCapture(g, producer)
				} else {
					a, oka = g.composeModernFacilityMarker(&modernStableFacilitySource{resource: 14}, 0, producer)
					b, okb = g.composeModernFacilityMarker(&modernStableFacilitySource{resource: 14}, 255, producer)
					hd, runs, okhd = churchListPrototypeCapture(g, producer)
				}
				if !oka || !okb || !okhd || len(runs) == 0 || len(a) != 64000 || len(b) != 64000 || len(hd) != 64000 {
					t.Fatal("source capture absent")
				}
				for i := range a {
					if a[i] == b[i] {
						if a[i] != source[i] {
							t.Fatal("writer differs")
						}
						writerBytes++
					} else if hd[i] != source[i] {
						t.Fatal("HD changed outside writer")
					}
				}
				for option, c := range cells {
					icon := set.uiServiceIcons[[2]int{resource, 3 + 2*option}]
					if c.Width != 24 || c.Height != 20 || len(c.Pixels) != 480 || icon == nil || icon.native != modernUIRawDescription(3+2*option, c) || icon.picture.Bounds() != image.Rect(0, 0, 144, 120) {
						t.Fatal("icon does not bind normal raw cell")
					}
					for _, value := range c.Pixels {
						if value == 0 {
							zeroPixels++
						}
					}
				}
				before, _ := json.Marshal([]any{g.partyRoster, g.shopPending, g.gold, g.nativeRNGState})
				for _, closing := range []bool{false, true} {
					g.nativeShopUIJob, g.nativeChurchUIJob = nil, nil
					called := false
					if !serviceMenuTestBegin(g, owner, closing, func() { called = true }) {
						t.Fatal("begin absent")
					}
					actual, restore := serviceMenuTestCache(g, owner)
					if len(actual) != 4 || called {
						t.Fatal("native lifecycle differs")
					}
					for step, raw := range actual {
						expected := append([]byte(nil), source...)
						divisor := 4 - step
						if closing {
							divisor = step + 1
						}
						// Preserve signed integer division, original option order and index-zero skip.
						for option, offset := range []int{-39, -13, 13, 39} {
							if err := cells[option].BlitAt(expected, 320, 240+offset/divisor, 169); err != nil {
								t.Fatal(err)
							}
						}
						if !bytes.Equal(raw, expected) {
							t.Fatal("menu frame differs", owner, variant, closing, step)
						}
						frames++
					}
					if closing {
						if !bytes.Equal(restore, source) {
							t.Fatal("restore differs")
						}
						restores++
					} else if len(restore) != 0 {
						t.Fatal("invented restore")
					}
					jobs++
				}
				after, _ := json.Marshal([]any{g.partyRoster, g.shopPending, g.gold, g.nativeRNGState})
				if !bytes.Equal(before, after) {
					t.Fatal("menu changes party or gold")
				}
				states++
			}
		}
	}
	result := map[string]any{"status": "PASS", "states": states, "jobs": jobs, "native_frames": frames, "restores": restores, "normal_cell_zero_pixels": zeroPixels, "writer_bytes": writerBytes, "scope": "existing native service menu producers, no new original oracle"}
	if out := os.Getenv("FD2_HD_SERVICE_MENU_PROTOTYPE_OUT"); out != "" {
		raw, _ := json.MarshalIndent(result, "", "  ")
		if err := os.WriteFile(filepath.Join(out, "prototype-report.json"), append(raw, '\n'), 0644); err != nil {
			t.Fatal(err)
		}
	}
	t.Log(result)
}
