//! Go 來源：`remake/cmd/fd2/native_field_event61.go`（269 行）。
//!
//! Go 檔的 6 個函式都已移植；HD 的 `modernHDFieldEvent61` 凍結與 HD 世界、HUD 在 R6（PORTING.md「D 組：HD 預凍結延後」）。

use super::*;

/// Go：`native_field_event61.go:12`
///
/// 不 derive `Debug`／`Clone`：欄位含 callback（`Box<dyn FnOnce>`）、檔案或亂數產生器等做不到的型別。
#[derive(Default)]
pub(crate) struct NativeFieldEvent61Job {
    pub(crate) plan: battle::NativeFieldEvent61Plan,
    pub(crate) frames: Vec<Vec<u8>>,
    pub(crate) frame: i64,
    pub(crate) drawn: bool,
    pub(crate) last_tick: i64,
    pub(crate) has_tick: bool,
    pub(crate) clock: NativeBIOSClock,
    pub(crate) after: Option<Box<dyn FnOnce(&mut Game) + Send>>,
    pub(crate) hd: Option<ModernHDFieldEvent61>,
}

/// `assets/story/ch26.json` 第 `scene_index` 幕的 `line..line+count` 句轉成可編輯對白；有 `speaker_slot`、
/// 越界或 `count` 不為正時回 `None`（失敗即關閉）。
///
/// Go：`native_field_event61.go:24`，`func event61DialogueActions(g *Game, sceneIndex, line, count int) ([]battle.Action, bool)`
pub(crate) fn event61_dialogue_actions(
    g: &mut Game,
    scene_index: i64,
    line: i64,
    count: i64,
) -> Option<Vec<battle::Action>> {
    if line < 0 || count <= 0 {
        return None;
    }
    story_dialogue_actions(g, "assets/story/ch26.json", scene_index, line as usize, count as usize, "event61")
}

impl Game {
    /// beginNativeFieldEvent61 owns the selector1 event only after a successful
    /// unit action. A coordinate without event61 returns false; an event61 whose
    /// editable/original provenance is malformed blocks its mutation and reports
    /// the error rather than silently finishing a guessed handler.
    ///
    /// Go：`native_field_event61.go:51`，`func (g *Game) beginNativeFieldEvent61(actor *battle.Unit, after func()) bool`
    /// Go 檢查接收者是否為 nil；Rust 的 `self` 不會是 null，該分支不移植。
    /// Go 的 func 參數 `after` 會被存起來（工作結構的延續），Rust 收擁有的 `Box`。
    pub(crate) fn begin_native_field_event61(
        &mut self,
        actor: Option<UnitIdx>,
        after: Option<Box<dyn FnOnce(&mut Game) + Send>>,
    ) -> bool {
        let (Some(st), Some(actor)) = (self.st.as_ref(), actor) else {
            return false;
        };
        if self.native_field_event61.is_some() {
            return false;
        }
        let (x, y) = (st.units[actor].x, st.units[actor].y);
        let (event_id, bound) = battle::native_field_event_id_at(Some(st), x, y, 1);
        if !bound || event_id != 61 {
            return false;
        }
        let fail = |g: &mut Game, after: Option<Box<dyn FnOnce(&mut Game) + Send>>| {
            if let Some(after) = after {
                after(g);
            }
        };
        let plan = match battle::plan_native_field_event61(Some(st), Some(actor), x, y) {
            Ok(plan) => plan,
            Err(err) => {
                self.load_err = err.to_string();
                self.msg = String::from("事件 61 的原版證據不完整，已停止事件寫入");
                fail(self, after);
                return true;
            }
        };
        if plan.missing_item {
            let Some(mut actions) = event61_dialogue_actions(self, 0, 10, 1) else {
                self.load_err = String::from("event61: FDTXT2 editable dialogue unavailable");
                fail(self, after);
                return true;
            };
            self.start_battle_event(&mut actions, after);
            return true;
        }
        let Some(mut actions) = event61_dialogue_actions(self, 0, 11, 1) else {
            self.load_err = String::from("event61: FDTXT3 editable dialogue unavailable");
            fail(self, after);
            return true;
        };
        self.start_battle_event(
            &mut actions,
            Some(Box::new(move |g: &mut Game| {
                // Go 的 `after` 同時被這個閉包與 `beginNativeFieldEvent61Presentation` 的工作捕捉；失敗時才在這裡
                // 呼叫。Rust 的 `FnOnce` 只能交出一次，由演出函式在失敗時交回。
                if let Err((err, after)) = g.begin_native_field_event61_presentation(plan, after) {
                    g.load_err = err.message().to_string();
                    if let Some(after) = after {
                        after(g);
                    }
                }
            })),
        );
        true
    }

