//! Go 來源：`remake/cmd/fd2/ending_preview.go`（864 行）。
//!
//! 原版索引色的戰役結局（0x2bce5 前綴、0x2c39b 文字、0x2c548 角色蒙太奇、0x2c194 尾段與 THE END 終局）。

use super::*;

/// nativeEndingPreview preserves the recovered 0x2bce5 prefix and also owns
/// the source-bound campaign ending path. It still grades call-time record and
/// global continuity, sound/input ownership and player-path E2 separately;
/// FD2_ENDING_PREFIX=1 remains a developer-only direct preview entry.
///
/// Go：`ending_preview.go:27`。`view`／`dialogue_view` 改存擁有的 `Picture`（Go 是可寫的 `*ebiten.Image`）。
#[derive(Debug, Clone, Default)]
pub(crate) struct NativeEndingPreview {
    pub(crate) player: Option<ending::Player>,
    pub(crate) view: Option<Picture>,
    pub(crate) last: GameInstant,
    pub(crate) remainder: GoDuration,
    pub(crate) chapter: i64,
    pub(crate) queued: bool,
    pub(crate) campaign_source_bound: bool,
    pub(crate) audio_cue_consumed: bool,
    pub(crate) montage: Option<ending::MontageCycle>,
    pub(crate) montage_wait: GoDuration,
    pub(crate) montage_input_pending: bool,
    pub(crate) montage_start_attempted: bool,
    pub(crate) montage_start_error: String,
    pub(crate) tail: Option<ending::MontageTailAssets>,
    pub(crate) tail_player: Option<ending::MontageTailPlayer>,
    pub(crate) tail_wait: GoDuration,
    pub(crate) tail_start_attempted: bool,
    pub(crate) tail_start_error: String,
    pub(crate) review_party_outcomes: bool,
    pub(crate) review_cycles: i64,
    pub(crate) dialogue: Option<ending::NativeDialoguePlayback>,
    pub(crate) dialogue_view: Option<Picture>,
    pub(crate) dialogue_mouth: dato::MouthState,
    pub(crate) dialogue_mouth_ready: bool,
}

/// Go：`ending_preview.go:697` 的 `approximateNativeMontageTick`（55 ms）。
const APPROXIMATE_NATIVE_MONTAGE_TICK: i64 = 55 * GoDuration::MILLISECOND.0;

fn ending_image() -> Picture {
    Picture::new_image(ending::WIDTH, ending::HEIGHT)
}

/// Go：`ending_preview.go:70`，`func newNativeEndingPreviewForCampaign(prefix *campaign.NativeEndingPrefixConfig) (*nativeEndingPreview, error)`
pub(crate) fn new_native_ending_preview_for_campaign(
    prefix: &campaign::NativeEndingPrefixConfig,
) -> Result<NativeEndingPreview> {
    if !prefix.is_recovered_prefix_contract() {
        return Err(game_err!("ending: campaign native prefix is not a recovered 0x2bce5 contract"));
    }
    new_native_ending_preview_for_timeline(&prefix.timeline, prefix.chapter)
}

/// Go：`ending_preview.go:54`，`const nativeEndingTimelinePath`
pub(crate) const NATIVE_ENDING_TIMELINE_PATH: &str = "assets/endings/native_2bce5.json";

/// 開發者直接預覽（`FD2_ENDING_PREFIX`）：預設第 29 章，`FD2_ENDING_CHAPTER` 只接受已驗證的 26 或 29。
///
/// Go：`ending_preview.go:56`，`func newNativeEndingPreview() (*nativeEndingPreview, error)`
pub(crate) fn new_native_ending_preview() -> Result<NativeEndingPreview> {
    let mut chapter = 29;
    if let Ok(raw) = std::env::var("FD2_ENDING_CHAPTER")
        && !raw.is_empty()
    {
        match raw.parse::<i64>() {
            Ok(value @ (26 | 29)) => chapter = value,
            _ => return Err(game_err!("ending: FD2_ENDING_CHAPTER must be 26 or 29")),
        }
    }
    new_native_ending_preview_for_timeline(NATIVE_ENDING_TIMELINE_PATH, chapter)
}

/// Go：`ending_preview.go:77`，`func newNativeEndingPreviewForTimeline(timelinePath string, chapter int) (*nativeEndingPreview, error)`
pub(crate) fn new_native_ending_preview_for_timeline(timeline_path: &str, chapter: i64) -> Result<NativeEndingPreview> {
    if !matches!(chapter, 26 | 29 | 30) {
        return Err(game_err!("ending: native chapter must be 26, 29 or 30"));
    }
    let timeline = ending::load_timeline(&asset_path(timeline_path))?;
    if timeline.resource.archive != "FDOTHER.DAT" || timeline.resource.index != 54 {
        return Err(game_err!("ending: timeline frame source is not the recovered FDOTHER #54 contract"));
    }
    let frames =
        fdother::load_separated_ending_prefix_frames(separated_asset_path("animations/fdother_054_ending_prefix"))?;
    let clip = afm::load_separated_resource(separated_asset_path("animations"), 2)?;
    let compositor: ending::SharedCompositor =
        std::sync::Arc::new(std::sync::Mutex::new(ending::new_indexed_compositor()));
    let mut player = ending::new_player(timeline, &frames, Some(&clip), Some(compositor))?;
    let phase0 = ending::load_finale_phase(&asset_path("assets/endings/native_2c405.json"))?;
    let strings = fdtxt::load_separated_resource(separated_asset_path("text"), 31)?;
    let font = fdtxt::load_separated_font(separated_asset_path("fonts"))?;
    player.enable_recovered_phase0(
        &phase0,
        &ending::Phase0Assets { strings: Some(strings), font: Some(font), ..Default::default() },
    )?;
    Ok(NativeEndingPreview { player: Some(player), view: Some(ending_image()), chapter, ..Default::default() })
}

