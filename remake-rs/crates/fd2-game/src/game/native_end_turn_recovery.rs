//! Go 來源：`remake/cmd/fd2/native_end_turn_recovery.go`（179 行）。
//!
//! 骨架：Go 檔的 4 個函式中產生 3 個（階段 0 清單），其餘標 `r8`，見 `PORTING.json`。

use super::*;

/// Evidence/spec: docs/data/ida/fd2_end_turn_recovery_20261004.json (READY).
/// 0x1A30B publishes all mode2 candidates together, then all mode0 restores.
/// The two zero-tick waits retain Draw boundaries without claiming DOS duration.
///
/// Go：`native_end_turn_recovery.go:18`
///
/// 不 derive `Debug`／`Clone`：欄位含 callback（`Box<dyn FnOnce>`）、檔案或亂數產生器等做不到的型別。
#[derive(Default)]
pub(crate) struct NativeEndTurnRecoveryJob {
    pub(crate) plan: Vec<battle::NativeEndTurnRecovery>,
    pub(crate) entries: Vec<fdicon::NativeUnitLayerEntry>,
    /// initial, mask, raw restore
    pub(crate) frames: [Vec<u8>; 3],
    pub(crate) restore_work: Vec<u8>,
    pub(crate) frame: i64,
    pub(crate) drawn: bool,
    pub(crate) ready_at: GameInstant,
    pub(crate) before_gate: u8,
    pub(crate) cue_count: i64,
    pub(crate) committed: bool,
    pub(crate) after: Option<Box<dyn FnOnce(&mut Game) + Send>>,
}

