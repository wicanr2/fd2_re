//! Go 來源：`remake/cmd/fd2/modern_hd_service_menu_transition.go`（252 行）。
//!
//! 骨架：Go 檔的 4 個函式中產生 2 個（階段 0 清單），其餘標 `r8`，見 `PORTING.json`。
//! HD 檔：型別只保留第一章忠實模式執行到的欄位，其餘欄位與函式在 R6。

use super::*;

/// Go：`modern_hd_service_menu_transition.go:14`
///
/// HD 型別：只保留第一章忠實模式下執行到的函式用到的欄位（120「階段 1 骨架的對應規則」），其餘欄位在 R6 補上。
///
/// 不 derive `Debug`／`Clone`：欄位含 callback（`Box<dyn FnOnce>`）、檔案或亂數產生器等做不到的型別。
pub(crate) struct NativeServiceMenuPresentation {
    pub(crate) owner: String,
    pub(crate) locale: String,
    pub(crate) variant: i64,
    pub(crate) resource: i64,
    pub(crate) closing: bool,
    pub(crate) font: Option<Font>,
    pub(crate) palette: [ColorRgba; 256],
    pub(crate) source: ModernChurchConfirmationSource,
    pub(crate) cells: [fdother::RawCell; 4],
    pub(crate) raw_frames: Vec<Vec<u8>>,
    pub(crate) chrome: Option<ModernHDDialogueChrome>,
    pub(crate) shop_job: Option<Box<NativeClassUIJob>>,
    pub(crate) church_job: Option<Box<NativeChurchUIJob>>,
    pub(crate) last_draw: Option<ModernChurchConfirmationDraw>,
}

impl Default for NativeServiceMenuPresentation {
    fn default() -> Self {
        Self {
            owner: Default::default(),
            locale: Default::default(),
            variant: Default::default(),
            resource: Default::default(),
            closing: Default::default(),
            font: Default::default(),
            palette: std::array::from_fn(|_| Default::default()),
            source: Default::default(),
            cells: Default::default(),
            raw_frames: Default::default(),
            chrome: Default::default(),
            shop_job: Default::default(),
            church_job: Default::default(),
            last_draw: Default::default(),
        }
    }
}

