package main

import (
	"fmt"
	"reflect"
	"testing"

	"github.com/wicanr2/fd2_re/remake/internal/battle"
)

// 只供明示修改路徑的章重播。正式 Game 不含鎖血入口。
// 契約：docs/data/ida/fd2_ch27_flow_20261016.json#resample_hp_lock_policy。
type parityHPLockTarget struct {
	Index    int  `json:"index"`
	Camp     int  `json:"camp"`
	Identity int  `json:"identity"`
	HP       *int `json:"hp,omitempty"`
}
type parityHPLockCell struct {
	Index    int `json:"index"`
	Camp     int `json:"camp"`
	Identity int `json:"identity"`
	HP       int `json:"hp"`
}
type parityHPLockPolicy struct{ cells []parityHPLockCell }

var parityHPPolicies = map[*Game]*parityHPLockPolicy{}

func parityHPLockUnit(g *Game, cell parityHPLockCell) (*battle.Unit, error) {
	if g.st == nil || cell.Index < 0 || cell.Index >= len(g.st.Units) {
		return nil, fmt.Errorf("鎖血索引越界")
	}
	u := g.st.Units[cell.Index]
	identity, hasIdentity := 0, false
	if u != nil {
		identity, hasIdentity = u.NativeIdentity, u.HasNativeIdentity && u.NativeIdentity >= 0 && u.NativeIdentity <= 255
		if u.HasNativeRecordByte8 {
			identity, hasIdentity = int(u.NativeRecordByte8), true
		}
	}
	// 沿 NativeItemPanelRecordForUnit 的既有 player-persistent +8 相容契約。
	if u == nil || !u.OnField || u.HP <= 0 || u.HP > 32767 ||
		!u.HasNativeRecordByte5 || u.NativeRecordByte5&1 != 0 ||
		!u.HasNativeRecordByte6 || int(u.NativeRecordByte6) != cell.Camp ||
		!hasIdentity || identity != cell.Identity {
		return nil, fmt.Errorf("鎖血record%d缺少相同活躍來源", cell.Index)
	}
	return u, nil
}

func (p *parityHPLockPolicy) apply(g *Game) error {
	units := make([]*battle.Unit, len(p.cells))
	for i, cell := range p.cells {
		u, err := parityHPLockUnit(g, cell)
		if err != nil {
			return err
		}
		units[i] = u
	}
	for i, u := range units {
		u.HP = p.cells[i].HP
	}
	return nil
}

// 鎖血與指令傷害演出：原版 oracle 每一步寫回鎖血，原版演出不核對中間 HP；重製端的分段扣血
// （ApplyNativeCommandDamageStage）會核對目標仍是上一段的 HP。演出進行中先不寫回，演出收尾後的
// 下一次寫回再復原。演出把鎖血單位打到 0 時，下一次寫回因綁定失效而失敗，不會默默蓋過。
// 與 Rust parity_update_game 相同（2026-11-27 第 25 章 r7 樣本敵方指令 6 打中鎖血索爾時補上）。
func parityUpdateGame(g *Game) error {
	policy := parityHPPolicies[g]
	apply := func() error {
		if policy == nil || parityNativeCommandDamagePresentationActive(g) {
			return nil
		}
		return policy.apply(g)
	}
	if err := apply(); err != nil {
		return err
	}
	if err := g.Update(); err != nil {
		return err
	}
	return apply()
}

// 有分段發布 HP 的指令演出進行中（與 Rust native_command_damage_presentation_active 同一組工作）。
func parityNativeCommandDamagePresentationActive(g *Game) bool {
	return g.nativeHealPresentation != nil || g.nativeModifierPresentation != nil ||
		g.nativeAICommandModifier != nil || g.nativeAIItemPresentation != nil ||
		g.nativeCmd0Presentation != nil || g.nativeCmd1Presentation != nil ||
		g.nativeCmd2Presentation != nil || g.nativeCmd3Presentation != nil ||
		g.nativeCmd5Presentation != nil || g.nativeCmd6Presentation != nil ||
		g.nativeCmd7Presentation != nil || g.nativeCmd8Presentation != nil ||
		g.nativeCmd9Player != nil || g.nativeCmd9AIPresentation != nil ||
		g.nativeCmd1012 != nil || g.nativeCmd24Presentation != nil ||
		g.nativeCmd29Presentation != nil || g.nativeCmd32Presentation != nil ||
		g.nativeCmd33Presentation != nil || g.nativeCmd34Presentation != nil ||
		g.nativeCmd35Presentation != nil
}

