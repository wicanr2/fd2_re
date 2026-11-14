//! Go 來源：`remake/cmd/fd2/native_command29_presentation.go`（585 行）。
//!
//! 骨架：Go 檔的 8 個函式中產生 1 個（階段 0 清單），其餘標 `r8`，見 `PORTING.json`。

use super::*;

/// Go：`native_command29_presentation.go:14`
#[derive(Debug, Clone, Default, PartialEq, Eq)]
pub(crate) struct NativeCommand29TargetPresentation {
    pub(crate) unit: Option<UnitIdx>,
    pub(crate) base_before: Option<Arc<Picture>>,
    pub(crate) base_after: Option<Arc<Picture>>,
    pub(crate) idle_frames: Vec<Option<Arc<Picture>>>,
    pub(crate) idle_positions: Vec<[i64; 2]>,
    pub(crate) idle_delays: Vec<i64>,
    pub(crate) transition_frames: Vec<Option<Arc<Picture>>>,
    pub(crate) hp_before: i64,
    pub(crate) raw_byte5_before: u8,
    pub(crate) damage_published: bool,
}

/// Go：`native_command29_presentation.go:26`
///
/// 不 derive `Debug`／`Clone`：欄位含 callback（`Box<dyn FnOnce>`）、檔案或亂數產生器等做不到的型別。
#[derive(Default)]
pub(crate) struct NativeCommand29PresentationJob {
    pub(crate) command28: Option<NativeCommand28Sequence>,
    pub(crate) hd: Option<ModernHDSplitStrike>,
    pub(crate) actor: Option<UnitIdx>,
    pub(crate) plan: Option<battle::NativeCommandDerivedStrikePlan>,
    pub(crate) schedule: figani::NativeCommandDerivedStrikeSchedule,
    pub(crate) effect_frames: Vec<Option<Arc<Picture>>>,
    pub(crate) effect_positions: Vec<[i64; 2]>,
    pub(crate) prelude_frames: Vec<Option<Arc<Picture>>>,
    pub(crate) prelude_frame: i64,
    pub(crate) actor_base_before: Option<Arc<Picture>>,
    pub(crate) actor_base_after: Option<Arc<Picture>>,
    pub(crate) targets: Vec<NativeCommand29TargetPresentation>,
    pub(crate) target_index: i64,
    pub(crate) transition_frame: i64,
    pub(crate) frame: i64,
    pub(crate) repeat: i64,
    pub(crate) idle_frame: i64,
    pub(crate) idle_repeat: i64,
    pub(crate) shake_counter: i64,
    pub(crate) drawn: bool,
    pub(crate) mp_published: bool,
    pub(crate) actor_mp_before: i64,
    pub(crate) actor_acted_before: bool,
    pub(crate) sfx_actor: Vec<u8>,
    pub(crate) sfx_target: Vec<u8>,
    /// R4 D 組：Go `map[int][]byte` 的 nil 與空 map 在 `stepNativeCommand29Presentation`（Go 檔:436）行為不同
    /// （nil 才播 `sfxActor`），所以包 `Option`；Go 的 start（Go 檔:369）一律 `make`，對應 `Some`。
    pub(crate) actor_cues: Option<BTreeMap<i64, Vec<u8>>>,
    pub(crate) then: Option<Box<dyn FnOnce(&mut Game, Vec<battle::NativeCommand24Damage>) + Send>>,
}