impl Game {
    /// Go：`native_end_turn_recovery.go:32`，`func (g *Game) beginNativeEndTurnRecovery(after func()) error`
    /// Go 檢查接收者是否為 nil；Rust 的 `self` 不會是 null，該分支不移植。
    /// Go 的 func 參數 `after` 會被存起來（工作結構的延續），Rust 收擁有的 `Box`。
    pub(crate) fn begin_native_end_turn_recovery(
        &mut self,
        after: Option<Box<dyn FnOnce(&mut Game) + Send>>,
    ) -> Result<()> {
        let ready = self.native_end_turn_recovery.is_none()
            && self
                .st
                .as_ref()
                .is_some_and(|st| st.has_native_map_view_state && st.native_map_selector_cache.is_some())
            && native_map_assets_available(self.native_map_assets.as_ref())
            && self.native_map_dac.len() == 256 * 3
            && self.native_map_work.len() as i64 == indexedmap::NATIVE_UNIT_PRESENT_WORK_SIZE
            && self.native_map_vga.len() as i64 == indexedmap::NATIVE_MAP_VGA_SIZE;
        if !ready {
            return Err(game_err!("native END recovery: source frame/assets unavailable"));
        }
        let plan = self.st.as_ref().unwrap().plan_native_end_turn_recovery();
        if !plan.is_empty()
            && !os_mute_or_shot(Some(self))
            && (self.sfx.len() <= 4 || self.sfx.get(&4).is_none_or(|s| s.is_empty()))
        {
            return Err(game_err!("native END recovery: sample4 unavailable"));
        }
        let ready_at = self.game_now() + NATIVE_BIOS_TICK_PERIOD;
        let mut job = NativeEndTurnRecoveryJob { after, ready_at, ..Default::default() };
        job.frames[0] = self.native_map_vga.clone();
        let mut mask_work = self.native_map_work.clone();
        let mut restore_work = self.native_map_work.clone();
        let st = self.st.as_ref().unwrap();
        let roster =
            st.native_map_frame_roster().map_err(|e| GameError::from(e).wrap("native END recovery: roster: "))?;
        let view = st.native_map_view_state;
        let cache = st.native_map_selector_cache.as_ref().unwrap();
        let units = self.native_map_assets.as_ref().unwrap().units.as_ref().unwrap();
        for rec in &plan {
            let unit = rec.unit.expect("Go：rec.Unit 為 nil 時解參考 panic");
            let (entry, ok) = st.units[unit].native_unit_layer_entry();
            if !ok || entry.force_base || entry.inactive {
                return Err(game_err!("native END recovery: candidate lacks raw selector provenance"));
            }
            job.entries.push(entry);
            // 0x1DA53..0x1DA90 returns before sprite lookup outside this window.
            if entry.x < view.camera_x - 1
                || entry.x > view.camera_x + 12
                || entry.y < view.camera_y - 1
                || entry.y > view.camera_y + 8
            {
                continue;
            }
            let cycle =
                fdicon::native_frame_index(entry.motion_offset, false, roster.cycles.idle, roster.cycles.moving)?;
            let sprite = units
                .sprite_for_native_slot(cache, entry.slot, entry.pose, cycle)
                .map_err(|e| GameError::from(e).wrap("native END recovery: selector: "))?;
            let offset = fdicon::native_placement_offset(
                entry.x,
                entry.y,
                view.camera_x,
                view.camera_y,
                entry.pose,
                entry.motion_offset,
                0,
                false,
            );
            // Go `fmt.Errorf("... %d: %v", offset, err)`：失敗時 offset 是 0，err 為 nil 時印 `<nil>`。
            let offset = match offset {
                Ok(offset) if offset >= 0 => offset,
                Ok(offset) => return Err(game_err!("native END recovery: invalid placement {offset}: <nil>")),
                Err(err) => return Err(game_err!("native END recovery: invalid placement 0: {err}")),
            };
            // 4DDD7 reads arg8 both as stride and opaque mask color. argC=FD is
            // not read. Mask includes opaque source zero, unlike color-key blits.
            sprite
                .blit_constant_mask_at(&mut mask_work, 456, offset % 456, offset / 456, 0xc8)
                .map_err(|e| GameError::from(e).wrap("native END recovery: mode2: "))?;
            // 4DEDA does not consume +5 bit7. Restore raw pixels before gray redraw.
            sprite
                .blit_for_native_flags_at_offset(&mut restore_work, 456, offset, 0)
                .map_err(|e| GameError::from(e).wrap("native END recovery: mode0: "))?;
        }
        job.plan = plan;
        job.frames[1] = native_ai_idle_recovery_viewport(&mask_work, &job.frames[0])?;
        job.frames[2] = native_ai_idle_recovery_viewport(&restore_work, &job.frames[0])?;
        job.restore_work = restore_work;
        let st = self.st.as_mut().unwrap();
        if st.has_native_map_hud_state {
            job.before_gate = st.native_map_hud_state.display_gate_b;
            st.native_map_hud_state.display_gate_b = 0;
        }
        self.native_end_turn_recovery = Some(job);
        Ok(())
    }

    /// Go：`native_end_turn_recovery.go:104`，`func (g *Game) cancelNativeEndTurnRecovery(err error)`（`r8`，隨
    /// `stepNativeEndTurnRecovery` 補移植；Go `Draw` 在 END 回收畫不出來時也呼叫它，見 PORTING.md）。
    pub(crate) fn cancel_native_end_turn_recovery(&mut self, err: &GameError) {
        if let Some(job) = self.native_end_turn_recovery.as_ref()
            && !job.committed
            && self.st.as_ref().expect("Go：g.st 為 nil 時解參考 panic").has_native_map_hud_state
        {
            let before = job.before_gate;
            self.st.as_mut().unwrap().native_map_hud_state.display_gate_b = before;
        }
        self.native_end_turn_recovery = None;
        self.load_err = err.message().to_string();
    }

