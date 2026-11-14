//! Go 來源：`remake/cmd/fd2/native_command24_presentation.go`（537 行）。
//!
//! 骨架：Go 檔的 10 個函式中產生 3 個（階段 0 清單），其餘標 `r8`，見 `PORTING.json`。

use super::*;

/// Go：`native_command24_presentation.go:16`
///
/// 不 derive `Debug`／`Clone`：欄位含 callback（`Box<dyn FnOnce>`）、檔案或亂數產生器等做不到的型別。
#[derive(Default)]
pub(crate) struct NativeCommand24PresentationJob {
    pub(crate) hd: Option<ModernHDSplitStrike>,
    pub(crate) actor: Option<UnitIdx>,
    pub(crate) target: Option<UnitIdx>,
    pub(crate) plan: Option<battle::NativeCommandDerivedStrikePlan>,
    pub(crate) schedule: figani::NativeCommand24Schedule,
    pub(crate) effect_frames: Vec<Option<Arc<Picture>>>,
    pub(crate) effect_positions: Vec<[i64; 2]>,
    pub(crate) target_frames: Vec<Option<Arc<Picture>>>,
    pub(crate) target_positions: Vec<[i64; 2]>,
    pub(crate) target_delays: Vec<i64>,
    pub(crate) prelude_frames: Vec<Option<Arc<Picture>>>,
    pub(crate) prelude_frame: i64,
    pub(crate) actor_base_before: Option<Arc<Picture>>,
    pub(crate) actor_base_after: Option<Arc<Picture>>,
    pub(crate) target_base_before: Option<Arc<Picture>>,
    pub(crate) target_base_after: Option<Arc<Picture>>,
    pub(crate) transition_frames: Vec<Option<Arc<Picture>>>,
    pub(crate) transition_frame: i64,
    pub(crate) frame: i64,
    pub(crate) repeat: i64,
    pub(crate) target_frame: i64,
    pub(crate) target_repeat: i64,
    pub(crate) shake_counter: i64,
    pub(crate) drawn: bool,
    pub(crate) mp_published: bool,
    pub(crate) damage_published: bool,
    pub(crate) actor_mp_before: i64,
    pub(crate) actor_acted_before: bool,
    pub(crate) target_hp_before: i64,
    pub(crate) target_raw_byte5_before: u8,
    pub(crate) then: Option<Box<dyn FnOnce(&mut Game, Vec<battle::NativeCommand24Damage>) + Send>>,
}

/// Go：`native_command24_presentation.go:61`，`func nativeCommand24RuntimeUnitIndex(st *battle.State, unit *battle.Unit) (int, error)`
pub(crate) fn native_command24_runtime_unit_index(st: Option<&battle::State>, unit: Option<UnitIdx>) -> Result<i64> {
    let (Some(st), Some(unit)) = (st, unit) else {
        return Err(game_err!("native command24 runtime unit unavailable"));
    };
    // Go 以指標相等在 `st.Units` 裡找；Rust 的單位就是 `st.units` 的索引，只要在範圍內。
    if unit < st.units.len() {
        return Ok(unit as i64);
    }
    Err(game_err!("native command24 unit is not in the runtime array"))
}

/// Go：`native_command24_presentation.go:89`，`func nativeCommand24PreludeImages(frames []battlepresent.NativeCommand24PreludeFrame) ([]*ebiten.Image, error)`
///
/// R4 D 組：回傳改成 `Vec<IndexedFrame>`（骨架是 `Vec<Option<Arc<Picture>>>`），畫格保存索引色像素與 DAC 轉出的調色盤。
pub(crate) fn native_command24_prelude_images(
    frames: &[battlepresent::NativeCommand24PreludeFrame],
) -> Result<Vec<IndexedFrame>> {
    if frames.len() != 9 {
        return Err(game_err!("native command24 prelude must contain nine indexed frames"));
    }
    let mut out = Vec::with_capacity(frames.len());
    for (i, frame) in frames.iter().enumerate() {
        if frame.stage != 8 - i as i64 || frame.pixels.len() != 320 * 200 {
            return Err(game_err!("native command24 prelude frame {i} is malformed"));
        }
        let palette = fdother::vga_palette_from_dac(&frame.dac)?;
        // Go `image.NewPaletted`＋`ebiten.NewImageFromImage`：Rust 保存索引色畫格（120「狀態只保存索引色畫面與調色盤」），
        // GPU 貼圖由 `fd2-app` 建立（R5）。
        out.push(IndexedFrame {
            width: 320,
            height: 200,
            pix: frame.pixels.clone(),
            palette: color_palette_from_nrgba(&palette),
        });
    }
    Ok(out)
}

