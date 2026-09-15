package campaign

import (
	"encoding/json"
	"fmt"
	"os"

	"github.com/wicanr2/fd2_re/remake/internal/battle"
)

// ItemStats is the editable combat contribution copied from the EXE item table.
type ItemStats struct {
	Type int `json:"type"`
	AP   int `json:"ap"`
	HIT  int `json:"hit"`
	DP   int `json:"dp"`
	EV   int `json:"ev"`
	MV   int `json:"mv,omitempty"`
	Min  int `json:"range_min,omitempty"`
	Max  int `json:"range_max,omitempty"`
}

func LoadItemStats(path string) (map[int]ItemStats, error) {
	raw, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	var rows []struct {
		ID int `json:"id"`
		ItemStats
		Range [2]int `json:"range"`
	}
	if err := json.Unmarshal(raw, &rows); err != nil {
		return nil, err
	}
	out := make(map[int]ItemStats, len(rows))
	for _, row := range rows {
		if row.ID < 0 || row.ID > 0xff {
			return nil, fmt.Errorf("invalid item id %d", row.ID)
		}
		row.ItemStats.Min, row.ItemStats.Max = row.Range[0], row.Range[1]
		out[row.ID] = row.ItemStats
	}
	return out, nil
}

// RecomputeEquipment is the editable/normalized equipment projection. It is
// not byte-equivalent to 0x1b750 or 0x1145a: it also owns MV and attack range,
// accepts a typed map, and skips missing rows. Raw native transactions use
// battle.ApplyNativeRuntimeEquipmentRecalc or ApplyNativeEquipmentRecalc.
func RecomputeEquipment(u *battle.Unit, stats map[int]ItemStats) {
	if u == nil {
		return
	}
	if !u.EquipmentBaseSet {
		u.BaseAP, u.BaseDP, u.BaseHIT, u.BaseEV, u.BaseMV = u.AP, u.DP, u.HIT, u.EV, u.MV
		u.BaseAtkMin, u.BaseAtkMax, u.EquipmentBaseSet = u.AtkMin, u.AtkMax, true
	}
	u.AP, u.DP, u.HIT, u.EV, u.MV = u.BaseAP, u.BaseDP, u.BaseHIT, u.BaseEV, u.BaseMV
	u.AtkMin, u.AtkMax = u.BaseAtkMin, u.BaseAtkMax
	for i, equipped := range u.Equipped {
		if !equipped || i >= len(u.Inventory) {
			continue
		}
		item, ok := stats[u.Inventory[i]]
		if !ok {
			continue
		}
		u.AP += item.AP
		u.DP += item.DP
		u.HIT += item.HIT
		u.EV += item.EV
		u.MV += item.MV
		if item.Min > 0 {
			u.AtkMin = item.Min
		}
		if item.Max > 0 {
			u.AtkMax = item.Max
		}
	}
}

// ApplyEquippedAttackRange sets only the normalized attack range from the
// equipped weapon's item.json range (doc32 weapon_range.json), the same rule
// RecomputeEquipment applies after a shop purchase. Persistent 0x50-byte
// records carry no range field, so a roster restored from FD2.SAV (LOAD or
// CONTINUE) would otherwise fall back to the default 1 and a spear/bow unit
// could not attack at distance 2 as it does in the original. AP/DP/HIT/EV stay
// as the raw record wrote them; only AtkMin/AtkMax and their base are touched.
func ApplyEquippedAttackRange(u *battle.Unit, stats map[int]ItemStats) {
	if u == nil {
		return
	}
	u.AtkMin, u.AtkMax = 0, 0
	for i, equipped := range u.Equipped {
		if !equipped || i >= len(u.Inventory) {
			continue
		}
		item, ok := stats[u.Inventory[i]]
		if !ok {
			continue
		}
		if item.Min > 0 {
			u.AtkMin = item.Min
		}
		if item.Max > 0 {
			u.AtkMax = item.Max
		}
	}
	u.BaseAtkMin, u.BaseAtkMax = u.AtkMin, u.AtkMax
}

// RecomputeAfterClassChange is the normalized counterpart of the proven
// 0x31602 -> 0x1b750 handoff. It preserves editable campaign fields and avoids
// double-counting gear, but it is not evidence for the raw transient modifiers
// or the x87 rounding performed by 0x1b750.
func RecomputeAfterClassChange(u *battle.Unit, stats map[int]ItemStats) {
	if u == nil {
		return
	}
	baseAP, baseDP, baseMV := u.AP, u.DP, u.MV
	if u.EquipmentBaseSet {
		for i, equipped := range u.Equipped {
			if !equipped || i >= len(u.Inventory) {
				continue
			}
			item, ok := stats[u.Inventory[i]]
			if !ok {
				continue
			}
			baseAP -= item.AP
			baseDP -= item.DP
			baseMV -= item.MV
		}
	}
	// +0x3e is the raw DX/HIT/EV base; equipped item +HIT/+EV are then
	// accumulated by RecomputeEquipment.
	u.HIT, u.EV = u.DX, u.DX
	u.BaseAP, u.BaseDP = baseAP, baseDP
	u.BaseHIT, u.BaseEV = u.DX, u.DX
	u.BaseMV = baseMV
	u.BaseAtkMin, u.BaseAtkMax = u.AtkMin, u.AtkMax
	u.EquipmentBaseSet = true
	RecomputeEquipment(u, stats)
}

