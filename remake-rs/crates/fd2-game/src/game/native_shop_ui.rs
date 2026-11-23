//! Go 來源：`remake/cmd/fd2/native_shop_ui.go`（2228 行）。
//!
//! 骨架：Go 檔的 52 個函式中產生 7 個（階段 0 清單），其餘標 `r8`，見 `PORTING.json`。
//! E 組另補移植 `stepNativeShopUIPulseTick`（`r8`，`stepNativeShopUILifecycle` 在商店主選單每幀呼叫）。
//! 商店的購買、販賣、裝備與轉移交易都在 `r8` 函式（`handleNativeShopInput` 等），第一章重播沒有執行到。

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

/// Go `map[int]int{1: 12, 3: 29, 5: 63}`（變體 → FDOTHER 資源）。
const NATIVE_SHOP_RESOURCES: [(i64, i64); 3] = [(1, 12), (3, 29), (5, 63)];
/// Go `map[int]int{1: 0x80, 3: 0x82, 5: 0x84}`（變體 → DATO 頭像）。
const NATIVE_SHOP_PORTRAITS: [(i64, i64); 3] = [(1, 0x80), (3, 0x82), (5, 0x84)];

/// Go：`native_shop_ui.go:1040`，`func loadNativeShopUIAssets( shared *nativeClassUIAssets, ) (*nativeShopUIAssets, error)`
///
/// Go 以 `range` map 載入三個變體（順序隨機）；Rust 依變體排序。只影響同時有多個缺件時回報哪一個。
pub(crate) fn load_native_shop_ui_assets(shared: Option<&NativeClassUIAssets>) -> Result<Option<NativeShopUIAssets>> {
    if !shared.is_some_and(|s| {
        s.units.is_some() && s.strings.is_some() && s.font.is_some() && s.dialogue.len() > 17 && s.digits.len() == 10
    }) {
        return Err(game_err!("native shop UI: shared facility assets unavailable"));
    }
    let pack_root = separated_asset_path("");
    let mut out = NativeShopUIAssets::default();
    for (variant, resource_id) in NATIVE_SHOP_RESOURCES {
        let assets = campaign::load_separated_native_shop_assets(&pack_root, resource_id)?;
        out.shops.insert(variant, assets);
    }
    for (variant, portrait_id) in NATIVE_SHOP_PORTRAITS {
        let frames = load_native_separated_portrait(portrait_id)?;
        let Some(first) = frames.into_iter().next() else {
            return Err(game_err!("native shop UI: DATO portrait has no frames"));
        };
        out.portraits.insert(variant, first);
    }
    out.item_assets = battle::load_native_item_panel_data_assets(&separated_asset_path(""))?;
    out.effect_rows =
        battle::load_native_item_effect_row_prefix(&asset_path("assets/data/native_item_effect_rows.json"))?;
    Ok(Some(out))
}

