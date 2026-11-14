//! 移植自 Go `localization/entities.go`。

use super::patterns::locale_pattern;
use super::{Result, SCHEMA_VERSION, decode_strict, err, go_quote, strict_decode_error};
use crate::go_filepath;
use fd2_rules::battle::null_as_default;
use serde::Deserialize;
use std::collections::BTreeMap;

/// Go：`entities.go:14`
#[derive(Debug, Clone, Default, Deserialize)]
#[serde(default, deny_unknown_fields)]
struct EntityNameEntry {
    #[serde(deserialize_with = "null_as_default")]
    name: String,
    #[serde(deserialize_with = "null_as_default")]
    source_string_ids: Vec<String>,
    /// Go 標籤：`status,omitempty`
    #[serde(deserialize_with = "null_as_default")]
    status: String,
}

/// Go：`entities.go:20`
#[derive(Debug, Clone, Default, Deserialize)]
#[serde(default, deny_unknown_fields)]
struct EntityPack {
    #[serde(deserialize_with = "null_as_default")]
    schema_version: i64,
    #[serde(deserialize_with = "null_as_default")]
    kind: String,
    #[serde(deserialize_with = "null_as_default")]
    locale: String,
    #[serde(deserialize_with = "null_as_default")]
    source_locale: String,
    #[serde(deserialize_with = "null_as_default")]
    item_count: i64,
    #[serde(deserialize_with = "null_as_default")]
    items: BTreeMap<String, EntityNameEntry>,
    #[serde(deserialize_with = "null_as_default")]
    character_count: i64,
    #[serde(deserialize_with = "null_as_default")]
    characters: BTreeMap<String, EntityNameEntry>,
    #[serde(deserialize_with = "null_as_default")]
    battle_name_count: i64,
    #[serde(deserialize_with = "null_as_default")]
    battle_names: BTreeMap<String, EntityNameEntry>,
    #[serde(deserialize_with = "null_as_default")]
    command_name_count: i64,
    #[serde(deserialize_with = "null_as_default")]
    command_names: BTreeMap<String, EntityNameEntry>,
    #[serde(deserialize_with = "null_as_default")]
    class_name_count: i64,
    #[serde(deserialize_with = "null_as_default")]
    class_names: BTreeMap<String, EntityNameEntry>,
}

/// EntityCatalog 是以遊戲實體 ID 為鍵的不可變官方名稱目錄。
///
/// Go：`entities.go:38`
#[derive(Debug, Clone, Default, PartialEq, Eq)]
pub struct EntityCatalog {
    pub locale: String,
    pub(crate) items: BTreeMap<i64, String>,
    pub(crate) characters: BTreeMap<i64, String>,
    pub(crate) battle_names: BTreeMap<i64, String>,
    pub(crate) command_names: BTreeMap<i64, String>,
    pub(crate) class_names: BTreeMap<i64, String>,
}

/// Go `strconv.Atoi` 的成功值：十進位、可帶 `+`／`-` 號、可有前導零，超出 `int` 範圍即失敗；
/// Rust 的 `i64::from_str` 接受的字串集合相同。
fn atoi(s: &str) -> Option<i64> {
    s.parse().ok()
}

