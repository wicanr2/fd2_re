//! Go 來源：`remake/internal/campaign/native_town_scene.go`（263 行）。

use super::*;

pub const NATIVE_TOWN_WIDTH: i64 = 320;

pub const NATIVE_TOWN_HEIGHT: i64 = 200;

pub(crate) const NATIVE_TOWN_LABEL_X: i64 = 244;

pub(crate) const NATIVE_TOWN_LABEL_Y: i64 = 162;

/// Go 原式：`168*NativeTownWidth + 252`
pub(crate) const NATIVE_TOWN_TEXT_OFFSET: i64 = 54012;

pub(crate) const NATIVE_TOWN_VIEWPORT_X: i64 = 4;

pub(crate) const NATIVE_TOWN_VIEWPORT_Y: i64 = 4;

pub(crate) const NATIVE_TOWN_VIEWPORT_W: i64 = 312;

pub(crate) const NATIVE_TOWN_VIEWPORT_H: i64 = 192;

pub(crate) const NATIVE_TOWN_TEXT_BASE: i64 = 0x1ef;

pub(crate) const NATIVE_TOWN_VARIANT_COUNT: i64 = 3;

/// Go：`native_town_scene.go:30`
pub(crate) static NATIVE_TOWN_BACKGROUND_RESOURCES: [i64; 3] = [11, 61, 62];

/// Go：`native_town_scene.go:32`
pub(crate) static NATIVE_TOWN_SELECTION_X: [[i64; 6]; 3] =
    [[29, 41, 59, 154, 182, 10], [90, 33, 53, 148, 222, 196], [59, 10, 59, 130, 242, 136]];

/// Go：`native_town_scene.go:38`
pub(crate) static NATIVE_TOWN_SELECTION_Y: [[i64; 6]; 3] =
    [[46, 109, 163, 139, 65, 10], [30, 105, 163, 139, 85, 8], [26, 144, 163, 150, 31, 20]];

/// NativeTownCursorGeometry keeps the original scene-to-VGA viewport offset.
///
/// Go：`native_town_scene.go:45`
#[derive(Debug, Clone, Copy, Default, PartialEq, Eq)]
pub struct NativeTownCursorGeometry {
    pub position: go_image::Point,
    pub frame: i64,
    pub viewport: go_image::Rectangle,
}

/// Go：`native_town_scene.go:51`，`func NativeTownCursorGeometryFor(variant, selection, pulse int) (NativeTownCursorGeometry, error)`
pub fn native_town_cursor_geometry_for(variant: i64, selection: i64, pulse: i64) -> Result<NativeTownCursorGeometry> {
    todo!("campaign/native_town_scene.go:NativeTownCursorGeometryFor")
}

/// NativeTownAssets are the resources consumed by 0x2cd46..0x2d05a: three
/// full-screen backgrounds, FDOTHER#10's opaque current-label panel, and three
/// cursor cycles. Pulse initially carries FDICON group 0 for provenance-free
/// editor fixtures; the production owner replaces it from the first persistent
/// party raw selector key, matching dword_53A61's first cache slot.
///
/// Go：`native_town_scene.go:71`
#[derive(Debug, Clone, Default, PartialEq, Eq)]
pub struct NativeTownAssets {
    pub backgrounds: [Vec<u8>; 3],
    pub label: fdother::LMI1Entry,
    pub pulse: [fdicon::Sprite; 3],
}

/// Go：`native_town_scene.go:77`，`func DecodeNativeTownAssetsArchive( fdotherPath, fdiconPath string, ) (*NativeTownAssets, error)`
pub fn decode_native_town_assets_archive(fdother_path: &str, fdicon_path: &str) -> Result<NativeTownAssets> {
    todo!("campaign/native_town_scene.go:DecodeNativeTownAssetsArchive")
}

/// Go：`native_town_scene.go:90`
#[derive(Debug, Clone, Default, PartialEq, Eq)]
pub(crate) struct SeparatedTownLabelDocument {
    /// Go 標籤：`json:"schema_version"`
    pub schema_version: i64,
    /// Go 標籤：`json:"kind"`
    pub kind: String,
    /// Go 標籤：`json:"asset_id"`
    pub asset_id: String,
    /// Go 標籤：`json:"status"`
    pub status: String,
    /// Go 標籤：`json:"evidence"`
    pub evidence: String,
    /// Go 標籤：`json:"codec"`
    pub codec: String,
    /// Go 標籤：`json:"width"`
    pub width: i64,
    /// Go 標籤：`json:"height"`
    pub height: i64,
    /// Go 標籤：`json:"frame"`
    pub frame: String,
    /// Go 標籤：`json:"source"`
    pub source: SeparatedTownLabelDocumentSource,
}

