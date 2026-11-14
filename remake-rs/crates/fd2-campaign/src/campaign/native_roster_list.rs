//! Go 來源：`remake/internal/campaign/native_roster_list.go`（164 行）。

use super::*;

pub const NATIVE_ROSTER_VISIBLE: i64 = 6;

/// NativeRosterRow is one caller-resolved party entry consumed by 0x2ea90.
/// The native FDICON pointer and FDTXT name index are kept explicit so missing
/// provenance fails closed instead of guessing from a remake roster key.
///
/// Go：`native_roster_list.go:17`
#[derive(Debug, Clone, Default, PartialEq, Eq)]
pub struct NativeRosterRow {
    pub sprite: fdicon::Sprite,
    pub name_text_index: i64,
}

/// NativeTwoColumnWindow mirrors 0x2e6b8's stateful [0x5412f] viewport. The
/// caller preserves start between moves. It changes by two only after the
/// selection crosses the current six-entry window, so moving back within the
/// window does not prematurely reset the scroll origin.
///
/// Go：`native_roster_list.go:26`，`func NativeTwoColumnWindow(count, selected, start int) (nextStart, visible int)`
/// Go 具名回傳值依序為：nextStart、visible。
pub fn native_two_column_window(count: i64, selected: i64, start: i64) -> (i64, i64) {
    if count <= 0 || selected < 0 || selected >= count || start < 0 || start % 2 != 0 {
        return (0, 0);
    }
    let mut max_start = (count - NATIVE_ROSTER_VISIBLE).max(0);
    if max_start % 2 != 0 {
        max_start += 1;
    }
    let mut start = start.min(max_start);
    while selected < start && start >= 2 {
        start -= 2;
    }
    while selected >= start + NATIVE_ROSTER_VISIBLE && start < max_start {
        start += 2;
    }
    let visible = (count - start).min(NATIVE_ROSTER_VISIBLE);
    (start, visible)
}

/// RenderNativeRosterRows reproduces 0x2ea90's two-column, six-entry draw.
///
/// Go：`native_roster_list.go:54`，`func RenderNativeRosterRows( dst []byte, rows []NativeRosterRow, selected int, strings *fdtxt.Strings, font *fdtxt.Font, ) error`
pub fn render_native_roster_rows(
    dst: &mut [u8],
    rows: &[NativeRosterRow],
    selected: i64,
    strings: Option<&fdtxt::Strings>,
    font: Option<&fdtxt::Font>,
) -> Result<()> {
    render_native_roster_rows_inner(dst, rows, selected, strings, font, true)
}

/// RenderNativeRosterRowsWithoutNames 保留 0x2ea90 的人物圖示與選取狀態，
/// 將名稱矩形留給多語 renderer；它不改變名冊身分、順序或 viewport。
///
/// Go：`native_roster_list.go:66`，`func RenderNativeRosterRowsWithoutNames( dst []byte, rows []NativeRosterRow, selected int, ) error`
pub fn render_native_roster_rows_without_names(dst: &mut [u8], rows: &[NativeRosterRow], selected: i64) -> Result<()> {
    render_native_roster_rows_inner(dst, rows, selected, None, None, false)
}

