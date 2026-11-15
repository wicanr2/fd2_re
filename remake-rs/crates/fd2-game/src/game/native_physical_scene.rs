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
    /// R4 D 組：Go 的 `[]*ebiten.Image` 改成索引色畫格（`nativeCommand24PreludeImages`），GPU 貼圖在 R5（簽名變更紀錄）。
    pub(crate) prelude_images: Vec<IndexedFrame>,
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
        let unavailable = || game_err!("native physical terrain control unavailable");
        let (Some(st), Some(m), Some(unit)) = (self.st.as_ref(), self.m.as_ref(), unit) else {
            return Err(unavailable());
        };
        let u = &st.units[unit];
        if m.w != st.w
            || m.h != st.h
            || m.w <= 0
            || m.h <= 0
            || u.x < 0
            || u.y < 0
            || u.x >= m.w
            || u.y >= m.h
            || m.native_terrain_control.is_empty()
            || m.native_terrain_control.len() % 4 != 0
        {
            return Err(unavailable());
        }
        let tile = if st.has_native_map_event_grid {
            let (tile, ok) = st.native_map_tile_at(u.x, u.y);
            if !ok {
                return Err(game_err!("native physical mutable map is malformed"));
            }
            tile
        } else {
            if m.tiles.len() as i64 != m.w * m.h {
                return Err(game_err!("native physical map tiles unavailable"));
            }
            m.tiles[(u.y * m.w + u.x) as usize]
        };
        if !(0..=0x3ff).contains(&tile) || tile >= (m.native_terrain_control.len() / 4) as i64 {
            return Err(game_err!("native physical terrain selector out of range"));
        }
        let at = (tile * 4) as usize;
        let mut control = [0u8; 4];
        control.copy_from_slice(&m.native_terrain_control[at..at + 4]);
        Ok(control)
    }
}

/// nativePhysicalSurfaceImage 保留 mask 與 index0 的差別，不以透明 PNG 補值。
///
/// Go：`native_physical_scene.go:82`，`func nativePhysicalSurfaceRGBA(frame fdother.Frame, palette color.Palette) (*image.RGBA, error)`
pub(crate) fn native_physical_surface_rgba(frame: &fdother::Frame, palette: &ColorPalette) -> Result<Option<Rgba>> {
    if frame.width <= 0
        || frame.height <= 0
        || frame.width > 320
        || frame.height > 200
        || palette.len() != 256
        || frame.indexed.len() as i64 != frame.width * frame.height
        || frame.mask.len() != frame.indexed.len()
        || !frame.pixels.is_empty()
    {
        return Err(game_err!("native physical surface is malformed"));
    }
    let mut img = Rgba {
        pix: vec![0; (frame.width * frame.height * 4) as usize],
        stride: frame.width * 4,
        rect: Rectangle { min: Point { x: 0, y: 0 }, max: Point { x: frame.width, y: frame.height } },
    };
    for (at, &index) in frame.indexed.iter().enumerate() {
        if frame.mask[at] == 0 {
            continue;
        }
        if frame.mask[at] != 255 {
            return Err(game_err!("native physical surface mask is malformed"));
        }
        // Go `palette[index].RGBA()` 再右移 8 位：`color.RGBA` 的每個分量就是原值。
        let c = palette[index as usize];
        img.pix[at * 4..at * 4 + 4].copy_from_slice(&[c.r, c.g, c.b, 255]);
    }
    Ok(Some(img))
}

