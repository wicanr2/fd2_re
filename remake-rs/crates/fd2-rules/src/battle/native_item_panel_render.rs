//! Go 來源：`remake/internal/battle/native_item_panel_render.go`（146 行）。

use super::*;

/// Go 原式：`320 * 200`
pub(crate) const NATIVE_ITEM_PANEL_BYTES: i64 = 64000;

/// RenderNativeItemPanelBase executes the complete recovered 0x17eef ordering:
/// 0x168b6 raw grid, DATO frame zero, then FDOTHER#5 entries 20 and 21 through
/// opaque 0x4e8af writes. 0x17fc0's dynamic data overlays are deliberately a
/// separate pass.
///
/// Go：`native_item_panel_render.go:18`，`func RenderNativeItemPanelBase( cells []fdother.RawCell, portrait dato.Frame, upper, bottom fdother.LMI1Entry, dst []byte, ) error`
pub fn render_native_item_panel_base(
    cells: &[fdother::RawCell],
    portrait: &dato::Frame,
    upper: &fdother::LMI1Entry,
    bottom: &fdother::LMI1Entry,
    dst: &mut [u8],
) -> Result<()> {
    todo!("battle/native_item_panel_render.go:RenderNativeItemPanelBase")
}

/// RenderNativeItemPanelBaseResources loads the separated FDOTHER #5 UI bank
/// and portrait pack proven by 0x17eef. The portrait resource is the caller's
/// unit-record byte +7; frame zero is the exact pointer selected by native's
/// first DATO offset. It never falls back to the original archives.
///
/// Go：`native_item_panel_render.go:75`，`func RenderNativeItemPanelBaseResources( assetPackRoot, portraitRoot string, portraitResource int, dst []byte, ) error`
pub fn render_native_item_panel_base_resources(
    asset_pack_root: &str,
    portrait_root: &str,
    portrait_resource: i64,
    dst: &mut [u8],
) -> Result<()> {
    todo!("battle/native_item_panel_render.go:RenderNativeItemPanelBaseResources")
}

/// RenderNativeItemPanelBaseResourcesArchive is retained only for fixed-source
/// oracle comparisons. Production callers use RenderNativeItemPanelBaseResources.
///
/// Go：`native_item_panel_render.go:114`，`func RenderNativeItemPanelBaseResourcesArchive( fdotherPath, portraitRoot string, portraitResource int, dst []byte, ) error`
pub fn render_native_item_panel_base_resources_archive(
    fdother_path: &str,
    portrait_root: &str,
    portrait_resource: i64,
    dst: &mut [u8],
) -> Result<()> {
    todo!("battle/native_item_panel_render.go:RenderNativeItemPanelBaseResourcesArchive")
}
