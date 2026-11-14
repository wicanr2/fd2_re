//! Go 來源：`remake/internal/campaign/native_continue_runtime_units.go`（210 行）。

use super::*;

/// MaterializeNativeContinueRuntimeUnits projects the saved 0x50-byte runtime
/// roster fields needed by the current battle into battle.Units without
/// consulting FDFIELD constructor
/// rows. The saved list order is authoritative. Raw +7 is consumed separately
/// as the battle-figure selector and as the CONTINUE selector-cache rebuild key;
/// raw +8 is promoted to persistent identity only for native camp 2.
///
/// This adapter deliberately does not seed map timing, switch to interactive
/// range mode, bind future FDFIELD groups, or publish the Game/controller
/// handoff.
///
/// Go：`native_continue_runtime_units.go:22`，`func MaterializeNativeContinueRuntimeUnits( state *battle.State, input fdsave.ContinueRuntimeInput, catalog *NativeCharacterCatalog, ) error`
pub fn materialize_native_continue_runtime_units(
    state: Option<&mut battle::State>,
    input: &save::ContinueRuntimeInput,
    catalog: &mut NativeCharacterCatalog,
) -> Result<()> {
    let Some(state) = state else {
        return Err(campaign_err!("native CONTINUE runtime units: nil battle state"));
    };
    if !input.validated_for_runtime_bridge() {
        return Err(campaign_err!("native CONTINUE runtime units: input did not pass preflight"));
    }
    catalog.validate().map_err(|err| err.wrap("native CONTINUE runtime units: "))?;
    if !state.has_native_field_control_state || state.native_runtime_records.len() != input.runtime_records.len() {
        return Err(campaign_err!("native CONTINUE runtime units: field boundary is incomplete"));
    }
    for (index, record) in input.runtime_records.iter().enumerate() {
        let want = battle::NativeRuntimeRecordState {
            raw: record.raw.raw,
            selector_key: record.selector_key,
            selector_slot: record.selector_slot,
        };
        if state.native_runtime_records[index] != want {
            return Err(campaign_err!("native CONTINUE runtime units: field record {index} mismatch"));
        }
    }

    let mut units = Vec::with_capacity(input.runtime_records.len());
    for (index, record) in input.runtime_records.iter().enumerate() {
        let unit = materialize_native_continue_runtime_unit(
            *record,
            input.context.field_width,
            input.context.field_height,
            catalog,
        )
        .map_err(|err| err.wrap(format!("native CONTINUE runtime units: record {index}: ")))?;
        units.push(unit);
    }

    let mut cache = fdicon::NativeSelectorCache::default();
    battle::materialize_native_map_selector_slots(&mut units, Some(&mut cache))
        .map_err(|err| CampaignError::from(err).wrap("native CONTINUE runtime units: "))?;
    for (index, record) in input.runtime_records.iter().enumerate() {
        if units[index].map_selector_slot != i64::from(record.selector_slot) {
            return Err(campaign_err!("native CONTINUE runtime units: record {index} selector order mismatch"));
        }
        let raw = record.raw.raw;
        units[index].native_map_presentation =
            battle::NativeMapPresentationState { x: raw[0], y: raw[1], pose: raw[3], motion: raw[4] };
        units[index].has_native_map_presentation = true;
        units[index].dir = i64::from(raw[3]);
        (units[index].off_x, units[index].off_y) = (0.0, 0.0);
    }

    let mut candidate = state.clone();
    candidate.units = units;
    candidate.native_map_selector_cache = Some(cache);
    candidate.native_map_selector_error = None;
    candidate.has_native_runtime_unit_projection = true;
    *state = candidate;
    Ok(())
}

