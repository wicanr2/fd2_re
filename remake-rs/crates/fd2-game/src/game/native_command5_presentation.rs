//! Go 來源：`remake/cmd/fd2/native_command5_presentation.go`（577 行）。
//!
//! 骨架：Go 檔的 6 個函式中產生 1 個（階段 0 清單），其餘標 `r8`，見 `PORTING.json`。

use super::*;

/// Go：`native_command5_presentation.go:15`
#[derive(Debug, Clone, Copy, Default, PartialEq, Eq, PartialOrd, Ord, Hash)]
pub(crate) struct NativeCommand5PresentationPhase(pub(crate) u8);

/// Go 原式：`iota`
pub(crate) const NATIVE_COMMAND5_COMMON_PRELUDE: NativeCommand5PresentationPhase = NativeCommand5PresentationPhase(0);

/// Go：沿用上一個常數的式子（`iota` 序列）。
pub(crate) const NATIVE_COMMAND5_ACTOR: NativeCommand5PresentationPhase = NativeCommand5PresentationPhase(1);

/// Go：沿用上一個常數的式子（`iota` 序列）。
pub(crate) const NATIVE_COMMAND5_HANDLER: NativeCommand5PresentationPhase = NativeCommand5PresentationPhase(2);

/// Go：沿用上一個常數的式子（`iota` 序列）。
pub(crate) const NATIVE_COMMAND5_COMMON_TAIL: NativeCommand5PresentationPhase = NativeCommand5PresentationPhase(3);

/// Go：`native_command5_presentation.go:24`
#[derive(Debug, Clone, Default, PartialEq, Eq)]
pub(crate) struct NativeCommand5HandlerFrame {
    pub(crate) image: Option<Arc<Picture>>,
    pub(crate) target_index: i64,
    pub(crate) hp_stage: i64,
    pub(crate) play_primary: bool,
    pub(crate) play_secondary: bool,
}

/// Go：`native_command5_presentation.go:32`
///
/// 不 derive `Debug`／`Clone`：欄位含 callback（`Box<dyn FnOnce>`）、檔案或亂數產生器等做不到的型別。
#[derive(Default)]
pub(crate) struct NativeCommand5PresentationJob {
    pub(crate) command_id: i64,
    pub(crate) actor: Option<UnitIdx>,
    pub(crate) plan: Option<battle::NativeCommandDamagePlan>,
    pub(crate) prelude: Vec<Option<Arc<Picture>>>,
    pub(crate) actor_black: Vec<Option<Arc<Picture>>>,
    pub(crate) actor_pulse: Vec<Option<Arc<Picture>>>,
    pub(crate) actor_specs: Vec<battlepresent::NativeCommand0ActorFrame>,
    pub(crate) handler: Vec<NativeCommand5HandlerFrame>,
    pub(crate) tail: Vec<Option<Arc<Picture>>>,
    pub(crate) hd: Option<ModernHDCommand5Frames>,
    pub(crate) phase: NativeCommand5PresentationPhase,
    pub(crate) frame: i64,
    pub(crate) pulse_black: bool,
    pub(crate) drawn: bool,
    pub(crate) mp_published: bool,
    pub(crate) actor_mp_before: i64,
    pub(crate) actor_acted_before: bool,
    pub(crate) target_hp_before: Vec<i64>,
    pub(crate) actor_sfx: Vec<u8>,
    pub(crate) target_sfx: Vec<u8>,
    pub(crate) then: Option<Box<dyn FnOnce(&mut Game, Vec<battle::NativeCommandDamageResult>) + Send>>,
}

