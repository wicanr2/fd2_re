//! HD 開場清冊載入與逐格解碼，對應 Go `modern_hd_opening.go`。
//! ANI #0 由天空之鑰圖庫提供，完整主題載入時需另外驗證。

use std::io::Cursor;
use std::path::Path;

use serde::Deserialize;
use sha2::{Digest as _, Sha256};

use super::modern_hd_sky_key_assets::{SkyKeyAsset, SkyKeyAssets};
use super::*;

#[derive(Debug, Clone, Copy, PartialEq, Eq, PartialOrd, Ord, Deserialize)]
pub(crate) enum OpeningArchive {
    #[serde(rename = "ANI.DAT")]
    Ani,
    #[serde(rename = "FDOTHER.DAT")]
    Fdother,
}

impl OpeningArchive {
    fn name(self) -> &'static str {
        match self {
            Self::Ani => "ANI.DAT",
            Self::Fdother => "FDOTHER.DAT",
        }
    }
}

#[derive(Debug, Clone, Copy, PartialEq, Eq, PartialOrd, Ord, Deserialize)]
pub(crate) struct OpeningSource {
    pub(crate) source_archive: OpeningArchive,
    pub(crate) source_resource: u16,
}

impl OpeningSource {
    fn shape(self) -> Option<(u32, u32, usize)> {
        use OpeningArchive::{Ani, Fdother};
        let (height, count) = match (self.source_archive, self.source_resource) {
            (Ani, 1) => (1200, 51),
            (Ani, 3) => (1200, 28),
            (Ani, 4 | 6 | 8) => (1200, 12),
            (Ani, 5) => (1200, 35),
            (Ani, 7) => (1200, 17),
            (Fdother, 69) => (4410, 2),
            (Fdother, 7 | 74 | 75 | 100) => (1200, 1),
            _ => return None,
        };
        Some((1920, height, count))
    }

    fn metadata_digest(self, original: &Path) -> Result<String> {
        let resource = self.source_resource;
        let paths = match (self.source_archive, resource) {
            (OpeningArchive::Ani, _) => vec![format!("animations/ANI_{resource:03}/animation.json")],
            (OpeningArchive::Fdother, 69) => (69..=73)
                .map(|i| format!("surfaces/FDOTHER_{i:03}/resource.json"))
                .chain(["palette/fdother_101.json".into(), "palette/fdother_102.json".into()])
                .collect(),
            (OpeningArchive::Fdother, _) => {
                let (surface, palette) = match resource {
                    7 => ("surfaces/FDOTHER_007/entry_000/resource.json".into(), 8),
                    74 | 75 => (format!("surfaces/FDOTHER_{resource:03}/resource.json"), 76),
                    100 => ("surfaces/FDOTHER_100/resource.json".into(), 99),
                    _ => return Err(game_err!("modern opening: unknown original source")),
                };
                vec![surface, format!("palette/fdother_{palette:03}.json")]
            }
        };
        let mut digest = Sha256::new();
        for path in paths {
            let raw = read_theme_file(&original.join(&path))?;
            digest.update(format!("{path} {:x}\n", Sha256::digest(raw)));
        }
        Ok(format!("{:x}", digest.finalize()))
    }
}

#[derive(Deserialize)]
#[serde(tag = "role")]
enum CatalogEntry {
    #[serde(rename = "opening_presentation_hd_set")]
    Opening(Box<OpeningAsset>),
    #[serde(rename = "story_animation_hd_set")]
    SkyKey(Box<SkyKeyAsset>),
    #[serde(other)]
    Other,
}

#[derive(Deserialize)]
struct OpeningAsset {
    #[serde(flatten)]
    source: OpeningSource,
    width: u32,
    height: u32,
    frame_count: usize,
    status: String,
    consumer_contract: String,
    layout_policy: String,
    alpha_contract: String,
    files: Vec<String>,
    frame_sha256: Vec<String>,
    master_file: String,
    source_files: Vec<String>,
    source_sha256: Vec<String>,
    source_metadata_sha256: String,
}