impl Game {
    /// Go：`modern_hd_service_menu_transition.go:31`，`func (g *Game) bindModernFacilityServiceMenu(shopJob *nativeClassUIJob, churchJob *nativeChurchUIJob, closing bool) *nativeServiceMenuPresentation`
    ///
    /// 簽名改寫：兩個工作收共享借用（Go 只讀 `frames`／`restore`，另存指標供 HD 繪製比對身分）。Rust 的
    /// `shop_job`／`church_job` 欄位不存工作本身（工作由 `Game` 擁有），留 `None` 給 R6 決定身分比對方式。
    ///
    /// 忠實模式：Go 在 TTF 已載入時（與 HD 介面開關無關）就預先建立這份 HD 描述，只有 HD 繪製
    /// （`drawModernHDFacilityServiceMenu`，R6）與輸入稽核的 `lastDraw`（只由 HD 繪製寫入）讀它。建立過程要以
    /// TTF 擷取 HD 文字（`modernHDFacilityFont`，R6），所以 Rust 在 HD 介面關閉時於 Go 的前置檢查之後回 `None`
    /// （KNOWN_DIFFERENCES `facility-hd-precompute-faithful-skip`），流程、節點與交易不受影響。
    pub(crate) fn bind_modern_facility_service_menu(
        &mut self,
        shop_job: Option<&NativeClassUIJob>,
        church_job: Option<&NativeChurchUIJob>,
        closing: bool,
    ) -> Option<NativeServiceMenuPresentation> {
        let class_ui = self.native_class_ui.as_ref()?;
        let font = self.font.as_ref()?;
        if font.sf.is_none() || font.indexed_hd.is_some() {
            return None;
        }
        let node = self.camp.as_ref()?.node()?;
        if self.native_shop_ui_job.is_some()
            || self.native_class_ui_job.is_some()
            || self.native_church_ui_job.is_some()
        {
            return None;
        }
        let mut d = NativeServiceMenuPresentation {
            owner: node.r#type.clone(),
            locale: self.locale_id.clone(),
            font: Some(font.clone()),
            closing,
            ..Default::default()
        };
        let compose: fn(&mut Game) -> (Vec<u8>, bool);
        let (frames, restore): (&[Vec<u8>], &[u8]);
        match d.owner.as_str() {
            "shop" => {
                let (a, _, _, ok) = self.native_shop_state();
                let expected = [(1, 12), (3, 29), (5, 63)]
                    .iter()
                    .find(|(v, _)| *v == self.native_shop_variant)
                    .map_or(0, |(_, r)| *r);
                let a = a.filter(|_| ok)?;
                let shop_job = shop_job?;
                if church_job.is_some()
                    || self.native_shop_mode != "menu"
                    || node.native_hub_variant != self.native_shop_variant
                    || a.resource_id != expected
                {
                    return None;
                }
                (d.variant, d.resource) = (self.native_shop_variant, a.resource_id);
                compose = |g| g.compose_native_shop_stable();
                for i in 0..d.cells.len() {
                    d.cells[i] = a.service_cells[i][0].clone();
                }
                (frames, restore) = (&shop_job.frames, &shop_job.restore);
            }
            "church" => {
                let church_job = church_job?;
                if shop_job.is_some() || self.church_mode != "menu" || class_ui.entries.len() <= 10 {
                    return None;
                }
                d.resource = 14;
                compose = |g| g.compose_native_church_source_frame();
                for i in 0..d.cells.len() {
                    let c = &class_ui.entries[3 + 2 * i];
                    d.cells[i] = fdother::RawCell { width: c.width, height: c.height, pixels: c.pixels.clone() };
                }
                (frames, restore) = (&church_job.frames, &church_job.restore);
            }
            _ => return None,
        }
        if d.cells.iter().any(|c| c.width != 24 || c.height != 20 || c.pixels.len() != 480) {
            return None;
        }
        if !self.modern_hd_interface_enabled() {
            return None;
        }
        let (frames, restore) = (frames.to_vec(), restore.to_vec());
        let (palette, ok) = preparation_cursor_palette(&class_ui.palette);
        if !ok {
            return None;
        }
        d.palette.copy_from_slice(&palette);
        let owner = d.owner.clone();
        let (source, ok) = self.capture_modern_facility_confirmation_source(&owner, &mut |g: &mut Game| compose(g));
        if !ok {
            return None;
        }
        d.source = source;
        let chrome = match self.prepare_modern_hd_facility_dialogue() {
            Ok(Some(h)) if facility_dialogue_border_visible(&d.source.raw, &h.borders[1]) => h.chrome.clone(),
            _ => return None,
        };
        d.chrome = chrome;
        if frames.len() != 4 || closing && restore != d.source.raw || !closing && !restore.is_empty() {
            return None;
        }
        for (i, frame) in frames.iter().enumerate() {
            let (raw, _) = d.compose(i as i64, false);
            if raw.pixels != *frame {
                return None;
            }
            d.raw_frames.push(frame.clone());
        }
        Some(d)
    }
}

impl NativeServiceMenuPresentation {
    /// All four entries are normal cells. Preserve native signed division and order.
    ///
    /// Go：`modern_hd_service_menu_transition.go:101`，`func (d *nativeServiceMenuPresentation) compose(index int, hd bool) (p modernChurchConfirmationFrame, ok bool)`
    /// Go 檢查接收者是否為 nil；Rust 的 `self` 不會是 null，該分支不移植。
    ///
    /// 簽名改寫：`&self`（Go 只讀接收者）。
    pub(crate) fn compose(&self, index: i64, hd: bool) -> (ModernChurchConfirmationFrame, bool) {
        let mut p = ModernChurchConfirmationFrame::default();
        if !(0..=4).contains(&index) || index == 4 && !self.closing {
            return (p, false);
        }
        (p.pixels, p.mask) = (self.source.raw.clone(), self.source.mask.clone());
        if hd {
            p.pixels = self.source.hd.clone();
            p.runs = self.source.runs.clone();
            p.panels = vec![ModernHDDialoguePanel {
                chrome: self.chrome.clone(),
                rect: Rectangle { min: Point { x: 5, y: 112 }, max: Point { x: 315, y: 198 } },
                gold: true,
                ..Default::default()
            }];
        }
        p.name = String::from("restore");
        if index == 4 {
            return (p, true);
        }
        p.name = String::from("menu");
        let divisor = if self.closing { index + 1 } else { 4 - index };
        for (option, offset) in [-39i64, -13, 13, 39].into_iter().enumerate() {
            let cell = &self.cells[option];
            let x0 = 240 + offset / divisor;
            if cell.blit_at(&mut p.pixels, 320, x0, 169).is_err() {
                return (p, false);
            }
            for y in 0..20i64 {
                for x in 0..24i64 {
                    if cell.pixels[(y * 24 + x) as usize] != 0 {
                        p.mask[((169 + y) * 320 + x0 + x) as usize] = 255;
                    }
                }
            }
        }
        (p, true)
    }
}