/// LoadOfficialEntities 驗證 <root>/<locale>/entities.json。
///
/// Go：`entities.go:48`。Go 以 map 的隨機順序走訪各表（見模組文件）；Rust 依 JSON 鍵字串排序走訪。
/// 兩個鍵字串轉成同一個 ID（例如 `"7"` 與 `"07"`）時，Go 留下的名稱也由走訪順序決定。
pub fn load_official_entities(root: &str, locale: &str) -> Result<EntityCatalog> {
    if !locale_pattern(locale) {
        return Err(err(format!("invalid locale {}", go_quote(locale))));
    }
    let path = go_filepath::join(&[root, locale, "entities.json"]);
    let qpath = go_quote(&path);
    let raw = std::fs::read(&path).map_err(|e| err(format!("read locale entities {qpath}: {e}")))?;
    let pack: EntityPack =
        decode_strict(&raw).map_err(|e| strict_decode_error(&format!("decode locale entities {qpath}"), e))?;
    let n = |m: &BTreeMap<String, EntityNameEntry>| m.len() as i64;
    if pack.schema_version != SCHEMA_VERSION
        || pack.kind != "fd2_locale_entities"
        || pack.locale != locale
        || pack.source_locale != "zh-Hant"
        || pack.item_count != n(&pack.items)
        || pack.item_count == 0
        || pack.character_count != n(&pack.characters)
        || pack.character_count != 32
    {
        return Err(err(format!("validate locale entities {qpath}: identity or count mismatch")));
    }
    if pack.battle_name_count != n(&pack.battle_names) || pack.battle_name_count != 94 {
        return Err(err(format!("validate locale entities {qpath}: battle name count mismatch")));
    }
    if pack.command_name_count != n(&pack.command_names) || pack.command_name_count != 35 {
        return Err(err(format!("validate locale entities {qpath}: command name count mismatch")));
    }
    if pack.class_name_count != n(&pack.class_names) || pack.class_name_count != 29 {
        return Err(err(format!("validate locale entities {qpath}: class name count mismatch")));
    }
    let mut items = BTreeMap::new();
    for (raw_id, entry) in &pack.items {
        let id = match atoi(raw_id) {
            Some(id) if (0..=255).contains(&id) && !entry.name.is_empty() && !entry.source_string_ids.is_empty() => id,
            _ => return Err(err(format!("validate locale entities {qpath}: invalid item {}", go_quote(raw_id)))),
        };
        if entry.source_string_ids.iter().any(String::is_empty) {
            return Err(err(format!("validate locale entities {qpath}: item {id} has empty provenance")));
        }
        if !matches!(
            entry.status.as_str(),
            "original_confirmed" | "deterministic_script_conversion" | "machine_draft" | "curated_remake_translation"
        ) {
            return Err(err(format!(
                "validate locale entities {qpath}: item {id} has invalid status {}",
                go_quote(&entry.status)
            )));
        }
        items.insert(id, entry.name.clone());
    }
    let mut characters = BTreeMap::new();
    for (raw_id, entry) in &pack.characters {
        let id = match atoi(raw_id) {
            Some(id) if (0..32).contains(&id) && !entry.name.is_empty() && entry.source_string_ids.len() == 2 => id,
            _ => return Err(err(format!("validate locale entities {qpath}: invalid character {}", go_quote(raw_id)))),
        };
        if !matches!(
            entry.status.as_str(),
            "original_confirmed" | "deterministic_script_conversion" | "curated_remake_transliteration"
        ) {
            return Err(err(format!(
                "validate locale entities {qpath}: invalid character status {}",
                go_quote(&entry.status)
            )));
        }
        characters.insert(id, entry.name.clone());
    }
    let mut battle_names = BTreeMap::new();
    for (raw_id, entry) in &pack.battle_names {
        let id = match atoi(raw_id) {
            Some(id) if (0..=138).contains(&id) && !entry.name.is_empty() && entry.source_string_ids.len() == 1 => id,
            _ => {
                return Err(err(format!("validate locale entities {qpath}: invalid battle name {}", go_quote(raw_id))));
            }
        };
        if !matches!(
            entry.status.as_str(),
            "original_confirmed"
                | "deterministic_script_conversion"
                | "curated_remake_transliteration"
                | "curated_remake_translation"
        ) {
            return Err(err(format!(
                "validate locale entities {qpath}: invalid battle name status {}",
                go_quote(&entry.status)
            )));
        }
        battle_names.insert(id, entry.name.clone());
    }
    let mut command_names = BTreeMap::new();
    for (raw_id, entry) in &pack.command_names {
        let id = match atoi(raw_id) {
            Some(id)
                if (0..=35).contains(&id)
                    && id != 31
                    && !entry.name.is_empty()
                    && entry.source_string_ids.len() == 1 =>
            {
                id
            }
            _ => {
                return Err(err(format!(
                    "validate locale entities {qpath}: invalid command name {}",
                    go_quote(raw_id)
                )));
            }
        };
        if !matches!(
            entry.status.as_str(),
            "original_confirmed"
                | "deterministic_script_conversion"
                | "curated_remake_translation"
                | "curated_source_glyph_correction"
        ) {
            return Err(err(format!(
                "validate locale entities {qpath}: invalid command name status {}",
                go_quote(&entry.status)
            )));
        }
        command_names.insert(id, entry.name.clone());
    }
    let mut class_names = BTreeMap::new();
    for (raw_id, entry) in &pack.class_names {
        let id = match atoi(raw_id) {
            Some(id) if (0..29).contains(&id) && !entry.name.is_empty() && entry.source_string_ids.len() == 1 => id,
            _ => return Err(err(format!("validate locale entities {qpath}: invalid class name {}", go_quote(raw_id)))),
        };
        let placeholder = id == 26 || id == 27 || id == 28;
        if placeholder && entry.status != "original_placeholder" {
            return Err(err(format!("validate locale entities {qpath}: class {id} lost placeholder status")));
        }
        if !placeholder
            && !matches!(
                entry.status.as_str(),
                "original_confirmed" | "deterministic_script_conversion" | "curated_remake_translation"
            )
        {
            return Err(err(format!(
                "validate locale entities {qpath}: invalid class status {}",
                go_quote(&entry.status)
            )));
        }
        class_names.insert(id, entry.name.clone());
    }
    Ok(EntityCatalog { locale: locale.to_string(), items, characters, battle_names, command_names, class_names })
}

fn lookup(map: &BTreeMap<i64, String>, id: i64, what: &str) -> Result<String> {
    map.get(&id).cloned().ok_or_else(|| err(format!("missing localized {what} {id}")))
}

// Go 對 nil 接收者回 `nil locale entity catalog`；Rust 的 `self` 不會是 null，那條分支不移植。
impl EntityCatalog {
    /// Go：`entities.go:159`
    pub fn character_name(&self, native_identity: i64) -> Result<String> {
        lookup(&self.characters, native_identity, "character")
    }

    /// Go：`entities.go:170`
    pub fn item_name(&self, id: i64) -> Result<String> {
        lookup(&self.items, id, "item")
    }

    /// BattleName 以原版戰鬥 record +8 的 raw selector 查詢面板名稱。
    /// 它與隊伍 native_identity 是兩個不同契約，不能互相替代。
    ///
    /// Go：`entities.go:183`
    pub fn battle_name(&self, raw_id: i64) -> Result<String> {
        lookup(&self.battle_names, raw_id, "battle name")
    }

    /// CommandName 以原版指令 ID 查詢玩家指令格名稱；ID31 是已證實空槽。
    ///
    /// Go：`entities.go:195`
    pub fn command_name(&self, raw_id: i64) -> Result<String> {
        lookup(&self.command_names, raw_id, "command name")
    }

    /// ClassName 以原版ClassID查詢職業名稱；26／27／28保留原版占位語意。
    ///
    /// Go：`entities.go:207`
    pub fn class_name(&self, class_id: i64) -> Result<String> {
        lookup(&self.class_names, class_id, "class")
    }
}

#[cfg(test)]
mod entities_test;