/// LoadNativeTownAssets uses only separated FDOTHER surfaces/label and the
/// separated FDICON bank. The archive adapter above remains source-oracle only.
///
/// Go：`native_town_scene.go:112`，`func LoadNativeTownAssets(packRoot string) (*NativeTownAssets, error)`
pub fn load_native_town_assets(pack_root: &str) -> Result<NativeTownAssets> {
    todo!("campaign/native_town_scene.go:LoadNativeTownAssets")
}

/// Go：`native_town_scene.go:159`，`func decodeNativeTownAssets(fdotherPath string, bank *fdicon.Bank) (*NativeTownAssets, error)`
pub(crate) fn decode_native_town_assets(fdother_path: &str, bank: Option<&fdicon::Bank>) -> Result<NativeTownAssets> {
    todo!("campaign/native_town_scene.go:decodeNativeTownAssets")
}

/// ComposeNativeTownFrame reproduces 0x2cf71's steady redraw. It starts from
/// the caller-selected FDOTHER background, redraws the current-label panel and
/// FDTXT_000 selection name, applies FDICON pulse sequence 0,1,2,1, then copies
/// the native 312x192 viewport to VGA (4,4).
///
/// Go：`native_town_scene.go:200`，`func ComposeNativeTownFrame( assets *NativeTownAssets, strings *fdtxt.Strings, font *fdtxt.Font, variant, selection, pulse int, ) ([]byte, error)`
pub fn compose_native_town_frame(
    assets: Option<&NativeTownAssets>,
    strings: Option<&fdtxt::Strings>,
    font: Option<&fdtxt::Font>,
    variant: i64,
    selection: i64,
    pulse: i64,
) -> Result<Vec<u8>> {
    compose_native_town_frame_inner(assets, strings, font, variant, selection, pulse, true)
}

/// ComposeNativeTownFrameWithoutCursor supplies the same opaque UI underlay for
/// the HD cursor. Native rendering continues to use ComposeNativeTownFrame.
///
/// Go：`native_town_scene.go:211`，`func ComposeNativeTownFrameWithoutCursor( assets *NativeTownAssets, strings *fdtxt.Strings, font *fdtxt.Font, variant, selection, pulse int, ) ([]byte, error)`
pub fn compose_native_town_frame_without_cursor(
    assets: Option<&NativeTownAssets>,
    strings: Option<&fdtxt::Strings>,
    font: Option<&fdtxt::Font>,
    variant: i64,
    selection: i64,
    pulse: i64,
) -> Result<Vec<u8>> {
    compose_native_town_frame_inner(assets, strings, font, variant, selection, pulse, false)
}

/// Go：`native_town_scene.go:218`，`func composeNativeTownFrame( assets *NativeTownAssets, strings *fdtxt.Strings, font *fdtxt.Font, variant, selection, pulse int, drawCursor bool, ) ([]byte, error)`
pub(crate) fn compose_native_town_frame_inner(
    assets: Option<&NativeTownAssets>,
    strings: Option<&fdtxt::Strings>,
    font: Option<&fdtxt::Font>,
    variant: i64,
    selection: i64,
    pulse: i64,
    draw_cursor: bool,
) -> Result<Vec<u8>> {
    todo!("campaign/native_town_scene.go:composeNativeTownFrame")
}

/// Go 版此處是匿名 struct（field）；Rust 需要具名型別，名稱由骨架產生。
#[derive(Debug, Clone, Default, PartialEq, Eq)]
pub struct SeparatedTownLabelDocumentSource {
    /// Go 標籤：`json:"file"`
    pub file: String,
    /// Go 標籤：`json:"resource"`
    pub resource: i64,
    /// Go 標籤：`json:"size"`
    pub size: i64,
    /// Go 標籤：`json:"md5"`
    pub md5: String,
    /// Go 標籤：`json:"sha256"`
    pub sha256: String,
    /// Go 標籤：`json:"raw_size"`
    pub raw_size: i64,
}
