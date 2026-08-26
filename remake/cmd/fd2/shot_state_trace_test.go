package main

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	"github.com/wicanr2/fd2_re/remake/internal/battle"
	"github.com/wicanr2/fd2_re/remake/internal/campaign"
	"github.com/wicanr2/fd2_re/remake/internal/ending"
)

func TestWriteShotStateTraceRecordsNativeInteractionState(t *testing.T) {
	unit := &battle.Unit{X: 8, Y: 17, HP: 10, OnField: true, NativeCommandMask: [5]byte{1}}
	g := &Game{
		frame:                        500,
		curX:                         8,
		curY:                         17,
		sel:                          unit,
		ring:                         true,
		nativeCommandOpen:            true,
		nativeCommand0Targeting:      true,
		nativeCommandTargetID:        0,
		nativeContinueOpeningConfirm: true,
		nativeSystemCursorOverlay:    true,
		camp: campaign.NewRunner(&campaign.Campaign{
			Start: "battle_ch01",
			Nodes: map[string]*campaign.Node{
				"battle_ch01": {Type: "battle"},
			},
		}),
		st: &battle.State{
			W: 30, H: 22, Turn: 7, NativeRoundCounter: 9, Units: []*battle.Unit{unit},
			HasNativeMapViewState: true,
			NativeMapViewState: battle.NativeMapViewState{
				CameraX: 1, CameraY: 13, CursorX: 8, CursorY: 17,
				VisibleCursorX: 7, VisibleCursorY: 4,
			},
		},
	}
	path := filepath.Join(t.TempDir(), "shot-state.json")
	if err := g.writeShotStateTrace(path); err != nil {
		t.Fatal(err)
	}
	raw, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	var got screenshotStateTrace
	if err := json.Unmarshal(raw, &got); err != nil {
		t.Fatal(err)
	}
	if got.Frame != 500 || got.CampaignNode != "battle_ch01" ||
		got.Cursor != [2]int{8, 17} || !got.HasSelection ||
		got.Selection == nil || *got.Selection != [2]int{8, 17} || !got.ActionOverlayOpen ||
		!got.NativeCommandOpen || !got.NativeCommandTargeting ||
		got.NativeCommandTargetID == nil || *got.NativeCommandTargetID != 0 || got.Battle == nil ||
		got.NativeItemTargeting || got.NativeItemTargetID != nil || got.NativeItemRelocating ||
		got.Battle.NativeMapView == nil || got.Battle.NativeMapView.CameraY != 13 ||
		!got.NativeContinueOpeningConfirm || !got.NativeContinueCursorOverlay || got.DialogCount != 0 ||
		got.BattleEventActive || got.NativeTurnStagingActive || got.CursorUnit == nil ||
		!got.CursorUnit.OnField || got.CursorUnit.NativeCommandCount != 1 {
		t.Fatalf("shot state trace=%#v", got)
	}
}

func TestWriteShotStateTraceRecordsNativeItemModal(t *testing.T) {
	for _, tc := range []struct {
		name       string
		targeting  bool
		relocating bool
		wantID     bool
	}{
		{name: "first target", targeting: true, wantID: true},
		{name: "relocation destination", relocating: true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			g := &Game{
				nativeItemTargeting:  tc.targeting,
				nativeItemTargetID:   0,
				nativeItemRelocating: tc.relocating,
			}
			path := filepath.Join(t.TempDir(), "shot-state-item.json")
			if err := g.writeShotStateTrace(path); err != nil {
				t.Fatal(err)
			}
			raw, err := os.ReadFile(path)
			if err != nil {
				t.Fatal(err)
			}
			var got screenshotStateTrace
			if err := json.Unmarshal(raw, &got); err != nil {
				t.Fatal(err)
			}
			if got.NativeItemTargeting != tc.targeting ||
				got.NativeItemRelocating != tc.relocating ||
				(got.NativeItemTargetID != nil) != tc.wantID {
				t.Fatalf("item modal trace=%#v", got)
			}
			if tc.wantID && *got.NativeItemTargetID != 0 {
				t.Fatalf("item modal ID=%d, want 0", *got.NativeItemTargetID)
			}
		})
	}
}

func TestWriteShotStateTraceOmitsSelectionWithoutOwner(t *testing.T) {
	g := &Game{
		curX: 8, curY: 17, ring: true, nativeSystemCursorOverlay: true,
		st: &battle.State{W: 24, H: 24},
	}
	path := filepath.Join(t.TempDir(), "shot-state-empty-cursor.json")
	if err := g.writeShotStateTrace(path); err != nil {
		t.Fatal(err)
	}
	raw, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	var got screenshotStateTrace
	if err := json.Unmarshal(raw, &got); err != nil {
		t.Fatal(err)
	}
	if got.HasSelection || got.Selection != nil || !got.NativeContinueCursorOverlay ||
		got.NativeCommandTargeting || got.NativeCommandTargetID != nil ||
		got.NativeItemTargeting || got.NativeItemTargetID != nil || got.NativeItemRelocating {
		t.Fatalf("empty cursor trace=%#v", got)
	}
}

