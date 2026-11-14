//! Go 來源：`remake/internal/ending/finale.go`（223 行）。

use super::*;

/// FinalePhase is an editable, evidence-only transcription of a native finale
/// stage. Its recovered phase-0 adapter still blocks before the next native
/// montage, so it cannot make the finale generally playable.
///
/// Go：`finale.go:14`
#[derive(Debug, Clone, Default, PartialEq, Eq, serde::Serialize, serde::Deserialize)]
#[serde(default)]
pub struct FinalePhase {
    /// Go 標籤：`json:"schema_version"`
    pub schema_version: i64,
    /// Go 標籤：`json:"native_handler"`
    pub native_handler: String,
    /// Go 標籤：`json:"status"`
    pub status: String,
    /// Go 標籤：`json:"phase"`
    pub phase: FinalePhaseSpec,
    /// Go 標籤：`json:"gate"`
    pub gate: FinalePhaseGate,
}

/// Go：`finale.go:22`
#[derive(Debug, Clone, Default, PartialEq, Eq, serde::Serialize, serde::Deserialize)]
#[serde(default)]
pub struct FinalePhaseSpec {
    /// Go 標籤：`json:"source"`
    pub source: String,
    /// Go 標籤：`json:"source_dat"`
    pub source_dat: String,
    /// Go 標籤：`json:"string_index"`
    pub string_index: i64,
    /// Go 標籤：`json:"script"`
    pub script: String,
    /// Go 標籤：`json:"scene_index"`
    pub scene_index: i64,
    /// Go 標籤：`json:"line"`
    pub line: i64,
    /// Go 標籤：`json:"count"`
    pub count: i64,
    /// Go 標籤：`json:"staging_bytes"`
    pub staging_bytes: i64,
    /// Go 標籤：`json:"text_offset"`
    pub text_offset: i64,
    /// Go 標籤：`json:"stride"`
    pub stride: i64,
    /// Go 標籤：`json:"line_advance_rows"`
    pub line_advance_rows: i64,
    /// Go 標籤：`json:"viewport_rows"`
    pub viewport_rows: i64,
    /// Go 標籤：`json:"iterations"`
    pub iterations: i64,
    /// Go 標籤：`json:"delay_ms"`
    pub delay_ms: i64,
    /// Go 標籤：`json:"baseline_palette_initial_delta"`
    pub baseline_palette_initial_delta: i64,
    /// Go 標籤：`json:"fade_out_through_iteration"`
    pub fade_out_through_iteration: i64,
    /// Go 標籤：`json:"palette_step_cadence"`
    pub palette_step_cadence: i64,
}

/// Go：`finale.go:42`
#[derive(Debug, Clone, Default, PartialEq, Eq, serde::Serialize, serde::Deserialize)]
#[serde(default)]
pub struct FinalePhaseGate {
    /// Go 標籤：`json:"source"`
    pub source: String,
    /// Go 標籤：`json:"reason"`
    pub reason: String,
}

/// Go：`finale.go:47`，`func LoadFinalePhase(path string) (*FinalePhase, error)`
pub fn load_finale_phase(path: &str) -> Result<FinalePhase> {
    let raw = battle::read_file(path)?;
    let phase: FinalePhase = battle::decode_json(&raw)?;
    let p = &phase.phase;
    if phase.schema_version != 1
        || phase.native_handler != "0x2c405"
        || phase.status != "recovered_phase0_bridge_fail_closed"
        || p.source != "0x2c469"
        || p.source_dat != "FDTXT_031"
        || p.string_index != 44
        || p.script != "ch32.json"
        || p.scene_index != 0
        || p.line != 0
        || p.count != 1
        || p.staging_bytes != 0x36b00
        || p.text_offset != 0x12c30
        || p.stride != WIDTH
        || p.line_advance_rows != 25
        || p.viewport_rows != HEIGHT
        || p.iterations != 500
        || p.delay_ms != 1
        || p.baseline_palette_initial_delta != 40
        || p.fade_out_through_iteration != 300
        || p.palette_step_cadence != 5
        || phase.gate.source != "0x2c548"
        || phase.gate.reason.is_empty()
    {
        return Err(campaign_err!("ending finale phase {} is incomplete or unsupported", battle::go_quote(path)));
    }
    Ok(phase)
}

impl FinalePhase {
    /// Ready always remains false until the exact native text compositor and its
    /// staging source have an evidence-backed remake adapter.
    ///
    /// Go：`finale.go:69`，`func (p FinalePhase) Ready() bool`
    pub fn ready(&self) -> bool {
        false
    }

