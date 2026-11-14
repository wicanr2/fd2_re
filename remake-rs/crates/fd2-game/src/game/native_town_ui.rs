//! Go 來源：`remake/cmd/fd2/native_town_ui.go`（234 行）。
//!
//! 骨架：Go 檔的 16 個函式中產生 11 個（階段 0 清單），其餘標 `r8`，見 `PORTING.json`。

use super::*;

/// Go：`native_town_ui.go:13`
#[derive(Debug, Clone, Default, PartialEq, Eq)]
pub(crate) struct NativeTownUIAssets {
    pub(crate) scene: Option<campaign::NativeTownAssets>,
}

/// A caller owns its resolved source, independently of later roster changes.
/// The decoded assets are immutable; the selected pulse array is a value copy.
///
/// Go：`native_town_ui.go:19`
#[derive(Debug, Clone, Default)]
pub(crate) struct NativeTownFrameSource {
    pub(crate) scene: campaign::NativeTownAssets,
    pub(crate) strings: Option<fdtxt::Strings>,
    pub(crate) font: Option<fdtxt::Font>,
    pub(crate) variant: i64,
    pub(crate) selection: i64,
    pub(crate) pulse: i64,
    pub(crate) group: i64,
    pub(crate) indexed: Vec<u8>,
}

impl Game {
    /// Go：`native_town_ui.go:27`，`func (g *Game) snapshotNativeTownSource() *nativeTownFrameSource`
    pub(crate) fn snapshot_native_town_source(&mut self) -> Option<NativeTownFrameSource> {
        todo!("native_town_ui.go:snapshotNativeTownSource")
    }
}

impl NativeTownFrameSource {
    /// Go：`native_town_ui.go:62`，`func (d *nativeTownFrameSource) compose(fill *byte, cursor bool, font *fdtxt.Font) ([]byte, bool)`
    /// Go 檢查接收者是否為 nil；Rust 的 `self` 不會是 null，該分支不移植。
    pub(crate) fn compose(&mut self, fill: Option<&u8>, cursor: bool, font: &mut fdtxt::Font) -> (Vec<u8>, bool) {
        todo!("native_town_ui.go:compose")
    }
}

/// nativeTownMoveSelection preserves 0x2cf01..0x2cf65: right decrements and
/// wraps normal 0 to 4; left increments and wraps every value above 4 to 0.
/// This also defines how the revealed hidden selection 5 returns to the five
/// normal facilities.
///
/// Go：`native_town_ui.go:130`，`func nativeTownMoveSelection(selection, delta int) (int, bool)`
pub(crate) fn native_town_move_selection(selection: i64, delta: i64) -> (i64, bool) {
    todo!("native_town_ui.go:nativeTownMoveSelection")
}

impl Game {
    /// moveNativeTownSelection changes only the selector. The original
    /// 0x2ce7a/0x2ceac branches do not write the shared pulse counter at 0x54133.
    ///
    /// Go：`native_town_ui.go:146`，`func (g *Game) moveNativeTownSelection(delta int) bool`
    pub(crate) fn move_native_town_selection(&mut self, delta: i64) -> bool {
        todo!("native_town_ui.go:moveNativeTownSelection")
    }
}

/// Go：`native_town_ui.go:166`，`func loadNativeTownUIAssets() (*nativeTownUIAssets, error)`
pub(crate) fn load_native_town_ui_assets() -> Result<Option<NativeTownUIAssets>> {
    todo!("native_town_ui.go:loadNativeTownUIAssets")
}

impl Game {
    /// Go：`native_town_ui.go:174`，`func (g *Game) resetNativeTownUIPulse()`
    pub(crate) fn reset_native_town_ui_pulse(&mut self) {
        todo!("native_town_ui.go:resetNativeTownUIPulse")
    }

    /// 0x2d1b5 advances the four-state counter after a signed BIOS low-word delta
    /// of four. Counter 3 is rendered with FDICON sprite 1 by the compositor.
    ///
    /// Go：`native_town_ui.go:183`，`func (g *Game) stepNativeTownUIPulseTick(rawTick int)`
    pub(crate) fn step_native_town_ui_pulse_tick(&mut self, raw_tick: i64) {
        todo!("native_town_ui.go:stepNativeTownUIPulseTick")
    }

    /// Go：`native_town_ui.go:197`，`func (g *Game) stepNativeTownUILifecycle(nowTick int)`
    pub(crate) fn step_native_town_ui_lifecycle(&mut self, now_tick: i64) {
        todo!("native_town_ui.go:stepNativeTownUILifecycle")
    }

    /// Go：`native_town_ui.go:208`，`func (g *Game) composeNativeTownFrame() ([]byte, bool)`
    pub(crate) fn compose_native_town_frame(&mut self) -> (Vec<u8>, bool) {
        todo!("native_town_ui.go:composeNativeTownFrame")
    }

    /// Go：`native_town_ui.go:212`，`func (g *Game) composeNativeTownFrameWithBackground(fill *byte) ([]byte, bool)`
    pub(crate) fn compose_native_town_frame_with_background(&mut self, fill: &mut u8) -> (Vec<u8>, bool) {
        todo!("native_town_ui.go:composeNativeTownFrameWithBackground")
    }

    /// Go：`native_town_ui.go:216`，`func (g *Game) composeNativeTownFrameLayers(fill *byte, cursor bool) ([]byte, bool)`
    pub(crate) fn compose_native_town_frame_layers(&mut self, fill: &mut u8, cursor: bool) -> (Vec<u8>, bool) {
        todo!("native_town_ui.go:composeNativeTownFrameLayers")
    }
}
