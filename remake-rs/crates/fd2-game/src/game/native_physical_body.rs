//! Go 來源：`remake/cmd/fd2/native_physical_body.go`（526 行）。
//!
//! 骨架：Go 檔的 7 個函式中產生 6 個（階段 0 清單），其餘標 `r8`，見 `PORTING.json`。

use super::*;

/// nativePhysicalBodyResources 在方向、HP與RNG修改前建立。
/// 正式入口與證據：docs/data/ida/fd2_physical_background_selection_20261004.json。
///
/// Go：`native_physical_body.go:20`
#[derive(Debug, Clone, Default)]
pub(crate) struct NativePhysicalBodyResources {
    pub(crate) attack: Option<figani::Animation>,
    pub(crate) counter: Option<figani::Animation>,
    pub(crate) actor_idle: Option<figani::Animation>,
    pub(crate) target_idle: Option<figani::Animation>,
    /// 原版 54103／54107，反擊不交換。
    pub(crate) actor_bg: fdother::Frame,
    /// Go 與 `actorBG` 同列宣告，註解見該欄位。
    pub(crate) target_bg: fdother::Frame,
    pub(crate) layers: [fdother::Frame; 3],
    pub(crate) platform: fdother::Frame,
    /// 29164 返回後的共享 base。
    pub(crate) base: Vec<u8>,
    pub(crate) actor_side: u8,
    pub(crate) target_side: u8,
    /// R4 D 組：Go 的 `map[int][]byte` 由所有工作共用同一份；Rust 以 `Arc` 共用（簽名變更紀錄）。
    pub(crate) main_sounds: Arc<BTreeMap<i64, Vec<u8>>>,
    pub(crate) counter_sounds: Arc<BTreeMap<i64, Vec<u8>>>,
}

/// 一個工作項目可只有cue；只有present項目才等待正式Draw。
/// body項目共用不可變的panel base，不為每個idle tick建立GPU貼圖。
///
/// Go：`native_physical_body.go:32`
#[derive(Debug, Clone, Default)]
pub(crate) struct NativePhysicalBodyJob {
    pub(crate) hd_sources: Option<ModernHDBattleSources>,
    pub(crate) hd_phase: String,
    pub(crate) pixels: Vec<u8>,
    pub(crate) base: Vec<u8>,
    pub(crate) attack: figani::Frame,
    pub(crate) idle: figani::Frame,
    pub(crate) present: Option<battlepresent::NativePhysicalBodyPresent>,
    pub(crate) raw_side: u8,
    pub(crate) palette0: Option<ColorNrgba>,
    pub(crate) wait_millis: f64,
    pub(crate) cue: i64,
    /// R4 D 組：`Arc` 共用同一份音效庫（見 [`NativePhysicalBodyResources::main_sounds`]）。
    pub(crate) sounds: Arc<BTreeMap<i64, Vec<u8>>>,
    pub(crate) sound_bank: i64,
    pub(crate) hd_impact: bool,
}

/// Go：`native_physical_body.go:48`
#[derive(Debug, Clone, Default)]
pub(crate) struct NativePhysicalBodyPlayback {
    pub(crate) jobs: Vec<NativePhysicalBodyJob>,
    pub(crate) index: i64,
    pub(crate) drawn: bool,
    pub(crate) cued: bool,
    pub(crate) elapsed_millis: f64,
    pub(crate) hd_extra_millis: f64,
    pub(crate) image: Option<Arc<Picture>>,
    pub(crate) last_present: i64,
    /// 正常2909D／2909F的17AA9(6)，與descriptor等待分開。
    pub(crate) return_wait_millis: f64,
    pub(crate) return_ramp_started: bool,
}

