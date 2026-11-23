//! Go 來源：`remake/cmd/fd2/save.go`（422 行）。
//!
//! save.go — 存檔/讀檔(自有 JSON 格式,doc19;非破解原版 FD2.SAV)。
//! 語意:存「campaign 節點邊界」進度(目前節點/旗標/金幣/道具)。戰鬥中我方可操作時另附
//! battle_snapshot(battle_snapshot.go),讀檔回到同一回合;其他時刻戰鬥中拒絕存檔。
//!
//! 骨架：Go 檔的 13 個函式中產生 7 個（階段 0 清單），其餘標 `r8`，見 `PORTING.json`。
//! F 組另外移植同檔的 `savePath`、`validateSavePartyTopology`、`saveGame`、`loadGame`、
//! `loadGameFromSlot`、`loadJSONGameFromSlot`（R4 要能讀 Go 版存檔；`confirmTitleLoadSlot` 的 JSON
//! 分支與 Go 測試都經過它們），以及 `saveData`／`battleSnapshot` 兩個 JSON 型別。
//!
//! # JSON 與 Go `encoding/json` 的對應
//!
//! - 欄位名、順序與 `omitempty` 逐欄照 Go 標籤（`rename`、宣告順序、`skip_serializing_if`）。
//! - Go nil 切片／map 寫 `null`、空的寫 `[]`／`{}`。`items`、`flags` 沒有 `omitempty`，型別用 `Option`
//!   區分兩者，讀入再寫出與 Go 原檔相同；`omitempty` 的欄位 nil 與空都省略，不需要區分。
//! - Go 以整數為鍵的 map 依鍵的十進位字串排序寫出（`"10"` 在 `"2"` 前面）；本檔的整數鍵 map 以
//!   [`go_json_int_key_map`] 依同樣順序寫出。
//! - `json.MarshalIndent(v, "", " ")` 對應 [`go_json_marshal_indent`]：serde_json 的 `PrettyFormatter`
//!   縮排一個空白，格式（`"key": value`、空容器 `{}`／`[]`、結尾不換行）與 Go 相同；浮點數照 Go
//!   `encoding/json` 寫（`crate::go_json_float`，整數值寫 `0` 而非 `0.0`）。
//! - battle 套件型別（名冊單位、戰場快照）的 nil 切片與 map 由 fd2-rules 的 `NilableVec`／`NilableMap` 保留。
//! - 刻意不照做的 Go 行為列在本模組的 `KNOWN_DIFFERENCES`（測試與黃金共用）。

use super::*;

/// 刻意不照做的 Go `encoding/json`／`os` 行為（`rust-port-no-go-stdlib`）。黃金案例不落在這些輸入上。
#[cfg(test)]
pub(crate) const SAVE_JSON_KNOWN_DIFFERENCES: &[&str] = &[
    // Go 解碼時欄位名不分大小寫；Rust 只認標籤原文（同 fd2-rules）。
    "JSON 欄位名大小寫",
    // Go 讀 JSON null 到 int、bool、string 欄位時保留原值。本 crate 的型別以 `null_as_default` 照做
    // （起點都是零值）；battle 套件的型別（Unit、StateSnapshot 等）的純量欄位拒收 null（同 fd2-rules）。
    "battle 套件型別的純量 null",
    // Go Unmarshal 遇到型別不符的欄位會略過它、繼續填其他欄位再回報錯誤；Rust 整份拒收。
    // loadSettings 忽略錯誤，所以設定檔有型別錯誤時 Go 保留其他欄位，Rust 全部回預設。
    "型別錯誤時的部分填入",
    // Go 的字串寫出時把 <、>、&、U+2028、U+2029 寫成 \u 加四位十六進位，控制字元也一律用這種形式；
    // serde_json 原樣寫出（控制字元用 \b、\f 等短形式）。存檔與設定的字串不含這些字元。
    "字串的 HTML 與控制字元跳脫",
    // Go 讀整數鍵接受 strconv.ParseInt 能解析的寫法（例如 "+1"、"01"）；serde_json 只收一般十進位。
    "整數鍵的非標準寫法",
    // os 錯誤文字（開檔、改名失敗等）由 Rust 標準函式庫產生，與 Go 不同；只比成功或失敗。
    "作業系統錯誤文字",
    // Go 的 NaN／Inf 寫出時回錯誤；serde_json 寫成 null。存檔沒有非有限小數。
    "非有限小數",
    // serde 衍生的結構接受 JSON 陣列寫法（依欄位順序填入），Go 拒收；頂層由 go_json_unmarshal_zero 擋下，
    // 巢狀的結構欄位（例如 "battle_snapshot": []）Rust 會接受。
    "巢狀結構欄位的陣列寫法",
    // Go 讀到重複的鍵以最後一個為準；serde 衍生的結構拒收。設定檔先讀成物件再解碼，與 Go 相同；
    // 存檔與附檔由程式寫出，不會有重複的鍵。
    "存檔與附檔 JSON 的重複鍵",
];

