//! Go 來源：`remake/cmd/fd2/native_action_death_present.go`（241 行）。
//!
//! 骨架：Go 檔的 11 個函式中產生 11 個（階段 0 清單），其餘標 `r8`，見 `PORTING.json`。
//!
//! R4 D 組：Go 的 `map[*battle.Unit]byte` 是 `BTreeMap<UnitIdx, u8>`，`map[int]byte` 是 `BTreeMap<i64, u8>`；
//! Go `range` map 的順序隨機，Rust 依鍵升冪（只影響多筆同時不合法時回報哪一筆）。

use super::*;

/// Settlement already writes HP and may mirror +5=1. Preserve the actual entry
/// flag for private presentation snapshots; never reconstruct it from a corpse.
///
/// Go：`native_action_death_present.go:11`，`func captureNativeActionDeathFlags(units ...*battle.Unit) map[*battle.Unit]byte`
///
/// R4 D 組：Go 的單位指標在 Rust 是 `st.units` 的索引，另收 `st` 讀 HP 與 +5（簽名變更紀錄）。
pub(crate) fn capture_native_action_death_flags(
    st: &battle::State,
    units: &[Option<UnitIdx>],
) -> BTreeMap<UnitIdx, u8> {
    let mut before = BTreeMap::new();
    for &unit in units {
        let Some(index) = unit else {
            continue;
        };
        let u = &st.units[index];
        if u.hp > 0 && u.has_native_record_byte5 && u.native_record_byte5 & 1 == 0 {
            before.insert(index, u.native_record_byte5);
        }
    }
    before
}

impl Game {
    /// Go：`native_action_death_present.go:21`，`func (g *Game) queueNativeActionDeathPresentation(before map[*battle.Unit]byte)`
    /// Go 檢查接收者是否為 nil；Rust 的 `self` 不會是 null，該分支不移植。
    pub(crate) fn queue_native_action_death_presentation(&mut self, before: &BTreeMap<UnitIdx, u8>) {
        let Some(st) = self.st.as_ref().filter(|st| st.has_native_map_view_state) else {
            return;
        };
        for (slot, unit) in st.units.iter().enumerate() {
            let Some(&flag) = before.get(&slot) else {
                continue;
            };
            if unit.hp != 0 {
                continue;
            }
            self.native_action_death_before.insert(slot as i64, flag);
        }
    }
}

/// Go：`native_action_death_present.go:37`，`func nativeDeathFlagPrelude(flags map[int]byte) func(*battle.State) error`
///
/// R4 D 組：回傳 B 組 `start_native_unit_death_present_mode` 收的 `FnMut(&mut battle::State)`（簽名變更紀錄）。
pub(crate) fn native_death_flag_prelude(
    flags: &BTreeMap<i64, u8>,
) -> impl FnMut(&mut battle::State) -> Result<()> + use<> {
    let flags = flags.clone();
    move |state: &mut battle::State| {
        for (&slot, &flag) in &flags {
            if slot < 0
                || slot >= state.units.len() as i64
                || state.units[slot as usize].hp != 0
                || !state.units[slot as usize].has_native_record_byte5
                || flag & 1 != 0
            {
                return Err(game_err!("native death presentation: slot{} has no new active death", slot));
            }
            state.units[slot as usize].native_record_byte5 = flag;
            if state.has_native_runtime_unit_projection {
                if state.native_runtime_records.len() != state.units.len() {
                    return Err(game_err!("native death presentation: raw roster differs"));
                }
                state.native_runtime_records[slot as usize].raw[5] = flag;
            }
        }
        Ok(())
    }
}

impl Game {
    /// Go：`native_action_death_present.go:56`，`func (g *Game) nativeMapDeathPlan(flags map[int]byte) (bool, error)`
    pub(crate) fn native_map_death_plan(&mut self, flags: &BTreeMap<i64, u8>) -> Result<bool> {
        let mut candidate = clone_native_ch28_post_state(self.st.as_mut().expect("Go：g.st 為 nil 時解參考 panic"))?
            .expect("cloneNativeCh28PostState 成功時不回 nil");
        native_death_flag_prelude(flags)(&mut candidate)?;
        let view = candidate.native_map_view_state;
        let plan = battle::plan_native_ch28_post_presentation(Some(&candidate), view.camera_x, view.camera_y);
        // Go `return len(plan.Targets) != 0, err`：錯誤時 plan 是零值，回 false 與錯誤。
        match plan {
            Ok(plan) => Ok(!plan.targets.is_empty()),
            Err(err) => Err(err.into()),
        }
    }

