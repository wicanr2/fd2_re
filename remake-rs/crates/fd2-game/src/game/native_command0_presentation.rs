//! Go 來源：`remake/cmd/fd2/native_command0_presentation.go`（546 行）。
//!
//! 骨架：Go 檔的 9 個函式中產生 1 個（階段 0 清單），其餘標 `r8`，見 `PORTING.json`。

use super::*;

/// Go：`native_command0_presentation.go:17`
#[derive(Debug, Clone, Copy, Default, PartialEq, Eq, PartialOrd, Ord, Hash)]
pub(crate) struct NativeCommand0PresentationPhase(pub(crate) u8);

/// Go 原式：`iota`
pub(crate) const NATIVE_COMMAND0_PRELUDE: NativeCommand0PresentationPhase = NativeCommand0PresentationPhase(0);

/// Go：沿用上一個常數的式子（`iota` 序列）。
pub(crate) const NATIVE_COMMAND0_ACTOR: NativeCommand0PresentationPhase = NativeCommand0PresentationPhase(1);

/// Go：沿用上一個常數的式子（`iota` 序列）。
pub(crate) const NATIVE_COMMAND0_TARGET: NativeCommand0PresentationPhase = NativeCommand0PresentationPhase(2);

/// Go：沿用上一個常數的式子（`iota` 序列）。
pub(crate) const NATIVE_COMMAND0_TAIL: NativeCommand0PresentationPhase = NativeCommand0PresentationPhase(3);

/// Go：`native_command0_presentation.go:26`
///
/// 不 derive `Debug`／`Clone`：欄位含 callback（`Box<dyn FnOnce>`）、檔案或亂數產生器等做不到的型別。
#[derive(Default)]
pub(crate) struct NativeCommand0PresentationJob {
    pub(crate) actor: Option<UnitIdx>,
    pub(crate) target: Option<UnitIdx>,
    pub(crate) plan: Option<battle::NativeCommandDamagePlan>,
    pub(crate) prelude: Vec<Option<Arc<Picture>>>,
    pub(crate) actor_black: Vec<Option<Arc<Picture>>>,
    pub(crate) actor_pulse: Vec<Option<Arc<Picture>>>,
    pub(crate) actor_specs: Vec<battlepresent::NativeCommand0ActorFrame>,
    pub(crate) target_frames: Vec<Option<Arc<Picture>>>,
    pub(crate) target_impact: Vec<bool>,
    pub(crate) tail: Vec<Option<Arc<Picture>>>,
    pub(crate) hd: Option<ModernHDCommand0Frames>,
    pub(crate) phase: NativeCommand0PresentationPhase,
    pub(crate) frame: i64,
    pub(crate) pulse_black: bool,
    pub(crate) drawn: bool,
    pub(crate) mp_published: bool,
    pub(crate) damage_stages: i64,
    pub(crate) actor_mp_before: i64,
    pub(crate) actor_acted_before: bool,
    pub(crate) target_hp_before: i64,
    pub(crate) then: Option<Box<dyn FnOnce(&mut Game, Vec<battle::NativeCommandDamageResult>) + Send>>,
}

