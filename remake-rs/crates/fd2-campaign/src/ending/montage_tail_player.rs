//! Go 來源：`remake/internal/ending/montage_tail_player.go`（357 行）。

use super::*;

/// MontageTailPhase describes the source-bound visual bridge for the proven
/// 20-entry 0x2c194 schedule. It implements the reachable header-byte1-zero
/// 0x2939d pair loop but does not claim exact call-time record continuity,
/// sound ownership, DOS timing or player-path E2.
///
/// Go：`montage_tail_player.go:14`
#[derive(Debug, Clone, Copy, Default, PartialEq, Eq, PartialOrd, Ord, Hash)]
pub struct MontageTailPhase(pub &'static str);

pub const MONTAGE_TAIL_PHASE_INTRO: MontageTailPhase = MontageTailPhase("intro");

pub const MONTAGE_TAIL_PHASE_RECORD0_AUX: MontageTailPhase = MontageTailPhase("record0_aux");

pub const MONTAGE_TAIL_PHASE_RECORD1_AUX: MontageTailPhase = MontageTailPhase("record1_aux");

pub const MONTAGE_TAIL_PHASE_BASE_HOLD: MontageTailPhase = MontageTailPhase("base_hold");

pub const MONTAGE_TAIL_PHASE_OVERLAY: MontageTailPhase = MontageTailPhase("fdother_58_overlay");

pub const MONTAGE_TAIL_PHASE_FINAL: MontageTailPhase = MontageTailPhase("final");

pub const MONTAGE_TAIL_PHASE_COMPLETED: MontageTailPhase = MontageTailPhase("completed");

pub(crate) const NATIVE_MONTAGE_TAIL_INTRO_WAIT_TICKS: i64 = 0x50;

/// Go：`montage_tail_player.go:28`
pub(crate) static NATIVE_MONTAGE_TAIL_HORIZONTAL_OFFSETS: [i64; 6] = [0, 4, 9, 14, 18, 14];

/// Go：`montage_tail_player.go:29`
pub(crate) static NATIVE_MONTAGE_TAIL_VERTICAL_OFFSETS: [i64; 6] = [0, 2, 4, 6, 8, 10];

/// MontageTailPlayer turns the source-proven resource schedule into a bounded,
/// deterministic indexed presentation. Campaign admission is owned by the
/// source-bound E1 caller; this player owns no campaign, party, battle or save
/// state and does not claim exact native 0x28a6c rendering.
///
/// Go：`montage_tail_player.go:36`
#[derive(Debug, Clone, Default)]
pub struct MontageTailPlayer {
    pub tail: MontageTail,
    pub assets: MontageTailAssets,
    pub visual_sets: Vec<MontageTailVisualSet>,
    pub entries: Vec<MontageTailEntry>,
    pub compositor: Option<SharedCompositor>,
    pub phase: MontageTailPhase,
    pub segment: i64,
    pub frame: i64,
    pub inner: i64,
    pub base_frame: i64,
    pub base_inner: i64,
    pub effect_step: i64,
    /// SoundMarker preserves raw frame +5 for tracing. The player deliberately
    /// does not emit audio while the two native sound-owner globals are unknown.
    pub sound_marker: u8,
    pub delay_ticks: i64,
}

/// NewMontageTailPlayer validates every source asset before exposing any
/// playable state. This all-or-nothing admission prevents a missing late entry
/// from producing a partially plausible ending.
///
/// Go：`montage_tail_player.go:58`，`func NewMontageTailPlayer(tail MontageTail, assets MontageTailAssets, sets []MontageTailVisualSet, compositor *IndexedCompositor) (*MontageTailPlayer, error)`
pub fn new_montage_tail_player(
    tail: &MontageTail,
    assets: &MontageTailAssets,
    sets: &[MontageTailVisualSet],
    compositor: Option<SharedCompositor>,
) -> Result<MontageTailPlayer> {
    let entries = tail.plan()?;
    let Some(compositor) = compositor.filter(|c| {
        assets.loop_palette.len() == lock_compositor(c).palette.len()
            && assets.loop_frames.len() == entries.len()
            && sets.len() == entries.len()
    }) else {
        return Err(campaign_err!("ending: montage tail player source provenance is incomplete"));
    };
    if assets.intro.width != WIDTH
        || assets.intro.height != HEIGHT
        || assets.r#final.width != WIDTH
        || assets.r#final.height != HEIGHT
    {
        return Err(campaign_err!("ending: montage tail player full-screen assets are incomplete"));
    }
    for (index, set) in sets.iter().enumerate() {
        let (Some(record0_base), Some(record0_aux), Some(record1_base), Some(record1_aux)) = (
            set.record0_figani_base.as_ref(),
            set.record0_figani_aux.as_ref(),
            set.record1_figani_base.as_ref(),
            set.record1_figani_aux.as_ref(),
        ) else {
            return Err(campaign_err!("ending: montage tail visual set {index} is incomplete"));
        };
        if set.plan.index != index as i64
            || record0_base.frames.is_empty()
            || record0_aux.frames.is_empty()
            || record1_base.frames.is_empty()
            || record1_aux.frames.is_empty()
        {
            return Err(campaign_err!("ending: montage tail visual set {index} is incomplete"));
        }
        validate_montage_tail_player_animation(Some(record0_base))
            .map_err(|err| err.wrap(format!("ending: montage tail entry {index} record0 base: ")))?;
        validate_montage_tail_player_animation(Some(record0_aux))
            .map_err(|err| err.wrap(format!("ending: montage tail entry {index} record0 aux: ")))?;
        validate_montage_tail_player_animation(Some(record1_base))
            .map_err(|err| err.wrap(format!("ending: montage tail entry {index} record1 base: ")))?;
        validate_montage_tail_player_animation(Some(record1_aux))
            .map_err(|err| err.wrap(format!("ending: montage tail entry {index} record1 aux: ")))?;
        // Go 以 map 迭代這四個動畫；只影響同時有多個不合法時先報哪一個（PORTING.md）。
        for (name, animation) in [
            ("record0 aux", record0_aux),
            ("record0 base", record0_base),
            ("record1 aux", record1_aux),
            ("record1 base", record1_base),
        ] {
            if animation.header_byte1 != 0 || animation.header_byte2 as usize != animation.frames.len() {
                return Err(campaign_err!(
                    "ending: montage tail entry {index} {name} has unsupported native prelude header {}/{}",
                    animation.header_byte1,
                    animation.header_byte2
                ));
            }
        }
        for (name, animation) in [("record0 base", record0_base), ("record1 base", record1_base)] {
            for (frame, descriptor) in animation.frames.iter().enumerate() {
                if descriptor.delay <= 0 {
                    return Err(campaign_err!(
                        "ending: montage tail entry {index} {name} frame {frame} has invalid base delay {}",
                        descriptor.delay
                    ));
                }
            }
        }
    }
    Ok(MontageTailPlayer {
        tail: tail.clone(),
        assets: assets.clone(),
        visual_sets: sets.to_vec(),
        entries,
        compositor: Some(compositor),
        phase: MONTAGE_TAIL_PHASE_INTRO,
        ..Default::default()
    })
}

/// Go：`montage_tail_player.go:112`，`func validateMontageTailPlayerAnimation(animation *figani.Animation) error`
pub(crate) fn validate_montage_tail_player_animation(animation: Option<&figani::Animation>) -> Result<()> {
    let Some(animation) = animation.filter(|a| !a.frames.is_empty()) else {
        return Err(campaign_err!("FIGANI animation is unavailable"));
    };
    for (index, frame) in animation.frames.iter().enumerate() {
        if frame.x < 0 || frame.y < 0 || frame.x + frame.width > WIDTH || frame.y + frame.height > HEIGHT {
            return Err(campaign_err!("FIGANI frame {index} is outside the indexed viewport"));
        }
    }
    Ok(())
}

impl MontageTailPlayer {
    /// Ready reports that all 20 source transactions completed and the stable
    /// FDOTHER#59 frame was presented. Further Step calls are rejected so the
    /// caller can hold that image indefinitely.
    ///
    /// Go：`montage_tail_player.go:127`，`func (p *MontageTailPlayer) Ready() bool`
    /// Go 版先檢查 nil 接收者；Rust 的 `self` 不會是 null，該分支不移植。
    pub fn ready(&self) -> bool {
        self.phase == MONTAGE_TAIL_PHASE_COMPLETED
    }

    /// Step presents one source-bound native inner iteration and exposes the next
    /// raw-tick delay. Each auxiliary descriptor +6 owns that many presentations;
    /// the paired base scheduler advances after every one. Zero-delay descriptors
    /// publish no frame. The 20/78 waits around FDOTHER#58 remain caller-owned.
    ///
    /// Go：`montage_tail_player.go:135`，`func (p *MontageTailPlayer) Step() error`
    /// Go 版先檢查 nil 接收者；Rust 的 `self` 不會是 null，該分支不移植。
    pub fn step(&mut self) -> Result<()> {
        if self.ready() {
            return Err(campaign_err!("ending: montage tail player is not running"));
        }
        let Some(shared) = self.compositor.clone() else {
            return Err(campaign_err!("ending: montage tail player is not running"));
        };
        self.delay_ticks = 0;
        match self.phase {
            MONTAGE_TAIL_PHASE_INTRO => {
                {
                    let mut c = lock_compositor(&shared);
                    // Go `copy(dst[:], src)`：長度已在建構時確認同為 768。
                    let n = c.palette.len().min(self.assets.loop_palette.len());
                    c.palette[..n].copy_from_slice(&self.assets.loop_palette[..n]);
                    c.baseline[..n].copy_from_slice(&self.assets.loop_palette[..n]);
                    c.baseline_known = true;
                    c.vga.fill(0);
                    self.assets
                        .intro
                        .blit(&mut c.vga, WIDTH, -1)
                        .map_err(|err| CampaignError::from(err).wrap("ending: montage tail intro: "))?;
                }
                // 0x2c1fd calls 0x17aa9(0x50) after FDOTHER#60 is decoded and
                // track 18 is selected, before the following presentation helper.
                self.delay_ticks = NATIVE_MONTAGE_TAIL_INTRO_WAIT_TICKS;
                self.phase = MONTAGE_TAIL_PHASE_RECORD0_AUX;
                self.reset_pair_state();
                Ok(())
            }
            MONTAGE_TAIL_PHASE_RECORD0_AUX | MONTAGE_TAIL_PHASE_RECORD1_AUX => {
                let record0 = self.phase == MONTAGE_TAIL_PHASE_RECORD0_AUX;
                self.current_set()?;
                // Go 以值傳遞 set；Rust 暫時把 visual_sets 移出，避免與 `&mut self` 重疊借用。
                let sets = std::mem::take(&mut self.visual_sets);
                let set = &sets[self.segment as usize];
                let (auxiliary, base) = if record0 {
                    (set.record0_figani_aux.as_ref(), set.record1_figani_base.as_ref())
                } else {
                    (set.record1_figani_aux.as_ref(), set.record0_figani_base.as_ref())
                };
                let result = match (auxiliary, base) {
                    (Some(auxiliary), Some(base)) => self.step_native_pair(set, auxiliary, base, record0, !record0),
                    _ => Err(campaign_err!("ending: montage tail visual set {} is incomplete", self.segment)),
                };
                self.visual_sets = sets;
                let (_, done) = result?;
                if done {
                    self.phase = if record0 { MONTAGE_TAIL_PHASE_RECORD1_AUX } else { MONTAGE_TAIL_PHASE_BASE_HOLD };
                    self.reset_pair_state();
                }
                Ok(())
            }
            MONTAGE_TAIL_PHASE_BASE_HOLD => {
                // 0x2c2a6 waits on the last frame already published by the second
                // 0x2939d call; it does not synthesize an extra two-base pose.
                self.delay_ticks = self.tail.r#loop.wait_before_frame_ticks;
                self.phase = MONTAGE_TAIL_PHASE_OVERLAY;
                Ok(())
            }
            MONTAGE_TAIL_PHASE_OVERLAY => {
                if self.segment < 0 || self.segment >= self.assets.loop_frames.len() as i64 {
                    return Err(campaign_err!("ending: montage tail overlay index is out of range"));
                }
                self.assets.loop_frames[self.segment as usize]
                    .blit(&mut lock_compositor(&shared).vga, WIDTH, -1)
                    .map_err(|err| {
                        CampaignError::from(err)
                            .wrap(format!("ending: montage tail FDOTHER#58 frame {}: ", self.segment))
                    })?;
                self.delay_ticks = self.tail.r#loop.wait_after_frame_ticks;
                self.segment += 1;
                self.reset_pair_state();
                self.phase = if self.segment >= self.visual_sets.len() as i64 {
                    MONTAGE_TAIL_PHASE_FINAL
                } else {
                    MONTAGE_TAIL_PHASE_RECORD0_AUX
                };
                Ok(())
            }
            MONTAGE_TAIL_PHASE_FINAL => {
                self.assets.present_final(Some(&mut lock_compositor(&shared)))?;
                self.phase = MONTAGE_TAIL_PHASE_COMPLETED;
                Ok(())
            }
            other => Err(campaign_err!("ending: unknown montage tail phase {}", battle::go_quote(other.0))),
        }
    }

    /// Go：`montage_tail_player.go:216`，`func (p *MontageTailPlayer) resetPairState()`
    pub(crate) fn reset_pair_state(&mut self) {
        self.frame = 0;
        self.inner = 0;
        self.base_frame = 0;
        self.base_inner = 0;
        self.effect_step = 0;
        self.sound_marker = 0;
    }

    /// Go：`montage_tail_player.go:225`，`func (p *MontageTailPlayer) currentSet() (*MontageTailVisualSet, error)`
    pub(crate) fn current_set(&self) -> Result<&MontageTailVisualSet> {
        if self.segment < 0 || self.segment >= self.visual_sets.len() as i64 {
            return Err(campaign_err!("ending: montage tail visual index is out of range"));
        }
        Ok(&self.visual_sets[self.segment as usize])
    }

    /// Go：`montage_tail_player.go:232`，`func (p *MontageTailPlayer) stepNativePair(set MontageTailVisualSet, auxiliary, base *figani.Animation, record0, terminal bool) (presented, done bool, err error)`
    /// Go 具名回傳值依序為：presented、done、err。
    pub(crate) fn step_native_pair(
        &mut self,
        set: &MontageTailVisualSet,
        auxiliary: &figani::Animation,
        base: &figani::Animation,
        record0: bool,
        terminal: bool,
    ) -> Result<(bool, bool)> {
        let mut last_effect = -1i64;
        if terminal {
            for (index, frame) in auxiliary.frames.iter().enumerate() {
                if frame.raw_byte4 != 0 {
                    last_effect = index as i64;
                }
            }
        }
        for _skipped in 0..=auxiliary.frames.len() {
            if self.frame >= auxiliary.frames.len() as i64 {
                return Ok((false, true));
            }
            let frame_index = self.frame;
            let frame = &auxiliary.frames[frame_index as usize];
            if self.inner == 0 && frame.raw_byte4 != 0 {
                self.effect_step = NATIVE_MONTAGE_TAIL_HORIZONTAL_OFFSETS.len() as i64 - 1;
            }
            if frame.delay <= 0 {
                self.frame += 1;
                self.inner = 0;
                if terminal && last_effect >= 0 && frame_index == last_effect {
                    return Ok((false, true));
                }
                continue;
            }
            let base_frame = &base.frames[self.base_frame as usize];
            let opaque_fill = if frame.raw_byte4 != 0 && self.inner == 0 { Some(33u8) } else { None };
            self.render_source_bound_pair(set, frame, base_frame, record0, self.effect_step, opaque_fill)?;
            self.sound_marker = frame.raw_byte5;
            self.delay_ticks = 1;
            self.advance_base_scheduler(base);
            self.inner += 1;
            if self.effect_step > 0 {
                self.effect_step -= 1;
            }
            if self.inner >= frame.delay {
                self.frame += 1;
                self.inner = 0;
                if terminal && last_effect >= 0 && frame_index == last_effect {
                    return Ok((true, true));
                }
            }
            return Ok((true, false));
        }
        Err(campaign_err!("ending: montage tail zero-delay frame scan exceeded animation"))
    }

    /// Go：`montage_tail_player.go:286`，`func (p *MontageTailPlayer) advanceBaseScheduler(base *figani.Animation)`
    pub(crate) fn advance_base_scheduler(&mut self, base: &figani::Animation) {
        self.base_inner += 1;
        if self.base_inner >= base.frames[self.base_frame as usize].delay {
            self.base_inner = 0;
            self.base_frame += 1;
            if self.base_frame >= base.frames.len() as i64 {
                self.base_frame = 0;
            }
        }
    }

    /// renderSourceBoundPair reproduces the reachable header-byte1-zero branch of
    /// the two 0x2939d calls. The native 400-stride staging margin is represented
    /// by clipped translated blits on the final 320-wide indexed surface.
    ///
    /// Go：`montage_tail_player.go:300`，`func (p *MontageTailPlayer) renderSourceBoundPair(set MontageTailVisualSet, auxiliary, base figani.Frame, record0 bool, effectStep int, opaqueFill *byte) error`
    pub(crate) fn render_source_bound_pair(
        &mut self,
        set: &MontageTailVisualSet,
        auxiliary: &figani::Frame,
        base: &figani::Frame,
        record0: bool,
        effect_step: i64,
        opaque_fill: Option<u8>,
    ) -> Result<()> {
        let Some(shared) = self.compositor.as_ref() else {
            return Err(campaign_err!("ending: montage tail player is not running"));
        };
        let mut guard = lock_compositor(shared);
        let vga = &mut guard.vga;
        vga.fill(0);
        let mut bg = set.bg.clone();
        bg.y = 50;
        bg.blit(vga, WIDTH, -1)
            .map_err(|err| CampaignError::from(err).wrap(format!("ending: montage tail BG#{}: ", set.plan.bg)))?;
        let mut tai = set.tai.clone();
        (tai.x, tai.y) = (164, 157);
        tai.blit(vga, WIDTH, -1)
            .map_err(|err| CampaignError::from(err).wrap(format!("ending: montage tail TAI#{}: ", set.plan.tai)))?;
        // The two proven 0x2939d calls pair record0's auxiliary animation with
        // record1's base, then record1's auxiliary animation with record0's base.
        // Do not draw the animated record's own base underneath it: that creates a
        // remake-only double figure which the call shape does not support.
        let entry = &self.entries[self.segment as usize];
        let (base_resource, side) = if record0 {
            (set.plan.record1_figani_base, entry.record0_byte6)
        } else {
            (set.plan.record0_figani_base, entry.record1_byte6)
        };
        if effect_step < 0 || effect_step >= NATIVE_MONTAGE_TAIL_HORIZONTAL_OFFSETS.len() as i64 {
            return Err(campaign_err!("ending: montage tail effect displacement is out of range"));
        }
        let (mut dx, mut dy) = (
            NATIVE_MONTAGE_TAIL_HORIZONTAL_OFFSETS[effect_step as usize],
            NATIVE_MONTAGE_TAIL_VERTICAL_OFFSETS[effect_step as usize],
        );
        if side != 0 {
            (dx, dy) = (-dx, -dy);
        }
        let aux_first = (side != 0 && auxiliary.raw_byte7 & 1 != 0) || (side == 0 && auxiliary.raw_byte7 & 1 == 0);
        let aux_err = |err: fd2_formats::figani::FiganiError| {
            CampaignError::from(err).wrap("ending: montage tail auxiliary FIGANI: ")
        };
        if aux_first {
            auxiliary.blit_translated(vga, WIDTH, 0, 0, None).map_err(aux_err)?;
        }
        base.blit_translated(vga, WIDTH, dx, dy, opaque_fill).map_err(|err| {
            CampaignError::from(err).wrap(format!("ending: montage tail paired base FIGANI#{base_resource}: "))
        })?;
        if !aux_first {
            auxiliary.blit_translated(vga, WIDTH, 0, 0, None).map_err(aux_err)?;
        }
        if auxiliary.width <= 0 || auxiliary.height <= 0 {
            let owner = if record0 { "record0" } else { "record1" };
            return Err(campaign_err!("ending: montage tail {owner} auxiliary FIGANI is empty"));
        }
        Ok(())
    }
}

#[cfg(test)]
mod montage_tail_player_test;
