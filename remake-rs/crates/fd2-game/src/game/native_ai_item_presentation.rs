//! Go 來源：`remake/cmd/fd2/native_ai_item_presentation.go`（646 行）。
//!
//! 骨架：Go 檔的 8 個函式中產生 2 個（階段 0 清單），其餘標 `r8`，見 `PORTING.json`。

use super::*;

/// Go：`native_ai_item_presentation.go:14`
///
/// 不 derive `Debug`／`Clone`：欄位含 callback（`Box<dyn FnOnce>`）、檔案或亂數產生器等做不到的型別。
#[derive(Default)]
pub(crate) struct NativeAIItemPresentationJob {
    pub(crate) transaction: Option<NativeAIItemTransaction>,
    pub(crate) frames: Vec<NativeCompoundPresentedFrame>,
    pub(crate) frame: i64,
    pub(crate) repeat: i64,
    pub(crate) publish_at: i64,
    pub(crate) published: bool,
    pub(crate) drawn: bool,
    pub(crate) holding: bool,
    pub(crate) hold: i64,
    pub(crate) baseline_work: Vec<u8>,
    pub(crate) baseline_vga: Vec<u8>,
    pub(crate) post_transaction_work: Vec<u8>,
    pub(crate) post_vga: Vec<u8>,
    pub(crate) then: Option<Box<dyn FnOnce(&mut Game) + Send>>,
}

impl Game {
    pub(crate) fn execute_native_ai_item_restore(&mut self, plan: &battle::AIPlan) -> Result<()> {
        let transaction = self.plan_native_ai_item_restore(plan)?;
        let actor = plan.u.expect("已驗證敵方道具使用者");
        let state = self.st.as_ref().expect("已驗證戰場");
        let (x, y) = (state.units[actor].x, state.units[actor].y);
        self.ai_focus_cursor(x, y);
        self.ai_focus_cursor(plan.native_action_destination.x, plan.native_action_destination.y);
        let item = plan.native_item_id;
        let after = Box::new(move |game: &mut Game| {
            game.msg = format!("使用道具 {item:02X}h");
            let after = game.native_ai_action_after.take();
            game.finish_successful_unit_action(Some(actor), after);
            game.check_result();
        });
        let job = self.build_native_target_item_presentation(transaction, after)?;
        self.native_ai_item_presentation = Some(job);
        self.st.as_mut().expect("已驗證戰場").native_experience_accumulator = 0;
        Ok(())
    }

    /// 依交易選 `0x20c6f` 呼叫端的演出：回復走指令 33 尾段，指令傷害依路線走 `0x1cd17` 或 `0x1cac7`。
    /// 敵方與我方道具共用。
    ///
    /// Go：`native_ai_item_presentation.go:46`，`func (g *Game) startNativeTargetItemPresentation(plan *battle.AIPlan, then func()) (bool, error)`
    pub(crate) fn build_native_target_item_presentation(
        &mut self,
        transaction: NativeAIItemTransaction,
        then: Box<dyn FnOnce(&mut Game) + Send>,
    ) -> Result<NativeAIItemPresentationJob> {
        if transaction.restore.is_some() {
            return self.build_native_ai_item_restore_presentation(transaction, then);
        }
        match transaction.damage_route.map(|route| route.presentation) {
            Some(0x1cd17) => self.build_native_ai_item_damage_presentation(transaction, then),
            Some(0x1cac7) => self.build_native_ai_item_type21_presentation(transaction, then),
            _ => Err(game_err!("native item indexed presentation unavailable")),
        }
    }

