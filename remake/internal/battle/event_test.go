package battle

import "testing"

func TestScenarioPartyUnitsPreserveRuntimeOrderAndDeployment(t *testing.T) {
	sc := &Scenario{
		Party: []PartyMember{
			{Name: "索爾", Fig: 0, HP: 42},
			{Name: "悠妮", Fig: 9, HP: 28},
			{Name: "亞雷斯", Fig: 4, HP: 48},
			{Name: "蓋亞", Fig: 30, HP: 50},
		},
		DeployCells: [][2]int{{7, 20}, {8, 22}, {10, 21}, {11, 23}},
	}
	units := sc.PartyUnits(nil)
	if len(units) != 4 {
		t.Fatalf("party units=%d, want 4", len(units))
	}
	for slot, want := range []struct{ fig, x, y int }{
		{0, 7, 20}, {9, 8, 22}, {4, 10, 21}, {30, 11, 23},
	} {
		u := units[slot]
		if u.Fig != want.fig || u.X != want.x || u.Y != want.y || !u.OnField || u.Camp != Own {
			t.Fatalf("runtime slot %d = %#v, want fig=%d at (%d,%d)", slot, u, want.fig, want.x, want.y)
		}
	}
}

func TestScenarioPartyUnitsUseFDFIELDFallbackCells(t *testing.T) {
	sc := &Scenario{Party: []PartyMember{{Fig: 0}, {Fig: 9}}}
	units := sc.PartyUnits([]Cell{{X: 3, Y: 4}, {X: 5, Y: 6}})
	if units[0].X != 3 || units[0].Y != 4 || units[1].X != 5 || units[1].Y != 6 {
		t.Fatalf("fallback deployment lost: %#v", units)
	}
}

func TestChapter2RuntimeAppendOrderMatchesOriginalHandlerSlots(t *testing.T) {
	st, err := Load("../../assets/maps/map1/map1_units.json")
	if err != nil {
		t.Fatal(err)
	}
	sc, err := LoadScenario("../../assets/scenarios/ch02.json")
	if err != nil {
		t.Fatal(err)
	}
	sc.Setup(st)
	if len(st.Units) != 21 {
		t.Fatalf("setup runtime units=%d, want party5 + group1/2=16", len(st.Units))
	}
	for slot, portrait := range []int{0, 4, 9, 30, 1} {
		if st.Units[slot].Portrait != portrait || st.Units[slot].Camp != Own {
			t.Fatalf("party slot %d = portrait%d camp%s", slot, st.Units[slot].Portrait, st.Units[slot].Camp)
		}
	}
	for slot, portrait := range []int{134, 133, 134, 133, 134, 133} {
		u := st.Units[slot+5]
		if u.Portrait != portrait || u.Group != 1 || u.Camp != Ally {
			t.Fatalf("villager slot %d = portrait%d group%d camp%s", slot+5, u.Portrait, u.Group, u.Camp)
		}
	}
	if got := st.PendingCount(Enemy); got != 6 {
		t.Fatalf("pending enemies=%d, want scheduled group3 only", got)
	}
	if got := st.SpawnGroup(3, Ally, true, false); got != 6 || len(st.Units) != 27 {
		t.Fatalf("turn3 spawn=%d runtime units=%d, want 6/27", got, len(st.Units))
	}
	if got := st.AppendGroup(4); got != 1 || len(st.Units) != 28 {
		t.Fatalf("post SPAWN4=%d runtime units=%d, want 1/28", got, len(st.Units))
	}
	hilia := st.Units[27]
	if hilia.Portrait != 8 || hilia.Group != 4 || hilia.X != 22 || hilia.Y != 4 || !hilia.OnField {
		t.Fatalf("post slot27 = %#v", hilia)
	}
	for _, u := range st.Units {
		if u.Group == 255 {
			t.Fatal("group255 placeholder polluted canonical runtime slots")
		}
	}
}
