package battle

import (
	"bytes"
	"encoding/json"
	"testing"
)

func TestNativeDeathPresentationFlagPreservesWriterAndStaysOutOfSaves(t *testing.T) {
	unit := &Unit{HP: 10, MaxHP: 10, NativeRecordByte5: 0x80, HasNativeRecordByte5: true}
	unit.ApplyHPDamage(3)
	if _, valid := unit.NativeDeathPresentationFlag(); valid {
		t.Fatal("nonlethal damage created a corpse")
	}
	unit.ApplyHPDamage(7)
	unit.ApplyHPDamage(9)
	if flag, valid := unit.NativeDeathPresentationFlag(); !valid || flag != 0x80 || unit.NativeRecordByte5 != 1 {
		t.Fatal("death writer flag was guessed from the already inactive record")
	}
	with, err := json.Marshal(unit)
	if err != nil {
		t.Fatal(err)
	}
	var loaded Unit
	if err := json.Unmarshal(with, &loaded); err != nil {
		t.Fatal(err)
	}
	if _, valid := loaded.NativeDeathPresentationFlag(); valid {
		t.Fatal("presentation transition was serialized")
	}
	unit.ClearNativeDeathPresentationFlag()
	without, _ := json.Marshal(unit)
	if !bytes.Equal(with, without) || unit.HP != 0 || unit.NativeRecordByte5 != 1 {
		t.Fatal("clearing presentation metadata changed the save or death writer")
	}
	unit.RestoreNativeHP()
	unit.ApplyHPDamage(unit.HP)
	unit.RestoreNativeHP()
	unit.HP = 0 // a later unrelated HP write must not expose the previous corpse flag
	if _, valid := unit.NativeDeathPresentationFlag(); valid {
		t.Fatal("revival retained a previous death")
	}
}
