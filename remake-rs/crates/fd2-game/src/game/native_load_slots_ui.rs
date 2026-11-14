//! Go 來源：`remake/cmd/fd2/native_load_slots_ui.go`（360 行）。
//!
//! 骨架：Go 檔的 10 個函式中產生 7 個（階段 0 清單），其餘標 `r8`，見 `PORTING.json`。

use super::*;

/// Go：`native_load_slots_ui.go:19`
#[derive(Debug, Clone, Default)]
pub(crate) struct NativeLoadSlotsUIAssets {
    pub(crate) hd: Option<ModernHDTitleLoadSlots>,
    pub(crate) dialogue_box: fdother::LMI1Entry,
    pub(crate) strings: Option<fdtxt::Strings>,
    pub(crate) font: Option<fdtxt::Font>,
    pub(crate) palette: ColorPalette,
}

/// Go：`native_load_slots_ui.go:32`，`func loadNativeLoadSlotsUIAssets() (*nativeLoadSlotsUIAssets, error)`
pub(crate) fn load_native_load_slots_ui_assets() -> Result<Option<NativeLoadSlotsUIAssets>> {
    todo!("native_load_slots_ui.go:loadNativeLoadSlotsUIAssets")
}

/// Go：`native_load_slots_ui.go:64`，`func nativeLoadSlotMetadata() ([4]campaign.NativeLoadSlot, bool)`
pub(crate) fn native_load_slot_metadata() -> ([campaign::NativeLoadSlot; 4], bool) {
    todo!("native_load_slots_ui.go:nativeLoadSlotMetadata")
}

/// Go：`native_load_slots_ui.go:90`，`func nativeLoadSlotMetadataFromFD2SAV( path string, ) ([4]campaign.NativeLoadSlot, bool)`
pub(crate) fn native_load_slot_metadata_from_fd2_sav(path: &str) -> ([campaign::NativeLoadSlot; 4], bool) {
    todo!("native_load_slots_ui.go:nativeLoadSlotMetadataFromFD2SAV")
}

/// Go：`native_load_slots_ui.go:116`，`func nativeLoadSlotConfirmable(slot int) (confirmable, native bool)`
pub(crate) fn native_load_slot_confirmable(slot: i64) -> (bool, bool) {
    todo!("native_load_slots_ui.go:nativeLoadSlotConfirmable")
}

impl Game {
    /// confirmTitleLoadSlot 是標題四槽 selector 的正式確認擁有者。原版槽只有在
    /// checksum、typed restore 與戰間節點進入全部成功後才離開 loadslots；失敗會
    /// 保留選槽畫面，避免看似進入遊戲卻留下部分還原狀態。
    ///
    /// Go：`native_load_slots_ui.go:130`，`func (g *Game) confirmTitleLoadSlot(selected int) bool`
    pub(crate) fn confirm_title_load_slot(&mut self, selected: i64) -> bool {
        todo!("native_load_slots_ui.go:confirmTitleLoadSlot")
    }

    /// loadNativeGameFromSlot owns the original four-slot LOAD transaction. It is
    /// deliberately separate from title CONTINUE/current-snapshot restore.
    ///
    /// Go：`native_load_slots_ui.go:166`，`func (g *Game) loadNativeGameFromSlot(path string, slot int) error`
    pub(crate) fn load_native_game_from_slot(&mut self, path: &str, slot: i64) -> Result<()> {
        todo!("native_load_slots_ui.go:loadNativeGameFromSlot")
    }
}

/// Go：`native_load_slots_ui.go:170`
/// Go `errors.New` 的哨兵錯誤；`errors.Is` 以 [`GameError::is`] 判斷。
pub(crate) static ERR_NATIVE_HOTEL_PREPARATION_SLOT: GameError = GameError::sentinel("本章不能由商店內讀取！");

impl Game {
    /// restoreNativeChapterSlot共用四槽資料交易。酒店caller先查302BF的gate，
    /// 成功後仍在酒店顯示1DE；標題LOAD則沿原plan進城鎮／整備。
    ///
    /// Go：`native_load_slots_ui.go:196`，`func (g *Game) restoreNativeChapterSlot(path string, slot int, inHotel bool) error`
    pub(crate) fn restore_native_chapter_slot(&mut self, path: &str, slot: i64, in_hotel: bool) -> Result<()> {
        todo!("native_load_slots_ui.go:restoreNativeChapterSlot")
    }
}
