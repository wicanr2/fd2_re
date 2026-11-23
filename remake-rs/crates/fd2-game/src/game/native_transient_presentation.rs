//! Go 來源：`remake/cmd/fd2/native_transient_presentation.go`（214 行）。
//!
//! 骨架：Go 檔的 4 個函式中產生 2 個（階段 0 清單），其餘標 `r8`，見 `PORTING.json`。

use super::*;

pub(crate) const NATIVE_TRANSIENT_DAMAGE_TEXT_INDEX: i64 = 0x1e7;

pub(crate) const NATIVE_TRANSIENT_DAMAGE_CLOSE_FRAMES: i64 = 33;

impl Game {
    /// beginNativeTransientPhases 依 sub_1A866 的順序演出一次掃描：先逐筆 +0x25 扣血回覆
    /// （0x1A8D4 [0x51A83]=0 → 0x12D7B 聚焦該記錄 → [0x51A83]=1 → 0x1956B 開框 → 0x1E7 →
    /// 0x1E5C0(10) → 0x196CB 關框），再逐筆到期提示（0x1A9AA 同一套聚焦後開框，FDTXT
    /// 0x1E1..0x1E6）。所有到期幀的素材與肖像在發布交易前先預檢；缺素材時 HP／狀態交易與
    /// 玩家可見回饋都不變。扣血訊息後、到期提示前同步呈現 sub_1DB65；不分派死亡效果。
    ///
    /// Go：`native_transient_presentation.go:24`，`func (g *Game) beginNativeTransientPhases(selectors []byte, then func()) error`
    /// Go 檢查接收者是否為 nil；Rust 的 `self` 不會是 null，該分支不移植。
    /// Go 的 func 參數 `then` 會被存起來（工作結構的延續），Rust 收擁有的 `Box`。
    ///
    /// R4 D 組：Go 的兩個遞迴閉包 `runDamage`／`runExpiry` 改成
    /// [`Game::run_native_transient_damage`]、[`Game::run_native_transient_expiry`]；`resolve` 是恆等（見
    /// `adopt_native_state_candidate`）。
    pub(crate) fn begin_native_transient_phases(
        &mut self,
        selectors: &mut [u8],
        then: Option<Box<dyn FnOnce(&mut Game) + Send>>,
    ) -> Result<()> {
        if self.native_class_ui_job.is_some()
            || self.transient_ui
            || self.native_level_up_dialogue.is_some()
            || self.native_ch28_post_present.is_some()
        {
            return Err(game_err!("native transient presentation: another indexed owner is active"));
        }
        let (candidate, expired, damage) = self.build_native_transient_phases(selectors)?;
        let death_flags = native_transient_death_flags(self.st.as_ref(), Some(&candidate));
        self.preflight_native_transient_death(&candidate, &death_flags, &damage)?;
        if expired.is_empty() && damage.is_empty() && death_flags.is_empty() {
            self.adopt_native_state_candidate(Some(candidate));
            if let Some(then) = then {
                then(self);
            }
            return Ok(());
        }
        if !expired.is_empty() && (self.native_class_ui.is_none() || self.native_map_vga.len() != 320 * 200) {
            return Err(game_err!("native transient presentation: indexed source assets are unavailable"));
        }
        let mut expiries = Vec::with_capacity(expired.len());
        let source = self.native_map_vga.clone();
        for event in &expired {
            let (unit, counter) = native_transient_presentation_event(&candidate, event)?;
            let portrait = load_native_separated_portrait(candidate.units[unit].battle_fig)?
                .into_iter()
                .next()
                .ok_or_else(|| game_err!("到期提示缺少原版頭像"))?;
            self.compose_native_transient_expiry(&source, &portrait, counter)?;
            expiries.push(NativeTransientExpiryFrame { unit, counter, portrait });
        }
        let mut damage_pages = Vec::with_capacity(damage.len());
        let mut damage_waits = Vec::with_capacity(damage.len());
        for event in &damage {
            let presentable = event.unit.is_some_and(|u| {
                let unit = &candidate.units[u];
                unit.has_battle_fig && unit.has_native_map_presentation
            });
            if !presentable {
                return Err(game_err!(
                    "native transient presentation: damaged record has no DATO selector or map presentation"
                ));
            }
            let Some(strings) = self.native_preparation_ui.as_ref().and_then(|ui| ui.status.strings.as_ref()) else {
                return Err(game_err!("native transient presentation: FDTXT_000 strings are unavailable"));
            };
            let (pages, wait) = native_message_pages(strings, NATIVE_TRANSIENT_DAMAGE_TEXT_INDEX, event.amount)?;
            damage_pages.push(pages);
            damage_waits.push(wait);
        }
        self.adopt_native_state_candidate(Some(candidate));
        self.msg = String::new();
        let run = Arc::new(NativeTransientRun { damage, damage_pages, damage_waits, death_flags, expiries });
        if run.damage.is_empty() {
            // 到期演出是同步的：呼叫端看 transientUI 等它結束。
            self.transient_ui = true;
        }
        self.run_native_transient_damage(run, 0, then);
        Ok(())
    }