/// Go：`native_continue_runtime_units.go:94`，`func materializeNativeContinueRuntimeUnit( record fdsave.ContinueRuntimeRecord, fieldWidth, fieldHeight int, catalog *NativeCharacterCatalog, ) (*battle.Unit, error)`
pub(crate) fn materialize_native_continue_runtime_unit(
    record: save::ContinueRuntimeRecord,
    field_width: i64,
    field_height: i64,
    catalog: &NativeCharacterCatalog,
) -> Result<Unit> {
    let raw = record.raw.raw;
    let view = record.raw.view();
    let camp = match view.raw_camp {
        0 => battle::ENEMY,
        1 => battle::ALLY,
        2 => battle::OWN,
        other => return Err(campaign_err!("raw camp {other} is outside 0..2")),
    };
    if view.class as usize >= catalog.class_names.len() {
        return Err(campaign_err!("class {} outside catalog", view.class));
    }
    let active = raw[5] & 1 == 0;
    if active && (i64::from(raw[0]) >= field_width || i64::from(raw[1]) >= field_height || raw[3] > 3 || raw[4] > 6) {
        return Err(campaign_err!("active map presentation is outside field"));
    }

    // `Unit` 有 fd2-rules 私有欄位，crate 外不能用結構更新語法；從零值逐欄設定，與 Go 的複合字面值相同。
    let mut unit = Unit::default();
    unit.camp = camp;
    unit.name = catalog.class_names[view.class as usize].clone();
    unit.cls_name = catalog.class_names[view.class as usize].clone();
    unit.class_id = i64::from(view.class);
    unit.lv = i64::from(view.level);
    unit.hp = i64::from(view.hp);
    unit.max_hp = i64::from(view.max_hp);
    unit.mp = i64::from(view.mp);
    unit.max_mp = i64::from(view.max_mp);
    unit.ap = i64::from(view.ap);
    unit.dp = i64::from(view.dp);
    unit.hit = i64::from(view.hit);
    unit.ev = i64::from(view.ev);
    unit.mv = i64::from(view.movement);
    unit.x = i64::from(raw[0]);
    unit.y = i64::from(raw[1]);
    unit.dir = i64::from(raw[3]);
    // 0x117E7 tests bit7 before admitting a player unit, while successful
    // action paths set it through 0x13512. Acted is only that controller
    // projection; the rest of raw +5 retains caller-specific semantics.
    unit.acted = raw[5] & 0x80 != 0;
    unit.on_field = active;
    (unit.map_selector_key, unit.has_map_selector_key) = (i64::from(record.selector_key), true);
    (unit.battle_fig, unit.has_battle_fig) = (i64::from(raw[7]), true);
    (unit.native_record_byte8, unit.has_native_record_byte8) = (raw[8], true);
    (unit.native_record_race, unit.has_native_record_race) = (view.race, true);
    (unit.native_record_class, unit.has_native_record_class) = (view.class, true);
    unit.native_command_mask = view.command_mask;
    unit.native_transient = view.transient;
    (unit.native_record_byte5, unit.has_native_record_byte5) = (view.raw_byte5, true);
    (unit.native_record_byte6, unit.has_native_record_byte6) = (view.raw_camp, true);
    (unit.native_record_byte34, unit.has_native_record_byte34) = (raw[0x34], true);
    (unit.native_record_byte35, unit.has_native_record_byte35) = (raw[0x35], true);
    (unit.native_record_byte36, unit.has_native_record_byte36) = (raw[0x36], true);
    (unit.native_record_byte3_d, unit.has_native_record_byte3_d) = (raw[0x3d], true);
    unit.native_record_death_effect = [raw[0x31], raw[0x32], raw[0x33]];
    unit.has_native_record_death_effect = true;
    unit.native_record_word42 = u16::from(raw[0x42]) | (u16::from(raw[0x43]) << 8);
    unit.has_native_record_word42 = true;
    unit.native_record_word46 = u16::from(raw[0x46]) | (u16::from(raw[0x47]) << 8);
    unit.has_native_record_word46 = true;
    unit.base_ap = i64::from(view.base_ap);
    unit.base_dp = i64::from(view.base_dp);
    unit.base_hit = i64::from(view.dx);
    unit.base_ev = i64::from(view.dx);
    unit.base_mv = i64::from(view.movement);
    unit.equipment_base_set = true;
    unit.dx = i64::from(view.dx);
    unit.exp = f64::from(view.experience);
    unit.inventory_slots = vec![0; view.inventory.len()];
    unit.native_inventory_flags = vec![0; view.inventory.len()];
    unit.native_map_presentation =
        battle::NativeMapPresentationState { x: raw[0], y: raw[1], pose: raw[3], motion: raw[4] };
    unit.has_native_map_presentation = true;
    for (slot, cell) in view.inventory.iter().enumerate() {
        unit.inventory_slots[slot] = i64::from(cell.item_id);
        unit.native_inventory_flags[slot] = i64::from(cell.flags);
        if (cell.flags as i8) < 0 {
            continue;
        }
        unit.inventory.push(i64::from(cell.item_id));
        unit.equipped.push(cell.flags & 0x40 != 0);
    }
    if camp == battle::OWN {
        let identity = i64::from(raw[8]);
        if identity >= catalog.identity_names.len() as i64 {
            return Err(campaign_err!("own identity {identity} outside catalog"));
        }
        unit.native_identity = identity;
        unit.has_native_identity = true;
        unit.name = catalog.identity_names[identity as usize].clone();
    }
    Ok(unit)
}

#[cfg(test)]
pub(crate) mod native_continue_runtime_units_test;
