package main

import (
	"bytes"
	"encoding/json"
	"image"
	"os"
	"path/filepath"
	"testing"

	"github.com/wicanr2/fd2_re/remake/internal/campaign"
)

func shopConfirmationPrototypeQuestion(g *Game) func() ([]byte, bool) {
	switch g.nativeShopMode {
	case "confirm":
		return g.composeNativeShopPurchaseQuestion
	case "equip_confirm":
		return g.composeNativeShopEquipQuestion
	case "sell_confirm":
		return g.composeNativeShopSellQuestionBase
	}
	return nil
}

func shopConfirmationPrototypeTarget(g *Game) func() ([]byte, bool) {
	if g.nativeShopMode == "confirm" {
		return g.nativeShopPostChoiceCloseFrame
	}
	return func() ([]byte, bool) {
		q := shopConfirmationPrototypeQuestion(g)
		if q == nil {
			return nil, false
		}
		question, ok := q()
		if !ok {
			return nil, false
		}
		frames, err := campaign.NativeClassConfirmationClosingFrames(question, g.nativeClassUI.choices)
		if err != nil || len(frames) != 4 {
			return nil, false
		}
		return frames[3], true
	}
}

func shopConfirmationPrototypeBegin(g *Game, kind string, after func()) bool {
	if kind == "dialogue_only" {
		frame, ok := g.nativeShopPostChoiceCloseFrame()
		return ok && g.beginNativeShopDialogueClosing(frame, after)
	}
	if g.nativeShopMode == "confirm" {
		if kind == "opening" {
			return g.beginNativeShopConfirmationOpening()
		}
		return g.beginNativeShopConfirmationChoiceClosing(after)
	}
	if g.nativeShopMode == "equip_confirm" {
		if kind == "opening" {
			return g.beginNativeShopEquipConfirmationOpening()
		}
		return g.beginNativeShopEquipConfirmationClosing(after)
	}
	if kind == "opening" {
		return g.beginNativeShopSellConfirmationOpening()
	}
	return g.beginNativeShopSellConfirmationClosing(after)
}

