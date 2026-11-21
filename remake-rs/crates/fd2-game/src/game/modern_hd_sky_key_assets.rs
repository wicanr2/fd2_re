//! 天空之鑰場景與開場共用圖庫，對應 Go `modern_hd_sky_key_assets.go`。

use std::collections::VecDeque;
use std::io::Cursor;
use std::path::{Path, PathBuf};

use serde::Deserialize;
use sha2::{Digest as _, Sha256};

use super::*;

const CACHE_SIZE: usize = 3;

#[derive(Clone, Copy, Debug, PartialEq, Eq)]
enum Source {
    Fd34,
    Ani0,
}

impl Source {
    fn metadata(self) -> &'static str {
        match self {
            Self::Fd34 => "animations/fdother_034_ch20_sky_key/bank.json",
            Self::Ani0 => "animations/ANI_000/animation.json",
        }
    }

    fn offset(self) -> usize {
        usize::from(self == Self::Fd34)
    }
}

#[derive(Debug, Clone, Copy, Default, Deserialize, PartialEq, Eq)]
#[serde(default)]
struct Descriptor {
    x: i64,
    y: i64,
    width: i64,
    height: i64,
    delay_native: i64,
    raw_byte_4: u8,
    raw_byte_5: u8,
    raw_byte_7: u8,
}

impl Descriptor {
    fn hd_size(self) -> (i64, i64) {
        (self.width * 6, self.height * 6)
    }
}

const ANI_CANVAS: Descriptor =
    Descriptor { x: 0, y: 0, width: 320, height: 200, delay_native: 0, raw_byte_4: 0, raw_byte_5: 0, raw_byte_7: 0 };

#[derive(Debug, Deserialize)]
struct SourceRow {
    index: usize,
    #[serde(flatten)]
    rect: Descriptor,
    #[serde(rename = "native_canvas_sha256", default)]
    canvas: String,
    #[serde(rename = "native_canvas_ff_sha256", default)]
    canvas_ff: String,
    #[serde(default)]
    pixel_sha256: String,
    #[serde(default)]
    dac_sha256: String,
    #[serde(with = "battle::go_bytes")]
    source_colours: Vec<u8>,
    #[serde(default)]
    blank: bool,
}

#[derive(Deserialize)]
struct SourcePlan {
    schema_version: u32,
    kind: String,
    source_metadata_sha256: String,
    frames: Vec<SourceRow>,
}

#[derive(Deserialize)]
pub(crate) struct SkyKeyAsset {
    source_archive: String,
    source_resource: u16,
    status: String,
    consumer_contract: String,
    layout_policy: String,
    alpha_contract: String,
    width: i64,
    height: i64,
    frame_count: usize,
    files: Vec<String>,
    frame_sha256: Vec<String>,
    native_frames: Vec<Descriptor>,
    master_file: String,
    source_files: Vec<String>,
    source_sha256: Vec<String>,
    source_metadata_sha256: String,
}

#[derive(Debug)]
struct CheckedFile {
    path: PathBuf,
    sha: String,
}

impl CheckedFile {
    fn read(&self) -> Result<Vec<u8>> {
        read_verified_theme_file(&self.path, &self.sha)
    }
}

#[derive(Debug)]
struct FrameFile {
    file: CheckedFile,
    size: (i64, i64),
}

impl FrameFile {
    fn read(&self) -> Result<Vec<u8>> {
        let raw = self.file.read()?;
        let reader = png::Decoder::new(Cursor::new(&raw))
            .read_info()
            .map_err(|err| game_err!("modern sky key: {}: {err}", self.file.path.display()))?;
        if (i64::from(reader.info().width), i64::from(reader.info().height)) != self.size {
            return Err(game_err!("modern sky key: PNG geometry changed"));
        }
        Ok(raw)
    }
}

#[derive(Debug)]
struct Cached {
    hash: String,
    picture: Arc<Picture>,
    transparent: bool,
    opaque: bool,
    partial: bool,
    nonblack: bool,
}

