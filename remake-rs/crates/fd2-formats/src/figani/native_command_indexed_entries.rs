//! 原版 `funcs_2AC25` 逐 entry 的 raw mode 契約表。移植自 Go `native_command_indexed_entries.go`。
//!
//! 表格以 `serde_json` 讀取（見 `doc` 模組的刻意不照做說明）；契約檢查與錯誤文字和 Go 版相同。
//! Go 的 nil 與空 slice、nil 與空 map 在驗證中只看長度與鍵是否存在，這裡一律對應空 `Vec`／`BTreeMap`。

use super::doc::{self, Object};
use super::{Cause, FiganiError, msg};
use std::collections::BTreeMap;

#[derive(Debug, Clone, Default, PartialEq, Eq)]
pub struct NativeCommandIndexedEntrySource {
    pub file: String,
    pub size: i64,
    pub md5: String,
    pub sha256: String,
    pub ida_version: String,
    pub address_space: String,
    pub entry_table: String,
    pub consumer: String,
}

#[derive(Debug, Clone, Default, PartialEq, Eq)]
pub struct NativeCommandIndexedSampleMarker {
    pub mode: i64,
    pub channel: i64,
    pub counter: i64,
    pub counter_phase: String,
    pub frame_base_nonzero: Option<bool>,
    pub callee: String,
}

/// 保存原版 `funcs_2AC25` 的逐 entry raw mode 契約；欄位只描述排程，不替畫面、音效或狀態命名。
#[derive(Debug, Clone, Default, PartialEq, Eq)]
pub struct NativeCommandIndexedEntrySchedule {
    pub command_id: i64,
    pub entry: String,
    pub mode_returns: BTreeMap<String, i64>,
    pub init_channels: i64,
    pub draw_channels: i64,
    pub draw_modes: Vec<i64>,
    pub raw_side_zero_x_shift: i64,
    pub offset_tables: Vec<String>,
    pub state_ranges: Vec<String>,
    pub uses_rng: bool,
    pub sample_markers: Vec<NativeCommandIndexedSampleMarker>,
}

#[derive(Debug, Clone, Default, PartialEq, Eq)]
pub struct NativeCommandIndexedEntryTable {
    pub schema_version: i64,
    pub source: NativeCommandIndexedEntrySource,
    pub entries: Vec<NativeCommandIndexedEntrySchedule>,
}

fn marker_doc(o: &Object) -> Result<NativeCommandIndexedSampleMarker, String> {
    Ok(NativeCommandIndexedSampleMarker {
        mode: doc::int(o, "mode")?,
        channel: doc::int(o, "channel")?,
        counter: doc::int(o, "counter")?,
        counter_phase: doc::string(o, "counter_phase")?,
        frame_base_nonzero: doc::opt_bool(o, "frame_base_nonzero")?,
        callee: doc::string(o, "callee")?,
    })
}

fn entry_doc(o: &Object) -> Result<NativeCommandIndexedEntrySchedule, String> {
    Ok(NativeCommandIndexedEntrySchedule {
        command_id: doc::int(o, "command_id")?,
        entry: doc::string(o, "entry")?,
        mode_returns: doc::int_map(o, "mode_returns")?,
        init_channels: doc::int(o, "init_channels")?,
        draw_channels: doc::int(o, "draw_channels")?,
        draw_modes: doc::ints(o, "draw_modes")?,
        raw_side_zero_x_shift: doc::int(o, "raw_side_zero_x_shift")?,
        offset_tables: doc::strings(o, "offset_tables")?,
        state_ranges: doc::strings(o, "state_ranges")?,
        uses_rng: doc::boolean(o, "uses_rng")?,
        sample_markers: doc::objects(o, "sample_markers")?.into_iter().map(marker_doc).collect::<Result<_, _>>()?,
    })
}

fn table_doc(raw: &[u8]) -> Result<NativeCommandIndexedEntryTable, String> {
    let o = doc::parse_object(raw)?;
    let s = doc::object(&o, "source")?;
    Ok(NativeCommandIndexedEntryTable {
        schema_version: doc::int(&o, "schema_version")?,
        source: NativeCommandIndexedEntrySource {
            file: doc::string(s, "file")?,
            size: doc::int(s, "size")?,
            md5: doc::string(s, "md5")?,
            sha256: doc::string(s, "sha256")?,
            ida_version: doc::string(s, "ida_version")?,
            address_space: doc::string(s, "address_space")?,
            entry_table: doc::string(s, "entry_table")?,
            consumer: doc::string(s, "consumer")?,
        },
        entries: doc::objects(&o, "entries")?.into_iter().map(entry_doc).collect::<Result<_, _>>()?,
    })
}

