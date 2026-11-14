//! Go 來源：`remake/internal/editorcanonical/bundle.go`（135 行）。
//!
//! Package editorcanonical validates the versioned editor bundle admitted by
//! the packaged runtime. It does not interpret legacy extensions as gameplay
//! semantics; it only proves that the canonical documents shipped together are
//! the reviewed, deterministic set.

use super::*;

/// Go：`bundle.go:20`
#[derive(Debug, Clone, Copy, Default, PartialEq, Eq)]
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
#[derive(Debug, Clone, Default, PartialEq, Eq)]
pub struct Document {
    /// Go 標籤：`json:"kind"`
    pub kind: String,
    /// Go 標籤：`json:"source"`
    pub source: String,
    /// Go 標籤：`json:"output"`
    pub output: String,
    /// Go 標籤：`json:"document_id"`
    pub document_id: String,
    /// Go 標籤：`json:"sha256"`
    pub sha256: String,
    /// Go 標籤：`json:"diagnostics"`
    pub diagnostics: i64,
}

/// Go：`bundle.go:36`
#[derive(Debug, Clone, Default, PartialEq, Eq)]
pub struct Summary {
    /// Go 標籤：`json:"schema_version"`
    pub schema_version: i64,
    /// Go 標籤：`json:"exporter_version"`
    pub exporter_version: String,
    /// Go 標籤：`json:"importer_version"`
    pub importer_version: String,
    /// Go 標籤：`json:"source_root"`
    pub source_root: String,
    /// Go 標籤：`json:"include_animations"`
    pub include_animations: bool,
    /// Go 標籤：`json:"counts"`
    pub counts: Counts,
    /// Go 標籤：`json:"documents"`
    pub documents: Vec<Document>,
    /// Go 標籤：`json:"diagnostics"`
    pub diagnostics: Vec<GoRawMessage>,
    /// Go 標籤：`json:"identity_diagnostics"`
    pub identity_diagnostics: i64,
}

/// Go：`bundle.go:48`，`func decodeStrict(path string, target any) error`
/// Rust：Go 以 `any` 收任意解碼目標；Rust 改為泛型。Go 的 `DisallowUnknownFields` 在 Rust 由目標型別加 `#[serde(deny_unknown_fields)]` 表達（serde 沒有解碼器層級的開關），階段 2 移植時逐一加在 `Summary`、`Counts`、`Document` 與 `compile.go` 的 canonical 型別上；尾端多餘 JSON 以 `serde_json::Deserializer::end` 檢查。
pub(crate) fn decode_strict<T: serde::de::DeserializeOwned>(path: &str, target: &mut T) -> Result<()> {
    todo!("editorcanonical/bundle.go:decodeStrict")
}

/// ValidateBundle verifies the public core bundle. Private animation metadata
/// is optional input to the exporter and is deliberately not required by the
/// distributable runtime contract.
///
/// Go：`bundle.go:70`，`func ValidateBundle(root string) (*Summary, error)`
pub fn validate_bundle(root: &str) -> Result<Summary> {
    todo!("editorcanonical/bundle.go:ValidateBundle")
}
