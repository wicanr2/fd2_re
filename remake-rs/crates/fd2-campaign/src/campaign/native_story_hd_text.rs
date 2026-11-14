//! Go 來源：`remake/internal/campaign/native_story_hd_text.go`（77 行）。

use super::*;

/// NativeStoryVisibleGlyph describes presentation only. It uses the same typed
/// tokens, glyph clock and scroll passes as ComposeNativeStoryDialogueProgressiveFrames.
///
/// Go：`native_story_hd_text.go:7`
#[derive(Debug, Clone, Default, PartialEq, Eq)]
pub struct NativeStoryVisibleGlyph {
    pub token: String,
    pub x: i64,
    pub y: i64,
    /// -1 when the typed layout does not bind an original variant.
    pub glyph_id: i64,
}

/// Go：`native_story_hd_text.go:13`，`func NativeStoryDialogueVisibleGlyphs(layout *NativeDialogueLayout, page, progress int) ([]NativeStoryVisibleGlyph, error)`
pub fn native_story_dialogue_visible_glyphs(
    layout: &NativeDialogueLayout,
    page: i64,
    progress: i64,
) -> Result<Vec<NativeStoryVisibleGlyph>> {
    layout.validate()?;
    if page < 0 || page as u64 >= layout.pages.len() as u64 || progress < 0 {
        return Err(campaign_err!("campaign: invalid visible dialogue page or progress"));
    }
    let (x, y, _, step, rows) = native_story_dialogue_text_geometry(&layout.control)?;
    // Previous FFFD pages retain their rows. Their clocks have already elapsed.
    let mut budget = progress;
    for p in 0..page {
        let clock = native_story_dialogue_glyph_steps(layout, p)?;
        budget = budget.wrapping_add(clock.len() as i64 - 1);
    }
    let mut result: Vec<NativeStoryVisibleGlyph> = Vec::new();
    let mut row = 0i64;
    for p in 0..=page as usize {
        for (r, line) in layout.pages[p].iter().enumerate() {
            let mut visible_row = row;
            if row >= rows {
                let shifts = [3i64, 3, 3, 3, 3, 4];
                let mut pass = 0usize;
                for tick in 1..=NATIVE_STORY_SCROLL_FRAMES {
                    if budget == 0 {
                        return Ok(result);
                    }
                    budget -= 1;
                    while (pass as i64) < (shifts.len() as i64) * tick / NATIVE_STORY_SCROLL_FRAMES {
                        for glyph in &mut result {
                            glyph.y -= shifts[pass];
                        }
                        pass += 1;
                    }
                }
                visible_row = rows - 1;
            }
            let tokens = layout.glyph_tokens(p as i64, r as i64, line)?;
            for (column, token) in tokens.into_iter().enumerate() {
                if budget == 0 {
                    return Ok(result);
                }
                budget -= 1;
                let mut id = -1;
                if !layout.glyph_ids.is_empty() {
                    id = layout.glyph_ids[p][r][column];
                }
                result.push(NativeStoryVisibleGlyph {
                    token,
                    x: x + column as i64 * NATIVE_STORY_GLYPH_STEP,
                    y: y + visible_row * step,
                    glyph_id: id,
                });
            }
            row += 1;
        }
    }
    if budget != 0 {
        return Err(campaign_err!("campaign: dialogue progress exceeds glyph clock"));
    }
    Ok(result)
}