/// Go `encoding/json` 對整數鍵 map 的寫出順序：鍵先轉成十進位字串再依位元組排序。讀取與
/// `fd2_rules::battle::null_as_default` 相同（`null` 讀成空 map）。
pub(crate) mod go_json_int_key_map {
    use serde::ser::SerializeMap;
    use serde::{Deserialize, Deserializer, Serialize, Serializer};
    use std::collections::BTreeMap;

    pub(crate) fn serialize<V: Serialize, S: Serializer>(
        v: &BTreeMap<i64, V>,
        s: S,
    ) -> std::result::Result<S::Ok, S::Error> {
        let mut keys: Vec<(String, &V)> = v.iter().map(|(k, v)| (k.to_string(), v)).collect();
        keys.sort_by(|a, b| a.0.cmp(&b.0));
        let mut map = s.serialize_map(Some(keys.len()))?;
        for (k, v) in keys {
            map.serialize_entry(&k, v)?;
        }
        map.end()
    }

    pub(crate) fn deserialize<'de, V: Deserialize<'de>, D: Deserializer<'de>>(
        d: D,
    ) -> std::result::Result<BTreeMap<i64, V>, D::Error> {
        Ok(Option::<BTreeMap<i64, V>>::deserialize(d)?.unwrap_or_default())
    }
}

/// `Option<battle::BalanceOptions>` 以 Go 的整數鍵順序寫出。`fd2_rules` 的 `BalanceOptions` serde 把
/// `characters`、`skill_hit` 依整數鍵升冪寫出（見 PORTING.md「fd2-rules／fd2-campaign 缺口」），本檔與
/// `balance_save.rs` 寫出它時改走這個鏡像，欄位名與順序照 Go 標籤（`balance.go:7`）。讀取沿用
/// `BalanceOptions` 自己的 serde。
pub(crate) mod go_json_balance_options {
    use super::battle;
    use serde::{Serialize, Serializer};
    use std::collections::BTreeMap;

    #[derive(Serialize)]
    struct BalanceOptionsGo<'a> {
        growth: &'a battle::GrowthRates,
        #[serde(skip_serializing_if = "BTreeMap::is_empty")]
        characters: BTreeMap<String, &'a battle::GrowthRates>,
        #[serde(skip_serializing_if = "BTreeMap::is_empty")]
        skill_hit: BTreeMap<String, BTreeMap<String, i64>>,
        enemy_percent: i64,
    }

    pub(crate) fn serialize<S: Serializer>(
        v: &Option<battle::BalanceOptions>,
        s: S,
    ) -> std::result::Result<S::Ok, S::Error> {
        let Some(v) = v else {
            return s.serialize_none();
        };
        // 字串鍵的 BTreeMap 依位元組排序，與 Go 對整數鍵轉成字串後的排序相同。
        BalanceOptionsGo {
            growth: &v.growth,
            characters: v.characters.iter().map(|(k, r)| (k.to_string(), r)).collect(),
            skill_hit: v
                .skill_hit
                .iter()
                .map(|(k, rows)| (k.to_string(), rows.iter().map(|(c, n)| (c.to_string(), *n)).collect()))
                .collect(),
            enemy_percent: v.enemy_percent,
        }
        .serialize(s)
    }
}

/// Go `json.Unmarshal(raw, &v)`，`v` 是零值：頂層 `null` 不改任何欄位、不回錯誤，結果是零值。
///
/// serde 衍生的結構也接受 JSON 陣列寫法（依欄位順序），Go 拒收；頂層先擋下陣列，巢狀欄位的陣列寫法
/// 列在 `SAVE_JSON_KNOWN_DIFFERENCES`。
pub(crate) fn go_json_unmarshal_zero<T: serde::de::DeserializeOwned + Default>(raw: &[u8]) -> serde_json::Result<T> {
    if raw.iter().find(|b| !matches!(b, b' ' | b'\t' | b'\n' | b'\r')) == Some(&b'[') {
        return Err(<serde_json::Error as serde::de::Error>::custom("cannot unmarshal array into Go struct"));
    }
    Ok(serde_json::from_slice::<Option<T>>(raw)?.unwrap_or_default())
}

/// Go `json.MarshalIndent(v, "", " ")`。
pub(crate) fn go_json_marshal_indent<T: serde::Serialize>(v: &T) -> Result<Vec<u8>> {
    let mut out = Vec::new();
    let mut ser =
        serde_json::Serializer::with_formatter(&mut out, crate::go_json_float::GoJsonFormatter::with_indent(b" "));
    v.serialize(&mut ser).map_err(|err| GameError::new(err.to_string()))?;
    Ok(out)
}

