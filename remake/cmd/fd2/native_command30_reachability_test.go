package main

import (
	"context"
	"encoding/json"
	"os"
	"os/exec"
	"testing"
	"time"

	"github.com/wicanr2/fd2_re/remake/internal/battle"
	"github.com/wicanr2/fd2_re/remake/internal/campaign"
	"github.com/wicanr2/fd2_re/remake/internal/fdsave"
)

// The grant comes from the real map16 constructor and level-up table, never
// from EnableNativeCommand in a fixture. Experience is a controlled stimulus.
func TestNativeCommand30MitiAcquisitionAndPlayerEntry(t *testing.T) {
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
	if len(st.Units) <= 36 {
		t.Fatal("missing fixed map16 roster")
	}
	miti := st.Units[36]
	if miti.BattleFig != 18 || !miti.HasBattleFig || miti.Camp != battle.Own || miti.NativeRecordByte6 != 2 || miti.Lv != 1 || len(miti.NativeCommandIDs()) != 0 {
		t.Fatalf("fixed constructor differs: %+v", miti)
	}
	before := *miti
	got, events, afterSeed := st.AwardExpNative(miti, 300, 7)
	if got != 300 || len(events) != 3 || miti.Lv != 4 || len(events[0].LearnedCommandIDs) != 1 || events[0].LearnedCommandIDs[0] != 30 || len(miti.NativeCommandIDs()) != 1 || miti.NativeCommandIDs()[0] != 30 || miti.MaxMP < st.NativeCommandBook[30].MPCost {
		t.Fatalf("real selector18->learn5 grant did not occur: lv=%d cmds=%v mp=%d events=%+v", miti.Lv, miti.NativeCommandIDs(), miti.MaxMP, events)
	}
	supported := g.nativeCommandTargetSupported(30)
	if out := os.Getenv("FD2_COMMAND30_ACQUISITION_OUT"); out != "" {
		if _, err := os.Stat(out); !os.IsNotExist(err) {
			t.Fatal("do not overwrite acquisition receipt")
		}
		raw, err := json.MarshalIndent(map[string]any{"status": "RECORDED-CONTROLLED-MITI-COMMAND30-GRANT", "source_roster": "remake/assets/maps/map16/map16_units.json", "slot": 36, "selector": 18, "learn_row": 5, "before_level": before.Lv, "before_commands": before.NativeCommandIDs(), "after_level": miti.Lv, "after_commands": miti.NativeCommandIDs(), "max_mp": miti.MaxMP, "command_mp_cost": st.NativeCommandBook[30].MPCost, "native_seed_before": 7, "native_seed_after": afterSeed, "seed_setup": "fixed before single run; no reroll", "experience_stimulus": 300, "events": events, "player_target_supported": supported, "normal_campaign_verified": false, "scope": "real roster/constructor, production native growth/command writer and actual player admission gate; controlled EXP, not normal chapter proof"}, "", "  ")
		if err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(out, append(raw, '\n'), 0644); err != nil {
			t.Fatal(err)
		}
	}
	if !supported {
		t.Fatal("legitimately learned command30 is rejected by the player target gate")
	}
	assertNativeCommand30LearnedMaskSave(t, miti)
}

