//! Go 來源：`remake/cmd/fd2/native_item_panel_ui.go`（985 行）。
//!
//! 骨架：Go 檔的 24 個函式中產生 1 個（階段 0 清單），其餘標 `r8`，見 `PORTING.json`。

use super::*;

impl Game {
    /// Go：`native_item_panel_ui.go:156`，`func (g *Game) clearNativeItemPanel()`
    pub(crate) fn clear_native_item_panel(&mut self) {
        self.native_item_panel_text = None;
        self.native_item_panel = None;
        self.native_item_panel_base = Vec::new();
        self.native_item_panel_record = Vec::new();
        self.native_item_panel_assets = None;
        self.native_item_effect_rows = Vec::new();
        self.item_anim_step = 0;
        self.item_closing = false;
    }
}

/// applyNativeAITargetItem consumes the complete 0x1567e winner list at the
/// 0x15055 boundary. It deliberately bypasses the player-owned cursor/modal:
/// the original AI owner rebuilds the whole list from its saved destination
/// and passes that list to 0x20c6f. All mutable records stay detached until
/// item, target and effect provenance have passed validation.
///
/// Go：`native_item_panel_ui.go:433`
#[derive(Debug, Clone, Default, PartialEq, Eq)]
pub(crate) struct NativeAIItemTransaction {
    pub(crate) before: Vec<u8>,
    pub(crate) after: Vec<u8>,
    pub(crate) targets: Vec<UnitIdx>,
    pub(crate) restore: Option<battle::NativeRawRestoreBatch>,
    pub(crate) damage: Vec<battle::NativeCommandDamage>,
    pub(crate) damage_route: Option<battle::NativeItemCommandDamageRoute>,
    pub(crate) rng_before: u16,
    pub(crate) rng_after: u16,
}

/// Go：`native_item_panel_ui.go:386`，`func nativeItemRuntimeRecords(units []*battle.Unit) ([]byte, error)`（`r8`，隨
/// `commitNativeAIItemTransaction` 補移植）。
pub(crate) fn native_item_runtime_records(units: &[battle::Unit]) -> Result<Vec<u8>> {
    let mut records = Vec::with_capacity(units.len() * 80);
    for (index, unit) in units.iter().enumerate() {
        let mut record = battle::native_item_panel_record_for_unit(Some(unit))
            .map_err(|e| GameError::from(e).wrap(format!("runtime unit {index} lacks native item record: ")))?;
        if unit.x < 0 || unit.x > 0xff || unit.y < 0 || unit.y > 0xff || !unit.has_native_record_byte5 {
            return Err(game_err!("runtime unit {index} lacks native coordinate/activity provenance"));
        }
        (record[0], record[1], record[5]) = (unit.x as u8, unit.y as u8, unit.native_record_byte5);
        records.extend_from_slice(&record);
    }
    Ok(records)
}

/// Go：`native_item_panel_ui.go:403`，`func syncNativeItemRuntimeRecord(unit *battle.Unit, record []byte)`（`r8`，隨
/// `cancelNativeAIItemPresentation` 補移植）。
pub(crate) fn sync_native_item_runtime_record(unit: &mut battle::Unit, record: &[u8]) {
    let word = |offset: usize| i64::from(i16::from_le_bytes([record[offset], record[offset + 1]]));
    unit.native_record_byte5 = record[5];
    unit.has_native_record_byte5 = true;
    unit.hp = word(0x40);
    unit.mp = word(0x44);
    let dir = unit.dir;
    unit.set_map_placement(i64::from(record[0]), i64::from(record[1]), dir);
    unit.ap = word(0x48);
    unit.dp = word(0x4a);
    unit.hit = word(0x4c);
    unit.ev = word(0x4e);
    unit.native_transient.copy_from_slice(&record[0x22..0x28]);
    unit.inventory_slots = vec![0; 8];
    unit.native_inventory_flags = vec![0; 8];
    unit.inventory.clear();
    unit.equipped.clear();
    for slot in 0..8 {
        let (flag, item) = (i64::from(record[0x0a + slot * 2]), i64::from(record[0x0b + slot * 2]));
        (unit.native_inventory_flags[slot], unit.inventory_slots[slot]) = (flag, item);
        if flag & 0x80 == 0 {
            unit.inventory.push(item);
            unit.equipped.push(flag & 0x40 != 0);
        }
    }
}

impl Game {
    /// Go：`native_item_panel_ui.go:528`，`func (g *Game) commitNativeAIItemTransaction(tx *nativeAIItemTransaction) error`
    /// （`r8`，隨 `stepNativeAIItemPresentation` 補移植）。
    pub(crate) fn commit_native_ai_item_transaction(&mut self, tx: Option<&NativeAIItemTransaction>) -> Result<()> {
        let (Some(st), Some(tx)) = (self.st.as_mut(), tx) else {
            return Err(game_err!("native AI item transaction is unavailable"));
        };
        if tx.before.is_empty() || tx.after.len() != tx.before.len() {
            return Err(game_err!("native AI item transaction is unavailable"));
        }
        let current = native_item_runtime_records(&st.units)?;
        if current != tx.before || self.native_rng_state != tx.rng_before {
            return Err(game_err!("native AI item source state changed before publication"));
        }
        for (index, unit) in st.units.iter_mut().enumerate() {
            sync_native_item_runtime_record(unit, &tx.after[index * 80..(index + 1) * 80]);
        }
        self.native_rng_state = tx.rng_after;
        Ok(())
    }
}