#[derive(Deserialize)]
struct Derivation {
    schema_version: u32,
    kind: String,
    #[serde(flatten)]
    source: OpeningSource,
    source_metadata_sha256: String,
    frames: Vec<serde::de::IgnoredAny>,
}

#[derive(Debug)]
struct EncodedFrame {
    data: FrameData,
    sha256: String,
}

/// 正式包只記路徑：載入時已核對 SHA-256 與 PNG 標頭，用到時重讀並再驗一次 SHA-256，
/// 不常駐壓縮檔。測試可以直接放位元組。
#[derive(Debug, Clone)]
enum FrameData {
    File(Arc<Path>),
    #[cfg_attr(not(test), allow(dead_code))]
    Memory(Arc<[u8]>),
}

impl FrameData {
    /// 檔案不見或內容改變是硬性錯誤；解碼或不透明檢查失敗回 `Ok(None)`。
    fn decode(&self, sha256: &str, verified: bool) -> Result<Option<Picture>> {
        Ok(match self {
            FrameData::File(path) => decode_opening_frame(&read_verified_theme_file(path, sha256)?, verified),
            FrameData::Memory(bytes) => decode_opening_frame(bytes, verified),
        })
    }
}

#[derive(Debug)]
struct OpeningBank {
    shape: (u32, u32),
    frames: Vec<EncodedFrame>,
    verified: BTreeSet<String>,
    rejected: bool,
}

/// 完整的開場圖組。只記路徑與 SHA-256，解碼後保留目前一格，另在背景預解下一格。
#[derive(Debug, Default)]
pub(crate) struct ModernOpening {
    banks: BTreeMap<OpeningSource, OpeningBank>,
    pub(crate) sky_key: SkyKeyAssets,
    current: Option<(String, Arc<Picture>)>,
    view: Option<(OpeningFrame, Arc<Picture>)>,
    notices: Vec<String>,
    /// 目前這格之後的下一格，在背景執行緒先解碼。只留一格。
    prefetch: Option<OpeningPrefetch>,
}

#[derive(Debug)]
struct OpeningPrefetch {
    sha256: String,
    decoded: std::thread::JoinHandle<Result<Option<Picture>>>,
}

/// 已通過不透明檢查的畫格只需解碼；第一次見到的要連不透明一起驗。
fn decode_opening_frame(bytes: &[u8], verified: bool) -> Option<Picture> {
    if verified { decode_png_picture(bytes) } else { decode_opaque_png_picture(bytes) }
}

#[derive(Debug, Clone, Copy, PartialEq)]
struct OpeningFrame {
    source: OpeningSource,
    index: usize,
    scroll: f64,
    factor: f32,
    tint: [u8; 3],
}

impl OpeningFrame {
    fn fdother(source_resource: u16) -> Self {
        Self {
            source: OpeningSource { source_archive: OpeningArchive::Fdother, source_resource },
            index: 0,
            scroll: 0.0,
            factor: 1.0,
            tint: [0; 3],
        }
    }
}