/// 呼叫端的 slot 轉成 `fdsave` 的 `usize`；負數照 Go `fdsave.SlotBounds` 的錯誤文字拒收。
pub(crate) fn fdsave_slot(slot: i64) -> Result<usize> {
    if slot < 0 {
        return Err(game_err!("fdsave: slot {slot} outside 0..{}", fdsave::SLOT_COUNT - 1));
    }
    Ok(slot as usize)
}

/// savePath/saveSlotPath 存檔位置:$XDG_DATA_HOME/fd2_re/fd2_save_N.json；四槽
/// selector 對應 native 0x30550 的 UI contract，但 JSON 不是原版 FD2.SAV ABI。
///
/// Go：`save.go:19`，`func savePath() string`（F 組補移植，原標 `r8`）
pub(crate) fn save_path() -> String {
    save_slot_path(0)
}

/// Go：`save.go:21`，`func saveSlotPath(slot int) string`
pub(crate) fn save_slot_path(slot: i64) -> String {
    let mut slot = slot;
    if !(0..=3).contains(&slot) {
        slot = 0;
    }
    if slot == 0 {
        // Preserve the pre-selector single-save file as slot 1.
        return user_data_path("fd2_save.json");
    }
    user_data_path(&format!("fd2_save_{slot}.json"))
}

/// 在 `dir` 建立名稱以 `prefix` 開頭的新暫存檔（Go `os.CreateTemp(dir, prefix+"*")`）。Go 的後綴是
/// 亂數；名稱不影響結果，Rust 以行程編號與計數器保證不重複，`create_new` 拒絕覆蓋既有檔。
pub(crate) fn create_temp_in(
    dir: &std::path::Path,
    prefix: &str,
) -> std::io::Result<(std::fs::File, std::path::PathBuf)> {
    use std::sync::atomic::{AtomicU64, Ordering};
    static NEXT: AtomicU64 = AtomicU64::new(0);
    loop {
        let n = NEXT.fetch_add(1, Ordering::Relaxed);
        let path = dir.join(format!("{prefix}{}{n}", std::process::id()));
        match std::fs::OpenOptions::new().write(true).create_new(true).open(&path) {
            Ok(file) => return Ok((file, path)),
            Err(err) if err.kind() == std::io::ErrorKind::AlreadyExists => continue,
            Err(err) => return Err(err),
        }
    }
}

/// Go `(*os.File).Chmod(mode)`。非 Unix 平台 Go 只切換唯讀屬性，這裡的權限都可寫，所以不動。
pub(crate) fn chmod_file(file: &std::fs::File, mode: u32) -> std::io::Result<()> {
    #[cfg(unix)]
    {
        use std::os::unix::fs::PermissionsExt;
        file.set_permissions(std::fs::Permissions::from_mode(mode))
    }
    #[cfg(not(unix))]
    {
        let _ = (file, mode);
        Ok(())
    }
}

/// Go 改名後 `os.Open(dir)` 再 `Sync`，錯誤忽略。
pub(crate) fn sync_dir(dir: &std::path::Path) {
    if let Ok(handle) = std::fs::File::open(dir) {
        let _ = handle.sync_all();
    }
}

fn io_err(err: std::io::Error) -> GameError {
    GameError::new(err.to_string())
}

/// writeSaveFile replaces the save in one rename. Campaign progress is only
/// persisted at node boundaries, so a truncated JSON file must never turn a
/// valid town/preparation save into an unreadable slot after a process stop.
///
/// Go：`save.go:35`，`func writeSaveFile(path string, raw []byte) error`
pub(crate) fn write_save_file(path: &str, raw: &[u8]) -> Result<()> {
    use std::io::Write as _;
    // Go `filepath.Dir`：沒有目錄部分時是 "."。
    let dir = match std::path::Path::new(path).parent() {
        Some(p) if !p.as_os_str().is_empty() => p.to_path_buf(),
        _ => std::path::PathBuf::from("."),
    };
    let (mut tmp, tmp_name) = create_temp_in(&dir, ".fd2-save-").map_err(io_err)?;
    // Go `defer os.Remove(tmpName)`：改名成功後暫存名已不存在，移除失敗被忽略。
    struct RemoveOnDrop(std::path::PathBuf);
    impl Drop for RemoveOnDrop {
        fn drop(&mut self) {
            let _ = std::fs::remove_file(&self.0);
        }
    }
    let _remove = RemoveOnDrop(tmp_name.clone());
    chmod_file(&tmp, 0o644).map_err(io_err)?;
    tmp.write_all(raw).map_err(io_err)?;
    tmp.sync_all().map_err(io_err)?;
    drop(tmp);
    std::fs::rename(&tmp_name, path).map_err(io_err)?;
    sync_dir(&dir);
    Ok(())
}

