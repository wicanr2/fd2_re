package campaign

import (
	"fmt"

	"github.com/wicanr2/fd2_re/remake/internal/battle"
)

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