func TestModernHDShopConfirmationPrototype(t *testing.T) {
	g := facilityDialogueTestGame(t, nil)
	facilityRosterTestEightParty(g)
	stages, jobs, frames, restores, writerBytes, croppedPixels := 0, 0, 0, 0, 0, 0
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
		for _, variant := range []int{1, 3, 5} {
			for _, mode := range []string{"confirm", "equip_confirm", "sell_confirm"} {
				shopDialogueTestState(g, variant, mode, 0, 0)
				questionProducer := shopConfirmationPrototypeQuestion(g)
				question, ok := questionProducer()
				if !ok {
					t.Fatal("original shop question absent", variant, mode)
				}
				stable, ok := g.composeNativeShopStable()
				if !ok {
					t.Fatal("original shop source absent")
				}
				targetProducer := shopConfirmationPrototypeTarget(g)
				target, ok := targetProducer()
				if !ok {
					t.Fatal("original post-choice target absent")
				}
				if bytes.Equal(question, target) != (mode == "confirm") {
					t.Fatal("purchase question restore and equip/sell target confused")
				}
				party, _ := json.Marshal(g.partyRoster)
				gold, rng, shared, shop := g.gold, g.nativeRNGState, g.nativeClassUI, g.nativeShopUI
				for _, stage := range []struct {
					name    string
					raw     []byte
					compose func() ([]byte, bool)
				}{{"question", question, questionProducer}, {"source", stable, g.composeNativeShopStable}, {"target", target, targetProducer}} {
					a, okA := prototypeShopTransitionMarker(g, 0, stage.compose)
					b, okB := prototypeShopTransitionMarker(g, 255, stage.compose)
					hd, runs, ok := prototypeShopTransitionCapture(g, stage.compose)
					if !okA || !okB || !ok || len(a) != 64000 || len(b) != 64000 || len(hd) != 64000 || len(runs) == 0 {
						t.Fatal("shop source capture absent", variant, mode, stage.name)
					}
					for i := range a {
						if a[i] == b[i] {
							if a[i] != stage.raw[i] {
								t.Fatal("literal writer disagrees with source")
							}
							writerBytes++
						} else if hd[i] != stage.raw[i] {
							t.Fatal("HD capture changed outside writer")
						}
					}
					stages++
				}
				kinds := []string{"opening", "closing"}
				if mode == "confirm" {
					kinds = []string{"opening", "choice_only", "dialogue_only"}
				}
				for _, kind := range kinds {
					g.nativeShopUIJob = nil
					var expected [][]byte
					if kind == "opening" {
						expected, err = campaign.NativeClassConfirmationOpeningFrames(question, shared.choices)
					} else if kind != "dialogue_only" {
						expected, err = campaign.NativeClassConfirmationClosingFrames(question, shared.choices)
					}
					if err != nil {
						t.Fatal(err)
					}
					if kind == "closing" || kind == "dialogue_only" {
						tail, err := campaign.NativeClassListClosingFrames(stable, target)
						if err != nil {
							t.Fatal(err)
						}
						expected = append(expected, tail...)
					}
					called := false
					if !shopConfirmationPrototypeBegin(g, kind, func() { called = true }) || g.nativeShopUIJob == nil {
						t.Fatal("real shop job absent", variant, mode, kind)
					}
					job := g.nativeShopUIJob
					if len(job.frames) != len(expected) || called || kind != "opening" && job.after == nil {
						t.Fatal("original job or continuation changed")
					}
					for i := range expected {
						if !bytes.Equal(expected[i], job.frames[i]) {
							t.Fatal("original shop cache differs")
						}
						frames++
					}
					closing := kind == "closing" || kind == "dialogue_only"
					if closing {
						if !bytes.Equal(job.restore, stable) {
							t.Fatal("shop source restore differs")
						}
						restores++
					} else if len(job.restore) != 0 {
						t.Fatal("invented shop restore")
					}
					jobs++
				}
				if mode != "confirm" {
					tail, _ := campaign.NativeClassListClosingFrames(stable, target)
					for step := 1; step <= 5; step++ {
						shift := step * 13
						button := image.Rect(248, 168+shift, 272, 188+shift)
						cover := image.Rect(5, 112+shift, 315, min(198+shift, 200))
						visible := button.Intersect(cover)
						wantedHeight := 0
						if step == 1 {
							wantedHeight = 19
						} else if step == 2 {
							wantedHeight = 6
						}
						if visible.Dy() != wantedHeight {
							t.Fatal("native moving choice crop differs")
						}
						for y := visible.Min.Y; y < visible.Max.Y; y++ {
							for x := visible.Min.X; x < visible.Max.X; x++ {
								if tail[step-1][y*320+x] != target[(y-shift)*320+x] {
									t.Fatal("post-choice target was not moved with dialogue")
								}
								croppedPixels++
							}
						}
					}
				}
				after, _ := json.Marshal(g.partyRoster)
				if !bytes.Equal(party, after) || g.gold != gold || g.nativeRNGState != rng || g.nativeClassUI != shared || g.nativeShopUI != shop || g.font.indexedHD != nil || g.modernHDText != nil {
					t.Fatal("shop prototype changed UI or player state")
				}
			}
		}
	}
	t.Logf("stages=%d jobs=%d original_frames=%d restores=%d writer_bytes=%d cropped_choice_pixels=%d", stages, jobs, frames, restores, writerBytes, croppedPixels)
	if out := os.Getenv("FD2_TEST_FACILITY_OUT"); out != "" {
		if err := os.MkdirAll(out, 0755); err != nil {
			t.Fatal(err)
		}
		raw, _ := json.MarshalIndent(map[string]any{"status": "PASS-SHOP-CONFIRMATION-PROTOTYPE", "stages": stages, "jobs": jobs, "original_frames": frames, "restores": restores, "writer_bytes": writerBytes, "cropped_choice_pixels": croppedPixels, "moving_choice_visible_native_heights": []int{19, 6, 0, 0, 0}}, "", "  ")
		if err := os.WriteFile(filepath.Join(out, "receipt.json"), append(raw, '\n'), 0644); err != nil {
			t.Fatal(err)
		}
	}
}