/// 只讀 0x2c548 消費的原始位元組（+6／+7／+8／+0x20），依 LOADCH 的部署順序取持續名冊。
///
/// Go：`ending_preview.go:249`，`func nativeEndingMontageRecords(order []int, roster map[int]battle.Unit) ([][]byte, []byte, error)`
pub(crate) fn native_ending_montage_records(
    order: &[i64],
    roster: &BTreeMap<i64, battle::Unit>,
) -> Result<(Vec<Vec<u8>>, Vec<u8>)> {
    if order.len() < 2 || roster.is_empty() {
        return Err(game_err!("ending: native montage needs at least two persistent party records"));
    }
    let (mut units, mut groups) = (Vec::with_capacity(order.len()), Vec::with_capacity(order.len()));
    for id in order {
        let unit = roster.get(id).filter(|u| {
            u.has_native_record_byte6
                && u.has_battle_fig
                && u.has_native_record_class
                && (0..=0xff).contains(&u.battle_fig)
        });
        let Some(unit) = unit else {
            return Err(game_err!("ending: persistent party record {id} lacks raw montage provenance"));
        };
        let identity = if unit.has_native_record_byte8 {
            unit.native_record_byte8
        } else {
            // 持續 JOIN 記錄以 +8 當 NativeIdentity；這是來源攜帶的後備，不由 Fig 或 BattleFig 推測。
            if !unit.has_native_identity || !(0..=0xff).contains(&unit.native_identity) {
                return Err(game_err!("ending: persistent party record {id} lacks raw +8 provenance"));
            }
            unit.native_identity as u8
        };
        let mut record = vec![0u8; 0x21];
        record[6] = unit.native_record_byte6;
        record[7] = unit.battle_fig as u8;
        record[8] = identity;
        record[0x20] = unit.native_record_class;
        groups.push(record[7]);
        units.push(record);
    }
    Ok((units, groups))
}

/// Go：`ending_preview.go:280`，`func (p *nativeEndingPreview) montageArchivePaths() (ending.MontageArchivePaths, error)`
fn montage_archive_paths() -> Result<ending::MontageArchivePaths> {
    let paths = ending::MontageArchivePaths {
        text_root: separated_asset_path("text"),
        surface_root: separated_asset_path("surfaces"),
        item_panel_root: separated_asset_path("ui"),
        animation_root: separated_asset_path("animations"),
        portrait_root: separated_asset_path("portraits"),
        font_root: separated_asset_path("fonts"),
    };
    if paths.surface_root.is_empty() {
        return Err(game_err!("ending: separated montage surface root is unavailable"));
    }
    Ok(paths)
}

impl NativeEndingPreview {
    fn compositor(&self) -> Option<ending::SharedCompositor> {
        self.player.as_ref().and_then(|player| player.compositor.clone())
    }

    /// Go：`ending_preview.go:140`，`func (p *nativeEndingPreview) atNativeMontageGate() bool`
    pub(crate) fn at_native_montage_gate(&self) -> bool {
        self.campaign_source_bound
            && !self.queued
            && self.player.as_ref().is_some_and(|player| {
                player.state == ending::PLAYBACK_BLOCKED
                    && player
                        .blocked
                        .as_ref()
                        .is_some_and(|b| b.op == "native_finale_montage_opaque" && b.source == "0x2c548")
            })
    }

    /// Go：`ending_preview.go:147`，`func (p *nativeEndingPreview) runningCampaignMontage() bool`
    pub(crate) fn running_campaign_montage(&self) -> bool {
        self.at_native_montage_gate()
            && !self.review_party_outcomes
            && self.montage.as_ref().is_some_and(|m| !m.ready())
    }

    /// Go：`ending_preview.go:151`，`func (p *nativeEndingPreview) runningCampaignTail() bool`
    pub(crate) fn running_campaign_tail(&self) -> bool {
        self.at_native_montage_gate()
            && !self.review_party_outcomes
            && self.tail.is_some()
            && self.tail_player.as_ref().is_some_and(|t| !t.ready())
    }

    /// 最終 FDOTHER#59 呈現後的穩定終局畫面（來源約束 E1）。
    ///
    /// Go：`ending_preview.go:159`，`func (p *nativeEndingPreview) presentingCampaignTerminal() bool`
    pub(crate) fn presenting_campaign_terminal(&self) -> bool {
        self.at_native_montage_gate()
            && self.tail.is_some()
            && self.tail_player.as_ref().is_some_and(|t| t.ready())
            && !self.review_party_outcomes
    }