    /// Go `beginNativeTransientPhases` 的閉包 `runDamage`（`native_transient_presentation.go:144`..`170`）。
    fn run_native_transient_damage(
        &mut self,
        run: Arc<NativeTransientRun>,
        index: usize,
        then: Option<Box<dyn FnOnce(&mut Game) + Send>>,
    ) {
        if index >= run.damage.len() {
            if !run.death_flags.is_empty() && self.st.as_ref().is_some_and(|st| st.has_native_map_view_state) {
                self.transient_ui = false; // the death owner now blocks Update and phase continuation
                let expiry_run = Arc::clone(&run);
                let next: Box<dyn FnOnce(&mut Game) + Send> =
                    Box::new(move |g: &mut Game| g.run_native_transient_expiry(expiry_run, 0, then));
                // Go 的閉包捕捉 `then`；Rust 交給死亡演出的延續，沒開始時由這裡取回（共用格子）。
                let slot: SharedAfter = Arc::new(std::sync::Mutex::new(Some(next)));
                let deferred = Arc::clone(&slot);
                let started = self.start_native_map_death_visual(
                    &run.death_flags,
                    "status",
                    Some(Box::new(move |g: &mut Game| {
                        if let Some(next) = take_shared_after(&deferred) {
                            next(g);
                        }
                    })),
                );
                match started {
                    Err(err) => {
                        self.load_err = format!("native transient death presentation: {}", err.message());
                        return;
                    }
                    Ok(true) => return,
                    Ok(false) => {}
                }
                if let Some(next) = take_shared_after(&slot) {
                    next(self);
                }
                return;
            }
            self.run_native_transient_expiry(run, 0, then);
            return;
        }
        let unit = run.damage[index].unit;
        if let Err(err) = self.focus_native_transient_record(unit) {
            self.load_err = format!("native transient presentation: {}", err.message());
            return;
        }
        let (pages, wait) = (run.damage_pages[index].clone(), run.damage_waits[index]);
        let next_run = Arc::clone(&run);
        let next: Box<dyn FnOnce(&mut Game) + Send> =
            Box::new(move |g: &mut Game| g.run_native_transient_damage(next_run, index + 1, then));
        if let Err(err) = self.begin_native_timed_message_dialogue(
            unit,
            NATIVE_TRANSIENT_DAMAGE_TEXT_INDEX,
            &pages,
            wait,
            NATIVE_TRANSIENT_DAMAGE_CLOSE_FRAMES,
            Some(next),
        ) {
            self.load_err = format!("native transient presentation: {}", err.message());
        }
    }

    /// Go `beginNativeTransientPhases` 的逐筆到期提示。
    fn run_native_transient_expiry(
        &mut self,
        run: Arc<NativeTransientRun>,
        index: usize,
        then: Option<Box<dyn FnOnce(&mut Game) + Send>>,
    ) {
        let Some(event) = run.expiries.get(index) else {
            self.transient_ui = false;
            if let Some(then) = then {
                then(self);
            }
            return;
        };
        let job = (|| -> Result<NativeClassUIJob> {
            self.focus_native_transient_record(Some(event.unit))?;
            let source = self.native_map_vga.clone();
            let frame = self.compose_native_transient_expiry(&source, &event.portrait, event.counter)?;
            let mut frames = campaign::native_class_list_opening_frames(&source, &frame)?;
            frames.extend(campaign::native_class_list_closing_frames(&source, &frame)?);
            Ok(NativeClassUIJob { frames, restore: source, ..Default::default() })
        })();
        match job {
            Ok(mut job) => {
                job.after = Some(Box::new(move |g| g.run_native_transient_expiry(run, index + 1, then)));
                self.transient_ui = true;
                self.native_class_ui_job = Some(job);
            }
            Err(err) => {
                self.transient_ui = false;
                self.load_err = format!("狀態到期提示：{err}");
            }
        }
    }

