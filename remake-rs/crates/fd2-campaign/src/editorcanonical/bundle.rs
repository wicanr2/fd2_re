//! Go 來源：`remake/internal/editorcanonical/bundle.go`（135 行）。
//!
//! Package editorcanonical validates the versioned editor bundle admitted by
//! the packaged runtime. It does not interpret legacy extensions as gameplay
//! semantics; it only proves that the canonical documents shipped together are
//! the reviewed, deterministic set.

use super::*;

/// Go：`bundle.go:20`
#[derive(Debug, Clone, Copy, Default, PartialEq, Eq, serde::Serialize, serde::Deserialize)]
#[serde(default, deny_unknown_fields)]
pub struct Counts {
    /// Go 標籤：`json:"campaign"`
    pub campaign: i64,
    /// Go 標籤：`json:"scenario"`
    pub scenario: i64,
    /// Go 標籤：`json:"story"`
    pub story: i64,
    /// Go 標籤：`json:"animation"`
    pub animation: i64,
}

/// Go：`bundle.go:27`
#[derive(Debug, Clone, Default, PartialEq, Eq, serde::Serialize, serde::Deserialize)]
#[serde(default, deny_unknown_fields)]
pub struct Document {
    /// Go 標籤：`json:"kind"`
    #[serde(deserialize_with = "fd2_rules::battle::null_as_default")]
    pub kind: String,
    /// Go 標籤：`json:"source"`
    #[serde(deserialize_with = "fd2_rules::battle::null_as_default")]
    pub source: String,
    /// Go 標籤：`json:"output"`
    #[serde(deserialize_with = "fd2_rules::battle::null_as_default")]
    pub output: String,
    /// Go 標籤：`json:"document_id"`
    #[serde(deserialize_with = "fd2_rules::battle::null_as_default")]
    pub document_id: String,
    /// Go 標籤：`json:"sha256"`
    #[serde(deserialize_with = "fd2_rules::battle::null_as_default")]
    pub sha256: String,
    /// Go 標籤：`json:"diagnostics"`
    pub diagnostics: i64,
}

/// Go：`bundle.go:36`
#[derive(Debug, Clone, Default, PartialEq, Eq, serde::Serialize, serde::Deserialize)]
#[serde(default, deny_unknown_fields)]
pub struct Summary {
    /// Go 標籤：`json:"schema_version"`
    pub schema_version: i64,
    /// Go 標籤：`json:"exporter_version"`
    #[serde(deserialize_with = "fd2_rules::battle::null_as_default")]
    pub exporter_version: String,
    /// Go 標籤：`json:"importer_version"`
    #[serde(deserialize_with = "fd2_rules::battle::null_as_default")]
    pub importer_version: String,
    /// Go 標籤：`json:"source_root"`
    #[serde(deserialize_with = "fd2_rules::battle::null_as_default")]
    pub source_root: String,
    /// Go 標籤：`json:"include_animations"`
    pub include_animations: bool,
    /// Go 標籤：`json:"counts"`
    pub counts: Counts,
    /// Go 標籤：`json:"documents"`
    #[serde(deserialize_with = "fd2_rules::battle::null_as_default")]
    pub documents: Vec<Document>,
    /// Go 標籤：`json:"diagnostics"`
    #[serde(deserialize_with = "fd2_rules::battle::null_as_default")]
    pub diagnostics: Vec<GoRawMessage>,
    /// Go 標籤：`json:"identity_diagnostics"`
    pub identity_diagnostics: i64,
}

/// Go：`bundle.go:48`，`func decodeStrict(path string, target any) error`
/// Rust：Go 以 `any` 收任意解碼目標；Rust 改為泛型。Go 的 `DisallowUnknownFields` 在 Rust 由目標型別加 `#[serde(deny_unknown_fields)]` 表達（serde 沒有解碼器層級的開關），階段 2 移植時逐一加在 `Summary`、`Counts`、`Document` 與 `compile.go` 的 canonical 型別上；尾端多餘 JSON 以 `serde_json::Deserializer::end` 檢查。
pub(crate) fn decode_strict<T: serde::de::DeserializeOwned>(path: &str, target: &mut T) -> Result<()> {
    let raw = std::fs::read(path).map_err(|e| campaign_err!("open {path}: {e}"))?;
    let mut stream = serde_json::Deserializer::from_slice(&raw).into_iter::<T>();
    // Go 的 `decoder.Decode` 與標準函式庫錯誤文字不逐字比對（只比接受或拒收）。
    match stream.next() {
        Some(Ok(value)) => *target = value,
        Some(Err(err)) => return Err(CampaignError::new(err.to_string())),
        None => return Err(CampaignError::new("EOF")),
    }
    let rest = &raw[stream.byte_offset()..];
    // Go 第二次 `Decode`：成功讀到另一個值是 "trailing JSON"；只剩空白是 `io.EOF`（通過）；
    // 其餘是解碼錯誤。
    match serde_json::Deserializer::from_slice(rest).into_iter::<serde::de::IgnoredAny>().next() {
        None => Ok(()),
        Some(Ok(_)) => Err(CampaignError::new("trailing JSON")),
        Some(Err(err)) => Err(CampaignError::new(err.to_string())),
    }
}