/// Go：`native_roster_list.go:74`，`func renderNativeRosterRows( dst []byte, rows []NativeRosterRow, selected int, strings *fdtxt.Strings, font *fdtxt.Font, renderNames bool, ) error`
pub(crate) fn render_native_roster_rows_inner(
    dst: &mut [u8],
    rows: &[NativeRosterRow],
    selected: i64,
    strings: Option<&fdtxt::Strings>,
    font: Option<&fdtxt::Font>,
    render_names: bool,
) -> Result<()> {
    if dst.len() as i64 != NATIVE_CLASS_LIST_STRIDE * NATIVE_CLASS_LIST_HEIGHT {
        return Err(campaign_err!("campaign: native roster requires a 320x200 indexed frame"));
    }
    if rows.is_empty() || rows.len() as i64 > NATIVE_ROSTER_VISIBLE || selected < 0 || selected >= rows.len() as i64 {
        return Err(campaign_err!("campaign: invalid native roster rows or selection"));
    }
    let text = match (strings, font) {
        (Some(s), Some(f)) => Some((s, f)),
        _ if render_names => return Err(campaign_err!("campaign: native roster text assets are unavailable")),
        _ => None,
    };
    let mut staged = dst.to_vec();
    for (i, row) in rows.iter().enumerate() {
        let i = i as i64;
        let (column, line) = (i % 2, i / 2);
        row.sprite
            .blit_at(&mut staged, NATIVE_CLASS_LIST_STRIDE, 14 + 132 * column, 117 + 26 * line)
            .map_err(|e| CampaignError::from(e).wrap(format!("campaign: native roster row {i} sprite: ")))?;
        let foreground: u8 = if i == selected { 201 } else { 205 };
        if render_names && let Some((strings, font)) = text {
            blit_native_class_list_text(
                &mut staged,
                strings,
                font,
                40 + 132 * column,
                121 + 26 * line,
                row.name_text_index,
                foreground,
            )
            .map_err(|e| e.wrap(format!("campaign: native roster row {i} name: ")))?;
        }
    }
    dst.copy_from_slice(&staged);
    Ok(())
}

/// ComposeNativeRosterFrame applies FDOTHER#14 entry16 and then 0x2ea90.
///
/// Go：`native_roster_list.go:118`，`func ComposeNativeRosterFrame( background []byte, panel fdother.LMI1Entry, rows []NativeRosterRow, selected int, strings *fdtxt.Strings, font *fdtxt.Font, ) ([]byte, error)`
pub fn compose_native_roster_frame(
    background: &[u8],
    panel: &fdother::LMI1Entry,
    rows: &[NativeRosterRow],
    selected: i64,
    strings: Option<&fdtxt::Strings>,
    font: Option<&fdtxt::Font>,
) -> Result<Vec<u8>> {
    let mut frame = native_roster_panel_frame(background, panel)?;
    render_native_roster_rows(&mut frame, rows, selected, strings, font)?;
    Ok(frame)
}

/// Go：`native_roster_list.go:144`，`func ComposeNativeRosterFrameWithoutNames( background []byte, panel fdother.LMI1Entry, rows []NativeRosterRow, selected int, ) ([]byte, error)`
pub fn compose_native_roster_frame_without_names(
    background: &[u8],
    panel: &fdother::LMI1Entry,
    rows: &[NativeRosterRow],
    selected: i64,
) -> Result<Vec<u8>> {
    let mut frame = native_roster_panel_frame(background, panel)?;
    render_native_roster_rows_without_names(&mut frame, rows, selected)?;
    Ok(frame)
}

/// `ComposeNativeRosterFrame` 與 `ComposeNativeRosterFrameWithoutNames` 共用的前半段
/// （`native_roster_list.go:121`、`:146` 各寫一份）：檢查背景與面板尺寸，複製背景後貼 FDOTHER#14
/// entry16。
fn native_roster_panel_frame(background: &[u8], panel: &fdother::LMI1Entry) -> Result<Vec<u8>> {
    if background.len() as i64 != NATIVE_CLASS_LIST_STRIDE * NATIVE_CLASS_LIST_HEIGHT {
        return Err(campaign_err!("campaign: native roster background requires a 320x200 indexed frame"));
    }
    if panel.width != NATIVE_CLASS_PANEL_W || panel.height != NATIVE_CLASS_PANEL_H {
        return Err(campaign_err!("campaign: native roster panel is {}x{}, want 310x86", panel.width, panel.height));
    }
    let mut frame = background.to_vec();
    panel
        .blit_opaque_at(&mut frame, NATIVE_CLASS_LIST_STRIDE, NATIVE_CLASS_PANEL_X, NATIVE_CLASS_PANEL_Y, false)
        .map_err(|e| CampaignError::from(e).wrap("campaign: native roster panel: "))?;
    Ok(frame)
}

#[cfg(test)]
pub(crate) mod native_roster_list_test;
