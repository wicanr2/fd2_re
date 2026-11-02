package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"testing"

	"github.com/wicanr2/fd2_re/remake/internal/campaign"
	"github.com/wicanr2/fd2_re/remake/internal/fdother"
)

var churchListPrototypeModes = []string{"class", "status_roster", "transfer_source", "transfer_dest", "transfer_item", "revive"}

func churchListPrototypeState(t *testing.T, g *Game, mode string, scroll bool) {
	t.Helper()
	if mode == "transfer_item" {
		facilityItemTestState(g, mode, 0, facilityItemTestIDs(t, g), scroll)
		return
	}
	facilityRosterTestState(g, "church", mode, 0, 0)
	if mode == "class" {
		// Deliberate UI-only clones provide a full viewport of eligible rows.
		// They test slots and clipping, not character identity or progression.
		template := g.partyRoster[9]
		g.churchIDs = append([]int(nil), g.partyJoinOrder...)
		for _, id := range g.churchIDs {
			unit := template
			unit.NativeIdentity, unit.HasNativeIdentity = id, true
			g.partyRoster[id] = unit
		}
	}
	if scroll {
		if mode == "class" || mode == "revive" {
			g.churchSel, g.churchVerticalStart = 7, 5
		} else {
			g.churchSel, g.churchRosterStart = 7, 2
		}
	}
}

func churchListPrototypeProducer(g *Game) func() ([]byte, bool) {
	switch g.churchMode {
	case "class":
		return g.composeNativeClassListFrame
	case "status_roster", "transfer_source", "transfer_dest":
		return g.composeNativeChurchRosterFrame
	case "transfer_item":
		return g.composeNativeChurchTransferItemFrame
	case "revive":
		return g.composeNativeChurchReviveListFrame
	}
	return nil
}

func churchListPrototypeCapture(g *Game, compose func() ([]byte, bool)) ([]byte, []modernHDTextRun, bool) {
	previous, indexed := g.modernHDText, g.font.indexedHD
	g.modernHDText = &modernHDTextFrame{}
	defer func() { g.modernHDText, g.font.indexedHD = previous, indexed }()
	capture, restore := g.beginModernHDFacilityText()
	defer restore()
	if capture == nil {
		return nil, nil, false
	}
	g.font.indexedHD = capture
	g.nativeClassUI.digits = append([]fdother.Frame(nil), g.nativeClassUI.digits...)
	for digit, frame := range g.nativeClassUI.digits {
		g.nativeClassUI.digits[digit] = g.modernHDDecimalSprite(frame, digit)
	}
	frame, ok := compose()
	return frame, append([]modernHDTextRun(nil), capture.runs...), ok
}

func churchListPrototypeBegin(g *Game, closing bool) bool {
	opening, ending := g.beginNativeChurchRosterOpening, g.beginNativeChurchRosterClosing
	switch g.churchMode {
	case "class":
		opening, ending = g.beginNativeClassListOpening, g.beginNativeClassListClosing
	case "transfer_item":
		opening, ending = g.beginNativeChurchTransferItemOpening, g.beginNativeChurchTransferItemClosing
	case "revive":
		opening, ending = g.beginNativeChurchReviveListOpening, g.beginNativeChurchReviveListClosing
	}
	if closing {
		return ending(nil)
	}
	return opening()
}

