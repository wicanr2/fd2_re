package main

import (
	"bytes"
	"encoding/binary"
	"fmt"
	"image"
	"image/color"
	"os"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/wicanr2/fd2_re/remake/internal/battle"
	"github.com/wicanr2/fd2_re/remake/internal/campaign"
)

func nativeOriginalArchivePath(environment, name string) string {
	if path := os.Getenv(environment); path != "" {
		return path
	}
	path := assetPath("assets/original/" + name)
	if fileExists(path) {
		return path
	}
	return ""
}

func nativeFDTXTPath() string {
	return nativeOriginalArchivePath("FD2_ORIGINAL_FDTXT", "FDTXT.DAT")
}

func nativeDATOPath() string {
	return nativeOriginalArchivePath("FD2_ORIGINAL_DATO", "DATO.DAT")
}

func nativeItemRawSlots(unit *battle.Unit) []int {
	if unit == nil || len(unit.InventorySlots) != 8 || len(unit.NativeInventoryFlags) != 8 {
		return nil
	}
	slots := make([]int, 0, 8)
	for slot := 0; slot < 8; slot++ {
		if unit.NativeInventoryFlags[slot]&0x80 == 0 {
			slots = append(slots, slot)
		}
	}
	return slots
}

func (g *Game) prepareNativeItemPanel(unit *battle.Unit) bool {
	return g.prepareNativeItemPanelMode(unit, false)
}

func (g *Game) prepareNativeItemPanelMode(unit *battle.Unit, allowEmpty bool) bool {
	g.clearNativeItemPanel()
	fdotherPath, fdtxtPath, datoPath := nativeFDOTHERPath(), nativeFDTXTPath(), nativeDATOPath()
	if fdotherPath == "" || fdtxtPath == "" || datoPath == "" || len(g.nativeUIPalette) < 256 {
		return false
	}
	record, err := battle.NativeItemPanelRecordForUnit(unit)
	if err != nil {
		return false
	}
	pixels := make([]byte, 320*200)
	if err := battle.RenderNativeItemPanelResources(fdotherPath, fdtxtPath, datoPath, record, pixels); err != nil {
		return false
	}
	assets, err := battle.LoadNativeItemPanelDataAssets(fdotherPath, fdtxtPath)
	if err != nil {
		return false
	}
	rows, err := battle.LoadNativeItemEffectRowPrefix(assetPath("assets/data/native_item_effect_rows.json"))
	if err != nil {
		return false
	}
	g.nativeItemPanelBase = pixels
	g.nativeItemPanelRecord = record
	g.nativeItemPanelAssets = &assets
	g.nativeItemEffectRows = rows
	return g.refreshNativeItemPanelMode(unit, allowEmpty)
}

func (g *Game) refreshNativeItemPanel(unit *battle.Unit) bool {
	return g.refreshNativeItemPanelMode(unit, false)
}

func (g *Game) refreshNativeItemPanelMode(unit *battle.Unit, allowEmpty bool) bool {
	if len(g.nativeItemPanelBase) != 320*200 || len(g.nativeItemPanelRecord) != 80 ||
		g.nativeItemPanelAssets == nil || len(g.nativeItemEffectRows) == 0 {
		return false
	}
	rawSlots := nativeItemRawSlots(unit)
	if len(rawSlots) == 0 {
		if !allowEmpty {
			return false
		}
		pixels := append([]byte(nil), g.nativeItemPanelBase...)
		if err := battle.RenderNativeItemPanelRows(
			*g.nativeItemPanelAssets, g.nativeItemPanelRecord,
			-1, g.nativeItemEffectRows, pixels,
		); err != nil {
			return false
		}
		return g.setNativeItemPanelPixels(pixels)
	}
	if g.itemSel < 0 {
		g.itemSel = 0
	}
	if g.itemSel >= len(rawSlots) {
		g.itemSel = len(rawSlots) - 1
	}
	pixels := append([]byte(nil), g.nativeItemPanelBase...)
	if err := battle.RenderNativeItemPanelRows(
		*g.nativeItemPanelAssets, g.nativeItemPanelRecord,
		rawSlots[g.itemSel], g.nativeItemEffectRows, pixels,
	); err != nil {
		return false
	}
	return g.setNativeItemPanelPixels(pixels)
}

func (g *Game) setNativeItemPanelPixels(pixels []byte) bool {
	if len(pixels) != 320*200 || len(g.nativeUIPalette) < 256 {
		return false
	}
	palette := append(color.Palette(nil), g.nativeUIPalette...)
	palette[0] = color.NRGBA{A: 0xff}
	frame := image.NewPaletted(image.Rect(0, 0, 320, 200), palette)
	copy(frame.Pix, pixels)
	g.nativeItemPanel = ebiten.NewImageFromImage(frame)
	return true
}

