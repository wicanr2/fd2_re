package main

import (
	"encoding/binary"
	"path/filepath"
	"reflect"
	"testing"
	"time"

	"github.com/wicanr2/fd2_re/remake/internal/battle"
)

func TestNativeTransientPhaseExpiresAndRecomputesAtomically(t *testing.T) {
	g, _, _ := nativeCurrentSaveTestGame(t)
	unit := g.st.Units[0]
	unit.NativeRecordByte6 = 0
	unit.NativeTransient[0] = 1
	unit.BaseAP = 100
	unit.BaseDP = 60
	unit.DX = 20
	unit.AP = 114
	unit.DP = 60
	unit.HIT = 20
	unit.EV = 20
	g.st.NativeRuntimeRecords[0].Raw[6] = 0
	g.st.NativeRuntimeRecords[0].Raw[0x22] = 1
	binary.LittleEndian.PutUint16(g.st.NativeRuntimeRecords[0].Raw[0x37:], 100)
	binary.LittleEndian.PutUint16(g.st.NativeRuntimeRecords[0].Raw[0x39:], 60)
	binary.LittleEndian.PutUint16(g.st.NativeRuntimeRecords[0].Raw[0x3e:], 20)
	expired, err := g.applyNativeTransientPhase(0)
	if err != nil {
		t.Fatal(err)
	}
	unit = g.st.Units[0]
	if len(expired) != 1 || expired[0].Offset != 0x22 ||
		unit.NativeTransient[0] != 0 || unit.AP != 100 ||
		g.st.NativeRuntimeRecords[0].Raw[0x22] != 0 {
		t.Fatalf("expiry=%+v transient=%v AP=%d raw=%d",
			expired, unit.NativeTransient, unit.AP, g.st.NativeRuntimeRecords[0].Raw[0x22])
	}
}

// 從城鎮正常進戰場沒有 saved runtime raw 投影：名冊每筆都有 raw +5／+6 出處時走 typed
// 掃描（第六章 r4：敵方法師的 +0x22 每回合 −1，歸零時 AP 從 1.15 倍回到基礎值）；
// 缺任何一筆的 raw 出處就整段拒絕、不動任何單位。
func TestNativeTransientPhaseTypedSweepWithoutRawProjection(t *testing.T) {
	g, _, _ := nativeCurrentSaveTestGame(t)
	g.st.HasNativeRuntimeUnitProjection = false
	unit := g.st.Units[0]
	unit.NativeRecordByte6 = 0
	unit.NativeTransient[0] = 1
	unit.BaseAP = 100
	unit.BaseDP = 60
	unit.EquipmentBaseSet = true
	unit.DX = 20
	unit.AP = 114
	unit.DP = 60
	unit.HIT = 20
	unit.EV = 20
	expired, err := g.applyNativeTransientPhase(0)
	if err != nil {
		t.Fatal(err)
	}
	unit = g.st.Units[0]
	if len(expired) != 1 || expired[0].Offset != 0x22 || unit.NativeTransient[0] != 0 || unit.AP != 100 {
		t.Fatalf("expiry=%+v transient=%v AP=%d", expired, unit.NativeTransient, unit.AP)
	}

	g, _, _ = nativeCurrentSaveTestGame(t)
	g.st.HasNativeRuntimeUnitProjection = false
	g.st.Units[0].HasNativeRecordByte5 = false
	g.st.Units[0].NativeTransient[0] = 2
	before := g.st.Units[0].NativeTransient
	if _, err := g.applyNativeTransientPhase(0); err == nil {
		t.Fatal("roster without raw +5 provenance was accepted")
	}
	if g.st.Units[0].NativeTransient != before {
		t.Fatal("failed transient phase mutated live unit")
	}
}

