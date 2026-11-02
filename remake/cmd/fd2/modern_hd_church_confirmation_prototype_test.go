package main

import (
	"bytes"
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"testing"

	"github.com/wicanr2/fd2_re/remake/internal/campaign"
)

func churchConfirmationPrototypeState(g *Game, mode string) {
	facilityDialogueTestOwner(g, "church", mode, 0, 0)
	g.nativeShopUI = nil
}

func churchConfirmationPrototypeQuestion(g *Game) func() ([]byte, bool) {
	if g.churchMode == "class_confirm" {
		return g.composeNativeClassConfirmationQuestion
	}
	return g.composeNativeChurchReviveQuestion
}

func churchConfirmationPrototypeBegin(g *Game, kind string, after func()) bool {
	if kind == "choice_only" {
		return g.beginNativeChurchReviveChoiceClosing(after)
	}
	if g.churchMode == "class_confirm" {
		if kind == "opening" {
			return g.beginNativeClassConfirmationOpening()
		}
		return g.beginNativeClassConfirmationClosing(after)
	}
	if kind == "opening" {
		return g.beginNativeChurchReviveConfirmationOpening()
	}
	return g.beginNativeChurchReviveConfirmationClosing(after)
}

func TestModernHDChurchConfirmationPrototype(t *testing.T) {
	g := facilityDialogueTestGame(t, nil)
	facilityRosterTestEightParty(g)
	stages, jobs, frames, writerBytes := 0, 0, 0, 0
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
		for _, mode := range []string{"class_confirm", "revive_confirm"} {
			churchConfirmationPrototypeState(g, mode)
			questionProducer := churchConfirmationPrototypeQuestion(g)
			question, ok := questionProducer()
			if !ok {
				t.Fatal("original question absent", mode)
			}
			source, ok := g.composeNativeChurchSourceFrame()
			if !ok {
				t.Fatal("original source absent")
			}
			dialogue, ok := g.composeNativeChurchDialogueBase()
			if !ok {
				t.Fatal("original dialogue absent")
			}
			party, _ := json.Marshal(g.partyRoster)
			gold, rng, ui := g.gold, g.nativeRNGState, g.nativeClassUI
			for _, stage := range []struct {
				name    string
				raw     []byte
				compose func() ([]byte, bool)
			}{{"question", question, questionProducer}, {"source", source, g.composeNativeChurchSourceFrame}, {"dialogue", dialogue, g.composeNativeChurchDialogueBase}} {
				a, okA := g.composeModernFacilityMarker(&modernStableFacilitySource{resource: 14}, 0, stage.compose)
				b, okB := g.composeModernFacilityMarker(&modernStableFacilitySource{resource: 14}, 255, stage.compose)
				hd, runs, ok := churchListPrototypeCapture(g, stage.compose)
				if !okA || !okB || !ok || len(a) != 64000 || len(b) != 64000 || len(hd) != 64000 || stage.name != "dialogue" && len(runs) == 0 {
					t.Fatal("capture or writer unavailable", mode, stage.name)
				}
				for i := range a {
					if a[i] == b[i] {
						if a[i] != stage.raw[i] {
							t.Fatal("writer disagrees with original")
						}
						writerBytes++
					} else if hd[i] != stage.raw[i] {
						t.Fatal("HD capture changed bytes outside writer")
					}
				}
				stages++
			}
			h, err := g.prepareModernHDFacilityDialogue()
			if err != nil {
				t.Fatal(err)
			}
			for _, kind := range []string{"opening", "closing", "choice_only"} {
				if kind == "choice_only" && mode != "revive_confirm" {
					continue
				}
				g.nativeClassUIJob = nil
				var expected [][]byte
				if kind == "opening" {
					expected, err = campaign.NativeClassConfirmationOpeningFrames(question, ui.choices)
				} else {
					expected, err = campaign.NativeClassConfirmationClosingFrames(question, ui.choices)
				}
				if err != nil {
					t.Fatal(err)
				}
				for i := 0; i < 4; i++ {
					phase, index := fullRewardOpening, i+6
					if kind != "opening" {
						phase, index = fullRewardChoiceClosing, i
					}
					placements, err := h.buttons.placements(phase, index, 0, 0)
					if err != nil || len(placements) != 2 {
						t.Fatal("existing HD button geometry absent")
					}
					rebuilt := append([]byte(nil), question...)
					for _, p := range placements {
						if err := ui.choices[p.sourceIndex].BlitAt(rebuilt, 320, p.rect.Min.X, p.rect.Min.Y); err != nil {
							t.Fatal(err)
						}
					}
					if !bytes.Equal(rebuilt, expected[i]) {
						t.Fatal("existing HD button geometry differs from native")
					}
				}
				if kind == "closing" {
					tail, err := campaign.NativeClassListClosingFrames(source, dialogue)
					if err != nil {
						t.Fatal(err)
					}
					expected = append(expected, tail...)
				}
				called := false
				if !churchConfirmationPrototypeBegin(g, kind, func() { called = true }) || g.nativeClassUIJob == nil {
					t.Fatal("native confirmation job absent")
				}
				job := g.nativeClassUIJob
				if len(job.frames) != len(expected) || called || kind != "opening" && job.after == nil {
					t.Fatal("job shape or continuation changed")
				}
				for i := range expected {
					if !bytes.Equal(job.frames[i], expected[i]) {
						t.Fatal("native confirmation cache changed")
					}
					frames++
				}
				if kind == "closing" && !bytes.Equal(job.restore, source) || kind != "closing" && len(job.restore) != 0 {
					t.Fatal("native restore changed")
				}
				jobs++
			}
			after, _ := json.Marshal(g.partyRoster)
			if !bytes.Equal(party, after) || g.gold != gold || g.nativeRNGState != rng || g.nativeClassUI != ui || g.nativeShopUI != nil || g.font.indexedHD != nil || g.modernHDText != nil {
				t.Fatal("prototype leaked player or UI state")
			}
		}
	}
	var cells []map[string]any
	for _, index := range []int{16, 17} {
		cell := g.nativeClassUI.choices[index]
		cells = append(cells, map[string]any{"index": index, "width": cell.Width, "height": cell.Height, "sha256": fmt.Sprintf("%x", sha256.Sum256(cell.Pixels)), "zero_pixels": bytes.Count(cell.Pixels, []byte{0})})
	}
	t.Logf("stages=%d jobs=%d original_frames=%d writer_bytes=%d cells=%v", stages, jobs, frames, writerBytes, cells)
	if out := os.Getenv("FD2_TEST_FACILITY_OUT"); out != "" {
		if err := os.MkdirAll(out, 0755); err != nil {
			t.Fatal(err)
		}
		raw, _ := json.MarshalIndent(map[string]any{"status": "PASS-CHURCH-CONFIRMATION-PROTOTYPE", "stages": stages, "jobs": jobs, "original_frames": frames, "writer_bytes": writerBytes, "without_shop_assets": true, "cells": cells}, "", "  ")
		if err := os.WriteFile(filepath.Join(out, "receipt.json"), append(raw, '\n'), 0644); err != nil {
			t.Fatal(err)
		}
	}
}