// rebuildNativeItemPanelContents mirrors 0x1c084..0x1c0cc after a successful
// standalone equip: rebuild the status/item buffers in place without replaying
// the 12-frame opening.
func (g *Game) rebuildNativeItemPanelContents(unit *battle.Unit, allowEmpty bool) bool {
	fdotherPath, fdtxtPath, datoPath := nativeFDOTHERPath(), nativeFDTXTPath(), nativeDATOPath()
	if fdotherPath == "" || fdtxtPath == "" || datoPath == "" {
		return false
	}
	record, err := battle.NativeItemPanelRecordForUnit(unit)
	if err != nil {
		return false
	}
	pixels := make([]byte, 320*200)
	if err := battle.RenderNativeItemPanelResources(
		fdotherPath, fdtxtPath, datoPath, record, pixels,
	); err != nil {
		return false
	}
	g.nativeItemPanelRecord = record
	g.nativeItemPanelBase = pixels
	return g.refreshNativeItemPanelMode(unit, allowEmpty)
}

func (g *Game) clearNativeItemPanel() {
	g.nativeItemPanel = nil
	g.nativeItemPanelBase = nil
	g.nativeItemPanelRecord = nil
	g.nativeItemPanelAssets = nil
	g.nativeItemEffectRows = nil
	g.itemAnimStep = 0
	g.itemClosing = false
}

// stepNativeItemPanelAnimation returns true while input must remain blocked.
func (g *Game) stepNativeItemPanelAnimation() bool {
	if g.nativeItemPanel == nil {
		return false
	}
	if g.itemClosing {
		if g.itemAnimStep < 11 {
			g.itemAnimStep++
			return true
		}
		g.itemOpen = false
		g.beginActionOverlayOpen(g.ringSel)
		g.clearNativeItemPanel()
		return true
	}
	if g.itemAnimStep < 11 {
		g.itemAnimStep++
		return true
	}
	return false
}

func (g *Game) beginNativeItemPanelClose() {
	if g.nativeItemPanel == nil {
		g.itemOpen = false
		g.beginActionOverlayOpen(g.ringSel)
		return
	}
	g.itemClosing = true
	g.itemAnimStep = 0
}

// applyNativeImmediateItem executes only the fully closed, non-RNG type
// 8/9/10 transaction. Their tracked rows use selection/effect mode zero, so
// actor confirmation is still validated through the recovered two-stage
// target planner instead of being assumed from the item name.
func (g *Game) applyNativeImmediateItem(rawSlot, itemID int) (bool, error) {
	if g == nil || g.st == nil || g.sel == nil {
		return false, fmt.Errorf("native item transaction context is unavailable")
	}
	rowOffset, err := battle.NativeItemEffectRowOffset(itemID)
	if err != nil || rowOffset+battle.NativeItemEffectRowSize > len(g.nativeItemEffectRows) {
		return false, fmt.Errorf("native item row %d is unavailable", itemID)
	}
	row := g.nativeItemEffectRows[rowOffset : rowOffset+battle.NativeItemEffectRowSize]
	wordRoute, wordSupported := battle.NativeItemWordDeltaRouteForType(int(row[0x0d]))
	delta := binary.LittleEndian.Uint16(row[0x0e:0x10])
	capacityRoute, capacitySupported := battle.NativeItemCapacityStepRouteForType(row[0x0d], delta)
	if !wordSupported && !capacitySupported {
		return false, nil
	}
	plan, err := battle.NativeItemTargetPlanFromRow(row)
	if err != nil {
		return false, err
	}
	flags, err := g.st.NativeCommandBaseFlags()
	if err != nil {
		return false, err
	}
	targets, err := battle.NativeItemEffectTargets(
		g.st.W, g.st.H, g.sel, g.sel, plan,
		flags, g.st.Units,
	)
	if err != nil {
		return false, err
	}
	if len(targets) != 1 || targets[0] != g.sel {
		return false, fmt.Errorf("native immediate item target list is not actor-only")
	}
	if wordSupported {
		if g.shopItemStats == nil {
			return false, fmt.Errorf("native equipment recomputation table is unavailable")
		}
		for index, equipped := range g.sel.Equipped {
			if equipped && index < len(g.sel.Inventory) {
				if _, ok := g.shopItemStats[g.sel.Inventory[index]]; !ok {
					return false, fmt.Errorf("native equipped item %d is absent from recomputation table", g.sel.Inventory[index])
				}
			}
		}
		if _, err := battle.ApplyNativeItemBaseStatDeltaToUnit(
			g.sel, g.sel, wordRoute, delta, rawSlot,
		); err != nil {
			return false, err
		}
		campaign.RecomputeEquipment(g.sel, g.shopItemStats)
	} else {
		if _, err := battle.ApplyNativeItemCapacityToUnit(
			g.sel, g.sel, capacityRoute, rawSlot,
		); err != nil {
			return false, err
		}
	}
	// 0x1bbdc calls 0x13512 immediately after successful 0x20c6f.
	actor := g.sel
	actor.NativeRecordByte5 |= 0x80
	actor.HasNativeRecordByte5 = true
	g.finishSuccessfulUnitAction(actor, func() {
		g.itemOpen, g.ring = false, false
		g.clearNativeItemPanel()
		g.sel, g.reach, g.moved = nil, nil, false
	})
	return true, nil
}