    /// 重製端選用的回顧迴圈；不宣稱原版終局自迴圈接受現代回顧指令。
    ///
    /// Go：`ending_preview.go:168`，`func (p *nativeEndingPreview) reviewingCampaignPartyOutcomes() bool`
    pub(crate) fn reviewing_campaign_party_outcomes(&self) -> bool {
        self.at_native_montage_gate()
            && self.tail.is_some()
            && self.tail_player.as_ref().is_some_and(|t| t.ready())
            && self.review_party_outcomes
    }

    /// Go：`ending_preview.go:195`，`func (p *nativeEndingPreview) awaitingCampaignFallback() bool`
    pub(crate) fn awaiting_campaign_fallback(&self) -> bool {
        self.at_native_montage_gate()
            && self.montage_start_attempted
            && match self.montage.as_ref() {
                None => true,
                Some(m) => {
                    m.ready() && self.tail_start_attempted && (self.tail.is_none() || self.tail_player.is_none())
                }
            }
    }

    /// Go：`ending_preview.go:442`，`func (p *nativeEndingPreview) resetCampaignPartyReviewCycle() error`
    fn reset_campaign_party_review_cycle(&mut self) -> Result<()> {
        let compositor = self.compositor();
        let (Some(compositor), Some(tail), Some(template)) =
            (compositor, self.tail.as_ref(), self.montage.as_ref().filter(|m| m.ready()))
        else {
            return Err(game_err!("ending: party outcome review requires a completed terminal montage"));
        };
        let mut groups = Vec::with_capacity(template.units.len());
        for (index, unit) in template.units.iter().enumerate() {
            let Some(&group) = unit.get(7) else {
                return Err(game_err!("ending: party outcome review unit {index} lacks raw +7 provenance"));
            };
            groups.push(group);
        }
        // 完整一輪後調色盤已淡出；原版尾段在最終畫面前還原保存的調色盤，這個非原版的重播沿用同一基準。
        tail.present_final(Some(&mut compositor.lock().unwrap()))?;
        let cycle = ending::new_montage_cycle(
            &template.montage,
            template.assets.clone(),
            &template.units,
            &groups,
            Some(compositor),
        )
        .map_err(|err| game_err!("ending: party outcome review: {err}"))?;
        self.montage = Some(cycle);
        self.montage_wait = GoDuration::default();
        self.montage_input_pending = false;
        self.tail_wait = GoDuration::default();
        self.last = GameInstant::default();
        Ok(())
    }

    /// Go：`ending_preview.go:697`，`func (p *nativeEndingPreview) advanceMontage(elapsed time.Duration, nativeRNG *uint16) error`
    fn advance_montage(&mut self, elapsed: GoDuration, native_rng: &mut u16) -> Result<()> {
        let Some(montage) = self.montage.as_mut() else {
            return Err(game_err!("ending: montage runtime is incomplete"));
        };
        let mut remaining = elapsed.0;
        for _ in 0..1024 {
            if montage.ready() {
                return Ok(());
            }
            if self.montage_wait.0 > 0 {
                if remaining < self.montage_wait.0 {
                    self.montage_wait.0 -= remaining;
                    return Ok(());
                }
                remaining -= self.montage_wait.0;
                self.montage_wait = GoDuration::default();
            }
            if self.montage_input_pending && montage.observe_input_change() {
                self.montage_input_pending = false;
            }
            let mut random_byte = 0u8;
            if montage.needs_random_byte() {
                *native_rng = fd2_formats::native_rng::native_rng_step(*native_rng);
                random_byte = *native_rng as u8;
            }
            montage.step(random_byte)?;
            if montage.ready() {
                return Ok(());
            }
            self.montage_wait = GoDuration(
                montage.delay_ticks * APPROXIMATE_NATIVE_MONTAGE_TICK + montage.delay_ms * GoDuration::MILLISECOND.0,
            );
            if remaining == 0 && self.montage_wait.0 > 0 {
                return Ok(());
            }
        }
        Err(game_err!("ending: montage advance exceeded bounded step budget"))
    }

    /// Go：`ending_preview.go:736`，`func (p *nativeEndingPreview) advanceTail(elapsed time.Duration) error`
    fn advance_tail(&mut self, elapsed: GoDuration) -> Result<()> {
        let Some(tail) = self.tail_player.as_mut() else {
            return Err(game_err!("ending: montage tail runtime is incomplete"));
        };
        let mut remaining = elapsed.0;
        for _ in 0..4096 {
            if tail.ready() {
                return Ok(());
            }
            if self.tail_wait.0 > 0 {
                if remaining < self.tail_wait.0 {
                    self.tail_wait.0 -= remaining;
                    return Ok(());
                }
                remaining -= self.tail_wait.0;
                self.tail_wait = GoDuration::default();
            }
            tail.step()?;
            self.tail_wait = GoDuration(tail.delay_ticks * APPROXIMATE_NATIVE_MONTAGE_TICK);
            if remaining == 0 && self.tail_wait.0 > 0 {
                return Ok(());
            }
        }
        Err(game_err!("ending: montage tail advance exceeded bounded step budget"))
    }

