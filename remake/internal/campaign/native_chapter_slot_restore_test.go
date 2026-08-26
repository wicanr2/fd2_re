package campaign

import (
	"testing"

	"github.com/wicanr2/fd2_re/remake/internal/fdsave"
)

func loadNativeRestoreTestAssets(
	t *testing.T,
) (*NativeCharacterCatalog, *NativeIntermissionGateTable) {
	t.Helper()
	catalog, err := LoadNativeCharacterCatalog(
		"../../assets/data/native_character_catalog.json",
	)
	if err != nil {
		t.Fatal(err)
	}
	table, err := LoadNativeIntermissionGateTable(
		"../../assets/data/native_intermission_gate.json",
	)
	if err != nil {
		t.Fatal(err)
	}
	return catalog, table
}

func nativeRestoreRecord(identity, class byte) fdsave.PersistentRecord {
	var record fdsave.PersistentRecord
	record.Raw[8] = identity
	record.Raw[0x20] = class
	record.Raw[0x21] = 7
	record.Raw[0x3b] = 6
	return record
}

func TestNativeIntermissionGateAssetMatchesOriginalDistribution(t *testing.T) {
	_, table := loadNativeRestoreTestAssets(t)
	for _, index := range []int{22, 23, 24, 27, 28, 29} {
		if table.Entries[index] != 1 {
			t.Fatalf("gate[%d]=%d, want 1", index, table.Entries[index])
		}
	}
	for index, value := range table.Entries {
		if value != 0 && index != 22 && index != 23 && index != 24 &&
			index != 27 && index != 28 && index != 29 {
			t.Fatalf("unexpected gate[%d]=%d", index, value)
		}
	}
}

func TestNativeIntermissionGateRoutesEverySaveableChapterToCampaignNode(
	t *testing.T,
) {
	catalog, table := loadNativeRestoreTestAssets(t)
	graph, err := Load("../../assets/scenarios/campaign_full.json")
	if err != nil {
		t.Fatal(err)
	}
	for rawChapter := 1; rawChapter < len(table.Entries); rawChapter++ {
		snapshot := fdsave.ChapterSlotSnapshot{
			Slot:     0,
			Verified: fdsave.VerifiedMetadata{Chapter: byte(rawChapter)},
		}
		plan, err := BuildNativeChapterSlotRestorePlan(
			snapshot, catalog, table, graph,
		)
		if err != nil {
			t.Fatalf("raw chapter %d: %v", rawChapter, err)
		}
		wantType := "town"
		if table.Entries[rawChapter] != 0 {
			wantType = "preparation"
		}
		if node := graph.Nodes[plan.EntryNode]; node == nil ||
			node.Type != wantType {
			t.Fatalf(
				"raw chapter %d route=%q type=%v, want %s",
				rawChapter, plan.EntryNode, node, wantType,
			)
		}
	}
}

func TestNativeIntermissionGatePreservesLateEffectiveRouteOverride(t *testing.T) {
	catalog, table := loadNativeRestoreTestAssets(t)
	graph, err := Load("../../assets/scenarios/campaign_full.json")
	if err != nil {
		t.Fatal(err)
	}
	snapshot := fdsave.ChapterSlotSnapshot{
		Slot:     0,
		Verified: fdsave.VerifiedMetadata{Chapter: 28},
	}
	plan, err := BuildNativeChapterSlotRestorePlan(snapshot, catalog, table, graph)
	if err != nil {
		t.Fatal(err)
	}
	if plan.NativeChapterIndex != 28 || plan.DisplayChapter != 29 ||
		plan.EntryNode != "preparation_ch30" {
		t.Fatalf("raw chapter 28 effective route=%#v", plan)
	}
}

