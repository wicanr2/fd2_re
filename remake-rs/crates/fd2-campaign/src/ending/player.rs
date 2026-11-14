//! Go 來源：`remake/internal/ending/player.go`（370 行）。

use super::*;

/// Player advances the evidence-backed prefix of a native ending against a
/// millisecond presentation clock.  It deliberately blocks at the first
/// unsupported native operation, retaining the last verified VGA frame for a
/// UI adapter to display.  It is not a permissive generic scripting runtime.
///
/// Go：`player.go:15`
#[derive(Debug, Clone, Default)]
pub struct Player {
    pub timeline: Timeline,
    pub frames: Vec<fdother::Frame>,
    pub ani: Option<afm::Clip>,
    pub compositor: Option<SharedCompositor>,
    pub segment: i64,
    pub wait_ms: i64,
    /// Rust 名稱改為 `ani_player`：Go 欄位 `ani` 與 `ANI` 轉成 snake_case 後相同。
    pub(crate) ani_player: Option<ANIPlayer>,
    pub(crate) ramp: Option<PaletteRamp>,
    pub(crate) composite: Option<Composite40>,
    pub(crate) composite200: Option<Composite200>,
    pub(crate) phase0: Option<Phase0Player>,
    pub(crate) phase0_spec: Option<FinalePhase>,
    pub(crate) phase0_assets: Option<Phase0Assets>,
    pub blocked: Option<Segment>,
    pub state: PlaybackState,
}

impl Player {
    /// EnableRecoveredPhase0 supplies the separated text/font assets that 0x2c405
    /// consumes after the already-recovered 0x2bce5 prefix. It does not make the
    /// ending generally playable: it only permits the exact 0x2c172 hand-off and
    /// will block again at the verified 0x2c548 montage boundary.
    ///
    /// Palette input deliberately is not accepted here.  The native routine keeps
    /// the DAC state produced by the preceding ending presentation, so the bridge
    /// takes it only from this Player's compositor after PresentANI has captured
    /// it.
    ///
    /// Go：`player.go:43`，`func (p *Player) EnableRecoveredPhase0(phase FinalePhase, assets Phase0Assets) error`
    /// Go 版先檢查 nil 接收者；Rust 的 `self` 不會是 null，該分支不移植。
    pub fn enable_recovered_phase0(&mut self, phase: &FinalePhase, assets: &Phase0Assets) -> Result<()> {
        let assets_missing = (assets.strings.is_none() || assets.font.is_none())
            && (assets.text_resource.is_empty() || assets.font_resource.is_empty());
        if self.state != PLAYBACK_RUNNING
            || phase.ready()
            || phase.native_handler != "0x2c405"
            || phase.gate.source != "0x2c548"
            || assets_missing
        {
            return Err(campaign_err!("ending: invalid recovered phase-0 inputs"));
        }
        self.phase0_spec = Some(phase.clone());
        self.phase0_assets = Some(assets.clone());
        Ok(())
    }
}

/// Go：`player.go:55`
#[derive(Debug, Clone, Copy, Default, PartialEq, Eq)]
pub(crate) struct Composite40 {
    pub(crate) iteration: i64,
    pub(crate) delay: i64,
}

/// Go：`player.go:56`
#[derive(Debug, Clone, Copy, Default, PartialEq, Eq)]
pub(crate) struct Composite200 {
    pub(crate) iteration: i64,
    pub(crate) delay: i64,
}

/// Go：`player.go:58`
#[derive(Debug, Clone, Copy, Default, PartialEq, Eq)]
pub(crate) struct PaletteRamp {
    pub(crate) value: i64,
    pub(crate) end: i64,
    pub(crate) step: i64,
    pub(crate) delay: i64,
    pub(crate) next: bool,
}