    fn build_native_ai_item_restore_presentation(
        &mut self,
        transaction: NativeAIItemTransaction,
        then: Box<dyn FnOnce(&mut Game) + Send>,
    ) -> Result<NativeAIItemPresentationJob> {
        if self.native_ai_item_presentation.is_some() || self.atk.is_some() || self.indexed_transition.is_some() {
            return Err(game_err!("另一個戰鬥演出尚未完成"));
        }
        let (hud, hud_ready) = self.native_map_hud_input();
        let map = self
            .native_map_assets
            .as_ref()
            .filter(|map| native_map_assets_available(Some(map)))
            .ok_or_else(|| game_err!("敵方道具缺少原版地圖素材"))?;
        let state = self
            .st
            .as_ref()
            .filter(|state| state.has_native_map_view_state)
            .ok_or_else(|| game_err!("敵方道具缺少原版視圖"))?;
        if !hud_ready
            || self.native_map_work.len() as i64 != indexedmap::NATIVE_UNIT_PRESENT_WORK_SIZE
            || self.native_map_vga.len() as i64 != indexedmap::NATIVE_MAP_VGA_SIZE
        {
            return Err(game_err!("敵方道具缺少完整地圖畫面"));
        }
        let tail = fdother::build_native_command33_tail_schedule()?;
        let view = state.native_map_view_state;
        let idle = state.native_map_frame_roster()?.cycles.idle;
        let targets = transaction
            .targets
            .iter()
            .map(|&index| {
                let unit = &state.units[index];
                if !unit.has_native_map_presentation || !unit.has_map_selector_slot {
                    return Err(game_err!("敵方道具目標 {index} 缺少原生圖示"));
                }
                Ok(indexedmap::NativeCommandHealTailTarget {
                    record_index: index as i64,
                    x: i64::from(unit.native_map_presentation.x),
                    y: i64::from(unit.native_map_presentation.y),
                    selector_slot: unit.map_selector_slot,
                })
            })
            .collect::<Result<Vec<_>>>()?;
        let effect_targets = targets
            .iter()
            .copied()
            .filter(|target| {
                (view.camera_x - 1..=view.camera_x + 12).contains(&target.x)
                    && (view.camera_y - 1..=view.camera_y + 8).contains(&target.y)
            })
            .collect::<Vec<_>>();
        let palette = color_palette_from_nrgba(&fdother::vga_palette_from_dac(&self.native_map_dac)?);
        let image = |pix| {
            Arc::new(Picture::from_indexed(&IndexedFrame { width: 320, height: 200, pix, palette: palette.clone() }))
        };
        let baseline_work = self.native_map_work.clone();
        let baseline_vga = self.native_map_vga.clone();
        let mut frames = Vec::new();
        for index in tail.effect_start..tail.effect_start + tail.effect_frames {
            let effect = map.fdother6.get(index as usize).ok_or_else(|| game_err!("缺少敵方道具演出格 {index}"))?;
            let mut work = baseline_work.clone();
            let mut pixels = baseline_vga.clone();
            indexedmap::compose_native_command_heal_effect_frame(
                &mut work,
                &mut pixels,
                &baseline_work,
                effect,
                &effect_targets,
                view.camera_x,
                view.camera_y,
            )?;
            frames.push(NativeCompoundPresentedFrame {
                image: Some(image(pixels)),
                delay: tail.effect_frame_delay_ticks,
                ..Default::default()
            });
        }
        let mut work = baseline_work.clone();
        let mut mask = baseline_vga.clone();
        indexedmap::compose_native_command_heal_mask_frame(
            &mut work,
            &mut mask,
            &baseline_work,
            map.units.as_ref(),
            state.native_map_selector_cache.as_ref(),
            &effect_targets,
            view.camera_x,
            view.camera_y,
            idle,
            tail.mask_index as u8,
        )?;
        let masks = [image(baseline_vga.clone()), image(mask)];
        let mask_start = frames.len();
        for index in 0..tail.mask_pairs * 2 + 1 {
            frames.push(NativeCompoundPresentedFrame {
                image: Some(masks[index as usize % 2].clone()),
                delay: 1,
                ..Default::default()
            });
        }
        let publish_at = frames.len();
        let mut post = state.clone();
        for (unit, record) in post.units.iter_mut().zip(transaction.after.as_chunks::<80>().0) {
            sync_native_item_runtime_record(unit, record);
        }
        let mut input = build_native_map_frame_input(
            Some(map),
            self.m.as_ref(),
            Some(&post),
            NativeMapFrameRuntime { hud, ..Default::default() },
        )?;
        let mut post_transaction_work = baseline_work.clone();
        let mut post_vga = baseline_vga.clone();
        indexedmap::compose_native_frame(&mut post_transaction_work, &mut post_vga, &mut input)?;
        let results = &transaction.restore.as_ref().expect("已驗證回復交易").results;
        let mut queue = Vec::new();
        for (target, result) in targets.iter().zip(results) {
            let visible = (view.camera_x..view.camera_x + 12).contains(&target.x)
                && (view.camera_y - 1..=view.camera_y + 7).contains(&target.y);
            queue = battle::append_native_presentation_digits(
                &queue,
                result.actual,
                tail.digit_bias,
                target.record_index,
                visible,
            )?;
        }
        for index in 0..tail.digit_frames {
            let mut work = post_transaction_work.clone();
            let mut pixels = post_vga.clone();
            indexedmap::compose_native_command_heal_digit_frame(
                &mut work,
                &mut pixels,
                &post_transaction_work,
                &map.command_heal_digits,
                &queue,
                &targets,
                view.camera_x,
                view.camera_y,
                tail.digit_vertical,
                index,
            )?;
            frames.push(NativeCompoundPresentedFrame { image: Some(image(pixels)), delay: 1, ..Default::default() });
        }
        for (index, selector) in [(0, tail.effect_sample), (mask_start, tail.mask_sample)] {
            frames[index].sound =
                self.separated_command_sfx.get(&80).and_then(|bank| bank.get(&selector)).cloned().unwrap_or_default();
            if !os_mute_or_shot(Some(self)) && frames[index].sound.is_empty() {
                return Err(game_err!("敵方道具缺少原版樣本 80/{selector}"));
            }
        }
        Ok(NativeAIItemPresentationJob {
            transaction: Some(transaction),
            frames,
            publish_at: publish_at as i64,
            baseline_work,
            baseline_vga,
            post_transaction_work,
            post_vga,
            then: Some(then),
            ..Default::default()
        })
    }

