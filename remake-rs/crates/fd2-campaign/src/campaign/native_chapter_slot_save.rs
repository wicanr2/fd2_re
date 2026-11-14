//! Go 來源：`remake/internal/campaign/native_chapter_slot_save.go`（171 行）。

use super::*;

// native_chapter_slot_save.go — 酒店存檔（0x30012）寫回原版四槽的 roster 與 metadata。
//
// 原版把記憶體裡的 32 筆持續紀錄整段抄進 `0x312B + slot×0xA28`，再寫 metadata
// `+0` 章節、`+1` 筆數、`+2..+5` 金幣（`+6..+9` 由系統設定寫，這裡沿用 baseline）。
// 重製端的持續隊伍是 battle.Unit 投影，沒有保留每一個 byte，所以每筆紀錄以
// 讀檔時的 raw baseline 為底，只覆寫已證實的欄位；沒有 raw 來源的 byte 一律不動。

/// EncodeNativePersistentRecord 把一個持續隊伍單位覆寫到它讀檔時的 raw 紀錄上。
/// 覆寫的欄位與 BuildNativeCurrentRuntimeRecords 同一組（座標、+5、+6..+8、物品欄、
/// 指令位元、種族／職業／等級、暫時狀態、+0x34..+0x36、+0x3B..+0x4F），再加上升級
/// 會動的 +0x37／+0x39 基礎攻防（`0x1E529` 寫的兩個 word）。
///
/// Go：`native_chapter_slot_save.go:23`，`func EncodeNativePersistentRecord(baseline fdsave.PersistentRecord, unit *battle.Unit) (fdsave.PersistentRecord, error)`
pub fn encode_native_persistent_record(
    baseline: save::PersistentRecord,
    unit: Option<&Unit>,
) -> Result<save::PersistentRecord> {
    let Some(unit) = unit else {
        return Err(campaign_err!("native chapter slot save: unit is nil"));
    };
    if unit.x < 0
        || unit.x > 0xff
        || unit.y < 0
        || unit.y > 0xff
        || !unit.has_native_record_byte5
        || unit.base_ap < -0x8000
        || unit.base_ap > 0x7fff
        || unit.base_dp < -0x8000
        || unit.base_dp > 0x7fff
    {
        return Err(campaign_err!("native chapter slot save: unit lacks raw provenance for the persistent record"));
    }
    let panel = battle::native_item_panel_record_for_unit(Some(unit))
        .map_err(|err| CampaignError::from(err).wrap("native chapter slot save: "))?;
    if panel[8] != baseline.raw[8] {
        return Err(campaign_err!(
            "native chapter slot save: identity {} does not match baseline {}",
            panel[8],
            baseline.raw[8]
        ));
    }
    let mut record = baseline;
    // JOIN 之後尚未被 0x11506 同步的記錄：+0..+4、+0x34..+0x36、+0x3d 是建構器沒碰的殘值，
    // 要等 0x11506 整筆抄回才會有場上的值（第七章 r6 凱麗）。
    let pending = unit.native_join_persistent_pending;
    if !pending {
        // #87：11506只抄回raw身分匹配的runtime紀錄。未出戰者的
        // LOAD投影沒有呈現來源，保留持續槽的原始欄位。
        if unit.has_native_map_presentation {
            record.raw[0] = unit.x as u8;
            record.raw[1] = unit.y as u8;
            record.raw[3] = unit.native_map_presentation.pose;
            record.raw[4] = unit.native_map_presentation.motion;
        }
        if unit.has_map_selector_slot && unit.map_selector_slot >= 0 && unit.map_selector_slot <= 0xff {
            record.raw[2] = unit.map_selector_slot as u8;
        }
    }
    record.raw[5] = unit.native_record_byte5;
    (record.raw[6], record.raw[8]) = (panel[6], panel[8]);
    if unit.has_battle_fig {
        record.raw[7] = panel[7];
    }
    record.raw[0x0a..0x28].copy_from_slice(&panel[0x0a..0x28]);
    if unit.has_native_record_death_effect && !pending {
        // 建構器只寫 +0x31=0xff，+0x32／+0x33 留殘值。
        record.raw[0x31..0x34].copy_from_slice(&unit.native_record_death_effect);
    }
    // Go 以 map 逐一覆寫四個互不重疊的偏移，迭代順序不影響結果。
    for (offset, value, has) in [
        (0x34, unit.native_record_byte34, unit.has_native_record_byte34),
        (0x35, unit.native_record_byte35, unit.has_native_record_byte35),
        (0x36, unit.native_record_byte36, unit.has_native_record_byte36),
        (0x3d, unit.native_record_byte3_d, unit.has_native_record_byte3_d),
    ] {
        if has && !pending {
            record.raw[offset] = value;
        }
    }
    if unit.equipment_base_set {
        record.raw[0x37..0x39].copy_from_slice(&(unit.base_ap as i16 as u16).to_le_bytes());
        record.raw[0x39..0x3b].copy_from_slice(&(unit.base_dp as i16 as u16).to_le_bytes());
    }
    record.raw[0x3b..0x3d].copy_from_slice(&panel[0x3b..0x3d]);
    record.raw[0x3e..0x50].copy_from_slice(&panel[0x3e..0x50]);
    Ok(record)
}

