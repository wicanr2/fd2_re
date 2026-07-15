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

func TestChapter1Turn3JoinsHanoBeforeSpawningHisGroup(t *testing.T) {
	st, err := Load("../../assets/maps/map0/map0_units.json")
	if err != nil {
		t.Fatal(err)
	}
	sc, err := LoadScenario("../../assets/scenarios/ch01.json")
	if err != nil {
		t.Fatal(err)
	}
	sc.Setup(st)
	st.Turn = 3
	sc.Fire(st, "on_turn_end", "")
	joins := sc.TakePartyJoins()
	if len(joins) != 1 || joins[0] != 1 {
		t.Fatalf("turn3 joins=%#v, want Hano char1", joins)
	}
	var hano, hawat *Unit
	for _, unit := range st.Units {
		if unit.Fig == 1 {
			hano = unit
		}
		if unit.Fig == 3 {
			hawat = unit
		}
	}
	if hano == nil || !hano.OnField || hano.Camp != Own {
		t.Fatalf("Hano spawn = %#v, want recruited OWN unit", hano)
	}
	if hawat == nil || !hawat.OnField || hawat.Camp != Ally {
		t.Fatalf("Hawat spawn = %#v, want allied NPC", hawat)
	}
	if got := sc.TakePartyJoins(); len(got) != 0 {
		t.Fatalf("party joins were not consumed: %#v", got)
	}
}

func TestChapter3RuntimeAppendOrderMatchesPreHandlerSlots(t *testing.T) {
	st, err := Load("../../assets/maps/map2/map2_units.json")
	if err != nil {
		t.Fatal(err)
	}
	sc, err := LoadScenario("../../assets/scenarios/ch03.json")
	if err != nil {
		t.Fatal(err)
	}
	sc.Setup(st)
	if len(st.Units) != 15 {
		t.Fatalf("chapter3 initial slots=%d, want six party + nine group1 records", len(st.Units))
	}
	for slot, id := range []int{0, 4, 9, 30, 1, 8} {
		if st.Units[slot].Fig != id {
			t.Fatalf("authored party slot%d fig=%d, want %d before JOIN-order adapter", slot, st.Units[slot].Fig, id)
		}
	}
	if st.Units[6].Fig != 2 || st.Units[6].Camp != Ally {
		t.Fatalf("chapter3 slot6 = %#v, want Tino ally", st.Units[6])
	}
	for slot := 7; slot <= 14; slot++ {
		if st.Units[slot].Camp != Enemy {
			t.Fatalf("chapter3 slot%d camp=%v, want enemy", slot, st.Units[slot].Camp)
		}
	}
	for _, unit := range st.Units {
		if unit.Group == 255 {
			t.Fatalf("group255 source padding polluted runtime: %#v", unit)
		}
	}
}

func TestChapter3Turn3ReinforcementRequiresLivingTinoInRuntimeSlot6(t *testing.T) {
	load := func(t *testing.T) (*State, *Scenario) {
		t.Helper()
		st, err := Load("../../assets/maps/map2/map2_units.json")
		if err != nil {
			t.Fatal(err)
		}
		sc, err := LoadScenario("../../assets/scenarios/ch03.json")
		if err != nil {
			t.Fatal(err)
		}
		sc.Setup(st)
		st.Turn = 3
		return st, sc
	}

	dead, deadScenario := load(t)
	dead.Units[6].HP = 0
	deadDialogues := deadScenario.Fire(dead, "on_turn_end", "")
	if len(dead.Units) != 15 {
		t.Fatalf("dead Tino spawned group2: runtime units=%d, want 15", len(dead.Units))
	}
	if len(deadDialogues) != 0 {
		t.Fatalf("dead Tino played living-only #4 dialogue: %#v", deadDialogues)
	}

	alive, aliveScenario := load(t)
	aliveDialogues := aliveScenario.Fire(alive, "on_turn_end", "")
	if len(alive.Units) != 27 {
		t.Fatalf("living Tino runtime units=%d, want 15+12 group2", len(alive.Units))
	}
	if len(aliveDialogues) != 7 || aliveDialogues[0].Speaker != 77 || aliveDialogues[1].Speaker != 2 || aliveDialogues[6].Speaker != 77 {
		t.Fatalf("turn3 FDTXT_003 #4 dialogues = %#v", aliveDialogues)
	}
	if aliveDialogues[1].Text != "如果不是這些年輕人幫忙的話,我早就沒命了!不過既然我還活著,我還是要問你一個問題:到底是誰命令你來殺我?" {
		t.Fatalf("turn3 Tino line drifted: %q", aliveDialogues[1].Text)
	}
}
