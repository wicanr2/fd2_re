package battle

import "fmt"

// ValidateNativeInventoryProjection proves that the compact editable arrays
// still represent exactly the occupied native cells in raw order. Ignored
// cells may retain stale item bytes; their signed flag remains authoritative.
func ValidateNativeInventoryProjection(unit *Unit) error {
	if unit == nil || len(unit.InventorySlots) != nativeInventoryCells ||
		len(unit.NativeInventoryFlags) != nativeInventoryCells ||
		len(unit.Inventory) != len(unit.Equipped) {
		return fmt.Errorf("native inventory projection is incomplete")
	}
	compact := 0
	for rawSlot, flag := range unit.NativeInventoryFlags {
		if flag&0x80 != 0 {
			continue
		}
		if compact >= len(unit.Inventory) ||
			unit.InventorySlots[rawSlot] != unit.Inventory[compact] ||
			(flag&0x40 != 0) != unit.Equipped[compact] {
			return fmt.Errorf(
				"native inventory projection diverges at raw slot %d",
				rawSlot,
			)
		}
		compact++
	}
	if compact != len(unit.Inventory) {
		return fmt.Errorf("native inventory projection length diverges")
	}
	return nil
}

// TransferNativeInventoryItem mirrors the mutation topology proven in the
// 0x2f8ea -> 0x1b8e7 -> 0x1bb8c branch.  The source item is removed from its
// compact inventory slot and inserted into the first empty destination cell;
// 0x1bb8c writes an un-equipped destination cell.  The caller must already
// have applied the native signed-flag eligibility filter (0x2f8ea's list
// builder); this function does not infer equipped state from an item ID.
//
// It is deliberately a raw operation rather than a named church service.
func TransferNativeInventoryItem(source *Unit, sourceIndex int, destination *Unit) error {
	if source == nil || destination == nil {
		return fmt.Errorf("native inventory transfer: missing unit")
	}
	if err := ValidateNativeInventoryProjection(source); err != nil {
		return err
	}
	if source != destination {
		if err := ValidateNativeInventoryProjection(destination); err != nil {
			return err
		}
	}
	if sourceIndex < 0 || sourceIndex >= len(source.Inventory) {
		return fmt.Errorf("native inventory transfer: source slot %d out of bounds", sourceIndex)
	}
	itemID := source.Inventory[sourceIndex]
	if itemID < 0 || itemID > 0xff {
		return fmt.Errorf("native inventory transfer: invalid item id %d", itemID)
	}
	// Preflight both fixed-slot layouts before changing either unit.  A full
	// destination is the native failure path and must not consume the source.
	free := false
	for _, flag := range destination.NativeInventoryFlags {
		if flag&0x80 != 0 {
			free = true
			break
		}
	}
	if !free || len(destination.Inventory) >= 8 {
		return fmt.Errorf("native inventory transfer: destination inventory full")
	}
	// Snapshot the compact and source-slot views so an unexpected insertion
	// failure remains atomic even if a future Unit representation changes.
	sourceInventory := append([]int(nil), source.Inventory...)
	sourceEquipped := append([]bool(nil), source.Equipped...)
	sourceSlots := append([]int(nil), source.InventorySlots...)
	sourceFlags := append([]int(nil), source.NativeInventoryFlags...)
	destinationInventory := append([]int(nil), destination.Inventory...)
	destinationEquipped := append([]bool(nil), destination.Equipped...)
	destinationSlots := append([]int(nil), destination.InventorySlots...)
	destinationFlags := append([]int(nil), destination.NativeInventoryFlags...)
	if !removeNativeCompactInventory(source, sourceIndex) || !destination.AddInventoryItem(itemID, false) {
		source.Inventory, source.Equipped, source.InventorySlots, source.NativeInventoryFlags = sourceInventory, sourceEquipped, sourceSlots, sourceFlags
		destination.Inventory, destination.Equipped, destination.InventorySlots, destination.NativeInventoryFlags = destinationInventory, destinationEquipped, destinationSlots, destinationFlags
		return fmt.Errorf("native inventory transfer: mutation failed")
	}
	return nil
}

