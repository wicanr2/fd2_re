//! Go 來源：`remake/internal/campaign/native_current_snapshot.go`（135 行）。

use super::*;

/// BuildNativeCurrentRuntimeRecords overlays the proven live Unit fields on an
/// exact current-runtime raw baseline. Unknown bytes remain byte-identical.
/// This is intentionally narrower than a general Unit serializer: it accepts
/// only a stable CONTINUE-projected roster in original record order.
///
/// Go：`native_current_snapshot.go:15`，`func BuildNativeCurrentRuntimeRecords( state *battle.State, ) ([]fdsave.PersistentRecord, error)`
pub fn build_native_current_runtime_records(state: Option<&battle::State>) -> Result<Vec<save::PersistentRecord>> {
    let state = match state {
        Some(state)
            if state.has_native_runtime_unit_projection
                && !state.units.is_empty()
                && state.native_runtime_records.len() == state.units.len() =>
        {
            state
        }
        _ => {
            return Err(campaign_err!("native current save: runtime roster lacks exact CONTINUE provenance"));
        }
    };
    let mut out = vec![save::PersistentRecord::default(); state.units.len()];
    for (index, unit) in state.units.iter().enumerate() {
        if !unit.has_native_map_presentation
            || unit.x < 0
            || unit.x > 0xff
            || unit.y < 0
            || unit.y > 0xff
            || i64::from(unit.native_map_presentation.x) != unit.x
            || i64::from(unit.native_map_presentation.y) != unit.y
            || !unit.has_native_record_byte5
            || (unit.native_record_byte5 & 1 == 0) != unit.on_field
            || (unit.native_record_byte5 & 0x80 != 0) != unit.acted
        {
            return Err(campaign_err!("native current save: runtime unit {index} has inconsistent live provenance"));
        }
        let panel = battle::native_item_panel_record_for_unit(Some(unit))
            .map_err(|err| CampaignError::from(err).wrap(format!("native current save: runtime unit {index}: ")))?;
        let baseline = &state.native_runtime_records[index];
        if baseline.selector_key != unit.battle_fig as u8
            || !unit.has_map_selector_key
            || unit.map_selector_key != i64::from(baseline.selector_key)
            || !unit.has_map_selector_slot
            || unit.map_selector_slot != i64::from(baseline.selector_slot)
        {
            return Err(campaign_err!("native current save: runtime unit {index} selector identity changed"));
        }

        let mut record = baseline.raw;
        record[0] = unit.x as u8;
        record[1] = unit.y as u8;
        record[3] = unit.native_map_presentation.pose;
        record[4] = unit.native_map_presentation.motion;
        record[5] = unit.native_record_byte5;
        record[6..9].copy_from_slice(&panel[6..9]);
        record[0x0a..0x28].copy_from_slice(&panel[0x0a..0x28]);
        if unit.has_native_record_death_effect {
            record[0x31..0x34].copy_from_slice(&unit.native_record_death_effect);
        }
        // Go 以 map 逐一覆寫四個互不重疊的偏移，迭代順序不影響結果。
        for (offset, value, has) in [
            (0x34, unit.native_record_byte34, unit.has_native_record_byte34),
            (0x35, unit.native_record_byte35, unit.has_native_record_byte35),
            (0x36, unit.native_record_byte36, unit.has_native_record_byte36),
            (0x3d, unit.native_record_byte3_d, unit.has_native_record_byte3_d),
        ] {
            if has {
                record[offset] = value;
            }
        }
        record[0x3b..0x3d].copy_from_slice(&panel[0x3b..0x3d]);
        record[0x3e..0x50].copy_from_slice(&panel[0x3e..0x50]);
        out[index].raw = record;
    }
    Ok(out)
}

/// BuildNativeCurrentPersistentRecords extends the exact persistent baseline
/// only for newly appended own-camp JOIN records. Enemy/ally reinforcements are
/// battle-local and do not consume persistent slots. Every new player record is
/// rebuilt by the proven JOIN constructor; all pre-existing and unused bytes
/// remain byte-identical to the loaded snapshot.
///
/// Go：`native_current_snapshot.go:80`，`func BuildNativeCurrentPersistentRecords( state *battle.State, baseline fdsave.CurrentSnapshot, table NativeJoinConstructorTable, itemRows []byte, ) ([fdsave.RosterUnits]fdsave.PersistentRecord, byte, error)`
pub fn build_native_current_persistent_records(
    state: Option<&battle::State>,
    baseline: &save::CurrentSnapshot,
    table: &NativeJoinConstructorTable,
    item_rows: &[u8],
) -> Result<([save::PersistentRecord; 32], u8)> {
    let mut out = baseline.persistent_records;
    let mut count = i64::from(baseline.header.persistent_count);
    let old_runtime_count = i64::from(baseline.header.runtime_count);
    let state = match state {
        Some(state)
            if count <= out.len() as i64
                && old_runtime_count <= state.units.len() as i64
                && state.units.len() == state.native_runtime_records.len() =>
        {
            state
        }
        _ => return Err(campaign_err!("native current save: persistent/runtime baseline is inconsistent")),
    };
    let mut identities = BTreeSet::new();
    for record in &out[..count as usize] {
        let id = record.raw[8];
        if !identities.insert(id) {
            return Err(campaign_err!("native current save: duplicate persistent identity {id}"));
        }
    }
    for (index, unit) in state.units.iter().enumerate().skip(old_runtime_count as usize) {
        if unit.camp != battle::OWN {
            continue;
        }
        if !unit.has_native_identity
            || unit.native_identity < 0
            || unit.native_identity > 0xff
            || !unit.has_native_record_byte8
            || i64::from(unit.native_record_byte8) != unit.native_identity
        {
            return Err(campaign_err!("native current save: appended own unit {index} lacks JOIN identity"));
        }
        let id = unit.native_identity as u8;
        if identities.contains(&id) {
            return Err(campaign_err!("native current save: appended own identity {id} is not a new JOIN"));
        }
        if count >= out.len() as i64 {
            return Err(campaign_err!("native current save: persistent roster exceeds 32 records"));
        }
        let record = table.materialize_persistent_record_on(out[count as usize], i64::from(id), item_rows)?;
        let view = record.view();
        if !unit.has_native_record_race
            || view.race != unit.native_record_race
            || !unit.has_native_record_class
            || view.class != unit.native_record_class
            || !unit.has_map_selector_key
            || unit.map_selector_key != i64::from(view.raw_presentation_key)
        {
            return Err(campaign_err!("native current save: JOIN identity {id} constructor/runtime mismatch"));
        }
        out[count as usize] = record;
        count += 1;
        identities.insert(id);
    }
    Ok((out, count as u8))
}

#[cfg(test)]
pub(crate) mod native_current_snapshot_test;
