package battle

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
)

func chapter13ResultScenario(t *testing.T) *Scenario {
	t.Helper()
	sc, err := LoadScenario("../../assets/scenarios/ch13.json")
	if err != nil {
		t.Fatal(err)
	}
	return sc
}

func nativeResultRecords(count int) *State {
	st := &State{NativeRoundCounter: 1}
	for i := 0; i < count; i++ {
		st.Units = append(st.Units, &Unit{HP: 999, HasNativeRecordByte5: true,
			HasNativeRecordByte6: true, NativeRecordByte6: 2})
	}
	return st
}

func TestNativeDefaultResultUsesRawCampAndBit0Priority(t *testing.T) {
	st := nativeResultRecords(2)
	st.Units[1].NativeRecordByte6 = 0
	st.Units[1].Camp, st.Units[1].HP = Own, 0 // 正規化Camp／HP不可替代原始讀取。
	st.Roster = []*Unit{{NativeRecordByte6: 0, HP: 999}}
	for _, tc := range []struct {
		own, enemy byte
		want       int
	}{
		{0, 0, 0}, {0, 0x80, 0}, {0, 1, 2}, {1, 0, 1}, {1, 1, 1},
	} {
		st.Units[0].NativeRecordByte5, st.Units[1].NativeRecordByte5 = tc.own, tc.enemy
		code, err := NativeDefaultResultCode(st)
		if err != nil || code != tc.want {
			t.Fatalf("raw旗標%d/%d結果=%d，預期%d：%v", tc.own, tc.enemy, code, tc.want, err)
		}
	}
	st.Units[1].HasNativeRecordByte6 = false
	if _, err := NativeDefaultResultCode(st); err == nil {
		t.Fatal("原始camp缺漏卻推測結果")
	}
}

func TestChapter13AllInitialAlliesUsesEveryRawRecord(t *testing.T) {
	rule := chapter13ResultScenario(t).NativeResultRules[0]
	st := nativeResultRecords(59)
	for i := 15; i <= 26; i++ {
		st.Units[i].NativeRecordByte5 = 0x81
	}
	if matched, err := rule.Match(st); err != nil || !matched {
		t.Fatalf("全部bit0為1未命中：%v", err)
	}
	st.Units[20].HP, st.Units[20].NativeRecordByte5 = 0, 0x80
	if matched, err := rule.Match(st); err != nil || matched {
		t.Fatalf("HP0／bit7被誤判成bit0：%v", err)
	}
	st.Units[26].HasNativeRecordByte5 = false
	if _, err := rule.Match(st); err == nil {
		t.Fatal("發現一名活動者後漏查後方原始欄位")
	}
}

func TestChapter13RoundGatePrecedesUncreatedRecord59(t *testing.T) {
	rule := chapter13ResultScenario(t).NativeResultRules[1]
	st := nativeResultRecords(59)
	st.NativeRoundCounter = 5
	if matched, err := rule.Match(st); err != nil || matched {
		t.Fatalf("回合5提前讀取59：%v", err)
	}
	st.NativeRoundCounter = 6
	if _, err := rule.Match(st); err == nil {
		t.Fatal("回合6缺記錄59卻猜成存活")
	}
	st.Units = append(st.Units, &Unit{HasNativeRecordByte5: true, NativeRecordByte5: 1, HP: 999})
	if matched, err := rule.Match(st); err != nil || !matched {
		t.Fatalf("回合6的bit0未命中：%v", err)
	}
	st.NativeRoundCounter = 0
	if _, err := rule.Match(st); err == nil {
		t.Fatal("原生回合缺漏卻借用正規化回合")
	}
}

func TestLoadScenarioRejectsUnknownOrIncompleteNativeResults(t *testing.T) {
	for _, mutate := range []func(*Scenario){
		func(sc *Scenario) { sc.NativeResultRules[0].Code = 2 },
		func(sc *Scenario) { sc.NativeResultRules[0].RecordsAllInactive = nil },
		func(sc *Scenario) { sc.NativeResultRules[0].Actions[0].NativeDialogueRef = nil },
		func(sc *Scenario) { sc.NativeResultRules[1].ID = sc.NativeResultRules[0].ID },
		func(sc *Scenario) { sc.NativeResultCode1Records = []int{59} },
	} {
		sc := chapter13ResultScenario(t)
		mutate(sc)
		raw, err := json.Marshal(sc)
		if err != nil {
			t.Fatal(err)
		}
		path := filepath.Join(t.TempDir(), "scenario.json")
		if err := os.WriteFile(path, raw, 0600); err != nil {
			t.Fatal(err)
		}
		if _, err := LoadScenario(path); err == nil {
			t.Fatal("不完整或未閉合的結果列被正式載入")
		}
	}
}