    fn compose_native_transient_expiry(
        &mut self,
        source: &[u8],
        portrait: &dato::Frame,
        counter: i64,
    ) -> Result<Vec<u8>> {
        let ui = self.native_class_ui.as_ref().ok_or_else(|| game_err!("到期提示缺少對話框素材"))?;
        if self.locale_id.is_empty() || self.locale_id == "zh-Hant" {
            return Ok(campaign::compose_native_transient_expiry_frame(
                source,
                &ui.dialogue,
                portrait,
                ui.strings.as_ref(),
                ui.font.as_ref(),
                counter,
            )?);
        }
        let keys = [
            "battle.status.expiry.attack",
            "battle.status.expiry.defence",
            "battle.status.expiry.speed",
            "battle.status.expiry.poison",
            "battle.status.expiry.paralysis",
            "battle.status.expiry.silence",
        ];
        let key = keys.get(counter as usize).ok_or_else(|| game_err!("到期提示的原生計數器無效"))?;
        let mut frame = campaign::compose_native_transient_expiry_dialogue(source, &ui.dialogue, portrait)?;
        let (message, ok) = self.locale_message(key, &[]);
        let font = self.font.as_mut().filter(|_| ok).ok_or_else(|| game_err!("到期提示缺少翻譯或字型"))?;
        let lines = font.wrap(&message, 0.78, 205.0);
        if lines.is_empty() || lines.len() > 3 {
            return Err(game_err!("到期提示的翻譯超過文字區"));
        }
        for (row, line) in lines.iter().enumerate() {
            draw_indexed_localized_text(
                &mut frame,
                Some(font),
                line,
                99,
                127 + row as i64 * 19,
                205,
                19,
                0.78,
                0xcd,
                0x4c,
            )?;
        }
        Ok(frame)
    }

    /// nativeTransientPresentationActive 回報 sub_1A866 的扣血訊息或到期提示還沒播完。
    ///
    /// Go：`native_transient_presentation.go:181`，`func (g *Game) nativeTransientPresentationActive() bool`
    /// Go 檢查接收者是否為 nil；Rust 的 `self` 不會是 null，該分支不移植。
    pub(crate) fn native_transient_presentation_active(&self) -> bool {
        self.transient_ui || self.native_level_up_dialogue.is_some() || self.native_ch28_post_present.is_some()
    }

    /// focusNativeTransientRecord 是 0x1A8D4／0x1A9AA 的聚焦：[0x51A83]=0 → 0x12D7B(record) →
    /// [0x51A83]=1。聚焦改到視圖（游標、鏡頭或 HUD anchor）時整幀重繪，讓之後的框貼在新畫面上。
    ///
    /// Go：`native_transient_presentation.go:187`，`func (g *Game) focusNativeTransientRecord(unit *battle.Unit) error`
    /// （`r8`，隨呼叫端 `beginNativeTransientPhases` 補移植）。
    pub(crate) fn focus_native_transient_record(&mut self, unit: Option<UnitIdx>) -> Result<()> {
        let (Some(st), Some(unit)) = (self.st.as_mut(), unit) else {
            return Err(game_err!("focus record is unavailable"));
        };
        if !st.has_native_map_view_state {
            return Ok(());
        }
        let (view_before, hud_before) = (st.native_map_view_state, st.native_map_hud_state);
        let (x, y) = (st.units[unit].x, st.units[unit].y);
        st.materialize_native_map_range_mode(0);
        self.ai_focus_cursor(x, y);
        let st = self.st.as_mut().expect("aiFocusCursor 不會清掉 g.st");
        st.materialize_native_map_range_mode(1);
        if st.native_map_view_state == view_before
            && st.native_map_hud_state == hud_before
            && self.native_map_vga.len() == 320 * 200
        {
            return Ok(());
        }
        self.compose_native_map_frame()
    }
}

/// R4 D 組新增，沒有 Go 對應：`beginNativeTransientPhases` 兩個閉包共用的區域變數。
pub(crate) struct NativeTransientRun {
    damage: Vec<battle::NativeTransientDamage>,
    damage_pages: Vec<Vec<Vec<Vec<u16>>>>,
    damage_waits: Vec<bool>,
    death_flags: BTreeMap<i64, u8>,
    expiries: Vec<NativeTransientExpiryFrame>,
}

struct NativeTransientExpiryFrame {
    unit: UnitIdx,
    counter: i64,
    portrait: dato::Frame,
}

/// Go：`native_transient_presentation.go:205`，`func nativeTransientPresentationEvent(event battle.NativeTransientExpiry) (*battle.Unit, int, error)`
/// （`r8`，隨呼叫端 `beginNativeTransientPhases` 補移植）。Rust 另收單位所在的 `state`。
pub(crate) fn native_transient_presentation_event(
    state: &battle::State,
    event: &battle::NativeTransientExpiry,
) -> Result<(UnitIdx, i64)> {
    let Some(unit) = event.unit.filter(|&u| state.units[u].has_battle_fig) else {
        return Err(game_err!("native transient presentation: raw DATO selector is unavailable"));
    };
    let counter_index = event.offset - battle::NATIVE_TRANSIENT_OFFSET;
    if !(0..battle::NATIVE_TRANSIENT_COUNT).contains(&counter_index) {
        return Err(game_err!("native transient presentation: raw counter offset is invalid"));
    }
    Ok((unit, counter_index))
}