impl Game {
    /// Go：`native_command29_presentation.go:409`，`func (g *Game) stepNativeCommand29Presentation()`
    pub(crate) fn step_native_command29_presentation(&mut self) {
        let Some(j) = self.native_cmd29_presentation.as_mut() else {
            return;
        };
        if !j.drawn {
            return;
        }
        if j.command28.is_some() {
            todo!(
                "R8: native_command28_presentation.go:stepNativeCommand28Presentation（第 28 號指令的序列，第一章重播沒有執行到）"
            );
        }
        j.drawn = false;
        if j.prelude_frame < j.prelude_frames.len() as i64 {
            j.prelude_frame += 1;
            return;
        }
        if j.transition_frame >= 0 {
            j.transition_frame += 1;
            if j.transition_frame >= j.targets[j.target_index as usize].transition_frames.len() as i64 {
                (j.transition_frame, j.idle_frame, j.idle_repeat) = (-1, 0, 0);
            }
            return;
        }
        if j.frame == j.schedule.actor_impact_frame && !j.mp_published {
            let units = presentation_units(&mut self.st);
            let j = self.native_cmd29_presentation.as_mut().expect("Go：j 仍是同一個工作");
            if let Err(err) = battle::apply_native_command_derived_strike_mp(units, j.plan.as_ref()) {
                self.fail_native_command29_presentation(&err.into());
                return;
            }
            j.mp_published = true;
            if j.actor_cues.is_none() {
                let sfx = j.sfx_actor.clone();
                self.play_raw(&sfx);
            }
        }
        let j = self.native_cmd29_presentation.as_mut().expect("Go：j 仍是同一個工作");
        if j.frame < j.schedule.target_start
            && j.repeat == 0
            && let Some(cue) = j.actor_cues.as_ref().and_then(|cues| cues.get(&j.frame))
        {
            let cue = cue.clone();
            self.play_raw(&cue);
        }
        let j = self.native_cmd29_presentation.as_mut().expect("Go：j 仍是同一個工作");
        if j.frame >= j.schedule.target_start {
            let target_index = j.target_index;
            if j.targets[target_index as usize].damage_published && j.shake_counter >= 0 {
                j.shake_counter -= 1;
            }
            if j.frame == j.schedule.target_impact_frame && !j.targets[target_index as usize].damage_published {
                if !j.mp_published {
                    self.fail_native_command29_presentation(&game_err!("target marker preceded actor MP marker"));
                    return;
                }
                let units = presentation_units(&mut self.st);
                let j = self.native_cmd29_presentation.as_mut().expect("Go：j 仍是同一個工作");
                if let Err(err) =
                    battle::apply_native_command_derived_strike_target(units, j.plan.as_ref(), target_index)
                {
                    self.fail_native_command29_presentation(&err.into());
                    return;
                }
                j.targets[target_index as usize].damage_published = true;
                j.shake_counter = 5;
                let sfx = j.sfx_target.clone();
                self.play_raw(&sfx);
            }
            let j = self.native_cmd29_presentation.as_mut().expect("Go：j 仍是同一個工作");
            j.idle_repeat += 1;
            let target = &j.targets[target_index as usize];
            if j.idle_repeat >= target.idle_delays[j.idle_frame as usize] {
                let idle_len = target.idle_frames.len() as i64;
                j.idle_repeat = 0;
                // Go `%`：`len(target.idleFrames)` 為 0 時 panic，Rust 同樣 panic。
                j.idle_frame = (j.idle_frame + 1) % idle_len;
            }
        }
        let j = self.native_cmd29_presentation.as_mut().expect("Go：j 仍是同一個工作");
        j.repeat += 1;
        if j.repeat < j.schedule.frame_delays[j.frame as usize] {
            return;
        }
        j.repeat = 0;
        j.frame += 1;
        if j.frame == j.schedule.target_start {
            j.transition_frame = 0;
            return;
        }
        if j.frame < j.effect_frames.len() as i64 {
            return;
        }
        if !j.targets[j.target_index as usize].damage_published {
            self.fail_native_command29_presentation(&game_err!("required target marker was not presented"));
            return;
        }
        if j.target_index + 1 < j.targets.len() as i64 {
            j.target_index += 1;
            (j.frame, j.repeat, j.idle_frame, j.idle_repeat, j.transition_frame) =
                (j.schedule.target_start, 0, 0, 0, 0);
            return;
        }
        if !j.mp_published {
            self.fail_native_command29_presentation(&game_err!("required actor marker was not presented"));
            return;
        }
        let units = presentation_units(&mut self.st);
        let j = self.native_cmd29_presentation.as_mut().expect("Go：j 仍是同一個工作");
        if let Err(err) = battle::complete_native_command_derived_strike(units, j.plan.as_ref()) {
            self.fail_native_command29_presentation(&err.into());
            return;
        }
        let mut j = self.native_cmd29_presentation.take().expect("Go：j 仍是同一個工作");
        let (then, results) =
            (j.then.take(), j.plan.take().expect("Go：CompleteNativeCommandDerivedStrike 已檢查 plan").results);
        if let Some(then) = then {
            then(self, results);
        }
    }

    /// Go：`native_command29_presentation.go:387`，`func (g *Game) failNativeCommand29Presentation(err error)`
    /// （`r8`，隨呼叫端 `stepNativeCommand29Presentation` 補移植）。
    pub(crate) fn fail_native_command29_presentation(&mut self, err: &GameError) {
        let Some(j) = self.native_cmd29_presentation.take() else {
            return;
        };
        let units = presentation_units(&mut self.st);
        let actor = j.actor.expect("Go：j.actor 為 nil 時解參考 panic");
        (units[actor].mp, units[actor].acted) = (j.actor_mp_before, j.actor_acted_before);
        for target in &j.targets {
            let unit = &mut units[target.unit.expect("Go：target.unit 為 nil 時解參考 panic")];
            unit.hp = target.hp_before;
            if unit.has_native_record_byte5 {
                unit.native_record_byte5 = target.raw_byte5_before;
            }
        }
        battle::restore_native_command30_targets(units, j.plan.as_ref());
        let command = j.plan.as_ref().map_or(29, |plan| plan.command_id);
        self.load_err = format!("native command{command} presentation: {}", err.message());
    }
}
