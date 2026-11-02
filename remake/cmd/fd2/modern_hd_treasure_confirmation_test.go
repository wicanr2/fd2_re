package main

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"image"
	"image/png"
	"os"
	"path/filepath"
	"reflect"
	"testing"

	"github.com/hajimehoshi/ebiten/v2"

	"github.com/wicanr2/fd2_re/remake/internal/battle"
	"github.com/wicanr2/fd2_re/remake/internal/campaign"
)

// Test-only source replay. Production admission and presentation are separate.
func treasureHDSourceState(t *testing.T, g *Game, figure int, reward battle.Treasure) *nativeSystemEndTurnUIState {
	t.Helper()
	portraits, err := loadNativeSeparatedPortrait(figure)
	if err != nil || len(portraits) < 4 {
		t.Fatal("actor portrait source", figure, err)
	}
	portraits = append(portraits[:0:0], portraits...)
	for i := range portraits {
		portraits[i].Pixels = append([]byte(nil), portraits[i].Pixels...)
	}
	u := g.nativePreparationUI
	s := &nativeSystemEndTurnUIState{source: append([]byte(nil), g.nativeMapVGA...), portraits: portraits,
		treasure: &nativeTreasurePrompt{actor: &battle.Unit{BattleFig: figure, HasBattleFig: true}, reward: reward, x: 7, y: 13}}
	s.dialogue, err = campaign.ComposeNativePreparationConfirmationDialogue(s.source, u.dialogue, portraits[0])
	if err != nil {
		t.Fatal(err)
	}
	s.question, err = campaign.ComposeNativeTreasureItemQuestion(s.dialogue, portraits[0], u.status.Strings, u.status.Font, reward.Hidden)
	if err != nil {
		t.Fatal(err)
	}
	switch reward.Kind {
	case "item":
		s.accepted, err = campaign.NativeTreasureItemResponseFrames(s.question, u.status.Strings, u.status.Font, reward.Value, reward.Hidden)
	case "gold":
		s.accepted, err = campaign.NativeTreasureGoldResponseFrames(s.question, u.status.Strings, u.status.Font, reward.Value, reward.Hidden)
	default:
		t.Fatal("unknown prototype kind")
	}
	if err != nil {
		t.Fatal(err)
	}
	s.canceled, err = campaign.NativeBattleEndTurnResponseFrames(s.question, u.status.Strings, u.status.Font, false)
	if err != nil {
		t.Fatal(err)
	}
	return s
}

