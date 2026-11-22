//! 外殼的公開入口（R5 階段 4，r5w，Rust 新增，沒有單一 Go 檔對應）。
//!
//! 對應 Go 主程式 `main()`（`main.go:13466`）建立的主迴圈 runner `modernHDPresentation` 在忠實模式下的子集：
//! `Update`（`modern_hd_presentation.go:35`）、`Layout`（`player_toolbar.go:154`）、`Draw`（`player_toolbar.go:158`）。
//! 另外把 Ebiten 每次 `Update` 之前做的兩件事放在這裡：`inpututil` 的每幀狀態（[`EbitenInputState`]）與虛擬時鐘
//! （[`VirtualFrameClock`]，取代 Go 的牆鐘）。視窗、貼圖、實體鍵盤與觸控的讀取、`SetTPS` 的排程是 `fd2-app` 的事。
//!
//! # 與 Go runner 的差異
//!
//! - 工具列、速度選單、海報與數值編輯器（`toolbar`、`speedMenuOpen`、`poster`、`balanceUI`）在 R6：runner 以
//!   `toolbar == false` 的形狀執行，`toolbarOffset`／`posterOffset` 都是 0，所以 `Layout` 是 640×400、`Draw` 直接畫進
//!   畫面（`player_toolbar.go:159`）、`playerViewportLeft／Top` 是 0、`playerToolbarTouchBlocked` 是 false。數值編輯器已開啟、
//!   F10 要求切換海報時停在 `todo!("r6:…")`。
//! - HD 呈現（`hdPresentationEnabled`）改變視窗大小的部分在 R6；忠實模式下它是 false。
//!
//! - 工具列在 R6，所以 Go `sampleInput` 的 `gameInput` 只剩「設定面板沒開」一個條件（海報與數值編輯器不會吃掉輸入）。
//!
//! 逐項說明見 `PORTING.md`「R5 階段 4：r5w」與「R5 階段 4：r5i」（滑鼠、手把與按鍵設定面板）。

use super::*;

/// 外殼每次 `Update` 的結果。
#[derive(Debug, Clone, PartialEq, Eq)]
pub enum RunnerStep {
    /// 繼續執行。
    Continue,
    /// Go `Update` 回傳 `ebiten.Termination`：正常結束。
    Terminated,
}

/// 外殼的主迴圈 runner（Go `modernHDPresentation` 的忠實模式子集）。
pub struct PlayerRunner {
    game: Game,
    clock: VirtualFrameClock,
    input: EbitenInputState,
    /// R5 階段 4（r5i）：滑鼠右鍵與手把按鈕的每次 `Update` 計時（Go 由 `inpututil` 負責）。
    devices: DeviceInputState,
    /// Go `modernHDPresentation.appliedPlayerSpeed`：最後一次 `SetTPS(60 * speed)` 的 speed。
    applied_player_speed: i64,
    /// R5 階段 4（r5i）：Go `modernHDPresentation.controlsSurface`（1920×1200 疊層）與畫它時的面板狀態；狀態沒變就不重畫。
    controls_surface: Option<(String, Picture)>,
}

impl PlayerRunner {
    /// Go `main()` 的 `loadGame()` 到 `ebiten.SetTPS(60 * normalizePlayerSpeed(g.playerSpeed))`（`main.go:13484`..`13493`）。
    /// `applyPackagedPlayerDefaults`（設 `FD2_CAMPAIGN`）要在呼叫之前由外殼處理：工作區禁用 `unsafe`，不能在行程內改環境變數。
    pub fn load() -> PlayerRunner {
        let mut game = load_game().expect("loadGame 一律回傳 Some");
        // Go `main.go:13497`：`g.input = newPlayerInput(loadControlsSettings())`、`activeInput = g.input`（r5i）。
        let (bindings, status) = load_controls_settings();
        game.player_input = Some(PlayerInput::new(bindings, status));
        let applied_player_speed = normalize_player_speed(game.player_speed);
        PlayerRunner {
            game,
            clock: VirtualFrameClock::default(),
            input: EbitenInputState::default(),
            devices: DeviceInputState::default(),
            applied_player_speed,
            controls_surface: None,
        }
    }

    /// Go `g.loadErr`：載入或執行期錯誤（空字串表示沒有）。
    pub fn load_error(&self) -> &str {
        &self.game.load_err
    }

    /// Go `ebiten.SetTPS` 目前的值：`60 × appliedPlayerSpeed`。
    pub fn ticks_per_second(&self) -> i64 {
        VIRTUAL_TICKS_PER_SECOND * self.applied_player_speed
    }

    /// Go `Layout`（`player_toolbar.go:154`）：沒有工具列與海報時是 640×400。
    pub fn layout(&self) -> (i64, i64) {
        (LOGICAL_W, LOGICAL_H)
    }

