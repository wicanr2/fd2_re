//! Go 來源：`remake/cmd/fd2/native_physical_scene.go`（471 行）。
//!
//! 骨架：Go 檔的 11 個函式中產生 8 個（階段 0 清單），其餘標 `r8`，見 `PORTING.json`。

use super::*;

/// nativePhysicalScene 是一次正常 0x28A6C 的場景快照，反擊沿用同一份。
/// READY 子規格見 docs/data/ida/fd2_physical_background_selection_20261004.json。
/// 首次非零 header departure／scroll 已接線；完整 DAC 與尾段仍屬 E1。
///
/// Go：`native_physical_scene.go:20`
///
/// 不 derive `Debug`／`Clone`：欄位含 callback（`Box<dyn FnOnce>`）、檔案或亂數產生器等做不到的型別。
#[derive(Clone, Default)]
pub(crate) struct NativePhysicalScene {
    pub(crate) hd_panel_text: Vec<ModernBattlePanelText>,
    pub(crate) actor_battle_fig: i64,
    pub(crate) target_battle_fig: i64,
    pub(crate) prelude_base: Vec<u8>,
    pub(crate) selection: battle::NativePhysicalSceneSelection,
    pub(crate) background: Option<Arc<Picture>>,
    pub(crate) pedestal: Option<Arc<Picture>>,
    pub(crate) background_frame: fdother::Frame,
    pub(crate) panel_assets: battle::NativeItemPanelDataAssets,
    pub(crate) actor_record: Vec<u8>,
    pub(crate) target_record: Vec<u8>,
    pub(crate) actor_index: i64,
    pub(crate) target_index: i64,
    pub(crate) prelude: Vec<battlepresent::NativeCommand24PreludeFrame>,
    pub(crate) prelude_images: Vec<Option<Arc<Picture>>>,
    pub(crate) prelude_frame: i64,
    pub(crate) prelude_drawn: bool,
    pub(crate) counter_active: bool,
    pub(crate) base_image: Option<Arc<Picture>>,
    pub(crate) base_actor_hp: i64,
    pub(crate) base_target_hp: i64,
    pub(crate) departure: Vec<Vec<u8>>,
    pub(crate) transition: Vec<Vec<u8>>,
    pub(crate) lead_images: Vec<Option<Arc<Picture>>>,
    pub(crate) lead_sources: Vec<ModernHDBattleSources>,
    pub(crate) lead_delays: Vec<i64>,
    pub(crate) lead_frame: i64,
    pub(crate) lead_ticks: i64,
    pub(crate) lead_drawn: bool,
    pub(crate) lead_body_ticks: i64,
    pub(crate) attack_start: i64,
    pub(crate) target_background: fdother::Frame,
    pub(crate) platform_frame: fdother::Frame,
    pub(crate) raw_side: u8,
    pub(crate) body_resources: Option<NativePhysicalBodyResources>,
    pub(crate) body: Option<NativePhysicalBodyPlayback>,
}

impl Game {
    /// Go：`native_physical_scene.go:53`，`func (g *Game) nativePhysicalTerrainControl(unit *battle.Unit) ([4]byte, error)`
    /// Go 檢查接收者是否為 nil；Rust 的 `self` 不會是 null，該分支不移植。
    pub(crate) fn native_physical_terrain_control(&self, unit: Option<UnitIdx>) -> Result<[u8; 4]> {
        todo!("native_physical_scene.go:nativePhysicalTerrainControl")
    }
}

/// nativePhysicalSurfaceImage 保留 mask 與 index0 的差別，不以透明 PNG 補值。
///
/// Go：`native_physical_scene.go:82`，`func nativePhysicalSurfaceRGBA(frame fdother.Frame, palette color.Palette) (*image.RGBA, error)`
pub(crate) fn native_physical_surface_rgba(frame: &fdother::Frame, palette: &ColorPalette) -> Result<Option<Rgba>> {
    todo!("native_physical_scene.go:nativePhysicalSurfaceRGBA")
}

/// Go：`native_physical_scene.go:102`，`func nativePhysicalSurfaceImage(frame fdother.Frame, palette color.Palette) (*ebiten.Image, error)`
pub(crate) fn native_physical_surface_image(
    frame: &fdother::Frame,
    palette: &ColorPalette,
) -> Result<Option<Arc<Picture>>> {
    todo!("native_physical_scene.go:nativePhysicalSurfaceImage")
}

impl Game {
    /// prepareNativePhysicalScene 在任何結算寫入前驗證正常場景的 BG／TAI 來源。
    /// 無原生地圖狀態的相容場景仍由自己的既有 owner 處理。
    ///
    /// Go：`native_physical_scene.go:112`，`func (g *Game) prepareNativePhysicalScene(actor, target *battle.Unit) (*nativePhysicalScene, error)`
    /// Go 檢查接收者是否為 nil；Rust 的 `self` 不會是 null，該分支不移植。
    pub(crate) fn prepare_native_physical_scene(
        &mut self,
        actor: Option<UnitIdx>,
        target: UnitIdx,
    ) -> Result<Option<NativePhysicalScene>> {
        todo!("native_physical_scene.go:prepareNativePhysicalScene")
    }

    /// preflightNativePhysicalMapReturn 用候選資料驗證既有11CAC合成器。
    /// 不發布候選的State、clock或畫布；資產與單位保持唯讀。
    ///
    /// Go：`native_physical_scene.go:175`，`func (g *Game) preflightNativePhysicalMapReturn() error`
    pub(crate) fn preflight_native_physical_map_return(&self) -> Result<()> {
        todo!("native_physical_scene.go:preflightNativePhysicalMapReturn")
    }

    /// nativePhysicalBase 保留 0x28CE7→0x28D48→0x28D62 的順序。
    /// 320×200 base 的清零來自 0x28B41..0x28B4E 明確 memset。
    ///
    /// Go：`native_physical_scene.go:193`，`func (g *Game) nativePhysicalBase(scene *nativePhysicalScene, actorRecord, targetRecord []byte) ([]byte, error)`
    pub(crate) fn native_physical_base(
        &mut self,
        scene: &mut NativePhysicalScene,
        actor_record: &mut [u8],
        target_record: &mut [u8],
    ) -> Result<Vec<u8>> {
        todo!("native_physical_scene.go:nativePhysicalBase")
    }

    /// Go：`native_physical_scene.go:211`，`func (g *Game) prepareNativePhysicalPrelude(scene *nativePhysicalScene, actor, target *battle.Unit, attack *figani.Animation, bg, tai fdother.Frame) error`
    pub(crate) fn prepare_native_physical_prelude(
        &mut self,
        scene: &mut NativePhysicalScene,
        actor: UnitIdx,
        target: UnitIdx,
        attack: &mut figani::Animation,
        bg: &fdother::Frame,
        tai: &fdother::Frame,
    ) -> Result<()> {
        todo!("native_physical_scene.go:prepareNativePhysicalPrelude")
    }

    /// Go：`native_physical_scene.go:372`，`func (g *Game) drawNativePhysicalPrelude(screen *ebiten.Image) bool`
    pub(crate) fn draw_native_physical_prelude(&mut self, screen: &mut Picture) -> bool {
        todo!("native_physical_scene.go:drawNativePhysicalPrelude")
    }
}