    /// 敵方指令傷害物品（0x1cd17）：效果格 → 單位混色格 → 交易發布後的數字格。原版索引色分支，HD 另期。
    ///
    /// Go：`native_ai_item_presentation.go:362`，`func (g *Game) startNativeAIItemDamagePresentation(tx *nativeAIItemTransaction, then func()) (bool, error)`
    fn build_native_ai_item_damage_presentation(
        &mut self,
        transaction: NativeAIItemTransaction,
        then: Box<dyn FnOnce(&mut Game) + Send>,
    ) -> Result<NativeAIItemPresentationJob> {
        let Some(route) = transaction.damage_route.filter(|_| transaction.damage.len() == transaction.targets.len())
        else {
            return Err(game_err!("native AI item damage presentation context unavailable"));
        };
        if !self.current_native_system_options().full_presentation_enabled() {
            return Err(game_err!("native AI item indexed presentation unavailable"));
        }
        if self.native_ai_item_presentation.is_some() || self.atk.is_some() || self.indexed_transition.is_some() {
            return Err(game_err!("native AI item presentation already active"));
        }
        let (hud, hud_ready) = self.native_map_hud_input();
        let map = self
            .native_map_assets
            .as_ref()
            .filter(|map| native_map_assets_available(Some(map)))
            .ok_or_else(|| game_err!("native AI item indexed map state unavailable"))?;
        let state = self
            .st
            .as_ref()
            .filter(|state| state.has_native_map_view_state && state.has_native_map_cycle_state)
            .ok_or_else(|| game_err!("native AI item indexed map state unavailable"))?;
        if !hud_ready
            || self.native_map_work.len() as i64 != indexedmap::NATIVE_UNIT_PRESENT_WORK_SIZE
            || self.native_map_vga.len() as i64 != indexedmap::NATIVE_MAP_VGA_SIZE
            || self.native_map_dac.len() != 256 * 3
        {
            return Err(game_err!("native AI item indexed map state unavailable"));
        }
        let schedule = fdother::build_native_ai_item_damage_tail_schedule(route.command_id)?;
        if (map.fdother6.len() as i64) < schedule.effect_start + schedule.effect_frames
            || map.command_heal_digits.len() as i64 <= schedule.digit_bias + 9
        {
            return Err(game_err!("native AI item damage descriptors unavailable"));
        }
        let view = state.native_map_view_state;
        let targets = transaction
            .targets
            .iter()
            .map(|&index| {
                let unit = &state.units[index];
                if !unit.has_native_map_presentation || !unit.has_map_selector_slot {
                    return Err(game_err!("敵方道具目標 {index} 缺少原生圖示"));
                }
                Ok(indexedmap::NativeCommandHealTailTarget {
                    record_index: index as i64,
                    x: i64::from(unit.native_map_presentation.x),
                    y: i64::from(unit.native_map_presentation.y),
                    selector_slot: unit.map_selector_slot,
                })
            })
            .collect::<Result<Vec<_>>>()?;
        let visible = targets
            .iter()
            .copied()
            .filter(|t| {
                (view.camera_x - 1..=view.camera_x + 12).contains(&t.x)
                    && (view.camera_y - 1..=view.camera_y + 8).contains(&t.y)
            })
            .collect::<Vec<_>>();
        let palette = color_palette_from_nrgba(&fdother::vga_palette_from_dac(&self.native_map_dac)?);
        let image = |pix| {
            Arc::new(Picture::from_indexed(&IndexedFrame { width: 320, height: 200, pix, palette: palette.clone() }))
        };
        let baseline_work = self.native_map_work.clone();
        let baseline_vga = self.native_map_vga.clone();
        let mut frames = Vec::new();
        for index in schedule.effect_start..schedule.effect_start + schedule.effect_frames {
            let effect = &map.fdother6[index as usize];
            let (mut work, mut pixels) = (baseline_work.clone(), baseline_vga.clone());
            indexedmap::compose_native_command_heal_effect_frame(
                &mut work,
                &mut pixels,
                &baseline_work,
                effect,
                &visible,
                view.camera_x,
                view.camera_y,
            )?;
            frames.push(NativeCompoundPresentedFrame {
                image: Some(image(pixels)),
                delay: schedule.effect_frame_delay_ticks,
                ..Default::default()
            });
        }
        for &blend in schedule.blend.iter().take(schedule.blend_frames as usize) {
            let (mut work, mut pixels) = (baseline_work.clone(), baseline_vga.clone());
            indexedmap::compose_native_ai_item_damage_blend_frame(
                &mut work,
                &mut pixels,
                &baseline_work,
                map.units.as_ref(),
                state.native_map_selector_cache.as_ref(),
                &visible,
                view.camera_x,
                view.camera_y,
                state.native_map_cycle_state.idle,
                blend,
                schedule.raw_base as u8,
            )?;
            frames.push(NativeCompoundPresentedFrame {
                image: Some(image(pixels)),
                delay: schedule.blend_delay_ticks,
                ..Default::default()
            });
        }
        let publish_at = frames.len();
        let mut post = state.clone();
        for (unit, record) in post.units.iter_mut().zip(transaction.after.as_chunks::<80>().0) {
            sync_native_item_runtime_record(unit, record);
        }
        let mut input = build_native_map_frame_input(
            Some(map),
            self.m.as_ref(),
            Some(&post),
            NativeMapFrameRuntime { hud, ..Default::default() },
        )?;
        let (mut post_transaction_work, mut post_vga) = (baseline_work.clone(), baseline_vga.clone());
        indexedmap::compose_native_frame(&mut post_transaction_work, &mut post_vga, &mut input)?;
        let mut queue = Vec::new();
        for (target, result) in targets.iter().zip(&transaction.damage) {
            let in_camera = (view.camera_x..view.camera_x + 12).contains(&target.x)
                && (view.camera_y - 1..=view.camera_y + 7).contains(&target.y);
            if result.hit {
                queue = battle::append_native_presentation_digits(
                    &queue,
                    result.damage,
                    schedule.digit_bias,
                    target.record_index,
                    in_camera,
                )?;
            } else if in_camera {
                queue = battle::append_native_presentation_miss(&queue, target.record_index);
            }
        }
        for index in 0..schedule.digit_frames {
            let (mut work, mut pixels) = (post_transaction_work.clone(), post_vga.clone());
            indexedmap::compose_native_command_heal_digit_frame(
                &mut work,
                &mut pixels,
                &post_transaction_work,
                &map.command_heal_digits,
                &queue,
                &targets,
                view.camera_x,
                view.camera_y,
                schedule.digit_vertical,
                index,
            )?;
            frames.push(NativeCompoundPresentedFrame {
                image: Some(image(pixels)),
                delay: schedule.digit_frame_delay_ticks,
                ..Default::default()
            });
        }
        frames[0].sound =
            self.separated_command_sfx.get(&80).and_then(|bank| bank.get(&6)).cloned().unwrap_or_default();
        if !os_mute_or_shot(Some(self)) && frames[0].sound.is_empty() {
            return Err(game_err!("native AI item required raw sample unavailable"));
        }
        Ok(NativeAIItemPresentationJob {
            transaction: Some(transaction),
            frames,
            publish_at: publish_at as i64,
            baseline_work,
            baseline_vga,
            post_transaction_work,
            post_vga,
            then: Some(then),
            ..Default::default()
        })
    }

