package battle

import "testing"

func TestNativeEndTurnRecoveryFollows1A30BGates(t *testing.T) {
	mk := func(hp, maxHP int, byte5 byte) *Unit {
		u := completeNativeAIScoringUnit()
		u.OnField, u.Camp = true, Own
		u.NativeRecordByte6, u.NativeRecordByte5 = 2, byte5
		u.NativeTransient = [6]byte{}
		u.HP, u.MaxHP = hp, maxHP
		return u
	}
	fresh := mk(87, 147, 0)     // 147/5 = 29 → 116（第四章 r8 收據 seq 671 的 id 4）
	nearFull := mk(140, 147, 0) // 140+29 > 147 → 取 147
	acted := mk(50, 100, 0x80)  // 已行動不回復
	dead := mk(0, 100, 1)       // 不在場／陣亡不回復
	full := mk(100, 100, 0)     // 滿血不回復
	transient := mk(50, 100, 0) // +0x26 非零不回復
	transient.NativeTransient[4] = 3
	enemy := mk(50, 100, 0)
	enemy.NativeRecordByte6 = 0
	legacy := &Unit{OnField: true, Camp: Own, HP: 10, MaxHP: 50} // 沒有 raw 旗標走 Camp／Acted
	s := &State{Units: []*Unit{fresh, nearFull, acted, dead, full, transient, enemy, legacy}}
	got := s.ApplyNativeEndTurnRecovery()
	if len(got) != 3 || got[0].Unit != fresh || got[1].Unit != nearFull || got[2].Unit != legacy {
		t.Fatalf("回復名單=%+v", got)
	}
	if fresh.HP != 116 || nearFull.HP != 147 || legacy.HP != 20 ||
		acted.HP != 50 || dead.HP != 0 || full.HP != 100 || transient.HP != 50 || enemy.HP != 50 {
		t.Fatalf("HP：fresh=%d nearFull=%d legacy=%d acted=%d dead=%d full=%d transient=%d enemy=%d",
			fresh.HP, nearFull.HP, legacy.HP, acted.HP, dead.HP, full.HP, transient.HP, enemy.HP)
	}
	// 0x1A477 → 0x13512：回復過的記錄 +5 bit7 設起來；沒回復的不動。
	if fresh.NativeRecordByte5 != 0x80 || nearFull.NativeRecordByte5 != 0x80 ||
		full.NativeRecordByte5 != 0 || transient.NativeRecordByte5 != 0 {
		t.Fatalf("+5：fresh=%#x nearFull=%#x full=%#x transient=%#x",
			fresh.NativeRecordByte5, nearFull.NativeRecordByte5, full.NativeRecordByte5, transient.NativeRecordByte5)
	}
}

func TestMarkNativeDeadRecordsWritesByte5AfterCommandDamage(t *testing.T) {
	dead := completeNativeAIScoringUnit()
	dead.HP, dead.NativeRecordByte5 = 0, 0x80 // 指令傷害只寫 HP，+5 還留著已行動位
	alive := completeNativeAIScoringUnit()
	alive.HP, alive.NativeRecordByte5 = 5, 0x80
	legacy := &Unit{HP: 0}
	s := &State{Units: []*Unit{dead, alive, legacy}}
	got := s.MarkNativeDeadRecords()
	if len(got) != 1 || got[0] != dead || dead.NativeRecordByte5 != 1 || alive.NativeRecordByte5 != 0x80 {
		t.Fatalf("marked=%v dead.+5=%#x alive.+5=%#x", got, dead.NativeRecordByte5, alive.NativeRecordByte5)
	}
	if again := s.MarkNativeDeadRecords(); len(again) != 0 {
		t.Fatalf("第二次掃描不該再標：%v", again)
	}
	// 標記後 0x1A30B 的回復不會再碰它。
	dead.OnField, dead.Camp, dead.NativeRecordByte6, dead.MaxHP = true, Own, 2, 100
	dead.NativeTransient = [6]byte{}
	if rec := s.ApplyNativeEndTurnRecovery(); len(rec) != 0 || dead.HP != 0 {
		t.Fatalf("陣亡單位被回復：%+v hp=%d", rec, dead.HP)
	}
}
