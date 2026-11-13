//! 移植自 Go `separated_church_ui.go`：教會選單、名冊、轉職與復活呈現使用的 FDOTHER #14 子集。

use super::separated_item_panel::{SeparatedItemPanelDocument, load_item_panel_indexed_png, load_item_panel_mask_png};
use super::separated_surface::read_document;
use super::{FdotherError, Frame, LMI1Entry, RawCell, Result};
use std::collections::BTreeSet;
use std::path::Path;

const CHURCH_UI_SOURCE_SIZE: i64 = 3382481;
const CHURCH_UI_SOURCE_MD5: &str = "22f56e5027edc7c766ad34ca4e5aca93";
const CHURCH_UI_SOURCE_SHA256: &str = "a81b13493725fb70e750c4d9e0dce4e1b57d0df312c4ad4157e6d45171b13bce";
const CHURCH_UI_RESOURCE_SIZE: i64 = 51157;

pub(super) const CHURCH_UI_OPAQUE_ENTRIES: [i64; 2] = [1, 16];
pub(super) const CHURCH_UI_RAW_ENTRIES: [i64; 9] = [3, 4, 5, 6, 7, 8, 9, 10, 15];
pub(super) const CHURCH_UI_FRAME_ENTRIES: [i64; 10] = [0, 23, 24, 25, 26, 27, 28, 29, 30, 31];

/// 教會選單、名冊、轉職與復活呈現路徑消費的 FDOTHER #14 子集。
#[derive(Debug, Clone, Default)]
pub struct SeparatedChurchUIAssets {
    pub background: Frame,
    pub entries: Vec<LMI1Entry>,
    pub price_cell: RawCell,
    pub revive_fx: Vec<Frame>,
}

fn expected_church_ui_codec(index: i64) -> &'static str {
    if CHURCH_UI_OPAQUE_ENTRIES.contains(&index) {
        "opaque_high_run"
    } else if CHURCH_UI_RAW_ENTRIES.contains(&index) {
        "raw_indexed_opaque"
    } else if CHURCH_UI_FRAME_ENTRIES.contains(&index) {
        "four_mode_frame"
    } else {
        ""
    }
}

pub fn load_separated_church_ui_assets(root: impl AsRef<Path>) -> Result<SeparatedChurchUIAssets> {
    let directory = root.as_ref().join("fdother_014_church");
    let document: SeparatedItemPanelDocument = read_document(
        &directory.join("resource.json"),
        "fdother: separated church UI metadata: ",
        "fdother: separated church UI JSON: ",
    )?;
    let want_count = CHURCH_UI_OPAQUE_ENTRIES.len() + CHURCH_UI_RAW_ENTRIES.len() + CHURCH_UI_FRAME_ENTRIES.len();
    if document.schema_version != 1
        || document.kind != "fdother_lmi1_church_ui"
        || document.asset_id != "ui/FDOTHER_014/church"
        || document.status != "decoded"
        || document.evidence != "confirmed"
        || document.source.file != "FDOTHER.DAT"
        || document.source.resource != 14
        || document.source.size != CHURCH_UI_SOURCE_SIZE
        || document.source.md5 != CHURCH_UI_SOURCE_MD5
        || document.source.sha256 != CHURCH_UI_SOURCE_SHA256
        || document.source.raw_size != CHURCH_UI_RESOURCE_SIZE
        || document.entries.len() != want_count
    {
        return Err(FdotherError::fixed("fdother: separated church UI metadata mismatch"));
    }
    let mut result = SeparatedChurchUIAssets {
        entries: vec![LMI1Entry::default(); 17],
        revive_fx: vec![Frame::default(); 9],
        ..Default::default()
    };
    let mut seen = BTreeSet::new();
    for entry in &document.entries {
        let index = entry.index;
        let codec = expected_church_ui_codec(index);
        if codec.is_empty()
            || seen.contains(&index)
            || entry.codec != codec
            || entry.width <= 0
            || entry.height <= 0
            || entry.frame != format!("entry_{index:03}/frame.png")
        {
            return Err(FdotherError::message(format!("fdother: separated church UI entry {index} metadata mismatch")));
        }
        seen.insert(index);
        let pixels = load_item_panel_indexed_png(&directory.join(&entry.frame), entry.width, entry.height)
            .map_err(|e| e.wrap(format!("fdother: separated church UI entry {index}: ")))?;
        match codec {
            "opaque_high_run" => {
                if !entry.mask.is_empty() || index >= result.entries.len() as i64 {
                    return Err(FdotherError::message(format!("fdother: invalid opaque church UI entry {index}")));
                }
                result.entries[index as usize] = LMI1Entry { width: entry.width, height: entry.height, pixels };
            }
            "raw_indexed_opaque" => {
                if !entry.mask.is_empty() {
                    return Err(FdotherError::message(format!("fdother: invalid raw church UI entry {index}")));
                }
                if index == 15 {
                    result.price_cell = RawCell { width: entry.width, height: entry.height, pixels };
                } else {
                    // 0x2d77a/0x2d8e4/0x2da7e 呼叫 0x4e9e4：原始像素，在既有服務格的
                    // BlitAt 中 0 是透明。
                    if !(3..=10).contains(&index) || entry.width != 24 || entry.height != 20 {
                        return Err(FdotherError::message(format!("fdother: invalid raw church service {index}")));
                    }
                    result.entries[index as usize] = LMI1Entry { width: entry.width, height: entry.height, pixels };
                }
            }
            _ => {
                if entry.mask != format!("entry_{index:03}/mask.png") {
                    return Err(FdotherError::message(format!("fdother: church UI frame {index} lacks mask")));
                }
                let mask = load_item_panel_mask_png(&directory.join(&entry.mask), entry.width, entry.height)
                    .map_err(|e| e.wrap(format!("fdother: church UI frame {index} mask: ")))?;
                let frame = Frame::separated(0, 0, entry.width, entry.height, pixels, mask);
                if index == 0 {
                    if entry.width != 320 || entry.height != 200 {
                        return Err(FdotherError::fixed("fdother: church UI background is not 320x200"));
                    }
                    result.background = frame;
                } else {
                    result.revive_fx[(index - 23) as usize] = frame;
                }
            }
        }
    }
    for index in CHURCH_UI_OPAQUE_ENTRIES {
        if !seen.contains(&index) {
            return Err(FdotherError::message(format!("fdother: missing church UI entry {index}")));
        }
    }
    Ok(result)
}
