package main

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/wicanr2/fd2_re/remake/internal/campaign"
	"github.com/wicanr2/fd2_re/remake/internal/ending"
)

func TestNativeEndingPreviewReachesRecoveredPhase0MontageGate(t *testing.T) {
	const base = "../../../org_game/炎龍騎士團/FLAME2"
	for _, name := range []string{"FDOTHER.DAT", "FDTXT.DAT", "ANI.DAT"} {
		if _, err := os.Stat(filepath.Join(base, name)); os.IsNotExist(err) {
			t.Skip("player-provided ending resources are unavailable")
		} else if err != nil {
			t.Fatal(err)
		}
	}
	t.Setenv("FD2_FDOTHER", filepath.Join(base, "FDOTHER.DAT"))
	t.Setenv("FD2_FDTXT", filepath.Join(base, "FDTXT.DAT"))
	t.Setenv("FD2_ANI", filepath.Join(base, "ANI.DAT"))
	t.Setenv("FD2_ENDING_CHAPTER", "29")
	preview, err := newNativeEndingPreview()
	if err != nil {
		t.Fatal(err)
	}
	p := preview.player
	for _, elapsed := range []int{0, 1000, 2500, 0, 256, 2000} {
		if _, err := p.Advance(elapsed); err != nil {
			t.Fatal(err)
		}
	}
	if p.State != ending.PlaybackBlocked || p.Blocked == nil ||
		p.Blocked.Source != "0x2be44" || !p.ResumeBlockedDialogue() {
		t.Fatalf("first dialogue gate = state=%s blocked=%#v", p.State, p.Blocked)
	}
	if state, err := p.Advance(5000); err != nil || state != ending.PlaybackBlocked ||
		p.Blocked == nil || p.Blocked.Source != "0x2bf1c" ||
		!p.ResumeBlockedDialogue() {
		t.Fatalf("second dialogue gate = state=%s err=%v blocked=%#v", state, err, p.Blocked)
	}
	if state, err := p.Advance(7500); err != nil || state != ending.PlaybackBlocked ||
		p.Blocked == nil || p.Blocked.Op != "native_finale_montage_opaque" ||
		p.Blocked.Source != "0x2c548" {
		t.Fatalf("montage gate = state=%s err=%v blocked=%#v", state, err, p.Blocked)
	}
}

func TestNativeEndingDialogLinesUseNativePortraitIDs(t *testing.T) {
	lines, err := nativeEndingDialogLines([]ending.DialogueBlock{{PortraitID: 37, Script: "ch30.json", SceneIndex: 1, Line: 0, Count: 6}, {PortraitID: 21, Script: "ch30.json", SceneIndex: 1, Line: 6, Count: 2}})
	if err != nil || len(lines) != 8 {
		t.Fatalf("lines=%d err=%v", len(lines), err)
	}
	for i, line := range lines {
		want := 37
		if i >= 6 {
			want = 21
		}
		if line.Speaker != want || line.Text == "" {
			t.Fatalf("line %d=%#v want speaker %d", i, line, want)
		}
	}
}

func TestNativeEndingDialogueResumeAllowsSecondNativeTextGate(t *testing.T) {
	player := &ending.Player{
		State: ending.PlaybackBlocked,
		Blocked: &ending.Segment{
			Op: "native_text_branch_opaque",
			ElseDialogue: []ending.DialogueBlock{{
				PortraitID: 37, Script: "ch30.json", SceneIndex: 1, Line: 0, Count: 1,
			}},
		},
		Segment: 9,
	}
	g := &Game{nativeEnding: &nativeEndingPreview{
		player: player, chapter: 29, queued: true,
	}}
	if !g.resumeNativeEndingDialogue() {
		t.Fatal("first recovered ending dialogue gate did not resume")
	}
	if g.nativeEnding.queued || player.State != ending.PlaybackRunning ||
		player.Blocked != nil || player.Segment != 10 {
		t.Fatalf("first resume = queued=%v state=%s blocked=%#v segment=%d",
			g.nativeEnding.queued, player.State, player.Blocked, player.Segment)
	}

	player.State = ending.PlaybackBlocked
	player.Blocked = &ending.Segment{
		Op: "native_text_branch_opaque",
		ElseDialogue: []ending.DialogueBlock{{
			PortraitID: 45, Script: "ch30.json", SceneIndex: 1, Line: 13, Count: 1,
		}},
	}
	player.Segment = 13
	if err := g.queueNativeEndingDialogue(); err != nil {
		t.Fatal(err)
	}
	if !g.nativeEnding.queued || len(g.dialog) != 1 ||
		g.dialog[0].Speaker != 45 || g.dialog[0].Text == "" {
		t.Fatalf("second recovered ending dialogue = queued=%v lines=%#v",
			g.nativeEnding.queued, g.dialog)
	}
}

