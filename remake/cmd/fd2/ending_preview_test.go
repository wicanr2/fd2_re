package main

import (
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/wicanr2/fd2_re/remake/internal/battle"
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
	g := &Game{nativeEnding: &nativeEndingPreview{
		player: p, campaignApproximate: true, montageStartAttempted: true,
		montageStartError: "test admission failure",
	}}
	t.Setenv("FD2_MUTE", "1")
	g.consumeNativeEndingAudioAtGate()
	if !g.nativeEnding.audioCueConsumed {
		t.Fatal("verified ending audio cue was not consumed at 0x2c548")
	}
	if !g.finishCampaignNativeEndingFallback() || g.nativeEnding != nil || g.endingNotice == "" {
		t.Fatalf("campaign fallback = preview=%#v notice=%q", g.nativeEnding, g.endingNotice)
	}
}

func TestNativeEndingMontageRecordsUseOnlyPersistentRawProvenance(t *testing.T) {
	order := []int{9, 4}
	roster := map[int]battle.Unit{
		9: {
			Fig: 99, BattleFig: 4, HasBattleFig: true,
			NativeRecordByte6: 2, HasNativeRecordByte6: true,
			NativeRecordByte8: 7, HasNativeRecordByte8: true,
			NativeRecordClass: 3, HasNativeRecordClass: true,
		},
		4: {
			Fig: 88, BattleFig: 5, HasBattleFig: true,
			NativeRecordByte6: 0, HasNativeRecordByte6: true,
			NativeIdentity: 6, HasNativeIdentity: true,
			NativeRecordClass: 4, HasNativeRecordClass: true,
		},
	}
	units, groups, err := nativeEndingMontageRecords(order, roster)
	if err != nil {
		t.Fatal(err)
	}
	if len(units) != 2 || string(groups) != string([]byte{4, 5}) ||
		units[0][6] != 2 || units[0][7] != 4 || units[0][8] != 7 || units[0][0x20] != 3 ||
		units[1][6] != 0 || units[1][7] != 5 || units[1][8] != 6 || units[1][0x20] != 4 {
		t.Fatalf("montage records=%#v groups=%v", units, groups)
	}
	broken := roster[9]
	broken.HasBattleFig = false
	roster[9] = broken
	if _, _, err := nativeEndingMontageRecords(order, roster); err == nil {
		t.Fatal("missing raw BattleFig provenance was accepted")
	}
}

