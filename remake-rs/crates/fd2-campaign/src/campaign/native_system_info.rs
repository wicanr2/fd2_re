//! Go 來源：`remake/internal/campaign/native_system_info.go`（129 行）。

use super::*;

/// Go：`native_system_info.go:12`
#[derive(Clone, Default)]
pub struct NativeSystemInfoAssets {
    pub panels: [fdother::Frame; 4],
    pub numbers: battle::NativeItemPanelDataAssets,
    pub font: Option<fdtxt::Font>,
}

/// Go：`native_system_info.go:18`
#[derive(Debug, Clone, Default, PartialEq, Eq)]
pub struct NativeSystemInfoInput {
    pub raw_chapter: u8,
    pub raw_round: u8,
    pub currency: u32,
    /// RawCampCounts is indexed by the original record+6 selector. sub_1B41D
    /// displays selectors 0, 2, 1 from left to right.
    pub raw_camp_counts: [i64; 3],
    pub lines: [Vec<u16>; 2],
}

/// NativeSystemInfoCampCounts reproduces sub_1B41D's three calls to its
/// record-array counter. Every record field used by the native predicate must
/// retain explicit provenance; otherwise the operation fails closed.
///
/// Go：`native_system_info.go:31`，`func NativeSystemInfoCampCounts(units []*battle.Unit) ([3]int, error)`
pub fn native_system_info_camp_counts(units: &[Unit]) -> Result<[i64; 3]> {
    todo!("campaign/native_system_info.go:NativeSystemInfoCampCounts")
}

/// Go：`native_system_info.go:49`，`func blitNativeSystemInfoLine(dst []byte, font *fdtxt.Font, words []uint16, y int) error`
pub(crate) fn blit_native_system_info_line(
    dst: &mut [u8],
    font: Option<&fdtxt::Font>,
    words: &[u16],
    y: i64,
) -> Result<()> {
    todo!("campaign/native_system_info.go:blitNativeSystemInfoLine")
}

/// ComposeNativeSystemInfoSurface reproduces sub_1B41D's four FDOTHER #5
/// panels, six 0x187D6 decimal fields and two caller-selected FDTXT lines.
/// RawChapter remains zero-based, matching [0x53C03]; RawRound and Currency
/// are passed without normalized campaign reinterpretation.
///
/// Go：`native_system_info.go:84`，`func ComposeNativeSystemInfoSurface(assets NativeSystemInfoAssets, input NativeSystemInfoInput) ([]byte, error)`
pub fn compose_native_system_info_surface(
    assets: &NativeSystemInfoAssets,
    input: &NativeSystemInfoInput,
) -> Result<Vec<u8>> {
    todo!("campaign/native_system_info.go:ComposeNativeSystemInfoSurface")
}
