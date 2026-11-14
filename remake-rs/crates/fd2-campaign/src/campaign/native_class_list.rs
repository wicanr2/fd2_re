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
    if dst.len() as i64 != NATIVE_CLASS_LIST_STRIDE * NATIVE_CLASS_LIST_HEIGHT {
        return Err(campaign_err!("campaign: native class list requires a 320x200 indexed frame"));
    }
    if rows.is_empty() || rows.len() as i64 > NATIVE_CLASS_LIST_ROWS || selected < 0 || selected >= rows.len() as i64 {
        return Err(campaign_err!("campaign: invalid native class list rows or selection"));
    }
    let (Some(strings), Some(font)) = (strings, font) else {
        return Err(campaign_err!("campaign: native class list text assets are unavailable"));
    };
    let mut staged = dst.to_vec();
    for (i, row) in rows.iter().enumerate() {
        let i = i as i64;
        row.sprite
            .blit_at(&mut staged, NATIVE_CLASS_LIST_STRIDE, 14, 117 + 26 * i)
            .map_err(|e| CampaignError::from(e).wrap(format!("campaign: native class row {i} sprite: ")))?;
        let foreground: u8 = if i == selected { 201 } else { 205 };
        let y = 121 + 26 * i;
        let texts = [
            (40, row.name_text_index),
            (130, row.current_class_text_id.wrapping_add(150)),
            (175, 593),
            (239, row.target_class_text_id.wrapping_add(150)),
        ];
        for (x, index) in texts {
            blit_native_class_list_text(&mut staged, strings, font, x, y, index, foreground)
                .map_err(|e| e.wrap(format!("campaign: native class row {i} text {index}: ")))?;
        }
    }
    dst.copy_from_slice(&staged);
    Ok(())
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
    if background.len() as i64 != NATIVE_CLASS_LIST_STRIDE * NATIVE_CLASS_LIST_HEIGHT {
        return Err(campaign_err!("campaign: native class background requires a 320x200 indexed frame"));
    }
    if panel.width != NATIVE_CLASS_PANEL_W || panel.height != NATIVE_CLASS_PANEL_H {
        return Err(campaign_err!("campaign: native class panel is {}x{}, want 310x86", panel.width, panel.height));
    }
    let mut frame = background.to_vec();
    panel
        .blit_opaque_at(&mut frame, NATIVE_CLASS_LIST_STRIDE, NATIVE_CLASS_PANEL_X, NATIVE_CLASS_PANEL_Y, false)
        .map_err(|e| CampaignError::from(e).wrap("campaign: native class panel: "))?;
    render_native_class_list_rows(&mut frame, rows, selected, strings, font)?;
    Ok(frame)
}

/// NativeClassListOpeningFrames reproduces 0x311DC's six calls to 0x1974c.
/// Each pass starts from the untouched snapshot and copies an increasingly
/// complete 310-pixel-wide slice from composed y=112 to destination y.
///
/// Go：`native_class_list.go:110`，`func NativeClassListOpeningFrames(background, composed []byte) ([][]byte, error)`
pub fn native_class_list_opening_frames(background: &[u8], composed: &[u8]) -> Result<Vec<Vec<u8>>> {
    if background.len() as i64 != NATIVE_CLASS_LIST_STRIDE * NATIVE_CLASS_LIST_HEIGHT
        || composed.len() as i64 != NATIVE_CLASS_LIST_STRIDE * NATIVE_CLASS_LIST_HEIGHT
    {
        return Err(campaign_err!("campaign: native class opening requires two 320x200 indexed frames"));
    }
    Ok((0..=5).rev().map(|i| native_class_list_slice_frame(background, composed, i)).collect())
}

/// NativeClassListClosingFrames reproduces 0x2d31b's five 0x1974c passes
/// (i=1..5). The caller must present the untouched source restore afterward.
///
/// Go：`native_class_list.go:135`，`func NativeClassListClosingFrames(background, composed []byte) ([][]byte, error)`
pub fn native_class_list_closing_frames(background: &[u8], composed: &[u8]) -> Result<Vec<Vec<u8>>> {
    if background.len() as i64 != NATIVE_CLASS_LIST_STRIDE * NATIVE_CLASS_LIST_HEIGHT
        || composed.len() as i64 != NATIVE_CLASS_LIST_STRIDE * NATIVE_CLASS_LIST_HEIGHT
    {
        return Err(campaign_err!("campaign: native class closing requires two 320x200 indexed frames"));
    }
    Ok((1..6).map(|i| native_class_list_slice_frame(background, composed, i)).collect())
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
    let words = strings.words(index)?;
    let style = fdtxt::NativeGlyphStyle { foreground, shadow: 76, background: 0 };
    for (i, &word) in words.iter().enumerate() {
        if word >= fdtxt::CONTROL_MIN {
            return Err(campaign_err!("text contains native control {}", battle::go_sharp_x(word as i64)));
        }
        let offset = y * NATIVE_CLASS_LIST_STRIDE + x + i as i64 * fdtxt::GLYPH_WIDTH as i64;
        font.blit_native_glyph(dst, NATIVE_CLASS_LIST_STRIDE, offset, word as i64, style)?;
    }
    Ok(())
}

/// `0x1974c` 的一次複製：從未改動的背景開始，把合成畫面 y=112 起的 310 像素寬切片貼到
/// `13*i + 112`。Go 的開啟與收合兩個函式各寫一份同樣的迴圈（`native_class_list.go:118`、`:143`），
/// Rust 共用這個輔助函式。
fn native_class_list_slice_frame(background: &[u8], composed: &[u8], i: i64) -> Vec<u8> {
    let destination_y = 13 * i + NATIVE_CLASS_PANEL_Y;
    let height = NATIVE_CLASS_PANEL_H.min(NATIVE_CLASS_LIST_HEIGHT - destination_y);
    let mut frame = background.to_vec();
    let w = NATIVE_CLASS_PANEL_W as usize;
    for row in 0..height {
        let src = ((NATIVE_CLASS_PANEL_Y + row) * NATIVE_CLASS_LIST_STRIDE + NATIVE_CLASS_PANEL_X) as usize;
        let dst = ((destination_y + row) * NATIVE_CLASS_LIST_STRIDE + NATIVE_CLASS_PANEL_X) as usize;
        frame[dst..dst + w].copy_from_slice(&composed[src..src + w]);
    }
    frame
}

#[cfg(test)]
pub(crate) mod native_class_list_test;