/// Go `battleSnapshot`（`battle_snapshot.go:20`）：戰鬥中我方可操作時附在存檔的戰場快照。
///
/// Go 把型別放在 `battle_snapshot.go`；該檔在第一章沒有執行到，骨架沒有它的子模組，F 組先把
/// JSON 型別放在這裡（讀 Go 存檔需要），擷取、驗證與還原的函式留給 R8。
#[derive(Debug, Clone, Default, PartialEq, serde::Serialize, serde::Deserialize)]
#[serde(default)]
pub(crate) struct BattleSnapshot {
    /// Go 標籤：`json:"version"`
    #[serde(deserialize_with = "battle::null_as_default")]
    pub(crate) version: i64,
    /// Go 標籤：`json:"node"`
    #[serde(deserialize_with = "battle::null_as_default")]
    pub(crate) node: String,
    /// Go 標籤：`json:"units"`
    #[serde(deserialize_with = "battle::null_as_default")]
    pub(crate) units: String,
    /// Go 標籤：`json:"scenario"`
    #[serde(deserialize_with = "battle::null_as_default")]
    pub(crate) scenario: String,
    /// Go 標籤：`json:"map_sha256"`
    #[serde(deserialize_with = "battle::null_as_default")]
    pub(crate) map_sha256: String,
    /// Go 標籤：`json:"state"`
    pub(crate) state: battle::StateSnapshot,
    /// Go 標籤：`json:"fired_events,omitempty"`
    #[serde(deserialize_with = "battle::null_as_default", skip_serializing_if = "Vec::is_empty")]
    pub(crate) fired_events: Vec<battle::FiredEventRef>,
    /// Go 標籤：`json:"native_rng_state"`
    #[serde(deserialize_with = "battle::null_as_default")]
    pub(crate) native_rng_state: u16,
    /// Go 標籤：`json:"cursor_x"`
    #[serde(deserialize_with = "battle::null_as_default")]
    pub(crate) cursor_x: i64,
    /// Go 標籤：`json:"cursor_y"`
    #[serde(deserialize_with = "battle::null_as_default")]
    pub(crate) cursor_y: i64,
    /// Go 標籤：`json:"native_map_clock_tick"`（指標、沒有 `omitempty`：nil 寫 `null`）
    pub(crate) native_map_clock_tick: Option<i64>,
}

/// Go `saveData`（`save.go:68`）：重製端自有 JSON 存檔。
#[derive(Debug, Clone, Default, PartialEq, serde::Serialize, serde::Deserialize)]
#[serde(default)]
pub(crate) struct SaveData {
    /// Go 標籤：`json:"balance,omitempty"`
    #[serde(serialize_with = "go_json_balance_options::serialize", skip_serializing_if = "Option::is_none")]
    pub(crate) balance: Option<battle::BalanceOptions>,
    /// Go 標籤：`json:"node"`
    #[serde(deserialize_with = "battle::null_as_default")]
    pub(crate) node: String,
    /// Go 標籤：`json:"flags"`（沒有 `omitempty`：nil 寫 `null`、空的寫 `{}`）
    pub(crate) flags: Option<BTreeMap<String, bool>>,
    /// Go 標籤：`json:"gold"`
    #[serde(deserialize_with = "battle::null_as_default")]
    pub(crate) gold: i64,
    /// Go 標籤：`json:"items"`（沒有 `omitempty`：nil 寫 `null`、空的寫 `[]`）
    pub(crate) items: Option<Vec<String>>,
    /// Go 標籤：`json:"party_members,omitempty"`
    #[serde(with = "go_json_int_key_map", skip_serializing_if = "BTreeMap::is_empty")]
    pub(crate) party_members: BTreeMap<i64, bool>,
    /// Go 標籤：`json:"party_join_order,omitempty"`
    #[serde(deserialize_with = "battle::null_as_default", skip_serializing_if = "Vec::is_empty")]
    pub(crate) party_join_order: Vec<i64>,
    /// Go 標籤：`json:"party_deploy,omitempty"`
    #[serde(with = "go_json_int_key_map", skip_serializing_if = "BTreeMap::is_empty")]
    pub(crate) party_deploy: BTreeMap<i64, bool>,
    /// Go 標籤：`json:"party_roster,omitempty"`
    #[serde(with = "go_json_int_key_map", skip_serializing_if = "BTreeMap::is_empty")]
    pub(crate) party_roster: BTreeMap<i64, battle::Unit>,
    /// Go 標籤：`json:"chapter,omitempty"`
    #[serde(deserialize_with = "battle::null_as_default", skip_serializing_if = "battle::go_is_zero")]
    pub(crate) chapter: i64,
    /// Go 標籤：`json:"native_hud_gate_a,omitempty"`
    #[serde(skip_serializing_if = "Option::is_none")]
    pub(crate) native_hud_gate_a: Option<i64>,
    /// Go 標籤：`json:"native_raw_53af9,omitempty"`
    #[serde(skip_serializing_if = "Option::is_none")]
    pub(crate) native_raw_53af9: Option<i64>,
    /// Go 標籤：`json:"native_raw_51e61,omitempty"`
    #[serde(skip_serializing_if = "Option::is_none")]
    pub(crate) native_raw_51e61: Option<i64>,
    /// Go 標籤：`json:"native_raw_51e62,omitempty"`
    #[serde(skip_serializing_if = "Option::is_none")]
    pub(crate) native_raw_51e62: Option<i64>,
    /// Go 標籤：`json:"battle_snapshot,omitempty"`
    #[serde(skip_serializing_if = "Option::is_none")]
    pub(crate) battle_snapshot: Option<BattleSnapshot>,
}