// beginNativeTargetItem enters the recovered first-stage 0x14818 selector
// only for effect families with a closed raw transaction. The mutation stays
// deferred until a concrete runtime unit passes both target-planner stages.
func (g *Game) beginNativeTargetItem(rawSlot, itemID int) (bool, error) {
	rowOffset, err := battle.NativeItemEffectRowOffset(itemID)
	if err != nil || rowOffset+battle.NativeItemEffectRowSize > len(g.nativeItemEffectRows) {
		return false, fmt.Errorf("native item row %d is unavailable", itemID)
	}
	row := g.nativeItemEffectRows[rowOffset : rowOffset+battle.NativeItemEffectRowSize]
	amount := binary.LittleEndian.Uint16(row[0x0e:0x10])
	_, hp := battle.NativeItemHPRestoreRouteForType(row[0x0d], amount)
	_, mp := battle.NativeItemMPRestoreRouteForType(row[0x0d], amount)
	_, markerClear := battle.NativeItemMarkerClearRestoreRouteForType(row[0x0d])
	_, hitEV := battle.NativeItemHITEVStepRouteForType(row[0x0d])
	_, apDP := battle.NativeItemAPDPStepRouteForType(row[0x0d])
	_, markerApply := battle.NativeItemMarkerApplicationRouteForType(row[0x0d])
	_, commandDamage := battle.NativeItemCommandDamageRouteForType(row[0x0d], amount)
	_, relocation := battle.NativeItemRelocationRouteForType(row[0x0d], amount)
	if !hp && !mp && !markerClear && !hitEV && !apDP && !markerApply && !commandDamage && !relocation {
		return false, nil
	}
	plan, err := battle.NativeItemTargetPlanFromRow(row)
	if err != nil {
		return false, err
	}
	flags, err := g.st.NativeCommandBaseFlags()
	if err != nil {
		return false, err
	}
	fieldBytes, err := battle.NativeCommandTargetFieldBytes(
		g.st.W, g.st.H, battle.Cell{X: g.sel.X, Y: g.sel.Y},
		plan.SelectionMode, plan.SelectionInnerMark, flags,
	)
	if err != nil || len(g.st.NativeTileBlitModes) != len(fieldBytes) {
		return false, fmt.Errorf("native item target field is unavailable")
	}
	// 未修改第一戰的一般玩家證據顯示：四名我方都滿 HP 時，type 5
	// 草藥確認後仍留在 caller-owned item panel，不發布 target modal。
	// 只對具共同 raw restore transaction 的 type 5 套用這個窄 gate；
	// type 13 與其他物品沒有同等動態證據，不能外推。
	if hp && row[0x0d] == 5 {
		targets, err := battle.NativeAttackCandidates(
			g.st.W, g.st.H, battle.Cell{X: g.sel.X, Y: g.sel.Y},
			plan.SelectionMode, plan.SelectionInnerMark, plan.TargetCode,
			flags, g.st.Units,
		)
		if err != nil {
			return false, err
		}
		hasApplicableTarget := false
		for _, target := range targets {
			if target != nil && target.HP < target.MaxHP {
				hasApplicableTarget = true
				break
			}
		}
		if !hasApplicableTarget {
			return false, nil
		}
	}
	if plan.EffectMode < 0 || plan.EffectMode > 0xff ||
		!g.st.MaterializeNativeMapRangeMode(
			battle.NativeMapOverlaySelectorFromRecordByte(byte(plan.EffectMode)),
		) {
		return false, fmt.Errorf("native item overlay selector is invalid")
	}
	copy(g.st.NativeTileBlitModes, fieldBytes)
	g.nativeItemTargeting = true
	g.nativeItemTargetID = itemID
	g.nativeItemTargetRawSlot = rawSlot
	g.itemOpen = false
	rows := g.nativeItemEffectRows
	g.clearNativeItemPanel()
	g.nativeItemEffectRows = rows
	g.curX, g.curY = g.sel.X, g.sel.Y
	g.reach = nil
	return true, nil
}

