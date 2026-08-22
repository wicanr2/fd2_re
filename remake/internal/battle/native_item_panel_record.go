package battle

import (
	"encoding/binary"
	"errors"
	"math"
)

// NativeItemPanelRecordForUnit materializes only the 80-byte fields consumed
// by 0x17eef/0x17fc0/0x184c0. Raw provenance is mandatory for selectors,
// constructor bytes and inventory cells; normalized fields are used only
// where the native record meaning is independently established.
func NativeItemPanelRecordForUnit(unit *Unit) ([]byte, error) {
	if unit == nil || !unit.HasNativeRecordByte6 ||
		!unit.HasNativeRecordRace || !unit.HasNativeRecordClass ||
		len(unit.InventorySlots) != nativeInventoryCells ||
		len(unit.NativeInventoryFlags) != nativeInventoryCells {
		return nil, errors.New("native item panel: unit lacks raw record provenance")
	}
	rawByte8, hasRawByte8 := nativeRecordByte8ForUnit(unit)
	if !hasRawByte8 {
		return nil, errors.New("native item panel: unit lacks raw record byte +8")
	}
	if unit.BattleFig < 0 || unit.BattleFig > 0xff ||
		unit.Lv < 0 || unit.Lv > 0xff ||
		unit.MV < 0 || unit.MV > 0xff ||
		unit.Exp < 0 || unit.Exp > 0xff || unit.Exp != math.Trunc(unit.Exp) {
		return nil, errors.New("native item panel: byte field is outside native range")
	}
	for slot := 0; slot < nativeInventoryCells; slot++ {
		if unit.InventorySlots[slot] < 0 || unit.InventorySlots[slot] > 0xff ||
			unit.NativeInventoryFlags[slot] < 0 || unit.NativeInventoryFlags[slot] > 0xff {
			return nil, errors.New("native item panel: inventory cell is outside native range")
		}
	}
	for _, value := range []int{
		unit.DX, unit.HP, unit.MaxHP, unit.MP, unit.MaxMP,
		unit.AP, unit.DP, unit.HIT, unit.EV,
	} {
		if value < math.MinInt16 || value > math.MaxInt16 {
			return nil, errors.New("native item panel: word field is outside native range")
		}
	}

	record := make([]byte, nativeRecordSize)
	record[6] = unit.NativeRecordByte6
	record[7] = byte(unit.BattleFig)
	record[8] = rawByte8
	for slot := 0; slot < nativeInventoryCells; slot++ {
		record[0x0a+slot*2] = byte(unit.NativeInventoryFlags[slot])
		record[0x0b+slot*2] = byte(unit.InventorySlots[slot])
	}
	copy(record[0x1a:0x1f], unit.NativeCommandMask[:])
	record[0x1f] = unit.NativeRecordRace
	record[0x20] = unit.NativeRecordClass
	record[0x21] = byte(unit.Lv)
	copy(record[0x22:0x28], unit.NativeTransient[:])
	record[0x3b] = byte(unit.MV)
	record[0x3c] = byte(unit.Exp)
	putNativeItemPanelWord(record, 0x3e, unit.DX)
	putNativeItemPanelWord(record, 0x40, unit.HP)
	putNativeItemPanelWord(record, 0x42, unit.MaxHP)
	putNativeItemPanelWord(record, 0x44, unit.MP)
	putNativeItemPanelWord(record, 0x46, unit.MaxMP)
	putNativeItemPanelWord(record, 0x48, unit.AP)
	putNativeItemPanelWord(record, 0x4a, unit.DP)
	putNativeItemPanelWord(record, 0x4c, unit.HIT)
	putNativeItemPanelWord(record, 0x4e, unit.EV)
	return record, nil
}

func nativeRecordByte8ForUnit(unit *Unit) (byte, bool) {
	if unit == nil {
		return 0, false
	}
	if unit.HasNativeRecordByte8 {
		return unit.NativeRecordByte8, true
	}
	// Compatibility assets created before raw +8 was separated stored the
	// same proven player-persistent byte only as NativeIdentity. New runtime
	// restore paths never use this fallback.
	if unit.HasNativeIdentity && unit.NativeIdentity >= 0 && unit.NativeIdentity <= 0xff {
		return byte(unit.NativeIdentity), true
	}
	return 0, false
}

func putNativeItemPanelWord(record []byte, offset, value int) {
	binary.LittleEndian.PutUint16(record[offset:], uint16(int16(value)))
}