    /// Go：`ending_preview.go:764`，`func (p *nativeEndingPreview) advance(now time.Time, nativeRNG *uint16) error`
    pub(crate) fn advance(&mut self, now: GameInstant, native_rng: &mut u16) -> Result<()> {
        let mut elapsed = 0i64;
        if !self.last.is_zero() {
            self.remainder.0 += (now - self.last).0;
            elapsed = self.remainder.0 / GoDuration::MILLISECOND.0;
            self.remainder.0 -= elapsed * GoDuration::MILLISECOND.0;
        }
        self.last = now;
        let elapsed_duration = GoDuration(elapsed * GoDuration::MILLISECOND.0);
        if self.reviewing_campaign_party_outcomes() {
            let Some(montage) = self.montage.as_ref() else {
                return Err(game_err!("ending: party outcome review has no cycle"));
            };
            if !montage.ready() {
                self.advance_montage(elapsed_duration, native_rng)?;
            }
            if self.montage.as_ref().is_some_and(|m| m.ready()) {
                self.reset_campaign_party_review_cycle()?;
                self.review_cycles += 1;
            }
        } else if self.running_campaign_tail() {
            self.advance_tail(elapsed_duration)?;
        } else if self.tail.is_some() {
            // 原版 caller 在 0x2bce5 返回後自迴圈；保持終局畫面，不前進戰役節點或模擬 DOS 鍵盤。
        } else if self.montage.as_ref().is_some_and(|m| !m.ready()) {
            self.advance_montage(elapsed_duration, native_rng)?;
        } else {
            self.player.as_mut().unwrap().advance(elapsed)?;
        }
        self.publish_view();
        Ok(())
    }

    fn publish_view(&mut self) {
        let Some(compositor) = self.compositor() else { return };
        let rgba = compositor.lock().unwrap().rgba();
        if let Some(view) = self.view.as_mut() {
            view.write_pixels(&rgba.pix);
        }
    }
}

impl Game {
    /// 只啟動已驗證戰役節點所選的前綴；後續 renderer 仍以來源約束 E1 標示。
    ///
    /// Go：`ending_preview.go:127`，`func (g *Game) startCampaignNativeEnding(prefix *campaign.NativeEndingPrefixConfig) error`
    pub(crate) fn start_campaign_native_ending(&mut self, prefix: &campaign::NativeEndingPrefixConfig) -> Result<()> {
        let mut preview = new_native_ending_preview_for_campaign(prefix)?;
        preview.campaign_source_bound = true;
        self.native_ending = Some(preview);
        Ok(())
    }

    /// 正式 Update 與回歸共用的終局輸入 owner；`any_key` 只保存 0x2c950 已證實的 raw-change 條件。
    ///
    /// Go：`ending_preview.go:175`，`func (g *Game) applyNativeEndingInput(confirm, escape, anyKey bool) error`
    pub(crate) fn apply_native_ending_input(&mut self, confirm: bool, escape: bool, any_key: bool) -> Result<()> {
        let Some(p) = self.native_ending.as_mut() else { return Ok(()) };
        if p.reviewing_campaign_party_outcomes() && (confirm || escape) {
            return self.return_campaign_terminal_from_review();
        }
        if p.presenting_campaign_terminal() && confirm {
            return self.start_campaign_party_outcome_review();
        }
        if !p.reviewing_campaign_party_outcomes() && p.montage.as_ref().is_some_and(|m| !m.ready()) && any_key {
            p.montage_input_pending = true;
        }
        Ok(())
    }

    /// 只消費 after_gate 與目前已還原 0x2c548 邊界相符的唯一音訊 cue。
    ///
    /// Go：`ending_preview.go:203`，`func (g *Game) consumeNativeEndingAudioAtGate()`
    pub(crate) fn consume_native_ending_audio_at_gate(&mut self) {
        let Some(p) = self.native_ending.as_mut().filter(|p| p.at_native_montage_gate() && !p.audio_cue_consumed)
        else {
            return;
        };
        let (cue, ok) = p.player.as_ref().unwrap().audio_cue_at_blocked_boundary();
        if !ok {
            return;
        }
        p.audio_cue_consumed = true;
        self.emit_native_ending_audio_cue(&cue);
    }

    /// Go：`ending_preview.go:216`，`func (g *Game) emitNativeEndingAudioCue(cue ending.AudioCue)`
    pub(crate) fn emit_native_ending_audio_cue(&mut self, cue: &ending::AudioCue) {
        if cue.track < 0 {
            self.stop_bgm();
            return;
        }
        self.play_bgm_count(&format!("FDMUS_{:03}", cue.track), cue.driver_arg);
    }

    /// 只在來源約束的原始資產預檢失敗時返回可編輯結語。
    ///
    /// Go：`ending_preview.go:227`，`func (g *Game) finishCampaignNativeEndingFallback() bool`
    pub(crate) fn finish_campaign_native_ending_fallback(&mut self) -> bool {
        let Some(p) = self.native_ending.as_ref().filter(|p| p.awaiting_campaign_fallback()) else {
            return false;
        };
        let key = if p.montage.as_ref().is_some_and(|m| m.ready()) {
            "ending.assets_missing_montage"
        } else {
            "ending.assets_missing"
        };
        let (notice, ok) = self.locale_message(key, &[]);
        if !ok {
            return false;
        }
        self.stop_bgm();
        self.native_ending = None;
        self.ending_notice = notice;
        true
    }

