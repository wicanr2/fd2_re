//! Go 來源：`remake/internal/ending/separated_fdfield.go`（125 行）。

use super::*;

pub(crate) const SEPARATED_FDFIELD_SIZE: i64 = 243169;

pub(crate) const SEPARATED_FDFIELDMD5: &str = "ecdb0436d26adfe5d107f2713fa7e9a2";

pub(crate) const SEPARATED_FDFIELDSHA256: &str = "b0cf75d94f58603f091c7462c0494f0e83bd6edfb04c1acbf83ed4d938c7a513";

pub(crate) const SELECTOR30_MAP_SHA256: &str = "fd44ebb3f269a45e16f84e3ea1ec3458aa672d7c85caa5ffc4dd9c628db168d3";

pub(crate) const SELECTOR30_CONTROL_SHA256: &str = "c17581097e0beb63ff892ea6acf0b50a9ab5e62aa94ceab560c46a6f2ede625a";

pub(crate) const SELECTOR30_POSITIONS_SHA256: &str = "371a43900435ff22887a94dc6c96341585e73d933beaf23fd2b6e77b6d3dc4f9";

/// Go：`separated_fdfield.go:22`
#[derive(Debug, Clone, Default, PartialEq, Eq)]
pub(crate) struct SeparatedFDFIELDSource {
    /// Go 標籤：`json:"file"`
    pub file: String,
    /// Go 標籤：`json:"resource"`
    pub resource: i64,
    /// Go 標籤：`json:"size"`
    pub size: i64,
    /// Go 標籤：`json:"md5"`
    pub md5: String,
    /// Go 標籤：`json:"sha256"`
    pub sha256: String,
    /// Go 標籤：`json:"raw_size"`
    pub raw_size: i64,
}

/// Go：`separated_fdfield.go:31`
#[derive(Debug, Clone, Copy, Default, PartialEq, Eq)]
pub(crate) struct SeparatedFDFIELDPosition {
    /// Go 標籤：`json:"x_word"`
    pub x_word: i64,
    /// Go 標籤：`json:"y_word"`
    pub y_word: i64,
    /// Go 標籤：`json:"raw_key"`
    pub raw_key: i64,
}

/// Go：`separated_fdfield.go:37`
#[derive(Debug, Clone, Default, PartialEq, Eq)]
pub(crate) struct SeparatedFDFIELDDocument {
    /// Go 標籤：`json:"schema_version"`
    pub schema_version: i64,
    /// Go 標籤：`json:"kind"`
    pub kind: String,
    /// Go 標籤：`json:"document_id"`
    pub document_id: String,
    /// Go 標籤：`json:"status"`
    pub status: String,
    /// Go 標籤：`json:"evidence"`
    pub evidence: String,
    /// Go 標籤：`json:"source"`
    pub source: SeparatedFDFIELDSource,
    /// Go 標籤：`json:"map_resource"`
    pub map_resource: i64,
    /// Go 標籤：`json:"control_resource"`
    pub control_resource: i64,
    /// Go 標籤：`json:"positions_resource"`
    pub positions_resource: i64,
    /// Go 標籤：`json:"map_sha256"`
    pub map_sha256: String,
    /// Go 標籤：`json:"control_sha256"`
    pub control_sha256: String,
    /// Go 標籤：`json:"positions_sha256"`
    pub positions_sha256: String,
    /// Go 標籤：`json:"width"`
    pub width: i64,
    /// Go 標籤：`json:"height"`
    pub height: i64,
    /// Go 標籤：`json:"tiles"`
    pub tiles: Vec<i64>,
    /// Go 標籤：`json:"event_bytes"`
    pub event_bytes: Vec<i64>,
    /// Go 標籤：`json:"blit_modes"`
    pub blit_modes: Vec<i64>,
    /// Go 標籤：`json:"control_bytes"`
    pub control_bytes: Vec<i64>,
    /// Go 標籤：`json:"positions"`
    pub positions: Vec<SeparatedFDFIELDPosition>,
}

/// Go：`separated_fdfield.go:59`，`func hashBytes(raw []byte) string`
pub(crate) fn hash_bytes(raw: &[u8]) -> String {
    todo!("ending/separated_fdfield.go:hashBytes")
}

/// Go：`separated_fdfield.go:64`，`func loadSeparatedFDFIELDSelector30(root string) ([]byte, []byte, []byte, error)`
pub(crate) fn load_separated_fdfield_selector30(root: &str) -> Result<(Vec<u8>, Vec<u8>, Vec<u8>)> {
    todo!("ending/separated_fdfield.go:loadSeparatedFDFIELDSelector30")
}