    /// Native and HD frames share the existing typed producer. The retained live
    /// state blocks through the same owner, without replacing actor/target pointers.
    ///
    /// Go：`native_action_death_present.go:71`，`func (g *Game) startNativeMapDeathVisual(flags map[int]byte, kind string, then func()) (bool, error)`
    /// Go 的 func 參數 `then` 會被存起來（工作結構的延續），Rust 收擁有的 `Box`。
    pub(crate) fn start_native_map_death_visual(
        &mut self,
        flags: &BTreeMap<i64, u8>,
        kind: &str,
        then: Option<Box<dyn FnOnce(&mut Game) + Send>>,
    ) -> Result<bool> {
        let visible = self.native_map_death_plan(flags)?;
        if !visible {
            return Ok(false);
        }
        self.start_native_unit_death_present_mode(&mut native_death_flag_prelude(flags), then, true)?;
        self.native_ch28_post_present.as_mut().expect("Go：startNativeUnitDeathPresentMode 成功後設好工作").kind =
            kind.to_string();
        Ok(true)
    }
}

/// Go：`native_action_death_present.go:83`，`func nativeTransientDeathFlags(before, after *battle.State) map[int]byte`
pub(crate) fn native_transient_death_flags(
    before: Option<&battle::State>,
    after: Option<&battle::State>,
) -> BTreeMap<i64, u8> {
    let mut flags = BTreeMap::new();
    let (Some(before), Some(after)) = (before, after) else {
        return flags;
    };
    if before.units.len() != after.units.len() {
        return flags;
    }
    for (slot, unit) in before.units.iter().enumerate() {
        if unit.has_native_record_byte5 && unit.native_record_byte5 & 1 == 0 && after.units[slot].hp == 0 {
            flags.insert(slot as i64, unit.native_record_byte5);
        }
    }
    flags
}

/// R4 D 組新增，沒有 Go 對應：Go `nativeTransientFocusPreview` 以 `preview := *g` 淺複製整個 `Game`，在副本上
/// 聚焦、重繪並試建 0x1DB65 演出，丟掉副本。Rust 的 `Game` 有不能複製的延續與工作欄位，所以改在 `self`
/// 上就地執行，事先保存副本路徑會改寫的欄位，結束後放回（[`Game::restore_native_transient_preview`]）。
///
/// 保存的欄位是預覽路徑（`redrawNativeMapHUD`、`nativeCursorStepHUD`、`syncNativeMapView`、
/// `composeNativeMapFrame`、`startNativeUnitDeathPresentMode` 與它發布的第一幀）在 Rust 實際寫入的全部欄位；
/// 單位視窗素材的延遲載入快取不還原（Go 副本載入的快取也留在副本，差別只是快取）。
#[derive(Default)]
pub(crate) struct NativeTransientPreviewSaved {
    pub(crate) st: Option<battle::State>,
    pub(crate) native_map_work: Vec<u8>,
    pub(crate) native_map_vga: Vec<u8>,
    pub(crate) native_ch28_post_present: Option<NativeCh28PostPresentJob>,
    pub(crate) load_err: String,
    pub(crate) cur: (i64, i64),
    pub(crate) cam: (f64, f64),
    pub(crate) native_ch23_state: Option<NativeCh23AdapterState>,
    pub(crate) native_interface_hud: Option<ModernHDMapHUDPanel>,
    pub(crate) native_map_clock: NativeBIOSClock,
    pub(crate) native_map_dac: Vec<u8>,
    pub(crate) native_fdother_palette: (i64, i64),
}

impl Game {
    /// 見 [`NativeTransientPreviewSaved`]：保存預覽會改寫的欄位，並把 `state` 換成 `self.st`（Go 的
    /// `preview.st = state`）。Go 的副本保留 `g.nativeCh28PostPresent` 的值，Rust 取出保存、預覽期間是
    /// `None`；兩個呼叫端（暫時狀態死亡預檢、物理攻擊返回預檢）進來時它都是 `None`。
    pub(crate) fn save_native_game_preview(&mut self, state: battle::State) -> NativeTransientPreviewSaved {
        NativeTransientPreviewSaved {
            st: self.st.replace(state),
            native_map_work: self.native_map_work.clone(),
            native_map_vga: self.native_map_vga.clone(),
            native_ch28_post_present: self.native_ch28_post_present.take(),
            load_err: self.load_err.clone(),
            cur: (self.cur_x, self.cur_y),
            cam: (self.cam_x, self.cam_y),
            native_ch23_state: self.native_ch23_state.clone(),
            native_interface_hud: self.native_interface_hud,
            native_map_clock: self.native_map_clock,
            native_map_dac: self.native_map_dac.clone(),
            native_fdother_palette: (self.native_fdother_palette_phase, self.native_fdother_palette_tick),
        }
    }

