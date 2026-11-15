//! Go 來源：`remake/cmd/fd2/native_town_ui.go`（234 行）。
//!
//! 骨架：Go 檔的 16 個函式中產生 11 個（階段 0 清單），其餘標 `r8`，見 `PORTING.json`。
//! E 組另補移植 `captureNativeTownCaller`、`revealNativeTownSecret`（`r8`，`campInput` 的城鎮分支呼叫）。
//! `drawNativeTown`（R5）不移植：它只呼叫 `composeNativeTownFrame` 再把畫面交給外殼。

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
    ///
    /// 簽名改寫：`&self`（Go 不改寫 `g`）。Go 的 `strings`／`font` 是指標，Rust 存複本。
    pub(crate) fn snapshot_native_town_source(&self) -> Option<NativeTownFrameSource> {
        let camp = self.camp.as_ref()?;
        let scene = self.native_town_ui.as_ref()?.scene.as_ref()?;
        let class_ui = self.native_class_ui.as_ref()?;
        let n = camp.node()?;
        if n.r#type != "town" {
            return None;
        }
        let variant = n.native_town_variant?;
        let mut d = NativeTownFrameSource {
            scene: scene.clone(),
            strings: class_ui.strings.clone(),
            font: class_ui.font.clone(),
            variant,
            selection: self.camp_sel,
            pulse: self.native_town_ui_pulse,
            ..Default::default()
        };
        campaign::native_town_cursor_geometry_for(d.variant, d.selection, d.pulse).ok()?;
        if let Some(first) = self.party_join_order.first() {
            let unit = self.party_roster.get(first)?;
            if unit.has_map_selector_key {
                let units = class_ui.units.as_ref()?;
                d.group = unit.map_selector_key;
                for cycle in 0..d.scene.pulse.len() {
                    let sprite = units.sprite_for(d.group, 0, cycle as i64).ok()?;
                    d.scene.pulse[cycle] = sprite.clone();
                }
            }
        }
        Some(d)
    }
}

impl NativeTownFrameSource {
    /// Go：`native_town_ui.go:62`，`func (d *nativeTownFrameSource) compose(fill *byte, cursor bool, font *fdtxt.Font) ([]byte, bool)`
    /// Go 檢查接收者是否為 nil；Rust 的 `self` 不會是 null，該分支不移植。
    ///
    /// 簽名改寫：`&self`（Go 複製 `scene` 後才改背景）；`fill` 是值（Go 只讀 `*fill`）；`font` 可為 nil。
    pub(crate) fn compose(&self, fill: Option<u8>, cursor: bool, font: Option<&fdtxt::Font>) -> (Vec<u8>, bool) {
        let mut scene = self.scene.clone();
        if let Some(fill) = fill {
            scene.backgrounds[self.variant as usize] = vec![fill; 320 * 200];
        }
        let compose = if cursor {
            campaign::compose_native_town_frame
        } else {
            campaign::compose_native_town_frame_without_cursor
        };
        match compose(Some(&scene), self.strings.as_ref(), font, self.variant, self.selection, self.pulse) {
            Ok(frame) => (frame, true),
            Err(_) => (Vec::new(), false),
        }
    }
}

impl Game {
    /// Go：`native_town_ui.go:78`，`func (g *Game) captureNativeTownCaller()`（`r8`，E 組隨 `campInput` 補移植）
    pub(crate) fn capture_native_town_caller(&mut self) {
        self.prep_prompt_source = Vec::new();
        self.prep_town_source = None;
        let Some(mut d) = self.snapshot_native_town_source() else {
            return;
        };
        let (frame, ok) = d.compose(None, true, d.font.as_ref());
        if !ok {
            return;
        }
        d.indexed = frame.clone();
        self.prep_prompt_source = frame;
        self.prep_town_source = Some(d);
    }
}

/// nativeTownMoveSelection preserves 0x2cf01..0x2cf65: right decrements and
/// wraps normal 0 to 4; left increments and wraps every value above 4 to 0.
/// This also defines how the revealed hidden selection 5 returns to the five
/// normal facilities.
///
/// Go：`native_town_ui.go:130`，`func nativeTownMoveSelection(selection, delta int) (int, bool)`
pub(crate) fn native_town_move_selection(selection: i64, delta: i64) -> (i64, bool) {
    if !(0..=5).contains(&selection) || (delta != -1 && delta != 1) {
        return (0, false);
    }
    let mut selection = selection + delta;
    if selection < 0 {
        selection = 4;
    }
    if selection > 4 {
        selection = 0;
    }
    (selection, true)
}

impl Game {
    /// moveNativeTownSelection changes only the selector. The original
    /// 0x2ce7a/0x2ceac branches do not write the shared pulse counter at 0x54133.
    ///
    /// Go：`native_town_ui.go:146`，`func (g *Game) moveNativeTownSelection(delta int) bool`
    pub(crate) fn move_native_town_selection(&mut self, delta: i64) -> bool {
        let (selection, ok) = native_town_move_selection(self.camp_sel, delta);
        if !ok {
            return false;
        }
        self.camp_sel = selection;
        true
    }