    /// Go：`native_ai_item_presentation.go:233`，`func (g *Game) startNativeAIItemType21Presentation(tx *nativeAIItemTransaction, then func()) (bool, error)`
    ///
    /// type21 呼叫端 `0x2111a` 只共用 `0x1cac7` 的畫面契約，不沿用指令 32 的玩家前導或交易擁有者。
    /// 只移植原版索引色分支；HD 複合地圖屬 R6。
    fn build_native_ai_item_type21_presentation(
        &mut self,
        transaction: NativeAIItemTransaction,
        then: Box<dyn FnOnce(&mut Game) + Send>,
    ) -> Result<NativeAIItemPresentationJob> {
        let Some(route) = transaction.damage_route.filter(|route| {
            route.item_type == 21
                && route.presentation == 0x1cac7
                && transaction.damage.len() == transaction.targets.len()
        }) else {
            return Err(game_err!("native AI item type21 presentation context unavailable"));
        };
        if !self.current_native_system_options().full_presentation_enabled() {
            return Err(game_err!("native AI item indexed presentation unavailable"));
        }
        if self.native_ai_item_presentation.is_some() || self.atk.is_some() || self.indexed_transition.is_some() {
            return Err(game_err!("native AI item presentation already active"));
        }
        let (hud, hud_ready) = self.native_map_hud_input();
        let map = self
            .native_map_assets
            .as_ref()
            .filter(|map| native_map_assets_available(Some(map)))
            .ok_or_else(|| game_err!("native AI item indexed map state unavailable"))?;
        let state = self
            .st
            .as_ref()
            .filter(|state| state.has_native_map_view_state)
            .ok_or_else(|| game_err!("native AI item indexed map state unavailable"))?;
        if self.native_map_work.len() as i64 != indexedmap::NATIVE_UNIT_PRESENT_WORK_SIZE
            || self.native_map_vga.len() as i64 != indexedmap::NATIVE_MAP_VGA_SIZE
            || self.native_map_dac.len() != 256 * 3
        {
            return Err(game_err!("native AI item indexed map state unavailable"));
        }
        let schedule = fdother::build_native_ai_item_type21_tail_schedule(
            route.command_id,
            &map.fdother6,
            &map.command_heal_digits,
        )?;
        if transaction.targets.is_empty() {
            return Err(game_err!("native command heal tail target array unavailable"));
        }
        let targets = transaction
            .targets
            .iter()
            .map(|&index| {
                let unit = &state.units[index];
                if !unit.has_native_map_presentation || !unit.has_map_selector_slot {
                    return Err(game_err!("native command heal tail target lacks raw record presentation"));
                }
                Ok(indexedmap::NativeCommandHealTailTarget {
                    record_index: index as i64,
                    x: i64::from(unit.native_map_presentation.x),
                    y: i64::from(unit.native_map_presentation.y),
                    selector_slot: unit.map_selector_slot,
                })
            })
            .collect::<Result<Vec<_>>>()?;
        let results = targets
            .iter()
            .zip(&transaction.damage)
            .map(|(target, result)| battlepresent::NativeAIItemType21TailResult {
                target_record: target.record_index,
                hit: result.hit,
                damage: result.damage,
            })
            .collect::<Vec<_>>();
        let baseline_work = self.native_map_work.clone();
        let baseline_vga = self.native_map_vga.clone();
        let mut post = state.clone();
        for (unit, record) in post.units.iter_mut().zip(transaction.after.as_chunks::<80>().0) {
            sync_native_item_runtime_record(unit, record);
        }
        if !hud_ready {
            return Err(game_err!("native AI item post-transaction HUD unavailable"));
        }
        let mut input = build_native_map_frame_input(
            Some(map),
            self.m.as_ref(),
            Some(&post),
            NativeMapFrameRuntime { hud, ..Default::default() },
        )?;
        let (mut post_transaction_work, mut post_vga) = (baseline_work.clone(), baseline_vga.clone());
        indexedmap::compose_native_frame(&mut post_transaction_work, &mut post_vga, &mut input)?;
        let view = state.native_map_view_state;
        let tail = battlepresent::build_native_ai_item_type21_tail_frames(
            &baseline_work,
            &baseline_vga,
            &post_transaction_work,
            &post_vga,
            &map.fdother6,
            &map.command_heal_digits,
            &targets,
            &results,
            view.camera_x,
            view.camera_y,
            &schedule,
        )?;
        let palette = color_palette_from_nrgba(&fdother::vga_palette_from_dac(&self.native_map_dac)?);
        let image = |pix: &Vec<u8>| {
            Arc::new(Picture::from_indexed(&IndexedFrame {
                width: 320,
                height: 200,
                pix: pix.clone(),
                palette: palette.clone(),
            }))
        };
        let mut frames = Vec::new();
        let push = |frames: &mut Vec<NativeCompoundPresentedFrame>, layer: &[Vec<u8>], delay: i64| -> Result<()> {
            if layer.is_empty() || delay <= 0 {
                return Err(game_err!("native command32 presented frames unavailable"));
            }
            for pix in layer {
                frames.push(NativeCompoundPresentedFrame { image: Some(image(pix)), delay, ..Default::default() });
            }
            Ok(())
        };
        push(&mut frames, &tail.effect, schedule.effect_delay_ticks)?;
        push(&mut frames, &tail.toggle, native_delay_ticks(schedule.toggle_delay_ms))?;
        let publish_at = frames.len();
        push(&mut frames, &tail.result, 1)?;
        frames[0].sound = self
            .separated_command_sfx
            .get(&80)
            .and_then(|bank| bank.get(&schedule.effect_sample))
            .cloned()
            .unwrap_or_default();
        if !os_mute_or_shot(Some(self)) && frames[0].sound.is_empty() {
            return Err(game_err!("native AI item required raw sample unavailable"));
        }
        Ok(NativeAIItemPresentationJob {
            transaction: Some(transaction),
            frames,
            publish_at: publish_at as i64,
            baseline_work,
            baseline_vga,
            post_transaction_work,
            post_vga,
            then: Some(then),
            ..Default::default()
        })
    }

