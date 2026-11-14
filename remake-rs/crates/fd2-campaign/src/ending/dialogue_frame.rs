//! Go 來源：`remake/internal/ending/dialogue_frame.go`（88 行）。

use super::*;

/// RenderDialogueFrameGrid executes only the recovered 0x168b6→0x4e9bb raw
/// cell copies into the caller's portrait restore buffer. It does not render
/// text, DATO, input, or any semantic UI layer.
///
/// Go：`dialogue_frame.go:12`，`func RenderDialogueFrameGrid(m Montage, cells []fdother.RawCell, dst []byte) error`
pub fn render_dialogue_frame_grid(m: &Montage, cells: &[fdother::RawCell], dst: &mut [u8]) -> Result<()> {
    if dst.len() as i64 != BYTES {
        return Err(campaign_err!("ending: dialogue frame destination must be {BYTES} bytes"));
    }
    let placements = m.plan_dialogue_frame_grid()?;
    for placement in &placements {
        if placement.resource_index < 0 || placement.resource_index >= cells.len() as i64 {
            return Err(campaign_err!(
                "ending: dialogue frame resource index {} is unavailable",
                placement.resource_index
            ));
        }
        cells[placement.resource_index as usize]
            .blit_opaque_at_offset(dst, m.party_cycle.dialogue_frame_layout.stride, placement.destination_byte)
            .map_err(|err| {
                CampaignError::from(err).wrap(format!(
                    "ending: dialogue frame resource {} at {}: ",
                    placement.resource_index,
                    battle::go_sharp_x(placement.destination_byte)
                ))
            })?;
    }
    Ok(())
}

/// RenderDialogueFrameGridResource loads the exact FDOTHER#5 cells referenced
/// by the recovered plan, then executes the raw compositor. Missing player
/// assets fail closed; no fallback UI is synthesized.
///
/// Go：`dialogue_frame.go:34`，`func RenderDialogueFrameGridResource(m Montage, datPath string, dst []byte) error`
pub fn render_dialogue_frame_grid_resource(m: &Montage, dat_path: &str, dst: &mut [u8]) -> Result<()> {
    let placements = m.plan_dialogue_frame_grid()?;
    let max_index = placements.iter().map(|p| p.resource_index).fold(0, i64::max);
    let mut cells = vec![fdother::RawCell::default(); (max_index + 1) as usize];
    let mut loaded = vec![false; (max_index + 1) as usize];
    for placement in &placements {
        let index = placement.resource_index as usize;
        if loaded[index] {
            continue;
        }
        let cell = fdother::decode_lmi1_raw_entry(
            dat_path,
            m.party_cycle.dialogue_frame_layout.resource,
            placement.resource_index,
        )
        .map_err(|err| CampaignError::from(err).wrap(format!("ending: FDOTHER#5 raw cell {index}: ")))?;
        cells[index] = cell;
        loaded[index] = true;
    }
    render_dialogue_frame_grid(m, &cells, dst)
}

/// RenderDialogueFrameGridSeparated loads the exact FDOTHER#5 raw cells from
/// the separated item-panel contract. It never falls back to FDOTHER.DAT.
///
/// Go：`dialogue_frame.go:64`，`func RenderDialogueFrameGridSeparated(m Montage, itemPanelRoot string, dst []byte) error`
pub fn render_dialogue_frame_grid_separated(m: &Montage, item_panel_root: &str, dst: &mut [u8]) -> Result<()> {
    let placements = m.plan_dialogue_frame_grid()?;
    let entries = fdother::load_separated_item_panel_entries(item_panel_root)?;
    let max_index = placements.iter().map(|p| p.resource_index).fold(0, i64::max);
    let mut cells = vec![fdother::RawCell::default(); (max_index + 1) as usize];
    for placement in &placements {
        let Some(cell) = entries.raw.get(&placement.resource_index) else {
            return Err(campaign_err!(
                "ending: separated FDOTHER#5 raw cell {} is unavailable",
                placement.resource_index
            ));
        };
        cells[placement.resource_index as usize] = cell.clone();
    }
    render_dialogue_frame_grid(m, &cells, dst)
}
