//! Go 來源：`remake/cmd/fd2/main.go:5919`..`5975`（指令格輸入）、`6105`..`6219`（指令目標選擇）、
//! `7598`..`8193`（指令目標確認的共用前段）。
//!
//! 指令 0x1CFF0 分兩段：record+3 先挑游標候選，確認後才由 record+4 建最後的效果名單。這裡只負責兩段之間的
//! 選擇狀態；各指令的玩家演出入口逐一補移植，還沒接上的指令在確認時顯示訊息並回到指令格。

use super::*;

impl Game {
    /// 有兩段目標契約與正式交易 owner 的指令 ID；其他 ID 仍顯示在格上，確認時失敗即關閉。
    ///
    /// Go：`main.go:6105`，`func (g *Game) nativeCommandTargetSupported(id int) bool`
    pub(crate) fn native_command_target_supported(&self, id: i64) -> bool {
        (0..=35).contains(&id) && id != 9
    }

    /// Go：`main.go:6119`，`func (g *Game) nativeCommandTargetUnits() ([]*battle.Unit, error)`
    pub(crate) fn native_command_target_units(&self) -> Result<Vec<UnitIdx>> {
        self.native_command_target_units_for(self.native_command_target_id)
    }

    /// 指令游標的候選單位：取所選原始指令 ID 的 record+3 選取欄位。
    ///
    /// Go：`main.go:6126`，`func (g *Game) nativeCommandTargetUnitsFor(id int) ([]*battle.Unit, error)`
    pub(crate) fn native_command_target_units_for(&self, id: i64) -> Result<Vec<UnitIdx>> {
        let (Some(st), Some(sel)) = (self.st.as_ref(), self.sel) else {
            return Err(game_err!("native target command context unavailable"));
        };
        if !self.native_command_target_supported(id) {
            return Err(game_err!("native target command context unavailable"));
        }
        if st.native_command_book.len() != 36 {
            return Err(game_err!("native command book incomplete"));
        }
        let record = &st.native_command_book[id as usize];
        let flags = st.native_command_base_flags()?;
        let origin = battle::Cell { x: st.units[sel].x, y: st.units[sel].y };
        if id == 23 {
            return Ok(battle::native_attack_candidates(
                st.w,
                st.h,
                origin,
                record.selection_mode,
                1,
                record.target_code,
                &flags,
                &st.units,
            )?);
        }
        Ok(battle::native_command_targets(
            st.w,
            st.h,
            origin,
            record.selection_mode,
            record.target_code,
            &flags,
            &st.units,
        )?)
    }

    /// 選目標期間的範圍標記：record+3 為 0 時改用 0x1D366 以施法者為中心的效果範圍（overlay 1）。
    ///
    /// Go：`main.go:6149`，`func (g *Game) materializeNativeCommandTargetField(record battle.NativeCommandRecord) error`
    pub(crate) fn materialize_native_command_target_field(
        &mut self,
        record: &battle::NativeCommandRecord,
    ) -> Result<()> {
        let (Some(st), Some(sel)) = (self.st.as_ref(), self.sel) else {
            return Err(game_err!("native command target field context unavailable"));
        };
        let flags = st.native_command_base_flags()?;
        let (mut selection_mode, mut overlay_selector) =
            (record.selection_mode, battle::native_map_overlay_selector_from_record_byte(record.effect_mode as u8));
        if selection_mode == 0 {
            (selection_mode, overlay_selector) = (record.effect_mode, 1);
        }
        let inner_radius = i64::from(record.id == 23);
        let origin = battle::Cell { x: st.units[sel].x, y: st.units[sel].y };
        let field = battle::native_command_target_field_bytes(st.w, st.h, origin, selection_mode, inner_radius, &flags);
        let field = match field {
            Ok(field) if field.len() == st.native_tile_blit_modes.len() => field,
            _ => return Err(game_err!("native command target field incomplete")),
        };
        let st = self.st.as_mut().unwrap();
        if !(0..=0xff).contains(&record.effect_mode) || !st.materialize_native_map_range_mode(overlay_selector) {
            return Err(game_err!("native command overlay selector invalid"));
        }
        st.native_tile_blit_modes.copy_from_slice(&field);
        Ok(())
    }

