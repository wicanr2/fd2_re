package main

import (
	"encoding/binary"
	"testing"
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