func TestNativeTransientPhasePersistsDecrementWithoutEarlyRecalc(t *testing.T) {
	g, _, _ := nativeCurrentSaveTestGame(t)
	unit := g.st.Units[0]
	unit.NativeRecordByte6 = 0
	unit.NativeTransient[0] = 2
	unit.AP = 114
	g.st.NativeRuntimeRecords[0].Raw[6] = 0
	g.st.NativeRuntimeRecords[0].Raw[0x22] = 2

	expired, err := g.applyNativeTransientPhase(0)
	if err != nil {
		t.Fatal(err)
	}
	unit = g.st.Units[0]
	if len(expired) != 0 || unit.NativeTransient[0] != 1 ||
		g.st.NativeRuntimeRecords[0].Raw[0x22] != 1 || unit.AP != 114 {
		t.Fatalf("expiry=%+v duration=%d raw=%d AP=%d",
			expired, unit.NativeTransient[0], g.st.NativeRuntimeRecords[0].Raw[0x22], unit.AP)
	}
}

func TestNativeTransientPhasesPreserveSelectorOrderAtomically(t *testing.T) {
	g, _, _ := nativeCurrentSaveTestGame(t)
	first := g.st.Units[0]
	first.NativeRecordByte6 = 1
	first.NativeTransient[0] = 1
	g.st.NativeRuntimeRecords[0].Raw[6] = 1
	g.st.NativeRuntimeRecords[0].Raw[0x22] = 1

	secondValue := *first
	second := &secondValue
	second.NativeRecordByte6 = 0
	second.NativeTransient = [6]byte{0, 1}
	second.NativeIdentity = 31
	second.NativeRecordByte8 = 31
	second.NativeMapPresentation.X++
	second.X++
	secondRaw := g.st.NativeRuntimeRecords[0]
	secondRaw.Raw[0]++
	secondRaw.Raw[6] = 0
	secondRaw.Raw[8] = 31
	secondRaw.Raw[0x22], secondRaw.Raw[0x23] = 0, 1
	g.st.Units = append(g.st.Units, second)
	g.st.NativeRuntimeRecords = append(g.st.NativeRuntimeRecords, secondRaw)

	expired, err := g.applyNativeTransientPhases(1, 0)
	if err != nil {
		t.Fatal(err)
	}
	if len(expired) != 2 || expired[0].Unit != g.st.Units[0] || expired[0].Offset != 0x22 ||
		expired[1].Unit != g.st.Units[1] || expired[1].Offset != 0x23 {
		t.Fatalf("ordered expiry=%+v", expired)
	}
}

func TestNativeTransientPhasesRejectDuplicateSelectorAtomically(t *testing.T) {
	g, _, _ := nativeCurrentSaveTestGame(t)
	g.st.Units[0].NativeRecordByte6 = 1
	g.st.Units[0].NativeTransient[0] = 2
	before := *g.st.Units[0]
	if _, err := g.applyNativeTransientPhases(1, 1); err == nil {
		t.Fatal("duplicate selector was accepted")
	}
	if !reflect.DeepEqual(*g.st.Units[0], before) {
		t.Fatal("duplicate selector mutated live unit")
	}
}