    /// 只在來源約束 E1 的終局邊界啟動；以 1088D(30) 已建構的 runtime record 為載體。
    ///
    /// Go：`ending_preview.go:301`，`func (g *Game) startCampaignNativeMontage() error`
    pub(crate) fn start_campaign_native_montage(&mut self) -> Result<()> {
        if !self.native_ending.as_ref().is_some_and(|p| p.at_native_montage_gate()) {
            return Err(game_err!("ending: native montage requires the source-bound campaign gate"));
        }
        if self.native_ending.as_ref().unwrap().montage_start_attempted {
            return Ok(());
        }
        self.native_ending.as_mut().unwrap().montage_start_attempted = true;
        let result = self.build_campaign_native_montage();
        let p = self.native_ending.as_mut().unwrap();
        match result {
            Ok(cycle) => {
                p.montage = Some(cycle);
                Ok(())
            }
            Err(err) => {
                p.montage_start_error = err.to_string();
                Err(err)
            }
        }
    }

    fn build_campaign_native_montage(&self) -> Result<ending::MontageCycle> {
        let p = self.native_ending.as_ref().unwrap();
        let (order, roster) = if p.chapter == 30 {
            // 2C548 讀 1088D(30) 已建構的當前 record；舊預覽保留原有持續名冊來源。
            if self.story_actors.len() != 31 {
                return Err(game_err!("ending: selector30 runtime capacity is not 31"));
            }
            let (mut order, mut roster) = (Vec::new(), BTreeMap::new());
            for (slot, unit) in self.story_actors.iter().enumerate() {
                if !unit.has_native_record_byte5 || !unit.has_native_record_byte6 || !unit.has_native_record_byte8 {
                    return Err(game_err!("ending: selector30 runtime record lacks raw source"));
                }
                if unit.native_record_byte5 & 1 != 0 {
                    continue;
                }
                if order.len() >= self.party_join_order.len()
                    || i64::from(unit.native_record_byte8) != self.party_join_order[order.len()]
                {
                    return Err(game_err!("ending: selector30 runtime order differs from JOIN"));
                }
                order.push(slot as i64);
                roster.insert(slot as i64, unit.clone());
            }
            if order.len() != self.party_join_order.len() {
                return Err(game_err!("ending: selector30 active count differs from JOIN"));
            }
            (order, roster)
        } else {
            (self.party_join_order.clone(), self.party_roster.clone())
        };
        let (units, groups) = native_ending_montage_records(&order, &roster)?;
        let montage = ending::load_montage(&asset_path("assets/endings/native_2c548.json"))?;
        let paths = montage_archive_paths()?;
        let assets = ending::load_montage_cycle_assets(&montage, &paths, &units)?;
        Ok(ending::new_montage_cycle(&montage, assets, &units, &groups, p.compositor())?)
    }

    /// 已證實 20 筆 0x2c194 資源排程的來源約束 E1 視覺橋接；資產完整後才依序發布 stop／start 兩筆 cue。
    ///
    /// Go：`ending_preview.go:373`，`func (g *Game) startCampaignNativeTail() error`
    pub(crate) fn start_campaign_native_tail(&mut self) -> Result<()> {
        let ready = self
            .native_ending
            .as_ref()
            .is_some_and(|p| p.at_native_montage_gate() && p.montage.as_ref().is_some_and(|m| m.ready()));
        if !ready {
            return Err(game_err!("ending: terminal tail requires a completed source-bound montage"));
        }
        let p = self.native_ending.as_mut().unwrap();
        if p.tail_start_attempted {
            if p.tail.is_some() && p.tail_player.is_some() {
                return Ok(());
            }
            return Err(game_err!("ending: terminal tail admission already failed"));
        }
        p.tail_start_attempted = true;
        let built = (|| -> Result<(ending::AudioCue, ending::AudioCue, ending::MontageTailAssets, ending::MontageTailPlayer)> {
            let player = p.player.as_ref().unwrap();
            let (stop, ok) = player.audio_cue_for_runtime_stage("tail_stop");
            if !ok || stop.source != "0x2c1ac" || stop.track != -1 || stop.driver_arg != 1 {
                return Err(game_err!("ending: verified tail stop cue is unavailable"));
            }
            let (start, ok) = player.audio_cue_for_runtime_stage("tail_start");
            if !ok || start.source != "0x2c1f5" || start.track != 18 || start.driver_arg != 0 {
                return Err(game_err!("ending: verified tail start cue is unavailable"));
            }
            let tail = ending::load_montage_tail(&asset_path("assets/endings/native_2c194_tail.json"))?;
            let assets = ending::load_montage_tail_assets(
                &tail,
                &ending::MontageTailAssetPaths {
                    surface_root: separated_asset_path("surfaces"),
                    palette_root: separated_asset_path("palette"),
                    animation_root: separated_asset_path("animations"),
                },
            )?;
            let paths = montage_archive_paths()?;
            let sets = ending::load_montage_tail_visual_sets(
                &tail,
                &ending::MontageTailVisualPaths {
                    surface_root: separated_asset_path("surfaces"),
                    animation_root: paths.animation_root,
                },
            )?;
            let tail_player = ending::new_montage_tail_player(&tail, &assets, &sets, p.compositor())?;
            Ok((stop, start, assets, tail_player))
        })();
        match built {
            Ok((stop, start, assets, tail_player)) => {
                (p.tail, p.tail_player) = (Some(assets), Some(tail_player));
                p.tail_wait = GoDuration::default();
                p.last = GameInstant::default();
                self.emit_native_ending_audio_cue(&stop);
                self.emit_native_ending_audio_cue(&start);
                Ok(())
            }
            Err(err) => {
                p.tail_start_error = err.to_string();
                Err(err)
            }
        }
    }

