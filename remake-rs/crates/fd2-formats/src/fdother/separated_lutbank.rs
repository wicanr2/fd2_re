//! 移植自 Go `separated_lutbank.go`：FDOTHER #3 完整 23×256 重映射表的分離素材。

use super::go_compat::{nullable, nullable_struct, nullable_struct_vec, nullable_vec};
use super::separated_surface::read_document;
use super::{FdotherError, Result};
use serde::Deserialize;
use std::path::Path;

#[derive(Debug, Default, Deserialize)]
#[serde(default)]
struct SeparatedLUTBankSource {
    #[serde(deserialize_with = "nullable")]
    file: String,
    #[serde(deserialize_with = "nullable")]
    resource: i64,
    #[serde(deserialize_with = "nullable")]
    size: i64,
    #[serde(deserialize_with = "nullable")]
    md5: String,
    #[serde(deserialize_with = "nullable")]
    sha256: String,
    #[serde(deserialize_with = "nullable")]
    raw_size: i64,
    #[serde(deserialize_with = "nullable")]
    raw_md5: String,
    #[serde(deserialize_with = "nullable")]
    raw_sha256: String,
}

#[derive(Debug, Default, Deserialize)]
#[serde(default)]
struct SeparatedLUTEntry {
    #[serde(deserialize_with = "nullable")]
    index: i64,
    #[serde(deserialize_with = "nullable_vec")]
    components: Vec<i64>,
}

#[derive(Debug, Default, Deserialize)]
#[serde(default)]
struct SeparatedLUTBankDocument {
    #[serde(deserialize_with = "nullable")]
    schema_version: i64,
    #[serde(deserialize_with = "nullable")]
    kind: String,
    #[serde(deserialize_with = "nullable")]
    asset_id: String,
    #[serde(deserialize_with = "nullable")]
    status: String,
    #[serde(deserialize_with = "nullable")]
    evidence: String,
    #[serde(deserialize_with = "nullable_struct")]
    source: SeparatedLUTBankSource,
    #[serde(deserialize_with = "nullable_struct_vec")]
    luts: Vec<SeparatedLUTEntry>,
}

/// 載入 FDOTHER #3 完整的 23×256 重映射表。保留原始索引與值，絕不開啟原始封存。
pub fn load_separated_lut_bank(root: impl AsRef<Path>) -> Result<Vec<Vec<u8>>> {
    let document: SeparatedLUTBankDocument = read_document(
        &root.as_ref().join("fdother_003_luts.json"),
        "fdother: separated LUT bank metadata: ",
        "fdother: separated LUT bank JSON: ",
    )?;
    if document.schema_version != 1
        || document.kind != "fdother_lut_bank"
        || document.asset_id != "lut/FDOTHER_003"
        || document.status != "decoded"
        || document.evidence != "confirmed"
        || document.source.file != "FDOTHER.DAT"
        || document.source.resource != 3
        || document.source.size != 3382481
        || document.source.md5 != "22f56e5027edc7c766ad34ca4e5aca93"
        || document.source.sha256 != "a81b13493725fb70e750c4d9e0dce4e1b57d0df312c4ad4157e6d45171b13bce"
        || document.source.raw_size != 5990
        || document.source.raw_md5 != "f37194b6d933187d86711c6701732b5f"
        || document.source.raw_sha256 != "fddfdbd91b048081ea621b5a4be7d29fb24234a62e19e2b21837df52cbb02ccc"
        || document.luts.len() != 23
    {
        return Err(FdotherError::fixed("fdother: separated LUT bank contract mismatch"));
    }
    let mut luts = vec![Vec::new(); 23];
    let mut seen = [false; 23];
    for entry in &document.luts {
        let index = entry.index;
        if !(0..23).contains(&index) || seen[index as usize] || entry.components.len() != 256 {
            return Err(FdotherError::message(format!("fdother: separated LUT {index} metadata mismatch")));
        }
        let mut lut = vec![0u8; 256];
        for (offset, &value) in entry.components.iter().enumerate() {
            if !(0..=255).contains(&value) {
                return Err(FdotherError::message(format!(
                    "fdother: separated LUT {index} component {offset} out of range"
                )));
            }
            lut[offset] = value as u8;
        }
        luts[index as usize] = lut;
        seen[index as usize] = true;
    }
    if let Some(index) = seen.iter().position(|&present| !present) {
        return Err(FdotherError::message(format!("fdother: separated LUT {index} is missing")));
    }
    Ok(luts)
}