    /// 0x356B7 先以 0x16C57／0x196CB 收起 FDTXT #3，再載 FDOTHER#45；底圖是中性的戰術重繪，不是帶進
    /// selector1 的行動／射程疊圖。素材與每一幀都先合成好，才發布演出工作。
    ///
    /// Go：`native_field_event61.go:99`，`func (g *Game) beginNativeFieldEvent61Presentation( plan battle.NativeFieldEvent61Plan, after func(), ) error`
    /// 失敗時 Go 回錯誤、`after` 留在呼叫端；Rust 的 `after` 已移進來，連同錯誤交回。
    #[allow(clippy::type_complexity)]
    pub(crate) fn begin_native_field_event61_presentation(
        &mut self,
        plan: battle::NativeFieldEvent61Plan,
        after: Option<Box<dyn FnOnce(&mut Game) + Send>>,
    ) -> std::result::Result<(), (GameError, Option<Box<dyn FnOnce(&mut Game) + Send>>)> {
        match self.prepare_native_field_event61_frames(&plan) {
            Ok(frames) => {
                // Go 在這裡以 `freezeModernHDFieldEvent61` 凍結 HD 來源（只有 HD 繪製讀），
                // `prepareModernHDFieldEvent61` 在沒有 HD 地圖人物時回 nil，`prepareModernHDFieldEvent61NativeHUD`
                // 在有 HD 主題或介面不是 HD 時回 nil；Rust 的 `hd` 是 `None`，HD 兩支在 R6。
                if self.modern_story_portraits.as_ref().is_some_and(|set| !set.map_sprites_hd.is_empty()) {
                    todo!("R6: modern_hd_field_event61.go:40 prepareModernHDFieldEvent61（HD 地圖人物）")
                }
                if self.modern_story_portraits.is_none() && self.modern_hd_interface_enabled() {
                    todo!("R6: modern_hd_field_event61.go:101 prepareModernHDFieldEvent61NativeHUD（HD 介面）")
                }
                self.native_field_event61 = Some(NativeFieldEvent61Job { plan, frames, after, ..Default::default() });
                Ok(())
            }
            Err(err) => Err((err, after)),
        }
    }

    /// `beginNativeFieldEvent61Presentation` 發布工作之前的部分：中性底圖與 FDOTHER#45 每一幀的累積合成。
    fn prepare_native_field_event61_frames(&mut self, plan: &battle::NativeFieldEvent61Plan) -> Result<Vec<Vec<u8>>> {
        if self.native_field_event61.is_some() {
            return Err(game_err!("event61: presentation is already active"));
        }
        if !self.reset_native_target_field() {
            let st = self.st.as_ref().unwrap();
            return Err(game_err!(
                "event61: neutral target field is unavailable ({}x{}/{})",
                st.w,
                st.h,
                st.native_tile_blit_modes.len()
            ));
        }
        if !self.st.as_mut().unwrap().materialize_native_map_range_mode(1) {
            return Err(game_err!("event61: neutral range mode is unavailable"));
        }
        self.compose_native_map_frame().map_err(|err| game_err!("event61: native source frame: {}", err.message()))?;
        let presentation = &plan.presentation;
        if presentation.resource != 45 {
            return Err(game_err!("event61: unsupported presentation resource {}", presentation.resource));
        }
        let decoded = fdother::load_separated_event61_frames(separated_asset_path("animations/fdother_045_event61"))
            .map_err(|err| game_err!("event61: separated FDOTHER resource: {err}"))?;
        if decoded.len() as i64 != presentation.frames {
            return Err(game_err!("event61: FDOTHER frame count={} want={}", decoded.len(), presentation.frames));
        }
        let mut current = self.native_map_vga.clone();
        let mut frames = Vec::with_capacity(decoded.len());
        for (index, frame) in decoded.iter().enumerate() {
            frame
                .blit_at(&mut current, presentation.stride, presentation.destination_offset, presentation.transparent)
                .map_err(|err| game_err!("event61: frame {index}: {err}"))?;
            frames.push(current.clone());
        }
        Ok(frames)
    }