func (g *Game) nativeItemSelectionTargets() []*battle.Unit {
	if g == nil || g.st == nil || g.sel == nil || !g.nativeItemTargeting {
		return nil
	}
	rowOffset, err := battle.NativeItemEffectRowOffset(g.nativeItemTargetID)
	if err != nil || rowOffset+battle.NativeItemEffectRowSize > len(g.nativeItemEffectRows) {
		return nil
	}
	plan, err := battle.NativeItemTargetPlanFromRow(
		g.nativeItemEffectRows[rowOffset : rowOffset+battle.NativeItemEffectRowSize],
	)
	if err != nil {
		return nil
	}
	flags, err := g.st.NativeCommandBaseFlags()
	if err != nil {
		return nil
	}
	targets, err := battle.NativeAttackCandidates(
		g.st.W, g.st.H, battle.Cell{X: g.sel.X, Y: g.sel.Y},
		plan.SelectionMode, plan.SelectionInnerMark, plan.TargetCode,
		flags, g.st.Units,
	)
	if err != nil {
		return nil
	}
	return targets
}

func nativeItemRuntimeRecords(units []*battle.Unit) ([]byte, error) {
	records := make([]byte, 0, len(units)*80)
	for index, unit := range units {
		record, err := battle.NativeItemPanelRecordForUnit(unit)
		if err != nil {
			return nil, fmt.Errorf("runtime unit %d lacks native item record: %w", index, err)
		}
		if unit.X < 0 || unit.X > 0xff || unit.Y < 0 || unit.Y > 0xff ||
			!unit.HasNativeRecordByte5 {
			return nil, fmt.Errorf("runtime unit %d lacks native coordinate/activity provenance", index)
		}
		record[0], record[1], record[5] = byte(unit.X), byte(unit.Y), unit.NativeRecordByte5
		records = append(records, record...)
	}
	return records, nil
}

func syncNativeItemRuntimeRecord(unit *battle.Unit, record []byte) {
	unit.NativeRecordByte5 = record[5]
	unit.HasNativeRecordByte5 = true
	unit.HP = int(int16(binary.LittleEndian.Uint16(record[0x40:0x42])))
	unit.MP = int(int16(binary.LittleEndian.Uint16(record[0x44:0x46])))
	unit.SetMapPlacement(int(record[0]), int(record[1]), unit.Dir)
	unit.AP = int(int16(binary.LittleEndian.Uint16(record[0x48:0x4a])))
	unit.DP = int(int16(binary.LittleEndian.Uint16(record[0x4a:0x4c])))
	unit.HIT = int(int16(binary.LittleEndian.Uint16(record[0x4c:0x4e])))
	unit.EV = int(int16(binary.LittleEndian.Uint16(record[0x4e:0x50])))
	copy(unit.NativeTransient[:], record[0x22:0x28])
	unit.InventorySlots = make([]int, 8)
	unit.NativeInventoryFlags = make([]int, 8)
	unit.Inventory = unit.Inventory[:0]
	unit.Equipped = unit.Equipped[:0]
	for slot := 0; slot < 8; slot++ {
		flag, item := int(record[0x0a+slot*2]), int(record[0x0b+slot*2])
		unit.NativeInventoryFlags[slot], unit.InventorySlots[slot] = flag, item
		if flag&0x80 == 0 {
			unit.Inventory = append(unit.Inventory, item)
			unit.Equipped = append(unit.Equipped, flag&0x40 != 0)
		}
	}
}

// applyNativeAITargetItem consumes the complete 0x1567e winner list at the
// 0x15055 boundary. It deliberately bypasses the player-owned cursor/modal:
// the original AI owner rebuilds the whole list from its saved destination
// and passes that list to 0x20c6f. All mutable records stay detached until
// item, target and effect provenance have passed validation.
type nativeAIItemTransaction struct {
	before      []byte
	after       []byte
	targets     []*battle.Unit
	restore     *battle.NativeRawRestoreBatch
	damage      []battle.NativeCommandDamage
	damageRoute *battle.NativeItemCommandDamageRoute
	rngBefore   uint16
	rngAfter    uint16
}

