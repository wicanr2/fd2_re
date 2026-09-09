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

func TestPlayerPhysicalAttackMessageFailsClosedWithoutNames(t *testing.T) {
	catalog, err := loadOfficialLocale("zh-Hant")
	if err != nil {
		t.Fatal(err)
	}
	for _, test := range []struct {
		name   string
		actor  *battle.Unit
		target *battle.Unit
	}{
		{name: "nil actor", target: &battle.Unit{Name: "盜賊"}},
		{name: "nil target", actor: &battle.Unit{Name: "索爾"}},
		{name: "empty actor", actor: &battle.Unit{}, target: &battle.Unit{Name: "盜賊"}},
		{name: "empty target", actor: &battle.Unit{Name: "索爾"}, target: &battle.Unit{}},
	} {
		t.Run(test.name, func(t *testing.T) {
			if message, err := playerPhysicalAttackMessage(catalog, test.actor, test.target, battle.AttackResult{Missed: true}); err == nil || message != "" {
				t.Fatalf("message=%q err=%v", message, err)
			}
		})
	}
}

// TestPhysicalAttackMessageStaysOutOfThePlayerPath 釘住「傷害數字不留在地圖上」。
//
// 原版沒有這一行：battle.attack.* 四筆的 source_string_id 都指向重製端自己的
// Go 原始碼，而原版分離出的 FDTXT 全庫沒有任何「造成 N 傷害」型的戰鬥結果
// 模板。字串保留作為語言包完整性檢查，但只有 F3 診斷模式才顯示。
func TestPhysicalAttackMessageStaysOutOfThePlayerPath(t *testing.T) {
	g := &Game{}
	g.publishPhysicalAttackMessage("索爾 攻擊 盜賊，造成 12 傷害")
	if g.msg != "" {
		t.Fatalf("一般玩家路徑不得顯示戰鬥結果字串，實得 %q", g.msg)
	}
	g.debug = true
	g.publishPhysicalAttackMessage("索爾 攻擊 盜賊，造成 12 傷害")
	if g.msg == "" {
		t.Fatal("F3 診斷模式應保留字串")
	}
}
