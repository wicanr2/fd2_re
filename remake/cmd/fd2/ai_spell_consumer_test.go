package main

import (
	"math/rand"
	"testing"

	"github.com/wicanr2/fd2_re/remake/internal/battle"
)

func TestAIStepConsumesEditableHealSpellThroughProductionLoop(t *testing.T) {
	caster := &battle.Unit{
		Camp: battle.Enemy, X: 0, Y: 0, OnField: true,
		HP: 40, MaxHP: 40, MP: 10, MaxMP: 10, Spells: []int{13},
	}
	ally := &battle.Unit{
		Camp: battle.Enemy, X: 1, Y: 0, OnField: true,
		HP: 10, MaxHP: 30, MP: 0, MaxMP: 0, Acted: true,
	}
	state := &battle.State{
		W: 3, H: 1, Units: []*battle.Unit{caster, ally},
		SpellBook: []battle.Spell{{ID: 13, Name: "回復", Dmg: 70, Dist: 3, MP: 5, Target: 1}},
	}
	g := &Game{st: state, aiBusy: true, rng: rand.New(rand.NewSource(7))}

	g.aiStep()
	if g.loadErr != "" || caster.HP != 40 || ally.HP <= 10 || caster.MP != 5 || !caster.Acted {
		t.Fatalf("heal spell was not consumed: err=%q caster=%#v ally=%#v", g.loadErr, caster, ally)
	}
	if state.Turn != 0 || !g.aiBusy {
		t.Fatalf("first spell action advanced turn prematurely: turn=%d ai=%v", state.Turn, g.aiBusy)
	}
	g.aiStep()
	if g.loadErr != "" || state.Turn != 1 || g.aiBusy {
		t.Fatalf("spell loop did not finish turn: turn=%d ai=%v err=%q", state.Turn, g.aiBusy, g.loadErr)
	}
}

func TestAIStepConsumesEditableAttackSpellAndMovesIntoRange(t *testing.T) {
	caster := &battle.Unit{
		Camp: battle.Enemy, X: 0, Y: 0, OnField: true,
		HP: 40, MaxHP: 40, MP: 10, MaxMP: 10, MV: 1, Spells: []int{0},
	}
	target := &battle.Unit{
		Camp: battle.Own, X: 2, Y: 0, OnField: true,
		HP: 30, MaxHP: 30, AP: 1, DP: 1,
	}
	state := &battle.State{
		W: 3, H: 1, Units: []*battle.Unit{caster, target},
		SpellBook: []battle.Spell{{ID: 0, Name: "火炎", Dmg: 50, Hit: 100, Dist: 1, MP: 5, Target: 0}},
	}
	g := &Game{
		m:  &MapData{W: 3, H: 1, TileW: 24, TileH: 24, Tiles: []int{0, 0, 0}},
		st: state, aiBusy: true, rng: rand.New(rand.NewSource(11)),
	}

	g.aiStep()
	if g.loadErr != "" || g.walk == nil || len(g.walk.path) != 2 || g.walk.path[len(g.walk.path)-1] != (battle.Cell{X: 1, Y: 0}) {
		t.Fatalf("attack spell did not choose cast-range path: walk=%#v err=%q", g.walk, g.loadErr)
	}
	for step := 0; g.walk != nil && step < 8; step++ {
		g.stepBattleWalk()
	}
	if g.loadErr != "" || g.walk != nil || target.HP >= 30 || !caster.Acted {
		t.Fatalf("attack spell was not consumed after walk: caster=%#v target=%#v err=%q", caster, target, g.loadErr)
	}
	g.aiStep()
	if g.loadErr != "" || state.Turn != 1 || g.aiBusy {
		t.Fatalf("attack spell loop did not finish turn: turn=%d ai=%v err=%q", state.Turn, g.aiBusy, g.loadErr)
	}
}

func TestAIStepStopsSpellWithoutRNGBeforeMutation(t *testing.T) {
	caster := &battle.Unit{
		Camp: battle.Enemy, X: 0, Y: 0, OnField: true,
		HP: 40, MaxHP: 40, MP: 10, MaxMP: 10, Spells: []int{13},
	}
	ally := &battle.Unit{Camp: battle.Enemy, X: 1, Y: 0, OnField: true, HP: 10, MaxHP: 30, Acted: true}
	state := &battle.State{
		W: 2, H: 1, Units: []*battle.Unit{caster, ally},
		SpellBook: []battle.Spell{{ID: 13, Name: "回復", Dmg: 70, Dist: 3, MP: 5, Target: 1}},
	}
	// planner 可以建立合法法術 route，但 production executor 在缺少決定性
	// （deterministic）RNG 邊界時，必須在 CastArea 改寫 MP 或 HP 前拒絕。
	g := &Game{st: state, aiBusy: true}
	g.aiStep()
	if g.loadErr == "" || g.aiBusy || caster.Acted || caster.MP != 10 || ally.HP != 10 {
		t.Fatalf("spell without RNG was not fail-closed: err=%q ai=%v acted=%v mp=%d allyHP=%d", g.loadErr, g.aiBusy, caster.Acted, caster.MP, ally.HP)
	}
}