func TestModernHDChurchListPrototype(t *testing.T) {
	g := facilityDialogueTestGame(t, nil)
	facilityRosterTestEightParty(g)
	cases, frames, writerBytes := 0, 0, 0
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
		for _, mode := range churchListPrototypeModes {
			for _, scroll := range []bool{false, true} {
				churchListPrototypeState(t, g, mode, scroll)
				shopUI := g.nativeShopUI
				g.nativeShopUI = nil // A fresh church visit must not require shop assets.
				compose := churchListPrototypeProducer(g)
				source, ok := g.composeNativeChurchSourceFrame()
				if !ok {
					t.Fatal("original church source absent", mode)
				}
				final, ok := compose()
				if !ok {
					t.Fatal("original church final absent", mode, scroll)
				}
				party, _ := json.Marshal(g.partyRoster)
				gold, rng, originalUI := g.gold, g.nativeRNGState, g.nativeClassUI
				for _, stage := range []struct {
					name    string
					raw     []byte
					compose func() ([]byte, bool)
				}{{"source", source, g.composeNativeChurchSourceFrame}, {"final", final, compose}} {
					hd, runs, ok := churchListPrototypeCapture(g, stage.compose)
					if !ok || len(hd) != 64000 || len(runs) == 0 || g.nativeClassUI != originalUI || g.font.indexedHD != nil || g.modernHDText != nil {
						t.Fatal("capture failed or leaked UI state", mode, stage.name)
					}
					if stage.name == "source" {
						numbers := 0
						for _, run := range runs {
							if run.y < 224 && len(run.text) > 0 && run.text[0] >= '0' && run.text[0] <= '9' {
								numbers++
							}
						}
						if numbers == 0 {
							t.Fatal("church source lost its decimal capture", locale, mode)
						}
					}
					a, okA := g.composeModernFacilityMarker(&modernStableFacilitySource{resource: 14}, 0, stage.compose)
					b, okB := g.composeModernFacilityMarker(&modernStableFacilitySource{resource: 14}, 255, stage.compose)
					if !okA || !okB || len(a) != 64000 || len(b) != 64000 {
						t.Fatal("original church literal producer failed")
					}
					for i := range a {
						if a[i] == b[i] {
							if a[i] != stage.raw[i] {
								t.Fatal("literal writer differs from original", mode, stage.name, i)
							}
							writerBytes++
						} else if hd[i] != stage.raw[i] {
							t.Fatal("capture erased outside original writer", mode, stage.name, i)
						}
					}
				}
				for _, closing := range []bool{false, true} {
					g.nativeClassUIJob = nil
					if !churchListPrototypeBegin(g, closing) {
						t.Fatal("actual native job failed", mode, closing)
					}
					var want [][]byte
					if closing {
						want, err = campaign.NativeClassListClosingFrames(source, final)
					} else {
						want, err = campaign.NativeClassListOpeningFrames(source, final)
					}
					job := g.nativeClassUIJob
					if err != nil || len(want) != len(job.frames) || closing && !bytes.Equal(job.restore, source) {
						t.Fatal("actual native job changed its source or geometry")
					}
					for i := range want {
						if !bytes.Equal(want[i], job.frames[i]) {
							t.Fatal("actual native frame differs", mode, closing, i)
						}
						frames++
					}
				}
				g.nativeClassUIJob = nil
				again, ok := compose()
				after, _ := json.Marshal(g.partyRoster)
				if !ok || !bytes.Equal(again, final) || !bytes.Equal(party, after) || g.gold != gold || g.nativeRNGState != rng {
					t.Fatal("prototype mutated original source or game state", mode)
				}
				g.nativeShopUI = shopUI
				cases++
			}
		}
	}
	if cases != 48 || frames != 528 || writerBytes == 0 {
		t.Fatal("incomplete church source matrix", cases, frames, writerBytes)
	}
	if out := os.Getenv("FD2_TEST_FACILITY_OUT"); out != "" {
		if err := os.MkdirAll(out, 0755); err != nil {
			t.Fatal(err)
		}
		data, _ := json.MarshalIndent(map[string]any{"status": "PASS-CHURCH-LIST-PROTOTYPE", "cases": cases, "native_frames": frames, "writer_bytes": writerBytes, "scope": "test-only typed source/capture/literal writer proof; four locale states, six modes, two viewports; UI-only class clones, not identity/progression or original E2"}, "", "  ")
		if err := os.WriteFile(filepath.Join(out, "receipt.json"), append(data, '\n'), 0644); err != nil {
			t.Fatal(err)
		}
	}
	fmt.Printf("church prototype: cases=%d native_frames=%d writer_bytes=%d\n", cases, frames, writerBytes)
}