impl Game {
    /// Go：`native_physical_body.go:60`，`func (g *Game) prepareNativePhysicalBodyResources(scene *nativePhysicalScene, actor, target *battle.Unit, attack *figani.Animation) error`
    pub(crate) fn prepare_native_physical_body_resources(
        &mut self,
        scene: &mut NativePhysicalScene,
        actor: UnitIdx,
        target: UnitIdx,
        attack: &figani::Animation,
    ) -> Result<()> {
        let st = self.st.as_ref().expect("Go：g.st 為 nil 時解參考 panic");
        let (actor_unit, target_unit) = (&st.units[actor], &st.units[target]);
        let mut r = NativePhysicalBodyResources {
            attack: Some(attack.clone()),
            actor_side: actor_unit.native_record_byte6,
            target_side: target_unit.native_record_byte6,
            actor_bg: scene.background_frame.clone(),
            ..Default::default()
        };
        let (actor_fig, target_fig) = (actor_unit.battle_fig, target_unit.battle_fig);
        let counter_eligible = st.preview_native_physical_counterattack(actor, target, self.native_rng_state)?;
        let animations = separated_asset_path("animations");
        r.actor_idle = Some(figani::load_separated_resource(&animations, actor_fig * 3)?);
        r.target_idle = Some(figani::load_separated_resource(&animations, target_fig * 3)?);
        // 主攻 FIGANI byte1=0 時原版 28A6C 不重載 BG0..2，非零旗標反擊經 29CC2 照讀 `dword_5410B/F/13`
        // （#321）。讀到的是最後一次寫入後 free 掉的指標，開機後從未寫過則是 BSS 的 0。寫入端只載入
        // `BG.DAT` 0／1／2，區塊未被覆用時畫面與一般反擊相同；使用者 2026-11-27 定案重製端取這個
        // 可見結果，照常載入三層，不重現覆用或空指標造成的損壞與中止。
        let mut counter_reads_stale_layers = false;
        if counter_eligible {
            let counter = figani::load_separated_resource(&animations, target_fig * 3 + 1)?;
            counter_reads_stale_layers = attack.header_byte1 == 0 && counter.header_byte1 != 0;
            r.counter = Some(counter);
        }
        r.platform = fdother::load_separated_single_frame(
            separated_asset_path("surfaces"),
            "TAI.DAT",
            scene.selection.tai as i64,
        )?;
        r.target_bg = scene.background_frame.clone();
        if scene.selection.has_separate_backgrounds {
            r.target_bg = scene.target_background.clone();
        }
        if scene.selection.has_separate_backgrounds || counter_reads_stale_layers {
            for (i, layer) in r.layers.iter_mut().enumerate() {
                *layer = fdother::load_separated_single_frame(separated_asset_path("surfaces"), "BG.DAT", i as i64)?;
            }
        }
        let (actor_record, target_record) = (scene.actor_record.clone(), scene.target_record.clone());
        r.base = self.native_physical_base(scene, &actor_record, &target_record)?;
        // 29164: 29258..29271在raw side非零時無條件保存TAI；
        // 2927E..2929E在raw side零且mode零時保存TAI。
        if r.actor_side != 0 || attack.header_byte1 == 0 {
            let mut tai = r.platform.clone();
            (tai.x, tai.y) = (164, 157);
            tai.blit(&mut r.base, 320, -1)?;
        }
        for (first, second) in
            [(r.attack.as_ref(), r.target_idle.as_ref()), (r.counter.as_ref(), r.actor_idle.as_ref())]
        {
            let Some(first) = first else {
                continue;
            };
            battlepresent::build_native_physical_body_plan(
                Some(first),
                second,
                0,
                1,
                &[battle::NativePhysicalStrike::default()],
            )?;
            for a in [Some(first), second].into_iter().flatten() {
                for f in &a.frames {
                    if f.width <= 0
                        || f.height <= 0
                        || f.width > 1024
                        || f.height > 1024
                        || f.pixels.len() as i64 != f.width * f.height
                        || f.mask.len() != f.pixels.len()
                    {
                        return Err(game_err!("native physical body FIGANI shape unavailable"));
                    }
                }
            }
            if first.header_byte1 != 0 {
                for f in &first.frames[..first.header_byte2 as usize] {
                    f.blit_at(&mut vec![0u8; 64000], 320)?;
                }
            }
        }
        r.main_sounds = Arc::new(self.native_physical_sound_bank(attack)?);
        if let Some(counter) = r.counter.clone() {
            r.counter_sounds = Arc::new(self.native_physical_sound_bank(&counter)?);
        }
        if self
            .modern_story_portraits
            .as_ref()
            .is_some_and(|set| !set.battle_animations.is_empty() || !set.battle_surfaces.is_empty())
        {
            todo!(
                "R6: modern_hd_battle_surface.go:validatePhysicalScenery、modern_hd_battle_assets.go:prepareBattleHDResources（HD 素材預檢）"
            )
        }
        scene.body_resources = Some(r);
        Ok(())
    }