func (r *parityReplay) updateHPLocks(action parityAction) {
	p := parityHPPolicies[r.g]
	if p == nil || len(action.HPLock) > 4 {
		r.t.Fatal("鎖血缺少明示測試政策")
	}
	if !pump(r.t, r.g, journeyStoryFrames, r.playerHasControl) {
		r.t.Fatal("鎖血前未取得正常游標")
	}
	next := make([]parityHPLockCell, 0, len(action.HPLock))
	seen := map[int]bool{}
	for _, target := range action.HPLock {
		cell := parityHPLockCell{Index: target.Index, Camp: target.Camp, Identity: target.Identity}
		if seen[cell.Index] || (cell.Camp != 0 && cell.Camp != 2) || cell.Identity < 0 || cell.Identity > 255 {
			r.t.Fatal("鎖血綁定重複或越界")
		}
		seen[cell.Index] = true
		u, err := parityHPLockUnit(r.g, cell)
		if err != nil {
			r.t.Fatal(err)
		}
		cell.HP = u.HP
		for _, old := range p.cells {
			if old.Index == cell.Index && old.Camp == cell.Camp && old.Identity == cell.Identity {
				cell.HP = old.HP
			}
		}
		if target.HP != nil {
			cell.HP = *target.HP
		}
		if cell.HP < 1 || cell.HP > 32767 {
			r.t.Fatal("鎖血HP超出正有號word")
		}
		next = append(next, cell)
	}
	if len(next) != len(action.Captured) || (len(next) > 0 && !reflect.DeepEqual(next, action.Captured)) {
		r.t.Fatalf("鎖血來源與原版不同：%+v／%+v", next, action.Captured)
	}
	p.cells = next
	if err := p.apply(r.g); err != nil {
		r.t.Fatal(err)
	}
	r.checkpoint("hp_lock", action.Seq, r.ui(), false, "修改路徑；每次Update前後復原明示HP")
}

func parityPreservesEnemy(t *testing.T, g *Game, indices []int, index int) bool {
	t.Helper()
	p := parityHPPolicies[g]
	locked := map[int]bool{}
	if p != nil {
		for _, cell := range p.cells {
			if cell.Camp == 0 {
				locked[cell.Index] = true
			}
		}
	}
	want := map[int]bool{}
	for _, i := range indices {
		if want[i] || !locked[i] {
			t.Fatal("清敵保留索引缺少鎖血綁定")
		}
		want[i] = true
	}
	if !reflect.DeepEqual(want, locked) {
		t.Fatal("清敵未明示保留全部敵方鎖")
	}
	return want[index]
}

func TestParitySelectiveHPLockAtomicAndLive(t *testing.T) {
	g := &Game{st: &battle.State{Units: []*battle.Unit{
		{OnField: true, HP: 300, HasNativeRecordByte5: true, HasNativeRecordByte6: true, NativeRecordByte6: 2, HasNativeRecordByte8: true},
		{OnField: true, HP: 100, HasNativeRecordByte5: true, HasNativeRecordByte6: true, HasNativeRecordByte8: true, NativeRecordByte8: 116},
	}}}
	p := &parityHPLockPolicy{cells: []parityHPLockCell{{Index: 0, Camp: 2, HP: 300}, {Index: 1, Identity: 116, HP: 1}}}
	if err := p.apply(g); err != nil || g.st.Units[1].HP != 1 {
		t.Fatal("明示HP1未套用", err)
	}
	g.st.Units[0].HP = 200
	g.st.Units[1].NativeRecordByte8 = 117
	if err := p.apply(g); err == nil || g.st.Units[0].HP != 200 {
		t.Fatal("錯誤綁定仍部分寫入")
	}
	g.st.Units[1].NativeRecordByte8 = 116
	g.st.Units[1].NativeRecordByte5 = 1
	if err := p.apply(g); err == nil {
		t.Fatal("鎖血復活inactive單位")
	}
	g.st.Units[1].NativeRecordByte5 = 0
	g.st.Units[0].HasNativeRecordByte8 = false
	g.st.Units[0].HasNativeIdentity = true
	if err := p.apply(g); err != nil {
		t.Fatal("既有持續角色+8相容來源未承接", err)
	}
	g.st.Units[0].HP = 200
	p.cells = nil
	if err := p.apply(g); err != nil || g.st.Units[0].HP != 200 {
		t.Fatal("解除後仍寫入", err)
	}
}
