//! Go 來源：`remake/cmd/fd2/native_command9_ai_presentation.go`（456 行）。
//!
//! 骨架：Go 檔的 4 個函式中產生 1 個（階段 0 清單），其餘標 `r8`，見 `PORTING.json`。

use super::*;

/// Go：`native_command9_ai_presentation.go:15`
#[derive(Debug, Clone, Copy, Default, PartialEq, Eq, PartialOrd, Ord, Hash)]
pub(crate) struct NativeCommand9AIPresentationPhase(pub(crate) u8);

/// Go 原式：`iota`
pub(crate) const NATIVE_COMMAND9_AI_COMMON_PRELUDE: NativeCommand9AIPresentationPhase =
    NativeCommand9AIPresentationPhase(0);

/// Go：沿用上一個常數的式子（`iota` 序列）。
pub(crate) const NATIVE_COMMAND9_AI_ACTOR: NativeCommand9AIPresentationPhase = NativeCommand9AIPresentationPhase(1);

/// Go：沿用上一個常數的式子（`iota` 序列）。
pub(crate) const NATIVE_COMMAND9_AI_HANDLER: NativeCommand9AIPresentationPhase = NativeCommand9AIPresentationPhase(2);

/// Go：沿用上一個常數的式子（`iota` 序列）。
pub(crate) const NATIVE_COMMAND9_AI_COMMON_TAIL: NativeCommand9AIPresentationPhase =
    NativeCommand9AIPresentationPhase(3);

/// Go：`native_command9_ai_presentation.go:24`
#[derive(Debug, Clone, Default, PartialEq, Eq)]
pub(crate) struct NativeCommand9AIHandlerFrame {
    pub(crate) image: Option<Arc<Picture>>,
    pub(crate) hp_stage: i64,
    pub(crate) play_sub1: bool,
    pub(crate) play_sub2: bool,
    pub(crate) hold_after: i64,
}

/// Go：`native_command9_ai_presentation.go:31`
///
/// 不 derive `Debug`／`Clone`：欄位含 callback（`Box<dyn FnOnce>`）、檔案或亂數產生器等做不到的型別。
#[derive(Default)]
pub(crate) struct NativeCommand9AIPresentationJob {
    pub(crate) hd: Option<ModernHDOrdinaryFrames>,
    pub(crate) actor: Option<UnitIdx>,
    pub(crate) plan: Option<battle::NativeCommandDamagePlan>,
    pub(crate) prelude: Vec<Option<Arc<Picture>>>,
    pub(crate) actor_black: Vec<Option<Arc<Picture>>>,
    pub(crate) actor_pulse: Vec<Option<Arc<Picture>>>,
    pub(crate) actor_specs: Vec<battlepresent::NativeCommand0ActorFrame>,
    pub(crate) handler: Vec<NativeCommand9AIHandlerFrame>,
    pub(crate) tail: Vec<Option<Arc<Picture>>>,
    pub(crate) phase: NativeCommand9AIPresentationPhase,
    pub(crate) frame: i64,
    pub(crate) hold: i64,
    pub(crate) pulse_black: bool,
    pub(crate) drawn: bool,
    pub(crate) mp_published: bool,
    pub(crate) actor_mp_before: i64,
    pub(crate) actor_acted_before: bool,
    pub(crate) target_hp_before: i64,
    pub(crate) then: Option<Box<dyn FnOnce(&mut Game, Vec<battle::NativeCommandDamageResult>) + Send>>,
}