/// Go：`player.go:63`，`func NewPlayer(t Timeline, frames []fdother.Frame, ani *afm.Clip, compositor *IndexedCompositor) (*Player, error)`
pub fn new_player(
    t: Timeline,
    frames: &[fdother::Frame],
    ani: Option<&afm::Clip>,
    compositor: Option<SharedCompositor>,
) -> Result<Player> {
    let Some(compositor) = compositor else {
        return Err(campaign_err!("ending: nil compositor"));
    };
    if t.segments.is_empty() {
        return Err(campaign_err!("ending: timeline has no segments"));
    }
    let mut t = t;
    let mut segments = Vec::with_capacity(t.segments.len());
    for segment in &t.segments {
        if segment.op == "blit_frame_sequence" {
            // Go 直接解參照 `*segment.FirstFrame`；未經 LoadTimeline 驗證的 nil 會 panic，Rust 視為空範圍。
            let (first, last) = (segment.first_frame.unwrap_or(0), segment.last_frame.unwrap_or(-1));
            for frame in first..=last {
                segments.push(Segment {
                    op: "blit_frame".into(),
                    source: segment.source.clone(),
                    frame: Some(frame),
                    target: segment.target.clone(),
                    stride: segment.stride,
                    transparent: segment.transparent,
                    ..Default::default()
                });
                segments.push(Segment {
                    op: "delay_ms".into(),
                    source: segment.source.clone(),
                    ms: segment.palette_delay,
                    ..Default::default()
                });
            }
            continue;
        }
        if segment.op != "palette_ramp_repeat" {
            segments.push(segment.clone());
            continue;
        }
        for _ in 0..segment.repeat {
            segments.push(Segment {
                op: "palette_ramp".into(),
                source: segment.source.clone(),
                palette_start: segment.palette_start,
                palette_end: segment.palette_end,
                palette_step: segment.palette_step,
                palette_delay: segment.palette_delay,
                ..Default::default()
            });
            segments.push(Segment {
                op: "delay_ms".into(),
                source: segment.source.clone(),
                ms: segment.tail_delay,
                ..Default::default()
            });
        }
    }
    t.segments = segments;
    Ok(Player {
        timeline: t,
        frames: frames.to_vec(),
        ani: ani.cloned(),
        compositor: Some(compositor),
        state: PLAYBACK_RUNNING,
        ..Default::default()
    })
}

impl Player {
    /// VerifiedAudioCues returns the raw ending audio schedule in source order.
    /// The method is intentionally observational: callers must not treat the
    /// returned list as permission to bypass the Player's blocked montage gate.
    /// A future montage owner can consume each cue only after reaching its
    /// Trigger stage with the corresponding native renderer evidence.
    ///
    /// Go：`player.go:104`，`func (p *Player) VerifiedAudioCues() []AudioCue`
    /// Go 版先檢查 nil 接收者；Rust 的 `self` 不會是 null，該分支不移植。
    pub fn verified_audio_cues(&self) -> Vec<AudioCue> {
        self.timeline.audio_cues.clone()
    }

    /// AudioCueForRuntimeStage 只依資料中已驗證且不重複的執行期階段取出 cue；
    /// 它不播放音訊，也不允許 caller 略過對應的畫面／資產 admission。
    ///
    /// Go：`player.go:113`，`func (p *Player) AudioCueForRuntimeStage(stage string) (AudioCue, bool)`
    /// Go 版先檢查 nil 接收者；Rust 的 `self` 不會是 null，該分支不移植。
    pub fn audio_cue_for_runtime_stage(&self, stage: &str) -> (AudioCue, bool) {
        if stage.is_empty() {
            return (AudioCue::default(), false);
        }
        match self.timeline.audio_cues.iter().find(|cue| cue.runtime_stage == stage) {
            Some(cue) => (cue.clone(), true),
            None => (AudioCue::default(), false),
        }
    }

    /// AudioCueAtBlockedBoundary 只回傳 after_gate 與目前失敗即關閉邊界精確相符的
    /// 音訊 cue。它不前進播放、不略過 renderer，也不讓尚未還原 owner 的 cue 被
    /// 提前消費。
    ///
    /// Go：`player.go:128`，`func (p *Player) AudioCueAtBlockedBoundary() (AudioCue, bool)`
    /// Go 版先檢查 nil 接收者；Rust 的 `self` 不會是 null，該分支不移植。
    pub fn audio_cue_at_blocked_boundary(&self) -> (AudioCue, bool) {
        let Some(blocked) = self.blocked.as_ref().filter(|_| self.state == PLAYBACK_BLOCKED) else {
            return (AudioCue::default(), false);
        };
        match self.timeline.audio_cues.iter().find(|cue| !cue.after_gate.is_empty() && cue.after_gate == blocked.source)
        {
            Some(cue) => (cue.clone(), true),
            None => (AudioCue::default(), false),
        }
    }