    /// 從終局畫面開始重播全隊結局（重製端選用功能，不是 DOS 輸入重建）。
    ///
    /// Go：`ending_preview.go:476`，`func (g *Game) startCampaignPartyOutcomeReview() error`
    pub(crate) fn start_campaign_party_outcome_review(&mut self) -> Result<()> {
        let Some(p) = self.native_ending.as_mut().filter(|p| p.presenting_campaign_terminal()) else {
            return Err(game_err!("ending: party outcome review requires the terminal image"));
        };
        p.reset_campaign_party_review_cycle()?;
        p.review_party_outcomes = true;
        p.review_cycles += 1;
        Ok(())
    }

    /// 離開選用回顧並還原終局畫面；不前進戰役或模擬原版鍵盤自迴圈。
    ///
    /// Go：`ending_preview.go:492`，`func (g *Game) returnCampaignTerminalFromReview() error`
    pub(crate) fn return_campaign_terminal_from_review(&mut self) -> Result<()> {
        let Some(p) = self.native_ending.as_mut().filter(|p| p.reviewing_campaign_party_outcomes()) else {
            return Err(game_err!("ending: party outcome review is not active"));
        };
        let compositor = p.compositor().unwrap();
        p.tail.as_ref().unwrap().present_final(Some(&mut compositor.lock().unwrap()))?;
        p.review_party_outcomes = false;
        p.montage_wait = GoDuration::default();
        p.montage_input_pending = false;
        p.tail_wait = GoDuration::default();
        p.last = GameInstant::default();
        p.publish_view();
        Ok(())
    }

    /// Go：`ending_preview.go:511`，`func (g *Game) queueNativeEndingDialogue() error`
    pub(crate) fn queue_native_ending_dialogue(&mut self) -> Result<()> {
        let Some(p) = self.native_ending.as_ref().filter(|p| !p.queued) else { return Ok(()) };
        let (blocks, ok) = p.player.as_ref().unwrap().blocked_dialogue(p.chapter);
        if !ok {
            return Ok(());
        }
        let playback = self.prepare_native_ending_dialogue(&blocks)?;
        let p = self.native_ending.as_mut().unwrap();
        p.dialogue = Some(playback);
        p.dialogue_view = Some(ending_image());
        p.dialogue_mouth = dato::MouthState::default();
        p.dialogue_mouth_ready = false;
        p.queued = true;
        self.sync_native_ending_dialogue_view()
    }

    /// Go：`ending_preview.go:532`，`func (g *Game) prepareNativeEndingDialogue(blocks []ending.DialogueBlock) (*ending.NativeDialoguePlayback, error)`
    fn prepare_native_ending_dialogue(
        &self,
        blocks: &[ending::DialogueBlock],
    ) -> Result<ending::NativeDialoguePlayback> {
        let compositor = self.native_ending.as_ref().and_then(NativeEndingPreview::compositor);
        let background = compositor
            .map(|c| c.lock().unwrap().vga.clone())
            .filter(|vga| vga.len() as i64 == ending::BYTES)
            .ok_or_else(|| game_err!("ending: indexed dialogue source is unavailable"))?;
        let (font, glyphs) = load_native_battle_name_assets()?;
        let entries = fdother::load_separated_item_panel_entries(separated_asset_path("ui"))?;
        let mut cells = Vec::with_capacity(20);
        for index in 0..20 {
            let cell = entries
                .raw
                .get(&index)
                .ok_or_else(|| game_err!("ending: separated FDOTHER #5 dialogue cell {index} is unavailable"))?;
            cells.push(cell.clone());
        }
        let special = |id: i64| (0x80..=0x84).contains(&id);
        let mut prepared = Vec::with_capacity(blocks.len());
        for (block_index, block) in blocks.iter().enumerate() {
            if special(block.portrait_id) {
                return Err(game_err!(
                    "ending: block {block_index} portrait {} requires an unimplemented special anchor",
                    block.portrait_id
                ));
            }
            let initial =
                load_native_separated_portrait(block.portrait_id).ok().filter(|frames| frames.len() >= 4).ok_or_else(
                    || game_err!("ending: block {block_index} initial portrait {} is unavailable", block.portrait_id),
                )?;
            let initial_base = ending::compose_native_dialogue_base(&background, &cells, &initial[0])?;
            let opening = ending::compose_native_dialogue_opening_frames(&background, &initial_base)?;
            let closing = ending::compose_native_dialogue_closing_frames(&background, &initial_base)?;
            let mut utterances = Vec::with_capacity(block.native_utterances.len());
            for (utterance_index, native) in block.native_utterances.iter().enumerate() {
                if special(native.operand) {
                    return Err(game_err!(
                        "ending: block {block_index} utterance {utterance_index} portrait {} requires an unimplemented special anchor",
                        native.operand
                    ));
                }
                let portraits = load_native_separated_portrait(native.operand)
                    .ok()
                    .filter(|frames| frames.len() >= 4)
                    .ok_or_else(|| {
                        game_err!(
                            "ending: block {block_index} utterance {utterance_index} portrait {} is unavailable",
                            native.operand
                        )
                    })?;
                let base = ending::compose_native_dialogue_base(&background, &cells, &portraits[0])?;
                let mut frames_by_page = Vec::with_capacity(native.pages.len());
                let mut mouth_open = Vec::with_capacity(native.pages.len());
                for page in 0..native.pages.len() {
                    let frames = ending::compose_native_dialogue_progressive_frames(
                        &base,
                        font.as_ref(),
                        Some(&glyphs),
                        &native.pages,
                        page as i64,
                    )?;
                    let last = frames.last().ok_or_else(|| game_err!("ending: empty dialogue page"))?;
                    mouth_open.push(ending::compose_native_dialogue_mouth_frame(last, &portraits[3])?);
                    frames_by_page.push(frames);
                }
                utterances.push(ending::NativeDialogueUtteranceFrames { pages: frames_by_page, mouth_open });
            }
            prepared.push(ending::NativeDialogueBlockFrames { opening, utterances, closing });
        }
        Ok(ending::new_native_dialogue_playback(&prepared)?)
    }

