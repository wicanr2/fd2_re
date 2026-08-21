package main

import "testing"

func TestNativeShopEquipmentRecipientObservedInputTrace(t *testing.T) {
	selected, start := 0, 0
	apply := func(name string, up, down bool, wantSelected, wantStart int) {
		t.Helper()
		var ok bool
		selected, start, ok = advanceNativeShopEquipmentRecipient(
			3, selected, start, up, down,
		)
		if !ok || selected != wantSelected || start != wantStart {
			t.Fatalf(
				"%s=(selected%d,start%d,ok%v), want (%d,%d,true)",
				name, selected, start, ok, wantSelected, wantStart,
			)
		}
	}

	apply("down", false, true, 1, 0)
	apply("up", true, false, 0, 0)
	apply("left-no-op", false, false, 0, 0)
	apply("right-no-op", false, false, 0, 0)
	apply("bounded-up", true, false, 0, 0)
	apply("simultaneous-up-then-down", true, true, 1, 0)
}

func TestNativeShopEquipmentRecipientStatefulThreeRowWindow(t *testing.T) {
	selected, start := 0, 0
	for i, step := range []struct {
		up, down        bool
		selected, start int
	}{
		{down: true, selected: 1, start: 0},
		{down: true, selected: 2, start: 0},
		{down: true, selected: 3, start: 1},
		{down: true, selected: 4, start: 2},
		{up: true, selected: 3, start: 2},
		{up: true, selected: 2, start: 2},
		{up: true, selected: 1, start: 1},
	} {
		var ok bool
		selected, start, ok = advanceNativeShopEquipmentRecipient(
			6, selected, start, step.up, step.down,
		)
		if !ok || selected != step.selected || start != step.start {
			t.Fatalf(
				"step%d=(selected%d,start%d,ok%v), want (%d,%d,true)",
				i, selected, start, ok, step.selected, step.start,
			)
		}
	}
}

func TestNativeShopEquipmentRecipientRejectsInvalidState(t *testing.T) {
	for _, state := range []struct {
		count, selected, start int
	}{
		{count: 0},
		{count: 3, selected: -1},
		{count: 3, selected: 3},
		{count: 3, start: -1},
	} {
		if _, _, ok := advanceNativeShopEquipmentRecipient(
			state.count, state.selected, state.start, false, false,
		); ok {
			t.Fatalf("invalid state accepted: %+v", state)
		}
	}
}

func TestNativeShopRecipientInputRejectsUnownedMode(t *testing.T) {
	g := &Game{nativeShopMode: "purchase"}
	if g.handleNativeShopRecipientInput(nativeShopRecipientInput{enter: true}) {
		t.Fatal("recipient input consumer claimed the purchase-list owner")
	}
	g.nativeShopMode = "recipient_full"
	if !g.handleNativeShopRecipientInput(nativeShopRecipientInput{}) {
		t.Fatal("recipient input consumer released an owned idle state")
	}
}