// Controlled save boundary, not a normal chapter receipt: retain the native
// save fixture's map/view while copying the legitimately learned Miti state.
func assertNativeCommand30LearnedMaskSave(t *testing.T, miti *battle.Unit) {
	t.Helper()
	g, path, _ := nativeCurrentSaveTestGame(t)
	u := g.st.Units[0]
	u.BattleFig, u.MapSelectorKey = miti.BattleFig, miti.BattleFig
	u.NativeRecordByte8 = miti.NativeRecordByte8
	u.NativeRecordRace, u.NativeRecordClass = miti.NativeRecordRace, miti.NativeRecordClass
	u.Lv, u.HP, u.MaxHP, u.MP, u.MaxMP = miti.Lv, miti.HP, miti.MaxHP, miti.MP, miti.MaxMP
	u.NativeCommandMask = miti.NativeCommandMask
	g.st.NativeRuntimeRecords[0].SelectorKey = byte(miti.BattleFig)
	for _, record := range [][]byte{g.st.NativeRuntimeRecords[0].Raw[:], g.nativeCurrentSavePlain[fdsave.CurrentRuntimeRosterOffset:], g.nativeCurrentSavePlain[fdsave.CurrentPersistentRosterOffset:]} {
		record[7], record[8] = byte(miti.BattleFig), miti.NativeRecordByte8
		record[0x1f], record[0x20] = miti.NativeRecordRace, miti.NativeRecordClass
	}
	_, stored, err := g.buildNativeCurrentSaveStored()
	if err != nil {
		t.Fatal(err)
	}
	if err := replaceNativeCurrentSaveAtomic(path, stored); err != nil {
		t.Fatal(err)
	}
	written, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	plain, err := fdsave.Decode(written)
	if err != nil {
		t.Fatal(err)
	}
	snapshot, err := fdsave.InspectCurrentSnapshot(plain)
	if err != nil || snapshot.RuntimeRecords[0].Raw[0x1d]&0x40 == 0 || snapshot.PersistentRecords[0].Raw[0x1d] != 0 {
		t.Fatalf("current save must publish live runtime mask and preserve persistent baseline: %v", err)
	}
	ctx, cancel := context.WithTimeout(context.Background(), 45*time.Second)
	defer cancel()
	cmd := exec.CommandContext(ctx, os.Args[0], "-test.run=^TestCommand30IsolatedColdRead$", "-test.count=1")
	cmd.Env = append(os.Environ(), "FD2_COMMAND30_COLD_SAVE="+path)
	if out, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("fresh process cold read: %v\n%s", err, out)
	}
}

func TestCommand30IsolatedColdRead(t *testing.T) {
	path := os.Getenv("FD2_COMMAND30_COLD_SAVE")
	if path == "" {
		t.Skip("child process of the controlled Miti acquisition/save test only")
	}
	stored, err := os.ReadFile(path)
	if err != nil || len(stored) != fdsave.FileSize {
		t.Fatalf("cold save input: %v", err)
	}
	plain, err := fdsave.Decode(stored)
	if err != nil {
		t.Fatal(err)
	}
	snapshot, err := fdsave.InspectCurrentSnapshot(plain)
	if err != nil {
		t.Fatal(err)
	}
	input, err := fdsave.BuildContinueRuntimeInput(snapshot, fdsave.ContinueRuntimeContext{Chapter: 0, FieldWidth: 20, FieldHeight: 20, SelectorGroupCount: 256, TitleTimerTick: 7, HasTitleTimerTick: true})
	if err != nil {
		t.Fatal(err)
	}
	catalog, err := campaign.LoadNativeCharacterCatalog(assetPath("assets/data/native_character_catalog.json"))
	if err != nil {
		t.Fatal(err)
	}
	cold := &battle.State{W: 20, H: 20, NativeFieldEventSlots: make([]int, 400), NativeFieldEvents: make([]battle.NativeFieldEvent, 16)}
	if err := campaign.MaterializeNativeContinueFieldBoundary(cold, input, 0); err != nil {
		t.Fatal(err)
	}
	if err := campaign.MaterializeNativeContinueRuntimeUnits(cold, input, catalog); err != nil {
		t.Fatal(err)
	}
	if len(cold.Units) != 1 || cold.Units[0].BattleFig != 18 || cold.Units[0].NativeCommandMask != ([5]byte{0, 0, 0, 0x40, 0}) || cold.Units[0].MaxMP != 29 || cold.Units[0].Lv != 4 {
		t.Fatal("cold native runtime adapter lost Miti identity, learned mask or stats")
	}
}
