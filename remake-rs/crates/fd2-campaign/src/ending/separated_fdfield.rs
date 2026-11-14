//! Go 來源：`remake/internal/ending/separated_fdfield.go`（125 行）。

use super::*;

pub(crate) const SEPARATED_FDFIELD_SIZE: i64 = 243169;

pub(crate) const SEPARATED_FDFIELDMD5: &str = "ecdb0436d26adfe5d107f2713fa7e9a2";

pub(crate) const SEPARATED_FDFIELDSHA256: &str = "b0cf75d94f58603f091c7462c0494f0e83bd6edfb04c1acbf83ed4d938c7a513";

pub(crate) const SELECTOR30_MAP_SHA256: &str = "fd44ebb3f269a45e16f84e3ea1ec3458aa672d7c85caa5ffc4dd9c628db168d3";

pub(crate) const SELECTOR30_CONTROL_SHA256: &str = "c17581097e0beb63ff892ea6acf0b50a9ab5e62aa94ceab560c46a6f2ede625a";

pub(crate) const SELECTOR30_POSITIONS_SHA256: &str = "371a43900435ff22887a94dc6c96341585e73d933beaf23fd2b6e77b6d3dc4f9";

/// Go：`separated_fdfield.go:22`
#[derive(Debug, Clone, Default, PartialEq, Eq, serde::Serialize, serde::Deserialize)]
#[serde(default)]
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
#[derive(Debug, Clone, Copy, Default, PartialEq, Eq, serde::Serialize, serde::Deserialize)]
#[serde(default)]
pub(crate) struct SeparatedFDFIELDPosition {
    /// Go 標籤：`json:"x_word"`
    pub x_word: i64,
    /// Go 標籤：`json:"y_word"`
    pub y_word: i64,
    /// Go 標籤：`json:"raw_key"`
    pub raw_key: i64,
}

/// Go：`separated_fdfield.go:37`
#[derive(Debug, Clone, Default, PartialEq, Eq, serde::Serialize, serde::Deserialize)]
#[serde(default)]
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
    #[serde(deserialize_with = "battle::null_as_default")]
    pub tiles: Vec<i64>,
    /// Go 標籤：`json:"event_bytes"`
    #[serde(deserialize_with = "battle::null_as_default")]
    pub event_bytes: Vec<i64>,
    /// Go 標籤：`json:"blit_modes"`
    #[serde(deserialize_with = "battle::null_as_default")]
    pub blit_modes: Vec<i64>,
    /// Go 標籤：`json:"control_bytes"`
    #[serde(deserialize_with = "battle::null_as_default")]
    pub control_bytes: Vec<i64>,
    /// Go 標籤：`json:"positions"`
    #[serde(deserialize_with = "battle::null_as_default")]
    pub positions: Vec<SeparatedFDFIELDPosition>,
}

/// Go：`separated_fdfield.go:59`，`func hashBytes(raw []byte) string`
pub(crate) fn hash_bytes(raw: &[u8]) -> String {
    use sha2::Digest as _;
    sha2::Sha256::digest(raw).iter().map(|b| format!("{b:02x}")).collect()
}

