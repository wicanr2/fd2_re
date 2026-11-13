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
    if dst.len() as i64 != NATIVE_ITEM_PANEL_BYTES {
        return Err(battle_err!("battle: native item panel destination={}, want {NATIVE_ITEM_PANEL_BYTES}", dst.len()));
    }
    let layout = native_item_panel_base_layout_for();
    let placements = fdother::plan_native_dialogue_frame_grid(
        layout.stride,
        layout.grid_origin.x,
        layout.grid_origin.y,
        layout.grid_columns,
        layout.grid_rows,
    )?;
    if cells.len() <= 17 {
        return Err(battle_err!("battle: native item panel frame cells 1..17 are unavailable"));
    }

    let mut staged = dst.to_vec();
    for placement in &placements {
        let cell = &cells[placement.resource_index as usize];
        cell.blit_opaque_at_offset(&mut staged, layout.stride, placement.destination_byte).map_err(|e| {
            battle_err!(
                "battle: native item panel frame cell {} at {}: {e}",
                placement.resource_index,
                go_sharp_x(placement.destination_byte)
            )
        })?;
    }
    portrait
        .blit_at(&mut staged, layout.stride, layout.portrait_destination.x, layout.portrait_destination.y)
        .map_err(|e| battle_err!("battle: native item panel portrait: {e}"))?;
    upper
        .blit_opaque_at(&mut staged, layout.stride, layout.upper_destination.x, layout.upper_destination.y, false)
        .map_err(|e| battle_err!("battle: native item panel upper entry: {e}"))?;
    bottom
        .blit_opaque_at(&mut staged, layout.stride, layout.bottom_destination.x, layout.bottom_destination.y, false)
        .map_err(|e| battle_err!("battle: native item panel bottom entry: {e}"))?;
    dst.copy_from_slice(&staged);
    Ok(())
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
    let layout = native_item_panel_base_layout_for();
    let entries = fdother::load_separated_item_panel_entries(std::path::Path::new(asset_pack_root).join("ui"))
        .map_err(|e| battle_err!("battle: separated native item panel base: {e}"))?;
    let mut cells = vec![fdother::RawCell::default(); 18];
    for (index, cell) in cells.iter_mut().enumerate().skip(1) {
        *cell = entries
            .raw
            .get(&(index as i64))
            .cloned()
            .ok_or_else(|| battle_err!("battle: separated native item panel raw cell {index} is unavailable"))?;
    }
    let (Some(upper), Some(bottom)) =
        (entries.opaque.get(&layout.upper_directory_entry), entries.opaque.get(&layout.bottom_directory_entry))
    else {
        return Err(battle_err!("battle: separated native item panel large entries are unavailable"));
    };
    let portraits = dato::load_separated_resource(portrait_root, portrait_resource)
        .map_err(|e| battle_err!("battle: native item panel separated portrait {portrait_resource}: {e}"))?;
    let Some(portrait) = portraits.first() else {
        return Err(battle_err!("battle: native item panel DATO frame zero is unavailable"));
    };
    render_native_item_panel_base(&cells, portrait, upper, bottom, dst)
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
    let layout = native_item_panel_base_layout_for();
    let raw = fdother::read_resource(fdother_path, layout.fdother_resource as usize)
        .map_err(|e| battle_err!("battle: native item panel FDOTHER resource: {e}"))?;
    let mut cells = vec![fdother::RawCell::default(); 18];
    for (index, cell) in cells.iter_mut().enumerate().skip(1) {
        *cell = fdother::parse_lmi1_raw_entry(&raw, index as i64)
            .map_err(|e| battle_err!("battle: native item panel raw cell {index}: {e}"))?;
    }
    let entries = fdother::parse_lmi1(&raw).map_err(|e| battle_err!("battle: native item panel LMI1 entries: {e}"))?;
    if layout.upper_directory_entry >= entries.len() as i64 || layout.bottom_directory_entry >= entries.len() as i64 {
        return Err(battle_err!("battle: native item panel large entries are unavailable"));
    }
    let portraits = dato::load_separated_resource(portrait_root, portrait_resource)
        .map_err(|e| battle_err!("battle: native item panel separated portrait {portrait_resource}: {e}"))?;
    let Some(portrait) = portraits.first() else {
        return Err(battle_err!("battle: native item panel DATO frame zero is unavailable"));
    };
    render_native_item_panel_base(
        &cells,
        portrait,
        &entries[layout.upper_directory_entry as usize],
        &entries[layout.bottom_directory_entry as usize],
        dst,
    )
}

#[cfg(test)]
mod native_item_panel_render_test;