impl Game {
    /// Go：`native_command24_presentation.go:378`，`func (g *Game) stepNativeCommand24Presentation()`
    pub(crate) fn step_native_command24_presentation(&mut self) {
        let Some(j) = self.native_cmd24_presentation.as_mut() else {
            return;
        };
        if !j.drawn {
            return;
        }
        j.drawn = false;
        if j.prelude_frame < j.prelude_frames.len() as i64 {
            j.prelude_frame += 1;
            return;
        }
        if j.transition_frame >= 0 {
            j.transition_frame += 1;
            if j.transition_frame >= j.transition_frames.len() as i64 {
                j.transition_frame = -1;
                (j.target_frame, j.target_repeat) = (0, 0);
            }
            return;
        }
        let frame = j.frame;
        if j.damage_published && j.shake_counter >= 0 {
            j.shake_counter -= 1;
        }
        if frame == j.schedule.actor_impact_frame && !j.mp_published {
            let units = presentation_units(&mut self.st);
            let j = self.native_cmd24_presentation.as_mut().expect("Go：j 仍是同一個工作");
            if let Err(err) = battle::apply_native_command_derived_strike_mp(units, j.plan.as_ref()) {
                self.fail_native_command24_presentation(&err.into());
                return;
            }
            j.mp_published = true;
            let sfx = self.sfx_command24_actor.clone();
            self.play_raw(&sfx);
        }
        let j = self.native_cmd24_presentation.as_mut().expect("Go：j 仍是同一個工作");
        if frame == j.schedule.target_impact_frame && !j.damage_published {
            if !j.mp_published {
                self.fail_native_command24_presentation(&game_err!("target marker preceded actor MP marker"));
                return;
            }
            let units = presentation_units(&mut self.st);
            let j = self.native_cmd24_presentation.as_mut().expect("Go：j 仍是同一個工作");
            if let Err(err) = battle::apply_native_command_derived_strike_target(units, j.plan.as_ref(), 0) {
                self.fail_native_command24_presentation(&err.into());
                return;
            }
            j.damage_published = true;
            j.shake_counter = 5;
            let sfx = self.sfx_command24_target.clone();
            self.play_raw(&sfx);
        }
        let j = self.native_cmd24_presentation.as_mut().expect("Go：j 仍是同一個工作");
        j.target_repeat += 1;
        if j.target_repeat >= j.target_delays[j.target_frame as usize] {
            j.target_repeat = 0;
            // Go `%`：`len(j.targetFrames)` 為 0 時 panic，Rust 的 `%` 同樣 panic。
            j.target_frame = (j.target_frame + 1) % j.target_frames.len() as i64;
        }
        j.repeat += 1;
        if j.repeat < native_command24_raw_frames_delay(Some(&*j)) {
            return;
        }
        j.repeat = 0;
        j.frame += 1;
        if j.frame == j.schedule.target_start {
            j.transition_frame = 0;
        }
        if j.frame < j.effect_frames.len() as i64 {
            return;
        }
        if !j.mp_published || !j.damage_published {
            self.fail_native_command24_presentation(&game_err!("required raw marker was not presented"));
            return;
        }
        let units = presentation_units(&mut self.st);
        let j = self.native_cmd24_presentation.as_mut().expect("Go：j 仍是同一個工作");
        if let Err(err) = battle::complete_native_command_derived_strike(units, j.plan.as_ref()) {
            self.fail_native_command24_presentation(&err.into());
            return;
        }
        let mut j = self.native_cmd24_presentation.take().expect("Go：j 仍是同一個工作");
        let (then, results) =
            (j.then.take(), j.plan.take().expect("Go：CompleteNativeCommandDerivedStrike 已檢查 plan").results);
        if let Some(then) = then {
            then(self, results);
        }
    }

    /// Go：`native_command24_presentation.go:365`，`func (g *Game) failNativeCommand24Presentation(err error)`
    /// （`r8`，隨呼叫端 `stepNativeCommand24Presentation` 補移植）。
    pub(crate) fn fail_native_command24_presentation(&mut self, err: &GameError) {
        let Some(j) = self.native_cmd24_presentation.take() else {
            return;
        };
        let units = presentation_units(&mut self.st);
        let actor = j.actor.expect("Go：j.actor 為 nil 時解參考 panic");
        let target = j.target.expect("Go：j.target 為 nil 時解參考 panic");
        (units[actor].mp, units[actor].acted, units[target].hp) =
            (j.actor_mp_before, j.actor_acted_before, j.target_hp_before);
        if units[target].has_native_record_byte5 {
            units[target].native_record_byte5 = j.target_raw_byte5_before;
        }
        self.load_err = format!("native command24 presentation: {}", err.message());
    }
}

/// Go：`native_command24_presentation.go:453`，`func nativeCommand24RawFramesDelay(j *nativeCommand24PresentationJob) int`
/// （`r8`，隨呼叫端 `stepNativeCommand24Presentation` 補移植）。
pub(crate) fn native_command24_raw_frames_delay(j: Option<&NativeCommand24PresentationJob>) -> i64 {
    let Some(j) = j.filter(|j| j.frame >= 0 && j.frame < NATIVE_COMMAND24_RAW_DELAYS.len() as i64) else {
        return 0;
    };
    // BuildNativeCommand24Schedule already fixed the complete resource98 raw
    // signature; the explicit table keeps Draw timing independent of PNG count.
    NATIVE_COMMAND24_RAW_DELAYS[j.frame as usize] as i64
}

/// Go：`native_command24_presentation.go:462`，`var nativeCommand24RawDelays = [15]byte{…}`（`r8` 函式用到，隨呼叫端補上）。
pub(crate) const NATIVE_COMMAND24_RAW_DELAYS: [u8; 15] = [2, 2, 2, 8, 2, 2, 2, 2, 8, 2, 2, 6, 2, 2, 4];
