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

// RecomputeEquipment mirrors EXE 0x1b750/0x1145a: start at the persistent
// base values and add only slots whose first byte has the equipped bit. The
// current scenario JSON already contains effective values, so those are
// intentionally captured as Base* when PartyUnits is materialized.
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

// InitializeEquipmentBase converts an authored effective stat line into the
// persistent base expected by 0x1145a by subtracting the source's equipped
// first-two inventory slots once. Subsequent saves carry EquipmentBaseSet and
// never repeat this conversion.
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
	RecomputeEquipment(u, stats)
	return nil
}
