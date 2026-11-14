//! Go 來源：`remake/internal/ending/dialogue_frame.go`（88 行）。

use super::*;

/// RenderDialogueFrameGrid executes only the recovered 0x168b6→0x4e9bb raw
/// cell copies into the caller's portrait restore buffer. It does not render
/// text, DATO, input, or any semantic UI layer.
///
/// Go：`dialogue_frame.go:12`，`func RenderDialogueFrameGrid(m Montage, cells []fdother.RawCell, dst []byte) error`
pub fn render_dialogue_frame_grid(m: &Montage, cells: &[fdother::RawCell], dst: &mut [u8]) -> Result<()> {
    todo!("ending/dialogue_frame.go:RenderDialogueFrameGrid")
}

/// RenderDialogueFrameGridResource loads the exact FDOTHER#5 cells referenced
/// by the recovered plan, then executes the raw compositor. Missing player
/// assets fail closed; no fallback UI is synthesized.
///
/// Go：`dialogue_frame.go:34`，`func RenderDialogueFrameGridResource(m Montage, datPath string, dst []byte) error`
pub fn render_dialogue_frame_grid_resource(m: &Montage, dat_path: &str, dst: &mut [u8]) -> Result<()> {
    todo!("ending/dialogue_frame.go:RenderDialogueFrameGridResource")
}

/// RenderDialogueFrameGridSeparated loads the exact FDOTHER#5 raw cells from
/// the separated item-panel contract. It never falls back to FDOTHER.DAT.
///
/// Go：`dialogue_frame.go:64`，`func RenderDialogueFrameGridSeparated(m Montage, itemPanelRoot string, dst []byte) error`
pub fn render_dialogue_frame_grid_separated(m: &Montage, item_panel_root: &str, dst: &mut [u8]) -> Result<()> {
    todo!("ending/dialogue_frame.go:RenderDialogueFrameGridSeparated")
}
