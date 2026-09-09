package main

import (
	"reflect"
	"testing"

	"github.com/wicanr2/fd2_re/remake/internal/battle"
)

func TestScenarioJoinBeforeNativeSpawnUsesPendingSource(t *testing.T) {
	g := &Game{}
	if err := g.loadMap("assets/maps/map0"); err != nil {
		t.Fatal(err)
	}
	g.resetBattle("assets/maps/map0/map0_units.json", "assets/scenarios/ch01.json")
	if g.loadErr != "" {
		t.Fatal(g.loadErr)
	}
	st, sc := g.st, g.sc
	g.dialog = nil // 本測試由明示的戰場資料起點開始，不列普通 START 證據。
	st.Turn = 3
	actions := sc.TriggerActions(st, "on_turn_end", "")
	if len(actions) < 4 || actions[0].Type != "join_party" || actions[0].CharID != 1 {
		t.Fatalf("首關原始動作順序不同：%+v", actions)
	}
	before := len(st.Units)
	done := false
	// 使用正式逐動作入口，刻意在第一個 JOIN 返回時檢查；不可先 Fire 整串。
	g.startBattleEvent(actions[:1], func() { done = true })
	if g.loadErr != "" || !done || len(st.Units) != before || !g.partyMembers[1] {
		t.Fatalf("JOIN 尚未登場時失敗／提前登場：%q units=%d members=%v", g.loadErr, len(st.Units), g.partyMembers)
	}
	u := g.partyRoster[1]
	if u.Lv != 3 || u.HP != 56 || u.NativeRecordWord42 != 56 || u.NativeRecordByte8 != 1 {
		t.Fatalf("永久記錄未使用原始 JOIN 表：%+v", u)
	}
	// 登場與完整對話生命週期由真實 START 長鏈測試驗證；本例只隔離 JOIN 來源。
	var spawnActions []battle.Action
	for _, action := range actions[1:] {
		if action.Type == "spawn_group" {
			spawnActions = append(spawnActions, action)
		}
	}
	g.startBattleEvent(spawnActions, func() {})
	if g.loadErr != "" || len(st.Units) != before+2 {
		t.Fatalf("後續援軍未達成：%q units=%d", g.loadErr, len(st.Units))
	}
	for _, unit := range st.Units[before:] {
		if unit.Camp != battle.Own || unit.NativeRecordByte6 != 2 {
			t.Fatalf("父子原始 raw2 我方記錄被錯分為 AI：%+v", unit)
		}
	}
	if g.partyMembers[3] {
		t.Fatal("本戰可操作的哈瓦特不應因此永久入隊")
	}
	if !reflect.DeepEqual(g.partyJoinOrder, []int{1}) {
		t.Fatalf("JOIN 順序重複：%v", g.partyJoinOrder)
	}
}

func TestScenarioJoinMissingOrDuplicateSourceDoesNotPublish(t *testing.T) {
	for _, duplicate := range []bool{false, true} {
		st := &battle.State{}
		if duplicate {
			st.Roster = []*battle.Unit{
				{Fig: 1, NativeRecordByte8: 1, HasNativeRecordByte8: true},
				{Fig: 1, NativeRecordByte8: 1, HasNativeRecordByte8: true},
			}
		}
		g := &Game{st: st, sc: &battle.Scenario{}}
		continued := false
		g.startBattleEvent([]battle.Action{{Type: "join_party", CharID: 1}}, func() { continued = true })
		if g.loadErr == "" || continued || len(g.partyMembers) != 0 || len(g.partyJoinOrder) != 0 || len(g.partyRoster) != 0 {
			t.Fatalf("來源拒絕後仍發布名冊或續行：%+v", g)
		}
	}
}