func TestApproximateCampaignMontageStartsFromPersistentLoadCHOrder(t *testing.T) {
	const base = "../../../org_game/炎龍騎士團/FLAME2"
	for _, name := range []string{"FDOTHER.DAT", "FDTXT.DAT", "ANI.DAT", "TAI.DAT", "FIGANI.DAT", "DATO.DAT"} {
		if _, err := os.Stat(filepath.Join(base, name)); err != nil {
			t.Skip("player-provided ending resources are unavailable")
		}
	}
	t.Setenv("FD2_FDOTHER", filepath.Join(base, "FDOTHER.DAT"))
	t.Setenv("FD2_FDTXT", filepath.Join(base, "FDTXT.DAT"))
	t.Setenv("FD2_ANI", filepath.Join(base, "ANI.DAT"))
	preview, err := newNativeEndingPreview()
	if err != nil {
		t.Fatal(err)
	}
	preview.campaignApproximate = true
	for _, elapsed := range []int{0, 1000, 2500, 0, 256, 2000} {
		if _, err := preview.player.Advance(elapsed); err != nil {
			t.Fatal(err)
		}
	}
	if !preview.player.ResumeBlockedDialogue() {
		t.Fatal("first recovered text gate was unavailable")
	}
	if _, err := preview.player.Advance(5000); err != nil || !preview.player.ResumeBlockedDialogue() {
		t.Fatalf("second recovered text gate err=%v blocked=%#v", err, preview.player.Blocked)
	}
	if _, err := preview.player.Advance(7500); err != nil || !preview.atNativeMontageGate() {
		t.Fatalf("montage gate err=%v preview=%#v", err, preview)
	}
	unit := func(side byte, group byte) battle.Unit {
		return battle.Unit{
			Fig: 99, BattleFig: int(group), HasBattleFig: true,
			NativeRecordByte6: side, HasNativeRecordByte6: true,
			NativeRecordByte8: group, HasNativeRecordByte8: true,
			NativeRecordClass: 2, HasNativeRecordClass: true,
		}
	}
	g := &Game{
		nativeEnding:    preview,
		approximateMode: true,
		partyMembers:    map[int]bool{0: true, 1: true, 2: true},
		partyJoinOrder:  []int{0, 1, 2},
		partyRoster: map[int]battle.Unit{
			0: unit(2, 4), 1: unit(0, 4), 2: unit(2, 4),
		},
	}
	if err := g.startCampaignNativeMontage(); err != nil {
		t.Fatal(err)
	}
	if g.nativeEnding.montage == nil || len(g.nativeEnding.montage.Units) != 3 ||
		g.nativeEnding.montage.Units[0][6] != 2 || g.nativeEnding.montage.Units[0][7] != 4 {
		t.Fatalf("persistent montage admission=%#v", g.nativeEnding.montage)
	}
	if g.finishCampaignNativeEndingFallback() {
		t.Fatal("editable epilogue became available before the admitted montage completed")
	}
	now := time.Unix(1, 0)
	if err := g.nativeEnding.advance(now, &g.nativeRNGState); err != nil {
		t.Fatal(err)
	}
	if !g.nativeEnding.runningCampaignMontage() || g.nativeEnding.montage.Phase != ending.MontagePhaseSecondary {
		t.Fatalf("montage did not begin from recovered gate: %#v", g.nativeEnding.montage)
	}
	for steps := 0; g.nativeEnding.montage.Phase != ending.MontagePhasePortrait && steps < 512; steps++ {
		now = now.Add(approximateNativeMontageTick)
		if err := g.nativeEnding.advance(now, &g.nativeRNGState); err != nil {
			t.Fatalf("reach portrait step %d: %v", steps, err)
		}
	}
	if g.nativeEnding.montage.Phase != ending.MontagePhasePortrait {
		t.Fatalf("campaign montage did not reach a portrait: phase=%s", g.nativeEnding.montage.Phase)
	}
	// Game.Update records a raw input change here. The preview must carry it
	// until the recovered portrait boundary consumes it; it must not interpret
	// a particular key or jump out before the current portrait completes.
	g.nativeEnding.montageInputPending = true
	for steps := 0; g.nativeEnding.montage.PlanIndex != len(g.nativeEnding.montage.Plans)-1 && steps < 1024; steps++ {
		now = now.Add(approximateNativeMontageTick)
		if err := g.nativeEnding.advance(now, &g.nativeRNGState); err != nil {
			t.Fatalf("consume raw input step %d: %v", steps, err)
		}
	}
	if g.nativeEnding.montageInputPending || g.nativeEnding.montage.PlanIndex != len(g.nativeEnding.montage.Plans)-1 {
		t.Fatalf("campaign raw input did not select the final loop: pending=%v phase=%s plan=%d/%d",
			g.nativeEnding.montageInputPending, g.nativeEnding.montage.Phase,
			g.nativeEnding.montage.PlanIndex, len(g.nativeEnding.montage.Plans))
	}
}

func TestApproximateCampaignMontageRejectsUncompiledCh29ShotPartyBinding(t *testing.T) {
	// The original ch29 handler still ends at the unrecovered 0x2bce5 owner.
	// Screenshot mode must not smuggle its partial LOADCH data into the ending
	// roster merely to make the montage appear reachable.
	_, issues, err := campaign.CompileHandlerBinding(assetPath("assets/cutscenes/bindings/ch29_post.json"))
	if err != nil {
		t.Fatal(err)
	}
	if len(issues) != 1 || issues[0].Source.Addr != "0x25970" || issues[0].Op != "unknown" {
		t.Fatalf("ch29 ending owner diagnostics=%#v", issues)
	}
	g := &Game{shotPath: filepath.Join(t.TempDir(), "ending.png")}
	if err := g.materializeShotPartyFromBinding("assets/cutscenes/bindings/ch29_post.json"); err == nil {
		t.Fatal("uncompiled ch29 post handler was accepted as montage roster provenance")
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

func TestCampaignMontageRequiresExplicitApproximateMode(t *testing.T) {
	g := &Game{nativeEnding: &nativeEndingPreview{
		campaignApproximate: true,
		player: &ending.Player{
			State:   ending.PlaybackBlocked,
			Blocked: &ending.Segment{Op: "native_finale_montage_opaque", Source: "0x2c548"},
		},
	}}
	if err := g.startCampaignNativeMontage(); err == nil {
		t.Fatal("campaign montage started without FD2_APPROXIMATE=1")
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
	if _, err := p.Advance(7500); err != nil || g.nativeEnding.awaitingCampaignFallback() {
		t.Fatalf("unstarted montage fallback err=%v preview=%#v", err, g.nativeEnding)
	}
	if err := g.startCampaignNativeMontage(); err == nil || !g.nativeEnding.awaitingCampaignFallback() {
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
