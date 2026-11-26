//! Go 來源：`remake/cmd/fd2/native_ai_mode11_execute.go`（155 行）。
//!
//! 指令／道具階段以 `Game::native_ai_action_after` 承接 Go `executeNativeAIActionWithContinuation` 的 `after`。

use super::*;

impl Game {
    /// 0x15311 已選定原始 producer 之後的兩段直接分派；兩段之間不再問 NextAIPlan，
    /// 0x15311 → 0x1548E／0x14121 → 0x13FD4 是同一個有序 owner。
    ///
    /// Go：`native_ai_mode11_execute.go:12`，`func (g *Game) startNativeAIMode11(plan *battle.AIPlan)`
    pub(crate) fn start_native_ai_mode11(&mut self, plan: battle::AIPlan) {
        if plan.u.is_none() || plan.native_mode11_stages.is_empty() {
            self.load_err = String::from("native AI mode 11: executable stages are unavailable");
            self.ai_busy = false;
            return;
        }
        self.run_native_ai_mode11_stage(Arc::new(plan), 0);
    }

    /// Go：`native_ai_mode11_execute.go:24`，`func (g *Game) runNativeAIMode11Stage(plan *battle.AIPlan, index int)`
    fn run_native_ai_mode11_stage(&mut self, plan: Arc<battle::AIPlan>, index: usize) {
        if !self.result.is_empty() || index >= plan.native_mode11_stages.len() {
            self.finish_successful_unit_action(plan.u, None);
            return;
        }
        let stage = &plan.native_mode11_stages[index];
        let next_plan = Arc::clone(&plan);
        let next: Box<dyn FnOnce(&mut Game) + Send> =
            Box::new(move |g: &mut Game| g.run_native_ai_mode11_stage(next_plan, index + 1));
        if let Some(recovery) = stage.recovery {
            if let Err(err) = self.begin_native_ai_idle_recovery(plan.u, recovery, Some(next)) {
                self.load_err = format!("native AI mode 11 0x13fd4: {}", err.message());
                self.ai_busy = false;
            }
            return;
        }
        let Some(action) = stage.action.clone() else {
            // 0x14121 回 0，但 0x13FD4 的 HP／閘門預檢拒收；原版共用尾段不做任何動作。
            next(self);
            return;
        };
        if action.native_mode_write_range_zero {
            let st = self.st.as_mut().unwrap();
            st.native_map_range_mode = 0;
            st.has_native_map_range_mode_state = true;
        }
        let action = Arc::new(action);
        let act_plan = Arc::clone(&action);
        let act: Box<dyn FnOnce(&mut Game) + Send> = Box::new(move |g: &mut Game| {
            let action = act_plan;
            if action.native_action_kind == battle::NATIVE_AI_ACTION_COMMAND
                || action.native_action_kind == battle::NATIVE_AI_ACTION_ITEM
            {
                g.native_ai_action_after = Some(next);
                if let Err(err) = g.execute_native_ai_action(&action) {
                    g.native_ai_action_after = None;
                    g.load_err = format!("native AI mode 11 action: {}", err.message());
                    g.ai_busy = false;
                }
                return;
            }
            if action.native_action_kind != battle::NATIVE_AI_ACTION_PHYSICAL {
                g.finish_successful_unit_action(action.u, Some(next));
                return;
            }
            g.execute_native_ai_mode11_physical(&action, next);
        });
        if action.path.len() >= 2 {
            let u = action.u.unwrap();
            let (x, y) = {
                let unit = &self.st.as_ref().unwrap().units[u];
                (unit.x, unit.y)
            };
            self.ai_focus_cursor(x, y);
            self.walk = Some(WalkAnim {
                u: Some(u),
                path: action.path.clone(),
                then: Some(act),
                follow_view: true,
                ..Default::default()
            });
            return;
        }
        act(self);
    }