    /// ComposePhase0Text reproduces the fully evidenced glyph-only part of
    /// 0x2c469: physical FDTXT_031 string #44 is written continuously from
    /// staging+0x12c30 with 16-byte glyph advance and the caller's native colour
    /// arguments (stride=320, foreground=CD, shadow=4C, transparent background).
    /// FFFE is the exactly recovered line advance to destination + line*25*stride;
    /// every other FFxx word remains rejected so this does not claim to be the
    /// unrecovered general 0x15f84 control-code renderer.
    ///
    /// Go：`finale.go:78`，`func (p FinalePhase) ComposePhase0Text(staging, textResource, fontResource []byte) (int, error)`
    pub fn compose_phase0_text(&self, staging: &mut [u8], text_resource: &[u8], font_resource: &[u8]) -> Result<i64> {
        if self.ready() || staging.len() as i64 != self.phase.staging_bytes {
            return Err(campaign_err!("ending: invalid phase-0 staging surface"));
        }
        let strings = fdtxt::parse(text_resource)?;
        let font = fdtxt::parse_font(font_resource)?;
        self.compose_phase0_text_typed(staging, Some(&strings), Some(&font))
    }

    /// ComposePhase0TextTyped consumes the separated FDTXT/font contract. The raw
    /// wrapper above remains only for source-oracle tests.
    ///
    /// Go：`finale.go:95`，`func (p FinalePhase) ComposePhase0TextTyped(staging []byte, strings *fdtxt.Strings, font *fdtxt.Font) (int, error)`
    pub fn compose_phase0_text_typed(
        &self,
        staging: &mut [u8],
        strings: Option<&fdtxt::Strings>,
        font: Option<&fdtxt::Font>,
    ) -> Result<i64> {
        let (Some(strings), Some(font)) = (strings, font) else {
            return Err(campaign_err!("ending: invalid phase-0 typed text assets"));
        };
        if self.ready() || staging.len() as i64 != self.phase.staging_bytes {
            return Err(campaign_err!("ending: invalid phase-0 typed text assets"));
        }
        let words = strings.words(self.phase.string_index)?;
        let mut base = self.phase.text_offset;
        let mut line = 0;
        for word in words {
            if word == 0xfffe {
                line += 1;
                base = self.phase.text_offset + line * self.phase.line_advance_rows * self.phase.stride;
                continue;
            }
            if word >= fdtxt::CONTROL_MIN {
                return Err(campaign_err!(
                    "ending: phase-0 string has unsupported control {}",
                    battle::go_sharp_x(i64::from(word))
                ));
            }
            font.blit_native_glyph(
                staging,
                self.phase.stride,
                base,
                i64::from(word),
                fdtxt::NativeGlyphStyle { foreground: 0xcd, shadow: 0x4c, background: 0 },
            )?;
            base += fdtxt::GLYPH_WIDTH as i64;
        }
        Ok(base)
    }
}

/// Phase0Player is the exact 500-pass row-scroll portion after 0x2c469. It is
/// intentionally separate from the ending Player: completing this phase does
/// not authorise execution of the unrecovered 0x2c548 montage.
///
/// Go：`finale.go:125`
#[derive(Debug, Clone, Default)]
pub struct Phase0Player {
    pub(crate) phase: FinalePhase,
    pub(crate) staging: Vec<u8>,
    pub(crate) compositor: Option<SharedCompositor>,
    pub(crate) iteration: i64,
    pub(crate) palette_delta: i64,
    pub(crate) wait_ms: i64,
    pub(crate) done: bool,
}

/// Phase0Assets keeps every non-global input needed by the recovered phase.
/// Baseline must come from the preceding native presentation state; callers
/// cannot silently substitute a convenience palette.
///
/// Go：`finale.go:138`
#[derive(Debug, Clone)]
pub struct Phase0Assets {
    pub text_resource: Vec<u8>,
    pub font_resource: Vec<u8>,
    pub strings: Option<fdtxt::Strings>,
    pub font: Option<fdtxt::Font>,
    pub baseline: [u8; 768],
}

impl Default for Phase0Assets {
    fn default() -> Self {
        Self {
            text_resource: Default::default(),
            font_resource: Default::default(),
            strings: Default::default(),
            font: Default::default(),
            baseline: std::array::from_fn(|_| Default::default()),
        }
    }
}