impl OpeningAsset {
    fn admit(self, root: &Path, original: &Path) -> Result<OpeningBank> {
        let Some((width, height, count)) = self.source.shape() else {
            return Err(game_err!("modern opening: unknown original source"));
        };
        if (self.width, self.height, self.frame_count) != (width, height, count)
            || self.files.len() != count
            || self.frame_sha256.len() != count
            || self.source_files.is_empty()
            || self.source_files.len() != self.source_sha256.len()
            || !matches!(self.status.as_str(), "runtime_candidate" | "runtime_ready")
            || self.consumer_contract != "opening_native_schedule_rgba_hd6_v1"
            || self.layout_policy != "full_native_canvas_hd6"
            || self.alpha_contract != "opaque"
        {
            return Err(game_err!("modern opening: invalid bank contract"));
        }
        let digest = self.source.metadata_digest(original)?;
        if digest != self.source_metadata_sha256 {
            return Err(game_err!("modern opening: original source metadata differs"));
        }
        self.verify_provenance(root, &digest)?;

        let mut paths = BTreeSet::new();
        let mut checked = BTreeSet::new();
        let mut frames = Vec::with_capacity(count);
        for (name, sha256) in self.files.into_iter().zip(self.frame_sha256) {
            if theme_file_extension(&name) != Some("png") || !paths.insert(name.clone()) {
                return Err(game_err!("modern opening: unsafe or duplicate frame path"));
            }
            let path: Arc<Path> = root.join(&name).into();
            let bytes = read_verified_theme_file(&path, &sha256)?;
            if checked.insert(sha256.clone()) {
                let reader = png::Decoder::new(Cursor::new(&bytes))
                    .read_info()
                    .map_err(|err| game_err!("modern opening: invalid PNG header for {name}: {err}"))?;
                if (reader.info().width, reader.info().height) != (width, height) {
                    return Err(game_err!("modern opening: frame geometry differs for {name}"));
                }
            }
            frames.push(EncodedFrame { data: FrameData::File(path), sha256 });
        }
        Ok(OpeningBank { shape: (width, height), frames, verified: BTreeSet::new(), rejected: false })
    }

    fn verify_provenance(&self, root: &Path, digest: &str) -> Result<()> {
        let mut seen = BTreeSet::new();
        let mut plans = 0;
        for (name, sha256) in self.source_files.iter().zip(&self.source_sha256) {
            let extension = theme_file_extension(name);
            if !matches!(extension, Some("json" | "png")) || !seen.insert(name) {
                return Err(game_err!("modern opening: unsafe or duplicate provenance"));
            }
            let bytes = read_verified_theme_file(&root.join(name), sha256)?;
            if extension == Some("json") {
                plans += 1;
                let plan: Derivation =
                    serde_json::from_slice(&bytes).map_err(|err| game_err!("modern opening: {name}: {err}"))?;
                if plan.schema_version != 1
                    || plan.kind != "opening_hd_art_derivation"
                    || plan.source != self.source
                    || plan.source_metadata_sha256 != digest
                    || plan.frames.len() != self.frame_count
                {
                    return Err(game_err!("modern opening: derivation plan differs"));
                }
            }
        }
        if plans != 1 || !seen.contains(&self.master_file) {
            return Err(game_err!("modern opening: master or unique plan absent"));
        }
        Ok(())
    }
}

impl ModernOpening {
    pub(crate) fn load(catalog: &Path, root: &Path, original: &Path) -> Result<Option<Self>> {
        let mut opening = Self::default();
        for entry in read_theme_catalog(catalog)? {
            match entry {
                CatalogEntry::Opening(asset) => {
                    if opening.banks.contains_key(&asset.source) {
                        return Err(game_err!("modern opening: duplicate source"));
                    }
                    opening.banks.insert(asset.source, asset.admit(root, original)?);
                }
                CatalogEntry::SkyKey(asset) => opening.sky_key.admit(*asset, root, original)?,
                CatalogEntry::Other => {}
            }
        }
        match opening.banks.len() {
            0 => Ok(None),
            12 if opening.sky_key.ani0.is_some() => Ok(Some(opening)),
            _ => Err(game_err!("modern opening: scene group incomplete")),
        }
    }