/// validateSavePartyTopology 在 loadGameFromSlot 發布戰役或介面狀態前，拒絕不完整的
/// 持續隊伍交易。JSON 存檔雖是重製邊界而非原版 FD2.SAV ABI，其 JOIN 順序、
/// membership、部署與名冊仍必須描述同一支一致的隊伍。
///
/// Go：`save.go:91`，`func validateSavePartyTopology(d saveData) error`（F 組補移植，原標 `r8`）
/// Go 以 map 迭代順序檢查；多筆錯誤同時存在時 Go 報哪一筆隨順序而定，Rust 依鍵的升冪。
pub(crate) fn validate_save_party_topology(d: &SaveData) -> Result<()> {
    if d.party_join_order.is_empty()
        && d.party_members.is_empty()
        && d.party_deploy.is_empty()
        && d.party_roster.is_empty()
    {
        return Ok(());
    }
    let member = |id: &i64| d.party_members.get(id).copied().unwrap_or(false);
    let mut seen = BTreeSet::new();
    for id in &d.party_join_order {
        if seen.contains(id) {
            return Err(game_err!("存檔持續隊伍的 JOIN 順序重複角色 {id}"));
        }
        seen.insert(*id);
        if !member(id) {
            return Err(game_err!("存檔持續隊伍的 JOIN 角色 {id} 不在隊伍中"));
        }
    }
    for (id, is_member) in &d.party_members {
        if !is_member {
            continue;
        }
        if !seen.contains(id) {
            return Err(game_err!("存檔隊伍角色 {id} 缺少 JOIN 順序"));
        }
    }
    for (id, deployed) in &d.party_deploy {
        if *deployed && (!member(id) || !seen.contains(id)) {
            return Err(game_err!("存檔部署角色 {id} 不在持續隊伍中"));
        }
        if *deployed && !d.party_roster.contains_key(id) {
            return Err(game_err!("存檔部署角色 {id} 缺少名冊紀錄"));
        }
    }
    for id in d.party_roster.keys() {
        if !member(id) || !seen.contains(id) {
            return Err(game_err!("存檔名冊角色 {id} 不在持續隊伍順序中"));
        }
    }
    Ok(())
}

impl Game {
    /// Go：`save.go:131`，`func (g *Game) saveGame()`（F 組補移植，原標 `r8`）
    pub(crate) fn save_game(&mut self) {
        let _ = self.save_game_to_slot(0);
    }

    /// Go：`save.go:133`，`func (g *Game) saveGameToSlot(slot int) error`
    pub(crate) fn save_game_to_slot(&mut self, slot: i64) -> Result<()> {
        self.save_game_to_slot_from(slot, false)
    }

    /// 由目前狀態組出 `saveData`（Go `save.go:172`～`191`，不含快照）。
    ///
    /// Rust：`Game` 的 `items` 是 `Vec`，分不出 Go 的 nil 與空切片；空的寫 `null`（Go 版 `items`
    /// 只由讀檔設定，沒有讀過存檔時是 nil）。`flags` 來自 `Runner`，Go `NewRunner` 一律建立非 nil
    /// map，所以寫 `{}`。
    pub(crate) fn build_save_data(&self) -> SaveData {
        let camp = self.camp.as_ref().expect("save.go:172 g.camp 已檢查非 nil");
        let mut d = SaveData {
            node: camp.cur.clone(),
            flags: Some(camp.flags.clone()),
            gold: self.gold,
            items: if self.items.is_empty() { None } else { Some(self.items.clone()) },
            party_members: self.party_members.clone(),
            party_join_order: self.party_join_order.clone(),
            party_deploy: self.party_deploy.clone(),
            party_roster: self.party_roster.clone(),
            chapter: self.handler_chapter,
            ..Default::default()
        };
        if self.native_map_hud_persistent.has_display_gate_a {
            d.native_hud_gate_a = Some(i64::from(self.native_map_hud_persistent.display_gate_a));
        }
        let mut options = self.current_native_system_options();
        if self.native_map_hud_persistent.has_display_gate_a {
            options.raw_51aab = self.native_map_hud_persistent.display_gate_a;
        }
        d.native_raw_53af9 = Some(i64::from(options.raw_53af9));
        d.native_raw_51e61 = Some(i64::from(options.raw_51e61));
        d.native_raw_51e62 = Some(i64::from(options.raw_51e62));
        // Go `!g.balance.IsOriginal()`：nil 視為原版。
        if let Some(balance) = self.balance.as_ref().filter(|b| !b.is_original()) {
            d.balance = Some(balance.clone());
        }
        d
    }

