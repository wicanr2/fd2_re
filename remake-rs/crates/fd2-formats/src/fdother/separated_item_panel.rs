//! 移植自 Go `separated_item_panel.go`：FDOTHER #5 的道具／狀態／移交面板混合編碼子集。

use super::go_compat::{GoImage, decode_png_file, nullable, nullable_struct, nullable_struct_vec};
use super::separated_surface::read_document;
use super::{FdotherError, Frame, LMI1Entry, RawCell, Result};
use serde::Deserialize;
use std::collections::BTreeMap;
use std::path::Path;

const ITEM_PANEL_SOURCE_SIZE: i64 = 3382481;
const ITEM_PANEL_SOURCE_MD5: &str = "22f56e5027edc7c766ad34ca4e5aca93";
const ITEM_PANEL_SOURCE_SHA256: &str = "a81b13493725fb70e750c4d9e0dce4e1b57d0df312c4ad4157e6d45171b13bce";
const ITEM_PANEL_RESOURCE_SIZE: i64 = 44181;

pub(super) const ITEM_PANEL_OPAQUE_ENTRIES: [i64; 3] = [20, 21, 22];
pub(super) const ITEM_PANEL_RAW_ENTRIES: [i64; 43] = [
    0, 1, 2, 3, 4, 5, 6, 7, 8, 9, 10, 11, 12, 13, 14, 15, 16, 17, 18, 19, 23, 24, 25, 26, 27, 28, 29, 30, 53, 54, 55,
    56, 57, 59, 60, 61, 62, 63, 64, 65, 66, 67, 92,
];
pub(super) const ITEM_PANEL_FRAME_ENTRIES: [i64; 42] = [
    31, 32, 33, 34, 35, 36, 37, 38, 39, 40, 41, 42, 43, 44, 45, 46, 47, 48, 49, 50, 51, 52, 93, 119, 120, 121, 122,
    123, 124, 125, 126, 127, 128, 129, 130, 131, 132, 133, 134, 135, 136, 137,
];

#[derive(Debug, Default, Deserialize)]
#[serde(default)]
pub(super) struct SeparatedItemPanelSource {
    #[serde(deserialize_with = "nullable")]
    pub file: String,
    #[serde(deserialize_with = "nullable")]
    pub resource: i64,
    #[serde(deserialize_with = "nullable")]
    pub size: i64,
    #[serde(deserialize_with = "nullable")]
    pub md5: String,
    #[serde(deserialize_with = "nullable")]
    pub sha256: String,
    #[serde(deserialize_with = "nullable")]
    pub raw_size: i64,
}

#[derive(Debug, Default, Deserialize)]
#[serde(default)]
pub(super) struct SeparatedItemPanelEntry {
    #[serde(deserialize_with = "nullable")]
    pub index: i64,
    #[serde(deserialize_with = "nullable")]
    pub codec: String,
    #[serde(deserialize_with = "nullable")]
    pub width: i64,
    #[serde(deserialize_with = "nullable")]
    pub height: i64,
    #[serde(deserialize_with = "nullable")]
    pub frame: String,
    #[serde(deserialize_with = "nullable")]
    pub mask: String,
}

/// Go `separatedItemPanelDocument`；載入格、動作格與教會介面共用。
#[derive(Debug, Default, Deserialize)]
#[serde(default)]
pub(super) struct SeparatedItemPanelDocument {
    #[serde(deserialize_with = "nullable")]
    pub schema_version: i64,
    #[serde(deserialize_with = "nullable")]
    pub kind: String,
    #[serde(deserialize_with = "nullable")]
    pub asset_id: String,
    #[serde(deserialize_with = "nullable")]
    pub status: String,
    #[serde(deserialize_with = "nullable")]
    pub evidence: String,
    #[serde(deserialize_with = "nullable_struct")]
    pub source: SeparatedItemPanelSource,
    #[serde(deserialize_with = "nullable_struct_vec")]
    pub entries: Vec<SeparatedItemPanelEntry>,
}

/// 共用道具／狀態／移交繪製器消費的混合編碼子集。Go 以 `map[int]` 保存，這裡用 `BTreeMap`。
#[derive(Debug, Clone, Default)]
pub struct SeparatedItemPanelEntries {
    pub opaque: BTreeMap<i64, LMI1Entry>,
    pub raw: BTreeMap<i64, RawCell>,
    pub frames: BTreeMap<i64, Frame>,
}

fn expected_item_panel_codec(index: i64) -> &'static str {
    if ITEM_PANEL_OPAQUE_ENTRIES.contains(&index) {
        "opaque_high_run"
    } else if ITEM_PANEL_RAW_ENTRIES.contains(&index) {
        "raw_indexed_opaque"
    } else if ITEM_PANEL_FRAME_ENTRIES.contains(&index) {
        "four_mode_frame"
    } else {
        ""
    }
}