impl Game {
    /// Go：`native_command5_presentation.go:441`，`func (g *Game) stepNativeCommand5Presentation()`
    pub(crate) fn step_native_command5_presentation(&mut self) {
        let Some(j) = self.native_cmd5_presentation.as_mut() else {
            return;
        };
        if !j.drawn {
            return;
        }
        j.drawn = false;
        match j.phase {
            NATIVE_COMMAND5_COMMON_PRELUDE => {
                j.frame += 1;
                if j.frame >= j.prelude.len() as i64 {
                    (j.phase, j.frame) = (NATIVE_COMMAND5_ACTOR, 0);
                }
            }
            NATIVE_COMMAND5_ACTOR => {
                let spec = &j.actor_specs[j.frame as usize];
                let (spec_pulse, spec_publish_mp) = (spec.pulse, spec.publish_mp);
                if spec_pulse && !j.pulse_black {
                    if spec_publish_mp && !j.mp_published {
                        if let Err(err) = self.publish_native_command5_mp() {
                            self.fail_native_command5_presentation(&err);
                            return;
                        }
                        let sfx =
                            self.native_cmd5_presentation.as_ref().expect("Go：j 仍是同一個工作").actor_sfx.clone();
                        self.play_raw(&sfx);
                    }
                    let j = self.native_cmd5_presentation.as_mut().expect("Go：j 仍是同一個工作");
                    j.pulse_black = true;
                    return;
                }
                if spec_publish_mp && !j.mp_published {
                    if let Err(err) = self.publish_native_command5_mp() {
                        self.fail_native_command5_presentation(&err);
                        return;
                    }
                    let sfx = self.native_cmd5_presentation.as_ref().expect("Go：j 仍是同一個工作").actor_sfx.clone();
                    self.play_raw(&sfx);
                }
                let j = self.native_cmd5_presentation.as_mut().expect("Go：j 仍是同一個工作");
                j.pulse_black = false;
                j.frame += 1;
                if j.frame >= j.actor_black.len() as i64 {
                    if !j.mp_published {
                        self.fail_native_command5_presentation(&game_err!("actor MP marker was not presented"));
                        return;
                    }
                    (j.phase, j.frame) = (NATIVE_COMMAND5_HANDLER, 0);
                }
            }
            NATIVE_COMMAND5_HANDLER => {
                let frame = &j.handler[j.frame as usize];
                let (target_index, hp_stage) = (frame.target_index, frame.hp_stage);
                let (play1, play2) = (frame.play_primary, frame.play_secondary);
                if hp_stage != 0 {
                    let units = presentation_units(&mut self.st);
                    let j = self.native_cmd5_presentation.as_mut().expect("Go：j 仍是同一個工作");
                    if let Err(err) =
                        battle::apply_native_command_damage_stage(units, j.plan.as_mut(), target_index, hp_stage)
                    {
                        self.fail_native_command5_presentation(&err.into());
                        return;
                    }
                }
                if play1 {
                    let sfx = self.native_cmd5_presentation.as_ref().expect("Go：j 仍是同一個工作").target_sfx.clone();
                    self.play_raw(&sfx);
                }
                if play2 {
                    let sfx = self.native_cmd5_presentation.as_ref().expect("Go：j 仍是同一個工作").target_sfx.clone();
                    self.play_raw(&sfx);
                }
                let j = self.native_cmd5_presentation.as_mut().expect("Go：j 仍是同一個工作");
                j.frame += 1;
                if j.frame < j.handler.len() as i64 {
                    return;
                }
                (j.phase, j.frame) = (NATIVE_COMMAND5_COMMON_TAIL, 0);
            }
            NATIVE_COMMAND5_COMMON_TAIL => {
                j.frame += 1;
                if j.frame < j.tail.len() as i64 {
                    return;
                }
                // 未命中的目標走 0x2AF61 分支，沒有 HP 分段畫格；數值上 HP 不變，直接收到終段。
                let (misses, damage_stages) = j.plan.as_ref().map_or((Vec::new(), 0), |plan| {
                    let misses = plan
                        .results
                        .iter()
                        .enumerate()
                        .filter(|(_, result)| !result.native_command_damage.hit)
                        .map(|(index, _)| index as i64)
                        .collect::<Vec<_>>();
                    (misses, plan.damage_stages)
                });
                for index in misses {
                    let units = presentation_units(&mut self.st);
                    let j = self.native_cmd5_presentation.as_mut().expect("Go：j 仍是同一個工作");
                    if let Err(err) =
                        battle::apply_native_command_damage_stage(units, j.plan.as_mut(), index, damage_stages)
                    {
                        self.fail_native_command5_presentation(&err.into());
                        return;
                    }
                }
                // plan.RNGAfter 已含 handler 的 mode 0／重置／marker 抖動與每個目標的
                // 0x1C75E 擲骰（r4 收據 seq 1451：18 步），是整段演出結束時的全域亂數。
                let units = presentation_units(&mut self.st);
                let j = self.native_cmd5_presentation.as_mut().expect("Go：j 仍是同一個工作");
                if let Err(err) = battle::complete_native_command_damage(units, j.plan.as_mut()) {
                    self.fail_native_command5_presentation(&err.into());
                    return;
                }
                let plan = j.plan.as_ref().expect("Go：CompleteNativeCommandDamage 已檢查 plan");
                self.native_rng_state = plan.rng_after;
                let results = plan.results.clone();
                let then = j.then.take();
                self.native_cmd5_presentation = None;
                if let Some(then) = then {
                    then(self, results);
                }
            }
            _ => {}
        }
    }

    /// Go `stepNativeCommand5Presentation` 兩處相同的 MP 發布：`ApplyNativeCommandDamageMP` 成功後設 `mpPublished`。
    fn publish_native_command5_mp(&mut self) -> Result<()> {
        let units = presentation_units(&mut self.st);
        let j = self.native_cmd5_presentation.as_mut().expect("Go：j 仍是同一個工作");
        battle::apply_native_command_damage_mp(units, j.plan.as_mut())?;
        j.mp_published = true;
        Ok(())
    }

    /// Go：`native_command5_presentation.go:426`，`func (g *Game) failNativeCommand5Presentation(err error)`
    /// （`r8`，隨呼叫端 `stepNativeCommand5Presentation` 補移植）。
    pub(crate) fn fail_native_command5_presentation(&mut self, err: &GameError) {
        let Some(j) = self.native_cmd5_presentation.take() else {
            return;
        };
        let units = presentation_units(&mut self.st);
        if let Some(actor) = j.actor {
            (units[actor].mp, units[actor].acted) = (j.actor_mp_before, j.actor_acted_before);
        }
        if let Some(plan) = j.plan.as_ref() {
            for (index, result) in plan.results.iter().enumerate() {
                if let Some(target) = result.target
                    && index < j.target_hp_before.len()
                {
                    units[target].hp = j.target_hp_before[index];
                }
            }
        }
        self.load_err = format!("native command{} presentation: {}", j.command_id, err.message());
    }
}

#[cfg(test)]
#[path = "native_command5_presentation/native_command5_presentation_test.rs"]
mod native_command5_presentation_test;
