package campaign

import (
	"encoding/json"
	"fmt"
	"os"

	"github.com/wicanr2/fd2_re/remake/internal/battle"
)

type shopItemRow struct {
	ID   int `json:"id"`
	Type int `json:"type"`
}
type shopEquipRow struct {
	ClassID int   `json:"cls"`
	Types   []int `json:"types"`
}

// LoadShopEligibility loads the editable EXE-derived item type and class whitelist tables.
func LoadShopEligibility(itemPath, equipPath string) (map[int]int, map[int][]int, error) {
	itemRaw, err := os.ReadFile(itemPath)
	if err != nil {
		return nil, nil, err
	}
	equipRaw, err := os.ReadFile(equipPath)
	if err != nil {
		return nil, nil, err
	}
	var items []shopItemRow
	var rows []shopEquipRow
	if err := json.Unmarshal(itemRaw, &items); err != nil {
		return nil, nil, err
	}
	if err := json.Unmarshal(equipRaw, &rows); err != nil {
		return nil, nil, err
	}
	types, equip := map[int]int{}, map[int][]int{}
	for _, item := range items {
		if item.ID < 0 || item.ID > 0xff {
			return nil, nil, fmt.Errorf("invalid item id %d", item.ID)
		}
		types[item.ID] = item.Type
	}
	for _, row := range rows {
		if len(row.Types) != 6 {
			return nil, nil, fmt.Errorf("class %d has %d equipment slots", row.ClassID, len(row.Types))
		}
		equip[row.ClassID] = append([]int(nil), row.Types...)
	}
	return types, equip, nil
}

// CanEquip mirrors original 0x1c1c3: an equip item is allowed exactly when
// its item.type appears in this class record's six type slots (0xff is empty).
func CanEquip(classID, itemType int, classEquip map[int][]int) bool {
	for _, allowed := range classEquip[classID] {
		if allowed == itemType {
			return true
		}
	}
	return false
}

// BuyGood is the atomic part of original shop purchase: the selected receiver
// gets the item in its first free inventory slot, then (and only then) gold is
// deducted. Confirmation, eligible-recipient selection and equip prompting are
// UI concerns layered above this operation.
func BuyGood(gold int, receiver *battle.Unit, good Good) (int, error) {
	if receiver == nil {
		return gold, fmt.Errorf("shop receiver missing")
	}
	if good.ID < 0 || good.ID > 0xff || good.Price < 0 {
		return gold, fmt.Errorf("invalid shop good")
	}
	if gold < good.Price {
		return gold, fmt.Errorf("insufficient gold")
	}
	if len(receiver.Inventory) >= 8 {
		return gold, fmt.Errorf("inventory full")
	}
	receiver.Inventory = append(receiver.Inventory, good.ID)
	return gold - good.Price, nil
}
