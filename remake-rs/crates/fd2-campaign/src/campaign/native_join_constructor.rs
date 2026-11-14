//! Go 來源：`remake/internal/campaign/native_join_constructor.go`（283 行）。

use super::*;

pub(crate) const NATIVE_JOIN_EXE_SIZE: i64 = 357074;

pub(crate) const NATIVE_JOIN_EXEMD5: &str = "b97caf2239a27a896069d03549d96e1e";

pub(crate) const NATIVE_JOIN_EXESHA256: &str = "222b7d067ad4450eb9c5f6e6bce1797d54bb050417ba39ced6067f8039f28c4f";

/// NativeJoinConstructorTable is the address-preserving projection of the
/// 32-row tables consumed by original sub_112A5. It deliberately does not reuse
/// the unrelated FDFIELD constructor table.
///
/// Go：`native_join_constructor.go:25`
#[derive(Debug, Clone, Default, PartialEq, Eq)]
pub struct NativeJoinConstructorTable {
    pub(crate) rows: BTreeMap<i64, NativeJoinConstructorRow>,
}

/// Go：`native_join_constructor.go:29`
#[derive(Debug, Clone, Copy, Default, PartialEq, Eq)]
pub(crate) struct NativeJoinConstructorRow {
    pub(crate) defaults: [u8; 24],
    pub(crate) growth: [u8; 11],
}

/// LoadNativeJoinConstructorTable validates the executable identity, row order,
/// source offsets and raw strides before exposing any constructor row.
///
/// Go：`native_join_constructor.go:36`，`func LoadNativeJoinConstructorTable(path string) (NativeJoinConstructorTable, error)`
pub fn load_native_join_constructor_table(path: &str) -> Result<NativeJoinConstructorTable> {
    #[derive(serde::Deserialize, Default)]
    #[serde(default)]
    struct WireSource {
        exe_size: i64,
        exe_md5: String,
        exe_sha256: String,
    }
    #[derive(serde::Deserialize, Default)]
    #[serde(default)]
    struct WireRow {
        id: i64,
        default_file_offset: String,
        growth_file_offset: String,
        default_raw: String,
        growth_raw: String,
    }
    #[derive(serde::Deserialize, Default)]
    #[serde(default)]
    struct Wire {
        schema_version: i64,
        source: WireSource,
        evidence_level: String,
        #[serde(deserialize_with = "battle::null_as_default")]
        rows: Vec<WireRow>,
    }
    let raw = battle::read_file(path)?;
    let wire: Wire = battle::decode_json(&raw)?;
    if wire.schema_version != 1
        || wire.source.exe_size != NATIVE_JOIN_EXE_SIZE
        || wire.source.exe_md5 != NATIVE_JOIN_EXEMD5
        || wire.source.exe_sha256 != NATIVE_JOIN_EXESHA256
        || wire.evidence_level != "已證實"
        || wire.rows.len() != 32
    {
        return Err(campaign_err!("native JOIN constructor manifest identity is invalid"));
    }

    let mut table = NativeJoinConstructorTable { rows: BTreeMap::new() };
    for (index, source) in wire.rows.iter().enumerate() {
        let index = index as i64;
        if source.id != index {
            return Err(campaign_err!("native JOIN row {index} is not position-indexed"));
        }
        match parse_native_join_offset(&source.default_file_offset) {
            Ok(offset) if offset == 0x55ba1 + index * 0x18 => {}
            _ => return Err(campaign_err!("native JOIN row {index} default offset is invalid")),
        }
        match parse_native_join_offset(&source.growth_file_offset) {
            Ok(offset) if offset == 0x55ea1 + index * 0x0b => {}
            _ => return Err(campaign_err!("native JOIN row {index} growth offset is invalid")),
        }
        let default_raw = match battle::decode_hex(&source.default_raw) {
            Some(b) if b.len() == 0x18 => b,
            _ => return Err(campaign_err!("native JOIN row {index} default stride is invalid")),
        };
        let growth_raw = match battle::decode_hex(&source.growth_raw) {
            Some(b) if b.len() == 0x0b => b,
            _ => return Err(campaign_err!("native JOIN row {index} growth stride is invalid")),
        };
        let mut row = NativeJoinConstructorRow::default();
        row.defaults.copy_from_slice(&default_raw);
        row.growth.copy_from_slice(&growth_raw);
        table.rows.insert(index, row);
    }
    Ok(table)
}

