//! Go 來源：`remake/cmd/fd2/modern_hd_title_interface.go`（281 行）。
//!
//! 骨架：Go 檔的 11 個函式中產生 4 個（階段 0 清單），其餘標 `r8`，見 `PORTING.json`。
//! HD 檔：型別只保留第一章忠實模式執行到的欄位，其餘欄位與函式在 R6。

use super::*;

/// Go：`modern_hd_title_interface.go:18`
pub(crate) static MODERN_HD_TITLE_ITEM_HASHES: LazyLock<[String; 6]> = LazyLock::new(|| {
    [
        String::from("384f6a9d22e430c71aff99587aa22dccc55ee24d474fa2ec563d73fde0da9014"),
        String::from("6b82c6b8693ec55b5cfcb9f1e74e1bc9e67f232483a36363f2c4f7c786da314a"),
        String::from("cf10d302eec5615156f8d6061bec36780662a2c62ba226a983da3387bcdd1a96"),
        String::from("f237dfcd75c398f841c9a7c12ecbd846d8a35bb67ec85180e7b31baef55cf3c8"),
        String::from("8ff71b5cc1af08bdeec80b23248d101049f8e8adb2d35336f41ab8dc75f542cb"),
        String::from("842648c39e8cf0a6d897ab1d7ae166d07194b256e34481910cd3303137fc77b8"),
    ]
});

/// Go：`modern_hd_title_interface.go:26`
pub(crate) static MODERN_HD_TITLE_MASK_HASHES: LazyLock<[String; 3]> = LazyLock::new(|| {
    [
        String::from("97762f67cb746e5c82034901351a88ea591ac674323a745bfce7772d84ea0af1"),
        String::from("6c1d60b8f28ad79070aa42defb4681bc7ffe060277f44fae6f3b1f73dea31b04"),
        String::from("1dd7b5aec086135580c588a9e91cb20983cfdb38ce06d49567355c7944980ffc"),
    ]
});

/// Go：`modern_hd_title_interface.go:32`
///
/// HD 型別：只保留第一章忠實模式下執行到的函式用到的欄位（120「階段 1 骨架的對應規則」），其餘欄位在 R6 補上。
#[derive(Debug, Clone, Default)]
pub(crate) struct ModernHDTitleMenuItem {
    pub(crate) item: i64,
    pub(crate) state: i64,
    pub(crate) source: fdother::Frame,
    pub(crate) blank: Vec<u8>,
    pub(crate) palette: ColorPalette,
    pub(crate) original: Option<Arc<Picture>>,
    pub(crate) picture: Option<Arc<Picture>>,
    pub(crate) picture_identity: Option<Arc<Picture>>,
    pub(crate) foreground: ColorRgba,
    pub(crate) shadow: ColorRgba,
    pub(crate) seal: [u8; 32],
}

/// Go：`modern_hd_title_interface.go:42`，`func titleMenuHDSeal(h *modernHDTitleMenuItem) [32]byte`
pub(crate) fn title_menu_hd_seal(h: Option<&mut ModernHDTitleMenuItem>) -> [u8; 32] {
    todo!("modern_hd_title_interface.go:titleMenuHDSeal")
}

/// Go：`modern_hd_title_interface.go:50`，`func freezeModernHDTitleMenuItem(source fdother.Frame, dac []byte, palette color.Palette, item, state int, original *ebiten.Image) (*modernHDTitleMenuItem, error)`
pub(crate) fn freeze_modern_hd_title_menu_item(
    source: &fdother::Frame,
    dac: &[u8],
    palette: &ColorPalette,
    item: i64,
    state: i64,
    original: Option<&mut Picture>,
) -> Result<Option<ModernHDTitleMenuItem>> {
    todo!("modern_hd_title_interface.go:freezeModernHDTitleMenuItem")
}

/// Go：`modern_hd_title_interface.go:167`
///
/// HD 型別：只保留第一章忠實模式下執行到的函式用到的欄位（120「階段 1 骨架的對應規則」），其餘欄位在 R6 補上。
#[derive(Debug, Clone, Default, PartialEq, Eq)]
pub(crate) struct ModernHDTitleLoadSlots {
    pub(crate) chrome: Option<ModernHDDialogueChrome>,
    pub(crate) chrome_identity: Option<ModernHDDialogueChrome>,
    pub(crate) json_chrome: Option<ModernHDDialogueChrome>,
    pub(crate) json_identity: Option<ModernHDDialogueChrome>,
    pub(crate) cells: Vec<fdother::RawCell>,
    pub(crate) seal: [u8; 32],
}

/// Go：`modern_hd_title_interface.go:176`，`func titleLoadSlotsHDSeal(a *nativeLoadSlotsUIAssets, h *modernHDTitleLoadSlots) [32]byte`
pub(crate) fn title_load_slots_hd_seal(
    a: Option<&mut NativeLoadSlotsUIAssets>,
    h: Option<&mut ModernHDTitleLoadSlots>,
) -> [u8; 32] {
    todo!("modern_hd_title_interface.go:titleLoadSlotsHDSeal")
}

/// Go：`modern_hd_title_interface.go:184`，`func freezeModernHDTitleLoadSlots(a *nativeLoadSlotsUIAssets, dac []byte) (*modernHDTitleLoadSlots, error)`
pub(crate) fn freeze_modern_hd_title_load_slots(
    a: Option<&mut NativeLoadSlotsUIAssets>,
    dac: &[u8],
) -> Result<Option<ModernHDTitleLoadSlots>> {
    todo!("modern_hd_title_interface.go:freezeModernHDTitleLoadSlots")
}