    /// Go：`main.go:6184`，`func (g *Game) beginNativeCommandTargetSelection(id int) error`
    pub(crate) fn begin_native_command_target_selection(&mut self, id: i64) -> Result<()> {
        let record = match self.st.as_ref() {
            Some(st) if st.native_command_book.len() == 36 && self.native_command_target_supported(id) => {
                st.native_command_book[id as usize]
            }
            _ => return Err(game_err!("native command target context unavailable")),
        };
        self.materialize_native_command_target_field(&record)?;
        (self.native_command_open, self.native_command0_targeting) = (false, true);
        self.native_command_target_id = id;
        self.native_command30_saved_cursor = (id == 30).then_some(battle::Cell { x: self.cur_x, y: self.cur_y });
        Ok(())
    }

    /// 選目標時 Esc：清掉標記，回到指令格。
    ///
    /// Go：`main.go:6201`，`func (g *Game) cancelNativeCommandTargetSelection()`
    pub(crate) fn cancel_native_command_target_selection(&mut self) {
        if self.native_command0_targeting && self.st.is_some() {
            self.reset_native_target_field();
            self.st.as_mut().unwrap().materialize_native_map_range_mode(1);
        }
        (self.cast_sp, self.native_command0_targeting, self.native_command_open) = (None, false, true);
        self.native_command30_saved_cursor = None;
    }

    /// 指令格開著時的輸入（方向鍵移動、Esc 回指令環、Enter 進選目標）。回傳值同 `ring_input`：已攔截。
    ///
    /// Go：`main.go:5919`..`5975`
    pub(crate) fn native_command_grid_input(&mut self, enter: bool, esc: bool) -> bool {
        let Some(sel) = self.sel else {
            self.native_command_open = false;
            return false;
        };
        let ids = self.st.as_ref().unwrap().units[sel].native_command_ids();
        if esc {
            self.native_command_open = false;
            self.begin_action_overlay_open(self.ring_sel);
            return true;
        }
        let count = ids.len() as i64;
        for (key, direction) in [(Key::ArrowUp, 0), (Key::ArrowDown, 1), (Key::ArrowLeft, 2), (Key::ArrowRight, 3)] {
            if self.tick_input.is_just_pressed(key) {
                self.native_command_sel = battle::native_command_grid_move(self.native_command_sel, count, direction);
            }
        }
        if !enter || self.native_command_sel < 0 || self.native_command_sel >= count {
            return true;
        }
        let id = ids[self.native_command_sel as usize];
        let book_ready = self.st.as_ref().is_some_and(|st| st.native_command_book.len() == 36);
        if self.native_command_target_supported(id) && book_ready {
            let st = self.st.as_ref().unwrap();
            let mp_cost = st.native_command_book[id as usize].mp_cost;
            if id == 0 && st.native_command_resistances.is_empty() {
                self.msg = String::from("原始指令 0：抗性資料未驗證");
                return true;
            }
            if st.units[sel].mp < mp_cost {
                if let (message, true) = self.locale_message("battle.mp.insufficient", &[]) {
                    self.msg = message;
                }
                return true;
            }
            if self.native_command_target_units_for(id).is_ok() {
                if self.begin_native_command_target_selection(id).is_err() {
                    self.msg = format!("原始指令 {id}：target grid 無法 materialize");
                    return true;
                }
                self.msg = format!("原始指令 {id}：選擇目標");
                return true;
            }
        }
        // 不把原始指令 ID 轉成舊的 CastArea：spells.json 的位元組相同只證明表身分，不證明目標／效果等價。
        self.native_command_open = false;
        self.begin_action_overlay_open(self.ring_sel);
        self.msg = format!("原始指令 {id}：目標／效果尚未驗證");
        true
    }