func (g *Game) planNativeAITargetItem(plan *battle.AIPlan) (*nativeAIItemTransaction, error) {
	if g == nil || g.st == nil || plan == nil || plan.U == nil ||
		plan.NativeActionKind != battle.NativeAIActionItem || len(plan.NativeItemTargetIndices) == 0 {
		return nil, fmt.Errorf("native AI item target-list context is unavailable")
	}
	actorIndex := -1
	for index, unit := range g.st.Units {
		if unit == plan.U {
			actorIndex = index
			break
		}
	}
	if actorIndex < 0 || plan.NativeItemSlot < 0 || plan.NativeItemSlot >= 8 ||
		len(plan.U.InventorySlots) != 8 || len(plan.U.NativeInventoryFlags) != 8 ||
		plan.U.NativeInventoryFlags[plan.NativeItemSlot]&0x80 != 0 ||
		plan.U.InventorySlots[plan.NativeItemSlot] != plan.NativeItemID {
		return nil, fmt.Errorf("native AI item source provenance is unavailable")
	}
	rowOffset, err := battle.NativeItemEffectRowOffset(plan.NativeItemID)
	if err != nil || rowOffset+battle.NativeItemEffectRowSize > len(g.nativeItemEffectRows) {
		return nil, fmt.Errorf("native AI item row %d is unavailable", plan.NativeItemID)
	}
	row := g.nativeItemEffectRows[rowOffset : rowOffset+battle.NativeItemEffectRowSize]
	targets := make([]*battle.Unit, len(plan.NativeItemTargetIndices))
	seen := make(map[byte]bool, len(targets))
	for index, rawIndex := range plan.NativeItemTargetIndices {
		if seen[rawIndex] || int(rawIndex) >= len(g.st.Units) || g.st.Units[rawIndex] == nil {
			return nil, fmt.Errorf("native AI item target index %d is invalid", rawIndex)
		}
		seen[rawIndex] = true
		targets[index] = g.st.Units[rawIndex]
	}
	if targets[0] != plan.Target {
		return nil, fmt.Errorf("native AI item first target does not match movement plan")
	}
	records, err := nativeItemRuntimeRecords(g.st.Units)
	if err != nil {
		return nil, err
	}
	before := append([]byte(nil), records...)
	tx := &nativeAIItemTransaction{
		before: before, targets: targets,
		rngBefore: g.nativeRNGState, rngAfter: g.nativeRNGState,
	}
	amount := binary.LittleEndian.Uint16(row[0x0e:0x10])
	if route, ok := battle.NativeItemHPRestoreRouteForType(row[0x0d], amount); ok {
		result, err := battle.ApplyNativeItemHPRestore(
			records, plan.NativeItemTargetIndices, route, g.nativeRNGState,
			actorIndex, plan.NativeItemSlot,
		)
		if err != nil {
			return nil, err
		}
		tx.restore = &result
		tx.rngAfter = result.RNGState
	} else if route, ok := battle.NativeItemCommandDamageRouteForType(row[0x0d], amount); ok {
		// The shared command helper mutates Unit HP. Use detached copies so a
		// later validation error cannot partially publish the multi-target list.
		clones := make([]*battle.Unit, len(targets))
		for index, target := range targets {
			clone := *target
			clones[index] = &clone
		}
		results, state, err := battle.ApplyNativeItemCommandDamage(
			clones, route, g.nativeCommandBook, g.nativeCommandResistances,
			g.nativeRNGState,
		)
		if err != nil {
			return nil, err
		}
		tx.damage, tx.rngAfter = results, state
		tx.damageRoute = &route
		for index, rawIndex := range plan.NativeItemTargetIndices {
			base := int(rawIndex) * 80
			records[base+5] = clones[index].NativeRecordByte5
			binary.LittleEndian.PutUint16(records[base+0x40:base+0x42], uint16(int16(clones[index].HP)))
		}
	} else {
		return nil, fmt.Errorf("native AI item type %d has no positive-score transaction owner", row[0x0d])
	}
	tx.after = records
	return tx, nil
}

func (g *Game) commitNativeAIItemTransaction(tx *nativeAIItemTransaction) error {
	if g == nil || g.st == nil || tx == nil || len(tx.before) == 0 || len(tx.after) != len(tx.before) {
		return fmt.Errorf("native AI item transaction is unavailable")
	}
	current, err := nativeItemRuntimeRecords(g.st.Units)
	if err != nil {
		return err
	}
	if !bytes.Equal(current, tx.before) || g.nativeRNGState != tx.rngBefore {
		return fmt.Errorf("native AI item source state changed before publication")
	}
	for index, unit := range g.st.Units {
		syncNativeItemRuntimeRecord(unit, tx.after[index*80:(index+1)*80])
	}
	g.nativeRNGState = tx.rngAfter
	return nil
}

func (g *Game) applyNativeAITargetItem(plan *battle.AIPlan) error {
	tx, err := g.planNativeAITargetItem(plan)
	if err != nil {
		return err
	}
	return g.commitNativeAIItemTransaction(tx)
}

