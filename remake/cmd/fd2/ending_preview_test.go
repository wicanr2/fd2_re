package main

import (
	"bytes"
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

func TestSourceBoundCampaignTailHoldsRecoveredTerminalFrame(t *testing.T) {
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
	t.Setenv("FD2_MUTE", "1")
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
	if !p.ResumeBlockedDialogue() {
		t.Fatal("first ending dialogue gate was unavailable")
	}
	if _, err := p.Advance(5000); err != nil || !p.ResumeBlockedDialogue() {
		t.Fatalf("second ending dialogue gate err=%v blocked=%#v", err, p.Blocked)
	}
	preview.campaignSourceBound = true
	if _, err := p.Advance(7500); err != nil || !preview.atNativeMontageGate() {
		t.Fatalf("montage gate err=%v preview=%#v", err, preview)
	}
	preview.montage = &ending.MontageCycle{Phase: ending.MontagePhaseCompleted}
	g := &Game{nativeEnding: preview}
	if err := g.startCampaignNativeTail(); err != nil {
		t.Fatal(err)
	}
	if !g.nativeEnding.runningCampaignTail() || g.nativeEnding.presentingCampaignTerminal() {
		t.Fatalf("tail did not start in its 20-entry visual schedule: %#v", g.nativeEnding.tailPlayer)
	}
	now := time.Unix(1, 0)
	if err := preview.advance(now, &g.nativeRNGState); err != nil {
		t.Fatal(err)
	}
	if err := preview.advance(now.Add(30*time.Minute), &g.nativeRNGState); err != nil {
		t.Fatal(err)
	}
	if !g.nativeEnding.presentingCampaignTerminal() || g.nativeEnding.awaitingCampaignFallback() {
		t.Fatalf("terminal state=%#v", g.nativeEnding)
	}
	before := append([]byte(nil), p.Compositor.VGA...)
	if err := preview.advance(time.Unix(1, 0), &g.nativeRNGState); err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(before, p.Compositor.VGA) {
		t.Fatal("terminal frame advanced after it was presented")
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

func TestSourceBoundCampaignEndingConsumesOnlyVerifiedGateCueThenReturnsToEditableEpilogue(t *testing.T) {
	p := &ending.Player{
		Timeline: ending.Timeline{AudioCues: []ending.AudioCue{{
			Source: "0x2c5cf", Track: 4, DriverArg: 0, AfterGate: "0x2c548", Trigger: "verified gate",
		}}},
		State:   ending.PlaybackBlocked,
		Blocked: &ending.Segment{Op: "native_finale_montage_opaque", Source: "0x2c548"},
	}
	g := &Game{nativeEnding: &nativeEndingPreview{
		player: p, campaignSourceBound: true, montageStartAttempted: true,
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

func TestSourceBoundCampaignMontageStartsFromPersistentLoadCHOrder(t *testing.T) {
	const base = "../../../org_game/炎龍騎士團/FLAME2"
	for _, name := range []string{"FDOTHER.DAT", "FDTXT.DAT", "ANI.DAT", "TAI.DAT", "FIGANI.DAT", "DATO.DAT"} {
		if _, err := os.Stat(filepath.Join(base, name)); err != nil {
			t.Skip("player-provided ending resources are unavailable")
		}
	}
	t.Setenv("FD2_FDOTHER", filepath.Join(base, "FDOTHER.DAT"))
	t.Setenv("FD2_FDTXT", filepath.Join(base, "FDTXT.DAT"))
	t.Setenv("FD2_ANI", filepath.Join(base, "ANI.DAT"))
	t.Setenv("FD2_MUTE", "1")
	preview, err := newNativeEndingPreview()
	if err != nil {
		t.Fatal(err)
	}
	preview.campaignSourceBound = true
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
		nativeEnding:   preview,
		partyMembers:   map[int]bool{0: true, 1: true, 2: true},
		partyJoinOrder: []int{0, 1, 2},
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

	t.Run("optional party outcome review loops and restores terminal", func(t *testing.T) {
		// The post-montage terminal is the default.  The repeat below is a
		// deliberately separate remake extension: it can replay every admitted
		// party outcome, automatically starts a new pass after completion, and
		// always allows the player to return to the source-derived terminal frame.
		g.nativeEnding.montage.Phase = ending.MontagePhaseCompleted
		if err := g.startCampaignNativeTail(); err != nil {
			t.Fatal(err)
		}
		for steps := 0; !g.nativeEnding.tailPlayer.Ready() && steps < 4096; steps++ {
			if err := g.nativeEnding.tailPlayer.Step(); err != nil {
				t.Fatalf("tail step %d: %v", steps, err)
			}
		}
		if !g.nativeEnding.tailPlayer.Ready() {
			t.Fatal("tail did not reach the terminal frame")
		}
		held := append([]byte(nil), g.nativeEnding.player.Compositor.VGA...)
		if err := g.startCampaignPartyOutcomeReview(); err != nil {
			t.Fatal(err)
		}
		if !g.nativeEnding.reviewingCampaignPartyOutcomes() || g.nativeEnding.montage.Ready() ||
			g.nativeEnding.reviewCycles != 1 {
			t.Fatalf("party outcome review did not begin: %#v", g.nativeEnding)
		}
		now = now.Add(approximateNativeMontageTick)
		if err := g.nativeEnding.advance(now, &g.nativeRNGState); err != nil {
			t.Fatal(err)
		}
		if g.nativeEnding.montage.Phase != ending.MontagePhaseSecondary {
			t.Fatalf("party outcome review did not restart indexed cycle: %s", g.nativeEnding.montage.Phase)
		}
		g.nativeEnding.montage.Phase = ending.MontagePhaseCompleted
		now = now.Add(approximateNativeMontageTick)
		if err := g.nativeEnding.advance(now, &g.nativeRNGState); err != nil {
			t.Fatal(err)
		}
		if !g.nativeEnding.reviewingCampaignPartyOutcomes() || g.nativeEnding.montage.Ready() ||
			g.nativeEnding.reviewCycles != 2 || !bytes.Equal(held, g.nativeEnding.player.Compositor.VGA) {
			t.Fatalf("party outcome review did not loop through terminal: review=%v phase=%s cycles=%d",
				g.nativeEnding.reviewingCampaignPartyOutcomes(), g.nativeEnding.montage.Phase, g.nativeEnding.reviewCycles)
		}
		if err := g.returnCampaignTerminalFromReview(); err != nil {
			t.Fatal(err)
		}
		if !g.nativeEnding.presentingCampaignTerminal() || g.nativeEnding.reviewingCampaignPartyOutcomes() ||
			!bytes.Equal(held, g.nativeEnding.player.Compositor.VGA) {
			t.Fatal("party outcome review did not restore the terminal frame")
		}
	})
}

func TestFinalBattleWinFeedsSynchronizedPartyToEndingMontage(t *testing.T) {
	const base = "../../../org_game/炎龍騎士團/FLAME2"
	for _, name := range []string{"FDOTHER.DAT", "FDTXT.DAT", "ANI.DAT", "TAI.DAT", "FIGANI.DAT", "DATO.DAT"} {
		if _, err := os.Stat(filepath.Join(base, name)); err != nil {
			t.Skip("player-provided ending resources are unavailable")
		}
	}
	t.Setenv("FD2_FDOTHER", filepath.Join(base, "FDOTHER.DAT"))
	t.Setenv("FD2_FDTXT", filepath.Join(base, "FDTXT.DAT"))
	t.Setenv("FD2_ANI", filepath.Join(base, "ANI.DAT"))
	t.Setenv("FD2_MUTE", "1")

	campaignData, err := campaign.Load("../../assets/scenarios/campaign_full.json")
	if err != nil {
		t.Fatal(err)
	}
	runner := campaign.NewRunner(campaignData)
	runner.Cur = "battle_ch30"
	unit := func(side, group byte, level int) battle.Unit {
		return battle.Unit{
			Fig: int(group), BattleFig: int(group), HasBattleFig: true,
			Camp: battle.Own, OnField: true, Lv: level, HP: 8, MaxHP: 42,
			NativeIdentity: int(group), HasNativeIdentity: true,
			NativeRecordByte6: side, HasNativeRecordByte6: true,
			NativeRecordByte8: group, HasNativeRecordByte8: true,
			NativeRecordClass: 2, HasNativeRecordClass: true,
		}
	}
	old0, old1 := unit(2, 4, 39), unit(0, 5, 38)
	final0, final1 := old0, old1
	final0.Lv, final0.Exp, final0.MaxHP = 40, 88.5, 47
	final1.Lv, final1.Exp, final1.MaxHP = 39, 44.25, 46
	g := &Game{
		camp: runner, result: "win",
		st:           &battle.State{Units: []*battle.Unit{&final0, &final1}},
		partyMembers: map[int]bool{0: true, 1: true}, partyJoinOrder: []int{0, 1},
		partyRoster: map[int]battle.Unit{0: old0, 1: old1},
	}
	if !g.confirmBattleResult() || g.camp.NodeID() != "ending" || g.nativeEnding == nil {
		t.Fatalf("最終戰結果未進入原資源結局前綴: node=%q preview=%v err=%q", g.camp.NodeID(), g.nativeEnding != nil, g.loadErr)
	}
	if g.partyRoster[0].Lv != 40 || g.partyRoster[1].Lv != 39 {
		t.Fatalf("最終戰結果未同步到結局隊伍: %#v", g.partyRoster)
	}
	for _, elapsed := range []int{0, 1000, 2500, 0, 256, 2000} {
		if _, err := g.nativeEnding.player.Advance(elapsed); err != nil {
			t.Fatal(err)
		}
	}
	if !g.nativeEnding.player.ResumeBlockedDialogue() {
		t.Fatal("第一個原版文字閘門不可用")
	}
	if _, err := g.nativeEnding.player.Advance(5000); err != nil || !g.nativeEnding.player.ResumeBlockedDialogue() {
		t.Fatalf("第二個原版文字閘門錯誤: %v", err)
	}
	if _, err := g.nativeEnding.player.Advance(7500); err != nil || !g.nativeEnding.atNativeMontageGate() {
		t.Fatalf("未抵達角色蒙太奇閘門: err=%v preview=%#v", err, g.nativeEnding)
	}
	if err := g.startCampaignNativeMontage(); err != nil {
		t.Fatal(err)
	}
	if got := g.nativeEnding.montage; got == nil || len(got.Units) != 2 ||
		got.Units[0][7] != 4 || got.Units[1][7] != 5 || got.Units[0][8] != 4 || got.Units[1][8] != 5 {
		t.Fatalf("結局蒙太奇未消費同步後的持續隊伍: %#v", got)
	}
}

func TestCampaignMontageRejectsUncompiledCh29ShotPartyBinding(t *testing.T) {
	// 截圖專用 ch29_post.json binding 的0x25970→0x2bce5 campaign owner仍未
	// 證實；不可把它的部分LOADCH資料偷渡成角色蒙太奇 provenance。
	_, issues, err := campaign.CompileHandlerBinding(assetPath("assets/cutscenes/bindings/ch29_post.json"))
	if err != nil {
		t.Fatal(err)
	}
	if len(issues) != 1 || issues[0].Source.Addr != "0x25970" ||
		issues[0].Source.Target != "0x2bce5" || issues[0].Op != "unresolved_native_call" {
		t.Fatalf("ch29 ending owner diagnostics=%#v", issues)
	}
	g := &Game{shotPath: filepath.Join(t.TempDir(), "ending.png")}
	if err := g.materializeShotPartyFromBinding("assets/cutscenes/bindings/ch29_post.json"); err == nil {
		t.Fatal("uncompiled ch29 post handler was accepted as montage roster provenance")
	}
}

func TestMissingSkyKeyEndingStartsChapter26NativeBranch(t *testing.T) {
	const base = "../../../org_game/炎龍騎士團/FLAME2"
	for _, name := range []string{"FDOTHER.DAT", "FDTXT.DAT", "ANI.DAT"} {
		if _, err := os.Stat(filepath.Join(base, name)); err != nil {
			t.Skip("player-provided ending resources are unavailable")
		}
	}
	t.Setenv("FD2_FDOTHER", filepath.Join(base, "FDOTHER.DAT"))
	t.Setenv("FD2_FDTXT", filepath.Join(base, "FDTXT.DAT"))
	t.Setenv("FD2_ANI", filepath.Join(base, "ANI.DAT"))
	t.Setenv("FD2_MUTE", "1")

	campaignData, err := campaign.Load(assetPath("assets/scenarios/campaign_full.json"))
	if err != nil {
		t.Fatal(err)
	}
	runner := campaign.NewRunner(campaignData)
	runner.Cur = "ending_ch27_no_sky_key"
	if _, err := newNativeEndingPreviewForCampaign(runner.Node().NativeEndingPrefix); err != nil {
		t.Fatalf("chapter26原生終局前置驗證失敗: %v", err)
	}
	g := &Game{camp: runner}
	g.enterNode()
	if g.nativeEnding == nil || !g.nativeEnding.campaignSourceBound || g.nativeEnding.chapter != 26 {
		t.Fatalf("缺少天空之鑰未進入chapter26原生終局: preview=%#v err=%q notice=%q node=%#v", g.nativeEnding, g.loadErr, g.endingNotice, g.camp.Node())
	}
	for _, elapsed := range []int{0, 1000, 2500, 0, 256, 2000} {
		if _, err := g.nativeEnding.player.Advance(elapsed); err != nil {
			t.Fatal(err)
		}
	}
	if err := g.queueNativeEndingDialogue(); err != nil {
		t.Fatal(err)
	}
	if len(g.dialog) != 1 || g.dialog[0].Speaker != 4 || g.dialog[0].Text != "看!是..是黃金城!" {
		t.Fatalf("chapter26第一文字閘門=%#v", g.dialog)
	}
	g.dialog = nil
	if !g.resumeNativeEndingDialogue() {
		t.Fatal("chapter26第一文字閘門無法恢復")
	}
	if _, err := g.nativeEnding.player.Advance(5000); err != nil {
		t.Fatal(err)
	}
	if err := g.queueNativeEndingDialogue(); err != nil {
		t.Fatal(err)
	}
	if len(g.dialog) != 3 || g.dialog[2].Speaker != 21 || g.dialog[1].Speaker != 24 || g.dialog[0].Speaker != 26 {
		t.Fatalf("chapter26第二文字閘門=%#v", g.dialog)
	}
}

func TestDirectEndingPreviewCannotUseCampaignFallback(t *testing.T) {
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

func TestCampaignMontageRequiresSourceBoundCampaignOwner(t *testing.T) {
	g := &Game{nativeEnding: &nativeEndingPreview{
		campaignSourceBound: true,
		player: &ending.Player{
			State:   ending.PlaybackBlocked,
			Blocked: &ending.Segment{Op: "native_finale_montage_opaque", Source: "0x2c548"},
		},
	}}
	if err := g.startCampaignNativeMontage(); err == nil {
		t.Fatal("campaign montage started without complete source-bound assets")
	}
}

func TestCampaignEndingRejectsProgrammaticUnprovenPrefix(t *testing.T) {
	if _, err := newNativeEndingPreviewForCampaign(&campaign.NativeEndingPrefixConfig{
		Timeline: nativeEndingTimelinePath, Handler: "0x2c548", Chapter: 29,
		Mode: campaign.NativeEndingPrefixSourceBoundE1,
	}); err == nil {
		t.Fatal("programmatic unproven ending prefix was accepted")
	}
}

func TestSourceBoundCampaignFinalNodeConsumesRecoveredPrefixThenStops(t *testing.T) {
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
	g := &Game{camp: runner}
	g.enterNode()
	if g.nativeEnding == nil || !g.nativeEnding.campaignSourceBound {
		t.Fatalf("source-bound final node did not admit recovered prefix: %#v", g.nativeEnding)
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
		t.Fatalf("source-bound final fallback did not return to editable ending: preview=%#v notice=%q", g.nativeEnding, g.endingNotice)
	}
}