impl Cached {
    fn new(hash: String, picture: Picture) -> Self {
        let mut result = Self {
            hash,
            picture: Arc::new(picture),
            transparent: false,
            opaque: false,
            partial: false,
            nonblack: false,
        };
        for px in result.picture.read_pixels().as_chunks::<4>().0 {
            result.transparent |= px[3] == 0;
            result.opaque |= px[3] == 255;
            result.partial |= px[3] != 0 && px[3] != 255;
            result.nonblack |= px[..3] != [0, 0, 0];
        }
        result
    }

    fn matches(&self, source: Source, blank: bool) -> bool {
        match source {
            Source::Fd34 => self.transparent && self.opaque,
            Source::Ani0 => self.opaque && !self.transparent && !self.partial && blank != self.nonblack,
        }
    }
}

#[derive(Debug)]
pub(crate) struct SkyKeyBank {
    source: Source,
    plan_file: CheckedFile,
    rows: Vec<SourceRow>,
    files: Vec<FrameFile>,
    recent: VecDeque<Cached>,
    decoded_sha: BTreeMap<String, [u8; 32]>,
    rejected: bool,
    notice: Option<String>,
}

#[derive(Debug, Default)]
pub(crate) struct SkyKeyAssets {
    pub(crate) fd34: Option<SkyKeyBank>,
    pub(crate) ani0: Option<SkyKeyBank>,
}

impl SkyKeyAssets {
    pub(crate) fn admit(&mut self, asset: SkyKeyAsset, root: &Path, original: &Path) -> Result<()> {
        let (source, kind, count, target) =
            match (asset.source_archive.as_str(), asset.source_resource, asset.consumer_contract.as_str()) {
                ("FDOTHER.DAT", 34, "sky_key_fdother34_rgba_hd6_v1") => {
                    (Source::Fd34, "sky_key_fdother34_hd6", 100, &mut self.fd34)
                }
                ("ANI.DAT", 0, "sky_key_ani0_rgba_hd6_v1") => (Source::Ani0, "sky_key_ani0_hd6", 96, &mut self.ani0),
                _ => return Err(game_err!("modern sky key: unsupported source")),
            };
        if target.is_some() {
            return Err(game_err!("modern sky key: duplicate source"));
        }
        let bank = asset.load(source, kind, count, root, original)?;
        *target = Some(bank);
        Ok(())
    }

    /// 第二十章使用的原版畫格、遮罩與色盤再次核對。回傳 false 時改走原版演出。
    pub(crate) fn preflight(&self, frames: &[fdother::Frame], clip: &afm::Clip) -> Result<bool> {
        let (fd34, ani0) = match (&self.fd34, &self.ani0) {
            (None, None) => return Ok(false),
            (Some(fd34), Some(ani0)) => (fd34, ani0),
            _ => return Err(game_err!("modern sky key: incomplete animation")),
        };
        if fd34.rejected || ani0.rejected {
            return Ok(false);
        }
        if frames.len() != fd34.rows.len()
            || clip.indexed_frames.len() != ani0.rows.len()
            || clip.palettes.len() != ani0.rows.len()
        {
            return Err(game_err!("modern sky key: native banks incomplete"));
        }
        fd34.recheck_files()?;
        ani0.recheck_files()?;
        for (row, frame) in fd34.rows.iter().zip(frames) {
            row.validate_fdother(frame)?;
        }
        for ((row, pixels), palette) in ani0.rows.iter().zip(&clip.indexed_frames).zip(&clip.palettes) {
            row.validate_ani(pixels, palette)?;
        }
        Ok(true)
    }
}