    /// `None` 表示整組的像素檢查曾失敗。上層可以畫同一格原版圖。
    pub(crate) fn frame(&mut self, source: OpeningSource, index: usize) -> Result<Option<Arc<Picture>>> {
        if source.source_archive == OpeningArchive::Ani && source.source_resource == 0 {
            return self
                .sky_key
                .ani0
                .as_mut()
                .ok_or_else(|| game_err!("modern opening: shared ANI0 unavailable"))?
                .picture(index);
        }
        let bank =
            self.banks.get_mut(&source).ok_or_else(|| game_err!("modern opening: source outside admitted banks"))?;
        let frame = bank.frames.get(index).ok_or_else(|| game_err!("modern opening: frame outside admitted bank"))?;
        if bank.rejected {
            return Ok(None);
        }
        if let Some((sha256, picture)) = &self.current
            && *sha256 == frame.sha256
        {
            return Ok(Some(picture.clone()));
        }
        let prefetched = self
            .prefetch
            .take()
            .filter(|prefetch| prefetch.sha256 == frame.sha256)
            .and_then(|prefetch| prefetch.decoded.join().ok());
        let picture = match prefetched {
            Some(picture) => picture?,
            None => frame.data.decode(&frame.sha256, bank.verified.contains(&frame.sha256))?,
        };
        let Some(picture) = picture.filter(|p| (p.width, p.height) == (bank.shape.0.into(), bank.shape.1.into()))
        else {
            bank.rejected = true;
            self.notices.push(format!(
                "modern opening: {} #{} frame {index}: frame geometry or opacity differs",
                source.source_archive.name(),
                source.source_resource
            ));
            return Ok(None);
        };
        let picture = Arc::new(picture);
        bank.verified.insert(frame.sha256.clone());
        self.current = Some((frame.sha256.clone(), picture.clone()));
        if let Some(next) = bank.frames.get(index + 1).filter(|next| next.sha256 != frame.sha256) {
            let (data, sha256, verified) =
                (next.data.clone(), next.sha256.clone(), bank.verified.contains(&next.sha256));
            if let Ok(decoded) = std::thread::Builder::new()
                .name("fd2-opening-prefetch".into())
                .spawn(move || data.decode(&sha256, verified))
            {
                self.prefetch = Some(OpeningPrefetch { sha256: next.sha256.clone(), decoded });
            }
        }
        Ok(Some(picture))
    }

    pub(crate) fn take_notices(&mut self) -> Vec<String> {
        let mut notices = std::mem::take(&mut self.notices);
        notices.extend(self.sky_key.ani0.as_mut().and_then(|bank| bank.take_notice()));
        notices
    }

    fn rejected(&self, source: OpeningSource) -> bool {
        if source.source_archive == OpeningArchive::Ani && source.source_resource == 0 {
            self.sky_key.ani0.as_ref().is_some_and(|bank| bank.is_rejected())
        } else {
            self.banks.get(&source).is_some_and(|bank| bank.rejected)
        }
    }

    fn present(&mut self, plan: OpeningFrame) -> Result<Option<Arc<Picture>>> {
        if let Some((previous, view)) = &self.view
            && *previous == plan
        {
            return Ok(Some(view.clone()));
        }
        self.view = None;
        let Some(picture) = self.frame(plan.source, plan.index)? else { return Ok(None) };
        let view = if (picture.width, picture.height) == (HD_MAP_WIDTH, HD_MAP_HEIGHT) && plan.factor == 1.0 {
            picture
        } else {
            Arc::new(opening_viewport(&picture, plan))
        };
        self.view = Some((plan, view.clone()));
        Ok(Some(view))
    }
}