    /// saveGameToSlotFrom 的 fromSystemMenu 由戰鬥系統選單的存檔入口使用；見
    /// battleSnapshotBlockerFor。
    ///
    /// Go：`save.go:137`，`func (g *Game) saveGameToSlotFrom(slot int, fromSystemMenu bool) error`
    pub(crate) fn save_game_to_slot_from(&mut self, slot: i64, from_system_menu: bool) -> Result<()> {
        if self.native_shop_mode == "sell_success" {
            return Err(game_err!("請等販賣完成後再存檔"));
        }
        if self.native_end_turn_recovery.is_some() {
            return Err(game_err!("save blocked during native END recovery"));
        }
        let Some(camp) = self.camp.as_ref() else {
            let (message, ok) = self.locale_message("save.unsupported", &[]);
            if ok {
                self.msg = message;
            }
            return Err(game_err!("save campaign unavailable"));
        };
        let (postbattle, battle_node) = match camp.node() {
            Some(n) => (
                n.r#type == "cutscene"
                    && ((!n.handler_binding.is_empty() && self.st.is_some())
                        || camp.node_id().starts_with("postbattle_")),
                n.r#type == "battle" && self.st.is_some(),
            ),
            None => (false, false),
        };
        if postbattle {
            // Post-battle handlers intentionally retain the completed canonical battle
            // array for slot predicates, rewards, SPAWN, ACT and sync_party. The save
            // format is node-boundary-only and does not serialize that transient array;
            // saving this node would reload into a guaranteed fail-closed context. This
            // applies equally to an unbound node: it must not create a fake save that
            // appears to have crossed the persistence boundary.
            let (message, ok) = self.locale_message("save.postbattle_blocked", &[]);
            if ok {
                self.msg = message;
            }
            return Err(game_err!("save transient postbattle node blocked"));
        }
        let snapshot: Option<BattleSnapshot> = None;
        if battle_node {
            // 戰鬥中只在我方可操作的地圖待機存快照；其他時刻存節點邊界會讓讀檔
            // 回到本戰開頭，與玩家按 F5 的意圖不符，所以拒絕並明示原因。
            let _ = from_system_menu;
            todo!("R8: battle_snapshot.go:captureBattleSnapshot（第一章受控流程不在戰鬥中存檔）");
        }
        self.capture_native_map_hud_persistence();
        let mut d = self.build_save_data();
        d.battle_snapshot = snapshot.clone();
        let raw = go_json_marshal_indent(&d)?;
        let write_result = write_save_file(&save_slot_path(slot), &raw);
        if write_result.is_ok() {
            let cur = self.camp.as_ref().map(|c| c.cur.clone()).unwrap_or_default();
            let (message, ok) = self.locale_message("save.saved", &[(slot + 1).into(), cur.into()]);
            if ok {
                self.msg = message;
            }
        }
        // 正常 START 或原版四槽 LOAD，且有明示可寫的 FD2.SAV 覆蓋層時，同步寫回原版槽
        // （0x30012）。失敗只留訊息，不回滾重製端自有存檔。
        // 原版章節槽寫入端 0x30012 只由整備與酒店呼叫；戰鬥中的快照不同步原版槽。
        if snapshot.is_none()
            && (self.native_chapter_slot_baseline.is_some() || self.native_chapter_slot_started)
            && !native_current_save_path().is_empty()
        {
            if let Err(err) = self.save_native_chapter_slot(slot) {
                self.msg = err.message().to_string();
                self.native_chapter_slot_save_err = Some(err.clone());
                return Err(err);
            }
            self.native_chapter_slot_save_err = None;
        }
        write_result
    }

    /// Go：`save.go:217`，`func (g *Game) loadGame()`（F 組補移植，原標 `r8`）
    pub(crate) fn load_game(&mut self) {
        self.load_game_from_slot(0);
    }

    /// Go：`save.go:219`，`func (g *Game) loadGameFromSlot(slot int)`（F 組補移植，原標 `r8`）
    pub(crate) fn load_game_from_slot(&mut self, slot: i64) {
        self.load_json_game_from_slot(slot, false);
    }

