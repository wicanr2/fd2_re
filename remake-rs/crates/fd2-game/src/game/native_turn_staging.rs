//! Go 來源：`remake/cmd/fd2/native_turn_staging.go`（505 行）。
//!
//! 骨架：Go 檔的 13 個函式中產生 4 個（階段 0 清單），其餘標 `r8`，見 `PORTING.json`。

use super::*;

/// Go：`native_turn_staging.go:46`
#[derive(Debug, Clone, Copy, Default, PartialEq, Eq, PartialOrd, Ord, Hash)]
pub(crate) struct NativeTurnStagingPhase(pub(crate) u8);

/// Go 原式：`iota`
pub(crate) const NATIVE_TURN_STAGING_PAN: NativeTurnStagingPhase = NativeTurnStagingPhase(0);

/// Go：沿用上一個常數的式子（`iota` 序列）。
pub(crate) const NATIVE_TURN_STAGING_DELAY: NativeTurnStagingPhase = NativeTurnStagingPhase(1);

/// Go：沿用上一個常數的式子（`iota` 序列）。
pub(crate) const NATIVE_TURN_STAGING_FLASH: NativeTurnStagingPhase = NativeTurnStagingPhase(2);

/// nativeTurnStagingJob 擁有已閉合 handler 的阻塞式 0x35822 呼叫。raw DAC 與
/// indexed frame 分開保存；第一個鏡頭 tick 前先私下建立所有 roster 快照，避免
/// 後續 call 失敗時留下半套已物化 group。
///
/// Go：`native_turn_staging.go:57`
///
/// 不 derive `Debug`／`Clone`：欄位含 callback（`Box<dyn FnOnce>`）、檔案或亂數產生器等做不到的型別。
#[derive(Default)]
pub(crate) struct NativeTurnStagingJob {
    pub(crate) event: battle::NativeTurnEvent,
    pub(crate) states: Vec<battle::State>,
    pub(crate) call: i64,
    pub(crate) phase: NativeTurnStagingPhase,
    pub(crate) ticks: i64,
    pub(crate) vga: Vec<u8>,
    pub(crate) dac: Vec<u8>,
    pub(crate) baseline: Vec<u8>,
    pub(crate) palette: ColorPalette,
    pub(crate) drawn: bool,
    pub(crate) indexed: bool,
    pub(crate) then: Option<Box<dyn FnOnce(&mut Game) + Send>>,
}

/// Go：`native_turn_staging.go:115`，`func cloneNativeTurnStagingState(source *battle.State) (*battle.State, error)`
///
/// Go 淺複製 `*source` 後逐一深複製單位、名冊、選擇器快取與三個位元組陣列；Rust 的 `State::clone` 本來就是
/// 深複製（R2 所有權第 5 條），結果相同。簽名（C 組）：`source` 只讀，收 `Option<&State>`。
pub(crate) fn clone_native_turn_staging_state(source: Option<&battle::State>) -> Result<Option<battle::State>> {
    let Some(source) = source.filter(|s| {
        s.w > 0 && s.h > 0 && s.native_map_selector_cache.is_some() && s.native_map_selector_error.is_none()
    }) else {
        return Err(game_err!("event63 lacks a complete runtime roster/selector state"));
    };
    Ok(Some(source.clone()))
}

/// Go `nativeDelayTicks(ms)`（`native_indexed_transition.go:55`，B 組的 `r8` 函式）：毫秒換成 60 Hz 幀數、
/// 無條件進位。C 組另取名字，避免與 B 組補移植的同名函式衝突；整合時可合併成一個。
pub(crate) fn native_delay_ticks_ms(ms: i64) -> i64 {
    if ms <= 0 {
        return 0;
    }
    (ms * 60 + 999) / 1000
}

impl Game {
    /// startNativeRawCamp0TurnEvents is the sub_1A813(0) owner. Scenarios opt in by
    /// supplying native_turn_events; once opted in, every matching live row must
    /// have exactly one typed consumer or the enemy AI remains stopped.
    ///
    /// Go：`native_turn_staging.go:253`，`func (g *Game) startNativeRawCamp0TurnEvents() (bool, error)`
    pub(crate) fn start_native_raw_camp0_turn_events(&mut self) -> Result<bool> {
        let Some(sc) = self.sc.as_ref().filter(|sc| !sc.native_turn_events.is_empty()) else {
            return Ok(false);
        };
        let events = sc.native_turn_events_at(self.st.as_ref(), 0)?;
        // 帶轉寫動作的控制列事件由 runEditableTurnEvents(0) 在橫幅之後執行。
        let events: Vec<battle::NativeTurnEvent> =
            events.into_iter().filter(|event| event.actions.is_empty()).collect();
        if events.is_empty() {
            return Ok(false);
        }
        if events.len() != 1 {
            return Err(game_err!("raw camp0 has {} simultaneous handlers; staging is sequential", events.len()));
        }
        if events[0].pair_mutation.is_some() {
            // 79交給既有aiStep於13536之後處理；目前還沒清bit7。
            return Ok(false);
        }
        todo!("R8: native_turn_staging.go:279 beginNativeStagingJob（第一章重播沒有執行到）")
    }