impl Game {
    /// Go：`native_shop_ui.go:1084`，`func (g *Game) setupNativeShop() bool`
    pub(crate) fn setup_native_shop(&mut self) -> bool {
        // Go 直接解參考 `g.camp`（呼叫端 `enterNode` 已確認 campaign 存在）。
        let Some(n) = self.camp.as_ref().expect("setupNativeShop: g.camp is nil").node() else {
            return false;
        };
        if n.r#type != "shop" || n.native_hub_variant == 0 {
            return false;
        }
        let variant = n.native_hub_variant;
        let Some(ui) = self.native_shop_ui.as_ref() else {
            return false;
        };
        if !ui.shops.contains_key(&variant) || !ui.portraits.contains_key(&variant) {
            return false;
        }
        self.native_shop_variant = variant;
        self.native_shop_mode = String::from("menu");
        self.native_shop_service_sel = 0;
        self.native_shop_item_start = 0;
        self.native_shop_confirm_sel = 0;
        self.shop_sel = 0;
        if variant == 5 {
            self.msg.clear();
        }
        self.reset_native_shop_ui_pulse();
        self.begin_native_shop_service_opening()
    }

    /// Go：`native_shop_ui.go:1106`，`func (g *Game) nativeShopState() ( *campaign.NativeShopAssets, dato.Frame, int, bool, )`
    ///
    /// 簽名改寫：第一個回傳值借用 `native_shop_ui` 裡的素材（Go 回傳指向它的指標）。
    pub(crate) fn native_shop_state(&self) -> (Option<&campaign::NativeShopAssets>, dato::Frame, i64, bool) {
        let Some(ui) = self.native_shop_ui.as_ref() else {
            return (None, dato::Frame::default(), 0, false);
        };
        if self.native_class_ui.is_none() || !matches!(self.native_shop_variant, 1 | 3 | 5) {
            return (None, dato::Frame::default(), 0, false);
        }
        let assets = ui.shops.get(&self.native_shop_variant);
        let portrait = ui.portraits.get(&self.native_shop_variant);
        let portrait_id =
            NATIVE_SHOP_PORTRAITS.iter().find(|(v, _)| *v == self.native_shop_variant).map_or(0, |(_, id)| *id);
        let ok = assets.is_some() && portrait.is_some();
        (assets, portrait.cloned().unwrap_or_default(), portrait_id, ok)
    }

    /// Go：`native_shop_ui.go:1120`，`func (g *Game) composeNativeShopStable() ([]byte, bool)`
    pub(crate) fn compose_native_shop_stable(&mut self) -> (Vec<u8>, bool) {
        let (assets, portrait, portrait_id, ok) = self.native_shop_state();
        if !ok {
            return (Vec::new(), false);
        }
        let text_index = if self.native_shop_variant == 1 { 501 } else { 440 };
        if (!self.locale_id.is_empty() && self.locale_id != "zh-Hant") || self.modern_hd_shop_text() {
            let assets = assets.cloned().unwrap_or_default();
            return match self.compose_localized_native_shop_stable(&assets, &portrait, portrait_id) {
                Ok(frame) => (frame, true),
                Err(_) => (Vec::new(), false),
            };
        }
        // `native_shop_state` 成功表示 `native_class_ui` 存在。
        let shared = self.native_class_ui.as_ref().unwrap();
        match campaign::compose_native_shop_scene(
            assets,
            &shared.dialogue,
            &shared.digits,
            &portrait,
            portrait_id,
            shared.strings.as_ref(),
            shared.font.as_ref(),
            self.gold,
            text_index,
        ) {
            Ok(frame) => (frame, true),
            Err(_) => (Vec::new(), false),
        }
    }

    /// Go：`native_shop_ui.go:1312`，`func (g *Game) beginNativeShopServiceOpening() bool`
    pub(crate) fn begin_native_shop_service_opening(&mut self) -> bool {
        let ok = self.native_shop_state().3;
        let (stable, stable_ok) = self.compose_native_shop_stable();
        if !ok || !stable_ok {
            return false;
        }
        let assets = self.native_shop_state().0;
        let mut frames = Vec::with_capacity(4);
        for step in 0..4 {
            match campaign::compose_native_shop_service_opening_frame(&stable, assets, step) {
                Ok(frame) => frames.push(frame),
                Err(_) => return false,
            }
        }
        self.reset_native_shop_ui_pulse();
        let mut job = NativeClassUIJob { frames, ..Default::default() };
        job.service_menu = self.bind_modern_facility_service_menu(Some(&job), None, false).map(Box::new);
        self.native_shop_ui_job = Some(job);
        true
    }

    /// Go：`native_shop_ui.go:1452`，`func (g *Game) stepNativeShopUILifecycle(now time.Time)`
    pub(crate) fn step_native_shop_ui_lifecycle(&mut self, now: GameInstant) {
        if self.native_shop_mode == "equip_panel" {
            todo!("R8: native_shop_ui.go:stepNativeShopEquipPanelLifecycle（購買後的裝備面板）");
        }
        if let Some(job) = self.native_shop_ui_job.as_mut()
            && !job.timeline.is_empty()
        {
            if self.native_shop_variant == 5 {
                if job.started.is_zero() {
                    job.started = now;
                }
                if job.drawn && now - job.started >= job.timeline[job.frame as usize].duration {
                    job.frame += 1;
                    job.drawn = false;
                    job.started = now;
                    if job.frame as usize == job.timeline.len()
                        && let Some(after) = self.native_shop_ui_job.take().and_then(|job| job.after)
                    {
                        after(self);
                    }
                }
                return;
            }
            if job.started.is_zero() {
                job.started = now;
            }
            job.elapsed = now - job.started;
            if job.frame == 1 && job.drawn {
                let mut job = self.native_shop_ui_job.take().unwrap();
                if let Some(after) = job.after.take() {
                    after(self);
                }
            }
            return;
        }
        if let Some(job) = self.native_shop_ui_job.as_mut()
            && job.drawn
        {
            job.drawn = false;
            let frames = job.frames.len() as i64;
            if job.frame < frames {
                job.frame += 1;
                if job.frame < frames || !job.restore.is_empty() {
                    return;
                }
            }
            if job.frame >= frames {
                let mut job = self.native_shop_ui_job.take().unwrap();
                if let Some(after) = job.after.take() {
                    after(self);
                }
            }
        }
        if self.native_shop_ui_job.is_none()
            && matches!(self.native_shop_mode.as_str(), "menu" | "confirm" | "equip_confirm" | "sell_confirm")
        {
            let tick = self.native_shop_ui_clock.sample(now);
            self.step_native_shop_ui_pulse_tick(tick);
        }
    }

    /// Go：`native_shop_ui.go:1617`，`func (g *Game) resetNativeShopUIPulse()`
    pub(crate) fn reset_native_shop_ui_pulse(&mut self) {
        self.native_shop_ui_clock.reset();
        self.native_shop_ui_pulse = 2;
        self.native_shop_ui_last_tick = 0;
        self.native_shop_ui_has_tick = false;
    }

    /// Go：`native_shop_ui.go:1624`，`func (g *Game) stepNativeShopUIPulseTick(rawTick int)`（`r8`，E 組隨
    /// `stepNativeShopUILifecycle` 補移植）
    pub(crate) fn step_native_shop_ui_pulse_tick(&mut self, raw_tick: i64) {
        if !self.native_shop_ui_has_tick {
            self.native_shop_ui_last_tick = raw_tick;
            self.native_shop_ui_has_tick = true;
            return;
        }
        let delta = (raw_tick as u16).wrapping_sub(self.native_shop_ui_last_tick as u16) as i16;
        if delta < 2 {
            return;
        }
        self.native_shop_ui_last_tick = raw_tick;
        self.native_shop_ui_pulse = (self.native_shop_ui_pulse + 1) & 3;
    }
}