/// Go：`native_join_constructor.go:95`，`func parseNativeJoinOffset(value string) (int, error)`
///
/// Go `strconv.ParseInt(strings.TrimPrefix(value, "0x"), 16, 32)`：去掉一次 `0x` 後，接受可選正負號
/// 與十六進位數字，範圍 int32；`i32::from_str_radix` 的接受範圍相同。錯誤文字不比對（呼叫端換成自己的訊息）。
pub(crate) fn parse_native_join_offset(value: &str) -> Result<i64> {
    let digits = value.strip_prefix("0x").unwrap_or(value);
    i32::from_str_radix(digits, 16)
        .map(i64::from)
        .map_err(|_| campaign_err!("strconv.ParseInt: parsing {}: invalid syntax", battle::go_quote(digits)))
}

/// Go：`native_join_constructor.go:100`，`func nativeJoinWord(raw []byte, offset int) int`
pub(crate) fn native_join_word(raw: &[u8], offset: i64) -> i64 {
    let offset = offset as usize;
    i64::from(raw[offset]) | (i64::from(raw[offset + 1]) << 8)
}

/// Go `int(int16(binary.LittleEndian.Uint16(raw[offset:])))`。
fn native_join_signed_word(raw: &[u8], offset: usize) -> i64 {
    i64::from(i16::from_le_bytes([raw[offset], raw[offset + 1]]))
}

/// 檢查建構列並算出等級與 HP／MP 上限（Go 兩個 `On` 方法開頭相同的檢查）。
fn native_join_row_stats(
    table: &NativeJoinConstructorTable,
    id: i64,
) -> Result<(NativeJoinConstructorRow, i64, i64, i64)> {
    let row = match table.rows.get(&id) {
        Some(row) if (0..=0xff).contains(&id) => *row,
        _ => return Err(campaign_err!("native JOIN character {id} has no proven constructor row")),
    };
    let level = i64::from(row.defaults[2]);
    if level <= 0 {
        return Err(campaign_err!("native JOIN character {id} has invalid level {level}"));
    }
    let max_hp = native_join_word(&row.defaults, 3) + i64::from(row.growth[6]) * (level - 1);
    let max_mp = native_join_word(&row.defaults, 5) + i64::from(row.growth[8]) * (level - 1);
    if !(0..=0xffff).contains(&max_hp) || !(0..=0xffff).contains(&max_mp) {
        return Err(campaign_err!("native JOIN character {id} HP/MP exceeds raw word"));
    }
    Ok((row, level, max_hp, max_mp))
}

impl NativeJoinConstructorTable {
    /// MaterializePersistentUnit transcribes the proven sub_112A5 writes into a
    /// local persistent record and runs its sub_1145A equipment tail. The local
    /// record is transaction staging, not a claim that untouched bytes of every
    /// native 0x50-byte record are known. No Unit is returned unless the raw item
    /// rows needed by every equipped slot are available.
    ///
    /// Go：`native_join_constructor.go:109`，`func (table NativeJoinConstructorTable) MaterializePersistentUnit( id int, base battle.Unit, itemTable []byte, ) (battle.Unit, error)`
    pub fn materialize_persistent_unit(&self, id: i64, base: &Unit, item_table: &[u8]) -> Result<Unit> {
        self.materialize_persistent_unit_on(save::PersistentRecord::default(), id, base.clone(), item_table)
    }

