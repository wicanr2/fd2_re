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