// InitializeEquipmentBase converts an authored effective stat line into the
// persistent base expected by 0x1145a by subtracting the source's equipped
// inventory contributions once. This is a normalized compatibility path, not
// the raw eight-cell 0x1145a transaction. Subsequent saves carry
// EquipmentBaseSet and never repeat this conversion.
func InitializeEquipmentBase(u *battle.Unit, stats map[int]ItemStats) {
	if u == nil || u.EquipmentBaseSet {
		return
	}
	u.BaseAP, u.BaseDP, u.BaseHIT, u.BaseEV, u.BaseMV = u.AP, u.DP, u.HIT, u.EV, u.MV
	u.BaseAtkMin, u.BaseAtkMax = u.AtkMin, u.AtkMax
	for i, equipped := range u.Equipped {
		if !equipped || i >= len(u.Inventory) {
			continue
		}
		item, ok := stats[u.Inventory[i]]
		if !ok {
			continue
		}
		u.BaseAP -= item.AP
		u.BaseDP -= item.DP
		u.BaseHIT -= item.HIT
		u.BaseEV -= item.EV
		u.BaseMV -= item.MV
	}
	u.EquipmentBaseSet = true
	RecomputeEquipment(u, stats)
}

// EquipItem applies the original 0x1c142 rule: weapon IDs below 0x80 replace
// an equipped weapon, while IDs >=0x80 replace an equipped armour/accessory.
func EquipItem(u *battle.Unit, slot int, stats map[int]ItemStats) error {
	if u == nil || slot < 0 || slot >= len(u.Inventory) {
		return fmt.Errorf("invalid equipment slot")
	}
	item, ok := stats[u.Inventory[slot]]
	if !ok || item.Type >= 0x20 {
		return fmt.Errorf("item is not equipment")
	}
	for len(u.Equipped) < len(u.Inventory) {
		u.Equipped = append(u.Equipped, false)
	}
	category := u.Inventory[slot] < 0x80
	for i := range u.Equipped {
		if i != slot && u.Equipped[i] && (u.Inventory[i] < 0x80) == category {
			u.Equipped[i] = false
		}
	}
	u.Equipped[slot] = true
	if len(u.InventorySlots) == 8 && len(u.NativeInventoryFlags) == 8 {
		compact := 0
		for rawSlot, itemID := range u.InventorySlots {
			if itemID == 0xff {
				continue
			}
			u.NativeInventoryFlags[rawSlot] = 0
			if compact < len(u.Equipped) && u.Equipped[compact] {
				u.NativeInventoryFlags[rawSlot] = 0x40
			}
			compact++
		}
	}
	RecomputeEquipment(u, stats)
	return nil
}

// EquipNativeCompactSlot applies 0x1c142 through the compact projection used
// by the remake, but only after proving that projection still matches the
// native eight raw cells consumed by 0x1b722/0x184c0. This prevents a raw slot
// index from being passed accidentally to EquipItem when editable data
// contains holes.
func EquipNativeCompactSlot(
	u *battle.Unit,
	compactSlot int,
	stats map[int]ItemStats,
) error {
	if u == nil || len(u.InventorySlots) != 8 ||
		len(u.NativeInventoryFlags) != 8 ||
		compactSlot < 0 || compactSlot >= len(u.Inventory) {
		return fmt.Errorf("invalid native equipment state")
	}
	rawForCompact := make([]int, 0, 8)
	for rawSlot, flag := range u.NativeInventoryFlags {
		if flag&0x80 != 0 {
			continue
		}
		compact := len(rawForCompact)
		if compact >= len(u.Inventory) ||
			u.InventorySlots[rawSlot] != u.Inventory[compact] ||
			compact >= len(u.Equipped) ||
			(flag&0x40 != 0) != u.Equipped[compact] {
			return fmt.Errorf("native equipment projection diverges at raw slot %d", rawSlot)
		}
		rawForCompact = append(rawForCompact, rawSlot)
	}
	if len(rawForCompact) != len(u.Inventory) ||
		len(u.Equipped) != len(u.Inventory) {
		return fmt.Errorf("native equipment projection length diverges")
	}

	staged := *u
	staged.Inventory = append([]int(nil), u.Inventory...)
	staged.Equipped = append([]bool(nil), u.Equipped...)
	staged.InventorySlots = append([]int(nil), u.InventorySlots...)
	staged.NativeInventoryFlags = append([]int(nil), u.NativeInventoryFlags...)
	if err := EquipItem(&staged, compactSlot, stats); err != nil {
		return err
	}
	// EquipItem also supports normalized units and therefore rebuilds native
	// flags when an eight-cell projection is present. Restore the original
	// reserved cells first: 0x1c142 writes only occupied raw cells and must not
	// erase an ignored cell's 0x80 marker merely because its stale item byte is
	// non-0xff.
	copy(staged.NativeInventoryFlags, u.NativeInventoryFlags)
	for compact, rawSlot := range rawForCompact {
		staged.NativeInventoryFlags[rawSlot] = 0
		if staged.Equipped[compact] {
			staged.NativeInventoryFlags[rawSlot] = 0x40
		}
	}
	*u = staged
	return nil
}
