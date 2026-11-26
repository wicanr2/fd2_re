package battle

import "testing"

// 第四章事件 12 讓第 2 群組改成模式 7 撤退；抵達目的座標時 0x32975 把 +5 寫成 1。
// 原版預設勝負規則 0x205BE 只計 +5 bit0 清零的 raw +6==0 列，撤退抵達的敵軍不再擋住
// 勝利。引擎的 Result 看 OnField，所以離場必須同時清掉 OnField 並同步原始記錄。
func TestMarkNativeUnitInactiveLetsDepartedEnemyStopBlockingResult(t *testing.T) {
	hero := &Unit{Name: "索爾", Camp: Own, OnField: true, HP: 10, MaxHP: 10,
		NativeRecordByte5: 0, HasNativeRecordByte5: true}
	fleeing := &Unit{Camp: Enemy, OnField: true, HP: 36, MaxHP: 36,
		NativeRecordByte5: 0x80, HasNativeRecordByte5: true}
	st := &State{
		Units:                          []*Unit{hero, fleeing},
		NativeRuntimeRecords:           make([]NativeRuntimeRecordState, 2),
		HasNativeRuntimeUnitProjection: true,
	}
	st.NativeRuntimeRecords[1].Raw[5] = 0x80
	if got := st.Result("索爾"); got != "" {
		t.Fatalf("before departure Result=%q, want empty", got)
	}
	if err := st.MarkNativeUnitInactive(fleeing); err != nil {
		t.Fatal(err)
	}
	if fleeing.OnField || fleeing.NativeRecordByte5 != 1 || !fleeing.HasNativeRecordByte5 {
		t.Fatalf("departed unit OnField=%v byte5=%#x", fleeing.OnField, fleeing.NativeRecordByte5)
	}
	if raw := st.NativeRuntimeRecords[1].Raw[5]; raw != 1 {
		t.Fatalf("raw record +5=%#x, want whole byte 1", raw)
	}
	if !fleeing.Alive() {
		t.Fatal("departure must not change HP")
	}
	if got := st.Result("索爾"); got != "win" {
		t.Fatalf("after departure Result=%q, want win", got)
	}
}

func TestMarkNativeUnitInactiveRejectsForeignUnit(t *testing.T) {
	st := &State{Units: []*Unit{{Camp: Enemy, OnField: true, HP: 1}}}
	stranger := &Unit{Camp: Enemy, OnField: true, HP: 1}
	if err := st.MarkNativeUnitInactive(stranger); err == nil {
		t.Fatal("unit outside the runtime array unexpectedly accepted")
	}
	if !stranger.OnField || !st.Units[0].OnField {
		t.Fatal("rejected departure must not write any unit")
	}
}