func TestBuildNativeChapterSlotRestorePlanRoutesAfterPostbattleGates(t *testing.T) {
	catalog, table := loadNativeRestoreTestAssets(t)
	graph := &Campaign{Nodes: map[string]*Node{
		"town_ch02":        {Type: "town"},
		"town_ch22":        {Type: "town"},
		"preparation_ch23": {Type: "preparation"},
		"preparation_ch28": {Type: "preparation"},
	}}
	for _, test := range []struct {
		raw  byte
		node string
	}{
		{raw: 1, node: "town_ch02"},
		{raw: 21, node: "town_ch22"},
		{raw: 22, node: "preparation_ch23"},
		{raw: 27, node: "preparation_ch28"},
	} {
		snapshot := fdsave.ChapterSlotSnapshot{
			Slot: 2,
			Verified: fdsave.VerifiedMetadata{
				Chapter: test.raw, RosterCount: 2, Currency: 3456,
				HUDGateA: 1, Raw53AF9: 2, Raw51E61: 3, Raw51E62: 4,
			},
		}
		snapshot.Records[0] = nativeRestoreRecord(0, 1)
		snapshot.Records[1] = nativeRestoreRecord(9, 5)
		plan, err := BuildNativeChapterSlotRestorePlan(
			snapshot, catalog, table, graph,
		)
		if err != nil {
			t.Fatalf("raw chapter %d: %v", test.raw, err)
		}
		if plan.EntryNode != test.node ||
			plan.DisplayChapter != int(test.raw)+1 ||
			plan.NativeChapterIndex != int(test.raw) ||
			plan.Currency != 3456 || plan.HUDGateA != 1 ||
			plan.Raw53AF9 != 2 || plan.Raw51E61 != 3 ||
			plan.Raw51E62 != 4 ||
			len(plan.PartyJoinOrder) != 2 ||
			plan.PartyJoinOrder[0] != 0 ||
			plan.PartyJoinOrder[1] != 9 ||
			plan.PartyRoster[9].Name != "悠妮" {
			t.Fatalf("raw chapter %d plan=%#v", test.raw, plan)
		}
	}
}

func TestBuildNativeChapterSlotRestorePlanFailsClosed(t *testing.T) {
	catalog, table := loadNativeRestoreTestAssets(t)
	graph := &Campaign{Nodes: map[string]*Node{
		"town_ch02": {Type: "town"},
	}}
	snapshot := fdsave.ChapterSlotSnapshot{
		Slot:     0,
		Verified: fdsave.VerifiedMetadata{Chapter: 1, RosterCount: 2},
	}
	snapshot.Records[0] = nativeRestoreRecord(9, 5)
	snapshot.Records[1] = nativeRestoreRecord(9, 5)
	if _, err := BuildNativeChapterSlotRestorePlan(
		snapshot, catalog, table, graph,
	); err == nil {
		t.Fatal("duplicate native identity unexpectedly accepted")
	}
	snapshot.Verified.RosterCount = 1
	snapshot.Verified.Chapter = 0
	if _, err := BuildNativeChapterSlotRestorePlan(
		snapshot, catalog, table, graph,
	); err == nil {
		t.Fatal("missing town_ch01 unexpectedly accepted")
	}
	snapshot.Verified.Chapter = 22
	if _, err := BuildNativeChapterSlotRestorePlan(
		snapshot, catalog, table, graph,
	); err == nil {
		t.Fatal("missing preparation_ch23 unexpectedly accepted")
	}
	snapshot.Slot = fdsave.SlotCount
	snapshot.Verified.Chapter = 1
	if _, err := BuildNativeChapterSlotRestorePlan(
		snapshot, catalog, table, graph,
	); err == nil {
		t.Fatal("out-of-range slot unexpectedly accepted")
	}
	snapshot.Slot = 0
	snapshot.Verified.RosterCount = fdsave.RosterUnits + 1
	if _, err := BuildNativeChapterSlotRestorePlan(
		snapshot, catalog, table, graph,
	); err == nil {
		t.Fatal("oversized roster count unexpectedly accepted")
	}
}