func TestNativeTransientStatusDeathsNeverDispatchDeathEffectsOrBorrowNextActor(t *testing.T) {
	g, _, _ := nativeCurrentSaveTestGame(t)
	g.gold = 77
	base := g.st.Units[0]
	base.NativeRecordByte6 = 2
	g.st.NativeRuntimeRecords[0].Raw[6] = 2
	g.st.Units = nil
	g.st.NativeRuntimeRecords = nil
	for kind := 0; kind <= 3; kind++ {
		clone := *base
		clone.HP, clone.MaxHP = 10, 100
		clone.NativeRecordByte5 = 0
		clone.NativeTransient = [6]byte{0, 0, 0, 2}
		clone.NativeRecordDeathEffect = [3]byte{byte(kind), byte(40 + kind), 0}
		clone.HasNativeRecordDeathEffect = true
		record := battle.NativeRuntimeRecordState{}
		record.Raw[5], record.Raw[6], record.Raw[0x25] = 0, 2, 2
		binary.LittleEndian.PutUint16(record.Raw[0x40:0x42], 10)
		binary.LittleEndian.PutUint16(record.Raw[0x42:0x44], 100)
		g.st.Units = append(g.st.Units, &clone)
		g.st.NativeRuntimeRecords = append(g.st.NativeRuntimeRecords, record)
	}

	if _, err := g.applyNativeTransientPhase(2); err != nil {
		t.Fatal(err)
	}
	for slot, unit := range g.st.Units {
		if unit.HP != 0 || unit.NativeRecordByte5 != 1 ||
			g.st.NativeRuntimeRecords[slot].Raw[5] != 1 || unit.NativeTransient[3] != 2 {
			t.Fatalf("slot %d did not finish through sub_1DB65 semantics: HP=%d byte5=%d duration=%d",
				slot, unit.HP, unit.NativeRecordByte5, unit.NativeTransient[3])
		}
	}
	if len(g.pendingDeathPrograms) != 0 || len(g.pendingNativeDeathRewards) != 0 ||
		g.deathProgramKiller != nil || g.gold != 77 {
		t.Fatalf("status death entered action dispatch: programs=%d rewards=%d killer=%v gold=%d",
			len(g.pendingDeathPrograms), len(g.pendingNativeDeathRewards), g.deathProgramKiller, g.gold)
	}

	// 下一位行動者到來前再掃一次也不得重收集或補派；死亡單位已由 +5 bit0 擋住。
	if _, err := g.applyNativeTransientPhase(2); err != nil {
		t.Fatal(err)
	}
	for _, dead := range g.st.Units {
		g.queueNativeDeathProgram(dead, nil)
		g.grantNativeDeathReward(1, 999, nil)
	}
	if len(g.pendingDeathPrograms) != 0 || len(g.pendingNativeDeathRewards) != 0 || g.gold != 77 {
		t.Fatalf("next actor inherited status kills: programs=%d rewards=%d gold=%d",
			len(g.pendingDeathPrograms), len(g.pendingNativeDeathRewards), g.gold)
	}
}

func TestCompleteTurnPlayerPhaseTicksRawSelector2BeforeInput(t *testing.T) {
	g, _, _ := nativeCurrentSaveTestGame(t)
	unit := g.st.Units[0]
	unit.NativeRecordByte6 = 2
	unit.NativeTransient[0] = 2
	g.st.NativeRuntimeRecords[0].Raw[6] = 2
	g.st.NativeRuntimeRecords[0].Raw[0x22] = 2
	g.result = ""

	g.completeTurnPlayerPhase()
	if g.loadErr != "" || g.st.Units[0].NativeTransient[0] != 1 ||
		g.st.NativeRuntimeRecords[0].Raw[0x22] != 1 || g.banner != "PLAYER PHASE" {
		t.Fatalf("err=%q transient=%d raw=%d banner=%q",
			g.loadErr, g.st.Units[0].NativeTransient[0],
			g.st.NativeRuntimeRecords[0].Raw[0x22], g.banner)
	}
}

func TestEndTurnTicksRawSelectorsOneThenZeroBeforeEnemyPhase(t *testing.T) {
	g, _, _ := nativeCurrentSaveTestGame(t)
	first := g.st.Units[0]
	first.NativeRecordByte6 = 1
	first.NativeTransient[0] = 2
	g.st.NativeRuntimeRecords[0].Raw[6] = 1
	g.st.NativeRuntimeRecords[0].Raw[0x22] = 2

	secondValue := *first
	second := &secondValue
	second.NativeRecordByte6 = 0
	second.NativeTransient = [6]byte{0, 2}
	second.NativeIdentity = 31
	second.NativeRecordByte8 = 31
	second.NativeMapPresentation.X++
	second.X++
	secondRaw := g.st.NativeRuntimeRecords[0]
	secondRaw.Raw[0]++
	secondRaw.Raw[6] = 0
	secondRaw.Raw[8] = 31
	secondRaw.Raw[0x22], secondRaw.Raw[0x23] = 0, 2
	g.st.Units = append(g.st.Units, second)
	g.st.NativeRuntimeRecords = append(g.st.NativeRuntimeRecords, secondRaw)

	g.endTurn()
	g.aiStep() // 0x1A30B 先跑友軍 AI（0x1D80B）那一遍，跑完才進橫幅
	if g.loadErr != "" || !g.aiBusy || g.banner != "ENEMY PHASE" ||
		g.st.Units[0].NativeTransient[0] != 1 ||
		g.st.Units[1].NativeTransient[1] != 1 ||
		g.st.NativeRuntimeRecords[0].Raw[0x22] != 1 ||
		g.st.NativeRuntimeRecords[1].Raw[0x23] != 1 {
		t.Fatalf("err=%q ai=%v banner=%q durations=%v/%v raw=%d/%d",
			g.loadErr, g.aiBusy, g.banner,
			g.st.Units[0].NativeTransient, g.st.Units[1].NativeTransient,
			g.st.NativeRuntimeRecords[0].Raw[0x22],
			g.st.NativeRuntimeRecords[1].Raw[0x23])
	}
}