    pub(crate) fn draw_native_ai_item_presentation(&mut self, screen: &mut Picture) -> bool {
        let Some(job) = self.native_ai_item_presentation.as_mut() else { return false };
        let Some(image) = job.frames.get(job.frame as usize).and_then(|frame| frame.image.as_ref()) else {
            return false;
        };
        let mut draw = DrawImageOptions::default();
        draw.geo_m.scale(2.0, 2.0);
        screen.draw_image(image, &draw);
        job.drawn = true;
        true
    }

    pub(crate) fn acknowledge_native_ai_item_presentation(&mut self) -> bool {
        let Some(job) = self.native_ai_item_presentation.as_mut() else { return false };
        job.drawn = job.frames.get(job.frame as usize).is_some_and(|frame| frame.image.is_some());
        job.drawn
    }

    /// Go：`native_ai_item_presentation.go:531`，`func (g *Game) cancelNativeAIItemPresentation()`
    /// Go 檢查接收者是否為 nil；Rust 的 `self` 不會是 null，該分支不移植。
    pub(crate) fn cancel_native_ai_item_presentation(&mut self) {
        let Some(j) = self.native_ai_item_presentation.take() else {
            return;
        };
        if j.published
            && let Some(tx) = j.transaction.as_ref()
        {
            match self.st.as_mut() {
                Some(st) if tx.before.len() == st.units.len() * 80 => {
                    // Go 逐一檢查 `unit == nil`；Rust 的單位不會是 nil，該分支不移植。
                    for (index, unit) in st.units.iter_mut().enumerate() {
                        sync_native_item_runtime_record(unit, &tx.before[index * 80..(index + 1) * 80]);
                    }
                    self.native_rng_state = tx.rng_before;
                }
                _ => {
                    if self.load_err.is_empty() {
                        self.load_err = String::from("native AI item rollback roster changed");
                    }
                }
            }
        }
        self.native_map_work = j.baseline_work.clone();
        self.native_map_vga = j.baseline_vga.clone();
        self.native_ai_item_presentation = None;
    }