    /// 2BC9A索引525D6的bytes30..35；header4零表示無bank。
    /// 既有證據：docs/data/ida/fd2_command24_presentation_ida.txt。
    ///
    /// Go：`native_physical_body.go:157`，`func (g *Game) nativePhysicalSoundBank(animation *figani.Animation) (map[int][]byte, error)`
    pub(crate) fn native_physical_sound_bank(
        &mut self,
        animation: &figani::Animation,
    ) -> Result<BTreeMap<i64, Vec<u8>>> {
        // Go 回傳 nil map；Rust 以空表代表（讀 nil map 得零值，與空表相同）。
        if animation.header_byte4 == 0 {
            return Ok(BTreeMap::new());
        }
        if animation.header_byte4 > 6 {
            return Err(game_err!("native physical FIGANI sound selector unavailable"));
        }
        let resource = 48 + animation.header_byte4 as i64 - 1;
        // Go `bank == nil` 時才解碼；Rust 的快取以「有沒有這個鍵」判斷（Go 不會存入 nil bank）。
        let bank = match self.native_physical_sound_banks.get(&resource) {
            Some(bank) => bank.clone(),
            None => {
                let bank = decode_separated_sound_bank(resource)?;
                self.native_physical_sound_banks.insert(resource, bank.clone());
                bank
            }
        };
        let missing = |cue: i64| bank.get(&cue).is_none_or(|pcm| pcm.is_empty());
        for f in &animation.frames {
            if f.raw_byte5 != 0 && (missing(f.raw_byte5 as i64) || (f.raw_byte4 != 0 && missing(0))) {
                return Err(game_err!(
                    "native physical FDOTHER{} cue{} or miss cue0 unavailable",
                    resource,
                    f.raw_byte5
                ));
            }
        }
        Ok(bank)
    }

    /// Go：`native_physical_body.go:206`，`func (g *Game) attachNativePhysicalBody(scene *nativePhysicalScene, result battle.NativePhysicalAttackResult) error`
    pub(crate) fn attach_native_physical_body(
        &mut self,
        scene: Option<&mut NativePhysicalScene>,
        result: &battle::NativePhysicalAttackResult,
    ) -> Result<()> {
        let Some(scene) = scene else {
            return Ok(()); // 無原生地圖狀態的既有相容呈現。
        };
        let Some(r) = scene.body_resources.clone() else {
            return Err(game_err!("native physical body resources not preflighted"));
        };
        let mut body = NativePhysicalBodyAssembly {
            p: NativePhysicalBodyPlayback {
                last_present: -1,
                return_wait_millis: 6.0 * indexedmap::PHASE_BANNER_STEP_MILLIS,
                ..Default::default()
            },
            base: r.base.clone(),
            base_scenery: Vec::new(),
        };
        if scene.selection.has_separate_backgrounds {
            body.base_scenery = modern_physical_scenery(Some(scene), r.actor_side != 0)?;
        }
        let (mut actor_record, mut target_record) = (scene.actor_record.clone(), scene.target_record.clone());
        let (actor_index, target_index) = (scene.actor_index, scene.target_index);
        let (actor_fig, target_fig) = (scene.actor_battle_fig, scene.target_battle_fig);
        let attack = r.attack.as_ref().expect("Go：prepareNativePhysicalBodyResources 已設 attack");
        let target_idle = r.target_idle.as_ref().expect("Go：已載入守方待機");
        let actor_idle = r.actor_idle.as_ref().expect("Go：已載入攻方待機");
        self.append_native_physical_body_stage(
            scene,
            &r,
            &mut body,
            Some(attack),
            target_idle,
            r.actor_side,
            &result.attack_strikes,
            &mut actor_record,
            &mut target_record,
            actor_index,
            target_index,
            &r.main_sounds,
            actor_fig * 3 + 1,
            target_fig * 3,
            "body",
        )?;
        if result.counter.is_some() {
            self.append_native_physical_body_stage(
                scene,
                &r,
                &mut body,
                r.counter.as_ref(),
                actor_idle,
                r.target_side,
                &result.counter_strikes,
                &mut target_record,
                &mut actor_record,
                target_index,
                actor_index,
                &r.counter_sounds,
                target_fig * 3 + 1,
                actor_fig * 3,
                "counter",
            )?;
        }
        let mut p = body.p;
        if attack.header_byte1 == 0 {
            let mut pixels = body.base.clone();
            let mut platform = r.platform.clone();
            (platform.x, platform.y) = (164, 157);
            platform.blit(&mut pixels, 320, -1)?;
            let hd_restore = ModernHDBattleSources {
                base: pixels.clone(),
                layers: vec![
                    ModernHDBattleSource {
                        frame: target_idle.frames[0].clone(),
                        resource: target_fig * 3,
                        fill: -1,
                        ..Default::default()
                    },
                    ModernHDBattleSource {
                        frame: actor_idle.frames[0].clone(),
                        resource: actor_fig * 3,
                        fill: -1,
                        ..Default::default()
                    },
                ],
                ..Default::default()
            };
            target_idle.frames[0].blit_at(&mut pixels, 320)?;
            actor_idle.frames[0].blit_at(&mut pixels, 320)?;
            // 28F4D..28FD6的final restore。
            p.jobs.push(NativePhysicalBodyJob {
                pixels,
                cue: -1,
                hd_sources: Some(hd_restore),
                hd_phase: String::from("restore"),
                ..Default::default()
            });
        }
        // Go 以 `bound` map 依指標去重，同一份來源只綁一次；Rust 每個工作各有一份（脈衝工作複製自前一個
        // 工作），逐份綁定。綁定對同一份來源是冪等的（清台座底層、補面板文字都不會重複），結果相同。
        for i in 0..p.jobs.len() {
            if let Some(sources) = p.jobs[i].hd_sources.as_mut() {
                bind_modern_physical_scenery(scene, Some(sources))?;
            }
            if !p.jobs[i].pixels.is_empty() || p.jobs[i].present.is_some() {
                p.last_present = i as i64;
            }
        }
        scene.body = Some(p);
        Ok(())
    }