func TestApproximateCampaignEndingConsumesOnlyVerifiedGateCueThenReturnsToEditableEpilogue(t *testing.T) {
	p := &ending.Player{
		Timeline: ending.Timeline{AudioCues: []ending.AudioCue{{
			Source: "0x2c5cf", Track: 4, DriverArg: 0, AfterGate: "0x2c548", Trigger: "verified gate",
		}}},
		State:   ending.PlaybackBlocked,
		Blocked: &ending.Segment{Op: "native_finale_montage_opaque", Source: "0x2c548"},
	}
	g := &Game{nativeEnding: &nativeEndingPreview{player: p, campaignApproximate: true}}
	t.Setenv("FD2_MUTE", "1")
	g.consumeNativeEndingAudioAtGate()
	if !g.nativeEnding.audioCueConsumed {
		t.Fatal("verified ending audio cue was not consumed at 0x2c548")
	}
	if !g.finishCampaignNativeEndingFallback() || g.nativeEnding != nil || g.endingNotice == "" {
		t.Fatalf("campaign fallback = preview=%#v notice=%q", g.nativeEnding, g.endingNotice)
	}
}

func TestDirectEndingPreviewCannotUseApproximateCampaignFallback(t *testing.T) {
	g := &Game{nativeEnding: &nativeEndingPreview{
		player: &ending.Player{
			State:   ending.PlaybackBlocked,
			Blocked: &ending.Segment{Op: "native_finale_montage_opaque", Source: "0x2c548"},
		},
	}}
	if g.finishCampaignNativeEndingFallback() || g.nativeEnding == nil {
		t.Fatalf("direct preview crossed fail-closed montage boundary: %#v", g.nativeEnding)
	}
}

func TestCampaignEndingRejectsProgrammaticUnprovenPrefix(t *testing.T) {
	if _, err := newNativeEndingPreviewForCampaign(&campaign.NativeEndingPrefixConfig{
		Timeline: nativeEndingTimelinePath, Handler: "0x2c548", Chapter: 29,
		Mode: campaign.NativeEndingPrefixRecoveredOnly,
	}); err == nil {
		t.Fatal("programmatic unproven ending prefix was accepted")
	}
}

func TestApproximateCampaignFinalNodeConsumesRecoveredPrefixThenStops(t *testing.T) {
	const base = "../../../org_game/炎龍騎士團/FLAME2"
	for _, name := range []string{"FDOTHER.DAT", "FDTXT.DAT", "ANI.DAT"} {
		if _, err := os.Stat(filepath.Join(base, name)); os.IsNotExist(err) {
			t.Skip("player-provided ending resources are unavailable")
		} else if err != nil {
			t.Fatal(err)
		}
	}
	t.Setenv("FD2_FDOTHER", filepath.Join(base, "FDOTHER.DAT"))
	t.Setenv("FD2_FDTXT", filepath.Join(base, "FDTXT.DAT"))
	t.Setenv("FD2_ANI", filepath.Join(base, "ANI.DAT"))
	t.Setenv("FD2_MUTE", "1")
	c, err := campaign.Load("../../assets/scenarios/campaign_full.json")
	if err != nil {
		t.Fatal(err)
	}
	runner := campaign.NewRunner(c)
	runner.Cur = "ending"
	g := &Game{camp: runner, approximateMode: true}
	g.enterNode()
	if g.nativeEnding == nil || !g.nativeEnding.campaignApproximate {
		t.Fatalf("approximate final node did not admit recovered prefix: %#v", g.nativeEnding)
	}
	p := g.nativeEnding.player
	for _, elapsed := range []int{0, 1000, 2500, 0, 256, 2000} {
		if _, err := p.Advance(elapsed); err != nil {
			t.Fatal(err)
		}
	}
	if !p.ResumeBlockedDialogue() {
		t.Fatal("first recovered text gate was unavailable")
	}
	if _, err := p.Advance(5000); err != nil || !p.ResumeBlockedDialogue() {
		t.Fatalf("second recovered text gate err=%v blocked=%#v", err, p.Blocked)
	}
	if _, err := p.Advance(7500); err != nil || !g.nativeEnding.awaitingCampaignFallback() {
		t.Fatalf("unrecovered montage boundary err=%v preview=%#v", err, g.nativeEnding)
	}
	g.consumeNativeEndingAudioAtGate()
	if !g.nativeEnding.audioCueConsumed {
		t.Fatal("verified FDMUS_004 cue was not consumed at the recovered boundary")
	}
	if !g.finishCampaignNativeEndingFallback() || g.nativeEnding != nil || g.endingNotice == "" {
		t.Fatalf("approximate final fallback did not return to editable ending: preview=%#v notice=%q", g.nativeEnding, g.endingNotice)
	}
}