    /// Go：`ending_preview.go:607`，`func (g *Game) syncNativeEndingDialogueView() error`
    pub(crate) fn sync_native_ending_dialogue_view(&mut self) -> Result<()> {
        let Some(p) = self.native_ending.as_mut() else { return Ok(()) };
        let (Some(dialogue), true) = (p.dialogue.as_ref(), p.dialogue_view.is_some()) else { return Ok(()) };
        let frame = dialogue.current_frame(p.dialogue_mouth.open);
        let rgba = p.compositor().unwrap().lock().unwrap().rgba_frame(&frame)?;
        p.dialogue_view.as_mut().unwrap().write_pixels(&rgba.pix);
        Ok(())
    }

    /// Go：`ending_preview.go:621`，`func (g *Game) stepNativeEndingDialogue(confirm bool) error`
    pub(crate) fn step_native_ending_dialogue(&mut self, confirm: bool) -> Result<()> {
        let Some(p) = self.native_ending.as_mut() else { return Ok(()) };
        let Some(dialogue) = p.dialogue.as_mut() else { return Ok(()) };
        let was_waiting = dialogue.waiting();
        if confirm {
            dialogue.confirm();
        } else {
            dialogue.step();
        }
        if dialogue.done() {
            (p.dialogue, p.dialogue_view) = (None, None);
            (p.dialogue_mouth, p.dialogue_mouth_ready) = (dato::MouthState::default(), false);
            self.resume_native_ending_dialogue();
            return Ok(());
        }
        let waiting = dialogue.waiting();
        let (mouth_ready, mouth_open) = (p.dialogue_mouth_ready, p.dialogue_mouth.open);
        if !waiting {
            (p.dialogue_mouth, p.dialogue_mouth_ready) = (dato::MouthState::default(), false);
        } else if !was_waiting || !mouth_ready {
            let countdown = self.story_mouth_intn30() + 2;
            let p = self.native_ending.as_mut().unwrap();
            p.dialogue_mouth = dato::MouthState { countdown, ..Default::default() };
            p.dialogue_mouth_ready = true;
        } else if self.frame % 2 == 0 {
            let random_mod30 = if mouth_open { self.story_mouth_intn30() } else { 0 };
            let p = self.native_ending.as_mut().unwrap();
            p.dialogue_mouth = p.dialogue_mouth.tick(random_mod30)?;
        }
        self.sync_native_ending_dialogue_view()
    }

    /// 只釋放所有排隊句子都已確認的那個文字閘門；重設 queued 是因為同一時間軸稍後還有第二次 0x2c39b。
    ///
    /// Go：`ending_preview.go:661`，`func (g *Game) resumeNativeEndingDialogue() bool`
    pub(crate) fn resume_native_ending_dialogue(&mut self) -> bool {
        let Some(p) = self.native_ending.as_mut().filter(|p| p.queued) else { return false };
        if !p.player.as_mut().unwrap().resume_blocked_dialogue() {
            return false;
        }
        p.queued = false;
        (p.dialogue, p.dialogue_view) = (None, None);
        (p.dialogue_mouth, p.dialogue_mouth_ready) = (dato::MouthState::default(), false);
        true
    }

