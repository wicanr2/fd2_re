package main

import (
	"crypto/sha256"
	"encoding/binary"
	"fmt"
	"os"
	"path/filepath"
	"testing"

	"github.com/wicanr2/fd2_re/remake/internal/battle"
	"github.com/wicanr2/fd2_re/remake/internal/campaign"
	"github.com/wicanr2/fd2_re/remake/internal/fdsave"
)

const lateChapterSlotSaveSHA256 = "f46d9c54d3037f84f05d72714569c282e63f39bf125251a9cf5cd9593ff3241f"

func writeNativeRestoreFixture(
	t *testing.T, rawChapter byte,
) string {
	t.Helper()
	plain := make([]byte, fdsave.FileSize)
	for slot := 0; slot < fdsave.SlotCount; slot++ {
		start, _, err := fdsave.SlotBounds(slot)
		if err != nil {
			t.Fatal(err)
		}
		plain[start+fdsave.RosterSize] = 0xff
	}
	start, _, _ := fdsave.SlotBounds(1)
	record := plain[start : start+fdsave.UnitSize]
	record[7], record[8], record[0x20], record[0x21], record[0x3b] =
		0x44, 9, 5, 7, 6
	binary.LittleEndian.PutUint16(record[0x40:], 31)
	binary.LittleEndian.PutUint16(record[0x42:], 40)
	metadata := start + fdsave.RosterSize
	plain[metadata], plain[metadata+1] = rawChapter, 1
	binary.LittleEndian.PutUint32(plain[metadata+2:], 789)
	copy(plain[metadata+6:], []byte{0, 0, 1, 1})
	stored, err := fdsave.Encode(plain)
	if err != nil {
		t.Fatal(err)
	}
	path := filepath.Join(t.TempDir(), "FD2.SAV")
	if err := os.WriteFile(path, stored, 0o644); err != nil {
		t.Fatal(err)
	}
	return path
}

func TestLoadNativeGameFromSlotRestoresTownAndTypedParty(t *testing.T) {
	graph := &campaign.Campaign{
		Start: "town_ch02",
		Flags: map[string]bool{"initial": true},
		Nodes: map[string]*campaign.Node{
			"town_ch02": {Type: "town"},
		},
	}
	g := &Game{
		camp: campaign.NewRunner(graph),
		gold: 1, items: []string{"stale"},
		partyMembers: map[int]bool{4: true},
	}
	if err := g.loadNativeGameFromSlot(
		writeNativeRestoreFixture(t, 1), 1,
	); err != nil {
		t.Fatal(err)
	}
	yuni, ok := g.partyRoster[9]
	if g.camp.NodeID() != "town_ch02" || g.gold != 789 ||
		len(g.items) != 0 || !ok || yuni.Name != "悠妮" ||
		yuni.HP != 31 || yuni.MaxHP != 40 ||
		!yuni.HasMapSelectorKey || yuni.MapSelectorKey != 0x44 ||
		len(g.partyJoinOrder) != 1 || g.partyJoinOrder[0] != 9 ||
		!g.partyMembers[9] || len(g.partyDeploy) != 0 ||
		g.handlerChapter != 1 || g.nativeChapterRestore == nil ||
		g.nativeChapterRestore.Raw51E62 != 1 ||
		g.currentNativeSystemOptions().Raw51E61 != 1 ||
		!g.nativeMapHUDPersistent.HasDisplayGateA ||
		g.nativeMapHUDPersistent.DisplayGateA != 0 ||
		!g.nativeMapHUDPersistent.HasAnchorX || g.nativeMapHUDPersistent.AnchorX != 1 {
		t.Fatalf("native restore state=%#v game=%#v", g.nativeChapterRestore, g)
	}
}

func TestLoadNativeGameFromSlotRestoresDirectPreparation(t *testing.T) {
	graph := &campaign.Campaign{
		Start: "preparation_ch23",
		Nodes: map[string]*campaign.Node{
			"preparation_ch23": {Type: "preparation"},
		},
	}
	g := &Game{camp: campaign.NewRunner(graph)}
	if err := g.loadNativeGameFromSlot(
		writeNativeRestoreFixture(t, 22), 1,
	); err != nil {
		t.Fatal(err)
	}
	if g.camp.NodeID() != "preparation_ch23" ||
		len(g.prepIDs) != 0 || !g.battlePartyMembers()[9] ||
		!g.partyRoster[9].HasMapSelectorKey ||
		g.partyRoster[9].MapSelectorKey != 0x44 ||
		g.prepPromptSource == nil {
		t.Fatalf(
			"direct preparation restore node=%q ids=%v unit=%#v prompt=%v",
			g.camp.NodeID(), g.prepIDs, g.partyRoster[9],
			g.prepPromptSource != nil,
		)
	}
}

func TestLoadNativeGameFromSlotDoesNotPartiallyApplyInvalidRoute(t *testing.T) {
	graph := &campaign.Campaign{
		Start: "safe",
		Nodes: map[string]*campaign.Node{
			"safe": {Type: "town"},
		},
	}
	g := &Game{
		camp: campaign.NewRunner(graph),
		gold: 99, items: []string{"keep"},
		partyMembers:   map[int]bool{4: true},
		handlerChapter: 8,
	}
	if err := g.loadNativeGameFromSlot(
		writeNativeRestoreFixture(t, 0), 1,
	); err == nil {
		t.Fatal("missing native route unexpectedly accepted")
	}
	if g.camp.NodeID() != "safe" || g.gold != 99 ||
		len(g.items) != 1 || !g.partyMembers[4] ||
		g.handlerChapter != 8 || g.nativeChapterRestore != nil {
		t.Fatalf("failed restore partially mutated game: %#v", g)
	}
}