func TestWriteShotStateTraceRecordsStableChurchMenuState(t *testing.T) {
	g := &Game{
		camp: campaign.NewRunner(&campaign.Campaign{
			Start: "church_ch02",
			Nodes: map[string]*campaign.Node{
				"church_ch02": {Type: "church"},
			},
		}),
		churchMode: "menu", churchSel: 0, nativeChurchUIPulse: 2,
		gold: 1000, nativeChurchUIShotHold: true,
	}
	path := filepath.Join(t.TempDir(), "shot-state-church.json")
	if err := g.writeShotStateTrace(path); err != nil {
		t.Fatal(err)
	}
	raw, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	var got screenshotStateTrace
	if err := json.Unmarshal(raw, &got); err != nil {
		t.Fatal(err)
	}
	if got.CampaignNode != "church_ch02" || got.Church == nil ||
		got.Church.Mode != "menu" || got.Church.Selection != 0 ||
		got.Church.Pulse != 2 || got.Church.Gold != 1000 || !got.Church.ShotHold {
		t.Fatalf("church trace=%#v", got)
	}
}

func TestWriteShotStateTracePreservesSourceBoundEndingGate(t *testing.T) {
	g := &Game{nativeEnding: &nativeEndingPreview{
		campaignSourceBound: true,
		audioCueConsumed:    true,
		montage: &ending.MontageCycle{
			Phase:     ending.MontagePhasePortrait,
			PlanIndex: 0,
			Plans:     make([]ending.PartyCyclePlan, 3),
		},
		player: &ending.Player{
			State:   ending.PlaybackBlocked,
			Blocked: &ending.Segment{Op: "native_finale_montage_opaque", Source: "0x2c548"},
		},
	}}
	path := filepath.Join(t.TempDir(), "shot-state-ending.json")
	if err := g.writeShotStateTrace(path); err != nil {
		t.Fatal(err)
	}
	raw, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	var got screenshotStateTrace
	if err := json.Unmarshal(raw, &got); err != nil {
		t.Fatal(err)
	}
	if got.NativeEnding == nil || got.NativeEnding.PlaybackState != string(ending.PlaybackBlocked) ||
		got.NativeEnding.BlockedOp != "native_finale_montage_opaque" ||
		got.NativeEnding.BlockedSource != "0x2c548" || !got.NativeEnding.CampaignSourceBound ||
		!got.NativeEnding.AudioCueConsumed || got.NativeEnding.MontagePhase != string(ending.MontagePhasePortrait) ||
		got.NativeEnding.MontagePlanIndex == nil || *got.NativeEnding.MontagePlanIndex != 0 ||
		got.NativeEnding.MontagePlanCount != 3 {
		t.Fatalf("ending trace=%#v", got.NativeEnding)
	}
}

func TestWriteShotStateTracePreservesNativeEndingDialogueOwner(t *testing.T) {
	playback := &ending.NativeDialoguePlayback{
		Blocks:    make([]ending.NativeDialogueBlockFrames, 5),
		Block:     0,
		Utterance: 0,
		Page:      0,
		Phase:     ending.NativeDialogueWaiting,
	}
	g := &Game{nativeEnding: &nativeEndingPreview{
		player: &ending.Player{
			State:   ending.PlaybackBlocked,
			Blocked: &ending.Segment{Op: "native_text_branch_opaque", Source: "0x2be44"},
		},
		dialogue: playback,
	}}
	path := filepath.Join(t.TempDir(), "shot-state-ending-dialogue.json")
	if err := g.writeShotStateTrace(path); err != nil {
		t.Fatal(err)
	}
	raw, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	var got screenshotStateTrace
	if err := json.Unmarshal(raw, &got); err != nil {
		t.Fatal(err)
	}
	endingTrace := got.NativeEnding
	if endingTrace == nil || endingTrace.DialoguePhase != "waiting" ||
		endingTrace.DialogueBlock == nil || *endingTrace.DialogueBlock != 0 ||
		endingTrace.DialogueBlockCount != 5 || endingTrace.DialogueUtterance == nil ||
		*endingTrace.DialogueUtterance != 0 || endingTrace.DialoguePage == nil ||
		*endingTrace.DialoguePage != 0 || !endingTrace.DialogueWaiting {
		t.Fatalf("ending dialogue trace=%#v", endingTrace)
	}
}