    /// Go `attachNativePhysicalBody` 的閉包 `appendStage`（`native_physical_body.go:259`..`367`）。
    #[allow(clippy::too_many_arguments)]
    fn append_native_physical_body_stage(
        &mut self,
        scene: &mut NativePhysicalScene,
        r: &NativePhysicalBodyResources,
        body: &mut NativePhysicalBodyAssembly,
        attack: Option<&figani::Animation>,
        idle: &figani::Animation,
        side: u8,
        strikes: &[battle::NativePhysicalStrike],
        current_actor: &mut [u8],
        current_target: &mut [u8],
        actor_index: i64,
        target_index: i64,
        sounds: &Arc<BTreeMap<i64, Vec<u8>>>,
        attack_resource: i64,
        idle_resource: i64,
        body_phase: &str,
    ) -> Result<()> {
        let Some(attack) = attack else {
            return Err(game_err!("native physical counter was not preflighted"));
        };
        let initial_hp = u16::from_le_bytes([current_target[64], current_target[65]]) as i64;
        let plan = battlepresent::build_native_physical_body_plan(Some(attack), Some(idle), side, initial_hp, strikes)?;
        let sound_bank = 48 + attack.header_byte4 as i64 - 1;
        let mut plan_cursor = 0usize;
        for si in 0..strikes.len() {
            if attack.header_byte1 != 0 {
                let mut source = vec![0; 320 * 200];
                for frame in &attack.frames[..usize::from(attack.header_byte2)] {
                    source = body.base.clone();
                    frame.blit_at(&mut source, 320)?;
                    let cue = if frame.raw_byte5 == 0 { -1 } else { i64::from(frame.raw_byte5) };
                    body.p.jobs.push(NativePhysicalBodyJob {
                        pixels: source.clone(),
                        cue,
                        wait_millis: frame.delay as f64 * indexedmap::PHASE_BANNER_STEP_MILLIS,
                        sounds: Arc::clone(sounds),
                        sound_bank,
                        hd_phase: "departure".into(),
                        ..Default::default()
                    });
                }
                let (base, frames) = self.native_physical_scroll(
                    scene,
                    &source,
                    &r.target_bg,
                    current_target,
                    target_index,
                    &idle.frames[0],
                    side == 0,
                    &r.layers,
                )?;
                body.append_scroll(
                    base,
                    frames,
                    modern_physical_backdrop_scenery(
                        scene,
                        &r.target_bg,
                        modern_physical_target_bg_index(scene),
                        side == 0,
                    )?,
                );
            }
            while plan_cursor < plan.len() && plan[plan_cursor].strike == si as i64 {
                let f = &plan[plan_cursor];
                plan_cursor += 1;
                if f.update_panel {
                    current_target[64..66].copy_from_slice(&(f.hp as u16).to_le_bytes());
                    let mut next = body.base.clone();
                    self.render_physical_battle_panel(scene, current_target, &mut next, target_index)?;
                    body.base = next;
                }
                if f.presents.is_empty() && f.cue >= 0 {
                    body.p.jobs.push(NativePhysicalBodyJob {
                        cue: f.cue,
                        sounds: Arc::clone(sounds),
                        sound_bank,
                        ..Default::default()
                    });
                }
                for (inner, present) in f.presents.iter().enumerate() {
                    let attack_frame = &attack.frames[f.frame as usize];
                    let idle_frame = &idle.frames[present.idle_frame as usize];
                    let mut job = NativePhysicalBodyJob {
                        base: body.base.clone(),
                        attack: attack_frame.clone(),
                        idle: idle_frame.clone(),
                        raw_side: side,
                        present: Some(*present),
                        cue: -1,
                        sounds: Arc::clone(sounds),
                        hd_phase: body_phase.to_string(),
                        sound_bank,
                        ..Default::default()
                    };
                    let mut sources = modern_hd_battle_body_sources(
                        &body.base,
                        attack_frame,
                        idle_frame,
                        side,
                        present,
                        attack_resource,
                        f.frame,
                        idle_resource,
                        present.idle_frame,
                    );
                    // Go 的閉包 bindBase。
                    bind_modern_physical_base_scenery(Some(&mut sources), &mut body.base_scenery.clone())?;
                    bind_modern_physical_panel_text(Some(scene), Some(&mut sources));
                    job.hd_sources = Some(sources);
                    if inner == 0 {
                        job.cue = f.cue;
                        job.hd_impact = f.cue > 0 && attack_frame.raw_byte4 != 0;
                    }
                    if !present.status_pulse && !present.crit_pulse {
                        job.wait_millis = indexedmap::PHASE_BANNER_STEP_MILLIS;
                        body.p.jobs.push(job);
                        continue;
                    }
                    let mut pulse = job.clone();
                    body.p.jobs.push(job);
                    pulse.cue = -1;
                    pulse.hd_impact = false;
                    let black = ColorNrgba { a: 255, ..Default::default() };
                    if present.status_pulse {
                        let green = ColorNrgba { r: 4, g: 130, a: 255, ..Default::default() }; // DAC(1,32,0)
                        (pulse.palette0, pulse.wait_millis) = (Some(green), 20.0);
                        body.p.jobs.push(pulse.clone());
                        (pulse.palette0, pulse.wait_millis) = (Some(black), 0.0);
                        body.p.jobs.push(pulse.clone());
                    }
                    if present.crit_pulse {
                        let white = ColorNrgba { r: 255, g: 255, b: 255, a: 255 };
                        (pulse.palette0, pulse.wait_millis) = (Some(white), 20.0);
                        body.p.jobs.push(pulse.clone());
                    }
                    (pulse.palette0, pulse.wait_millis) = (Some(black), indexedmap::PHASE_BANNER_STEP_MILLIS);
                    if present.crit_pulse {
                        pulse.wait_millis += 40.0;
                    }
                    body.p.jobs.push(pulse);
                }
            }
            if si + 1 < strikes.len() && attack.header_byte1 == 1 {
                let mut source = body.base.clone();
                idle.frames[0].blit_at(&mut source, 320)?;
                let (base, frames) = self.native_physical_scroll(
                    scene,
                    &source,
                    &r.actor_bg,
                    current_actor,
                    actor_index,
                    &attack.frames[0],
                    side != 0,
                    &r.layers,
                )?;
                body.append_scroll(
                    base,
                    frames,
                    modern_physical_backdrop_scenery(scene, &r.actor_bg, scene.selection.base_bg, side != 0)?,
                );
            }
        }
        Ok(())
    }
}