fn opening_viewport(picture: &Picture, plan: OpeningFrame) -> Picture {
    let stride = (HD_MAP_WIDTH * 4) as usize;
    let source = picture.read_pixels();
    let offset = (-canvas::adjust_destination_pixel((-plan.scroll * 6.0) as f32))
        .clamp(0.0, (picture.height - HD_MAP_HEIGHT) as f32);
    let top = offset.floor() as usize;
    let fraction = offset - top as f32;
    let rows = HD_MAP_HEIGHT as usize;
    let mut pix = source[top * stride..(top + rows) * stride].to_vec();
    if fraction != 0.0 {
        // 同一條 f32 算式先對全部 (a, b) 組合算好再查表，結果與逐位元組計算相同。
        let blend: Vec<u8> = (0..=255u8)
            .flat_map(|a| {
                (0..=255u8)
                    .map(move |b| (f32::from(a) * (1.0 - fraction) + f32::from(b) * fraction).round_ties_even() as u8)
            })
            .collect();
        let next = &source[(top + 1) * stride..(top + 1 + rows) * stride];
        for (a, &b) in pix.iter_mut().zip(next) {
            *a = blend[usize::from(*a) << 8 | usize::from(b)];
        }
    }
    if plan.factor != 1.0 {
        let factor = plan.factor;
        let tint = plan.tint.map(|value| f32::from(value) * (1.0 - factor));
        let tables = tint.map(|base| {
            let mut table = [0u8; 256];
            for (value, out) in table.iter_mut().enumerate() {
                *out = (f32::from(value as u8) * factor + base).round_ties_even() as u8;
            }
            table
        });
        for px in pix.as_chunks_mut::<4>().0 {
            for (channel, table) in px[..3].iter_mut().zip(&tables) {
                *channel = table[usize::from(*channel)];
            }
        }
    }
    Picture { width: HD_MAP_WIDTH, height: HD_MAP_HEIGHT, pix }
}

impl Game {
    /// 回傳是否由 HD 開場持有這個畫格。延後解碼拒收後，下一幀沿用原版圖。
    pub(crate) fn draw_modern_opening(&mut self, screen: &mut Picture) -> bool {
        if self.modern_story_portraits.is_none()
            || self.title_assets.is_none()
            || self.help_visible
            || self.startup_blocked
            || !matches!(self.title_phase.as_str(), "cutscene" | "menu")
        {
            return false;
        }
        let Some(opening) = &mut self.modern_opening else { return false };
        let mut plan = OpeningFrame::fdother(7);
        let mut red = false;
        if self.title_phase == "cutscene" {
            let Some(step) = CUT_SCRIPT.get(self.cut_idx as usize) else { return false };
            match step.kind.as_str() {
                "publisher" => {
                    plan.source.source_resource = 74;
                    plan.factor = title_publisher_brightness(self.cut_tick);
                }
                "static" => plan.source.source_resource = [100, 75][step.res as usize],
                "scroll" | "hold" => {
                    plan.source.source_resource = 69;
                    plan.scroll = self.scroll_y;
                    plan.index = usize::from(step.kind == "scroll" && title_scroll_uses_pulse_palette(self.scroll_y));
                }
                "redfade" => {
                    plan.source.source_resource = 69;
                    plan.factor = (40 - title_palette_phase(self.cut_tick)) as f32 / 40.0;
                    plan.tint = [255, 0, 0];
                }
                "redhold" => red = true,
                "titlefade" => {
                    plan.factor = title_palette_phase(self.cut_tick) as f32 / 40.0;
                    plan.tint = [0xe3, 0xf3, 255];
                }
                "afm" => {
                    plan.source =
                        OpeningSource { source_archive: OpeningArchive::Ani, source_resource: step.res as u16 };
                    plan.index = self.cut_frame as usize;
                }
                _ => return false,
            }
        }
        if opening.rejected(plan.source) {
            return false;
        }
        screen.clear();
        if red {
            screen.fill(ColorRgba { r: 255, g: 0, b: 0, a: 255 });
            return true;
        }
        match opening.present(plan) {
            Ok(Some(view)) => {
                if !self.modern_hd_separate_world {
                    let mut op = DrawImageOptions::default();
                    op.geo_m.scale(1.0 / 3.0, 1.0 / 3.0);
                    screen.draw_image(&view, &op);
                }
                self.modern_opening_view = Some(view);
            }
            Ok(None) => {
                for message in opening.take_notices() {
                    eprintln!("HD 開場延後檢查拒收，改用原版畫格：{message}");
                }
            }
            Err(err) => {
                self.startup_blocked = true;
                self.load_err = err.to_string();
            }
        }
        true
    }
}

#[cfg(test)]
#[path = "modern_opening_test.rs"]
mod tests;
