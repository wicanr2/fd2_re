//! Go 來源：`remake/cmd/fd2/native_church_ui.go`（283 行）。
//!
//! 骨架：Go 檔的 16 個函式中產生 5 個（階段 0 清單），其餘標 `r8`，見 `PORTING.json`。
//! E 組另補移植 `stepNativeChurchUIPulseTick`（`r8`，`stepNativeChurchUILifecycle` 在教會主選單每幀呼叫）。
//! `drawNativeChurchUIJob`（R5）不移植；`job.drawn` 由呈現回報設定。

use super::*;

/// Go：`native_church_ui.go:27`
///
/// 不 derive `Debug`／`Clone`：欄位含 callback（`Box<dyn FnOnce>`）、檔案或亂數產生器等做不到的型別。
#[derive(Default)]
pub(crate) struct NativeChurchUIJob {
    pub(crate) service_menu: Option<Box<NativeServiceMenuPresentation>>,
    pub(crate) frames: Vec<Vec<u8>>,
    pub(crate) restore: Vec<u8>,
    pub(crate) frame: i64,
    pub(crate) drawn: bool,
    pub(crate) after: Option<Box<dyn FnOnce(&mut Game) + Send>>,
}

impl Game {
    /// Go：`native_church_ui.go:97`，`func (g *Game) composeNativeChurchScene() ([]byte, bool)`
    ///
    /// Go 把 `a.strings`／`a.font`（可為 nil）直接交給 `campaign.ComposeNativeChurchScene`；Rust 的
    /// campaign 函式收借用，所以 nil 時與 campaign 回錯誤一樣回 `false`。
    pub(crate) fn compose_native_church_scene(&self) -> (Vec<u8>, bool) {
        let Some(a) = self.native_class_ui.as_ref() else {
            return (Vec::new(), false);
        };
        if a.entries.len() <= 16 || self.native_church_text_index < 0 {
            return (Vec::new(), false);
        }
        let (Some(strings), Some(font)) = (a.strings.as_ref(), a.font.as_ref()) else {
            return (Vec::new(), false);
        };
        match campaign::compose_native_church_scene(
            &a.background,
            &a.entries[1],
            &a.dialogue,
            &a.digits,
            &a.portrait,
            strings,
            font,
            self.gold,
            self.native_church_text_index,
        ) {
            Ok(frame) => (frame, true),
            Err(_) => (Vec::new(), false),
        }
    }

    /// Go：`native_church_ui.go:120`，`func (g *Game) composeNativeChurchSourceFrame() ([]byte, bool)`
    pub(crate) fn compose_native_church_source_frame(&self) -> (Vec<u8>, bool) {
        let (scene, ok) = self.compose_native_church_scene();
        if !ok {
            return (Vec::new(), false);
        }
        match campaign::native_church_menu_base(&scene) {
            Ok(frame) => (frame, true),
            Err(_) => (Vec::new(), false),
        }
    }

    /// Go：`native_church_ui.go:152`，`func (g *Game) beginNativeChurchMenuOpening() bool`
    pub(crate) fn begin_native_church_menu_opening(&mut self) -> bool {
        let (scene, ok) = self.compose_native_church_scene();
        if !ok {
            return false;
        }
        // `compose_native_church_scene` 成功表示 `native_class_ui` 存在。
        let a = self.native_class_ui.as_ref().unwrap();
        let frames = match campaign::native_church_menu_transition_frames(&scene, &a.entries, true) {
            Ok(frames) if frames.len() == 4 => frames,
            _ => return false,
        };
        self.reset_native_church_ui_pulse();
        let mut job = NativeChurchUIJob { frames, ..Default::default() };
        job.service_menu = self.bind_modern_facility_service_menu(None, Some(&job), false).map(Box::new);
        self.native_church_ui_job = Some(job);
        true
    }

    /// Go：`native_church_ui.go:191`，`func (g *Game) stepNativeChurchUILifecycle(now time.Time)`
    pub(crate) fn step_native_church_ui_lifecycle(&mut self, now: GameInstant) {
        if let Some(job) = self.native_church_ui_job.as_mut()
            && job.drawn
        {
            job.drawn = false;
            let frames = job.frames.len() as i64;
            if job.frame < frames {
                job.frame += 1;
                if job.frame < frames || !job.restore.is_empty() {
                    return;
                }
            }
            if job.frame >= frames {
                let mut job = self.native_church_ui_job.take().unwrap();
                if let Some(after) = job.after.take() {
                    after(self);
                }
            }
        }
        if self.native_church_ui_job.is_none() && self.church_mode == "menu" && !self.native_church_ui_shot_hold {
            let tick = self.native_church_ui_clock.sample(now);
            self.step_native_church_ui_pulse_tick(tick);
        }
    }

    /// Go：`native_church_ui.go:248`，`func (g *Game) resetNativeChurchUIPulse()`
    pub(crate) fn reset_native_church_ui_pulse(&mut self) {
        self.native_church_ui_clock.reset();
        self.native_church_ui_pulse = 2;
        self.native_church_ui_last_tick = 0;
        self.native_church_ui_has_tick = false;
    }

    /// Go：`native_church_ui.go:255`，`func (g *Game) stepNativeChurchUIPulseTick(rawTick int)`（`r8`，E 組隨
    /// `stepNativeChurchUILifecycle` 補移植）
    pub(crate) fn step_native_church_ui_pulse_tick(&mut self, raw_tick: i64) {
        if !self.native_church_ui_has_tick {
            self.native_church_ui_last_tick = raw_tick;
            self.native_church_ui_has_tick = true;
            return;
        }
        let delta = (raw_tick as u16).wrapping_sub(self.native_church_ui_last_tick as u16) as i16;
        if delta < 2 {
            return;
        }
        self.native_church_ui_last_tick = raw_tick;
        self.native_church_ui_pulse = (self.native_church_ui_pulse + 1) & 3;
    }
}

#[cfg(test)]
mod native_church_ui_test;

// ---- R5 階段 1 骨架（tools/rust_r5_skeleton.sh 產生；本體由 R5 階段 2 移植）----

impl Game {
    /// Go：`native_church_ui.go:214`，`func (g *Game) drawNativeChurchUIJob(screen *ebiten.Image) bool`
    ///
    /// R5 階段 2 r5c。`drawModernHDFacilityServiceMenu`（HD，R6）在 `job.serviceMenu` 為 nil 時回 false；Rust 在忠實
    /// 模式不建立這份描述（KNOWN_DIFFERENCES `facility-hd-precompute-faithful-skip`），描述存在時 `todo!("r6:…")`。
    pub(crate) fn draw_native_church_ui_job(&mut self, screen: &mut Picture) -> bool {
        let Some(job) = self.native_church_ui_job.as_ref() else {
            return false;
        };
        if job.service_menu.is_some() {
            todo!("r6:modern_hd_service_menu_transition.go:drawModernHDFacilityServiceMenu");
        }
        let frame = job.frame;
        let pix = if frame < job.frames.len() as i64 {
            // Go 以負的 frame 索引切片會 panic；Rust 同樣 panic。
            job.frames[usize::try_from(frame).expect("drawNativeChurchUIJob：frame 為負")].clone()
        } else if job.restore.len() == 320 * 200 {
            job.restore.clone()
        } else {
            return false;
        };
        self.present_native_class_frame(screen, &pix);
        self.native_church_ui_job.as_mut().unwrap().drawn = true;
        true
    }
}