/// R4 D 組新增，沒有 Go 對應：`attachNativePhysicalBody` 的閉包共用的區域變數 `p`、`base`、`baseScenery`。
struct NativePhysicalBodyAssembly {
    p: NativePhysicalBodyPlayback,
    base: Vec<u8>,
    base_scenery: Vec<ModernHDBattleSource>,
}

impl NativePhysicalBodyAssembly {
    fn append_scroll(
        &mut self,
        base: Vec<u8>,
        frames: Vec<battlepresent::NativeCommand24BackgroundPresent>,
        scenery: Vec<ModernHDBattleSource>,
    ) {
        self.p.jobs.extend(frames.into_iter().map(|frame| NativePhysicalBodyJob {
            pixels: frame.pixels,
            cue: -1,
            hd_phase: if frame.incoming { "scroll-in" } else { "scroll-out" }.into(),
            ..Default::default()
        }));
        self.base = base;
        self.base_scenery = scenery;
    }
}

impl NativePhysicalBodyJob {
    /// Go `drawNativePhysicalBody`（`native_physical_body.go:433..446`）在 `p.image == nil` 時建立快取畫面的那段
    /// （Rust 拆出，`draw_native_physical_body` 與 `present.rs` 的呈現回報共用）。像素為空（cue-only）回 `None`。
    pub(crate) fn present_image(&self, ui_palette: &[ColorRgba]) -> Result<Option<Picture>> {
        let pixels = self.indexed()?;
        if pixels.is_empty() {
            return Ok(None);
        }
        let mut palette = ui_palette.to_vec();
        if let Some(c) = self.palette0 {
            // Go：palette[0] = *job.palette0（color.NRGBA），NewImageFromImage 以 color.RGBAModel 預乘。
            palette[0] = c.premultiplied();
        }
        // Go：image.NewPaletted(320×200, palette)，copy(img.Pix, pixels)。
        let mut pix = vec![0u8; 320 * 200];
        let n = pix.len().min(pixels.len());
        pix[..n].copy_from_slice(&pixels[..n]);
        Ok(Some(Picture::from_indexed(&IndexedFrame { width: 320, height: 200, pix, palette })))
    }