    /// Go：`native_end_turn_recovery.go:113`，`func (g *Game) stepNativeEndTurnRecovery(now time.Time)`
    pub(crate) fn step_native_end_turn_recovery(&mut self, now: GameInstant) {
        let Some(job) = self.native_end_turn_recovery.as_ref() else {
            return;
        };
        if !job.drawn {
            return;
        }
        if job.frame == 0 && now < job.ready_at {
            return;
        }
        if job.frame == 1 {
            let st = self.st.as_ref().expect("Go：g.st 為 nil 時解參考 panic");
            let check = (|| -> Result<()> {
                for (i, rec) in job.plan.iter().enumerate() {
                    let unit = rec.unit.expect("Go：rec.Unit 為 nil 時解參考 panic");
                    let (entry, ok) = st.units[unit].native_unit_layer_entry();
                    if !ok || entry != job.entries[i] {
                        return Err(game_err!("native END recovery: raw selector changed before commit"));
                    }
                }
                Ok(st.validate_native_end_turn_recovery(&job.plan)?)
            })();
            if let Err(err) = check {
                self.cancel_native_end_turn_recovery(&err);
                return;
            }
            let job = self.native_end_turn_recovery.as_ref().unwrap();
            let has_plan = !job.plan.is_empty();
            if has_plan {
                self.native_end_turn_recovery.as_mut().unwrap().cue_count += 1;
                if !os_mute_or_shot(Some(self)) {
                    self.play_sfx(4);
                }
            }
            let job = self.native_end_turn_recovery.as_ref().unwrap();
            if let Err(err) = self.st.as_mut().unwrap().commit_native_end_turn_recovery(&job.plan) {
                self.cancel_native_end_turn_recovery(&GameError::from(err));
                return;
            }
            let job = self.native_end_turn_recovery.as_mut().unwrap();
            job.committed = true;
            self.native_map_work = job.restore_work.clone();
        }
        let job = self.native_end_turn_recovery.as_mut().unwrap();
        job.frame += 1;
        job.drawn = false;
        if job.frame < job.frames.len() as i64 {
            return;
        }
        if let Err(err) = self.compose_native_map_frame() {
            self.cancel_native_end_turn_recovery(&err.wrap("native END recovery: final 11CAC: "));
            return;
        }
        let after = self.native_end_turn_recovery.take().and_then(|mut job| job.after.take());
        if let Some(after) = after {
            after(self);
        }
    }

    /// Go：`native_end_turn_recovery.go:162`，`func (g *Game) drawNativeEndTurnRecovery(screen *ebiten.Image) bool`
    ///
    /// R5 階段 2 r5h 逐行移植。畫成才設 `job.drawn = true`（`draw` 路徑，與 Go `Draw` 相同）；畫不出來回 false，
    /// `Game::draw_frame` 接著呼叫 `cancelNativeEndTurnRecovery`。無畫面重播仍由呈現回報設定（PORTING.md「C 組：
    /// 呈現回報」），兩邊的前置條件相同。
    pub(crate) fn draw_native_end_turn_recovery(&mut self, screen: &mut Picture) -> bool {
        let Some(job) = self.native_end_turn_recovery.as_ref() else {
            return false;
        };
        if job.frame < 0
            || job.frame >= job.frames.len() as i64
            || job.frames[job.frame as usize].len() as i64 != indexedmap::NATIVE_MAP_VGA_SIZE
            || self.native_map_dac.len() != 256 * 3
        {
            return false;
        }
        let Ok(palette) = fdother::vga_palette_from_dac(&self.native_map_dac) else {
            return false;
        };
        let img = IndexedFrame {
            width: 320,
            height: 200,
            pix: job.frames[job.frame as usize].clone(),
            palette: color_palette_from_nrgba(&palette),
        };
        let mut op = DrawImageOptions::default();
        op.geo_m.scale(2.0, 2.0);
        screen.draw_image(&Picture::from_indexed(&img), &op);
        self.native_end_turn_recovery.as_mut().unwrap().drawn = true;
        true
    }
}

#[cfg(test)]
#[path = "native_end_turn_recovery/native_end_turn_recovery_test.rs"]
mod native_end_turn_recovery_test;