func TestNativeTransientPresentationFailsClosedBeforeCountdown(t *testing.T) {
	g, _, _ := nativeCurrentSaveTestGame(t)
	unit := g.st.Units[0]
	unit.NativeRecordByte6 = 2
	unit.NativeTransient[0] = 1
	g.st.NativeRuntimeRecords[0].Raw[6] = 2
	g.st.NativeRuntimeRecords[0].Raw[0x22] = 1

	if err := g.beginNativeTransientPhases([]byte{2}, nil); err == nil {
		t.Fatal("missing indexed presentation assets were accepted")
	}
	if g.st.Units[0].NativeTransient[0] != 1 ||
		g.st.NativeRuntimeRecords[0].Raw[0x22] != 1 ||
		g.nativeClassUIJob != nil || g.transientUI {
		t.Fatal("failed presentation published countdown or UI state")
	}
}

func TestNativeTransientPresentationUsesOriginalIndexedAssets(t *testing.T) {
	const base = "../../../org_game/炎龍騎士團/FLAME2"
	t.Setenv("FD2_ORIGINAL_FDOTHER", filepath.Join(base, "FDOTHER.DAT"))
	t.Setenv("FD2_ORIGINAL_FDTXT", filepath.Join(base, "FDTXT.DAT"))
	t.Setenv("FD2_ORIGINAL_DATO", filepath.Join(base, "DATO.DAT"))
	g, _, _ := nativeCurrentSaveTestGame(t)
	assets, err := loadNativeClassUIAssets()
	if err != nil {
		t.Fatal(err)
	}
	g.nativeClassUI = assets
	g.nativeMapVGA = make([]byte, 320*200)
	unit := g.st.Units[0]
	unit.NativeRecordByte6 = 2
	unit.NativeTransient[0] = 1
	g.st.NativeRuntimeRecords[0].Raw[6] = 2
	g.st.NativeRuntimeRecords[0].Raw[0x22] = 1
	continued := false

	if err := g.beginNativeTransientPhases([]byte{2}, func() { continued = true }); err != nil {
		t.Fatal(err)
	}
	if continued || !g.transientUI || g.nativeClassUIJob == nil ||
		len(g.nativeClassUIJob.frames) != 11 || len(g.nativeClassUIJob.restore) != 320*200 ||
		g.st.Units[0].NativeTransient[0] != 0 ||
		g.st.NativeRuntimeRecords[0].Raw[0x22] != 0 {
		t.Fatalf("continued=%v presenting=%v job=%v duration=%d raw=%d",
			continued, g.transientUI, g.nativeClassUIJob != nil,
			g.st.Units[0].NativeTransient[0], g.st.NativeRuntimeRecords[0].Raw[0x22])
	}
	for g.nativeClassUIJob != nil {
		g.nativeClassUIJob.drawn = true
		g.stepNativeClassUILifecycle(time.Time{})
	}
	if !continued || g.transientUI {
		t.Fatalf("continued=%v presenting=%v", continued, g.transientUI)
	}
}