// ReplaceNativeFullInventoryReward 對應已證實的死亡獎勵路徑
// 0x1B722 -> 0x1B8E7 -> 0x1BB8C：丟棄選定舊物品、後續格左移，再把新獎勵
// 以未裝備狀態附加到尾端。原版 caller 沒有隊友或目的角色，因此此操作刻意與
// TransferNativeInventoryItem 分開。
func ReplaceNativeFullInventoryReward(unit *Unit, selectedIndex, rewardItem int) error {
	if unit == nil || rewardItem < 0 || rewardItem > 0xff {
		return fmt.Errorf("native death reward replacement: invalid unit/item")
	}
	if err := ValidateNativeInventoryProjection(unit); err != nil {
		return fmt.Errorf("native death reward replacement: %w", err)
	}
	if len(unit.Inventory) != nativeInventoryCells || selectedIndex < 0 || selectedIndex >= nativeInventoryCells {
		return fmt.Errorf(
			"native death reward replacement: full selection %d/%d is invalid",
			selectedIndex, len(unit.Inventory),
		)
	}
	for slot, flag := range unit.NativeInventoryFlags {
		if flag&0x80 != 0 {
			return fmt.Errorf("native death reward replacement: raw slot %d is not occupied", slot)
		}
	}
	inventory := append([]int(nil), unit.Inventory...)
	equipped := append([]bool(nil), unit.Equipped...)
	slots := append([]int(nil), unit.InventorySlots...)
	flags := append([]int(nil), unit.NativeInventoryFlags...)
	if !removeNativeCompactInventory(unit, selectedIndex) || !unit.AddInventoryItem(rewardItem, false) {
		unit.Inventory, unit.Equipped = inventory, equipped
		unit.InventorySlots, unit.NativeInventoryFlags = slots, flags
		return fmt.Errorf("native death reward replacement: mutation failed")
	}
	return nil
}

func firstInventoryHole(slots []int) int {
	for i, item := range slots {
		if item == 0xff {
			return i
		}
	}
	return -1
}

// removeNativeCompactInventory combines the compact editable view with the
// fixed-cell shift performed by native 0x1b8e7.  The latter does not leave a
// hole: cells after the removed slot move left and the tail is marked 0x80.
func removeNativeCompactInventory(u *Unit, compactIndex int) bool {
	if u == nil || compactIndex < 0 || compactIndex >= len(u.Inventory) {
		return false
	}
	u.normalizeInventorySlots()
	seen, slot := 0, -1
	for i, id := range u.InventorySlots {
		if id != 0xff {
			if seen == compactIndex {
				slot = i
				break
			}
			seen++
		}
	}
	if slot < 0 {
		return false
	}
	copy(u.InventorySlots[slot:], u.InventorySlots[slot+1:])
	u.InventorySlots[len(u.InventorySlots)-1] = 0xff
	if len(u.NativeInventoryFlags) == nativeInventoryCells {
		copy(u.NativeInventoryFlags[slot:], u.NativeInventoryFlags[slot+1:])
		u.NativeInventoryFlags[len(u.NativeInventoryFlags)-1] = 0x80
	}
	u.Inventory = append(u.Inventory[:compactIndex], u.Inventory[compactIndex+1:]...)
	if compactIndex < len(u.Equipped) {
		u.Equipped = append(u.Equipped[:compactIndex], u.Equipped[compactIndex+1:]...)
	}
	return true
}

// RemoveNativeCompactInventory exposes the 0x1b8e7 compact-to-raw adapter to
// facility transaction owners. Exact eight-cell provenance is mandatory;
// unlike Unit.RemoveInventoryIndex this shifts later raw cells left.
func RemoveNativeCompactInventory(u *Unit, compactIndex int) error {
	if u == nil || len(u.InventorySlots) != nativeInventoryCells ||
		len(u.NativeInventoryFlags) != nativeInventoryCells {
		return fmt.Errorf("native inventory remove: missing eight-cell provenance")
	}
	if !removeNativeCompactInventory(u, compactIndex) {
		return fmt.Errorf(
			"native inventory remove: compact slot %d is invalid",
			compactIndex,
		)
	}
	return nil
}