/// Go：`finale.go:146`，`func NewPhase0PlayerFromAssets(phase FinalePhase, assets Phase0Assets, compositor *IndexedCompositor) (*Phase0Player, error)`
pub fn new_phase0_player_from_assets(
    phase: &FinalePhase,
    assets: &Phase0Assets,
    compositor: Option<SharedCompositor>,
) -> Result<Phase0Player> {
    let Some(compositor) = compositor else {
        return Err(campaign_err!("ending: nil phase-0 compositor"));
    };
    // Go `make([]byte, StagingBytes)`：負值會 panic；載入時已固定為 0x36b00。
    let mut staging = vec![0u8; phase.phase.staging_bytes.max(0) as usize];
    let parsed;
    let (strings, font) = match (assets.strings.as_ref(), assets.font.as_ref()) {
        (Some(strings), Some(font)) => (strings, font),
        _ => {
            parsed = (fdtxt::parse(&assets.text_resource)?, fdtxt::parse_font(&assets.font_resource)?);
            (&parsed.0, &parsed.1)
        }
    };
    phase.compose_phase0_text_typed(&mut staging, Some(strings), Some(font))?;
    lock_compositor(&compositor).baseline.copy_from_slice(&assets.baseline);
    new_phase0_player(phase, &staging, Some(compositor))
}

/// Go：`finale.go:170`，`func NewPhase0Player(phase FinalePhase, staging []byte, compositor *IndexedCompositor) (*Phase0Player, error)`
pub fn new_phase0_player(
    phase: &FinalePhase,
    staging: &[u8],
    compositor: Option<SharedCompositor>,
) -> Result<Phase0Player> {
    match compositor {
        Some(compositor) if !phase.ready() && staging.len() as i64 == phase.phase.staging_bytes => Ok(Phase0Player {
            phase: phase.clone(),
            staging: staging.to_vec(),
            compositor: Some(compositor),
            palette_delta: phase.phase.baseline_palette_initial_delta,
            ..Default::default()
        }),
        _ => Err(campaign_err!("ending: invalid phase-0 player")),
    }
}

impl Phase0Player {
    /// Advance consumes the native 1ms waits. The first iteration presents
    /// immediately, as the original falls through to 0x2c4b4 before its first
    /// 0x17aa9(1) call.
    ///
    /// Go：`finale.go:180`，`func (p *Phase0Player) Advance(elapsedMS int) (done bool, err error)`
    /// Go 具名回傳值依序為：done、err。
    pub fn advance(&mut self, elapsed_ms: i64) -> Result<bool> {
        let mut elapsed_ms = elapsed_ms;
        if elapsed_ms < 0 {
            return Err(campaign_err!("ending: negative phase-0 elapsed time"));
        }
        if self.done {
            return Ok(true);
        }
        let Some(shared) = self.compositor.clone() else {
            return Err(campaign_err!("ending: invalid phase-0 player"));
        };
        loop {
            if self.wait_ms > 0 {
                if elapsed_ms < self.wait_ms {
                    self.wait_ms -= elapsed_ms;
                    return Ok(false);
                }
                elapsed_ms -= self.wait_ms;
                self.wait_ms = 0;
                self.iteration += 1;
            }
            if self.iteration >= self.phase.phase.iterations {
                self.done = true;
                return Ok(true);
            }
            {
                let mut compositor = lock_compositor(&shared);
                compositor.set_baseline_delta(0, 255, self.palette_delta)?;
                let offset = (self.iteration * self.phase.phase.stride) as usize;
                // Go `p.staging[offset:]`：offset 超出長度會 panic；Rust 以空切片交給 CopyRect 拒絕。
                let source = self.staging.get(offset..).unwrap_or(&[]);
                copy_rect(
                    &mut compositor.vga,
                    WIDTH,
                    source,
                    self.phase.phase.stride,
                    WIDTH,
                    self.phase.phase.viewport_rows,
                    0,
                )?;
            }
            // 0x2c4f9: the fade-out branch is entered only for i<200; its
            // decrement follows the current frame's palette/copy work.
            let cadence = self.phase.phase.palette_step_cadence;
            if self.iteration < 200 && self.palette_delta > 0 && self.iteration % cadence == 0 {
                self.palette_delta -= 1;
            }
            // 0x2c51b: fade-in starts strictly after i=300, also after the
            // current frame has been presented.
            if self.iteration > self.phase.phase.fade_out_through_iteration && self.iteration % cadence == 0 {
                self.palette_delta += 1;
            }
            self.wait_ms = self.phase.phase.delay_ms;
            if elapsed_ms == 0 {
                return Ok(false);
            }
        }
    }
}

/// 取得共享合成器的鎖。Go 的合成器沒有鎖；同一執行緒內的持有者依序使用，鎖中毒只會出現在
/// 另一個持有者 panic 之後，這裡沿用其內容（與 Go 繼續使用同一塊記憶體相同）。
pub(crate) fn lock_compositor(shared: &SharedCompositor) -> std::sync::MutexGuard<'_, IndexedCompositor> {
    shared.lock().unwrap_or_else(std::sync::PoisonError::into_inner)
}

#[cfg(test)]
mod finale_test;