    /// revealNativeTownSecret changes only the selector. The original 0x2cef7
    /// branch writes 5 to 0x5412b and does not touch pulse counter 0x54133.
    ///
    /// Go：`native_town_ui.go:157`，`func (g *Game) revealNativeTownSecret(scanCode int) bool`（`r8`，E 組隨 `campInput` 補移植）
    pub(crate) fn reveal_native_town_secret(&mut self, scan_code: i64) -> bool {
        let Some(camp) = self.camp.as_ref() else {
            return false;
        };
        if !camp.match_native_town_secret(self.camp_sel, scan_code) {
            return false;
        }
        self.camp_sel = 5;
        true
    }
}

/// Go：`native_town_ui.go:166`，`func loadNativeTownUIAssets() (*nativeTownUIAssets, error)`
pub(crate) fn load_native_town_ui_assets() -> Result<Option<NativeTownUIAssets>> {
    let scene = campaign::load_native_town_assets(&separated_asset_path(""))?;
    Ok(Some(NativeTownUIAssets { scene: Some(scene) }))
}

impl Game {
    /// Go：`native_town_ui.go:174`，`func (g *Game) resetNativeTownUIPulse()`
    pub(crate) fn reset_native_town_ui_pulse(&mut self) {
        self.native_town_ui_clock.reset();
        self.native_town_ui_pulse = 0;
        self.native_town_ui_last_tick = 0;
        self.native_town_ui_has_tick = false;
    }

    /// 0x2d1b5 advances the four-state counter after a signed BIOS low-word delta
    /// of four. Counter 3 is rendered with FDICON sprite 1 by the compositor.
    ///
    /// Go：`native_town_ui.go:183`，`func (g *Game) stepNativeTownUIPulseTick(rawTick int)`
    pub(crate) fn step_native_town_ui_pulse_tick(&mut self, raw_tick: i64) {
        if !self.native_town_ui_has_tick {
            self.native_town_ui_last_tick = raw_tick;
            self.native_town_ui_has_tick = true;
            return;
        }
        let delta = (raw_tick as u16).wrapping_sub(self.native_town_ui_last_tick as u16) as i16;
        if delta < 4 {
            return;
        }
        self.native_town_ui_last_tick = raw_tick;
        self.native_town_ui_pulse = (self.native_town_ui_pulse + 1) & 3;
    }

    /// Go：`native_town_ui.go:197`，`func (g *Game) stepNativeTownUILifecycle(nowTick int)`
    pub(crate) fn step_native_town_ui_lifecycle(&mut self, now_tick: i64) {
        if self.native_town_ui.is_none() {
            return;
        }
        let Some(n) = self.camp.as_ref().and_then(|c| c.node()) else {
            return;
        };
        if n.r#type != "town" || n.native_town_variant.is_none() {
            return;
        }
        self.step_native_town_ui_pulse_tick(now_tick);
    }

    /// Go：`native_town_ui.go:208`，`func (g *Game) composeNativeTownFrame() ([]byte, bool)`
    pub(crate) fn compose_native_town_frame(&self) -> (Vec<u8>, bool) {
        self.compose_native_town_frame_with_background(None)
    }

    /// Go：`native_town_ui.go:212`，`func (g *Game) composeNativeTownFrameWithBackground(fill *byte) ([]byte, bool)`
    ///
    /// 簽名改寫：`fill: Option<u8>`（Go 的 `*byte` 可為 nil、只讀）。
    pub(crate) fn compose_native_town_frame_with_background(&self, fill: Option<u8>) -> (Vec<u8>, bool) {
        self.compose_native_town_frame_layers(fill, true)
    }

    /// Go：`native_town_ui.go:216`，`func (g *Game) composeNativeTownFrameLayers(fill *byte, cursor bool) ([]byte, bool)`
    ///
    /// 簽名改寫：同上。
    pub(crate) fn compose_native_town_frame_layers(&self, fill: Option<u8>, cursor: bool) -> (Vec<u8>, bool) {
        let Some(d) = self.snapshot_native_town_source() else {
            return (Vec::new(), false);
        };
        d.compose(fill, cursor, d.font.as_ref())
    }
}

#[cfg(test)]
mod native_town_ui_test;

// ---- R5 階段 1 骨架（tools/rust_r5_skeleton.sh 產生；本體由 R5 階段 2 移植）----

impl Game {
    /// Go：`native_town_ui.go:224`，`func (g *Game) drawNativeTown(screen *ebiten.Image) bool`
    ///
    /// R5 階段 2 r5c。`drawModernHDTown`（HD，R6）要 `modernFacility` 場景，`modernStoryPortraits` 為 nil 時回 false；
    /// 有 HD 主題時 `todo!("r6:…")`。
    pub(crate) fn draw_native_town(&mut self, screen: &mut Picture) -> bool {
        if self.modern_story_portraits.is_some() {
            todo!("r6:modern_hd_facility_scene.go:drawModernHDTown");
        }
        let (frame, ok) = self.compose_native_town_frame();
        if !ok {
            return false;
        }
        self.present_native_class_frame(screen, &frame);
        true
    }
}