    /// 0x1548E 的選定結果演出；沿用已閉合 0x14EF0 物理路線的型別化傷害與 FIGANI 排程，路線標籤保持原始。
    ///
    /// Go：`native_ai_mode11_execute.go:82`，`func (g *Game) executeNativeAIMode11Physical(plan *battle.AIPlan, after func())`
    fn execute_native_ai_mode11_physical(&mut self, plan: &battle::AIPlan, after: Box<dyn FnOnce(&mut Game) + Send>) {
        let st = self.st.as_ref().unwrap();
        let (Some(actor), Some(target)) = (plan.u, plan.target.filter(|&t| st.units[t].alive())) else {
            self.load_err = String::from("native AI mode 11 0x1548e: raw target is unavailable");
            self.ai_busy = false;
            return;
        };
        let (a_fig, t_fig) = (st.units[actor].battle_fig, st.units[target].battle_fig);
        if let Err(err) = self.ensure_native_attack_presentation(a_fig, t_fig) {
            self.load_err =
                format!("native AI mode 11 0x1548e FIGANI unavailable: {a_fig} -> {t_fig}: {}", err.message());
            self.ai_busy = false;
            return;
        }
        if !self.native_attack_presentation_available(a_fig, t_fig) {
            self.load_err = format!("native AI mode 11 0x1548e FIGANI unavailable: {a_fig} -> {t_fig}");
            self.ai_busy = false;
            return;
        }
        let scene = match self.prepare_native_physical_scene(Some(actor), target) {
            Ok(scene) => scene,
            Err(err) => {
                self.load_err = format!("native AI mode 11 physical scene unavailable: {}", err.message());
                self.ai_busy = false;
                return;
            }
        };
        let (ax, ay, tx, ty) = {
            let units = &self.st.as_ref().unwrap().units;
            (units[actor].x, units[actor].y, units[target].x, units[target].y)
        };
        // 0x154AD 聚焦自己在 0x14B78 移動之前；有走路時呼叫端已在原位聚焦過。
        if plan.path.len() < 2 {
            self.ai_focus_cursor(ax, ay);
        }
        self.ai_focus_cursor(tx, ty);
        let st = self.st.as_mut().unwrap();
        st.units[actor].set_map_pose(dir_toward(ax, ay, tx, ty));
        let hp0 = st.units[target].hp;
        let full = match self.resolve_physical_attack_full(Some(actor), Some(target)) {
            Ok(full) => full,
            Err(err) => {
                self.load_err = format!("native AI mode 11 0x1548e: {}", err.message());
                self.ai_busy = false;
                return;
            }
        };
        let attack = full.attack.clone();
        self.award_death_reward(Some(target), Some(actor));
        let units = &self.st.as_ref().unwrap().units;
        let message = match player_physical_attack_message(
            self.locale_catalog.as_ref(),
            Some(&units[actor]),
            Some(&units[target]),
            &attack,
        ) {
            Ok(message) => message,
            Err(err) => {
                self.load_err = format!("native AI mode 11 locale: {}", err.message());
                self.ai_busy = false;
                return;
            }
        };
        self.publish_physical_attack_message(&message);
        let units = &self.st.as_ref().unwrap().units;
        let (a, t) = (&units[actor], &units[target]);
        let name = |u: &battle::Unit| if u.name.is_empty() { u.cls_name.clone() } else { u.name.clone() };
        let (attacker, defender) = (name(a), name(t));
        let terrain = self.terrain_at(t.x, t.y);
        let mut atk = self.new_atk_anim(
            a.battle_fig,
            t.battle_fig,
            &attacker,
            &defender,
            a.hp,
            a.max_hp,
            a.lv,
            a.mp,
            a.max_mp,
            t.lv,
            t.mp,
            t.max_mp,
            hp0,
            t.hp,
            t.max_hp,
            terrain,
            a.camp == battle::OWN,
        );
        self.attach_counter_presentation(atk.as_mut(), Some(actor), Some(target), &attack);
        let Some(mut anim) = atk else {
            self.load_err = format!("native AI mode 11 0x1548e FIGANI unavailable: {a_fig} -> {t_fig}");
            self.ai_busy = false;
            return;
        };
        anim.native_scene = scene;
        if let Err(err) = self.attach_native_physical_body(anim.native_scene.as_mut(), &full) {
            self.load_err = format!("native AI mode 11 physical body: {}", err.message());
            self.atk = None;
            self.ai_busy = false;
            return;
        }
        anim.after = Some(Box::new(move |g: &mut Game| g.finish_successful_unit_action(Some(actor), Some(after))));
        self.atk = Some(anim);
        self.check_result();
    }
}
