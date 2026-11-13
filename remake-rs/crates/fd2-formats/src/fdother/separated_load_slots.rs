//! 移植自 Go `separated_load_slots.go`：原版四格 LOAD 畫面使用的 FDOTHER #13 entry16。

use super::separated_item_panel::{SeparatedItemPanelDocument, load_item_panel_indexed_png};
use super::separated_surface::read_document;
use super::{FdotherError, LMI1Entry, Result};
use std::path::Path;

const LOAD_SLOTS_SOURCE_SIZE: i64 = 3382481;
const LOAD_SLOTS_SOURCE_MD5: &str = "22f56e5027edc7c766ad34ca4e5aca93";
const LOAD_SLOTS_SOURCE_SHA256: &str = "a81b13493725fb70e750c4d9e0dce4e1b57d0df312c4ad4157e6d45171b13bce";
const LOAD_SLOTS_RESOURCE_SIZE: i64 = 53210;

/// 載入原版四格 LOAD 畫面使用的那一筆 FDOTHER #13 項目。封存解碼屬於離線匯入工具；
/// 正式執行期只接受驗證過的標準素材包。
pub fn load_separated_load_slots_frame(root: impl AsRef<Path>) -> Result<LMI1Entry> {
    let directory = root.as_ref().join("fdother_013_load_slots");
    let document: SeparatedItemPanelDocument = read_document(
        &directory.join("resource.json"),
        "fdother: separated load-slots metadata: ",
        "fdother: separated load-slots JSON: ",
    )?;
    if document.schema_version != 1
        || document.kind != "fdother_lmi1_load_slots"
        || document.asset_id != "ui/FDOTHER_013/load_slots"
        || document.status != "decoded"
        || document.evidence != "confirmed"
        || document.source.file != "FDOTHER.DAT"
        || document.source.resource != 13
        || document.source.size != LOAD_SLOTS_SOURCE_SIZE
        || document.source.md5 != LOAD_SLOTS_SOURCE_MD5
        || document.source.sha256 != LOAD_SLOTS_SOURCE_SHA256
        || document.source.raw_size != LOAD_SLOTS_RESOURCE_SIZE
        || document.entries.len() != 1
    {
        return Err(FdotherError::fixed("fdother: separated load-slots metadata mismatch"));
    }
    let entry = &document.entries[0];
    if entry.index != 16
        || entry.codec != "opaque_high_run"
        || entry.width != 310
        || entry.height != 86
        || entry.frame != "entry_016/frame.png"
        || !entry.mask.is_empty()
    {
        return Err(FdotherError::fixed("fdother: separated load-slots entry metadata mismatch"));
    }
    let pixels = load_item_panel_indexed_png(&directory.join(&entry.frame), entry.width, entry.height)
        .map_err(|e| e.wrap("fdother: separated load-slots frame: ".to_string()))?;
    Ok(LMI1Entry { width: entry.width, height: entry.height, pixels })
}
