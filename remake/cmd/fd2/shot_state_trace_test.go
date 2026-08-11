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
		nativeContinueOpeningConfirm: true,
		nativeContinueCursorOverlay:  true,
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
		!got.NativeCommandOpen || got.Battle == nil ||
		got.Battle.NativeMapView == nil || got.Battle.NativeMapView.CameraY != 13 ||
		!got.NativeContinueOpeningConfirm || !got.NativeContinueCursorOverlay || got.DialogCount != 0 ||
		got.BattleEventActive || got.NativeTurnStagingActive || got.CursorUnit == nil ||
		!got.CursorUnit.OnField || got.CursorUnit.NativeCommandCount != 1 {
		t.Fatalf("shot state trace=%#v", got)
	}
}

func TestWriteShotStateTraceOmitsSelectionWithoutOwner(t *testing.T) {
	g := &Game{
		curX: 8, curY: 17, ring: true, nativeContinueCursorOverlay: true,
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
	if got.HasSelection || got.Selection != nil || !got.NativeContinueCursorOverlay {
		t.Fatalf("empty cursor trace=%#v", got)
	}
}

func TestWriteShotStateTracePreservesApproximateEndingGate(t *testing.T) {
	g := &Game{nativeEnding: &nativeEndingPreview{
		campaignApproximate: true,
		audioCueConsumed:    true,
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
		got.NativeEnding.BlockedSource != "0x2c548" || !got.NativeEnding.CampaignApproximate ||
		!got.NativeEnding.AudioCueConsumed {
		t.Fatalf("ending trace=%#v", got.NativeEnding)
	}
}