    /// 每一幀畫過、且 BIOS tick 前進到 `delay_ticks` 才換下一幀；最後一幀之後提交事件、寫入 JOIN，
    /// 再播 FDTXT #4 的十句對白。
    ///
    /// Go：`native_field_event61.go:166`，`func (g *Game) stepNativeFieldEvent61Tick(rawTick int)`
    pub(crate) fn step_native_field_event61_tick(&mut self, raw_tick: i64) {
        let Some(job) = self.native_field_event61.as_mut() else {
            return;
        };
        if !job.drawn {
            return;
        }
        if !job.has_tick {
            (job.last_tick, job.has_tick) = (raw_tick, true);
            return;
        }
        let delta = (raw_tick as u16).wrapping_sub(job.last_tick as u16) as i16 as i64;
        if delta < job.plan.presentation.delay_ticks {
            return;
        }
        job.last_tick = raw_tick;
        job.drawn = false;
        job.frame += 1;
        if (job.frame as usize) < job.frames.len() {
            return;
        }
        let mut job = self.native_field_event61.take().unwrap();
        let after = job.after.take();
        let fail = |g: &mut Game, message: String, after: Option<Box<dyn FnOnce(&mut Game) + Send>>| {
            g.load_err = message;
            if let Some(after) = after {
                after(g);
            }
        };
        let joined = match battle::commit_native_field_event61(self.st.as_mut(), &job.plan, job.frames.len() as i64) {
            Ok(joined) => joined,
            Err(err) => return fail(self, err.to_string(), after),
        };
        if let Err(err) = self.persist_native_field_event61_join(joined) {
            return fail(self, err.message().to_string(), after);
        }
        let Some(mut actions) = event61_dialogue_actions(self, 1, 0, 10) else {
            return fail(self, String::from("event61: FDTXT4 editable dialogue unavailable"), after);
        };
        self.start_battle_event(&mut actions, after);
    }

    /// JOIN 的角色只能有一筆我方記錄；初始化裝備基準後寫進持續隊伍。
    ///
    /// Go：`native_field_event61.go:215`，`func (g *Game) persistNativeFieldEvent61Join(id int) error`
    pub(crate) fn persist_native_field_event61_join(&mut self, id: i64) -> Result<()> {
        if !campaign::joinable_character_id(id) || self.st.is_none() {
            return Err(game_err!("event61: invalid JOIN character {id}"));
        }
        let mut joined = None;
        for (index, unit) in self.st.as_ref().unwrap().units.iter().enumerate() {
            if unit.fig == id && unit.camp == battle::OWN {
                if joined.is_some() {
                    return Err(game_err!("event61: character {id} has multiple own records"));
                }
                joined = Some(index);
            }
        }
        let Some(joined) = joined else {
            return Err(game_err!("event61: character {id} own record is absent"));
        };
        // Go 建一個只含這筆記錄指標的暫時 State 交給 `initializeEquipmentBases`；Rust 把記錄移進暫時 State
        // 再移回同一格。
        let unit = std::mem::take(&mut self.st.as_mut().unwrap().units[joined]);
        let mut scratch = battle::State::default();
        scratch.units.push(unit);
        self.initialize_equipment_bases(Some(&mut scratch));
        let unit = scratch.units.pop().unwrap();
        self.st.as_mut().unwrap().units[joined] = unit;
        if !self.party_members.get(&id).copied().unwrap_or(false) {
            self.party_members.insert(id, true);
            self.party_join_order.push(id);
        }
        let unit = &self.st.as_ref().unwrap().units[joined];
        self.party_roster.insert(id, clone_native_shop_unit(unit));
        Ok(())
    }

    /// Go：`native_field_event61.go:246`，`func (g *Game) drawNativeFieldEvent61(screen *ebiten.Image) bool`
    pub(crate) fn draw_native_field_event61(&mut self, screen: Option<&mut Picture>) -> bool {
        let Some(job) = self.native_field_event61.as_ref() else {
            return false;
        };
        let palette_ok = self.native_map_assets.as_ref().is_some_and(|assets| assets.palette.len() >= 256);
        if job.frame < 0 || job.frame as usize >= job.frames.len() || !palette_ok {
            return false;
        }
        if self.modern_story_portraits.as_ref().is_some_and(|set| set.map_sprites_hd.len() == 125) {
            todo!("r6:native_field_event61.go:254 drawPreparedModernHDMapFrame（HD 地圖人物的事件 61 幀）")
        }
        if let Some(screen) = screen {
            let frame = job.frames[job.frame as usize].clone();
            let palette = self.native_map_assets.as_ref().unwrap().palette.clone();
            self.present_native_class_frame_with_palette(screen, &frame, &palette);
        }
        // Go 在沒有 HD 主題、介面 HD 且有 HD HUD 時改發布 HD HUD；Rust 的 `hd` 是 `None`（HD 介面在演出建立時已停在
        // R6），這一支不會成立。
        self.native_field_event61.as_mut().unwrap().drawn = true;
        true
    }
}

#[cfg(test)]
#[path = "native_field_event61/native_field_event61_test.rs"]
mod native_field_event61_test;