pub fn load_separated_item_panel_entries(root: impl AsRef<Path>) -> Result<SeparatedItemPanelEntries> {
    let directory = root.as_ref().join("fdother_005_item_panel");
    let document: SeparatedItemPanelDocument = read_document(
        &directory.join("resource.json"),
        "fdother: separated item panel metadata: ",
        "fdother: separated item panel JSON: ",
    )?;
    let want_count = ITEM_PANEL_OPAQUE_ENTRIES.len() + ITEM_PANEL_RAW_ENTRIES.len() + ITEM_PANEL_FRAME_ENTRIES.len();
    if document.schema_version != 1
        || document.kind != "fdother_lmi1_item_panel"
        || document.asset_id != "ui/FDOTHER_005/item_panel"
        || document.status != "decoded"
        || document.evidence != "confirmed"
        || document.source.file != "FDOTHER.DAT"
        || document.source.resource != 5
        || document.source.size != ITEM_PANEL_SOURCE_SIZE
        || document.source.md5 != ITEM_PANEL_SOURCE_MD5
        || document.source.sha256 != ITEM_PANEL_SOURCE_SHA256
        || document.source.raw_size != ITEM_PANEL_RESOURCE_SIZE
        || document.entries.len() != want_count
    {
        return Err(FdotherError::fixed("fdother: separated item panel metadata mismatch"));
    }
    let mut result = SeparatedItemPanelEntries::default();
    let mut seen = std::collections::BTreeSet::new();
    for entry in &document.entries {
        let index = entry.index;
        let codec = expected_item_panel_codec(index);
        if codec.is_empty()
            || seen.contains(&index)
            || entry.codec != codec
            || entry.width <= 0
            || entry.height <= 0
            || entry.frame != format!("entry_{index:03}/frame.png")
        {
            return Err(FdotherError::message(format!(
                "fdother: separated item panel entry {index} metadata mismatch"
            )));
        }
        seen.insert(index);
        let pixels = load_item_panel_indexed_png(&directory.join(&entry.frame), entry.width, entry.height)
            .map_err(|e| e.wrap(format!("fdother: separated item panel entry {index}: ")))?;
        match codec {
            "opaque_high_run" => {
                if !entry.mask.is_empty() {
                    return Err(FdotherError::message(format!("fdother: opaque item panel entry {index} has a mask")));
                }
                result.opaque.insert(index, LMI1Entry { width: entry.width, height: entry.height, pixels });
            }
            "raw_indexed_opaque" => {
                if !entry.mask.is_empty() {
                    return Err(FdotherError::message(format!("fdother: raw item panel entry {index} has a mask")));
                }
                result.raw.insert(index, RawCell { width: entry.width, height: entry.height, pixels });
            }
            _ => {
                if entry.mask != format!("entry_{index:03}/mask.png") {
                    return Err(FdotherError::message(format!(
                        "fdother: frame item panel entry {index} lacks its mask"
                    )));
                }
                let mask = load_item_panel_mask_png(&directory.join(&entry.mask), entry.width, entry.height)
                    .map_err(|e| e.wrap(format!("fdother: separated item panel entry {index} mask: ")))?;
                result.frames.insert(index, Frame::separated(0, 0, entry.width, entry.height, pixels, mask));
            }
        }
    }
    Ok(result)
}

/// 回傳 `sub_1B1E7` 使用的四個來源索引項目。原始索引與幾何是帶證據的身分；不替美術
/// 指定現代語意名稱。
pub fn load_separated_system_info_panels(root: impl AsRef<Path>) -> Result<[Frame; 4]> {
    let entries = load_separated_item_panel_entries(root)?;
    let want_geometry: [(i64, i64); 4] = [(102, 17), (170, 117), (170, 16), (63, 15)];
    let mut panels: [Frame; 4] = Default::default();
    for (offset, (w, h)) in want_geometry.into_iter().enumerate() {
        let index = 0x85 + offset as i64;
        match entries.frames.get(&index) {
            Some(entry)
                if entry.width == w
                    && entry.height == h
                    && entry.indexed.len() as i64 == w * h
                    && entry.mask.len() == entry.indexed.len() =>
            {
                panels[offset] = entry.clone();
            }
            _ => {
                return Err(FdotherError::message(format!(
                    "fdother: separated system info entry {index:#x} is invalid"
                )));
            }
        }
    }
    Ok(panels)
}

/// 單張 PNG 讀取的錯誤：外部函式庫的錯誤，或 Go 版自己的文字。
pub(super) enum PngLoadError {
    External(String),
    Contract(&'static str),
}

impl PngLoadError {
    /// Go `fmt.Errorf(prefix+"%w", err)`。
    pub(super) fn wrap(self, prefix: String) -> FdotherError {
        match self {
            PngLoadError::External(detail) => FdotherError::external(prefix, detail),
            PngLoadError::Contract(message) => FdotherError::message(format!("{prefix}{message}")),
        }
    }
}

/// Go `loadItemPanelIndexedPNG`：寬高相符的 Paletted PNG。Go 的 `Stride != width` 對解碼出的
/// 影像恆不成立，所以不另外檢查。
pub(super) fn load_item_panel_indexed_png(
    path: &Path,
    width: i64,
    height: i64,
) -> std::result::Result<Vec<u8>, PngLoadError> {
    match decode_png_file(path, false).map_err(PngLoadError::External)? {
        GoImage::Paletted { width: w, height: h, pix, .. } if w == width && h == height => Ok(pix),
        _ => Err(PngLoadError::Contract("indexed PNG mode or geometry mismatch")),
    }
}

/// Go `loadItemPanelMaskPNG`：寬高相符、值只能是 0／0xff 的 Gray PNG。
pub(super) fn load_item_panel_mask_png(
    path: &Path,
    width: i64,
    height: i64,
) -> std::result::Result<Vec<u8>, PngLoadError> {
    let GoImage::Gray { width: w, height: h, pix } = decode_png_file(path, false).map_err(PngLoadError::External)?
    else {
        return Err(PngLoadError::Contract("mask PNG mode or geometry mismatch"));
    };
    if w != width || h != height {
        return Err(PngLoadError::Contract("mask PNG mode or geometry mismatch"));
    }
    if pix.iter().any(|&v| v != 0 && v != 0xff) {
        return Err(PngLoadError::Contract("mask PNG is not binary"));
    }
    Ok(pix)
}