    /// 正式 Update 的結局分支（Go `main.go` Update 的 `g.nativeEnding != nil` 區塊）。
    pub(crate) fn update_native_ending(&mut self, confirm: bool, escape: bool, any_key: bool) -> Result<()> {
        if let Err(err) = self.apply_native_ending_input(confirm, escape, any_key) {
            self.load_err = format!("native ending input: {err}");
            return Err(err);
        }
        let now = match self.native_ending_now.take() {
            Some(mut clock) => {
                let now = clock(self);
                self.native_ending_now = Some(clock);
                now
            }
            None => self.game_now(),
        };
        let mut rng = self.native_rng_state;
        let advanced = self.native_ending.as_mut().unwrap().advance(now, &mut rng);
        self.native_rng_state = rng;
        if let Err(err) = advanced {
            self.load_err = format!("native ending: {err}");
            return Err(err);
        }
        let p = self.native_ending.as_ref().unwrap();
        if p.at_native_montage_gate() && p.montage.is_none() && !p.montage_start_attempted {
            // 失敗時保留明示近似的後備，忠實模式不代換猜測的原版 renderer。
            let _ = self.start_campaign_native_montage();
        }
        let p = self.native_ending.as_ref().unwrap();
        if p.at_native_montage_gate() && p.montage.as_ref().is_some_and(|m| m.ready()) && !p.tail_start_attempted {
            let _ = self.start_campaign_native_tail();
        }
        let dialogue_was_queued = self.native_ending.as_ref().unwrap().dialogue.is_some();
        if let Err(err) = self.queue_native_ending_dialogue() {
            self.load_err = format!("native ending dialogue: {err}");
            return Err(err);
        }
        self.consume_native_ending_audio_at_gate();
        if dialogue_was_queued && let Err(err) = self.step_native_ending_dialogue(confirm) {
            self.load_err = format!("native ending dialogue: {err}");
            return Err(err);
        }
        if confirm && self.finish_campaign_native_ending_fallback() {
            return Ok(());
        }
        Ok(())
    }

    /// 原版索引色結局畫面；除錯 HUD 與後備提示面板另期。`screen` 為 `None` 時只做無畫面重播。
    ///
    /// Go：`ending_preview.go:823`，`func (g *Game) drawNativeEndingPreview(screen *ebiten.Image)`
    pub(crate) fn draw_native_ending_preview(&mut self, screen: Option<&mut Picture>) {
        let Some(screen) = screen else { return };
        screen.fill(ColorRgba { r: 0, g: 0, b: 0, a: 0xff });
        let Some(p) = self.native_ending.as_ref() else { return };
        let view = p.dialogue_view.as_ref().or(p.view.as_ref());
        if let Some(view) = view {
            let mut options = DrawImageOptions::default();
            options.geo_m.scale(2.0, 2.0);
            screen.draw_image(view, &options);
        }
        if let Some((message, x)) = self.native_ending_evidence_overlay().filter(|_| self.font.is_some()) {
            self.draw_native_ending_panel(screen, message, x, 0.78);
        } else if self.native_ending.as_ref().is_some_and(|p| p.awaiting_campaign_fallback()) && self.font.is_some() {
            let p = self.native_ending.as_ref().unwrap();
            let key = if p.montage.as_ref().is_some_and(|montage| montage.ready()) {
                "ending.terminal_missing"
            } else if !p.montage_start_error.is_empty() {
                "ending.montage_requirements"
            } else {
                "ending.continue_editable"
            };
            let (message, ok) = self.locale_message(key, &[]);
            if !ok {
                return;
            }
            self.draw_native_ending_panel(screen, &message, 30.0, 0.82);
        }
        self.draw_capture_shot_point(screen);
    }

    /// 除錯模式下結局蒙太奇的證據提示；只在 `debug` 開啟時顯示。
    ///
    /// Go：`ending_preview.go:810`，`func (g *Game) nativeEndingEvidenceOverlay() (string, float64, bool)`
    pub(crate) fn native_ending_evidence_overlay(&self) -> Option<(&'static str, f64)> {
        let p = self.native_ending.as_ref().filter(|_| self.debug)?;
        if p.running_campaign_montage() {
            return Some(("角色蒙太奇（來源約束 E1）：任意按鍵會在本輪結束後略過中間角色。", 23.0));
        }
        if p.running_campaign_tail() {
            return Some(("終局蒙太奇（原版資源、來源約束 E1）：完成後將停留在 THE END。", 25.0));
        }
        None
    }

    /// `drawNativeEndingPreview` 兩個提示分支共用的底框與文字（`ending_preview.go:833`..`859`）。
    fn draw_native_ending_panel(&mut self, screen: &mut Picture, message: &str, x: f64, scale: f64) {
        let mut panel = Picture::new_image(LOGICAL_W - 32, 42);
        panel.fill(ColorRgba { r: 0x10, g: 0x1c, b: 0x40, a: 0xe8 });
        let mut options = DrawImageOptions::default();
        options.geo_m.translate(16.0, (LOGICAL_H - 56) as f64);
        screen.draw_image(&panel, &options);
        if let Some(font) = self.font.as_mut() {
            font.draw(
                screen,
                message,
                x,
                (LOGICAL_H - 44) as f64,
                scale,
                ColorRgba { r: 0xff, g: 0xe0, b: 0x90, a: 0xff },
            );
        }
    }
}
