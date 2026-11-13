//! 移植自 Go `separated_frame_bank.go`：FDOTHER 圖格表動畫（#45／#54／#34／#79）的分離素材。

use super::go_compat::{nullable, nullable_struct, nullable_struct_vec};
use super::separated_surface::{read_binary_mask, read_document, read_paletted_surface};
use super::{FdotherError, Frame, Result};
use serde::Deserialize;
use std::path::Path;

#[derive(Debug, Default, Deserialize)]
#[serde(default)]
struct SeparatedFrameBankSource {
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

/// Go `separatedFrameBankEntry`；幾何判斷函式直接看這些欄位。
#[derive(Debug, Default, Deserialize)]
#[serde(default)]
struct SeparatedFrameBankEntry {
    #[serde(deserialize_with = "nullable")]
    index: i64,
    #[serde(deserialize_with = "nullable")]
    x: i64,
    #[serde(deserialize_with = "nullable")]
    y: i64,
    #[serde(deserialize_with = "nullable")]
    width: i64,
    #[serde(deserialize_with = "nullable")]
    height: i64,
    #[serde(deserialize_with = "nullable")]
    frame: String,
    #[serde(deserialize_with = "nullable")]
    mask: String,
}

#[derive(Debug, Default, Deserialize)]
#[serde(default)]
struct SeparatedFrameBankDocument {
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
    #[serde(deserialize_with = "nullable")]
    codec: String,
    #[serde(deserialize_with = "nullable_struct")]
    source: SeparatedFrameBankSource,
    #[serde(deserialize_with = "nullable_struct_vec")]
    frames: Vec<SeparatedFrameBankEntry>,
}

pub(super) const NATIVE_EVENT61_FRAME_COUNT: i64 = 59;
pub(super) const NATIVE_ENDING_PREFIX_FRAME_COUNT: i64 = 111;
pub(super) const NATIVE_CH20_SKY_KEY_FRAME_COUNT: i64 = 101;
pub(super) const NATIVE_PENDING_CODE1_FRAME_COUNT: i64 = 2;

struct FrameBankContract {
    label: &'static str,
    asset_id: &'static str,
    resource: i64,
    raw_size: i64,
    raw_md5: &'static str,
    raw_sha256: &'static str,
    frame_count: i64,
    /// Go 允許 nil 幾何函式並拒收；這裡四個契約都有函式，該分支不存在。
    geometry: fn(&SeparatedFrameBankEntry) -> bool,
}

/// 載入 `sub_356B7` 消費的 FDOTHER #45 圖格表。絕不開啟或退回原始封存。
pub fn load_separated_event61_frames(root: impl AsRef<Path>) -> Result<Vec<Frame>> {
    load_separated_frame_bank(
        root.as_ref(),
        &FrameBankContract {
            label: "event61",
            asset_id: "animation/FDOTHER_045/event61",
            resource: 45,
            raw_size: 2073,
            raw_md5: "f26b4bc4ce9c8e9c9d12ec2e2f00f5b8",
            raw_sha256: "d7a0e73d7dec5403441c33cd84392a7683de329f7b0914a1bf23c47ccc56513a",
            frame_count: NATIVE_EVENT61_FRAME_COUNT,
            geometry: |e| e.x == 0 && e.y == 0 && e.width == 8 && e.height == 2,
        },
    )
}

/// 載入 `sub_2BCE5` 消費的完整 FDOTHER #54 圖格表。絕不開啟或退回原始封存。
pub fn load_separated_ending_prefix_frames(root: impl AsRef<Path>) -> Result<Vec<Frame>> {
    load_separated_frame_bank(
        root.as_ref(),
        &FrameBankContract {
            label: "ending prefix",
            asset_id: "animation/FDOTHER_054/ending_prefix",
            resource: 54,
            raw_size: 263655,
            raw_md5: "4a3853ed5570a319e3a7e6024fb8e84e",
            raw_sha256: "1e6cb912150102df4404a741846305033fe32f55db7d3a694ccb7fcc63496a46",
            frame_count: NATIVE_ENDING_PREFIX_FRAME_COUNT,
            geometry: valid_ending_prefix_geometry,
        },
    )
}

/// 載入 `sub_24336` 消費的完整 FDOTHER #34 圖格表。絕不開啟或退回原始封存。
pub fn load_separated_ch20_sky_key_frames(root: impl AsRef<Path>) -> Result<Vec<Frame>> {
    load_separated_frame_bank(
        root.as_ref(),
        &FrameBankContract {
            label: "ch20 sky key",
            asset_id: "animation/FDOTHER_034/ch20_sky_key",
            resource: 34,
            raw_size: 102345,
            raw_md5: "84dca404546a3a407d72f139cb934a40",
            raw_sha256: "53f120fa4b1fab74c6b3998ec3ef8a9a2363461980ad38a7ffef2400e79b0c4d",
            frame_count: NATIVE_CH20_SKY_KEY_FRAME_COUNT,
            geometry: valid_ch20_sky_key_geometry,
        },
    )
}

/// 載入 `sub_22E5C` 消費的固定 FDOTHER #79 圖格表。高階場景命名刻意從缺。
pub fn load_separated_pending_code1_frames(root: impl AsRef<Path>) -> Result<Vec<Frame>> {
    load_separated_frame_bank(
        root.as_ref(),
        &FrameBankContract {
            label: "pending code 1",
            asset_id: "animation/FDOTHER_079/pending_code1",
            resource: 79,
            raw_size: 6801,
            raw_md5: "5f7eeeeff593ad7a067af167ef92670e",
            raw_sha256: "34bb99917cdfd6b268674d7c1cf201cc3749cf3a58c574161dc5d37a36781373",
            frame_count: NATIVE_PENDING_CODE1_FRAME_COUNT,
            geometry: |e| {
                const WANT: [[i64; 4]; 2] = [[69, 61, 181, 75], [130, 141, 55, 8]];
                if e.index < 0 || e.index >= WANT.len() as i64 {
                    return false;
                }
                [e.x, e.y, e.width, e.height] == WANT[e.index as usize]
            },
        },
    )
}

/// 位於 320×200 之內；`x+width` 等加法照 Go 的 int 繞回。
fn within_screen(e: &SeparatedFrameBankEntry) -> bool {
    !(e.x < 0
        || e.y < 0
        || e.width <= 0
        || e.height <= 0
        || e.x.wrapping_add(e.width) > 320
        || e.y.wrapping_add(e.height) > 200)
}

fn valid_ch20_sky_key_geometry(e: &SeparatedFrameBankEntry) -> bool {
    if !within_screen(e) {
        return false;
    }
    let geometry = match e.index {
        0 => [148, 94, 24, 24],
        68 | 69 => [146, 95, 32, 23],
        100 => [148, 94, 24, 36],
        _ => return true,
    };
    [e.x, e.y, e.width, e.height] == geometry
}

fn valid_ending_prefix_geometry(e: &SeparatedFrameBankEntry) -> bool {
    if !within_screen(e) {
        return false;
    }
    let geometry = match e.index {
        0 => [0, 23, 320, 132],
        9 | 108 => [116, 39, 86, 81],
        110 => [0, 0, 320, 200],
        _ => return true,
    };
    [e.x, e.y, e.width, e.height] == geometry
}

fn load_separated_frame_bank(root: &Path, contract: &FrameBankContract) -> Result<Vec<Frame>> {
    let label = contract.label;
    let document: SeparatedFrameBankDocument = read_document(
        &root.join("bank.json"),
        &format!("fdother: separated {label} metadata: "),
        &format!("fdother: separated {label} JSON: "),
    )?;
    if document.schema_version != 1
        || document.kind != "fdother_frame_bank"
        || document.asset_id != contract.asset_id
        || document.status != "decoded"
        || document.evidence != "confirmed"
        || document.codec != "fd2_2935b_frame_table"
        || document.source.file != "FDOTHER.DAT"
        || document.source.resource != contract.resource
        || document.source.size != 3382481
        || document.source.md5 != "22f56e5027edc7c766ad34ca4e5aca93"
        || document.source.sha256 != "a81b13493725fb70e750c4d9e0dce4e1b57d0df312c4ad4157e6d45171b13bce"
        || document.source.raw_size != contract.raw_size
        || document.source.raw_md5 != contract.raw_md5
        || document.source.raw_sha256 != contract.raw_sha256
        || document.frames.len() as i64 != contract.frame_count
    {
        return Err(FdotherError::message(format!("fdother: separated {label} contract mismatch")));
    }
    let count = contract.frame_count;
    let mut frames = vec![Frame::default(); count as usize];
    let mut seen = vec![false; count as usize];
    for entry in &document.frames {
        let index = entry.index;
        if index < 0 || index >= count || seen[index as usize] || !(contract.geometry)(entry) {
            return Err(FdotherError::message(format!("fdother: invalid separated {label} frame {index}")));
        }
        let prefix = format!("frame_{index:03}/");
        if entry.frame != format!("{prefix}frame.png") || entry.mask != format!("{prefix}mask.png") {
            return Err(FdotherError::message(format!("fdother: {label} frame {index} path mismatch")));
        }
        let indexed = read_paletted_surface(&root.join(&entry.frame), entry.width, entry.height)
            .map_err(|e| wrap_go(e, format!("fdother: {label} frame {index} pixels: ")))?;
        let mask = read_binary_mask(&root.join(&entry.mask), entry.width, entry.height)
            .map_err(|e| wrap_go(e, format!("fdother: {label} frame {index} mask: ")))?;
        frames[index as usize] = Frame::separated(entry.x, entry.y, entry.width, entry.height, indexed, mask);
        seen[index as usize] = true;
    }
    if let Some(index) = seen.iter().position(|&present| !present) {
        return Err(FdotherError::message(format!("fdother: {label} frame {index} is missing")));
    }
    Ok(frames)
}

/// Go `fmt.Errorf(prefix+"%w", err)`，err 本身是 fdother 錯誤：Go 文字的部分接上前綴，
/// 外部函式庫的錯誤只延長前綴。
pub(super) fn wrap_go(err: FdotherError, prefix: String) -> FdotherError {
    match err {
        FdotherError::External { context, detail } => FdotherError::External { context: prefix + &context, detail },
        FdotherError::Archive(e @ crate::archive::ArchiveError::Io { .. }) => {
            FdotherError::external(prefix, e.to_string())
        }
        other => FdotherError::message(format!("{prefix}{other}")),
    }
}
