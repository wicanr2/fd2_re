//! Go 來源：`remake/cmd/fd2/native_shop_ui.go`（2228 行）。
//!
//! 骨架：Go 檔的 52 個函式中產生 7 個（階段 0 清單），其餘標 `r8`，見 `PORTING.json`。

use super::*;

/// Go：`native_shop_ui.go:16`
///
/// 不 derive `Debug`／`Clone`：欄位含 callback（`Box<dyn FnOnce>`）、檔案或亂數產生器等做不到的型別。
#[derive(Clone, Default)]
pub(crate) struct NativeShopUIAssets {
    pub(crate) shops: BTreeMap<i64, campaign::NativeShopAssets>,
    pub(crate) portraits: BTreeMap<i64, dato::Frame>,
    pub(crate) item_assets: battle::NativeItemPanelDataAssets,
    pub(crate) effect_rows: Vec<u8>,
}

/// Go：`native_shop_ui.go:1040`，`func loadNativeShopUIAssets( shared *nativeClassUIAssets, ) (*nativeShopUIAssets, error)`
pub(crate) fn load_native_shop_ui_assets(shared: Option<&NativeClassUIAssets>) -> Result<Option<NativeShopUIAssets>> {
    todo!("native_shop_ui.go:loadNativeShopUIAssets")
}

impl Game {
    /// Go：`native_shop_ui.go:1084`，`func (g *Game) setupNativeShop() bool`
    pub(crate) fn setup_native_shop(&mut self) -> bool {
        todo!("native_shop_ui.go:setupNativeShop")
    }

    /// Go：`native_shop_ui.go:1106`，`func (g *Game) nativeShopState() ( *campaign.NativeShopAssets, dato.Frame, int, bool, )`
    pub(crate) fn native_shop_state(&self) -> (Option<campaign::NativeShopAssets>, dato::Frame, i64, bool) {
        todo!("native_shop_ui.go:nativeShopState")
    }

    /// Go：`native_shop_ui.go:1120`，`func (g *Game) composeNativeShopStable() ([]byte, bool)`
    pub(crate) fn compose_native_shop_stable(&mut self) -> (Vec<u8>, bool) {
        todo!("native_shop_ui.go:composeNativeShopStable")
    }

    /// Go：`native_shop_ui.go:1312`，`func (g *Game) beginNativeShopServiceOpening() bool`
    pub(crate) fn begin_native_shop_service_opening(&mut self) -> bool {
        todo!("native_shop_ui.go:beginNativeShopServiceOpening")
    }

    /// Go：`native_shop_ui.go:1452`，`func (g *Game) stepNativeShopUILifecycle(now time.Time)`
    pub(crate) fn step_native_shop_ui_lifecycle(&mut self, now: GameInstant) {
        todo!("native_shop_ui.go:stepNativeShopUILifecycle")
    }

    /// Go：`native_shop_ui.go:1617`，`func (g *Game) resetNativeShopUIPulse()`
    pub(crate) fn reset_native_shop_ui_pulse(&mut self) {
        todo!("native_shop_ui.go:resetNativeShopUIPulse")
    }
}