    /// Go：`native_physical_body.go:407`，`func (job *nativePhysicalBodyJob) indexed() ([]byte, error)`
    pub(crate) fn indexed(&self) -> Result<Vec<u8>> {
        let Some(present) = self.present.as_ref() else {
            return Ok(self.pixels.clone());
        };
        Ok(battlepresent::compose_native_physical_body_present(
            &self.base,
            &self.attack,
            &self.idle,
            self.raw_side,
            *present,
        )?)
    }
}

impl Game {
    /// Go：`native_physical_body.go:414`，`func (g *Game) drawNativePhysicalBody(screen *ebiten.Image, scene *nativePhysicalScene) error`
    pub(crate) fn draw_native_physical_body(
        &mut self,
        screen: &mut Picture,
        scene: &mut NativePhysicalScene,
    ) -> Result<()> {
        // R5 階段 2 r5b：逐行。`p.drawn` 與快取畫面 `p.image` 照 Go 在這裡設（`draw` 路徑）；無畫面重播由
        // `present.rs` 的呈現回報做同樣的狀態改寫。
        let hd_wanted = self.modern_hd_battle_wanted(Some(scene));
        let ui_palette = &self.native_ui_palette;
        let Some(p) = scene.body.as_mut() else {
            return Err(game_err!("native physical body has no current present"));
        };
        let mut index = p.index;
        if index == p.jobs.len() as i64 && p.return_wait_millis > 0.0 {
            index = p.last_present; // caller等待保留最後影格，不重播cue或新增present。
        }
        if index < 0 || index >= p.jobs.len() as i64 {
            return Err(game_err!("native physical body has no current present"));
        }
        let job = &p.jobs[index as usize];
        // Go `drawModernHDBattleBody(screen, scene, job)`：`scene` 與 `job`（在 `scene.body.jobs` 裡）無法同時以
        // `&mut` 借出，這裡照它的守門（`modernHDBattleWanted(scene)` 且 `job.hdSources` 非 nil）判斷；忠實模式回 false。
        if hd_wanted && job.hd_sources.is_some() {
            todo!("r6:modern_hd_battle_runtime.go:331 drawModernHDBattleBody（HD 戰鬥本體）")
        }
        if p.image.is_none() {
            let Some(img) = job.present_image(ui_palette)? else {
                return Ok(()); // cue-only項目由Update消費。
            };
            p.image = Some(Arc::new(img));
        }
        let mut op = DrawImageOptions::default();
        op.geo_m.scale(2.0, 2.0);
        screen.draw_image(p.image.as_ref().unwrap(), &op);
        p.drawn = true;
        Ok(())
    }

