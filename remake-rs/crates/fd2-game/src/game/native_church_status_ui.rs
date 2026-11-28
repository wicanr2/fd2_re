//! Go 來源：`remake/cmd/fd2/native_church_status_ui.go`（265 行）。
//!
//! 本檔移植原版 0x17AED 狀態面板的資料準備與幀合成，供戰場玩家狀態（`beginNativePlayerStatus`）使用。
//! 教會的狀態檢視在 Rust 走 `native_church_ui/native_church_panels.rs` 的既有實作；Go 的
//! `prepareNativeChurchStatus`、`beginNativeChurchStatus` 等教會入口仍標 `r8`。

use super::*;

impl Game {
    /// 讀分離素材包的面板底圖、頭像、資料素材與物品效果表，合成能力值頁與指令頁（沒有指令時只有能力值頁）。
    ///
    /// Go：`native_church_status_ui.go:24`，`func (g *Game) prepareNativeUnitStatusData(unit *battle.Unit) (*nativeUnitStatusData, []byte, []byte, bool)`
    /// Go 回 `(data, status, commands, ok)`；Rust 以 `None` 表示 `ok=false`，沒有指令頁時 `commands` 是空向量（Go 的 nil）。
    pub(crate) fn prepare_native_unit_status_data(
        &self,
        unit: Option<&battle::Unit>,
    ) -> Option<(NativeUnitStatusData, Vec<u8>, Vec<u8>)> {
        let unit = unit?;
        let asset_pack_root = separated_asset_path("");
        let portrait_root = separated_asset_path("portraits");
        if asset_pack_root.is_empty() {
            return None;
        }
        let record = battle::native_item_panel_record_for_unit(Some(unit)).ok()?;
        let mut base = vec![0u8; 320 * 200];
        battle::render_native_item_panel_base_resources(
            &asset_pack_root,
            &portrait_root,
            i64::from(record[7]),
            &mut base,
        )
        .ok()?;
        let assets = battle::load_native_item_panel_data_assets(&asset_pack_root).ok()?;
        let item_rows =
            battle::load_native_item_effect_row_prefix(&asset_path("assets/data/native_item_effect_rows.json")).ok()?;
        let data = NativeUnitStatusData {
            record,
            base,
            item_rows,
            assets,
            command_ids: unit.native_command_ids(),
            command_book: self.native_command_book.clone(),
        };
        let status = data.compose(&data.assets, false)?;
        if data.command_ids.is_empty() {
            return Some((data, status, Vec::new()));
        }
        let commands = data.compose(&data.assets, true)?;
        Some((data, status, commands))
    }
}

/// 開啟（`opening=true`，幀 11→0）或關閉（幀 0→11）的 12 幀排程。
///
/// Go：`native_church_status_ui.go:181`，`func nativeChurchPanelFrames(source, panel []byte, opening bool) ([][]byte, error)`
pub(crate) fn native_church_panel_frames(source: &[u8], panel: &[u8], opening: bool) -> Result<Vec<Vec<u8>>> {
    if source.len() != 320 * 200 || panel.len() != 320 * 200 {
        return Err(game_err!("native church status: panel frames require two 320x200 buffers"));
    }
    let order: Vec<i64> = if opening { (0..=11).rev().collect() } else { (0..=11).collect() };
    order.into_iter().map(|i| native_church_panel_frame(source, panel, i)).collect()
}

/// Go：`native_church_status_ui.go:206`，`func nativeChurchPanelFrame(source, panel []byte, index int) ([]byte, error)`
pub(crate) fn native_church_panel_frame(source: &[u8], panel: &[u8], index: i64) -> Result<Vec<u8>> {
    let regions = native_church_panel_regions(index);
    let (Some(regions), true, true) = (regions, source.len() == 320 * 200, panel.len() == 320 * 200) else {
        return Err(game_err!("native church status: invalid panel buffers"));
    };
    let mut frame = source.to_vec();
    for region in &regions {
        copy_native_church_panel_region(&mut frame, panel, region);
    }
    Ok(frame)
}

/// Go：`native_church_status_ui.go:218`，`func nativeChurchPanelRegions(index int) ([]battle.NativeItemPanelRegion, bool)`
/// Go 在錯誤時仍回零值三格與 `false`；呼叫端只看 `ok`，Rust 回 `None`。
pub(crate) fn native_church_panel_regions(index: i64) -> Option<Vec<battle::NativeItemPanelRegion>> {
    let pass = battle::native_item_panel_frame_for(index).ok()?;
    Some(vec![pass.left, pass.upper, pass.bottom])
}

/// 能力值頁與指令頁之間的下半部捲動：`step` 0..6，下半部每步下移 16 列。
///
/// Go：`native_church_status_ui.go:223`，`func nativeChurchBottomPanelFrame(source, panel []byte, step int) ([]byte, error)`
pub(crate) fn native_church_bottom_panel_frame(source: &[u8], panel: &[u8], step: i64) -> Result<Vec<u8>> {
    let regions = native_church_bottom_panel_regions(step);
    let (true, true, Some(regions)) = (source.len() == 320 * 200, panel.len() == 320 * 200, regions) else {
        return Err(game_err!("native church status: invalid bottom transition"));
    };
    let mut frame = source.to_vec();
    for region in &regions {
        copy_native_church_panel_region(&mut frame, panel, region);
    }
    Ok(frame)
}

/// Go：`native_church_status_ui.go:235`，`func nativeChurchBottomPanelRegions(step int) ([]battle.NativeItemPanelRegion, bool)`
pub(crate) fn native_church_bottom_panel_regions(step: i64) -> Option<Vec<battle::NativeItemPanelRegion>> {
    if !(0..=6).contains(&step) {
        return None;
    }
    let region = |source_x, source_y, dest_y, width, height, enabled| battle::NativeItemPanelRegion {
        enabled,
        source_x,
        source_y,
        dest_x: source_x,
        dest_y,
        width,
        height,
    };
    let destination_y = 94 + 16 * step;
    let mut height = 102;
    if destination_y + height > 200 {
        height = 200 - destination_y;
    }
    Some(vec![
        region(5, 7, 7, 86, 86, true),
        region(92, 7, 7, 223, 86, true),
        region(5, 94, destination_y, 310, height, height > 0),
    ])
}

/// Go：`native_church_status_ui.go:257`，`func copyNativeChurchPanelRegion(dst, src []byte, region battle.NativeItemPanelRegion)`
pub(crate) fn copy_native_church_panel_region(dst: &mut [u8], src: &[u8], region: &battle::NativeItemPanelRegion) {
    if !region.enabled || region.width <= 0 || region.height <= 0 {
        return;
    }
    let width = region.width as usize;
    for y in 0..region.height {
        let source = ((region.source_y + y) * 320 + region.source_x) as usize;
        let destination = ((region.dest_y + y) * 320 + region.dest_x) as usize;
        dst[destination..destination + width].copy_from_slice(&src[source..source + width]);
    }
}

#[cfg(test)]
#[path = "native_church_status_ui/native_church_status_ui_test.rs"]
mod native_church_status_ui_test;
