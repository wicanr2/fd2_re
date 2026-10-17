package battle

import "testing"

func TestChapter30ResultRequiresBossFlagAndDefeatOverridesWin(t *testing.T) {
	sc, err := LoadScenario("../../assets/scenarios/ch30.json")
	if err != nil {
		t.Fatal(err)
	}
	for _, test := range []struct {
		boss, leader, yuni byte
		want               int
	}{
		{0, 0, 0, 0}, {0x80, 0, 0, 0}, {1, 0, 0, 2}, {1, 1, 0, 1}, {1, 0, 1, 1},
	} {
		st := nativeResultRecords(33)
		st.HasNativeTurnEventControlState = true
		st.Units[20].NativeRecordByte5 = test.boss
		st.Units[0].NativeRecordByte5 = test.leader
		st.Units[1].NativeRecordByte5 = test.yuni
		// HP與其他敵人是否在場都不能取代record20的raw+5。
		st.Units[20].HP = 0
		code, err := NativeInitialResultCode(st, sc)
		if err != nil {
			t.Fatal(err)
		}
		for _, rule := range sc.NativeResultRules {
			matched, err := rule.Match(st)
			if err != nil {
				t.Fatal(err)
			}
			if matched {
				code = rule.Code
			}
		}
		if code != test.want {
			t.Fatalf("flags=%+v result=%d", test, code)
		}
		st.Units[20].HasNativeRecordByte5 = false
		if _, err := sc.NativeResultRules[0].Match(st); err == nil {
			t.Fatal("缺boss raw旗標仍猜測勝利")
		}
	}
	if len(sc.NativeResultRules[2].Actions) != 1 || *sc.NativeResultRules[2].Actions[0].NativeTextIndex != 7 {
		t.Fatal("悠妮敗北文字庫索引錯誤")
	}
}

func TestChapter30DormantEvent82RejectsInvalidProgress(t *testing.T) {
	sc, err := LoadScenario("../../assets/scenarios/ch30.json")
	if err != nil {
		t.Fatal(err)
	}
	for _, phase := range []byte{0, 1, 2, 3, 4, 5} {
		st := &State{HasNativeTurnEventControlState: true, NativeRoundCounter: 2}
		st.NativeTurnEventControls[0] = NativeTurnEventControl{Turn: 2, EventID: 82, RawCamp: 0}
		st.NativeEventState[16] = phase
		events, err := sc.NativeTurnEventsAt(st, 0)
		if phase == 0 || phase == 5 {
			if err == nil {
				t.Fatalf("progress%d沒有拒收", phase)
			}
			continue
		}
		if err != nil || len(events) != 1 {
			t.Fatalf("progress%d: %v", phase, err)
		}
		var placements []NativeStagingPlacement
		var groups []int
		for _, action := range events[0].Actions {
			active, err := action.NativeWhen.Match(st)
			if err != nil {
				t.Fatal(err)
			}
			if !active {
				continue
			}
			if action.NativeStagingPresent != nil {
				placements = append(placements, *action.NativeStagingPresent)
			}
			groups = append(groups, action.Groups...)
		}
		if len(placements) == 0 || placements[0] != (NativeStagingPlacement{Slot: 24 - int(phase), X: 22, Y: 18}) {
			t.Fatalf("progress%d未保留原始魔神slot與目的座標: %v", phase, placements)
		}
		if phase < 4 {
			if len(groups) != 1 || groups[0] != int(phase) || len(placements) != 3 ||
				placements[1] != (NativeStagingPlacement{Slot: 25 + 2*int(phase), X: 21, Y: 18}) ||
				placements[2] != (NativeStagingPlacement{Slot: 26 + 2*int(phase), X: 23, Y: 18}) {
				t.Fatalf("progress%d增援順序不符: %v %v", phase, groups, placements)
			}
		} else if len(groups) != 0 || len(placements) != 1 {
			t.Fatal("最後phase不應追加增援")
		}
	}
}

func TestChapter30Field80AlwaysGateStillRequiresSource(t *testing.T) {
	for _, missing := range []bool{false, true} {
		st := &State{W: 1, H: 1, NativeFieldEventSlots: []int{0}, NativeFieldEvents: make([]NativeFieldEvent, 16)}
		st.NativeFieldEvents[0] = NativeFieldEvent{EventID: 80, Selector: 1}
		st.NativeFieldEventRules = []NativeFieldEventRule{{EventID: 80, Handler: "0x35f5a", Selector: 1, TriggerGate: "always", SetModeRanges: []NativeFieldModeRange{{Start: 20, End: 20, Mode: 11}}}}
		st.Units = make([]*Unit, 21)
		st.Units[20] = &Unit{NativeRecordByte34: 0xa8, HasNativeRecordByte34: !missing}
		// 35F5A不讀取actor raw+6，不能套其他格子事件的陣營gate。
		_, applied := ApplyNativeFieldModeEvent(st, &Unit{}, 0, 0, 1)
		if missing {
			if applied || st.Units[20].NativeRecordByte34 != 0xa8 {
				t.Fatal("缺來源仍發布模式")
			}
		} else if !applied || st.Units[20].NativeRecordByte34 != 0xab {
			t.Fatal("未保留mode高位或誤套actor gate")
		}
	}
}