/// Go：`native_physical_scene.go:102`，`func nativePhysicalSurfaceImage(frame fdother.Frame, palette color.Palette) (*ebiten.Image, error)`
pub(crate) fn native_physical_surface_image(
    frame: &fdother::Frame,
    palette: &ColorPalette,
) -> Result<Option<Arc<Picture>>> {
    // Go 轉成 `*ebiten.Image`；Rust 保存 RGBA 畫面，GPU 貼圖由外殼建立（R5）。
    let img = native_physical_surface_rgba(frame, palette)?.expect("nativePhysicalSurfaceRGBA 成功時不回 nil");
    Ok(Some(Arc::new(picture_from_rgba(&img))))
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
        let Some(st) = self.st.as_ref() else {
            return Err(game_err!("native physical scene state unavailable"));
        };
        if !st.has_native_map_view_state {
            return Ok(None);
        }
        self.preflight_native_physical_map_return()
            .map_err(|err| err.wrap("native physical map return preflight: "))?;
        let initial = match self.native_command_scene.as_ref() {
            Some(table) => table.initial_background(self.handler_chapter)?,
            // Go 以 nil 接收者呼叫 `(*NativeCommandSceneTable).InitialBackground`，它先檢查 nil。
            None => battle::NativeCommandSceneTable::default().initial_background(self.handler_chapter)?,
        };
        let actor_control = self.native_physical_terrain_control(actor)?;
        let target_control = self.native_physical_terrain_control(Some(target))?;
        let st = self.st.as_ref().unwrap();
        let Some(actor) = actor.filter(|&a| {
            let u = &st.units[a];
            u.has_battle_fig && u.battle_fig >= 0 && u.battle_fig <= 255
        }) else {
            return Err(game_err!("native physical actor FIGANI selector unavailable"));
        };
        let actor_fig = st.units[actor].battle_fig;
        let target_fig = st.units[target].battle_fig;
        let animation = figani::load_separated_resource(&separated_asset_path("animations"), actor_fig * 3 + 1)?;
        let selection = battle::select_native_physical_scene(
            initial,
            Some(&st.units[actor]),
            Some(&st.units[target]),
            actor_control,
            target_control,
            animation.header_byte1,
        )?;
        let root = separated_asset_path("surfaces");
        let bg = fdother::load_separated_single_frame(&root, "BG.DAT", selection.base_bg as i64)
            .map_err(|err| GameError::from(err).wrap(format!("native physical BG{}: ", selection.base_bg)))?;
        let tai = fdother::load_separated_single_frame(&root, "TAI.DAT", selection.tai as i64)
            .map_err(|err| GameError::from(err).wrap(format!("native physical TAI{}: ", selection.tai)))?;
        let background = native_physical_surface_image(&bg, &self.native_ui_palette)?;
        let pedestal = native_physical_surface_image(&tai, &self.native_ui_palette)?;
        let mut scene = NativePhysicalScene {
            selection,
            background,
            pedestal,
            platform_frame: tai.clone(),
            actor_battle_fig: actor_fig,
            target_battle_fig: target_fig,
            ..Default::default()
        };
        self.prepare_native_physical_prelude(&mut scene, actor, target, &animation, &bg, &tai)?;
        self.prepare_native_physical_body_resources(&mut scene, actor, target, &animation)?;
        Ok(Some(scene))
    }

    /// preflightNativePhysicalMapReturn 用候選資料驗證既有11CAC合成器。
    /// 不發布候選的State、clock或畫布；資產與單位保持唯讀。
    ///
    /// Go：`native_physical_scene.go:175`，`func (g *Game) preflightNativePhysicalMapReturn() error`
    ///
    /// R4 D 組：Go 以 `probe := *g` 淺複製整個 `Game` 試合成；Rust 在 `self` 上就地合成再還原（同
    /// [`NativeTransientPreviewSaved`]），所以收 `&mut self`（簽名變更紀錄）。
    pub(crate) fn preflight_native_physical_map_return(&mut self) -> Result<()> {
        let Some(assets) = self.native_map_assets.as_ref() else {
            return Ok(());
        };
        if self.native_palette_ramp.is_some() || assets.palette_dac.len() != 256 * 3 {
            return Err(game_err!("native physical return DAC owner or baseline unavailable"));
        }
        // Go：`state := *g.st; probe.st = &state`（淺複製 State）；work、VGA、DAC 都另行複製。
        let state = self.st.clone().expect("Go：g.st 為 nil 時解參考 panic");
        let mut saved = self.save_native_game_preview(state);
        // 合成不寫 `nativeCh28PostPresent`；照 Go 的副本讓它看到同一個值，結束時放回保存處。
        self.native_ch28_post_present = saved.native_ch28_post_present.take();
        let result = self.compose_native_map_frame_for_physical_return();
        saved.native_ch28_post_present = self.native_ch28_post_present.take();
        self.restore_native_transient_preview(saved);
        result
    }

    /// nativePhysicalBase 保留 0x28CE7→0x28D48→0x28D62 的順序。
    /// 320×200 base 的清零來自 0x28B41..0x28B4E 明確 memset。
    ///
    /// Go：`native_physical_scene.go:193`，`func (g *Game) nativePhysicalBase(scene *nativePhysicalScene, actorRecord, targetRecord []byte) ([]byte, error)`
    pub(crate) fn native_physical_base(
        &mut self,
        scene: &mut NativePhysicalScene,
        actor_record: &[u8],
        target_record: &[u8],
    ) -> Result<Vec<u8>> {
        let mut base = vec![0u8; 320 * 200];
        let actor_index = scene.actor_index;
        self.render_physical_battle_panel(scene, actor_record, &mut base, actor_index)?;
        let mut bg = scene.background_frame.clone();
        (bg.x, bg.y) = (0, 50);
        bg.blit(&mut base, 320, -1)?;
        if !scene.selection.has_separate_backgrounds {
            let target_index = scene.target_index;
            self.render_physical_battle_panel(scene, target_record, &mut base, target_index)?;
        }
        Ok(base)
    }

    /// Go：`native_physical_scene.go:211`，`func (g *Game) prepareNativePhysicalPrelude(scene *nativePhysicalScene, actor, target *battle.Unit, attack *figani.Animation, bg, tai fdother.Frame) error`
    pub(crate) fn prepare_native_physical_prelude(
        &mut self,
        scene: &mut NativePhysicalScene,
        actor: UnitIdx,
        target: UnitIdx,
        attack: &figani::Animation,
        bg: &fdother::Frame,
        tai: &fdother::Frame,
    ) -> Result<()> {
        let _ = attack; // 只有分離背景的出發段（`prepareNativePhysicalDeparture`，R8）用到。
        scene.background_frame = bg.clone();
        let st = self.st.as_ref().expect("Go：g.st 為 nil 時解參考 panic");
        scene.actor_record = battle::native_battle_panel_record_for_unit(Some(&st.units[actor]))?;
        scene.target_record = battle::native_battle_panel_record_for_unit(Some(&st.units[target]))?;
        scene.actor_index = native_command24_runtime_unit_index(Some(st), Some(actor))?;
        scene.target_index = native_command24_runtime_unit_index(Some(st), Some(target))?;
        let (actor_fig, target_fig, raw_side) =
            (st.units[actor].battle_fig, st.units[target].battle_fig, st.units[actor].native_record_byte6);
        scene.panel_assets = battle::load_native_item_panel_data_assets(&separated_asset_path(""))?;
        let (actor_record, target_record) = (scene.actor_record.clone(), scene.target_record.clone());
        let base = self.native_physical_base(scene, &actor_record, &target_record)?;
        let actor_idle = figani::load_separated_resource(&separated_asset_path("animations"), actor_fig * 3)?;
        let target_idle = figani::load_separated_resource(&separated_asset_path("animations"), target_fig * 3)?;
        if actor_idle.frames.is_empty() || target_idle.frames.is_empty() {
            return Err(game_err!("native physical prelude idle unavailable"));
        }
        let (dac, _) = load_native_battle_palette()?;
        let mode = u8::from(scene.selection.has_separate_backgrounds);
        let prelude = battlepresent::build_native_command_prelude_frames(&battlepresent::NativeCommandPreludeInput {
            base: &base,
            actor_idle: actor_idle.frames[0].clone(),
            first_target_idle: target_idle.frames[0].clone(),
            platform: Some(tai),
            raw_side,
            mode,
            baseline_dac: &dac,
        });
        scene.prelude_base = base;
        scene.prelude = prelude?;
        scene.prelude_images = native_command24_prelude_images(&scene.prelude)?;
        if !scene.selection.has_separate_backgrounds {
            return Ok(());
        }
        todo!(
            "R8: native_physical_scene.go:292 prepareNativePhysicalDeparture（分離背景的出發與捲動；第一章沒有執行到）"
        )
    }

    /// Go：`native_physical_scene.go:372`，`func (g *Game) drawNativePhysicalPrelude(screen *ebiten.Image) bool`
    ///
    /// R5 階段 2 r5b：逐行。`preludeDrawn`、body 的 `drawn` 與畫不出來時的 `loadErr` 照 Go 在這裡設（`draw` 路徑）；
    /// 無畫面重播由 `present.rs` 的呈現回報做同樣的狀態改寫。
    ///
    /// Go 以 `scene := g.atk.nativeScene` 指標同時讀寫場景與 `g`；Rust 在本函式期間把場景從 `self.atk` 暫時取出，
    /// 結束前放回。期間呼叫的 `drawModernHDBattlePrelude`／`drawNativePhysicalBody` 只經參數讀場景，不讀
    /// `g.atk.nativeScene`。
    ///
    /// 出發與捲動（`leadFrame < len(leadImages)`）在開場與第一章沒有走到（階段 0 覆蓋率 0 次；`leadImages` 由
    /// `prepareNativePhysicalDeparture` 建立，Rust 是 R8），`todo!("r8:…")`。
    pub(crate) fn draw_native_physical_prelude(&mut self, screen: &mut Picture) -> bool {
        let Some(mut scene) = self.atk.as_mut().and_then(|a| a.native_scene.take()) else {
            return false;
        };
        // Go 的每個 `return` 寫成 `break 'draw`，離開後把場景放回。
        let drawn = 'draw: {
            let scene = &mut scene;
            let img;
            if scene.prelude_frame < scene.prelude_images.len() as i64 {
                match self.draw_modern_hd_battle_prelude(screen, scene) {
                    Err(err) => {
                        self.load_err = format!("native physical HD prelude: {err}");
                        break 'draw true;
                    }
                    Ok(true) => {
                        scene.prelude_drawn = true;
                        break 'draw true;
                    }
                    Ok(false) => {}
                }
                // Go 的 preludeImages 是 NewImageFromImage 之後的貼圖；Rust 保存索引色畫格，這裡轉成同一份畫面。
                img = Picture::from_indexed(&scene.prelude_images[scene.prelude_frame as usize]);
                scene.prelude_drawn = true;
            } else if scene.lead_frame < scene.lead_images.len() as i64 {
                todo!(
                    "r8:native_physical_scene.go:388..408 出發與捲動（分離背景；body 本體、HD lead 與 leadImages；開場與第一章沒有走到）"
                )
            } else {
                if scene.body.is_some() {
                    if let Err(err) = self.draw_native_physical_body(screen, scene) {
                        self.load_err = format!("native physical body: {err}");
                    }
                    break 'draw true;
                }
                break 'draw false;
            }
            let mut op = DrawImageOptions::default();
            op.geo_m.scale(2.0, 2.0);
            screen.draw_image(&img, &op);
            true
        };
        self.atk.as_mut().expect("Draw 期間不會清掉 g.atk").native_scene = Some(scene);
        drawn
    }
}

#[cfg(test)]
#[path = "native_physical_scene/native_physical_scene_test.rs"]
pub(crate) mod native_physical_scene_test;
