//! Go 來源：`remake/internal/campaign/native_class_list.go`（180 行）。

use super::*;

pub const NATIVE_CLASS_LIST_STRIDE: i64 = 320;

pub const NATIVE_CLASS_LIST_HEIGHT: i64 = 200;

pub const NATIVE_CLASS_LIST_ROWS: i64 = 3;

pub(crate) const NATIVE_CLASS_PANEL_X: i64 = 5;

pub(crate) const NATIVE_CLASS_PANEL_Y: i64 = 112;

pub(crate) const NATIVE_CLASS_PANEL_W: i64 = 310;

pub(crate) const NATIVE_CLASS_PANEL_H: i64 = 86;

/// NativeClassListRow is one already-resolved row consumed by 0x31019.
/// Sprite is the party FDICON pointer selected by the native roster slot and
/// global cycle; resolving that pointer remains the caller's responsibility.
///
/// Go：`native_class_list.go:25`
#[derive(Debug, Clone, Default, PartialEq, Eq)]
pub struct NativeClassListRow {
    pub sprite: fdicon::Sprite,
    pub name_text_index: i64,
    pub current_class_text_id: i64,
    pub target_class_text_id: i64,
}

/// RenderNativeClassListRows reproduces the proven row subpass of 0x31019.
/// It deliberately does not invent the surrounding 0x1974c opening frame.
/// The destination is updated atomically only after all sprites and strings
/// validate.
///
/// Go：`native_class_list.go:36`，`func RenderNativeClassListRows( dst []byte, rows []NativeClassListRow, selected int, strings *fdtxt.Strings, font *fdtxt.Font, ) error`
pub fn render_native_class_list_rows(
    dst: &mut [u8],
    rows: &[NativeClassListRow],
    selected: i64,
    strings: Option<&fdtxt::Strings>,
    font: Option<&fdtxt::Font>,
) -> Result<()> {
    todo!("campaign/native_class_list.go:RenderNativeClassListRows")
}

/// ComposeNativeClassListFrame first applies FDOTHER #14 LMI1 entry #16 using
/// 0x4e8af's opaque path, then runs the 0x31019 row subpass.
///
/// Go：`native_class_list.go:83`，`func ComposeNativeClassListFrame( background []byte, panel fdother.LMI1Entry, rows []NativeClassListRow, selected int, strings *fdtxt.Strings, font *fdtxt.Font, ) ([]byte, error)`
pub fn compose_native_class_list_frame(
    background: &[u8],
    panel: &fdother::LMI1Entry,
    rows: &[NativeClassListRow],
    selected: i64,
    strings: Option<&fdtxt::Strings>,
    font: Option<&fdtxt::Font>,
) -> Result<Vec<u8>> {
    todo!("campaign/native_class_list.go:ComposeNativeClassListFrame")
}

/// NativeClassListOpeningFrames reproduces 0x311DC's six calls to 0x1974c.
/// Each pass starts from the untouched snapshot and copies an increasingly
/// complete 310-pixel-wide slice from composed y=112 to destination y.
///
/// Go：`native_class_list.go:110`，`func NativeClassListOpeningFrames(background, composed []byte) ([][]byte, error)`
pub fn native_class_list_opening_frames(background: &[u8], composed: &[u8]) -> Result<Vec<Vec<u8>>> {
    todo!("campaign/native_class_list.go:NativeClassListOpeningFrames")
}

/// NativeClassListClosingFrames reproduces 0x2d31b's five 0x1974c passes
/// (i=1..5). The caller must present the untouched source restore afterward.
///
/// Go：`native_class_list.go:135`，`func NativeClassListClosingFrames(background, composed []byte) ([][]byte, error)`
pub fn native_class_list_closing_frames(background: &[u8], composed: &[u8]) -> Result<Vec<Vec<u8>>> {
    todo!("campaign/native_class_list.go:NativeClassListClosingFrames")
}

/// Go：`native_class_list.go:158`，`func blitNativeClassListText( dst []byte, strings *fdtxt.Strings, font *fdtxt.Font, x, y, index int, foreground byte, ) error`
pub(crate) fn blit_native_class_list_text(
    dst: &mut [u8],
    strings: &fdtxt::Strings,
    font: &fdtxt::Font,
    x: i64,
    y: i64,
    index: i64,
    foreground: u8,
) -> Result<()> {
    todo!("campaign/native_class_list.go:blitNativeClassListText")
}