    /// Go：`save.go:223`，`func (g *Game) loadJSONGameFromSlot(slot int, inHotel bool) bool`
    /// （F 組補移植，原標 `r8`）
    ///
    /// 存檔帶 `battle_snapshot` 時的驗證與還原（`battle_snapshot.go`）留給 R8，停在 `todo!()`。
    pub(crate) fn load_json_game_from_slot(&mut self, slot: i64, in_hotel: bool) -> bool {
        if self.camp.is_none() {
            return false;
        }
        let raw = match std::fs::read(save_slot_path(slot)) {
            Ok(raw) => raw,
            Err(_) => {
                let (message, ok) = self.locale_message("save.none", &[]);
                if ok {
                    self.msg = message;
                }
                return false;
            }
        };
        let mut d: SaveData = match go_json_unmarshal_zero(&raw) {
            Ok(d) => d,
            Err(_) => return false,
        };
        if d.chapter < 0 || d.chapter > 0xff {
            return false;
        }
        if let Some(Err(err)) = d.balance.as_ref().map(|b| b.validate()) {
            self.msg = err.message().to_string();
            return false;
        }
        // Go `g.camp.C.Nodes[d.Node]`：C 是 nil 時 Go 會 panic。
        let graph = self.camp.as_ref().and_then(|c| c.c.as_ref()).expect("save.go:242 g.camp.C 為 nil");
        if !graph.nodes.contains_key(&d.node) {
            let (message, formatted) = self.locale_message("save.node_missing", &[d.node.clone().into()]);
            if formatted {
                self.msg = message;
            }
            return false;
        }
        if d.battle_snapshot.is_some() && !in_hotel {
            self.msg = "目前無法還原戰場中途存檔".into();
            return false;
        }
        if in_hotel {
            // 酒店讀檔回到城鎮節點，戰場快照不適用。
            d.battle_snapshot = None;
            let graph = self.camp.as_ref().and_then(|c| c.c.as_ref()).expect("save.go:258 g.camp.C 為 nil");
            let mut town = d.node.clone();
            if let Some(node) = graph.nodes.get(&d.node).filter(|n| n.r#type == "hotel") {
                town = node.next.clone();
            }
            match native_hotel_node_for_town(Some(graph), &town) {
                Ok(hotel) => d.node = hotel,
                Err(err) => {
                    self.msg = err.message().to_string();
                    return false;
                }
            }
        }
        if let Err(err) = validate_save_party_topology(&d) {
            self.msg = err.message().to_string();
            return false;
        }
        let mut options = self.current_native_system_options();
        if let Some(gate_a) = d.native_hud_gate_a {
            if !(0..=1).contains(&gate_a) {
                self.msg = "存檔 native HUD gate A 超出原始布林範圍".to_string();
                return false;
            }
            options.raw_51aab = gate_a as u8;
        }
        for (value, target) in [
            (d.native_raw_53af9, &mut options.raw_53af9),
            (d.native_raw_51e61, &mut options.raw_51e61),
            (d.native_raw_51e62, &mut options.raw_51e62),
        ] {
            let Some(value) = value else {
                continue;
            };
            if !(0..=1).contains(&value) {
                self.msg = "存檔 native 系統設定超出原始布林範圍".to_string();
                return false;
            }
            *target = value as u8;
        }
        if options.validate().is_err() {
            self.msg = "存檔 native 系統設定無效".to_string();
            return false;
        }
        // 節點邊界LOAD丟棄舊戰場owner，不能執行其續行或把舊經驗對話套到新章首。
        // 必須在驗證後、替換st之前清除移動目標，避免改寫新載入的戰場。
        self.clear_native_player_movement();
        self.native_level_up_dialogue = None;
        self.pending_native_level_ups = Vec::new();
        self.capture_native_map_hud_persistence();
        let node = d.node.clone();
        {
            let camp = self.camp.as_mut().expect("save.go:307 g.camp 已檢查非 nil");
            camp.cur = d.node.clone();
            // Go 讀 `"flags": null` 得到 nil map；Rust 的 `Runner::flags` 不分 nil，讀成空的。
            camp.flags = d.flags.take().unwrap_or_default();
        }
        self.gold = d.gold;
        self.items = d.items.take().unwrap_or_default();
        self.party_members = std::mem::take(&mut d.party_members);
        self.party_join_order = std::mem::take(&mut d.party_join_order);
        self.party_deploy = std::mem::take(&mut d.party_deploy);
        self.party_roster = std::mem::take(&mut d.party_roster);
        self.handler_chapter = d.chapter;
        self.balance = d.balance.take();
        self.balance_ui = BalanceEditor::default();
        if let Some(gate_a) = d.native_hud_gate_a {
            self.restore_native_map_hud_gate_a(gate_a as u8);
        }
        self.native_system_options = Some(options);
        // 節點邊界存檔不保留舊戰場陣列；先清除它也可避免 enterNode 以讀檔前的
        // gate A 覆蓋剛還原的存檔值。教會選單、候選與 indexed 工作同屬未序列化
        // 暫態，必須在進入存檔節點前一併丟棄。
        self.dialog = Vec::new();
        self.st = None;
        self.sel = None;
        // F9 can be pressed while the battle inspector owns the final frame.
        // Its pixels and lifecycle belong to the discarded scene, not the save.
        self.native_player_status = None;
        self.native_chapter_restore = None;
        self.clear_church_transient_state_for_load();
        self.clear_shop_transient_state_for_load();
        self.native_hotel_prompt_return = false;
        self.load_err = String::new();
        self.enter_node();
        let (message, ok) = self.locale_message("save.loaded", &[(slot + 1).into(), node.into()]);
        if ok {
            self.msg = message;
        }
        self.load_err.is_empty()
    }