    /// MaterializePersistentUnitOn 同 MaterializePersistentUnit，但建構器沒寫到的 byte 照
    /// residual（sub_112A5 寫進去的那一格在寫入前的內容）；空物品格的 item byte 也跟著
    /// residual 走，之後寫回槽時才會和原版逐 byte 相同。
    ///
    /// Go：`native_join_constructor.go:120`，`func (table NativeJoinConstructorTable) MaterializePersistentUnitOn( residual fdsave.PersistentRecord, id int, base battle.Unit, itemTable []byte, ) (battle.Unit, error)`
    pub fn materialize_persistent_unit_on(
        &self,
        residual: save::PersistentRecord,
        id: i64,
        base: Unit,
        item_table: &[u8],
    ) -> Result<Unit> {
        let (row, level, max_hp, max_mp) = native_join_row_stats(self, id)?;

        let persistent = self.materialize_persistent_record_on(residual, id, item_table)?;
        let record = persistent.raw;

        let mut unit = base.clone();
        // 建構器只寫 +5..+0x16、+0x18、+0x1a..+0x21、+0x31、+0x37.. 與裝備重算的欄位；
        // +0x34..+0x36、+0x3d 等留 residual。下一次戰後同步（0x11506 整筆抄回）之前，
        // 持續記錄裡就是這些 byte，不是場上記錄帶來的值。
        (unit.native_record_byte34, unit.has_native_record_byte34) = (record[0x34], true);
        (unit.native_record_byte35, unit.has_native_record_byte35) = (record[0x35], true);
        (unit.native_record_byte36, unit.has_native_record_byte36) = (record[0x36], true);
        (unit.native_record_byte3_d, unit.has_native_record_byte3_d) = (record[0x3d], true);
        unit.native_join_persistent_pending = true;
        unit.camp = battle::OWN;
        unit.lv = level;
        (unit.hp, unit.max_hp) = (max_hp, max_hp);
        (unit.mp, unit.max_mp) = (max_mp, max_mp);
        unit.mv = i64::from(row.defaults[7]);
        unit.exp = 0.0;
        unit.ap = native_join_signed_word(&record, 0x48);
        unit.dp = native_join_signed_word(&record, 0x4a);
        unit.hit = native_join_signed_word(&record, 0x4c);
        unit.ev = native_join_signed_word(&record, 0x4e);
        unit.dx = native_join_signed_word(&record, 0x3e);
        unit.base_ap = native_join_signed_word(&record, 0x37);
        unit.base_dp = native_join_signed_word(&record, 0x39);
        (unit.base_hit, unit.base_ev) = (unit.dx, unit.dx);
        unit.base_mv = i64::from(record[0x3b]);
        (unit.base_atk_min, unit.base_atk_max) = (base.atk_min, base.atk_max);
        unit.equipment_base_set = true;
        unit.class_id = i64::from(row.defaults[1]);
        (unit.native_record_race, unit.has_native_record_race) = (row.defaults[0], true);
        (unit.native_record_class, unit.has_native_record_class) = (row.defaults[1], true);
        (unit.native_record_byte5, unit.has_native_record_byte5) = (0, true);
        (unit.native_record_byte6, unit.has_native_record_byte6) = (2, true);
        (unit.native_identity, unit.has_native_identity) = (id, true);
        (unit.native_record_byte8, unit.has_native_record_byte8) = (id as u8, true);
        (unit.map_selector_key, unit.has_map_selector_key) = (id, true);
        (unit.battle_fig, unit.has_battle_fig) = (id, true);
        unit.native_command_mask = [0; 5];
        unit.native_command_mask[..4].copy_from_slice(&row.defaults[8..12]);
        unit.native_transient = [0; 6];
        (unit.native_record_word42, unit.has_native_record_word42) = (max_hp as u16, true);
        (unit.native_record_word46, unit.has_native_record_word46) = (max_mp as u16, true);
        unit.inventory = Vec::new();
        unit.equipped = Vec::new();
        unit.inventory_slots = vec![0; 8];
        unit.native_inventory_flags = vec![0; 8];
        for slot in 0..8 {
            let cell = 0x0a + slot * 2;
            let (flags, item) = (record[cell], record[cell + 1]);
            unit.inventory_slots[slot] = i64::from(item);
            unit.native_inventory_flags[slot] = i64::from(flags);
            if (flags as i8) < 0 {
                continue;
            }
            unit.inventory.push(i64::from(item));
            unit.equipped.push(flags & 0x40 != 0);
        }
        Ok(unit)
    }