#[cfg(test)]
mod native_shop_ui_test;

// ---- R5 階段 1 骨架（tools/rust_r5_skeleton.sh 產生；本體由 R5 階段 2 移植）----

impl Game {
    /// Go：`native_shop_ui.go:1495`，`func (g *Game) drawNativeShopUIJob(screen *ebiten.Image) bool`
    ///
    /// R5 階段 2 r5c。HD 路徑（`drawModernHDFacilityServiceMenu`、`drawModernHDFacilityConfirmation`、
    /// `drawModernHDFacilityList`、`drawModernHDShopFrame`，R6）在忠實模式都回 false：前三個要 HD 描述（Rust 在忠實
    /// 模式不建立），`drawModernHDShopFrame` 要 `modernFacility` 場景。描述或 HD 主題存在時 `todo!("r6:…")`；
    /// `drawModernHDFacilityList` 開頭的 `facilityListRosterHD = nil` 照做。timeline（購買、出售成功演出）在開場與
    /// 第一章沒有走到（`todo!("r8:…")`）。
    pub(crate) fn draw_native_shop_ui_job(&mut self, screen: &mut Picture) -> bool {
        let Some(job) = self.native_shop_ui_job.as_ref() else {
            return false;
        };
        if job.service_menu.is_some() && self.native_shop_variant != 5 {
            todo!("r6:modern_hd_service_menu_transition.go:drawModernHDFacilityServiceMenu");
        }
        if job.church_confirmation.is_some() {
            todo!("r6:modern_hd_church_confirmation_transition.go:drawModernHDFacilityConfirmation");
        }
        self.facility_list_roster_hd = None;
        let job = self.native_shop_ui_job.as_ref().unwrap();
        if job.facility_list.is_some() {
            todo!("r6:modern_hd_facility_list_transition.go:drawModernHDFacilityList");
        }
        if !job.timeline.is_empty() {
            if self.native_shop_variant == 5 {
                let frame = job.timeline[job.frame as usize].frame.clone();
                self.present_native_class_frame(screen, &frame);
                self.native_shop_ui_job.as_mut().unwrap().drawn = true;
                return true;
            }
            todo!("r8:native_shop_ui.go:1512..1525 timeline（購買／出售成功演出；開場與第一章沒有走到）");
        }
        let pix = if job.frame < job.frames.len() as i64 {
            // Go 以負的 frame 索引切片會 panic；Rust 同樣 panic。
            job.frames[usize::try_from(job.frame).expect("drawNativeShopUIJob：frame 為負")].clone()
        } else if job.restore.len() == 320 * 200 {
            job.restore.clone()
        } else {
            return false;
        };
        if self.modern_story_portraits.is_some() && self.native_shop_variant != 5 {
            todo!("r6:modern_hd_facility_scene.go:drawModernHDShopFrame");
        }
        self.present_native_class_frame(screen, &pix);
        self.native_shop_ui_job.as_mut().unwrap().drawn = true;
        true
    }

