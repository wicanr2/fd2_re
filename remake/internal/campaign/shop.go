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

// LoadItemPrices loads the editable list-price column used by the original
// shop sell table. SellGood applies the fixed 3/4 resale rule to this value.
func LoadItemPrices(path string) (map[int]int, error) {
	raw, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	var rows []struct {
		ID    int `json:"id"`
		Price int `json:"price"`
	}
	if err := json.Unmarshal(raw, &rows); err != nil {
		return nil, err
	}
	prices := make(map[int]int, len(rows))
	for _, row := range rows {
		if row.ID < 0 || row.ID > 0xff || row.Price < 0 {
			return nil, fmt.Errorf("invalid item price row id=%d price=%d", row.ID, row.Price)
		}
		prices[row.ID] = row.Price
	}
	return prices, nil
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
	if _, err := ReserveGood(gold, receiver, good); err != nil {
		return gold, err
	}
	return gold - good.Price, nil
}

// ReserveGood performs the original shop's inventory insertion before the
// optional equipment prompt. Gold is intentionally not deducted until the
// caller resolves that prompt via FinalizeGood.
func ReserveGood(gold int, receiver *battle.Unit, good Good) (int, error) {
	if receiver == nil {
		return -1, fmt.Errorf("shop receiver missing")
	}
	if good.ID < 0 || good.ID > 0xff || good.Price < 0 {
		return -1, fmt.Errorf("invalid shop good")
	}
	if gold < good.Price {
		return -1, fmt.Errorf("insufficient gold")
	}
	if len(receiver.Inventory) >= 8 {
		return -1, fmt.Errorf("inventory full")
	}
	receiver.Inventory = append(receiver.Inventory, good.ID)
	receiver.Equipped = append(receiver.Equipped, false)
	return len(receiver.Inventory) - 1, nil
}

// FinalizeGood deducts the already-confirmed good's price.
func FinalizeGood(gold int, good Good) int {
	return gold - good.Price
}

// SellGood mirrors the original 75%-of-list-price transaction. The slot is
// removed only after validation; its equipped marker follows the item out.
func SellGood(gold int, receiver *battle.Unit, itemID, listPrice int) (int, error) {
	if receiver == nil {
		return gold, fmt.Errorf("shop receiver missing")
	}
	if itemID < 0 || itemID > 0xff || listPrice < 0 {
		return gold, fmt.Errorf("invalid shop item")
	}
	for slot, id := range receiver.Inventory {
		if id == itemID {
			return SellSlot(gold, receiver, slot, listPrice)
		}
	}
	return gold, fmt.Errorf("item not found")
}

// SellSlot is the slot-addressed form used by the UI, so duplicate item IDs
// cannot cause the wrong inventory entry to be sold.
func SellSlot(gold int, receiver *battle.Unit, slot, listPrice int) (int, error) {
	if receiver == nil {
		return gold, fmt.Errorf("shop receiver missing")
	}
	if slot < 0 || slot >= len(receiver.Inventory) || listPrice < 0 {
		return gold, fmt.Errorf("invalid shop slot")
	}
	receiver.Inventory = append(receiver.Inventory[:slot], receiver.Inventory[slot+1:]...)
	if slot < len(receiver.Equipped) {
		receiver.Equipped = append(receiver.Equipped[:slot], receiver.Equipped[slot+1:]...)
	}
	return gold + listPrice*3/4, nil
}
