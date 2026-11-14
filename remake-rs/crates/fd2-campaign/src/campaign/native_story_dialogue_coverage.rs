//! Go 來源：`remake/internal/campaign/native_story_dialogue_coverage.go`（96 行）。

use super::*;

/// NativeStoryDialogueFrameBounds derives presentation bounds from the same
/// opaque writes as the native frame. The restored background and cursor tail
/// carry no dialogue border.
///
/// Go：`native_story_dialogue_coverage.go:14`，`func NativeStoryDialogueFrameBounds(cells []fdother.RawCell, control string, phase NativeDialogueFramePhase, index int) (image.Rectangle, error)`
pub fn native_story_dialogue_frame_bounds(
    cells: &[fdother::RawCell],
    control: &str,
    phase: NativeDialogueFramePhase,
    index: i64,
) -> Result<go_image::Rectangle> {
    if phase == NATIVE_DIALOGUE_CLOSING && index >= NATIVE_STORY_CLOSING_GRID_SIZES.len() as i64 {
        // Validate the source/control through the existing blank-frame contract.
        native_story_dialogue_coverage(cells, control, phase, NATIVE_STORY_CLOSING_GRID_SIZES.len() as i64)?;
        return Ok(go_image::Rectangle::default());
    }
    let mask = native_story_dialogue_coverage(cells, control, phase, index)?;
    let mut bounds = go_image::Rectangle::default();
    for (i, &painted) in mask.iter().enumerate() {
        if painted {
            let (x, y) = ((i % 320) as i64, (i / 320) as i64);
            bounds = go_rect_union(bounds, go_rect(x, y, x + 1, y + 1));
        }
    }
    Ok(bounds)
}

/// Go：`native_story_dialogue_coverage.go:33`
#[derive(Debug, Clone, Copy, Default, PartialEq, Eq, PartialOrd, Ord, Hash)]
pub struct NativeDialogueFramePhase(pub u8);

/// Go 原式：`iota`
pub const NATIVE_DIALOGUE_OPENING: NativeDialogueFramePhase = NativeDialogueFramePhase(0);

pub const NATIVE_DIALOGUE_STEADY: NativeDialogueFramePhase = NativeDialogueFramePhase(1);

pub const NATIVE_DIALOGUE_CLOSING: NativeDialogueFramePhase = NativeDialogueFramePhase(2);

/// NativeStoryDialogueCoverage follows the actual opaque raw-cell writes.
/// Closing covers the five snapshot restores, without the optional cursor tail.
///
/// Go：`native_story_dialogue_coverage.go:43`，`func NativeStoryDialogueCoverage(cells []fdother.RawCell, control string, phase NativeDialogueFramePhase, index int) ([]bool, error)`
pub fn native_story_dialogue_coverage(
    cells: &[fdother::RawCell],
    control: &str,
    phase: NativeDialogueFramePhase,
    index: i64,
) -> Result<Vec<bool>> {
    if cells.len() < 18 {
        return Err(campaign_err!("campaign: dialogue coverage cells unavailable"));
    }
    let frame_y = match control {
        "FFEF" | "FFED" => NATIVE_STORY_UPPER_FRAME_Y,
        "FFEE" | "FFEC" => NATIVE_STORY_LOWER_FRAME_Y,
        _ => return Err(campaign_err!("campaign: dialogue coverage control unavailable")),
    };
    let (mut columns, mut rows) = (19, 5);
    match phase {
        NATIVE_DIALOGUE_OPENING => {
            if index < 0 || index >= NATIVE_STORY_OPENING_GRID_SIZES.len() as i64 {
                return Err(campaign_err!("campaign: dialogue opening coverage index unavailable"));
            }
            [columns, rows] = NATIVE_STORY_OPENING_GRID_SIZES[index as usize];
        }
        NATIVE_DIALOGUE_CLOSING => {
            if index < 0 || index > NATIVE_STORY_CLOSING_GRID_SIZES.len() as i64 {
                return Err(campaign_err!("campaign: dialogue closing coverage index unavailable"));
            }
            if index == NATIVE_STORY_CLOSING_GRID_SIZES.len() as i64 {
                return Ok(vec![false; 320 * 200]);
            }
            [columns, rows] = NATIVE_STORY_CLOSING_GRID_SIZES[index as usize];
        }
        NATIVE_DIALOGUE_STEADY => {
            if index != 0 {
                return Err(campaign_err!("campaign: dialogue steady coverage index unavailable"));
            }
        }
        _ => return Err(campaign_err!("campaign: dialogue coverage phase unavailable")),
    }
    let placements = fdother::plan_native_dialogue_frame_grid(320, 5, frame_y, columns, rows)?;
    let mut mask = vec![false; 320 * 200];
    for placement in &placements {
        let cell = &cells[placement.resource_index as usize];
        let (x, y) = (placement.destination_byte % 320, placement.destination_byte / 320);
        if cell.width <= 0
            || cell.height <= 0
            || cell.pixels.len() as i64 != cell.width.wrapping_mul(cell.height)
            || x < 0
            || y < 0
            || x + cell.width > 320
            || y + cell.height > 200
        {
            return Err(campaign_err!("campaign: dialogue coverage cell {} exceeds source", placement.resource_index));
        }
        for row in 0..cell.height {
            for column in 0..cell.width {
                mask[((y + row) * 320 + x + column) as usize] = true;
            }
        }
    }
    Ok(mask)
}