    /// Advance consumes up to elapsedMS of wall-clock time.  The caller should use
    /// a monotonic presentation clock; zero is useful to present the first ANI
    /// frame immediately, exactly as 0x20421 does.
    ///
    /// Go：`player.go:143`，`func (p *Player) Advance(elapsedMS int) (PlaybackState, error)`
    pub fn advance(&mut self, elapsed_ms: i64) -> Result<PlaybackState> {
        let mut elapsed_ms = elapsed_ms;
        if elapsed_ms < 0 {
            return Err(campaign_err!("ending: negative elapsed time"));
        }
        if self.state != PLAYBACK_RUNNING {
            return Ok(self.state);
        }
        let Some(shared) = self.compositor.clone() else {
            return Err(campaign_err!("ending: nil compositor"));
        };
        // 合成器的鎖只在單一步驟內持有；Phase0Player 共用同一個合成器，呼叫它之前不能持有鎖。
        loop {
            if let Some(phase0) = self.phase0.as_mut() {
                let done = phase0.advance(elapsed_ms)?;
                if !done {
                    return Ok(self.state);
                }
                // 0x2c548 begins a distinct resource-heavy montage.  Reaching it
                // proves phase 0, not the finale, so retain its native boundary.
                self.phase0 = None;
                let gate = self.phase0_spec.as_ref().map(|spec| spec.gate.source.clone()).unwrap_or_default();
                self.blocked =
                    Some(Segment { op: "native_finale_montage_opaque".into(), source: gate, ..Default::default() });
                self.state = PLAYBACK_BLOCKED;
                return Ok(self.state);
            }
            if self.wait_ms > 0 {
                if elapsed_ms < self.wait_ms {
                    self.wait_ms -= elapsed_ms;
                    return Ok(self.state);
                }
                elapsed_ms -= self.wait_ms;
                self.wait_ms = 0;
                if let Some(ramp) = self.ramp.as_mut() {
                    ramp.next = true;
                } else if self.composite.is_some() || self.composite200.is_some() {
                    // The next pass is selected below; it owns the same segment.
                } else {
                    self.segment += 1;
                }
            }
            if let Some(ramp) = self.ramp.as_mut() {
                if ramp.next {
                    ramp.value += ramp.step;
                    ramp.next = false;
                    if (ramp.step < 0 && ramp.value < ramp.end) || (ramp.step > 0 && ramp.value > ramp.end) {
                        self.ramp = None;
                        self.segment += 1;
                        continue;
                    }
                }
                lock_compositor(&shared).palette_delta(0, 255, ramp.value)?;
                self.wait_ms = ramp.delay;
                ramp.next = true;
                if elapsed_ms == 0 {
                    return Ok(self.state);
                }
                continue;
            }
            if let Some(composite) = self.composite.as_mut() {
                lock_compositor(&shared).composite40(&self.frames, composite.iteration)?;
                composite.iteration += 1;
                if composite.iteration >= 40 {
                    self.composite = None;
                    self.segment += 1;
                    continue;
                }
                self.wait_ms = composite.delay;
                if elapsed_ms == 0 {
                    return Ok(self.state);
                }
                continue;
            }
            if let Some(composite) = self.composite200.as_mut() {
                lock_compositor(&shared).composite200(&self.frames, composite.iteration)?;
                composite.iteration += 1;
                if composite.iteration >= 200 {
                    self.composite200 = None;
                    self.segment += 1;
                    continue;
                }
                self.wait_ms = composite.delay;
                if elapsed_ms == 0 {
                    return Ok(self.state);
                }
                continue;
            }
            if let Some(ani) = self.ani_player.as_mut() {
                let done = ani.advance(&mut lock_compositor(&shared), elapsed_ms)?;
                if !done {
                    return Ok(self.state);
                }
                self.ani_player = None;
                self.segment += 1;
                // ANIPlayer owns its exact cadence.  On its completion tick any
                // remaining scheduler work starts on the following presentation tick.
                return Ok(self.state);
            }
            if self.segment >= self.timeline.segments.len() as i64 {
                self.state = PLAYBACK_COMPLETED;
                return Ok(self.state);
            }

            let s = self.timeline.segments[self.segment as usize].clone();
            match s.op.as_str() {
                "blit_frame" => {
                    self.blit(&s)?;
                    self.segment += 1;
                }
                "copy_buffer" => {
                    if s.from != "offscreen" || s.to != "vga" || s.bytes != BYTES {
                        return Err(campaign_err!("ending: unsupported copy at {}", s.source));
                    }
                    let mut compositor = lock_compositor(&shared);
                    let offscreen = std::mem::take(&mut compositor.offscreen);
                    let copied = compositor.copy_to_vga(&offscreen);
                    compositor.offscreen = offscreen;
                    copied?;
                    self.segment += 1;
                }
                "delay_ms" => {
                    if s.ms < 0 {
                        return Err(campaign_err!("ending: invalid delay at {}", s.source));
                    }
                    self.wait_ms = s.ms;
                    if self.wait_ms == 0 {
                        self.segment += 1;
                    }
                }
                "ani_play" => {
                    if s.ani_resource != Some(2)
                        || s.frame_delay_ms != 100
                        || s.skippable != Some(false)
                        || self.ani.is_none()
                    {
                        return Err(campaign_err!("ending: unavailable ANI at {}", s.source));
                    }
                    self.ani_player =
                        Some(ANIPlayer { clip: self.ani.clone(), delay_ms: s.frame_delay_ms, ..Default::default() });
                }
                "palette_update" => {
                    let (Some(start), Some(end), Some(value)) = (s.palette_start, s.palette_end, s.palette_value)
                    else {
                        return Err(campaign_err!("ending: incomplete palette update at {}", s.source));
                    };
                    lock_compositor(&shared).palette_delta(start, end, value)?;
                    self.segment += 1;
                }
                "palette_ramp" => {
                    let Some(start) = s
                        .palette_start
                        .filter(|_| s.palette_end.is_some() && s.palette_step != 0 && s.palette_delay > 0)
                    else {
                        return Err(campaign_err!("ending: incomplete palette ramp at {}", s.source));
                    };
                    self.ramp = Some(PaletteRamp {
                        value: start,
                        end: s.palette_end.unwrap_or_default(),
                        step: s.palette_step,
                        delay: s.palette_delay,
                        next: false,
                    });
                }
                "native_composite_loop_opaque" => {
                    // The first native loop is fully recovered.  Only its known call site
                    // is executable: other opaque operations must stay fail-closed.
                    if s.source != "0x2bf60" {
                        (self.blocked, self.state) = (Some(s), PLAYBACK_BLOCKED);
                        return Ok(self.state);
                    }
                    self.composite = Some(Composite40 { iteration: 0, delay: 20 });
                }
                "native_composite_loop_baseline" => {
                    self.composite200 = Some(Composite200 { iteration: 0, delay: 20 });
                }
                "native_post_composite_opaque" => {
                    // 0x2c172 clears VGA and enters 0x2c405.  The selector is narrow:
                    // any other native post-composite call remains an opaque stop.
                    let baseline_known = lock_compositor(&shared).baseline_known();
                    let (Some(spec), Some(assets)) = (self.phase0_spec.as_ref(), self.phase0_assets.as_ref()) else {
                        (self.blocked, self.state) = (Some(s), PLAYBACK_BLOCKED);
                        return Ok(self.state);
                    };
                    if s.source != "0x2c172" || !baseline_known {
                        (self.blocked, self.state) = (Some(s), PLAYBACK_BLOCKED);
                        return Ok(self.state);
                    }
                    let mut assets = assets.clone();
                    assets.baseline = lock_compositor(&shared).baseline;
                    let phase0 = new_phase0_player_from_assets(spec, &assets, Some(shared.clone()))?;
                    lock_compositor(&shared).vga.fill(0);
                    self.phase0 = Some(phase0);
                }
                _ => {
                    self.blocked = Some(s);
                    self.state = PLAYBACK_BLOCKED;
                    return Ok(self.state);
                }
            }
        }
    }

