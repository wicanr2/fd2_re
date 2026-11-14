//! Go 來源：`remake/internal/ending/montage_cycle.go`（403 行）。

use super::*;

/// MontageCycleAssets contains only the original indexed inputs consumed by
/// 0x2c405's 0x2c548 party cycle.  Nothing here is a PNG or a guessed UI
/// replacement; missing or malformed source assets reject the cycle.
///
/// Go：`montage_cycle.go:16`
#[derive(Debug, Clone, Default)]
pub struct MontageCycleAssets {
    pub backdrop: fdother::Frame,
    pub tai003: fdother::Frame,
    pub primary: BTreeMap<i64, figani::Animation>,
    pub secondary: BTreeMap<i64, figani::Animation>,
    pub grid: Vec<u8>,
    pub portraits: BTreeMap<i64, Vec<dato::Frame>>,
    pub current: Option<fdtxt::Strings>,
    pub permanent: Option<fdtxt::Strings>,
    pub font: Option<fdtxt::Font>,
}

/// MontageArchivePaths identifies the remaining player-provided archives and
/// separated asset roots used by the native cycle. Originals remain read-only.
///
/// Go：`montage_cycle.go:30`
#[derive(Debug, Clone, Default, PartialEq, Eq)]
pub struct MontageArchivePaths {
    pub surface_root: String,
    pub item_panel_root: String,
    pub animation_root: String,
    pub portrait_root: String,
    pub text_root: String,
    pub font_root: String,
}

/// LoadMontageCycleAssets decodes the exact resources named by the native
/// call-sites: FDOTHER#56, TAI#3, FDOTHER#5, FIGANI group*3/{+1}, DATO unit
/// +7, FDTXT_031/FDTXT_000 and the FDOTHER#4 font.  It deliberately takes the
/// raw unit records from the caller so no identity or slot meaning is guessed.
///
/// Go：`montage_cycle.go:43`，`func LoadMontageCycleAssets(montage Montage, paths MontageArchivePaths, units [][]byte) (MontageCycleAssets, error)`
pub fn load_montage_cycle_assets(
    montage: &Montage,
    paths: &MontageArchivePaths,
    units: &[Vec<u8>],
) -> Result<MontageCycleAssets> {
    if paths.surface_root.is_empty()
        || paths.item_panel_root.is_empty()
        || paths.animation_root.is_empty()
        || paths.portrait_root.is_empty()
        || paths.text_root.is_empty()
        || paths.font_root.is_empty()
        || units.len() < 2
    {
        return Err(campaign_err!("ending: incomplete montage archive paths or party"));
    }
    for (i, unit) in units.iter().enumerate() {
        if unit.len() < 0x21 {
            return Err(campaign_err!("ending: unit {i} is shorter than native stride"));
        }
    }
    let backdrop = fdother::load_separated_single_frame(&paths.surface_root, "FDOTHER.DAT", 56)
        .map_err(|err| CampaignError::from(err).wrap("ending: FDOTHER#56: "))?;
    let tai003 = fdother::load_separated_single_frame(&paths.surface_root, "TAI.DAT", 3)
        .map_err(|err| CampaignError::from(err).wrap("ending: TAI#3: "))?;
    if !is_transparent_tai003(&tai003) {
        return Err(campaign_err!("ending: separated TAI#3 is not the proven transparent resource"));
    }
    let mut grid = vec![0u8; BYTES as usize];
    render_dialogue_frame_grid_separated(montage, &paths.item_panel_root, &mut grid)
        .map_err(|err| err.wrap("ending: FDOTHER#5 dialogue grid: "))?;
    let current = fdtxt::load_separated_resource(&paths.text_root, 31)
        .map_err(|err| CampaignError::from(err).wrap("ending: separated FDTXT_031: "))?;
    let permanent = fdtxt::load_separated_resource(&paths.text_root, 0)
        .map_err(|err| CampaignError::from(err).wrap("ending: separated FDTXT_000: "))?;
    let font = fdtxt::load_separated_font(&paths.font_root)
        .map_err(|err| CampaignError::from(err).wrap("ending: separated FDOTHER#4 font: "))?;
    let mut assets = MontageCycleAssets {
        backdrop,
        tai003,
        grid,
        current: Some(current),
        permanent: Some(permanent),
        font: Some(font),
        ..Default::default()
    };
    for unit in units {
        let group = i64::from(unit[7]);
        for resource in [group * 3, group * 3 + 1] {
            if assets.secondary.contains_key(&resource) {
                continue;
            }
            let animation = figani::load_separated_resource(&paths.animation_root, resource)
                .map_err(|err| CampaignError::from(err).wrap(format!("ending: FIGANI#{resource}: ")))?;
            // Go 讓 Primary 與 Secondary 共用同一個 *Animation；動畫只讀，Rust 各存一份複本。
            assets.secondary.insert(resource, animation.clone());
            assets.primary.insert(resource, animation);
        }
        if let std::collections::btree_map::Entry::Vacant(entry) = assets.portraits.entry(group) {
            let frames = dato::load_separated_resource(&paths.portrait_root, group)
                .map_err(|err| CampaignError::from(err).wrap(format!("ending: separated portrait {group}: ")))?;
            entry.insert(frames);
        }
    }
    Ok(assets)
}

