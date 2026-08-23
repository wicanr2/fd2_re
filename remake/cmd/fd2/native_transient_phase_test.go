package main

import (
	"encoding/binary"
	"path/filepath"
	"reflect"
	"testing"
	"time"
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

func TestNativeTransientPhaseFailsClosedWithoutRawProjection(t *testing.T) {
	g, _, _ := nativeCurrentSaveTestGame(t)
	g.st.HasNativeRuntimeUnitProjection = false
	before := g.st.Units[0].NativeTransient
	if _, err := g.applyNativeTransientPhase(0); err == nil {
		t.Fatal("missing raw projection was accepted")
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