    /// 一次 Ebiten `Update`：先推進 `inpututil` 與虛擬時鐘，再照 Go `modernHDPresentation.Update` 的忠實模式子集執行。
    ///
    /// r5i：Go 的 `activeInput`（輸入對應層）在這裡打開：`Game::tick_input` 是 [`PlayerInput::logical_tick`] 換算後的
    /// 邏輯鍵，實體的那一份留在 `tick_input.physical`。設定面板開著時吃掉這一次 `Update`（`updateControlsPanel`）。
    pub fn update(&mut self, raw: &RawInputState) -> std::result::Result<RunnerStep, String> {
        let physical = self.input.update(raw);
        let devices = self.devices.update(&raw.devices, &physical);
        self.game.tick_input = physical.clone();
        let now = self.game.advance_virtual_frame(&mut self.clock);
        // modern_hd_presentation.go:36
        self.game.sync_player_clock_speed_at(now);
        // modern_hd_presentation.go:38..41（r5i）
        if self.game.update_controls_panel(now, &physical, &devices) {
            self.game.sample_input(false, &devices.snapshot);
            return Ok(RunnerStep::Continue);
        }
        if self.game.balance_ui.open {
            todo!("r6:player_balance.go:412 updateBalanceEditor（數值編輯器由工具列開啟，R6）")
        }
        // modern_hd_presentation.go:57：沒有海報選單與速度選單，`playerOverlayBlock` 是空矩形。
        self.game.player_overlay_block = Rectangle::default();
        // modern_hd_presentation.go:70：`poster.consumeInput` 與 `balanceUI.open` 都是 false，面板沒開（上面已返回）。
        self.game.sample_input(true, &devices.snapshot);
        if let Some(input) = self.game.player_input.as_ref() {
            self.game.tick_input = input.logical_tick(physical);
        }
        // modern_hd_presentation.go:71
        if let Err(err) = self.game.update() {
            if err.is(&ERR_TERMINATION) {
                return Ok(RunnerStep::Terminated);
            }
            return Err(err.message().to_string());
        }
        if self.game.player_poster_cycle_requested {
            // modern_hd_presentation.go:68
            todo!("r6:player_poster.go:116 cyclePosters（F10 海報）")
        }
        // modern_hd_presentation.go:77..84
        self.game.sync_player_clock_speed_at(now);
        let speed = self.game.effective_player_speed();
        if self.applied_player_speed != speed {
            self.applied_player_speed = speed;
        }
        Ok(RunnerStep::Continue)
    }

    /// Go `Draw`（`player_toolbar.go:158`）：沒有工具列與海報時直接 `Game.Draw(screen)`。`screen` 是 Ebiten 每幀清空的
    /// 640×400 畫布（`IsScreenClearedEveryFrame` 預設為 true，`internal/ui/context.go:191`），呼叫端每個顯示幀給一張全透明的。
    pub fn draw(&mut self, screen: &mut Picture) {
        self.game.draw(screen);
    }

    /// 保留 HD 世界或開場的原尺寸。呼叫端把這層放在 640×400 介面下方。
    pub fn draw_window_layers(&mut self, ui: &mut Picture) -> Option<Arc<Picture>> {
        self.game.modern_hd_separate_world = true;
        self.game.draw(ui);
        self.game.modern_hd_separate_world = false;
        if let Some(opening) = &self.game.modern_opening_view {
            return Some(opening.clone());
        }
        self.game.modern_hd_world_visible.then(|| self.game.modern_hd_map_gpu.as_ref()?.world.clone()).flatten()
    }

    pub fn has_hd_window_layers(&self) -> bool {
        self.game.hd_title_interface_picture().is_some()
            || self.game.modern_story_portraits.as_ref().is_some_and(|set| {
                set.map_sprites_hd.len() == 125
                    || (self.game.modern_opening.is_some() && !self.game.title_phase.is_empty())
            })
    }

    pub fn window_portrait(&self) -> Option<WindowPortrait> {
        if let Some(image) = self.game.hd_title_interface_picture() {
            return Some(WindowPortrait {
                image: image.clone(),
                rect: title_interface_rect(),
                brightness: 1.0,
                mirror_x: false,
            });
        }
        self.game.window_portrait()
    }

    /// 交給外殼同步播放，並接收一次性音效的播放完成狀態。
    pub fn audio_voices_mut(&mut self) -> Vec<(VoiceSlot, &mut AudioVoice)> {
        self.game.audio_voices_mut()
    }

    /// R5 階段 4（r5i）：設定面板開著時的 1920×1200 疊層（Go `drawControlsPanel` 畫進 `controlsSurface` 的內容；外殼以線性
    /// 濾鏡縮成內容區大小疊上去，Go `op.GeoM.Scale(1.0/3, 1.0/3)`＋`Concat(geometry)`）。面板沒開時回 `None`。面板狀態
    /// 沒變時沿用上次畫好的疊層；第一個回傳值是決定內容的面板狀態（外殼據此判斷要不要重新上傳貼圖）。
    pub fn controls_panel_surface(&mut self) -> Option<(&str, &Picture)> {
        if !self.game.controls.open {
            self.controls_surface = None;
            return None;
        }
        let key = self.game.controls_panel_render_key();
        if self.controls_surface.as_ref().is_none_or(|(k, _)| *k != key) {
            let mut s = Picture::new_image(CONTROLS_SURFACE_W, CONTROLS_SURFACE_H);
            self.game.draw_controls_panel_surface(&mut s, None, true);
            self.controls_surface = Some((key, s));
        }
        self.controls_surface.as_ref().map(|(k, s)| (k.as_str(), s))
    }

    /// Alt+Enter（`main.go:8390`）或工具列的全螢幕請求：Go 直接 `ebiten.SetFullscreen(!ebiten.IsFullscreen())`，Rust 由外殼
    /// 讀取後清除。
    pub fn take_fullscreen_request(&mut self) -> bool {
        std::mem::take(&mut self.game.fullscreen_toggle_requested)
    }

    /// 診斷用：Go `g.frame`（每次 `Update` 加 1）。
    pub fn game_frame(&self) -> i64 {
        self.game.frame
    }

    /// 診斷用：標題階段（Go `g.titlePhase`）與戰役節點 ID（Go `g.camp.NodeID()`，沒有戰役時是空字串）。
    pub fn progress(&self) -> (String, String) {
        let node = self.game.camp.as_ref().map(|c| c.node_id()).unwrap_or_default();
        (self.game.title_phase.clone(), node)
    }
}
