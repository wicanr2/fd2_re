//! Go 來源：`remake/cmd/fd2/native_church_ui.go`（283 行）。
//!
//! 骨架：Go 檔的 16 個函式中產生 5 個（階段 0 清單），其餘標 `r8`，見 `PORTING.json`。

use super::*;

/// Go：`native_church_ui.go:27`
///
/// 不 derive `Debug`／`Clone`：欄位含 callback（`Box<dyn FnOnce>`）、檔案或亂數產生器等做不到的型別。
#[derive(Default)]
pub(crate) struct NativeChurchUIJob {
    pub(crate) service_menu: Option<Box<NativeServiceMenuPresentation>>,
    pub(crate) frames: Vec<Vec<u8>>,
    pub(crate) restore: Vec<u8>,
    pub(crate) frame: i64,
    pub(crate) drawn: bool,
    pub(crate) after: Option<Box<dyn FnOnce(&mut Game) + Send>>,
}

impl Game {
    /// Go：`native_church_ui.go:97`，`func (g *Game) composeNativeChurchScene() ([]byte, bool)`
    pub(crate) fn compose_native_church_scene(&self) -> (Vec<u8>, bool) {
        todo!("native_church_ui.go:composeNativeChurchScene")
    }

    /// Go：`native_church_ui.go:120`，`func (g *Game) composeNativeChurchSourceFrame() ([]byte, bool)`
    pub(crate) fn compose_native_church_source_frame(&self) -> (Vec<u8>, bool) {
        todo!("native_church_ui.go:composeNativeChurchSourceFrame")
    }

    /// Go：`native_church_ui.go:152`，`func (g *Game) beginNativeChurchMenuOpening() bool`
    pub(crate) fn begin_native_church_menu_opening(&mut self) -> bool {
        todo!("native_church_ui.go:beginNativeChurchMenuOpening")
    }

    /// Go：`native_church_ui.go:191`，`func (g *Game) stepNativeChurchUILifecycle(now time.Time)`
    pub(crate) fn step_native_church_ui_lifecycle(&mut self, now: GameInstant) {
        todo!("native_church_ui.go:stepNativeChurchUILifecycle")
    }

    /// Go：`native_church_ui.go:248`，`func (g *Game) resetNativeChurchUIPulse()`
    pub(crate) fn reset_native_church_ui_pulse(&mut self) {
        todo!("native_church_ui.go:resetNativeChurchUIPulse")
    }
}