    /// 選目標時確認：共用的游標閘門（record+3 為 0 時先驗自身範圍，再以 targetCode 4 過 0x115B6）。
    /// 回傳 `Some((id, target))` 代表閘門通過、交給各指令的演出入口。
    ///
    /// Go：`main.go:7598`..`7630`
    pub(crate) fn native_command_target_confirmation(&mut self) -> Option<(i64, Option<UnitIdx>)> {
        let id = self.native_command_target_id;
        let st = self.st.as_ref()?;
        let (x, y) = (self.cur_x, self.cur_y);
        if st.native_command_book.len() != 36
            || !(0..36).contains(&id)
            || st.native_tile_blit_modes.len() as i64 != st.w * st.h
            || !(0..st.w).contains(&x)
            || !(0..st.h).contains(&y)
        {
            self.msg = format!("原始指令 {id}：cursor-confirm raw state 不完整");
            return None;
        }
        let record = &st.native_command_book[id as usize];
        let mut target_code = record.target_code;
        if record.selection_mode == 0 {
            let ok = st.native_command_base_flags().ok().is_some_and(|flags| {
                battle::native_command_self_centered_targets(
                    st.w,
                    st.h,
                    self.sel,
                    battle::Cell { x, y },
                    record.effect_mode,
                    &flags,
                    &st.units,
                )
                .is_ok()
            });
            if !ok {
                self.msg = format!("原始指令 {id}：自身範圍確認不合法");
                return None;
            }
            target_code = 4;
        }
        let allowed = battle::native_cursor_confirmation_allowed(
            battle::Cell { x, y },
            st.native_tile_blit_modes[(y * st.w + x) as usize],
            st.native_map_range_mode,
            target_code,
            &st.units,
        );
        if !matches!(allowed, Ok(true)) {
            self.msg = format!("原始指令 {id}：游標確認不合法");
            return None;
        }
        Some((id, st.unit_at(x, y)))
    }

    /// 閘門通過後交給各指令的玩家演出入口。回傳 false 代表這個 ID 還沒有玩家入口。
    ///
    /// Go：`main.go:7631`..`8193`（逐一補移植；目前接上指令 0）
    pub(crate) fn start_native_player_command(&mut self, id: i64, target: Option<UnitIdx>) -> bool {
        let Some(actor) = self.sel else { return true };
        let cursor = (self.cur_x, self.cur_y);
        match id {
            0 => {
                let Some(target) = target else {
                    self.msg = String::from("原始指令 0：indexed 演出不可用 (target unavailable)");
                    return true;
                };
                let then = Box::new(move |g: &mut Game, results: Vec<battle::NativeCommandDamageResult>| {
                    let (mut hit, mut total) = (0, 0);
                    for result in &results {
                        if result.native_command_damage.hit {
                            hit += 1;
                            total += result.native_command_damage.damage;
                            g.award_death_reward(result.target, Some(actor));
                        }
                    }
                    g.face_native_player_command_actor(actor, cursor);
                    g.msg = format!("原始指令 0：命中 {hit}，傷害 {total}");
                    g.award_native_command_damage_experience(actor, &results);
                    g.finish_successful_unit_action(Some(actor), Some(native_player_command_after()));
                    g.check_result();
                });
                if let Err(err) = self.start_native_command0_presentation(actor, target, then) {
                    self.msg = format!("原始指令 0：indexed 演出不可用 ({err})");
                }
                true
            }
            _ => false,
        }
    }

    /// 施法者轉向確認時的游標格（Go `actor.SetMapPose(dirToward(actor.X, actor.Y, g.curX, g.curY))`）。
    fn face_native_player_command_actor(&mut self, actor: UnitIdx, cursor: (i64, i64)) {
        let unit = &mut self.st.as_mut().unwrap().units[actor];
        let pose = dir_toward(unit.x, unit.y, cursor.0, cursor.1);
        unit.set_map_pose(pose);
    }
}

/// 玩家指令行動的收尾（Go 每個分支傳給 `finishSuccessfulUnitAction` 的同一段）：清目標標記、範圍模式回 1、
/// 結束選取。
fn native_player_command_after() -> Box<dyn FnOnce(&mut Game) + Send> {
    Box::new(|g: &mut Game| {
        g.reset_native_target_field();
        if let Some(st) = g.st.as_mut() {
            st.materialize_native_map_range_mode(1);
        }
        (g.native_command0_targeting, g.native_command_target_id, g.sel, g.reach, g.moved) =
            (false, 0, None, BTreeMap::new(), false);
    })
}