impl SkyKeyAsset {
    fn load(self, source: Source, kind: &str, count: usize, root: &Path, original: &Path) -> Result<SkyKeyBank> {
        if self.frame_count != count
            || self.files.len() != count
            || self.frame_sha256.len() != count
            || self.native_frames.len() != count
            || self.source_files.is_empty()
            || self.source_files.len() != self.source_sha256.len()
            || !matches!(self.status.as_str(), "runtime_candidate" | "runtime_ready")
            || self.layout_policy != "native_frame_xy_hd6"
            || self.alpha_contract != "coverage"
        {
            return Err(game_err!("modern sky key: invalid bank contract"));
        }
        read_verified_theme_file(&original.join(source.metadata()), &self.source_metadata_sha256)?;
        let (plan_file, rows) = self.read_plan(kind, count + source.offset(), root)?;
        let mut files = Vec::with_capacity(count);
        let (mut max_width, mut max_height) = (0, 0);
        for (index, row) in rows.iter().enumerate() {
            if row.index != index
                || row.source_colours.is_empty()
                || !row.source_colours.windows(2).all(|w| w[0] < w[1])
            {
                return Err(game_err!("modern sky key: invalid source row {index}"));
            }
            let Some(i) = index.checked_sub(source.offset()) else { continue };
            let descriptor = self.native_frames[i];
            match source {
                Source::Fd34 => {
                    let r = row.rect;
                    let want = Descriptor { x: r.x, y: r.y, width: r.width, height: r.height, ..Descriptor::default() };
                    if descriptor != want
                        || !(0..320).contains(&r.x)
                        || !(0..200).contains(&r.y)
                        || !(1..=320 - r.x).contains(&r.width)
                        || !(1..=200 - r.y).contains(&r.height)
                    {
                        return Err(game_err!("modern sky key: native FDOTHER descriptor changed at {index}"));
                    }
                }
                Source::Ani0 if descriptor != ANI_CANVAS => {
                    return Err(game_err!("modern sky key: native ANI geometry changed at {index}"));
                }
                Source::Ani0 => {}
            }
            let name = &self.files[i];
            if theme_file_extension(name) != Some("png") {
                return Err(game_err!("modern sky key: unsafe PNG path"));
            }
            let file = CheckedFile { path: root.join(name), sha: self.frame_sha256[i].clone() };
            let size = descriptor.hd_size();
            let frame = FrameFile { file, size };
            frame.read()?;
            files.push(frame);
            max_width = max_width.max(size.0);
            max_height = max_height.max(size.1);
        }
        if (self.width, self.height) != (max_width, max_height) {
            return Err(game_err!("modern sky key: bank bounds changed"));
        }
        Ok(SkyKeyBank {
            source,
            plan_file,
            rows,
            files,
            recent: VecDeque::with_capacity(CACHE_SIZE),
            decoded_sha: BTreeMap::new(),
            rejected: false,
            notice: None,
        })
    }

    fn read_plan(&self, kind: &str, row_count: usize, root: &Path) -> Result<(CheckedFile, Vec<SourceRow>)> {
        let mut seen = BTreeSet::new();
        let mut plan = None;
        for (name, sha) in self.source_files.iter().zip(&self.source_sha256) {
            if !seen.insert(name) {
                return Err(game_err!("modern sky key: duplicate provenance"));
            }
            match theme_file_extension(name) {
                Some("png") => {}
                Some("json") if plan.is_none() => {
                    let file = CheckedFile { path: root.join(name), sha: sha.clone() };
                    let data: SourcePlan = serde_json::from_slice(&file.read()?)
                        .map_err(|err| game_err!("modern sky key: invalid source plan {name}: {err}"))?;
                    if data.schema_version != 1
                        || data.kind != kind
                        || data.source_metadata_sha256 != self.source_metadata_sha256
                        || data.frames.len() != row_count
                    {
                        return Err(game_err!("modern sky key: source plan changed"));
                    }
                    plan = Some((file, data.frames));
                }
                _ => return Err(game_err!("modern sky key: invalid provenance")),
            }
        }
        if !seen.contains(&self.master_file) {
            return Err(game_err!("modern sky key: master absent"));
        }
        plan.ok_or_else(|| game_err!("modern sky key: source plan absent"))
    }
}