/// MontageCyclePhase is the directly recovered first-party schedule.  The
/// final palette fade is kept as a distinct phase because its native helper
/// has a different delay unit from the BIOS-tick waits in the party loops.
///
/// Go：`montage_cycle.go:111`
#[derive(Debug, Clone, Copy, Default, PartialEq, Eq, PartialOrd, Ord, Hash)]
pub struct MontageCyclePhase(pub &'static str);

pub const MONTAGE_PHASE_FIGURE_FADE: MontageCyclePhase = MontageCyclePhase("figure_fade");

pub const MONTAGE_PHASE_SECONDARY: MontageCyclePhase = MontageCyclePhase("secondary_intro");

pub const MONTAGE_PHASE_PRIMARY: MontageCyclePhase = MontageCyclePhase("primary_animation");

pub const MONTAGE_PHASE_PORTRAIT: MontageCyclePhase = MontageCyclePhase("portrait_text");

pub const MONTAGE_PHASE_FADE_OUT: MontageCyclePhase = MontageCyclePhase("palette_fade_out");

pub const MONTAGE_PHASE_COMPLETED: MontageCyclePhase = MontageCyclePhase("completed");

/// MontageCycle is a deterministic indexed executor for the verified
/// 0x2c548 schedule.  Step performs one native render iteration; the caller
/// owns the returned delay units and the raw random byte used by 0x4e893.
/// ObserveInputChange is deliberately a separate raw event hook: 0x2c950 only
/// observes a changed input-buffer condition during portrait presentation; it
/// does not decode a particular key in this branch.
///
/// Go：`montage_cycle.go:128`
#[derive(Debug, Clone, Default)]
pub struct MontageCycle {
    pub montage: Montage,
    pub assets: MontageCycleAssets,
    pub units: Vec<Vec<u8>>,
    pub plans: Vec<PartyCyclePlan>,
    pub compositor: Option<SharedCompositor>,
    pub plan_index: i64,
    pub phase: MontageCyclePhase,
    pub fade_index: i64,
    pub secondary: Option<figani::Animation>,
    pub primary: Option<figani::Animation>,
    pub secondary_sm: figani::NativeScheduler,
    pub primary_idx: i64,
    pub portrait: i64,
    pub portrait_max: i64,
    pub portrait_sm: MontagePortraitState,
    pub fade_out: i64,
    pub delay_ticks: i64,
    pub delay_ms: i64,
    pub(crate) skip_to_final: bool,
    pub(crate) portrait_boundary_pending: bool,
}

/// NewMontageCycle validates all source provenance before exposing an
/// executable renderer.  In particular, the caller must provide the palette
/// baseline captured from a native indexed presentation; an all-zero default
/// is not accepted as a substitute.
///
/// Go：`montage_cycle.go:155`，`func NewMontageCycle(m Montage, assets MontageCycleAssets, units [][]byte, groups []byte, compositor *IndexedCompositor) (*MontageCycle, error)`
/// Rust：Go 把 `units` 切片直接存進 `MontageCycle.Units`，之後只讀（`montage_cycle.go` 238、251、334 行）；
/// Rust 複製一份給結構擁有，所以收 `&[Vec<u8>]`（產生器因「存進之後會被改寫的結構」誤判成可變）。
pub fn new_montage_cycle(
    m: &Montage,
    assets: MontageCycleAssets,
    units: &[Vec<u8>],
    groups: &[u8],
    compositor: Option<SharedCompositor>,
) -> Result<MontageCycle> {
    let Some(compositor) = compositor.filter(|c| {
        m.status == "mapped_first_party_cycle_fail_closed"
            && lock_compositor(c).baseline_known()
            && units.len() >= 2
            && units.len() == groups.len()
            && assets.grid.len() as i64 == BYTES
            && is_transparent_tai003(&assets.tai003)
    }) else {
        return Err(campaign_err!("ending: montage source provenance is incomplete"));
    };
    if assets.backdrop.width != WIDTH
        || assets.backdrop.height != HEIGHT
        || assets.current.is_none()
        || assets.permanent.is_none()
        || assets.font.is_none()
    {
        return Err(campaign_err!("ending: montage backdrop or text assets are incomplete"));
    }
    let plans = m.plan_party_cycle(groups)?;
    for (i, unit) in units.iter().enumerate() {
        if unit.len() < 0x21 {
            return Err(campaign_err!("ending: unit {i} is shorter than native stride"));
        }
        let group = i64::from(unit[7]);
        if assets.portraits.get(&group).is_none_or(|frames| frames.len() < 4) {
            return Err(campaign_err!("ending: DATO portrait {group} is unavailable"));
        }
    }
    for plan in &plans {
        let primary = assets.primary.get(&plan.primary_figani);
        let secondary = assets.secondary.get(&plan.secondary_figani);
        let (Some(primary), Some(_)) =
            (primary.filter(|a| !a.frames.is_empty()), secondary.filter(|a| !a.frames.is_empty()))
        else {
            return Err(campaign_err!(
                "ending: FIGANI resources {}/{} are unavailable",
                plan.primary_figani,
                plan.secondary_figani
            ));
        };
        if primary.frames.iter().any(|frame| frame.delay <= 0) {
            return Err(campaign_err!("ending: FIGANI#{} contains zero frame delay", plan.primary_figani));
        }
    }
    let mut p = MontageCycle {
        montage: m.clone(),
        assets,
        units: units.to_vec(),
        plans,
        compositor: Some(compositor),
        phase: MONTAGE_PHASE_FIGURE_FADE,
        ..Default::default()
    };
    p.prepare_plan()?;
    Ok(p)
}

impl MontageCycle {
    /// Go：`montage_cycle.go:197`，`func (p *MontageCycle) preparePlan() error`
    pub(crate) fn prepare_plan(&mut self) -> Result<()> {
        let plan = self.plans[self.plan_index as usize];
        self.secondary = self.assets.secondary.get(&plan.secondary_figani).cloned();
        self.primary = self.assets.primary.get(&plan.primary_figani).cloned();
        self.fade_index = 0;
        self.portrait = 0;
        self.portrait_sm = MontagePortraitState::default();
        self.primary_idx = 0;
        self.delay_ticks = 0;
        self.delay_ms = 0;
        self.portrait_boundary_pending = false;
        self.render_backdrop()
    }

    /// Go：`montage_cycle.go:214`，`func (p *MontageCycle) renderBackdrop() error`
    pub(crate) fn render_backdrop(&mut self) -> Result<()> {
        let Some(shared) = self.compositor.as_ref() else {
            return Err(campaign_err!("ending: montage cycle is not running"));
        };
        let mut c = lock_compositor(shared);
        c.offscreen.fill(0);
        self.assets
            .backdrop
            .blit(&mut c.offscreen, WIDTH, -1)
            .map_err(|err| CampaignError::from(err).wrap("ending: FDOTHER#56 backdrop: "))
    }

    /// Step advances exactly one recovered rendering iteration.  randomByte is
    /// supplied by the caller because the original obtains it from 0x4e893; the
    /// executor never invents a replacement RNG.  The delay fields describe the
    /// native wait following the produced frame.
    ///
    /// Go：`montage_cycle.go:226`，`func (p *MontageCycle) Step(randomByte byte) error`
    /// Go 版先檢查 nil 接收者；Rust 的 `self` 不會是 null，該分支不移植。
    pub fn step(&mut self, random_byte: u8) -> Result<()> {
        if self.phase == MONTAGE_PHASE_COMPLETED {
            return Err(campaign_err!("ending: montage cycle is not running"));
        }
        let Some(shared) = self.compositor.clone() else {
            return Err(campaign_err!("ending: montage cycle is not running"));
        };
        (self.delay_ticks, self.delay_ms) = (0, 0);
        let mut plan = PartyCyclePlan::default();
        let mut unit: Vec<u8> = Vec::new();
        if self.phase != MONTAGE_PHASE_FADE_OUT {
            if self.plan_index < 0 || self.plan_index >= self.plans.len() as i64 {
                return Err(campaign_err!("ending: montage plan index is out of range"));
            }
            plan = self.plans[self.plan_index as usize];
            unit = self.units[plan.unit_slot as usize].clone();
        }
        let secondary = self.secondary.clone().unwrap_or_default();
        match self.phase {
            MONTAGE_PHASE_FIGURE_FADE => {
                if self.fade_index >= 9 {
                    self.phase = MONTAGE_PHASE_SECONDARY;
                    self.secondary_sm = figani::NativeScheduler::default();
                    self.secondary_sm.step(self.secondary.as_ref(), false)?;
                    return Ok(());
                }
                let frame = &secondary.frames[0];
                let mut guard = lock_compositor(&shared);
                let c = &mut *guard;
                // Go 把 c.Work／c.Offscreen 當參數傳給同一個合成器；三塊緩衝互不重疊，
                // Rust 先把 work 移出、事後放回，結果相同。
                let mut work = std::mem::take(&mut c.work);
                let result = if unit[6] != 0 {
                    match self.montage.plan_figure_fade(i64::from(unit[6])) {
                        Ok(passes) => {
                            let offscreen = c.offscreen.clone();
                            render_figure_fade_pass(
                                Some(c),
                                &mut work,
                                &offscreen,
                                &self.assets.tai003,
                                frame,
                                passes[self.fade_index as usize],
                            )
                        }
                        Err(err) => Err(err),
                    }
                } else {
                    match self.montage.plan_mirror_figure_fade(0, 1) {
                        Ok(passes) => copy_rect(&mut work, WIDTH * 2, &c.offscreen, WIDTH, WIDTH, HEIGHT, WIDTH)
                            .and_then(|()| {
                                render_mirror_figure_fade_pass(
                                    Some(c),
                                    &mut work,
                                    &self.assets.tai003,
                                    frame,
                                    frame,
                                    passes[self.fade_index as usize],
                                )
                            }),
                        Err(err) => Err(err),
                    }
                };
                c.work = work;
                result?;
                self.fade_index += 1;
                Ok(())
            }
            MONTAGE_PHASE_SECONDARY => {
                let (index, _) = self.secondary_sm.step(self.secondary.as_ref(), true)?;
                self.render_figani(&secondary.frames[index as usize])?;
                self.delay_ticks = 1;
                self.portrait += 1;
                if self.portrait >= 20 {
                    self.phase = MONTAGE_PHASE_PRIMARY;
                    self.portrait = 0;
                }
                Ok(())
            }
            MONTAGE_PHASE_PRIMARY => {
                let primary = self.primary.clone().unwrap_or_default();
                if self.primary_idx >= primary.frames.len() as i64 {
                    self.phase = MONTAGE_PHASE_PORTRAIT;
                    self.portrait = 0;
                    self.portrait_max = native_montage_portrait_iterations(plan.loop_index).unwrap_or(0);
                    self.portrait_sm = MontagePortraitState::default();
                    self.delay_ticks = 0;
                    return Ok(());
                }
                let frame = &primary.frames[self.primary_idx as usize];
                self.render_figani(frame)?;
                self.delay_ticks = frame.delay;
                self.primary_idx += 1;
                Ok(())
            }
            MONTAGE_PHASE_PORTRAIT => {
                // The native loop polls 0x10620 only after present+0x17aa9(1), then
                // increments EDI and decides whether to leave this portrait.  Keep that
                // boundary separate from rendering so an input arriving during the wait
                // can still affect the final current portrait, rather than one later.
                if self.portrait_boundary_pending {
                    self.portrait_boundary_pending = false;
                    self.portrait += 1;
                    if self.portrait >= self.portrait_max {
                        let mut next_plan = self.plan_index + 1;
                        // At 0x2c950 a pending raw input change sets the native outer
                        // counter to one.  The current portrait still completes; the next
                        // outer-loop iteration is therefore i=0, rather than an immediate
                        // abort.  The exact key code remains intentionally outside this API.
                        if self.skip_to_final {
                            next_plan = self.plans.len() as i64 - 1;
                            self.skip_to_final = false;
                        }
                        self.plan_index = next_plan;
                        if self.plan_index >= self.plans.len() as i64 {
                            self.phase = MONTAGE_PHASE_FADE_OUT;
                            self.fade_out = 0;
                        } else {
                            self.phase = MONTAGE_PHASE_FIGURE_FADE;
                            self.prepare_plan()?;
                        }
                    }
                    return Ok(());
                }
                // Go 以 map 取值，缺鍵得到 nil 切片；建構時已確認每個 unit 的頭像存在。
                let empty = Vec::new();
                let portrait_frames = self.assets.portraits.get(&i64::from(unit[7])).unwrap_or(&empty);
                let frame_index = self.portrait_sm.step(random_byte)?;
                let mut frame = compose_montage_portrait_frame(
                    &self.montage,
                    &self.assets.grid,
                    &unit,
                    self.portrait,
                    frame_index,
                    portrait_frames,
                    self.assets.current.as_ref(),
                    self.assets.permanent.as_ref(),
                    self.assets.font.as_ref(),
                )?;
                let (secondary_index, _) = self.secondary_sm.step(self.secondary.as_ref(), true)?;
                secondary.frames[secondary_index as usize].blit_at(&mut frame, WIDTH)?;
                lock_compositor(&shared).copy_to_vga(&frame)?;
                self.delay_ticks = 1;
                self.portrait_boundary_pending = true;
                Ok(())
            }
            MONTAGE_PHASE_FADE_OUT => {
                if self.fade_out >= 64 {
                    self.phase = MONTAGE_PHASE_COMPLETED;
                    return Ok(());
                }
                lock_compositor(&shared).palette_delta(0, 255, self.fade_out)?;
                self.fade_out += 1;
                self.delay_ms = 2;
                Ok(())
            }
            _ => Err(campaign_err!("ending: unknown montage phase")),
        }
    }

    /// NeedsRandomByte reports the sole recovered call site where 0x2c99c uses
    /// 0x4e893: portrait countdown reset.  Callers must not advance the native RNG
    /// on every rendered frame.
    ///
    /// Go：`montage_cycle.go:375`，`func (p *MontageCycle) NeedsRandomByte() bool`
    /// Go 版先檢查 nil 接收者；Rust 的 `self` 不會是 null，該分支不移植。
    pub fn needs_random_byte(&self) -> bool {
        self.phase == MONTAGE_PHASE_PORTRAIT && !self.portrait_boundary_pending && self.portrait_sm.countdown == 0
    }

    /// ObserveInputChange receives the raw condition observed at 0x2c950, not a
    /// named keyboard mapping.  The native branch is only polled while a portrait
    /// is being presented and, except for the final loop, causes middle portraits
    /// to be skipped after the current one finishes.
    ///
    /// Go：`montage_cycle.go:383`，`func (p *MontageCycle) ObserveInputChange() bool`
    /// Go 版先檢查 nil 接收者；Rust 的 `self` 不會是 null，該分支不移植。
    pub fn observe_input_change(&mut self) -> bool {
        if self.phase != MONTAGE_PHASE_PORTRAIT || !self.portrait_boundary_pending {
            return false;
        }
        if self.plan_index + 1 < self.plans.len() as i64 {
            self.skip_to_final = true;
        }
        true
    }

    /// Go：`montage_cycle.go:393`，`func (p *MontageCycle) renderFIGANI(frame figani.Frame) error`
    pub(crate) fn render_figani(&mut self, frame: &figani::Frame) -> Result<()> {
        let Some(shared) = self.compositor.as_ref() else {
            return Err(campaign_err!("ending: montage cycle is not running"));
        };
        let mut guard = lock_compositor(shared);
        let c = &mut *guard;
        c.offscreen.fill(0);
        frame.blit_at(&mut c.offscreen, WIDTH)?;
        let offscreen = std::mem::take(&mut c.offscreen);
        let copied = c.copy_to_vga(&offscreen);
        c.offscreen = offscreen;
        copied
    }

    /// Ready reports only completion of the standalone indexed cycle.  It does
    /// not grant campaign transition or generic input semantics.
    ///
    /// Go：`montage_cycle.go:403`，`func (p MontageCycle) Ready() bool`
    pub fn ready(&self) -> bool {
        self.phase == MONTAGE_PHASE_COMPLETED
    }
}

#[cfg(test)]
pub(crate) mod montage_cycle_test;