    /// Go：`player.go:326`，`func (p *Player) blit(s Segment) error`
    pub(crate) fn blit(&mut self, s: &Segment) -> Result<()> {
        let frame = match (s.frame, s.transparent) {
            (Some(frame), Some(-1)) if frame >= 0 && frame < self.frames.len() as i64 => frame as usize,
            _ => return Err(campaign_err!("ending: invalid recovered blit at {}", s.source)),
        };
        let Some(shared) = self.compositor.as_ref() else {
            return Err(campaign_err!("ending: nil compositor"));
        };
        let mut compositor = lock_compositor(shared);
        let compositor = &mut *compositor;
        let dst = match s.target.as_str() {
            "offscreen" => &mut compositor.offscreen,
            "vga" => &mut compositor.vga,
            other => return Err(campaign_err!("ending: unsupported blit target {}", battle::go_quote(other))),
        };
        Ok(self.frames[frame].blit(dst, s.stride, -1)?)
    }

    /// BlockedDialogue returns the exact branch selected by the native chapter
    /// comparison at an opaque text call.  The player remains blocked: presenting
    /// these lines never grants permission to execute following opaque segments.
    ///
    /// Go：`player.go:345`，`func (p *Player) BlockedDialogue(chapter int) ([]DialogueBlock, bool)`
    pub fn blocked_dialogue(&self, chapter: i64) -> (Vec<DialogueBlock>, bool) {
        let Some(blocked) =
            self.blocked.as_ref().filter(|b| self.state == PLAYBACK_BLOCKED && b.op == "native_text_branch_opaque")
        else {
            return (Vec::new(), false);
        };
        let blocks = if chapter == 26 { &blocked.then_dialogue } else { &blocked.else_dialogue };
        if blocks.is_empty() {
            return (Vec::new(), false);
        }
        (blocks.clone(), true)
    }

    /// ResumeBlockedDialogue advances exactly one already-presented native text
    /// gate.  It refuses every other opaque operation, so a UI cannot accidentally
    /// skip a not-yet-recovered renderer boundary.
    ///
    /// Go：`player.go:362`，`func (p *Player) ResumeBlockedDialogue() bool`
    pub fn resume_blocked_dialogue(&mut self) -> bool {
        if self.state != PLAYBACK_BLOCKED || self.blocked.as_ref().is_none_or(|b| b.op != "native_text_branch_opaque") {
            return false;
        }
        self.blocked = None;
        self.state = PLAYBACK_RUNNING;
        self.segment += 1;
        true
    }
}
