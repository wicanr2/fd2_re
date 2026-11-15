//! Go 來源：`remake/cmd/fd2/native_player_input.go`（237 行）。
//!
//! 骨架：Go 檔的 8 個函式中產生 4 個（階段 0 清單），其餘標 `r8`，見 `PORTING.json`。

use super::*;

impl Game {
    /// Go：`native_player_input.go:38`，`func (g *Game) beginNativePlayerFocus(u *battle.Unit) bool`
    pub(crate) fn begin_native_player_focus(&mut self, u: Option<UnitIdx>) -> bool {
        let ok = self.st.as_ref().is_some_and(|st| st.has_native_map_view_state)
            && u.is_some_and(|u| self.st.as_ref().unwrap().units[u].has_native_map_presentation);
        if !ok {
            self.load_err = String::from("native player focus: raw position unavailable");
            return false;
        }
        let st = self.st.as_ref().unwrap();
        let p = st.units[u.unwrap()].native_map_presentation;
        if i64::from(p.x) >= st.w || i64::from(p.y) >= st.h {
            self.load_err = String::from("native player focus: position outside field");
            return false;
        }
        self.native_player_focus = Some(battle::Cell { x: i64::from(p.x), y: i64::from(p.y) });
        self.native_player_focus_started = false;
        true
    }

    /// Go：`native_player_input.go:53`，`func (g *Game) stepNativePlayerFocus()`
    pub(crate) fn step_native_player_focus(&mut self) {
        let Some(p) = self.native_player_focus else {
            return;
        };
        if self.st.is_none() {
            return;
        }
        self.step_native_player_focus_body(p);
        // Go 的 defer：任何一條 return 之後都執行。
        if self.native_player_focus.is_none() && self.native_gate_b_after_focus {
            self.restore_native_display_gate_b();
        }
    }

    /// `stepNativePlayerFocus` 在 `defer` 之後的本體（Rust 拆出，讓呼叫端在每條 return 後執行 defer）。
    fn step_native_player_focus_body(&mut self, p: battle::Cell) {
        if !self.native_player_focus_started {
            // 0x1A7AB 0x12D7B(0)→0x12CEA：開頭先 0x11CAC(0)。
            self.native_player_focus_started = true;
            let hook = battle::NATIVE_FOCUS_TRACE_HOOK.read().unwrap_or_else(|e| e.into_inner()).clone();
            if let Some(hook) = hook {
                hook(p.x, p.y, self.st.as_ref().unwrap().native_map_view_state);
            }
            self.redraw_native_map_hud();
        }
        let v = self.st.as_ref().unwrap().native_map_view_state;
        let (mut dx, mut dy) = (0, 0);
        if v.cursor_x > p.x {
            dx = -1;
        } else if v.cursor_x < p.x {
            dx = 1;
        } else if v.cursor_y > p.y {
            dy = -1;
        } else if v.cursor_y < p.y {
            dy = 1;
        } else {
            self.native_player_focus = None;
            return;
        }
        let (moved, ok) = self.st.as_mut().unwrap().move_native_map_cursor(dx, dy);
        if !ok || !moved {
            self.load_err = String::from("native player focus: cursor step rejected");
            self.native_player_focus = None;
            return;
        }
        self.sync_native_map_view();
        let after = self.st.as_ref().unwrap().native_map_view_state;
        self.native_cursor_step_hud(v, after);
        if self.cur_x == p.x && self.cur_y == p.y {
            self.native_player_focus = None;
        }
    }

    /// inspectNativePlayerUnit 的回傳值表示已消費確認鍵，包括原版拒絕的特殊角色。
    ///
    /// Go：`native_player_input.go:99`，`func (g *Game) inspectNativePlayerUnit(u *battle.Unit, force bool) (bool, error)`
    pub(crate) fn inspect_native_player_unit(&mut self, u: UnitIdx, force: bool) -> Result<bool> {
        let unit = self.st.as_ref().expect("Go：g.st 為 nil 時解參考 panic").units[u].clone();
        let records = match battle::native_ai_scoring_records(std::slice::from_ref(&unit)) {
            Ok(records) => records,
            Err(err) => return Err(GameError::from(err).wrap("native player status: ")),
        };
        let r = &records[..0x50];
        if r[7] == 0x79 || r[0x1f] == 10 {
            return Ok(true);
        }
        if !force && r[6] == 2 && r[5] & 0x80 == 0 && r[0x26] == 0 {
            return Ok(false);
        }
        todo!("R8: native_player_input.go:111 beginNativePlayerStatus（第一章重播沒有執行到）")
    }
}

/// Go：`native_player_input.go:114`
#[derive(Debug, Clone, Default, PartialEq, Eq)]
pub(crate) struct NativePlayerStatusState {
    pub(crate) data: Option<NativeUnitStatusData>,
    pub(crate) text: Option<NativeStatusTransitionText>,
    pub(crate) source: Vec<u8>,
    pub(crate) status: Vec<u8>,
    pub(crate) commands: Vec<u8>,
    pub(crate) steady: Vec<u8>,
    pub(crate) frames: Vec<Vec<u8>>,
    pub(crate) frame: i64,
    pub(crate) phase: String,
    pub(crate) drawn: bool,
}

impl Game {
    /// Go：`native_player_input.go:152`，`func (g *Game) stepNativePlayerStatus()`
    pub(crate) fn step_native_player_status(&mut self) {
        let Some(s) = self.native_player_status.as_mut() else {
            return;
        };
        if !s.drawn || s.frames.is_empty() {
            return;
        }
        s.drawn = false;
        s.frame += 1;
        if s.frame < s.frames.len() as i64 {
            return;
        }
        if s.phase == "closing" {
            self.native_player_status = None;
            return;
        }
        (s.frames, s.frame) = (Vec::new(), 0);
        if s.phase == "opening" {
            s.phase = String::from("status");
        } else {
            s.phase = String::from("commands");
        }
    }
}

// ---- R5 階段 1 骨架（tools/rust_r5_skeleton.sh 產生；本體由 R5 階段 2 移植）----

impl Game {
    /// Go：`native_player_input.go:214`，`func (g *Game) drawNativePlayerStatus(screen *ebiten.Image)`
    ///
    /// R5 階段 2 r5h 逐行移植。開場與第一章只走到 `s == nil` 的守門。HD 的三條（`drawModernBattlePlayerStatus`、
    /// `drawModernHDUnitStatus`、`drawModernHDStatusTransition`）在忠實模式（`modernStoryPortraits`、`modernHDText`
    /// 都是 nil 且介面不是 HD）都回 false、沒有副作用；三者任一成立時 `todo!("r6:…")`。
    pub(crate) fn draw_native_player_status(&mut self, screen: &mut Picture) {
        if self.native_player_status.is_none() {
            return;
        }
        if self.facility_hd_draw_active() {
            todo!(
                "r6:modern_hd_panel_transition.go:drawModernBattlePlayerStatus、modern_hd_status_text.go:drawModernHDUnitStatus、drawModernHDStatusTransition"
            );
        }
        let s = self.native_player_status.as_ref().unwrap();
        let frame = if s.frames.is_empty() { s.steady.clone() } else { s.frames[s.frame as usize].clone() };
        let palette = self.native_ui_palette.clone();
        self.present_native_class_frame_with_palette(screen, &frame, &palette);
        self.native_player_status.as_mut().unwrap().drawn = true;
    }
}