    /// READY: fd2_ch29_flow_20261017.json event79_phase_contract。
    ///
    /// Go：`native_turn_staging.go:286`，`func (g *Game) applyNativeRawCamp0PairEvent() error`
    pub(crate) fn apply_native_raw_camp0_pair_event(&mut self) -> Result<()> {
        let Some(sc) = self.sc.as_ref().filter(|sc| !sc.native_turn_events.is_empty()) else {
            return Ok(());
        };
        if self.st.is_none() {
            return Ok(());
        }
        let events = sc.native_turn_events_at(self.st.as_ref(), 0)?;
        for event in events {
            if event.pair_mutation.is_none() {
                continue;
            }
            let mut seed = self.native_rng_state;
            if let Some(mut input) = self.native_turn_pair_rng_input.take() {
                let result = input(self, event.clone(), seed);
                self.native_turn_pair_rng_input = Some(input);
                seed = result?;
            }
            let (next, _) = battle::apply_native_turn_event79(self.st.as_mut(), &event, seed)?;
            self.native_rng_state = next;
        }
        Ok(())
    }

    /// Go：`native_turn_staging.go:404`，`func (g *Game) stepNativeTurnStaging()`
    pub(crate) fn step_native_turn_staging(&mut self) {
        let Some(job) = self.native_turn_staging.as_mut() else {
            return;
        };
        match job.phase {
            NATIVE_TURN_STAGING_PAN => {}
            NATIVE_TURN_STAGING_DELAY => {
                if job.ticks > 0 {
                    job.ticks -= 1;
                    return;
                }
                if job.indexed {
                    let s = &job.event.staging;
                    if let Err(err) = fdother::apply_vga_palette_delta(
                        &mut job.dac,
                        &job.baseline,
                        s.palette_start,
                        s.palette_end,
                        s.flash_delta,
                    ) {
                        self.fail_native_turn_staging(&format!("white flash: {err}"));
                        return;
                    }
                    match fdother::vga_palette_from_dac(&job.dac) {
                        Ok(palette) => job.palette = color_palette_from_nrgba(&palette),
                        Err(err) => {
                            self.fail_native_turn_staging(&format!("white palette: {err}"));
                            return;
                        }
                    }
                } else {
                    self.native_full_dac_white = true;
                }
                let job = self.native_turn_staging.as_mut().unwrap();
                (job.phase, job.ticks, job.drawn) =
                    (NATIVE_TURN_STAGING_FLASH, native_delay_ticks_ms(job.event.staging.flash_hold_ms), false);
            }
            NATIVE_TURN_STAGING_FLASH => {
                // Go 的 `drawn` 由 drawNativeTurnStaging（Draw）設；Rust 由呈現回報設（見 PORTING.md）。
                if !job.drawn {
                    return;
                }
                if job.ticks > 0 {
                    job.ticks -= 1;
                    return;
                }
                if job.indexed {
                    let s = &job.event.staging;
                    if let Err(err) = fdother::apply_vga_palette_delta(
                        &mut job.dac,
                        &job.baseline,
                        s.palette_start,
                        s.palette_end,
                        s.restore_delta,
                    ) {
                        self.fail_native_turn_staging(&format!("palette restore: {err}"));
                        return;
                    }
                    match fdother::vga_palette_from_dac(&job.dac) {
                        Ok(palette) => job.palette = color_palette_from_nrgba(&palette),
                        Err(err) => {
                            self.fail_native_turn_staging(&format!("restored palette: {err}"));
                            return;
                        }
                    }
                    if let Err(err) = self.compose_native_map_frame() {
                        self.fail_native_turn_staging(&format!("redraw: {}", err.message()));
                        return;
                    }
                    let vga = self.native_map_vga.clone();
                    let job = self.native_turn_staging.as_mut().unwrap();
                    go_copy(&mut job.vga, &vga);
                } else {
                    self.native_full_dac_white = false;
                }
                let job = self.native_turn_staging.as_mut().unwrap();
                job.call += 1;
                if job.call < job.event.staging.calls.len() as i64 {
                    todo!("R8: native_turn_staging.go:467 startNativeTurnStagingCall（第一章重播沒有執行到）");
                }
                let then = job.then.take();
                self.native_full_dac_white = false;
                self.native_turn_staging = None;
                if let Some(then) = then {
                    then(self);
                }
            }
            _ => {}
        }
    }

    /// Go：`native_turn_staging.go:470`，`func (g *Game) failNativeTurnStaging(message string)`（`r8`，隨
    /// `stepNativeTurnStaging` 補移植）。
    fn fail_native_turn_staging(&mut self, message: &str) {
        self.load_err = format!("native turn staging: {message}");
        self.native_full_dac_white = false;
        self.cam_pan = None;
        self.native_turn_staging = None;
    }
}