    pub(crate) fn handle_native_secret_shop_input(&mut self, input: &TickInput, enter: bool) -> bool {
        if self.native_shop_ui_job.is_some() {
            return true;
        }
        if !matches!(self.native_shop_mode.as_str(), "menu" | "purchase") {
            self.handle_native_shop_purchase(input, enter);
            return true;
        }
        let just = |key| input.is_just_pressed(key);
        if just(Key::Escape) {
            if self.native_shop_mode == "purchase" {
                self.native_shop_mode = "menu".into();
                self.native_shop_service_sel = 0;
                self.msg.clear();
                self.begin_native_shop_service_opening();
            } else {
                self.leave_shop();
            }
            return true;
        }
        if self.native_shop_mode == "menu" {
            if just(Key::ArrowLeft) || just(Key::ArrowRight) {
                let delta = if just(Key::ArrowLeft) { -1 } else { 1 };
                self.native_shop_service_sel = (self.native_shop_service_sel + delta).rem_euclid(4);
                self.reset_native_shop_ui_pulse();
            }
            if enter
                && self.native_shop_service_sel == 0
                && self.camp.as_ref().is_some_and(|camp| !camp.shop_goods().is_empty())
            {
                self.native_shop_mode = "purchase".into();
                self.shop_sel = 0;
                self.native_shop_item_start = 0;
                self.msg.clear();
            } else if enter {
                self.msg = "此項商店服務尚未移植".into();
            }
        } else if self.native_shop_mode == "purchase" {
            let len = self.camp.as_ref().map_or(0, |camp| camp.shop_goods().len()) as i64;
            let step = [(Key::ArrowLeft, -1), (Key::ArrowRight, 1), (Key::ArrowUp, -2), (Key::ArrowDown, 2)]
                .into_iter()
                .find_map(|(key, step)| just(key).then_some(step))
                .unwrap_or(0);
            let next = self.shop_sel + step;
            if (0..len).contains(&next) {
                self.shop_sel = next;
            }
            if enter {
                self.native_shop_mode = "confirm".into();
                self.native_shop_confirm_sel = 0;
                self.reset_native_shop_ui_pulse();
                self.msg.clear();
                if self.cutscene_log {
                    eprintln!("[shop] purchase confirmation index={}", self.shop_sel);
                }
            }
        }
        true
    }

    fn draw_native_secret_shop(&mut self, screen: &mut Picture) -> bool {
        let (stable, valid) = self.compose_native_shop_stable();
        if !valid {
            return false;
        }
        let (assets, _, _, _) = self.native_shop_state();
        let frame = if self.native_shop_mode == "purchase" {
            let Some(camp) = self.camp.as_ref() else { return false };
            let Some(ui) = self.native_shop_ui.as_ref() else { return false };
            let ids: Vec<_> = camp.shop_goods().iter().map(|good| good.id).collect();
            let (start, _) =
                campaign::native_two_column_window(ids.len() as i64, self.shop_sel, self.native_shop_item_start);
            let frame = campaign::compose_native_shop_item_list_frame(
                &stable,
                assets,
                &ui.item_assets,
                &ids,
                start,
                self.shop_sel,
                &ui.effect_rows,
                battle::NATIVE_FACILITY_FULL_PRICE,
            );
            self.native_shop_item_start = start;
            frame.ok()
        } else if self.native_shop_mode == "menu" {
            campaign::compose_native_shop_service_steady_frame(
                &stable,
                assets,
                self.native_shop_service_sel,
                self.native_shop_ui_pulse,
            )
            .ok()
        } else {
            self.compose_native_secret_purchase(&stable).ok()
        };
        self.present_native_class_frame(screen, &frame.unwrap_or(stable));
        if !self.msg.is_empty()
            && let Some(font) = self.font.as_mut()
        {
            font.draw(screen, &self.msg, 18.0, 370.0, 0.9, ColorRgba { r: 255, g: 240, b: 182, a: 255 });
        }
        true
    }

    /// Go：`native_shop_ui.go:1543`，`func (g *Game) drawNativeShop(screen *ebiten.Image) bool`
    ///
    /// R5 階段 2 r5c。第一章只走到開關框工作（`drawNativeShopUIJob`）；各模式的穩態合成（`composeNativeShop*`）
    /// 在 crate 裡還沒有，開場與第一章也沒有畫到，`todo!("r8:…")`。`modernHDText` 存在時的 HD 文字擷取 `todo!("r6:…")`。
    pub(crate) fn draw_native_shop(&mut self, screen: &mut Picture) -> bool {
        if self.native_shop_mode.is_empty() {
            return false;
        }
        if self.draw_native_shop_ui_job(screen) {
            return true;
        }
        if self.native_shop_variant == 5 {
            return self.draw_native_secret_shop(screen);
        }
        if self.modern_hd_text.is_some() && self.font.is_some() {
            todo!("r6:native_shop_ui.go:1550..1554 HD 文字擷取（modernHDIndexedText、beginModernHDShopDigits）");
        }
        todo!(
            "r8:native_shop_ui.go:1555..1610 商店穩態畫面（composeNativeShop*；開場與第一章沒有走到）, mode={}",
            self.native_shop_mode
        )
    }
}
