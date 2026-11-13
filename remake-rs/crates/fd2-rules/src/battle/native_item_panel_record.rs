//! Go 來源：`remake/internal/battle/native_item_panel_record.go`（90 行）。

use super::*;

/// NativeItemPanelRecordForUnit materializes only the 80-byte fields consumed
/// by 0x17eef/0x17fc0/0x184c0. Raw provenance is mandatory for selectors,
/// constructor bytes and inventory cells; normalized fields are used only
/// where the native record meaning is independently established.
///
/// Go：`native_item_panel_record.go:13`，`func NativeItemPanelRecordForUnit(unit *Unit) ([]byte, error)`
pub fn native_item_panel_record_for_unit(unit: Option<&Unit>) -> Result<Vec<u8>> {
    let Some(unit) = unit.filter(|u| {
        u.has_native_record_byte6
            && u.has_native_record_race
            && u.has_native_record_class
            && u.inventory_slots.len() as i64 == NATIVE_INVENTORY_CELLS
            && u.native_inventory_flags.len() as i64 == NATIVE_INVENTORY_CELLS
    }) else {
        return Err(battle_err!("native item panel: unit lacks raw record provenance"));
    };
    let (raw_byte8, has_raw_byte8) = native_record_byte8_for_unit(Some(unit));
    if !has_raw_byte8 {
        return Err(battle_err!("native item panel: unit lacks raw record byte +8"));
    }
    if !(0..=0xff).contains(&unit.battle_fig)
        || !(0..=0xff).contains(&unit.lv)
        || !(0..=0xff).contains(&unit.mv)
        || unit.exp < 0.0
        || unit.exp > 255.0
        || unit.exp != unit.exp.trunc()
    {
        return Err(battle_err!("native item panel: byte field is outside native range"));
    }
    for slot in 0..NATIVE_INVENTORY_CELLS as usize {
        if !(0..=0xff).contains(&unit.inventory_slots[slot]) || !(0..=0xff).contains(&unit.native_inventory_flags[slot])
        {
            return Err(battle_err!("native item panel: inventory cell is outside native range"));
        }
    }
    for value in [unit.dx, unit.hp, unit.max_hp, unit.mp, unit.max_mp, unit.ap, unit.dp, unit.hit, unit.ev] {
        if value < i16::MIN as i64 || value > i16::MAX as i64 {
            return Err(battle_err!("native item panel: word field is outside native range"));
        }
    }

    let mut record = vec![0u8; NATIVE_RECORD_SIZE as usize];
    record[6] = unit.native_record_byte6;
    record[7] = unit.battle_fig as u8;
    record[8] = raw_byte8;
    for slot in 0..NATIVE_INVENTORY_CELLS as usize {
        record[0x0a + slot * 2] = unit.native_inventory_flags[slot] as u8;
        record[0x0b + slot * 2] = unit.inventory_slots[slot] as u8;
    }
    record[0x1a..0x1f].copy_from_slice(&unit.native_command_mask);
    record[0x1f] = unit.native_record_race;
    record[0x20] = unit.native_record_class;
    record[0x21] = unit.lv as u8;
    record[0x22..0x28].copy_from_slice(&unit.native_transient);
    record[0x3b] = unit.mv as u8;
    record[0x3c] = unit.exp as u8;
    put_native_item_panel_word(&mut record, 0x3e, unit.dx);
    put_native_item_panel_word(&mut record, 0x40, unit.hp);
    put_native_item_panel_word(&mut record, 0x42, unit.max_hp);
    put_native_item_panel_word(&mut record, 0x44, unit.mp);
    put_native_item_panel_word(&mut record, 0x46, unit.max_mp);
    put_native_item_panel_word(&mut record, 0x48, unit.ap);
    put_native_item_panel_word(&mut record, 0x4a, unit.dp);
    put_native_item_panel_word(&mut record, 0x4c, unit.hit);
    put_native_item_panel_word(&mut record, 0x4e, unit.ev);
    Ok(record)
}

/// Go：`native_item_panel_record.go:72`，`func nativeRecordByte8ForUnit(unit *Unit) (byte, bool)`
pub(crate) fn native_record_byte8_for_unit(unit: Option<&Unit>) -> (u8, bool) {
    let Some(unit) = unit else {
        return (0, false);
    };
    if unit.has_native_record_byte8 {
        return (unit.native_record_byte8, true);
    }
    // Compatibility assets created before raw +8 was separated stored the
    // same proven player-persistent byte only as NativeIdentity. New runtime
    // restore paths never use this fallback.
    if unit.has_native_identity && (0..=0xff).contains(&unit.native_identity) {
        return (unit.native_identity as u8, true);
    }
    (0, false)
}

/// Go：`native_item_panel_record.go:88`，`func putNativeItemPanelWord(record []byte, offset, value int)`
pub(crate) fn put_native_item_panel_word(record: &mut [u8], offset: i64, value: i64) {
    let offset = offset as usize;
    record[offset..offset + 2].copy_from_slice(&(value as i16 as u16).to_le_bytes());
}

#[cfg(test)]
mod native_item_panel_record_test;