const NATIVE_COMMAND_INDEXED_ENTRIES: [(i64, &str); 9] = [
    (1, "0x262EF"),
    (2, "0x26528"),
    (3, "0x26795"),
    (4, "0x269D3"),
    (5, "0x26BFD"),
    (6, "0x26E39"),
    (7, "0x272B8"),
    (8, "0x274B0"),
    (9, "0x275D6"),
];

/// 已修正的線性 entry 位址（命令序號 → `funcs_2AC25` entry）。
pub fn native_command_indexed_entry_address(command_id: i64) -> Option<&'static str> {
    NATIVE_COMMAND_INDEXED_ENTRIES.iter().find(|(id, _)| *id == command_id).map(|(_, entry)| *entry)
}

fn invalid(message: &'static str) -> FiganiError {
    FiganiError::Static(message)
}

/// 讀入並驗證契約表。Go 版把 `os.ReadFile` 與 `json.Unmarshal` 的錯誤原樣傳回；這裡回傳
/// `FiganiError::Cause`，下層文字刻意不照做 Go 標準函式庫。
pub fn load_native_command_indexed_entry_table(path: &str) -> Result<NativeCommandIndexedEntryTable, FiganiError> {
    let raw = std::fs::read(path).map_err(|source| FiganiError::Cause(Cause::Io { path: path.to_string(), source }))?;
    let table = table_doc(&raw).map_err(|e| FiganiError::Cause(Cause::Json(e)))?;
    let s = &table.source;
    if table.schema_version != 1
        || s.file != "FD2.EXE"
        || s.size != 357074
        || s.md5 != "b97caf2239a27a896069d03549d96e1e"
        || s.sha256 != "222b7d067ad4450eb9c5f6e6bce1797d54bb050417ba39ced6067f8039f28c4f"
        || s.address_space != "IDA linear, image base 0"
        || s.entry_table != "0x523B9"
        || s.consumer != "0x2A6BD"
        || table.entries.len() != NATIVE_COMMAND_INDEXED_ENTRIES.len()
    {
        return Err(invalid("native command indexed entry metadata is invalid"));
    }
    let mut seen = std::collections::BTreeSet::new();
    for entry in &table.entries {
        let id = entry.command_id;
        let want = native_command_indexed_entry_address(id);
        let offsetless = id == 2 || id == 9;
        if want.is_none()
            || seen.contains(&id)
            || Some(entry.entry.as_str()) != want
            || entry.init_channels <= 0
            || entry.draw_channels <= 0
            || entry.draw_channels > entry.init_channels
            || entry.draw_modes.is_empty()
            || (!offsetless && entry.offset_tables.is_empty())
            || (offsetless && !entry.offset_tables.is_empty())
        {
            return Err(msg(format!("native command indexed entry {id} is invalid")));
        }
        for mode in ["0", "3", "6"] {
            if !entry.mode_returns.contains_key(mode) {
                return Err(msg(format!("native command indexed entry {id} mode {mode} is unavailable")));
            }
        }
        for &mode in &entry.draw_modes {
            if !(1..=8).contains(&mode) || mode == 3 || mode == 6 {
                return Err(msg(format!("native command indexed entry {id} draw mode {mode} is invalid")));
            }
        }
        for marker in &entry.sample_markers {
            if marker.mode < 0
                || marker.mode > 8
                || marker.channel >= entry.draw_channels
                || marker.channel < -1
                || (marker.callee != "0x25A96" && marker.callee != "0x25B45")
            {
                return Err(msg(format!("native command indexed entry {id} sample marker is invalid")));
            }
        }
        let ret = |mode: &str| entry.mode_returns.get(mode).copied().unwrap_or(0);
        let callees_ab = ["0x25A96", "0x25B45", "0x25A96", "0x25B45", "0x25A96", "0x25B45"];
        if id == 3 {
            if entry.init_channels != 12
                || entry.draw_channels != 12
                || entry.raw_side_zero_x_shift != 20
                || ret("0") != 2
                || ret("3") != 40
                || ret("6") != 20
                || entry.uses_rng
                || entry.offset_tables != ["0x52460", "0x52490", "0x5249C"]
                || entry.state_ranges != ["0x53F81..0x53FE3"]
                || entry.sample_markers.len() != 6
            {
                return Err(invalid("native command indexed entry 3 direct-instruction contract is invalid"));
            }
            for (index, marker) in entry.sample_markers.iter().enumerate() {
                let want_mode = [2, 2, 5, 5, 8, 8][index];
                let want_counter = [0, 3, 0, 3, 0, 3][index];
                let want_phase = ["pre", "post", "pre", "post", "pre", "post"][index];
                let want_frame_base_nonzero = index % 2 == 0;
                if marker.mode != want_mode
                    || marker.channel != -1
                    || marker.counter != want_counter
                    || marker.counter_phase != want_phase
                    || marker.frame_base_nonzero != Some(want_frame_base_nonzero)
                    || marker.callee != callees_ab[index]
                {
                    return Err(invalid("native command indexed entry 3 sample contract is invalid"));
                }
            }
        }
        if id == 6
            && (entry.init_channels != 5
                || entry.draw_channels != 5
                || ret("0") != 7
                || ret("3") != 12
                || ret("6") != 7
                || entry.uses_rng
                || entry.sample_markers.len() != 3)
        {
            return Err(invalid("native command indexed entry 6 direct-instruction contract is invalid"));
        }
        if id == 5 {
            if entry.init_channels != 6
                || entry.draw_channels != 6
                || ret("0") != 1
                || ret("3") != 12
                || ret("6") != 8
                || !entry.uses_rng
                || entry.offset_tables != ["0x524D0"]
                || entry.sample_markers.len() != 6
            {
                return Err(invalid("native command indexed entry 5 direct-instruction contract is invalid"));
            }
            for (index, marker) in entry.sample_markers.iter().enumerate() {
                let want_mode = [2, 2, 5, 5, 8, 8][index];
                let want_channel = [0, 3, 0, 3, 0, 3][index];
                if marker.mode != want_mode
                    || marker.channel != want_channel
                    || marker.counter != 0
                    || marker.callee != callees_ab[index]
                {
                    return Err(invalid("native command indexed entry 5 sample contract is invalid"));
                }
            }
        }
        if id == 7 {
            if entry.init_channels != 4
                || entry.draw_channels != 3
                || ret("0") != 2
                || ret("3") != 32
                || ret("6") != 16
                || entry.uses_rng
                || entry.offset_tables != ["0x52511"]
                || entry.sample_markers.len() != 6
            {
                return Err(invalid("native command indexed entry 7 direct-instruction contract is invalid"));
            }
            for (index, marker) in entry.sample_markers.iter().enumerate() {
                let want_mode = [2, 2, 5, 5, 8, 8][index];
                let want_channel = [0, 1, 0, 1, 0, 1][index];
                if marker.mode != want_mode
                    || marker.channel != want_channel
                    || marker.counter != 1
                    || marker.callee != callees_ab[index]
                {
                    return Err(invalid("native command indexed entry 7 sample contract is invalid"));
                }
            }
        }
        if id == 8 {
            if entry.init_channels != 16
                || entry.draw_channels != 16
                || entry.raw_side_zero_x_shift != 0
                || ret("0") != 3
                || ret("3") != 34
                || ret("6") != 2
                || entry.uses_rng
                || entry.offset_tables != ["0x52539"]
                || entry.state_ranges != ["0x540BA..0x540F9"]
                || entry.sample_markers.len() != 4
            {
                return Err(invalid("native command indexed entry 8 direct-instruction contract is invalid"));
            }
            for (index, marker) in entry.sample_markers.iter().enumerate() {
                let want_mode = [2, 2, 5, 5][index];
                let want_counter = [0, 4, 0, 4][index];
                if marker.mode != want_mode
                    || marker.channel != -1
                    || marker.counter != want_counter
                    || marker.counter_phase != "pre"
                    || marker.frame_base_nonzero.is_some()
                    || marker.callee != callees_ab[index]
                {
                    return Err(invalid("native command indexed entry 8 sample contract is invalid"));
                }
            }
        }
        if id == 9 {
            if entry.init_channels != 1
                || entry.draw_channels != 1
                || entry.raw_side_zero_x_shift != 0
                || ret("0") != 20
                || ret("3") != 60
                || ret("6") != 20
                || entry.uses_rng
                || !entry.offset_tables.is_empty()
                || entry.state_ranges != ["0x540FA..0x540FB"]
                || entry.sample_markers.len() != 2
            {
                return Err(invalid("native command indexed entry 9 direct-instruction contract is invalid"));
            }
            for (index, marker) in entry.sample_markers.iter().enumerate() {
                let want_counter = [6, 36][index];
                if marker.mode != 5
                    || marker.channel != -1
                    || marker.counter != want_counter
                    || marker.counter_phase != "pre"
                    || marker.frame_base_nonzero.is_some()
                    || marker.callee != callees_ab[index]
                {
                    return Err(invalid("native command indexed entry 9 sample contract is invalid"));
                }
            }
        }
        seen.insert(id);
    }
    Ok(table)
}

impl NativeCommandIndexedEntryTable {
    /// 取出某個指令的 entry。Go 版的 nil 表由 Rust 型別排除。
    pub fn schedule(&self, command_id: i64) -> Result<NativeCommandIndexedEntrySchedule, FiganiError> {
        self.entries
            .iter()
            .find(|e| e.command_id == command_id)
            .cloned()
            .ok_or_else(|| msg(format!("native command indexed entry {command_id} unavailable")))
    }
}
