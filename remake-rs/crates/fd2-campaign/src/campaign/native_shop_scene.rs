//! Go 來源：`remake/internal/campaign/native_shop_scene.go`（331 行）。

use super::*;

pub const NATIVE_SHOP_WIDTH: i64 = 320;

pub const NATIVE_SHOP_HEIGHT: i64 = 200;

/// Go 原式：`95*NativeShopWidth + 5`
pub const NATIVE_SHOP_DECORATION_OFFSET: i64 = 30405;

/// Go 原式：`99*NativeShopWidth + 16`
pub const NATIVE_SHOP_GOLD_OFFSET: i64 = 31696;

/// Go 原式：`98*NativeShopWidth + 16`
pub const NATIVE_SHOP_GOLD_ROLL_OFFSET: i64 = 31376;

/// Go 原式：`119*NativeShopWidth + 12`
pub const NATIVE_SHOP_TEXT_OFFSET: i64 = 38092;

/// Go：`native_shop_scene.go:22`
pub(crate) static NATIVE_FACILITY_PORTRAIT_OFFSETS: LazyLock<BTreeMap<i64, i64>> =
    LazyLock::new(|| BTreeMap::from([(128, 4283), (129, 1707), (130, 3939), (131, 1398), (132, 3644)]));

/// Go：`native_shop_scene.go:30`，`func nativeFacilityPortraitOffset(portraitID int) int`
pub(crate) fn native_facility_portrait_offset_inner(portrait_id: i64) -> i64 {
    todo!("campaign/native_shop_scene.go:nativeFacilityPortraitOffset")
}

/// NativeFacilityPortraitOffset 提供已證實的店員頭像目的位址，供多語文字層
/// 重建同一對話框時共用；未知頭像仍沿用原版下框預設位置。
///
/// Go：`native_shop_scene.go:39`，`func NativeFacilityPortraitOffset(portraitID int) int`
pub fn native_facility_portrait_offset(portrait_id: i64) -> i64 {
    native_facility_portrait_offset_inner(portrait_id)
}

/// NativeShopAssets preserves the mixed-codec FDOTHER resource selected by
/// 0x2e341. Entry zero is the 0x16886 four-mode 320x200 background and entry
/// one is the 0x4e8af opaque decoration. Entries 3..10 are decoded separately
/// as the literal transparent pairs consumed by 0x2d669 through 0x4e9e4;
/// every other entry remains raw until its direct caller proves a codec.
///
/// Go：`native_shop_scene.go:48`
#[derive(Debug, Clone, Default)]
pub struct NativeShopAssets {
    pub resource_id: i64,
    pub background: Vec<u8>,
    pub decoration: fdother::LMI1Entry,
    pub raw_entries: Vec<Vec<u8>>,
    pub gold_roll_strip: fdother::RawCell,
    pub service_cells: [[fdother::RawCell; 2]; 4],
    pub price_cell: fdother::RawCell,
    pub panel: fdother::LMI1Entry,
    pub compare_cells: [fdother::LMI1Entry; 5],
    pub success_frames: Vec<fdother::Frame>,
}

/// DecodeNativeShopAssets is the offline/source-oracle decoder for exactly the
/// three resources selected by the native hub variants. Production loads the
/// exported contract through LoadSeparatedNativeShopAssets and never falls
/// back here. Resources 12/63 use an outer LLLLLL directory while resource 29
/// uses a scene-flavoured LMI1 directory. Per-entry codecs are selected only
/// from recovered call sites and are not inferred by container.
///
/// Go：`native_shop_scene.go:67`，`func DecodeNativeShopAssets(datPath string, resourceID int) (*NativeShopAssets, error)`
pub fn decode_native_shop_assets(dat_path: &str, resource_id: i64) -> Result<NativeShopAssets> {
    todo!("campaign/native_shop_scene.go:DecodeNativeShopAssets")
}

/// Go：`native_shop_scene.go:164`，`func nativeShopEntries(raw []byte) ([][]byte, error)`
pub(crate) fn native_shop_entries(raw: &[u8]) -> Result<Vec<Vec<u8>>> {
    todo!("campaign/native_shop_scene.go:nativeShopEntries")
}

/// ComposeNativeShopScene executes the stable target of 0x2e341 after
/// 0x1956b's six-band reveal: FDOTHER#5's dialogue grid, the selected DATO
/// portrait, decoration cell #1, eight-digit gold counter, and the proven
/// FDTXT_000 greeting style. The reveal timing remains a presentation concern;
/// this compositor returns its exact final indexed framebuffer.
///
/// Go：`native_shop_scene.go:228`，`func ComposeNativeShopScene( assets *NativeShopAssets, dialogueCells []fdother.RawCell, digitFrames []fdother.Frame, portrait dato.Frame, portraitID int, strings *fdtxt.Strings, font *fdtxt.Font, gold, textIndex int, ) ([]byte, error)`
pub fn compose_native_shop_scene(
    assets: Option<&NativeShopAssets>,
    dialogue_cells: &[fdother::RawCell],
    digit_frames: &[fdother::Frame],
    portrait: &dato::Frame,
    portrait_id: i64,
    strings: Option<&fdtxt::Strings>,
    font: Option<&fdtxt::Font>,
    gold: i64,
    text_index: i64,
) -> Result<Vec<u8>> {
    todo!("campaign/native_shop_scene.go:ComposeNativeShopScene")
}

/// ComposeNativeShopSceneWithoutText 建立原版商店穩定畫面、店員與金額，
/// 但保留空白對話文字區，供已驗證的多語 renderer 使用。
///
/// Go：`native_shop_scene.go:254`，`func ComposeNativeShopSceneWithoutText( assets *NativeShopAssets, dialogueCells []fdother.RawCell, digitFrames []fdother.Frame, portrait dato.Frame, portraitID int, gold int, ) ([]byte, error)`
pub fn compose_native_shop_scene_without_text(
    assets: Option<&NativeShopAssets>,
    dialogue_cells: &[fdother::RawCell],
    digit_frames: &[fdother::Frame],
    portrait: &dato::Frame,
    portrait_id: i64,
    gold: i64,
) -> Result<Vec<u8>> {
    todo!("campaign/native_shop_scene.go:ComposeNativeShopSceneWithoutText")
}

/// ComposeNativeShopBareScene is the caller-owned framebuffer restored after
/// closing purchase/recipient dialogue. The FDOTHER background already contains
/// the facility portrait base. 0x2f4c6 draws its success effect onto this scene;
/// only its tail 0x16559(0) overlays DATO frame zero. Drawing that frame here
/// would move the native portrait restore before the success presentation.
///
/// Go：`native_shop_scene.go:300`，`func ComposeNativeShopBareScene( assets *NativeShopAssets, digitFrames []fdother.Frame, gold int, ) ([]byte, error)`
pub fn compose_native_shop_bare_scene(
    assets: Option<&NativeShopAssets>,
    digit_frames: &[fdother::Frame],
    gold: i64,
) -> Result<Vec<u8>> {
    todo!("campaign/native_shop_scene.go:ComposeNativeShopBareScene")
}
