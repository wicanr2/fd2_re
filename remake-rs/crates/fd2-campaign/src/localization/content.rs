//! 移植自 Go `localization/content.go`。

use super::patterns::{content_placeholder_matches, locale_pattern, sha256_pattern};
use super::{Result, SCHEMA_VERSION, decode_strict, err, go_quote, strict_decode_error};
use crate::go_filepath;
use fd2_rules::battle::null_as_default;
use serde::{Deserialize, Serialize};
use std::collections::{BTreeMap, BTreeSet};

/// ContentEntry is one player-visible occurrence in the full locale catalog.
/// Source remains provenance only; StringID is the runtime identity.
///
/// Go：`content.go:19`
#[derive(Debug, Clone, Default, PartialEq, Serialize, Deserialize)]
#[serde(default, deny_unknown_fields)]
pub struct ContentEntry {
    #[serde(deserialize_with = "null_as_default")]
    pub string_id: String,
    #[serde(deserialize_with = "null_as_default")]
    pub id_status: String,
    #[serde(deserialize_with = "null_as_default")]
    pub role: String,
    #[serde(deserialize_with = "null_as_default")]
    pub text: String,
    #[serde(deserialize_with = "null_as_default")]
    pub variables: Vec<String>,
    #[serde(deserialize_with = "null_as_default")]
    pub status: String,
    /// Go `map[string]any`：內容不驗證，只保留出處。
    #[serde(deserialize_with = "null_as_default")]
    pub source: BTreeMap<String, serde_json::Value>,
}

/// Go：`content.go:29`
#[derive(Debug, Clone, Default, Deserialize)]
#[serde(default, deny_unknown_fields)]
struct ContentPack {
    #[serde(deserialize_with = "null_as_default")]
    schema_version: i64,
    #[serde(deserialize_with = "null_as_default")]
    kind: String,
    #[serde(deserialize_with = "null_as_default")]
    locale: String,
    #[serde(deserialize_with = "null_as_default")]
    source_locale: String,
    #[serde(deserialize_with = "null_as_default")]
    entry_count: i64,
    #[serde(deserialize_with = "null_as_default")]
    inventory_sha256: String,
    #[serde(deserialize_with = "null_as_default")]
    glossary_sha256: String,
    /// Go `*string`：`null` 與缺欄位都是 nil。
    pivot_content_sha256: Option<String>,
    #[serde(deserialize_with = "null_as_default")]
    translation_engine: String,
    #[serde(deserialize_with = "null_as_default")]
    entries: Vec<ContentEntry>,
}

/// ContentCatalog is an immutable lookup view of one complete official locale.
///
/// Go：`content.go:43`
#[derive(Debug, Clone, Default, PartialEq)]
pub struct ContentCatalog {
    pub locale: String,
    pub inventory_sha256: String,
    pub glossary_sha256: String,
    pub translation_engine: String,
    pub(crate) entries: BTreeMap<String, ContentEntry>,
}

/// LoadOfficialContent validates <root>/<locale>/content.json. It intentionally
/// stays separate from pack.json: semantic UI messages and full authored content
/// have different schemas and rollout gates.
///
/// Go：`content.go:54`
pub fn load_official_content(root: &str, locale: &str) -> Result<ContentCatalog> {
    if !locale_pattern(locale) {
        return Err(err(format!("invalid locale {}", go_quote(locale))));
    }
    let path = go_filepath::join(&[root, locale, "content.json"]);
    let raw = std::fs::read(&path).map_err(|e| err(format!("read locale content {}: {e}", go_quote(&path))))?;
    let pack: ContentPack = decode_strict(&raw)
        .map_err(|e| strict_decode_error(&format!("decode locale content {}", go_quote(&path)), e))?;
    validate_content_pack(&pack, locale)
        .map_err(|e| e.wrap(format!("validate locale content {}: ", go_quote(&path))))?;
    let mut entries = BTreeMap::new();
    for entry in pack.entries {
        entries.insert(entry.string_id.clone(), entry);
    }
    Ok(ContentCatalog {
        locale: locale.to_string(),
        inventory_sha256: pack.inventory_sha256,
        glossary_sha256: pack.glossary_sha256,
        translation_engine: pack.translation_engine,
        entries,
    })
}

/// Go：`content.go:89`
fn validate_content_pack(pack: &ContentPack, locale: &str) -> Result<()> {
    if pack.schema_version != SCHEMA_VERSION || pack.kind != "fd2_full_locale_content" {
        return Err(err("schema_version or kind is invalid"));
    }
    if pack.locale != locale || pack.source_locale != "zh-Hant" {
        return Err(err(format!(
            "identity mismatch: locale={} source_locale={}",
            go_quote(&pack.locale),
            go_quote(&pack.source_locale)
        )));
    }
    if pack.entry_count != pack.entries.len() as i64 || pack.entry_count == 0 {
        return Err(err("entry_count does not match entries"));
    }
    if !sha256_pattern(&pack.inventory_sha256) || !sha256_pattern(&pack.glossary_sha256) {
        return Err(err("inventory or glossary SHA-256 is invalid"));
    }
    if pack.pivot_content_sha256.as_deref().is_some_and(|s| !sha256_pattern(s)) {
        return Err(err("pivot content SHA-256 is invalid"));
    }
    if pack.translation_engine.is_empty() {
        return Err(err("translation_engine is empty"));
    }
    let mut seen = BTreeSet::new();
    for entry in &pack.entries {
        if entry.string_id.is_empty() || entry.role.is_empty() || entry.text.is_empty() || entry.id_status.is_empty() {
            return Err(err("content entry has an empty required field"));
        }
        if !seen.insert(entry.string_id.as_str()) {
            return Err(err(format!("duplicate string_id {}", go_quote(&entry.string_id))));
        }
        if !matches!(entry.status.as_str(), "source" | "machine_draft" | "reviewed" | "blocked") {
            return Err(err(format!(
                "entry {} has invalid status {}",
                go_quote(&entry.string_id),
                go_quote(&entry.status)
            )));
        }
        if content_placeholders(&entry.text) != entry.variables {
            return Err(err(format!("entry {} variables do not match text placeholders", go_quote(&entry.string_id))));
        }
    }
    Ok(())
}

/// Go：`content.go:125`
pub(crate) fn content_placeholders(text: &str) -> Vec<String> {
    content_placeholder_matches(text).into_iter().filter(|m| *m != "%%").map(str::to_string).collect()
}

impl ContentCatalog {
    /// StoryText resolves one canonical authored dialogue line. Provisional source
    /// identities and non-dialogue roles cannot enter the formal story path.
    ///
    /// Go：`content.go:138`。Go 對 nil 接收者回 `nil locale content catalog`；Rust 的 `self` 不會是 null，
    /// 那條分支不移植。
    pub fn story_text(&self, line_id: &str) -> Result<String> {
        let key = format!("{line_id}/text");
        let Some(entry) = self.entries.get(&key) else {
            return Err(err(format!("missing story content {}", go_quote(&key))));
        };
        if entry.id_status != "stable_canonical" || entry.role != "dialogue" {
            return Err(err(format!(
                "story content {} has identity={} role={}",
                go_quote(&key),
                go_quote(&entry.id_status),
                go_quote(&entry.role)
            )));
        }
        if entry.status == "blocked" {
            return Err(err(format!("story content {} is blocked", go_quote(&key))));
        }
        Ok(entry.text.clone())
    }
}

#[cfg(test)]
mod content_test;
