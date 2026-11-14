//! Go 來源：`remake/internal/campaign/native_load_slots.go`（143 行）。

use super::*;

pub const NATIVE_LOAD_SLOT_WIDTH: i64 = 320;

pub const NATIVE_LOAD_SLOT_HEIGHT: i64 = 200;

pub(crate) const NATIVE_LOAD_FRAME_X: i64 = 5;

pub(crate) const NATIVE_LOAD_FRAME_Y: i64 = 112;

pub(crate) const NATIVE_LOAD_TEXT_Y: i64 = 119;

pub(crate) const NATIVE_LOAD_ROW_STRIDE: i64 = 19;

pub(crate) const NATIVE_LOAD_NUMBER_X: i64 = 10;

pub(crate) const NATIVE_LOAD_EMPTY_X: i64 = 88;

pub(crate) const NATIVE_LOAD_CHAPTER_X: i64 = 40;

pub(crate) const NATIVE_LOAD_TITLE_X: i64 = 130;

/// NativeLoadSlot preserves only metadata consumed by the proven 0x30437
/// renderer. Chapter is the raw saved byte; it is not normalized to a remake
/// campaign node.
///
/// Go：`native_load_slots.go:29`
#[derive(Debug, Clone, Copy, Default, PartialEq, Eq)]
pub struct NativeLoadSlot {
    pub empty: bool,
    pub chapter: u8,
}

/// ComposeNativeLoadSlotsFrame reproduces the stable 0x30437 frame used by
/// 0x30550. It deliberately accepts a caller-owned source snapshot because
/// 0x30550 copies the current VGA page before opening FDOTHER#13 LMI entry16.
///
/// Go：`native_load_slots.go:37`，`func ComposeNativeLoadSlotsFrame( source []byte, dialogueBox fdother.LMI1Entry, strings *fdtxt.Strings, font *fdtxt.Font, slots [4]NativeLoadSlot, selection int, ) ([]byte, error)`
pub fn compose_native_load_slots_frame(
    source: &[u8],
    dialogue_box: &fdother::LMI1Entry,
    strings: Option<&fdtxt::Strings>,
    font: Option<&fdtxt::Font>,
    slots: [NativeLoadSlot; 4],
    selection: i64,
) -> Result<Vec<u8>> {
    todo!("campaign/native_load_slots.go:ComposeNativeLoadSlotsFrame")
}

/// Go：`native_load_slots.go:106`，`func renderNativeLoadText( frame []byte, strings *fdtxt.Strings, font *fdtxt.Font, textIndex, x, y int, style fdtxt.NativeGlyphStyle, number int, ) error`
pub(crate) fn render_native_load_text(
    frame: &mut [u8],
    strings: &fdtxt::Strings,
    font: &fdtxt::Font,
    text_index: i64,
    x: i64,
    y: i64,
    style: fdtxt::NativeGlyphStyle,
    number: i64,
) -> Result<()> {
    todo!("campaign/native_load_slots.go:renderNativeLoadText")
}