// applyNativeTargetItem commits the closed targeted families using original
// raw target-list order and the shared process-lifetime 16-bit RNG state.
func (g *Game) applyNativeTargetItem(confirmed *battle.Unit) (bool, error) {
	if g == nil || g.st == nil || g.sel == nil || !g.nativeItemTargeting || confirmed == nil {
		return false, nil
	}
	rowOffset, err := battle.NativeItemEffectRowOffset(g.nativeItemTargetID)
	if err != nil || rowOffset+battle.NativeItemEffectRowSize > len(g.nativeItemEffectRows) {
		return false, fmt.Errorf("native item row %d is unavailable", g.nativeItemTargetID)
	}
	row := g.nativeItemEffectRows[rowOffset : rowOffset+battle.NativeItemEffectRowSize]
	plan, err := battle.NativeItemTargetPlanFromRow(row)
	if err != nil {
		return false, err
	}
	flags, err := g.st.NativeCommandBaseFlags()
	if err != nil {
		return false, err
	}
	targets, err := battle.NativeItemEffectTargets(
		g.st.W, g.st.H, g.sel, confirmed, plan,
		flags, g.st.Units,
	)
	if err != nil {
		return false, nil
	}
	// 0x1bd5e..0x1bd6c resets byte+3 immediately after the first selector
	// returns, then restores the global overlay selector to one before the
	// second target list and the type23 destination selector.
	g.resetNativeTargetField()
	g.st.MaterializeNativeMapRangeMode(1)
	sourceUnit := -1
	targetIndices := make([]byte, len(targets))
	for index, unit := range g.st.Units {
		if unit == g.sel {
			sourceUnit = index
		}
		for targetIndex, target := range targets {
			if unit == target {
				targetIndices[targetIndex] = byte(index)
			}
		}
	}
	if sourceUnit < 0 {
		return false, fmt.Errorf("native item source is absent from runtime roster")
	}
	records, err := nativeItemRuntimeRecords(g.st.Units)
	if err != nil {
		return false, err
	}
	amount := binary.LittleEndian.Uint16(row[0x0e:0x10])
	nextRNG := g.nativeRNGState
	if route, ok := battle.NativeItemHPRestoreRouteForType(row[0x0d], amount); ok {
		result, err := battle.ApplyNativeItemHPRestore(
			records, targetIndices, route, g.nativeRNGState,
			sourceUnit, g.nativeItemTargetRawSlot,
		)
		if err != nil {
			return false, err
		}
		nextRNG = result.RNGState
	} else if route, ok := battle.NativeItemMPRestoreRouteForType(row[0x0d], amount); ok {
		result, err := battle.ApplyNativeItemMPRestore(
			records, targetIndices, route, g.nativeRNGState,
			sourceUnit, g.nativeItemTargetRawSlot,
		)
		if err != nil {
			return false, err
		}
		nextRNG = result.RNGState
	} else if route, ok := battle.NativeItemMarkerClearRestoreRouteForType(row[0x0d]); ok {
		_, state, _, err := battle.ApplyNativeItemMarkerClearRestore(
			records, targetIndices, route, g.nativeRNGState,
			sourceUnit, g.nativeItemTargetRawSlot,
		)
		if err != nil {
			return false, err
		}
		nextRNG = state
	} else if route, ok := battle.NativeItemHITEVStepRouteForType(row[0x0d]); ok {
		_, state, _, err := battle.ApplyNativeItemHITEVStep(
			records, targetIndices, route, g.nativeRNGState,
		)
		if err != nil {
			return false, err
		}
		nextRNG = state
	} else if route, ok := battle.NativeItemAPDPStepRouteForType(row[0x0d]); ok {
		_, state, _, err := battle.ApplyNativeItemAPDPStep(
			records, targetIndices, route, g.nativeRNGState,
		)
		if err != nil {
			return false, err
		}
		nextRNG = state
	} else if route, ok := battle.NativeItemMarkerApplicationRouteForType(row[0x0d]); ok {
		_, state, _, err := battle.ApplyNativeItemMarkerApplication(
			records, targetIndices, route, g.nativeRNGState,
		)
		if err != nil {
			return false, err
		}
		nextRNG = state
	} else if route, ok := battle.NativeItemCommandDamageRouteForType(row[0x0d], amount); ok {
		_, state, err := battle.ApplyNativeItemCommandDamage(
			targets, route, g.nativeCommandBook,
			g.nativeCommandResistances, g.nativeRNGState,
		)
		if err != nil {
			return false, err
		}
		nextRNG = state
		for index, unit := range g.st.Units {
			binary.LittleEndian.PutUint16(
				records[index*80+0x40:index*80+0x42],
				uint16(int16(unit.HP)),
			)
		}
	} else if _, ok := battle.NativeItemRelocationRouteForType(row[0x0d], amount); ok {
		if len(targetIndices) == 0 {
			return false, fmt.Errorf("native relocation has no target")
		}
		rows, err := battle.LoadNativeMovementCostRows(
			assetPath("assets/data/native_movement_cost_rows.json"),
		)
		if err != nil {
			return false, err
		}
		g.nativeMovementCostRows = rows
		g.nativeItemRelocationUnit = int(targetIndices[0])
		g.nativeItemTargeting = false
		g.nativeItemRelocating = true
		target := g.st.Units[g.nativeItemRelocationUnit]
		g.curX, g.curY = target.X, target.Y
		return false, nil
	} else {
		return false, nil
	}
	for index, unit := range g.st.Units {
		syncNativeItemRuntimeRecord(unit, records[index*80:(index+1)*80])
	}
	g.nativeRNGState = nextRNG
	actor := g.sel
	actor.NativeRecordByte5 |= 0x80
	actor.HasNativeRecordByte5 = true
	g.finishSuccessfulUnitAction(actor, func() {
		g.nativeItemTargeting = false
		g.nativeItemEffectRows = nil
		g.sel, g.reach, g.moved = nil, nil, false
	})
	return true, nil
}