    /// stepNativePhysicalBodyMillis 以60Hz來源播放20/40ms與BIOS等待。
    /// 每個實際present必須由Draw確認；不累積缺畫面的時間、不跨格追趕。
    ///
    /// Go：`native_physical_body.go:458`，`func (g *Game) stepNativePhysicalBodyMillis(scene *nativePhysicalScene, elapsed float64) error`
    pub(crate) fn step_native_physical_body_millis(
        &mut self,
        scene: &mut NativePhysicalScene,
        elapsed: f64,
    ) -> Result<()> {
        let p = scene.body.as_mut().expect("Go：呼叫端只在 scene.body 非 nil 時呼叫");
        if p.return_ramp_started {
            return Ok(()); // 既有全DAC owner等待最後Draw後，才交接map與after。
        }
        if p.index == p.jobs.len() as i64 {
            p.elapsed_millis += elapsed;
        }
        while p.index < p.jobs.len() as i64 {
            let index = p.index as usize;
            if !p.cued {
                let job = &p.jobs[index];
                let impact = job.hd_impact && job.cue > 0 && job.present.is_some();
                p.hd_extra_millis = self.modern_physical_impact_hold_millis(impact);
                if job.cue >= 0 {
                    let original = job.sounds.get(&job.cue).cloned().unwrap_or_default();
                    let sound = self.modern_physical_sound(job.sound_bank, job.cue, &original);
                    self.play_native_physical_sound(&sound);
                }
                p.cued = true;
            }
            let job = &p.jobs[index];
            let presents = !job.pixels.is_empty() || job.present.is_some();
            if presents {
                if !p.drawn {
                    return Ok(());
                }
                p.elapsed_millis += elapsed;
                if p.elapsed_millis < job.wait_millis + p.hd_extra_millis {
                    return Ok(());
                }
            }
            if p.image.is_some() && p.index < p.last_present && presents {
                p.image = None;
            }
            p.index += 1;
            let mut remaining = p.elapsed_millis - job.wait_millis - p.hd_extra_millis;
            if remaining < 0.0 || job.wait_millis == 0.0 {
                remaining = 0.0;
            }
            (p.drawn, p.cued, p.elapsed_millis, p.hd_extra_millis) = (false, false, remaining, 0.0);
            if p.index < p.jobs.len() as i64 && presents {
                let next = &p.jobs[p.index as usize];
                if !next.pixels.is_empty() || next.present.is_some() {
                    return Ok(());
                }
            }
        }
        if p.elapsed_millis < p.return_wait_millis {
            return Ok(());
        }
        if self.native_map_assets.is_some() {
            // 290A7→1F882在最後body畫布上依序寫64份DAC；不能用舊map VGA。
            // READY：physical_owner_return_dac_spec。沿既有Draw確認與delay2近似。
            if p.last_present < 0 || p.last_present >= p.jobs.len() as i64 {
                return Err(game_err!("native physical return present unavailable"));
            }
            let last = &p.jobs[p.last_present as usize];
            let pixels = last.indexed()?;
            let hd_sources = last.hd_sources.clone();
            let then: Box<dyn FnOnce(&mut Game) + Send> = Box::new(|g: &mut Game| g.finish_attack_presentation());
            self.start_native_palette_ramp_on_frame(&pixels, 0, 63, 2, Some(then))
                .map_err(|err| err.wrap("native physical return DAC: "))?;
            self.native_palette_ramp
                .as_mut()
                .expect("Go：startNativePaletteRampOnFrame 成功後設好工作")
                .hd_battle_sources = hd_sources;
            scene.body.as_mut().unwrap().return_ramp_started = true;
            return Ok(());
        }
        self.finish_attack_presentation();
        if self.native_field_event61.is_some() || self.battle_event.is_some() {
            return Err(ERR_ATTACK_PRESENTATION_YIELD.clone());
        }
        Ok(())
    }
}

#[cfg(test)]
#[path = "native_physical_body/native_physical_body_test.rs"]
mod native_physical_body_test;