    /// 見 [`NativeTransientPreviewSaved`]：把預覽前的欄位放回 `self`。
    pub(crate) fn restore_native_transient_preview(&mut self, saved: NativeTransientPreviewSaved) {
        self.st = saved.st;
        self.native_map_work = saved.native_map_work;
        self.native_map_vga = saved.native_map_vga;
        self.native_ch28_post_present = saved.native_ch28_post_present;
        self.load_err = saved.load_err;
        (self.cur_x, self.cur_y) = saved.cur;
        (self.cam_x, self.cam_y) = saved.cam;
        self.native_ch23_state = saved.native_ch23_state;
        self.native_interface_hud = saved.native_interface_hud;
        self.native_map_clock = saved.native_map_clock;
        self.native_map_dac = saved.native_map_dac;
        (self.native_fdother_palette_phase, self.native_fdother_palette_tick) = saved.native_fdother_palette;
    }

    /// Preflight before the status transaction is published. Final focus can change
    /// the viewport after damage messages, so activation rebuilds from that view.
    ///
    /// Go：`native_action_death_present.go:99`，`func (g *Game) preflightNativeTransientDeath(candidate *battle.State, flags map[int]byte, damage []battle.NativeTransientDamage) error`
    ///
    /// R4 D 組：預覽在 `self` 上就地執行再還原（見 [`NativeTransientPreviewSaved`]），所以收 `&mut self`。
    pub(crate) fn preflight_native_transient_death(
        &mut self,
        candidate: &battle::State,
        flags: &BTreeMap<i64, u8>,
        damage: &[battle::NativeTransientDamage],
    ) -> Result<()> {
        if flags.is_empty() || !candidate.has_native_map_view_state {
            return Ok(());
        }
        let saved = self.native_transient_death_preview(candidate, damage)?;
        let result = self.start_native_unit_death_present_mode(&mut native_death_flag_prelude(flags), None, true);
        self.restore_native_transient_preview(saved);
        result
    }

    /// Go：`native_action_death_present.go:110`，`func (g *Game) nativeTransientDeathPreview(candidate *battle.State, damage []battle.NativeTransientDamage) (*Game, error)`
    ///
    /// R4 D 組：回傳預覽前保存的欄位，`self` 就是預覽（見 [`NativeTransientPreviewSaved`]）；呼叫端用完要
    /// [`Game::restore_native_transient_preview`]。錯誤時已經還原。
    pub(crate) fn native_transient_death_preview(
        &mut self,
        candidate: &battle::State,
        damage: &[battle::NativeTransientDamage],
    ) -> Result<NativeTransientPreviewSaved> {
        let slots: Vec<i64> = damage.iter().map(|event| event.slot).collect();
        self.native_transient_focus_preview(candidate, &slots)
    }

    /// Private focus uses the same cursor primitives without emitting replay input.
    ///
    /// Go：`native_action_death_present.go:119`，`func (g *Game) nativeTransientFocusPreview(candidate *battle.State, slots []int) (*Game, error)`
    ///
    /// R4 D 組：同 [`Game::native_transient_death_preview`]。
    pub(crate) fn native_transient_focus_preview(
        &mut self,
        candidate: &battle::State,
        slots: &[i64],
    ) -> Result<NativeTransientPreviewSaved> {
        let mut candidate = candidate.clone();
        let state = clone_native_ch28_post_state(&mut candidate)?.expect("cloneNativeCh28PostState 成功時不回 nil");
        let saved = self.save_native_game_preview(state);
        if let Err(err) = self.native_transient_focus_preview_slots(slots) {
            self.restore_native_transient_preview(saved);
            return Err(err);
        }
        Ok(saved)
    }

