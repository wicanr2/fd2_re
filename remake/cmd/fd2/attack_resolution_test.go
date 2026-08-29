package main

import (
	"math/rand"
	"strings"
	"testing"

	"github.com/wicanr2/fd2_re/remake/internal/battle"
)

func TestResolvePlayerPhysicalAttackUsesInjectedGameRNG(t *testing.T) {
	actor := &battle.Unit{
		Name: "索爾", Camp: battle.Own, OnField: true,
		HP: 100, MaxHP: 100, AP: 80, HIT: 100,
	}
	target := &battle.Unit{
		Name: "盜賊", Camp: battle.Enemy, OnField: true,
		HP: 120, MaxHP: 120, DP: 10,
	}
	state := &battle.State{W: 2, H: 1, Units: []*battle.Unit{actor, target}}
	g := &Game{st: state, rng: rand.New(rand.NewSource(9))}

	result, err := g.resolvePlayerPhysicalAttack(actor, target)
	if err != nil {
		t.Fatal(err)
	}
	if result.Missed || result.Amount <= 0 {
		t.Fatalf("result=%+v, want a deterministic hit", result)
	}
	if target.HP != target.MaxHP-result.Amount {
		t.Fatalf("target HP=%d, amount=%d", target.HP, result.Amount)
	}
	if !actor.Acted {
		t.Fatal("physical action did not consume actor turn")
	}
}

func TestResolvePlayerPhysicalAttackFailsClosedWithoutRNG(t *testing.T) {
	actor := &battle.Unit{Camp: battle.Own, OnField: true, HP: 10, MaxHP: 10, HIT: 100}
	target := &battle.Unit{Camp: battle.Enemy, OnField: true, HP: 10, MaxHP: 10, DP: 1}
	g := &Game{st: &battle.State{W: 2, H: 1, Units: []*battle.Unit{actor, target}}}

	if _, err := g.resolvePlayerPhysicalAttack(actor, target); err == nil {
		t.Fatal("missing RNG was accepted")
	}
	if actor.Acted || target.HP != target.MaxHP {
		t.Fatalf("failed-closed attack mutated state: acted=%v hp=%d", actor.Acted, target.HP)
	}
}

func TestPlayerPhysicalAttackMessagePreservesSettlementResult(t *testing.T) {
	catalog, err := loadOfficialLocale("zh-Hant")
	if err != nil {
		t.Fatal(err)
	}
	actor := &battle.Unit{Name: "亞雷斯"}
	target := &battle.Unit{Name: "盜賊"}
	miss, err := playerPhysicalAttackMessage(catalog, actor, target, battle.AttackResult{Missed: true})
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(miss, "未命中") {
		t.Fatalf("miss message=%q", miss)
	}
	hit, err := playerPhysicalAttackMessage(catalog, actor, target, battle.AttackResult{Amount: 12, Crit: true, ExpGained: 8})
	if err != nil {
		t.Fatal(err)
	}
	for _, want := range []string{"造成 12 傷害", "暴擊", "經驗 +8"} {
		if !strings.Contains(hit, want) {
			t.Fatalf("hit message=%q missing %q", hit, want)
		}
	}
}

func TestPlayerPhysicalAttackMessageUsesAllOfficialLocales(t *testing.T) {
	actor := &battle.Unit{Name: "Sol"}
	target := &battle.Unit{Name: "Bandit"}
	wants := map[string]string{"zh-Hant": "未命中", "zh-Hans": "未命中", "ja": "外れた", "en": "misses"}
	for localeID, want := range wants {
		catalog, err := loadOfficialLocale(localeID)
		if err != nil {
			t.Fatalf("%s: %v", localeID, err)
		}
		message, err := playerPhysicalAttackMessage(catalog, actor, target, battle.AttackResult{Missed: true})
		if err != nil || !strings.Contains(message, want) {
			t.Fatalf("%s message=%q err=%v, want %q", localeID, message, err, want)
		}
	}
}