func TestModernHDTreasureNativeProducers(t *testing.T) {
	g := systemHDTestGame(t)
	states, frames := 0, 0
	for _, figure := range []int{0, 9} {
		for _, kind := range []string{"item", "gold"} {
			for _, hidden := range []bool{false, true} {
				values := []int{192}
				if kind == "gold" {
					values = []int{0, 3500}
				}
				for _, value := range values {
					s := treasureHDSourceState(t, g, figure, battle.Treasure{Slot: 0, Kind: kind, Value: value, Hidden: hidden})
					originalPortrait := g.nativePreparationUI.portrait
					if !g.freezeModernHDSystemConfirmation(s) || s.hd.speaker != figure {
						t.Fatal("treasure source admission", figure, kind, hidden, value, g.loadErr)
					}
					if !reflect.DeepEqual(originalPortrait, g.nativePreparationUI.portrait) {
						t.Fatal("changed shared system portrait")
					}
					h, u := s.hd, g.nativePreparationUI
					opening, err := campaign.NativePreparationConfirmationOpeningFrames(s.source, s.dialogue, s.question, u.choices)
					if err != nil {
						t.Fatal(err)
					}
					close, err := campaign.NativeClassConfirmationClosingFrames(s.question, u.choices)
					if err != nil {
						t.Fatal(err)
					}
					blank, err := campaign.NativeClassListClosingFrames(s.source, s.dialogue)
					if err != nil {
						t.Fatal(err)
					}
					for phase, wants := range map[fullRewardPhase][][]byte{fullRewardOpening: opening, fullRewardChoiceClosing: close, fullRewardBlankClosing: blank} {
						for i, want := range wants {
							got, _, err := h.frame(s, s.source, u.status.Font, phase, i, 0, 0, nil)
							if err != nil || !bytes.Equal(got, want) {
								t.Fatal("changed native schedule", figure, kind, phase, i, err)
							}
							frames++
						}
					}
					for _, accepted := range []bool{false, true} {
						s.acceptedOutcome = accepted
						wants := s.canceled
						if accepted {
							wants = s.accepted
						}
						for i, want := range wants {
							got, _, err := h.frame(s, s.source, u.status.Font, fullRewardResponse, i, 0, 0, nil)
							if err != nil || !bytes.Equal(got, want) {
								t.Fatal("changed response", figure, kind, value, accepted, i, err)
							}
							frames++
						}
					}
					g.nativeSystemEndTurnUI = s
					g.nativeClassUIJob = nil
					g.nativeSystemEndTurnConfirm = true
					g.nativeSystemEndTurnDelay = 0
					screen := ebiten.NewImage(640, 400)
					finish := g.beginModernHDText(screen)
					before, _ := json.Marshal([]any{g.gold, g.nativeRNGState, s.treasure.actor.Inventory, s.treasure.reward})
					if !g.drawModernHDSystemConfirmation(screen) || len(g.modernHDText.runs) == 0 || len(g.modernHDText.choices) != 2 || len(g.modernHDText.panels) != 1 {
						t.Fatal("missing HD treasure question", figure, kind, value, g.loadErr)
					}
					finish()
					g.nativeSystemEndTurnConfirm = false
					s.acceptedOutcome = true
					s.treasure.claimed = true
					s.treasure.awaitAck = true
					h.kind = fullRewardResponse
					seal := h.treasureSeal
					if _, err := g.prepareModernHDSystemConfirmation(s, nil); err != nil || seal != treasureConfirmationSourceSeal(s) {
						t.Fatal("valid ACK latch treated as source mutation", err)
					}
					finish = g.beginModernHDText(screen)
					if !g.drawModernHDSystemConfirmation(screen) || len(g.modernHDText.runs) == 0 || len(g.modernHDText.choices) != 0 || len(g.modernHDText.panels) != 1 {
						t.Fatal("HD vanished while awaiting treasure ACK", figure, kind, value, g.loadErr)
					}
					finish()
					screen.Dispose()
					after, _ := json.Marshal([]any{g.gold, g.nativeRNGState, s.treasure.actor.Inventory, s.treasure.reward})
					if !bytes.Equal(before, after) {
						t.Fatal("Draw changed transaction")
					}
					g.nativeSystemEndTurnUI = nil
					states++
				}
			}
		}
	}
	t.Logf("treasure states=%d native_frames=%d; shared system portrait, raw buffers, gold and RNG kept", states, frames)
}

func TestModernHDTreasureAtomicRejection(t *testing.T) {
	for _, fault := range []string{"reward", "hidden", "slot", "position", "actor", "identity", "portrait", "source", "question", "response", "mixed_owner"} {
		t.Run(fault, func(t *testing.T) {
			g := systemHDTestGame(t)
			s := treasureHDSourceState(t, g, 0, battle.Treasure{Slot: 0, Kind: "item", Value: 192})
			if !g.freezeModernHDSystemConfirmation(s) {
				t.Fatal(g.loadErr)
			}
			g.nativeSystemEndTurnUI = s
			g.nativeSystemEndTurnConfirm = true
			g.interfaceMode = playerInterfaceOriginal
			g.modernThemeCache = &modernStoryPortraitSet{}
			old := s.hd
			switch fault {
			case "reward":
				s.treasure.reward.Value++
			case "hidden":
				s.treasure.reward.Hidden = true
			case "slot":
				s.treasure.reward.Slot++
			case "position":
				s.treasure.x++
			case "actor":
				s.treasure.actor = nil
			case "identity":
				s.treasure.actor.BattleFig = 9
			case "portrait":
				s.portraits[0].Pixels[0] ^= 1
			case "source":
				s.source[0] ^= 1
			case "question":
				s.question[0] ^= 1
			case "response":
				s.accepted[0][0] ^= 1
			case "mixed_owner":
				s.exitProgram = true
			}
			if err := g.setInterfaceMode(playerInterfaceHD); err == nil || s.hd != old || g.interfaceMode != playerInterfaceOriginal {
				t.Fatal("source corruption partially admitted", fault, err)
			}
		})
	}
}