/// BuildNativeChapterSlot 由讀檔時的槽快照、目前的持續隊伍與金幣組出要寫回的槽。
/// baseline 裡的每一筆有效紀錄都要在隊伍裡找得到同一個 +8 身分；隊伍多出來的身分
/// 依加入順序以 JOIN 建構列補在後面。不在場的 byte 全部照 baseline。
///
/// Go：`native_chapter_slot_save.go:90`，`func BuildNativeChapterSlot( baseline fdsave.ChapterSlotSnapshot, roster map[int]battle.Unit, joinOrder []int, chapter byte, gold uint32, table NativeJoinConstructorTable, itemRows []byte, ) (fdsave.Slot, error)`
pub fn build_native_chapter_slot(
    baseline: &save::ChapterSlotSnapshot,
    roster: &BTreeMap<i64, Unit>,
    join_order: &[i64],
    chapter: u8,
    gold: u32,
    table: &NativeJoinConstructorTable,
    item_rows: &[u8],
) -> Result<save::Slot> {
    let mut count = baseline.verified.roster_count as usize;
    if count > save::ROSTER_UNITS {
        return Err(campaign_err!(
            "native chapter slot save: baseline count {count} outside 0..{}",
            save::ROSTER_UNITS
        ));
    }
    // Go 依 map 迭代順序建索引；順序只影響同時有多個錯誤時先報哪一個（PORTING.md）。
    let mut by_identity: BTreeMap<i64, &Unit> = BTreeMap::new();
    for (id, unit) in roster {
        if !unit.has_native_identity {
            return Err(campaign_err!("native chapter slot save: roster member {id} lacks native identity"));
        }
        if by_identity.contains_key(&unit.native_identity) {
            return Err(campaign_err!("native chapter slot save: duplicate native identity {}", unit.native_identity));
        }
        by_identity.insert(unit.native_identity, unit);
    }
    let mut records = baseline.records;
    let mut seen = BTreeSet::new();
    for (index, record) in records.iter_mut().enumerate().take(count) {
        let identity = i64::from(record.raw[8]);
        let Some(unit) = by_identity.get(&identity) else {
            return Err(campaign_err!(
                "native chapter slot save: baseline record {index} identity {identity} is not in the roster"
            ));
        };
        *record = encode_native_persistent_record(*record, Some(unit))
            .map_err(|err| err.wrap(format!("record {index}: ")))?;
        seen.insert(identity);
    }
    for id in join_order {
        let Some(unit) = roster.get(id) else { continue };
        if seen.contains(&unit.native_identity) {
            continue;
        }
        if count >= save::ROSTER_UNITS {
            return Err(campaign_err!("native chapter slot save: persistent roster exceeds 32 records"));
        }
        // sub_112A5 寫進 count 那一格，沒寫到的 byte 是那一格 LOAD 時的殘值（第七章
        // 凱麗：JOIN12 的身分與場上 raw +8=140 不符，0x11506 不抄）。0x11506 按 raw +8
        // 匹配，不要求參戰或特定 camp；第二章新登場的 JOIN8 也會立即被同步。同步後
        // 的場上記錄覆蓋建構殘值；這份具型別投影以零基底補建，已知欄位由 unit 寫回。
        let residual =
            if unit.native_join_persistent_pending { records[count] } else { save::PersistentRecord::default() };
        let constructed = table.materialize_persistent_record_on(residual, unit.native_identity, item_rows)?;
        let encoded = encode_native_persistent_record(constructed, Some(unit))
            .map_err(|err| err.wrap(format!("joined record {count}: ")))?;
        records[count] = encoded;
        seen.insert(unit.native_identity);
        count += 1;
    }
    if seen.len() != by_identity.len() {
        return Err(campaign_err!(
            "native chapter slot save: {} roster members are not in the join order",
            by_identity.len() as i64 - seen.len() as i64
        ));
    }
    let mut slot = save::Slot { roster: Vec::with_capacity(save::ROSTER_SIZE), metadata: baseline.metadata.to_vec() };
    for record in &records {
        slot.roster.extend_from_slice(&record.raw);
    }
    slot.metadata[0] = chapter;
    slot.metadata[1] = count as u8;
    slot.metadata[2..6].copy_from_slice(&gold.to_le_bytes());
    Ok(slot)
}

#[cfg(test)]
pub(crate) mod native_chapter_slot_save_test;