func (g *Game) nativeRelocationDestinationAllowed(x, y int) (bool, error) {
	if g == nil || g.st == nil || !g.nativeItemRelocating ||
		x < 0 || y < 0 || x >= g.st.W || y >= g.st.H ||
		len(g.st.NativeTerrainMoveCodes) != g.st.W*g.st.H {
		return false, fmt.Errorf("native relocation terrain provenance is unavailable")
	}
	records, err := nativeItemRuntimeRecords(g.st.Units)
	if err != nil {
		return false, err
	}
	return battle.NativeRelocationDestinationAllowed(
		records, len(g.st.Units), g.nativeItemRelocationUnit,
		byte(x), byte(y), int(g.st.NativeTerrainMoveCodes[y*g.st.W+x]),
		g.nativeMovementCostRows,
	)
}

func (g *Game) nativeRelocationDestinations() map[battle.Cell]bool {
	result := map[battle.Cell]bool{}
	if g == nil || g.st == nil || !g.nativeItemRelocating ||
		len(g.st.NativeTerrainMoveCodes) != g.st.W*g.st.H {
		return result
	}
	records, err := nativeItemRuntimeRecords(g.st.Units)
	if err != nil {
		return result
	}
	for y := 0; y < g.st.H; y++ {
		for x := 0; x < g.st.W; x++ {
			allowed, err := battle.NativeRelocationDestinationAllowed(
				records, len(g.st.Units), g.nativeItemRelocationUnit,
				byte(x), byte(y), int(g.st.NativeTerrainMoveCodes[y*g.st.W+x]),
				g.nativeMovementCostRows,
			)
			if err == nil && allowed {
				result[battle.Cell{X: x, Y: y}] = true
			}
		}
	}
	return result
}