impl Game {
    /// Go：`native_command0_presentation.go:420`，`func (g *Game) stepNativeCommand0Presentation()`
    pub(crate) fn step_native_command0_presentation(&mut self) {
        let Some(j) = self.native_cmd0_presentation.as_mut() else {
            return;
        };
        if !j.drawn {
            return;
        }
        j.drawn = false;
        match j.phase {
            NATIVE_COMMAND0_PRELUDE => {
                j.frame += 1;
                if j.frame >= j.prelude.len() as i64 {
                    (j.phase, j.frame) = (NATIVE_COMMAND0_ACTOR, 0);
                }
            }
            NATIVE_COMMAND0_ACTOR => {
                let spec = &j.actor_specs[j.frame as usize];
                let (spec_pulse, spec_publish_mp) = (spec.pulse, spec.publish_mp);
                if spec_pulse && !j.pulse_black {
                    if spec_publish_mp && !j.mp_published {
                        if let Err(err) = self.publish_native_command0_mp() {
                            self.fail_native_command0_presentation(&err);
                            return;
                        }
                        let sfx = self.sfx_command0_actor.clone();
                        self.play_raw(&sfx);
                    }
                    let j = self.native_cmd0_presentation.as_mut().expect("Go：j 仍是同一個工作");
                    j.pulse_black = true;
                    return;
                }
                if spec_publish_mp && !j.mp_published {
                    if let Err(err) = self.publish_native_command0_mp() {
                        self.fail_native_command0_presentation(&err);
                        return;
                    }
                    let sfx = self.sfx_command0_actor.clone();
                    self.play_raw(&sfx);
                }
                let j = self.native_cmd0_presentation.as_mut().expect("Go：j 仍是同一個工作");
                j.pulse_black = false;
                j.frame += 1;
                if j.frame >= j.actor_black.len() as i64 {
                    if !j.mp_published {
                        self.fail_native_command0_presentation(&game_err!("actor MP marker was not presented"));
                        return;
                    }
                    (j.phase, j.frame) = (NATIVE_COMMAND0_TARGET, 0);
                }
            }
            NATIVE_COMMAND0_TARGET => {
                if j.target_impact[j.frame as usize] {
                    j.damage_stages += 1;
                    let stage = j.damage_stages;
                    let units = presentation_units(&mut self.st);
                    let j = self.native_cmd0_presentation.as_mut().expect("Go：j 仍是同一個工作");
                    if let Err(err) = battle::apply_native_command_damage_stage(units, j.plan.as_mut(), 0, stage) {
                        self.fail_native_command0_presentation(&err.into());
                        return;
                    }
                    let sfx = self.sfx_command0_target.clone();
                    self.play_raw(&sfx);
                }
                let j = self.native_cmd0_presentation.as_mut().expect("Go：j 仍是同一個工作");
                j.frame += 1;
                if j.frame >= j.target_frames.len() as i64 {
                    if j.damage_stages != figani::NATIVE_COMMAND0_DAMAGE_STAGES {
                        self.fail_native_command0_presentation(&game_err!("target impact stages incomplete"));
                        return;
                    }
                    (j.phase, j.frame) = (NATIVE_COMMAND0_TAIL, 0);
                }
            }
            NATIVE_COMMAND0_TAIL => {
                j.frame += 1;
                if j.frame < j.tail.len() as i64 {
                    return;
                }
                let units = presentation_units(&mut self.st);
                let j = self.native_cmd0_presentation.as_mut().expect("Go：j 仍是同一個工作");
                if let Err(err) = battle::complete_native_command_damage(units, j.plan.as_mut()) {
                    self.fail_native_command0_presentation(&err.into());
                    return;
                }
                let plan = j.plan.as_ref().expect("Go：CompleteNativeCommandDamage 已檢查 plan");
                self.native_rng_state = plan.rng_after;
                let results = plan.results.clone();
                let then = j.then.take();
                self.native_cmd0_presentation = None;
                if let Some(then) = then {
                    then(self, results);
                }
            }
            _ => {}
        }
    }

    /// Go `stepNativeCommand0Presentation` 兩處相同的 MP 發布（`native_command0_presentation.go:438`、`447`）：
    /// `ApplyNativeCommandDamageMP` 成功後設 `mpPublished`。
    fn publish_native_command0_mp(&mut self) -> Result<()> {
        let units = presentation_units(&mut self.st);
        let j = self.native_cmd0_presentation.as_mut().expect("Go：j 仍是同一個工作");
        battle::apply_native_command_damage_mp(units, j.plan.as_mut())?;
        j.mp_published = true;
        Ok(())
    }

    /// Go：`native_command0_presentation.go:410`，`func (g *Game) failNativeCommand0Presentation(err error)`
    /// （`r8`，隨呼叫端 `stepNativeCommand0Presentation` 補移植）。
    pub(crate) fn fail_native_command0_presentation(&mut self, err: &GameError) {
        let Some(j) = self.native_cmd0_presentation.take() else {
            return;
        };
        let units = presentation_units(&mut self.st);
        if let Some(actor) = j.actor {
            (units[actor].mp, units[actor].acted) = (j.actor_mp_before, j.actor_acted_before);
        }
        if let Some(target) = j.target {
            units[target].hp = j.target_hp_before;
        }
        self.load_err = format!("native command0 presentation: {}", err.message());
    }
}

/// R4 D 組新增，沒有 Go 對應：演出工作與計畫裡的 Go `*battle.Unit` 都指向 `g.st.Units`，Rust 是
/// `UnitIdx`，`fd2_rules` 的發布函式另收單位切片。Go 的演出工作只在戰場上建立，`g.st` 不會是 nil。
pub(crate) fn presentation_units(st: &mut Option<battle::State>) -> &mut [battle::Unit] {
    &mut st.as_mut().expect("Go：演出工作的單位指標指向 g.st.Units").units
}
