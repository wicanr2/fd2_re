package main

import (
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"reflect"
	"testing"

	"github.com/wicanr2/fd2_re/remake/internal/battle"
)

// Controlled chapter-entry source only. JOIN and AwardExpNative own the
// identity, growth and command mask; the test never grants a command directly.
func TestCommand30NormalGUISource(t *testing.T) {
	out, base := os.Getenv("FD2_COMMAND30_GUI_SOURCE_OUT"), os.Getenv("FD2_COMMAND30_GUI_SOURCE_BASE")
	if out == "" || base == "" {
		t.Skip("explicit controlled GUI source export")
	}
	if _, err := os.Stat(out); !os.IsNotExist(err) {
		t.Fatal("fresh output required")
	}
	raw, err := os.ReadFile(base)
	if err != nil {
		t.Fatal(err)
	}
	baseSHA := fmt.Sprintf("%x", sha256.Sum256(raw))
	if baseSHA != "5d64f3a0bf9aa56f7e77f51e8cfe45fe8c14ab78f2703b966fbc0e142e7bd871" {
		t.Fatal("unreviewed base source")
	}
	var source saveData
	if err := json.Unmarshal(raw, &source); err != nil {
		t.Fatal(err)
	}
	if source.Node != "battle_ch23" || source.PartyMembers[18] || source.PartyRoster[18].HasNativeIdentity {
		t.Fatal("unexpected source party")
	}
	if err := validateSavePartyTopology(source); err != nil {
		t.Fatal(err)
	}
	st, err := battle.Load(assetPath("assets/maps/map16/map16_units.json"))
	if err != nil {
		t.Fatal(err)
	}
	g := &Game{st: st}
	if err := g.bindNativeFutureItemRows(st); err != nil {
		t.Fatal(err)
	}
	st.NativeGrowthRows, err = battle.LoadNativeGrowthSlots(assetPath("assets/data/native_growth_slots.json"))
	if err != nil {
		t.Fatal(err)
	}
	st.CommandLearn, err = battle.LoadCommandLearn(assetPath("assets/data/command_learn.json"))
	if err != nil {
		t.Fatal(err)
	}
	st.CommandLearnSelectors, err = battle.LoadCommandLearnSelectors(assetPath("assets/data/native_growth_slots.json"))
	if err != nil {
		t.Fatal(err)
	}
	st.NativeCommandBook, err = battle.LoadNativeCommandRecords(assetPath("assets/spells.json"))
	if err != nil {
		t.Fatal(err)
	}
	baseUnit := st.Units[36]
	if baseUnit.BattleFig != 18 || baseUnit.Lv != 1 || len(baseUnit.NativeCommandIDs()) != 0 {
		t.Fatal("fixed original actor differs")
	}
	joined, err := g.materializeNativeJoinPersistentUnit(18, *baseUnit)
	if err != nil {
		t.Fatal(err)
	}
	st.Units[36] = &joined
	before := cloneNativeShopUnit(joined)
	granted, events, seed := st.AwardExpNative(&joined, 300, 7)
	if granted != 300 || joined.Lv != 4 || len(events) != 3 || len(joined.NativeCommandIDs()) != 1 || joined.NativeCommandIDs()[0] != 30 {
		t.Fatal("formal growth did not grant command30")
	}
	entities, err := loadOfficialLocaleEntities("zh-Hant")
	if err != nil {
		t.Fatal(err)
	}
	joined.Name, err = entities.CharacterName(18)
	if err != nil {
		t.Fatal(err)
	}
	g.partyRoster = map[int]battle.Unit{18: joined}
	g.resetPersistentRosterState()
	joined = g.partyRoster[18]
	learned := cloneNativeShopUnit(joined)
	joined.AP, joined.BaseAP = joined.AP+5000, joined.BaseAP+5000
	joined.DP, joined.BaseDP = joined.DP+300, joined.BaseDP+300
	joined.DX, joined.HIT, joined.EV = joined.DX+60, joined.HIT+60, joined.EV+60
	joined.BaseHIT, joined.BaseEV = joined.BaseHIT+60, joined.BaseEV+60
	joined.HP, joined.MaxHP = joined.HP+1000, joined.MaxHP+1000
	if !joined.HasNativeIdentity || joined.NativeIdentity != 18 || joined.MP < 24 || joined.NativeCommandMask != learned.NativeCommandMask {
		t.Fatal("invalid controlled source")
	}
	source.PartyMembers[18], source.PartyDeploy[18] = true, true
	source.PartyJoinOrder = append(source.PartyJoinOrder, 18)
	source.PartyRoster[18] = joined
	if err := validateSavePartyTopology(source); err != nil {
		t.Fatal(err)
	}
	if err := os.Mkdir(out, 0755); err != nil {
		t.Fatal(err)
	}
	write := func(name string, value any) []byte {
		t.Helper()
		b, err := json.MarshalIndent(value, "", "  ")
		if err != nil {
			t.Fatal(err)
		}
		b = append(b, '\n')
		if err := os.WriteFile(filepath.Join(out, name), b, 0644); err != nil {
			t.Fatal(err)
		}
		return b
	}
	output := write("fd2_save.json", source)
	var oldJSON, newJSON map[string]any
	if err := json.Unmarshal(raw, &oldJSON); err != nil {
		t.Fatal(err)
	}
	if err := json.Unmarshal(output, &newJSON); err != nil {
		t.Fatal(err)
	}
	for key, value := range oldJSON {
		if key != "party_roster" && key != "party_members" && key != "party_deploy" && key != "party_join_order" && !reflect.DeepEqual(value, newJSON[key]) {
			t.Fatal("source changed unrelated chapter field", key)
		}
	}
	for id, value := range oldJSON["party_roster"].(map[string]any) {
		if !reflect.DeepEqual(value, newJSON["party_roster"].(map[string]any)[id]) {
			t.Fatal("source changed another character", id)
		}
	}
	write("manifest.json", map[string]any{"status": "PASS-CONTROLLED-COMMAND30-GUI-SOURCE", "base_sha256": baseSHA, "output_sha256": fmt.Sprintf("%x", sha256.Sum256(output)), "node": source.Node, "actor_identity": 18, "original_map": 16, "original_unit_slot": 36, "before_joined_growth": before, "after_growth_and_rest": learned, "after_policy": joined, "growth_events": events, "growth_exp": 300, "growth_seed_before": 7, "growth_seed_after": seed, "policy": map[string]int{"base_ap": 5000, "base_dp": 300, "base_dx": 60, "max_hp": 1000}, "party_membership_added": 18, "all_other_roster_rows_preserved": true, "chapter_state_injection": false, "direct_command_grant": false, "normal_recruitment_proven": false, "scope": "原JOIN與原生升級writer建構的章前組隊來源；非自然招募或完整章節。正常鍵盤GUI另驗。"})
}