    /// `nativeTransientFocusPreview` 的逐筆聚焦迴圈（`native_action_death_present.go:128`..`175`），在預覽的 `self` 上執行。
    fn native_transient_focus_preview_slots(&mut self, slots: &[i64]) -> Result<()> {
        for &slot in slots {
            let state = self.st.as_ref().expect("預覽狀態已換入");
            if slot < 0 || slot >= state.units.len() as i64 {
                return Err(game_err!("native transient death preview: invalid slot{}", slot));
            }
            let (ux, uy) = (state.units[slot as usize].x, state.units[slot as usize].y);
            if ux < 0 || ux >= state.w || uy < 0 || uy >= state.h {
                return Err(game_err!("native transient death preview: slot{} outside field", slot));
            }
            let (view_before, hud_before) = (state.native_map_view_state, state.native_map_hud_state);
            self.st.as_mut().unwrap().materialize_native_map_range_mode(0);
            self.redraw_native_map_hud();
            // Same X-then-Y MoveNativeMapCursor primitives as FocusNativeMapCursorSteps,
            // without emitting its external trace hook for this private preflight.
            for (axis, destination) in [ux, uy].into_iter().enumerate() {
                loop {
                    let view = self.st.as_ref().unwrap().native_map_view_state;
                    let position = if axis == 1 { view.cursor_y } else { view.cursor_x };
                    if position == destination {
                        break;
                    }
                    let step = if position > destination { -1 } else { 1 };
                    let (dx, dy) = if axis == 1 { (0, step) } else { (step, 0) };
                    let st = self.st.as_mut().unwrap();
                    let before = st.native_map_view_state;
                    let (_, ok) = st.move_native_map_cursor(dx, dy);
                    if !ok {
                        return Err(game_err!("native transient death preview: rejected cursor step"));
                    }
                    let after = st.native_map_view_state;
                    self.native_cursor_step_hud(before, after);
                }
            }
            self.sync_native_map_view();
            let st = self.st.as_mut().unwrap();
            st.materialize_native_map_range_mode(1);
            if (st.native_map_view_state != view_before
                || st.native_map_hud_state != hud_before
                || self.native_map_vga.len() != 320 * 200)
                && let Err(err) = self.compose_native_map_frame()
            {
                return Err(err);
            }
            if !self.load_err.is_empty() {
                return Err(game_err!("native transient death preview: {}", self.load_err));
            }
        }
        Ok(())
    }

    /// Go：`native_action_death_present.go:179`，`func (g *Game) beginNativeActionDeathPresentation(actor *battle.Unit, after func()) bool`
    /// Go 檢查接收者是否為 nil；Rust 的 `self` 不會是 null，該分支不移植。
    ///
    /// R4 D 組：`after` 改收 `finishSuccessfulUnitAction` 的共用格子 [`SharedAfter`]（簽名變更紀錄）。Go 的續行
    /// 閉包捕捉同一個 `after`；這裡沒有開始演出時 `after` 留在格子裡，呼叫端照常往下用。
    pub(crate) fn begin_native_action_death_presentation(&mut self, actor: UnitIdx, after: &SharedAfter) -> bool {
        if !self.st.as_ref().is_some_and(|st| st.has_native_map_view_state) {
            self.native_action_death_before = BTreeMap::new();
            if let Some(st) = self.st.as_mut() {
                for unit in &mut st.units {
                    unit.clear_native_death_presentation_flag();
                }
            }
            return false;
        }
        let mut flags = self.native_action_death_before.clone();
        for (slot, unit) in self.st.as_ref().unwrap().units.iter().enumerate() {
            let slot = slot as i64;
            if let (flag, true) = unit.native_death_presentation_flag() {
                flags.entry(slot).or_insert(flag);
            }
            if unit.hp == 0 && unit.has_native_record_byte5 && unit.native_record_byte5 & 1 == 0 {
                flags.entry(slot).or_insert(unit.native_record_byte5);
            }
        }
        if flags.is_empty() {
            return false;
        }
        let after = Arc::clone(after);
        let then: Box<dyn FnOnce(&mut Game) + Send> = Box::new(move |g: &mut Game| {
            g.finish_successful_unit_action(Some(actor), take_shared_after(&after));
            g.check_result();
        });
        let started = match self.start_native_map_death_visual(&flags, "action", Some(then)) {
            Ok(started) => started,
            Err(err) => {
                self.load_err = format!("native action death presentation: {}", err.message());
                return true;
            }
        };
        self.native_action_death_before = BTreeMap::new();
        let st = self.st.as_mut().unwrap();
        for &slot in flags.keys() {
            st.units[slot as usize].clear_native_death_presentation_flag();
        }
        if started {
            st.mark_native_dead_records();
        }
        started
    }

    /// Go：`native_action_death_present.go:228`，`func (g *Game) nativeActionDeathPresentationPending() bool`
    /// Go 檢查接收者是否為 nil；Rust 的 `self` 不會是 null，該分支不移植。
    pub(crate) fn native_action_death_presentation_pending(&self) -> bool {
        let Some(st) = self.st.as_ref().filter(|st| st.has_native_map_view_state) else {
            return false;
        };
        if !self.native_action_death_before.is_empty() {
            return true;
        }
        st.units.iter().any(|unit| unit.native_death_presentation_flag().1)
    }
}