    /// Go：`native_ai_item_presentation.go:568`，`func (g *Game) failNativeAIItemPresentation(err error)`（`r8`，隨
    /// `stepNativeAIItemPresentation` 補移植；Go `Draw` 也呼叫它）。
    pub(crate) fn fail_native_ai_item_presentation(&mut self, err: &GameError) {
        self.cancel_native_ai_item_presentation();
        self.load_err = format!("native AI item presentation: {}", err.message());
    }

    /// Go：`native_ai_item_presentation.go:573`，`func (g *Game) stepNativeAIItemPresentation()`
    pub(crate) fn step_native_ai_item_presentation(&mut self) {
        let Some(j) = self.native_ai_item_presentation.as_mut() else {
            return;
        };
        if j.holding {
            if j.hold > 0 {
                j.hold -= 1;
                return;
            }
            let then = j.then.take();
            self.native_ai_item_presentation = None;
            if let Some(then) = then {
                then(self);
            }
            return;
        }
        if !j.drawn || j.frame < 0 || j.frame >= j.frames.len() as i64 {
            return;
        }
        j.drawn = false;
        let frame = &mut j.frames[j.frame as usize];
        if !frame.sound_played && !frame.sound.is_empty() {
            let sound = frame.sound.clone();
            frame.sound_played = true;
            self.play_raw(&sound);
        }
        let j = self.native_ai_item_presentation.as_mut().unwrap();
        j.repeat += 1;
        if j.repeat < j.frames[j.frame as usize].delay {
            return;
        }
        j.repeat = 0;
        let next = j.frame + 1;
        if next == j.publish_at && !j.published {
            let tx = j.transaction.clone();
            if let Err(err) = self.commit_native_ai_item_transaction(tx.as_ref()) {
                self.fail_native_ai_item_presentation(&err);
                return;
            }
            let j = self.native_ai_item_presentation.as_mut().unwrap();
            self.native_map_work = j.post_transaction_work.clone();
            self.native_map_vga = j.post_vga.clone();
            j.published = true;
        }
        let j = self.native_ai_item_presentation.as_mut().unwrap();
        j.frame = next;
        if j.frame >= j.frames.len() as i64 {
            if !j.published {
                self.fail_native_ai_item_presentation(&game_err!(
                    "required item publication boundary was not presented"
                ));
                return;
            }
            j.frame = j.frames.len() as i64 - 1;
            (j.holding, j.hold) = (true, native_delay_ticks(500));
        }
    }
}

#[cfg(test)]
#[path = "native_ai_item_presentation_test.rs"]
mod native_ai_item_presentation_test;