/// ValidateBundle verifies the public core bundle. Private animation metadata
/// is optional input to the exporter and is deliberately not required by the
/// distributable runtime contract.
///
/// Go：`bundle.go:70`，`func ValidateBundle(root string) (*Summary, error)`
pub fn validate_bundle(root: &str) -> Result<Summary> {
    use sha2::Digest as _;
    if root.is_empty() {
        return Err(CampaignError::new("editor canonical root is unavailable"));
    }
    let root_path = std::path::Path::new(root);
    let mut summary = Summary::default();
    decode_strict(&root_path.join("bundle-summary.json").to_string_lossy(), &mut summary)
        .map_err(|err| err.wrap("editor canonical summary: "))?;
    if summary.schema_version != 1
        || summary.exporter_version != "fd2-editor-canonical-exporter/1.0"
        || summary.importer_version != "fd2-editor-legacy-importer/1.0"
        || summary.source_root != "repository-relative"
        || summary.include_animations
    {
        return Err(CampaignError::new("editor canonical summary identity is invalid"));
    }
    if summary.counts != (Counts { campaign: 1, scenario: 30, story: 35, animation: 0 })
        || summary.documents.len() != 66
    {
        return Err(campaign_err!(
            "editor canonical counts={} documents={}",
            go_plus_v_counts(&summary.counts),
            summary.documents.len()
        ));
    }
    let mut seen = BTreeSet::new();
    let mut actual = Counts::default();
    for document in &summary.documents {
        let q = fd2_rules::battle::go_quote(&document.document_id);
        if document.document_id.is_empty() || seen.contains(&document.document_id) {
            return Err(campaign_err!("editor canonical duplicate or empty document_id {q}"));
        }
        seen.insert(document.document_id.clone());
        match document.kind.as_str() {
            "campaign" => actual.campaign += 1,
            "scenario" => actual.scenario += 1,
            "story" => actual.story += 1,
            _ => {
                return Err(campaign_err!(
                    "editor canonical document {q} has invalid kind {}",
                    fd2_rules::battle::go_quote(&document.kind)
                ));
            }
        }
        let Some(clean) = bundle_relative_path(&document.output) else {
            return Err(campaign_err!("editor canonical document {q} escapes bundle"));
        };
        let digest = fd2_rules::battle::decode_hex(&document.sha256);
        let Some(digest) = digest.filter(|d| d.len() == 32) else {
            return Err(campaign_err!("editor canonical document {q} has invalid SHA-256"));
        };
        let raw = std::fs::read(root_path.join(&clean)).map_err(|e| {
            campaign_err!("editor canonical document {q}: open {}: {e}", root_path.join(&clean).display())
        })?;
        if sha2::Sha256::digest(&raw).as_slice() != digest.as_slice() {
            return Err(campaign_err!("editor canonical document {q} hash mismatch"));
        }
    }
    if actual != summary.counts {
        return Err(campaign_err!(
            "editor canonical typed counts={} want {}",
            go_plus_v_counts(&actual),
            go_plus_v_counts(&summary.counts)
        ));
    }
    /// Go 函式內的匿名結構 `identity`（`bundle.go:118`）。
    #[derive(serde::Deserialize, Default)]
    #[serde(default, deny_unknown_fields)]
    struct Identity {
        schema_version: i64,
        #[serde(deserialize_with = "fd2_rules::battle::null_as_default")]
        kind: String,
        #[serde(deserialize_with = "fd2_rules::battle::null_as_default")]
        source: BTreeMap<String, String>,
        #[serde(deserialize_with = "fd2_rules::battle::null_as_default")]
        characters: Vec<GoRawMessage>,
        #[serde(deserialize_with = "fd2_rules::battle::null_as_default")]
        diagnostics: Vec<GoRawMessage>,
    }
    let mut identity = Identity::default();
    decode_strict(&root_path.join("character-identity.json").to_string_lossy(), &mut identity)
        .map_err(|err| err.wrap("editor canonical character identity: "))?;
    let source = |key: &str| identity.source.get(key).map(String::as_str).unwrap_or("");
    if identity.schema_version != 1
        || identity.kind != "character_identity_catalog"
        || source("evidence") != "direct legacy fields"
        || source("root") != "repository-relative"
        || identity.characters.is_empty()
        || identity.diagnostics.len() as i64 != summary.identity_diagnostics
    {
        return Err(CampaignError::new("editor canonical character identity contract is invalid"));
    }
    Ok(summary)
}

/// Go `fmt` 的 `%+v` 印 `Counts`（`bundle.go:83/113`）：`{Campaign:1 Scenario:30 Story:35 Animation:0}`。
pub(crate) fn go_plus_v_counts(c: &Counts) -> String {
    format!("{{Campaign:{} Scenario:{} Story:{} Animation:{}}}", c.campaign, c.scenario, c.story, c.animation)
}

/// Go `bundle.go:100` 的逃逸檢查：`filepath.Clean(filepath.FromSlash(output))` 不可是絕對路徑、`..`
/// 或以 `../` 開頭。Rust 以 `std::path` 的路徑元件逐段計算深度，等同字面清理後的判定（不移植
/// `filepath.Clean` 本身，`rust-port-no-go-stdlib`）；通過時回傳清理後的相對路徑。
pub(crate) fn bundle_relative_path(output: &str) -> Option<std::path::PathBuf> {
    use std::path::Component;
    if output.is_empty() {
        return None;
    }
    let mut parts: Vec<&std::ffi::OsStr> = Vec::new();
    for component in std::path::Path::new(output).components() {
        match component {
            Component::Normal(part) => parts.push(part),
            Component::CurDir => {}
            Component::ParentDir => {
                parts.pop()?;
            }
            Component::RootDir | Component::Prefix(_) => return None,
        }
    }
    Some(parts.iter().collect())
}

// 測試：Go `remake/internal/editorcanonical/bundle_test.go` 逐條移植。
#[cfg(test)]
mod bundle_test;
