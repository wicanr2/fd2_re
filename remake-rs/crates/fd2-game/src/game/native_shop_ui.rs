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