    /// MaterializePersistentRecord exposes the exact local 0x50-byte record built
    /// by the proven JOIN constructor. It is used when current-battle SAVE must add
    /// a newly joined character to the native persistent roster without inventing
    /// unknown bytes from a normalized Unit.
    ///
    /// Go：`native_join_constructor.go:208`，`func (table NativeJoinConstructorTable) MaterializePersistentRecord( id int, itemTable []byte, ) (fdsave.PersistentRecord, error)`
    pub fn materialize_persistent_record(&self, id: i64, item_table: &[u8]) -> Result<save::PersistentRecord> {
        self.materialize_persistent_record_on(save::PersistentRecord::default(), id, item_table)
    }

    /// MaterializePersistentRecordOn 把 sub_112A5 的寫入疊在 residual 上：持續名冊記憶體是
    /// 0x10010 從槽整批還原的，count 之後那一格在 JOIN 之前就是 LOAD 時槽裡的 bytes。
    /// 沒有 LOAD 時本引擎使用零基底，不宣稱原版新開局全零；首章 START 收據已有非零反證。
    /// 建構器沒寫到的 byte 原封不動留在存檔裡（第七章 r6 凱麗：+0..+4、+0x17、
    /// +0x19、+0x28..+0x30、+0x32..+0x36、+0x3d 都是 LOAD 進來的殘值）。
    ///
    /// Go：`native_join_constructor.go:220`，`func (table NativeJoinConstructorTable) MaterializePersistentRecordOn( residual fdsave.PersistentRecord, id int, itemTable []byte, ) (fdsave.PersistentRecord, error)`
    pub fn materialize_persistent_record_on(
        &self,
        residual: save::PersistentRecord,
        id: i64,
        item_table: &[u8],
    ) -> Result<save::PersistentRecord> {
        let (row, level, max_hp, max_mp) = native_join_row_stats(self, id)?;

        let mut record = residual;
        let raw = &mut record.raw[..];
        raw[5] = 0;
        raw[6] = 2;
        raw[7] = id as u8;
        raw[8] = id as u8;
        raw[9] = 0;
        (raw[0x0a], raw[0x0b]) = (0x40, row.defaults[0x0c]);
        (raw[0x0c], raw[0x0d]) = (0x40, row.defaults[0x0d]);
        for slot in 0..4 {
            let item = row.defaults[0x0e + slot];
            let cell = 0x0e + slot * 2;
            // 0x11383..0x1138B：空格旗標 0x80，有物品的格旗標寫 0（不留殘值）。
            raw[cell] = if item == 0xff { 0x80 } else { 0 };
            raw[cell + 1] = item;
        }
        (raw[0x16], raw[0x18]) = (0x80, 0x80);
        raw[0x1a..0x1e].copy_from_slice(&row.defaults[8..12]);
        raw[0x1e] = 0;
        raw[0x1f] = row.defaults[0];
        raw[0x20] = row.defaults[1];
        raw[0x21] = level as u8;
        raw[0x22..0x28].fill(0); // 0x113C9：0x375C0(+0x22, 0, 6)
        raw[0x31] = 0xff;
        let base_ap = native_join_word(&row.defaults, 0x12) + i64::from(row.growth[0]) * level;
        let base_dp = native_join_word(&row.defaults, 0x14) + i64::from(row.growth[2]) * level;
        let base_dx = native_join_word(&row.defaults, 0x16) + i64::from(row.growth[4]) * level;
        raw[0x37..0x39].copy_from_slice(&(base_ap as u16).to_le_bytes());
        raw[0x39..0x3b].copy_from_slice(&(base_dp as u16).to_le_bytes());
        raw[0x3b] = row.defaults[7];
        raw[0x3c] = 0;
        raw[0x3e..0x40].copy_from_slice(&(base_dx as u16).to_le_bytes());
        raw[0x40..0x42].copy_from_slice(&(max_hp as u16).to_le_bytes());
        raw[0x42..0x44].copy_from_slice(&(max_hp as u16).to_le_bytes());
        raw[0x44..0x46].copy_from_slice(&(max_mp as u16).to_le_bytes());
        raw[0x46..0x48].copy_from_slice(&(max_mp as u16).to_le_bytes());
        battle::apply_native_equipment_recalc(raw, item_table)
            .map_err(|err| CampaignError::from(err).wrap(format!("native JOIN character {id} equipment: ")))?;
        Ok(record)
    }
}

#[cfg(test)]
mod native_join_constructor_test;