impl SkyKeyBank {
    /// `None` 代表像素或黑幕條件不符，呼叫端可退回原版。
    pub(crate) fn picture(&mut self, index: usize) -> Result<Option<Arc<Picture>>> {
        let file = index
            .checked_sub(self.source.offset())
            .and_then(|i| self.files.get(i))
            .ok_or_else(|| game_err!("modern sky key: frame outside bank"))?;
        let raw = file.read()?;
        if self.rejected {
            return Ok(None);
        }
        let cached = if let Some(i) = self.recent.iter().position(|c| c.hash == file.file.sha) {
            self.recent.remove(i).unwrap()
        } else {
            let Some(picture) = decode_precise_png_picture(&raw).filter(|p| (p.width, p.height) == file.size) else {
                return Ok(self.reject(index));
            };
            let digest = Sha256::digest(picture.read_pixels()).into();
            let hash = file.file.sha.clone();
            let previous = self.decoded_sha.entry(hash.clone()).or_insert(digest);
            if *previous != digest {
                return Err(game_err!("modern sky key: decoded frame changed"));
            }
            Cached::new(hash, picture)
        };
        if (cached.picture.width, cached.picture.height) != file.size {
            return Err(game_err!("modern sky key: cached frame geometry changed"));
        }
        if !cached.matches(self.source, self.rows[index].blank) {
            return Ok(self.reject(index));
        }
        let picture = cached.picture.clone();
        if self.recent.len() == CACHE_SIZE {
            self.recent.pop_front();
        }
        self.recent.push_back(cached);
        Ok(Some(picture))
    }

    fn reject(&mut self, index: usize) -> Option<Arc<Picture>> {
        self.rejected = true;
        self.recent.clear();
        self.notice = Some(format!(
            "modern sky key: frame {index} cannot be decoded or has invalid coverage or native black frame"
        ));
        None
    }

    pub(crate) fn take_notice(&mut self) -> Option<String> {
        self.notice.take()
    }

    fn recheck_files(&self) -> Result<()> {
        self.plan_file.read()?;
        for entry in &self.files {
            entry.file.read()?;
        }
        Ok(())
    }
}

impl SourceRow {
    fn validate_fdother(&self, frame: &fdother::Frame) -> Result<()> {
        let r = self.rect;
        if (r.x, r.y, r.width, r.height) != (frame.x, frame.y, frame.width, frame.height) {
            return Err(game_err!("FDOTHER #34 frame {}: source geometry changed", self.index));
        }
        let mut zeros = vec![0; 64000];
        let mut ff = vec![255; 64000];
        frame.blit(&mut zeros, 320, -1)?;
        frame.blit(&mut ff, 320, -1)?;
        let used = zeros.iter().copied().zip(&ff).filter_map(|(a, &b)| (a == b).then_some(a));
        if format!("{:x}", Sha256::digest(&zeros)) != self.canvas
            || format!("{:x}", Sha256::digest(&ff)) != self.canvas_ff
            || used_colours(used).ne(self.source_colours.iter().copied())
        {
            return Err(game_err!("FDOTHER #34 frame {}: source pixels, mask or colours changed", self.index));
        }
        Ok(())
    }

    fn validate_ani(&self, pixels: &[u8], palette: &[u8]) -> Result<()> {
        if pixels.len() != 64000
            || palette.len() != 768
            || format!("{:x}", Sha256::digest(pixels)) != self.pixel_sha256
            || format!("{:x}", Sha256::digest(palette)) != self.dac_sha256
            || used_colours(pixels.iter().copied()).ne(self.source_colours.iter().copied())
        {
            return Err(game_err!("ANI #0 frame {}: native source changed", self.index));
        }
        Ok(())
    }
}

fn used_colours(pixels: impl Iterator<Item = u8>) -> impl Iterator<Item = u8> {
    let mut seen = [false; 256];
    for pixel in pixels {
        seen[usize::from(pixel)] = true;
    }
    (0..=255u8).filter(move |&i| seen[usize::from(i)])
}

#[cfg(test)]
#[path = "modern_hd_sky_key_assets_test.rs"]
mod tests;