impl Game {
    /// Go：`native_command9_ai_presentation.go:319`，`func (g *Game) stepNativeCommand9AIPresentation()`
    pub(crate) fn step_native_command9_ai_presentation(&mut self) {
        let Some(j) = self.native_cmd9_ai_presentation.as_mut() else {
            return;
        };
        if !j.drawn {
            return;
        }
        j.drawn = false;
        match j.phase {
            NATIVE_COMMAND9_AI_COMMON_PRELUDE => {
                j.frame += 1;
                if j.frame >= j.prelude.len() as i64 {
                    (j.phase, j.frame) = (NATIVE_COMMAND9_AI_ACTOR, 0);
                }
            }
            NATIVE_COMMAND9_AI_ACTOR => {
                let spec = &j.actor_specs[j.frame as usize];
                let (spec_pulse, spec_publish_mp) = (spec.pulse, spec.publish_mp);
                if spec_pulse && !j.pulse_black {
                    if spec_publish_mp && !j.mp_published {
                        if let Err(err) = self.publish_native_command9_ai_mp() {
                            self.fail_native_command9_ai_presentation(&err);
                            return;
                        }
                        let sfx = self.sfx_command8_actor.clone();
                        self.play_raw(&sfx);
                    }
                    let j = self.native_cmd9_ai_presentation.as_mut().expect("Go：j 仍是同一個工作");
                    j.pulse_black = true;
                    return;
                }
                if spec_publish_mp && !j.mp_published {
                    if let Err(err) = self.publish_native_command9_ai_mp() {
                        self.fail_native_command9_ai_presentation(&err);
                        return;
                    }
                    let sfx = self.sfx_command8_actor.clone();
                    self.play_raw(&sfx);
                }
                let j = self.native_cmd9_ai_presentation.as_mut().expect("Go：j 仍是同一個工作");
                j.pulse_black = false;
                j.frame += 1;
                if j.frame >= j.actor_black.len() as i64 {
                    if !j.mp_published {
                        self.fail_native_command9_ai_presentation(&game_err!("actor MP marker was not presented"));
                        return;
                    }
                    (j.phase, j.frame) = (NATIVE_COMMAND9_AI_HANDLER, 0);
                }
            }
            NATIVE_COMMAND9_AI_HANDLER => {
                if j.hold > 0 {
                    j.hold -= 1;
                    if j.hold == 0 {
                        j.frame += 1;
                        if j.frame >= j.handler.len() as i64 {
                            (j.phase, j.frame) = (NATIVE_COMMAND9_AI_COMMON_TAIL, 0);
                        }
                    }
                    return;
                }
                let frame = &j.handler[j.frame as usize];
                let hp_stage = frame.hp_stage;
                let (play1, play2, hold_after) = (frame.play_sub1, frame.play_sub2, frame.hold_after);
                if hp_stage != 0 {
                    let units = presentation_units(&mut self.st);
                    let j = self.native_cmd9_ai_presentation.as_mut().expect("Go：j 仍是同一個工作");
                    if let Err(err) = battle::apply_native_command_damage_stage(units, j.plan.as_mut(), 0, hp_stage) {
                        self.fail_native_command9_ai_presentation(&err.into());
                        return;
                    }
                }
                if play1 {
                    let sfx = self.sfx_command8_sub1.clone();
                    self.play_raw(&sfx);
                }
                if play2 {
                    let sfx = self.sfx_command8_sub2.clone();
                    self.play_raw(&sfx);
                }
                if hold_after > 0 {
                    self.native_cmd9_ai_presentation.as_mut().expect("Go：j 仍是同一個工作").hold = hold_after;
                    return;
                }
                let j = self.native_cmd9_ai_presentation.as_mut().expect("Go：j 仍是同一個工作");
                j.frame += 1;
                if j.frame >= j.handler.len() as i64 {
                    (j.phase, j.frame) = (NATIVE_COMMAND9_AI_COMMON_TAIL, 0);
                }
            }
            NATIVE_COMMAND9_AI_COMMON_TAIL => {
                j.frame += 1;
                if j.frame < j.tail.len() as i64 {
                    return;
                }
                let units = presentation_units(&mut self.st);
                let j = self.native_cmd9_ai_presentation.as_mut().expect("Go：j 仍是同一個工作");
                if let Err(err) = battle::complete_native_command_damage(units, j.plan.as_mut()) {
                    self.fail_native_command9_ai_presentation(&err.into());
                    return;
                }
                let plan = j.plan.as_ref().expect("Go：CompleteNativeCommandDamage 已檢查 plan");
                self.native_rng_state = plan.rng_after;
                let results = plan.results.clone();
                let then = j.then.take();
                self.native_cmd9_ai_presentation = None;
                if let Some(then) = then {
                    then(self, results);
                }
            }
            _ => {}
        }
    }

    /// Go `stepNativeCommand9AIPresentation` 兩處相同的 MP 發布：`ApplyNativeCommandDamageMP` 成功後設 `mpPublished`。
    fn publish_native_command9_ai_mp(&mut self) -> Result<()> {
        let units = presentation_units(&mut self.st);
        let j = self.native_cmd9_ai_presentation.as_mut().expect("Go：j 仍是同一個工作");
        battle::apply_native_command_damage_mp(units, j.plan.as_mut())?;
        j.mp_published = true;
        Ok(())
    }

    /// Go：`native_command9_ai_presentation.go:306`，`func (g *Game) failNativeCommand9AIPresentation(err error)`
    /// （`r8`，隨呼叫端 `stepNativeCommand9AIPresentation` 補移植）。
    pub(crate) fn fail_native_command9_ai_presentation(&mut self, err: &GameError) {
        let Some(j) = self.native_cmd9_ai_presentation.take() else {
            return;
        };
        let units = presentation_units(&mut self.st);
        if let Some(actor) = j.actor {
            (units[actor].mp, units[actor].acted) = (j.actor_mp_before, j.actor_acted_before);
        }
        if let Some(plan) = j.plan.as_ref()
            && plan.results.len() == 1
            && let Some(target) = plan.results[0].target
        {
            units[target].hp = j.target_hp_before;
        }
        self.load_err = format!("native command9 AI presentation: {}", err.message());
    }
}

#[cfg(test)]
#[path = "native_command9_ai_presentation/native_command9_ai_presentation_test.rs"]
mod native_command9_ai_presentation_test;