func TestConfirmTitleLoadSlotRestoresVerifiedNativeIntermission(t *testing.T) {
	graph := &campaign.Campaign{
		Start: "town_ch02",
		Nodes: map[string]*campaign.Node{
			"town_ch02": {Type: "town"},
		},
	}
	g := &Game{
		camp: campaign.NewRunner(graph), titlePhase: "loadslots", titleSlotSel: 1,
		gold: 1, items: []string{"stale"},
		partyMembers: map[int]bool{4: true}, handlerChapter: 8,
		st: &battle.State{W: 1, H: 1}, sel: &battle.Unit{Fig: 99},
	}
	path := writeNativeRestoreFixture(t, 1)
	t.Setenv("FD2_NATIVE_SAVE", path)
	if !g.confirmTitleLoadSlot(1) {
		t.Fatalf("verified title LOAD rejected: %s", g.msg)
	}
	yuni, ok := g.partyRoster[9]
	if g.titlePhase != "" || g.camp.NodeID() != "town_ch02" ||
		g.gold != 789 || !ok || yuni.Name != "悠妮" ||
		len(g.partyJoinOrder) != 1 || g.partyJoinOrder[0] != 9 ||
		g.handlerChapter != 1 || g.st != nil || g.sel != nil ||
		g.nativeChapterRestore == nil {
		t.Fatalf("title LOAD did not publish complete intermission restore: %#v", g)
	}
}

func TestConfirmTitleLoadSlotRejectsTamperedNativeEnvelopeAtomically(t *testing.T) {
	path := writeNativeRestoreFixture(t, 1)
	stored, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	stored[0x123] ^= 1
	if err := os.WriteFile(path, stored, 0o644); err != nil {
		t.Fatal(err)
	}
	t.Setenv("FD2_NATIVE_SAVE", path)
	graph := &campaign.Campaign{
		Start: "safe",
		Nodes: map[string]*campaign.Node{"safe": {Type: "town"}},
	}
	state := &battle.State{W: 3, H: 4}
	g := &Game{
		camp: campaign.NewRunner(graph), titlePhase: "loadslots", titleSlotSel: 1,
		gold: 99, items: []string{"keep"},
		partyMembers: map[int]bool{4: true}, handlerChapter: 8, st: state,
	}
	if g.confirmTitleLoadSlot(1) {
		t.Fatal("tampered title LOAD unexpectedly succeeded")
	}
	if g.titlePhase != "loadslots" || g.camp.NodeID() != "safe" ||
		g.gold != 99 || len(g.items) != 1 || !g.partyMembers[4] ||
		g.handlerChapter != 8 || g.st != state || g.nativeChapterRestore != nil ||
		g.msg != "無法讀取原版存檔槽" {
		t.Fatalf("tampered title LOAD leaked state: %#v", g)
	}
}

func TestConfirmTitleLoadSlotRestoresExternalLateSlotToFinalPreparation(t *testing.T) {
	path := os.Getenv("FD2_LATE_CHAPTER_SLOT_SAVE")
	if path == "" {
		t.Skip("未提供固定雜湊的外部晚期章節槽存檔")
	}
	stored, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if got := fmt.Sprintf("%x", sha256.Sum256(stored)); got != lateChapterSlotSaveSHA256 {
		t.Fatalf("外部晚期章節槽存檔 SHA-256=%s，不是已審查來源", got)
	}
	graph, err := campaign.Load(assetPath("assets/scenarios/campaign_full.json"))
	if err != nil {
		t.Fatal(err)
	}
	g := &Game{
		camp: campaign.NewRunner(graph), titlePhase: "loadslots", titleSlotSel: 0,
		gold: 1, items: []string{"stale"},
		partyMembers: map[int]bool{4: true}, handlerChapter: 8,
		st: &battle.State{W: 1, H: 1}, sel: &battle.Unit{Fig: 99},
	}
	t.Setenv("FD2_NATIVE_SAVE", path)
	if !g.confirmTitleLoadSlot(0) {
		t.Fatalf("已驗證晚期 slot 0 被正式 LOAD 拒絕：%s", g.msg)
	}
	if g.titlePhase != "" || g.camp.NodeID() != "preparation_ch30" ||
		g.gold != 60 || len(g.partyMembers) != 29 || len(g.partyJoinOrder) != 29 ||
		len(g.partyRoster) != 29 || len(g.partyDeploy) != 0 ||
		g.handlerChapter != 0x1c || g.st != nil || g.sel != nil ||
		g.nativeChapterRestore == nil || g.nativeChapterRestore.Raw51E62 != 1 ||
		g.currentNativeSystemOptions().Raw51E61 != 1 ||
		!g.nativeMapHUDPersistent.HasDisplayGateA ||
		g.nativeMapHUDPersistent.DisplayGateA != 1 {
		t.Fatalf("晚期 slot 0 未完整發布最終戰整備：node=%q gold=%d members=%d order=%d roster=%d deploy=%d chapter=%d state=%p selector=%p restore=%#v",
			g.camp.NodeID(), g.gold, len(g.partyMembers), len(g.partyJoinOrder),
			len(g.partyRoster), len(g.partyDeploy), g.handlerChapter, g.st, g.sel,
			g.nativeChapterRestore)
	}
}
