//! 移植自 Go `separated_action_cells.go`：完整的 FDOTHER #2 原始圖格庫。

use super::separated_item_panel::{SeparatedItemPanelDocument, load_item_panel_indexed_png};
use super::separated_surface::read_document;
use super::{FdotherError, RawCell, Result};
use std::path::Path;

const ACTION_CELLS_SOURCE_SIZE: i64 = 3382481;
const ACTION_CELLS_SOURCE_MD5: &str = "22f56e5027edc7c766ad34ca4e5aca93";
const ACTION_CELLS_SOURCE_SHA256: &str = "a81b13493725fb70e750c4d9e0dce4e1b57d0df312c4ad4157e6d45171b13bce";
const ACTION_CELLS_RESOURCE_SIZE: i64 = 37680;
const ACTION_CELLS_COUNT: i64 = 78;

/// 回傳完整的 FDOTHER #2 原始圖格庫。像素值 0 在繪圖時仍是保留目的端的索引。
pub fn load_separated_action_cells(root: impl AsRef<Path>) -> Result<Vec<RawCell>> {
    let directory = root.as_ref().join("action_cells");
    let document: SeparatedItemPanelDocument = read_document(
        &directory.join("resource.json"),
        "fdother: separated action-cell metadata: ",
        "fdother: separated action-cell JSON: ",
    )?;
    if document.schema_version != 1
        || document.kind != "fdother_raw_cell_bank"
        || document.asset_id != "ui/FDOTHER_002/action_cells"
        || document.status != "decoded"
        || document.evidence != "confirmed"
        || document.source.file != "FDOTHER.DAT"
        || document.source.resource != 2
        || document.source.size != ACTION_CELLS_SOURCE_SIZE
        || document.source.md5 != ACTION_CELLS_SOURCE_MD5
        || document.source.sha256 != ACTION_CELLS_SOURCE_SHA256
        || document.source.raw_size != ACTION_CELLS_RESOURCE_SIZE
        || document.entries.len() as i64 != ACTION_CELLS_COUNT
    {
        return Err(FdotherError::fixed("fdother: separated action-cell metadata mismatch"));
    }
    let count = ACTION_CELLS_COUNT as usize;
    let mut cells = vec![RawCell::default(); count];
    let mut seen = vec![false; count];
    for entry in &document.entries {
        let index = entry.index;
        if !(0..ACTION_CELLS_COUNT).contains(&index)
            || seen[index as usize]
            || entry.codec != "raw_indexed_transparent"
            || entry.width <= 0
            || entry.height <= 0
            || entry.frame != format!("cell_{index:03}.png")
            || !entry.mask.is_empty()
        {
            return Err(FdotherError::message(format!("fdother: action-cell entry {index} metadata mismatch")));
        }
        let pixels = load_item_panel_indexed_png(&directory.join(&entry.frame), entry.width, entry.height)
            .map_err(|e| e.wrap(format!("fdother: action cell {index}: ")))?;
        cells[index as usize] = RawCell { width: entry.width, height: entry.height, pixels };
        seen[index as usize] = true;
    }
    if let Some(index) = seen.iter().position(|&present| !present) {
        return Err(FdotherError::message(format!("fdother: action cell {index} is missing")));
    }
    Ok(cells)
}