/// Go：`separated_fdfield.go:64`，`func loadSeparatedFDFIELDSelector30(root string) ([]byte, []byte, []byte, error)`
pub(crate) fn load_separated_fdfield_selector30(root: &str) -> Result<(Vec<u8>, Vec<u8>, Vec<u8>)> {
    if root.is_empty() {
        return Err(campaign_err!("ending: separated FDFIELD root is unavailable"));
    }
    let path = std::path::Path::new(root).join("selector_30").join("field.json");
    let raw = battle::read_file(&path.to_string_lossy())?;
    let doc: SeparatedFDFIELDDocument = battle::decode_json(&raw)?;
    if doc.schema_version != 1
        || doc.kind != "fdfield_selector"
        || doc.document_id != "field/fdfield/selector_30"
        || doc.status != "decoded"
        || doc.evidence != "confirmed"
    {
        return Err(campaign_err!("ending: separated FDFIELD selector 30 identity is invalid"));
    }
    if doc.source.file != "FDFIELD.DAT"
        || doc.source.resource != 90
        || doc.source.size != SEPARATED_FDFIELD_SIZE
        || doc.source.md5 != SEPARATED_FDFIELDMD5
        || doc.source.sha256 != SEPARATED_FDFIELDSHA256
        || doc.source.raw_size != 6304
        || doc.map_resource != 90
        || doc.control_resource != 91
        || doc.positions_resource != 92
        || doc.map_sha256 != SELECTOR30_MAP_SHA256
        || doc.control_sha256 != SELECTOR30_CONTROL_SHA256
        || doc.positions_sha256 != SELECTOR30_POSITIONS_SHA256
    {
        return Err(campaign_err!("ending: separated FDFIELD selector 30 provenance is invalid"));
    }
    if doc.width != 35 || doc.height != 45 {
        return Err(campaign_err!("ending: separated FDFIELD selector 30 geometry={}x{}", doc.width, doc.height));
    }
    let cells = (doc.width * doc.height) as usize;
    if doc.tiles.len() != cells || doc.event_bytes.len() != cells || doc.blit_modes.len() != cells {
        return Err(campaign_err!("ending: separated FDFIELD selector 30 cell arrays are incomplete"));
    }
    let mut field_map = vec![0u8; 4 + cells * 4];
    field_map[0..2].copy_from_slice(&(doc.width as u16).to_le_bytes());
    field_map[2..4].copy_from_slice(&(doc.height as u16).to_le_bytes());
    for index in 0..cells {
        let (tile, event, blit) = (doc.tiles[index], doc.event_bytes[index], doc.blit_modes[index]);
        if !(0..=0xffff).contains(&tile) || !(0..=0xff).contains(&event) || !(0..=0xff).contains(&blit) {
            return Err(campaign_err!("ending: separated FDFIELD selector 30 cell {index} is outside raw range"));
        }
        let offset = 4 + index * 4;
        field_map[offset..offset + 2].copy_from_slice(&(tile as u16).to_le_bytes());
        field_map[offset + 2] = event as u8;
        field_map[offset + 3] = blit as u8;
    }
    let mut control = vec![0u8; doc.control_bytes.len()];
    for (index, &value) in doc.control_bytes.iter().enumerate() {
        if !(0..=0xff).contains(&value) {
            return Err(campaign_err!(
                "ending: separated FDFIELD selector 30 control byte {index} is outside raw range"
            ));
        }
        control[index] = value as u8;
    }
    let mut positions = vec![0u8; 2 + doc.positions.len() * 6];
    positions[0..2].copy_from_slice(&(doc.positions.len() as u16).to_le_bytes());
    for (index, row) in doc.positions.iter().enumerate() {
        if !(0..=0xffff).contains(&row.x_word)
            || !(0..=0xffff).contains(&row.y_word)
            || !(0..=0xffff).contains(&row.raw_key)
        {
            return Err(campaign_err!("ending: separated FDFIELD selector 30 position {index} is outside raw range"));
        }
        let offset = 2 + index * 6;
        positions[offset..offset + 2].copy_from_slice(&(row.x_word as u16).to_le_bytes());
        positions[offset + 2..offset + 4].copy_from_slice(&(row.y_word as u16).to_le_bytes());
        positions[offset + 4..offset + 6].copy_from_slice(&(row.raw_key as u16).to_le_bytes());
    }
    if hash_bytes(&field_map) != doc.map_sha256
        || hash_bytes(&control) != doc.control_sha256
        || hash_bytes(&positions) != doc.positions_sha256
    {
        return Err(campaign_err!("ending: separated FDFIELD selector 30 content hash mismatch"));
    }
    Ok((field_map, control, positions))
}

#[cfg(test)]
mod separated_fdfield_test;
