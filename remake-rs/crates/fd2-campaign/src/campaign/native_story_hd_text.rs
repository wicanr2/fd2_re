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
    todo!("campaign/native_story_hd_text.go:NativeStoryDialogueVisibleGlyphs")
}
