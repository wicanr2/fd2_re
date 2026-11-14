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
    todo!("campaign/native_roster_list.go:NativeTwoColumnWindow")
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
    todo!("campaign/native_roster_list.go:renderNativeRosterRows")
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
    todo!("campaign/native_roster_list.go:ComposeNativeRosterFrame")
}

/// Go：`native_roster_list.go:144`，`func ComposeNativeRosterFrameWithoutNames( background []byte, panel fdother.LMI1Entry, rows []NativeRosterRow, selected int, ) ([]byte, error)`
pub fn compose_native_roster_frame_without_names(
    background: &[u8],
    panel: &fdother::LMI1Entry,
    rows: &[NativeRosterRow],
    selected: i64,
) -> Result<Vec<u8>> {
    todo!("campaign/native_roster_list.go:ComposeNativeRosterFrameWithoutNames")
}