    /// Go：`save.go:348`，`func (g *Game) clearShopTransientStateForLoad()`
    pub(crate) fn clear_shop_transient_state_for_load(&mut self) {
        self.native_shop_ui_job = None;
        self.reset_native_shop_ui_pulse();
        self.native_shop_variant = 0;
        self.native_shop_mode = String::new();
        self.native_shop_service_sel = 0;
        self.native_shop_item_start = 0;
        self.native_shop_confirm_sel = 0;
        self.native_shop_recipient_start = 0;
        self.native_shop_recipient_cycle = 0;
        self.native_shop_equip_sel = 0;
        self.native_shop_pending_unit = battle::Unit::default();
        self.native_shop_has_pending_unit = false;
        self.native_shop_pending_gold = 0;
        self.native_shop_sell_roster_top = 0;
        self.native_shop_sell_roster_cycle = 0;
        self.native_shop_sell_item_top = 0;
        self.native_shop_sell_confirm_sel = 0;
        self.native_shop_sell_item_ids = Vec::new();
        self.native_shop_equip_roster_top = 0;
        self.native_shop_equip_unit_sel = 0;
        self.native_shop_transfer_source = -1;
        self.native_shop_transfer_item = -1;
        self.native_shop_transfer_items = Vec::new();
        self.native_shop_transfer_dest = -1;
        self.native_shop_transfer_ids = Vec::new();
        self.native_shop_transfer_sel = 0;
        self.native_shop_transfer_top = 0;

        self.shop_sel = 0;
        self.shop_recipient_sel = 0;
        self.shop_recipients = Vec::new();
        self.shop_picking = false;
        self.shop_pending = campaign::Good::default();
        self.shop_equip_prompt = false;
        self.shop_equip_unit = 0;
        self.shop_equip_slot = 0;
        self.shop_mode = String::new();
        self.shop_sell_picking = false;
        self.shop_sell_unit_sel = 0;
        self.shop_sell_slot_sel = 0;
        self.clear_native_item_panel();
    }

    /// Go：`save.go:392`，`func (g *Game) clearChurchTransientStateForLoad()`
    pub(crate) fn clear_church_transient_state_for_load(&mut self) {
        self.native_class_ui_job = None;
        self.native_church_ui_job = None;
        self.reset_native_class_ui_pulse();
        self.reset_native_church_ui_pulse();
        self.native_church_text_index = 0;
        self.church_sel = 0;
        self.church_mode = String::new();
        self.church_ids = Vec::new();
        self.church_roster_start = 0;
        self.church_vertical_start = 0;
        self.church_status_id = -1;
        self.church_status_panel = Vec::new();
        self.church_command_panel = Vec::new();
        self.church_item_start = 0;
        self.church_transfer_source = -1;
        self.church_transfer_item = -1;
        self.church_transfer_items = Vec::new();
        self.church_transfer_dest = -1;
        self.church_revive_id = -1;
        self.church_revive_fee = 0;
        self.church_class_id = -1;
        self.church_branches = Vec::new();
    }

    /// Go：`save.go:417`，`func (g *Game) restoreNativeMapHUDGateA(gateA byte)`
    pub(crate) fn restore_native_map_hud_gate_a(&mut self, gate_a: u8) {
        if !self.native_map_hud_persistent.has_anchor_x {
            self.native_map_hud_persistent = battle::initial_native_map_hud_persistent_state();
        }
        self.native_map_hud_persistent.restore_saved_gate_a(gate_a);
    }
}

#[cfg(test)]
pub(crate) mod f_test_env;
#[cfg(test)]
pub(crate) use f_test_env::f_child_env;

#[cfg(test)]
mod save_test;

#[cfg(test)]
mod golden_save_test;