func TestModernHDTreasureProductionAdmission(t *testing.T) {
	g, actor := rewardStageGame(t, nil)
	g.interfaceMode = playerInterfaceHD
	before, _ := json.Marshal([]any{g.gold, g.nativeRNGState, actor.Inventory, actor.InventorySlots, g.st.OpenedTreasure})
	reward := battle.Treasure{Slot: 0, Kind: "gold", Value: 3500, Hidden: true}
	if err := g.beginNativeTreasureItemPrompt(actor, reward); err != nil {
		t.Fatal(err)
	}
	s, job := g.nativeSystemEndTurnUI, g.nativeClassUIJob
	if s == nil || s.hd == nil || s.hd.speaker != actor.BattleFig || job == nil || s.hd.job != job || s.hd.kind != fullRewardOpening || len(job.frames) != 10 {
		t.Fatal("production opening was not bound")
	}
	after, _ := json.Marshal([]any{g.gold, g.nativeRNGState, actor.Inventory, actor.InventorySlots, g.st.OpenedTreasure})
	if !bytes.Equal(before, after) {
		t.Fatal("source admission changed transaction")
	}
	if _, err := g.prepareModernHDSystemConfirmation(s, nil); err != nil {
		t.Fatal(err)
	}
}

func TestModernHDTreasureSourcePrototype(t *testing.T) {
	out := os.Getenv("FD2_HD_TREASURE_PROTOTYPE_OUT")
	if out == "" {
		t.Skip("explicit source export output required")
	}
	g := systemHDTestGame(t)
	rows := []map[string]any{}
	total := 0
	sha := func(b []byte) string { s := sha256.Sum256(b); return hex.EncodeToString(s[:]) }
	for _, figure := range []int{0, 9} {
		for _, kind := range []string{"item", "gold"} {
			for _, hidden := range []bool{false, true} {
				value := 192
				if kind == "gold" {
					value = 3500
				}
				reward := battle.Treasure{Slot: 0, Kind: kind, Value: value, Hidden: hidden}
				s := treasureHDSourceState(t, g, figure, reward)
				before := append([]byte(nil), s.source...)
				opening, err := campaign.NativePreparationConfirmationOpeningFrames(s.source, s.dialogue, s.question, g.nativePreparationUI.choices)
				if err != nil || len(opening) != 10 {
					t.Fatal("opening", err)
				}
				choices, err := campaign.NativeClassConfirmationClosingFrames(s.question, g.nativePreparationUI.choices)
				if err != nil || len(choices) != 4 {
					t.Fatal("choice close", err)
				}
				blank, err := campaign.NativeClassListClosingFrames(s.source, s.dialogue)
				if err != nil || len(blank) != 5 {
					t.Fatal("dialogue close", err)
				}
				if !bytes.Equal(s.source, before) {
					t.Fatal("source changed")
				}
				stem := fmt.Sprintf("actor-%03d-%s-hidden-%t", figure, kind, hidden)
				for name, raw := range map[string][]byte{"question": s.question, "accepted": s.accepted[len(s.accepted)-1], "canceled": s.canceled[len(s.canceled)-1]} {
					im := image.NewPaletted(image.Rect(0, 0, 320, 200), g.nativeClassUI.palette)
					copy(im.Pix, raw)
					f, err := os.Create(filepath.Join(out, stem+"-"+name+".png"))
					if err != nil {
						t.Fatal(err)
					}
					err = png.Encode(f, im)
					_ = f.Close()
					if err != nil {
						t.Fatal(err)
					}
				}
				n := 10 + 4 + 5 + len(s.accepted) + len(s.canceled)
				total += n
				rows = append(rows, map[string]any{"figure": figure, "kind": kind, "hidden": hidden, "value": value, "opening_frames": 10, "choice_close_frames": 4, "blank_close_frames": 5, "accepted_frames": len(s.accepted), "canceled_frames": len(s.canceled), "source_sha256": sha(s.source), "question_sha256": sha(s.question), "accepted_final_sha256": sha(s.accepted[len(s.accepted)-1]), "canceled_final_sha256": sha(s.canceled[len(s.canceled)-1]), "actor_portrait_sha256": sha(s.portraits[0].Pixels)})
			}
		}
	}
	report := map[string]any{"status": "PASS-SOURCE-PROTOTYPE", "scope": "existing native campaign producers; typed black-map fixture; no original oracle or normal reachability proof", "states": len(rows), "frames": total, "rows": rows}
	raw, _ := json.MarshalIndent(report, "", "  ")
	if err := os.WriteFile(filepath.Join(out, "treasure-source-prototype.json"), append(raw, '\n'), 0644); err != nil {
		t.Fatal(err)
	}
	t.Logf("source states=%d native frames=%d", len(rows), total)
}