func (g *Game) applyNativeRelocationDestination(x, y int) (bool, error) {
	allowed, err := g.nativeRelocationDestinationAllowed(x, y)
	if err != nil || !allowed {
		return false, err
	}
	records, err := nativeItemRuntimeRecords(g.st.Units)
	if err != nil {
		return false, err
	}
	sourceUnit := -1
	for index, unit := range g.st.Units {
		if unit == g.sel {
			sourceUnit = index
			break
		}
	}
	if sourceUnit < 0 {
		return false, fmt.Errorf("native relocation source is absent")
	}
	rowOffset, err := battle.NativeItemEffectRowOffset(g.nativeItemTargetID)
	if err != nil {
		return false, err
	}
	row := g.nativeItemEffectRows[rowOffset : rowOffset+battle.NativeItemEffectRowSize]
	route, ok := battle.NativeItemRelocationRouteForType(
		row[0x0d], binary.LittleEndian.Uint16(row[0x0e:0x10]),
	)
	if !ok {
		return false, fmt.Errorf("native relocation route is unavailable")
	}
	plannedRecords := append([]byte(nil), records...)
	if _, err := battle.ApplyNativeItemRelocation(
		plannedRecords, sourceUnit, []byte{byte(g.nativeItemRelocationUnit)},
		byte(x), byte(y), route, g.nativeCommandBook,
	); err != nil {
		return false, err
	}
	if g.nativeUnitPresent != nil || g.native2189A != nil || g.transitionReveal != nil ||
		g.indexedTransition != nil || g.nativePaletteRamp != nil || g.nativePalettePulse != nil ||
		g.nativeCh20SkyKey != nil || g.nativeCh23Loop != nil {
		return false, fmt.Errorf("native relocation presentation is already active")
	}
	targetSlot := g.nativeItemRelocationUnit
	target := g.st.Units[targetSlot]
	disappear := campaign.NativeUnitPresent{
		Slot: targetSlot,
		NewX: 0xff, NewY: 0xff,
		VisualX: target.X, VisualY: target.Y,
	}
	disappearJob, err := g.buildNativeUnitPresentJob(
		disappear, g.st, g.nativeMapWork, g.nativeMapVGA, nil,
	)
	if err != nil {
		return false, err
	}
	offMapState, err := cloneNativeUnitPresentState(g.st)
	if err != nil || !offMapState.Units[targetSlot].SetNativeMapCoordinatesRaw(0xff, 0xff) {
		return false, fmt.Errorf("native relocation off-map preflight failed")
	}
	last := len(disappearJob.vgaFrames) - 1
	appear := campaign.NativeUnitPresent{
		Slot: targetSlot,
		NewX: x, NewY: y,
		VisualX: x, VisualY: y,
	}
	appearJob, err := g.buildNativeUnitPresentJob(
		appear, offMapState,
		disappearJob.workFrames[last], disappearJob.vgaFrames[last], nil,
	)
	if err != nil {
		return false, err
	}

	actor := g.sel
	beforeTarget := *target
	beforeWork := append([]byte(nil), g.nativeMapWork...)
	beforeVGA := append([]byte(nil), g.nativeMapVGA...)
	rollback := func() {
		if g.st != nil && targetSlot >= 0 && targetSlot < len(g.st.Units) && g.st.Units[targetSlot] != nil {
			*g.st.Units[targetSlot] = beforeTarget
		}
		g.nativeMapWork = append(g.nativeMapWork[:0], beforeWork...)
		g.nativeMapVGA = append(g.nativeMapVGA[:0], beforeVGA...)
	}
	disappearJob.rollback = rollback
	appearJob.rollback = rollback
	appearJob.then = func() {
		if g.st == nil || len(g.st.Units)*80 != len(plannedRecords) ||
			sourceUnit >= len(g.st.Units) || g.st.Units[sourceUnit] != actor ||
			targetSlot >= len(g.st.Units) || g.st.Units[targetSlot] != target {
			rollback()
			g.loadErr = "native command 23 relocation state changed during presentation"
			return
		}
		for index, unit := range g.st.Units {
			syncNativeItemRuntimeRecord(unit, plannedRecords[index*80:(index+1)*80])
		}
		actor.NativeRecordByte5 |= 0x80
		actor.HasNativeRecordByte5 = true
		g.finishSuccessfulUnitAction(actor, func() {
			g.resetNativeTargetField()
			g.st.MaterializeNativeMapRangeMode(1)
			g.nativeItemRelocating = false
			g.nativeMovementCostRows = nil
			g.nativeItemEffectRows = nil
			g.sel, g.reach, g.moved = nil, nil, false
		})
		g.msg = fmt.Sprintf("物品 %02Xh：原始移位效果完成", g.nativeItemTargetID)
	}
	disappearJob.then = func() {
		g.nativeUnitPresent = appearJob
		g.publishNativeUnitPresentFrame()
	}
	err = g.startNativeCommandPalettePresentation(23, func() error { return nil }, func() error {
		g.nativeUnitPresent = disappearJob
		g.publishNativeUnitPresentFrame()
		return nil
	}, nil)
	if err != nil {
		return false, err
	}
	return true, nil
}

func (g *Game) drawNativeItemPanel(screen *ebiten.Image) bool {
	if g.nativeItemPanel == nil {
		return false
	}
	frame := 11 - g.itemAnimStep
	if g.itemClosing {
		frame = g.itemAnimStep
	}
	if frame < 0 {
		frame = 0
	}
	if frame > 11 {
		frame = 11
	}
	pass, err := battle.NativeItemPanelFrameFor(frame)
	if err != nil {
		return false
	}
	for _, region := range []battle.NativeItemPanelRegion{pass.Left, pass.Upper, pass.Bottom} {
		if !region.Enabled || region.Width <= 0 || region.Height <= 0 {
			continue
		}
		source := g.nativeItemPanel.SubImage(image.Rect(
			region.SourceX, region.SourceY,
			region.SourceX+region.Width, region.SourceY+region.Height,
		)).(*ebiten.Image)
		op := &ebiten.DrawImageOptions{}
		op.GeoM.Scale(2, 2)
		op.GeoM.Translate(float64(region.DestX*2), float64(region.DestY*2))
		screen.DrawImage(source, op)
	}
	return true
}
