//! HD 開場清冊載入與逐格解碼，對應 Go `modern_hd_opening.go`。
//! ANI #0 由天空之鑰圖庫提供，完整主題載入時需另外驗證。

use std::io::Cursor;
use std::path::{Component, Path};

use serde::Deserialize;
use sha2::{Digest as _, Sha256};

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
            let raw = read_opening_file(&original.join(&path))?;
            digest.update(format!("{path} {:x}\n", Sha256::digest(raw)));
        }
        Ok(format!("{:x}", digest.finalize()))
    }
}

#[derive(Deserialize)]
struct Catalog {
    schema_version: u32,
    theme_id: String,
    assets: Vec<CatalogEntry>,
}

#[derive(Deserialize)]
#[serde(tag = "role")]
enum CatalogEntry {
    #[serde(rename = "opening_presentation_hd_set")]
    Opening(Box<OpeningAsset>),
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
    bytes: Arc<[u8]>,
    sha256: String,
}

#[derive(Debug)]
struct OpeningBank {
    shape: (u32, u32),
    frames: Vec<EncodedFrame>,
    verified: BTreeSet<String>,
    rejected: bool,
}

/// 完整的開場圖組。壓縮檔保留於記憶體，解碼後只保留目前一格。
#[derive(Debug, Default)]
pub(crate) struct ModernOpening {
    banks: BTreeMap<OpeningSource, OpeningBank>,
    current: Option<(String, Arc<Picture>)>,
    notices: Vec<String>,
}

fn asset_extension(name: &str) -> Option<&str> {
    let path = Path::new(name);
    let mut parts = path.components();
    match (parts.next(), parts.next()) {
        (Some(Component::Normal(base)), None) if base == name && !name.contains('\\') => path.extension()?.to_str(),
        _ => None,
    }
}

fn read_opening_file(path: &Path) -> Result<Vec<u8>> {
    std::fs::read(path).map_err(|err| game_err!("modern opening: {}: {err}", path.display()))
}

fn read_verified(path: &Path, sha256: &str) -> Result<Vec<u8>> {
    let bytes = read_opening_file(path)?;
    if format!("{:x}", Sha256::digest(&bytes)) != sha256 {
        return Err(game_err!("modern opening: SHA-256 differs for {}", path.display()));
    }
    Ok(bytes)
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
        let mut shared: BTreeMap<String, Arc<[u8]>> = BTreeMap::new();
        let mut frames = Vec::with_capacity(count);
        for (name, sha256) in self.files.into_iter().zip(self.frame_sha256) {
            if asset_extension(&name) != Some("png") || !paths.insert(name.clone()) {
                return Err(game_err!("modern opening: unsafe or duplicate frame path"));
            }
            let bytes = read_verified(&root.join(&name), &sha256)?;
            let bytes = match shared.entry(sha256.clone()) {
                std::collections::btree_map::Entry::Occupied(entry) => entry.get().clone(),
                std::collections::btree_map::Entry::Vacant(entry) => {
                    let reader = png::Decoder::new(Cursor::new(&bytes))
                        .read_info()
                        .map_err(|err| game_err!("modern opening: invalid PNG header for {name}: {err}"))?;
                    if (reader.info().width, reader.info().height) != (width, height) {
                        return Err(game_err!("modern opening: frame geometry differs for {name}"));
                    }
                    entry.insert(bytes.into()).clone()
                }
            };
            frames.push(EncodedFrame { bytes, sha256 });
        }
        Ok(OpeningBank { shape: (width, height), frames, verified: BTreeSet::new(), rejected: false })
    }

    fn verify_provenance(&self, root: &Path, digest: &str) -> Result<()> {
        let mut seen = BTreeSet::new();
        let mut plans = 0;
        for (name, sha256) in self.source_files.iter().zip(&self.source_sha256) {
            let extension = asset_extension(name);
            if !matches!(extension, Some("json" | "png")) || !seen.insert(name) {
                return Err(game_err!("modern opening: unsafe or duplicate provenance"));
            }
            let bytes = read_verified(&root.join(name), sha256)?;
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
        let catalog: Catalog = serde_json::from_slice(&read_opening_file(catalog)?)
            .map_err(|err| game_err!("modern theme catalog: {err}"))?;
        if catalog.schema_version != 1 || catalog.theme_id != MODERN_HANDPAINTED_THEME_ID {
            return Err(game_err!("modern theme catalog: unsupported identity"));
        }
        let mut opening = Self::default();
        for entry in catalog.assets {
            if let CatalogEntry::Opening(asset) = entry {
                if opening.banks.contains_key(&asset.source) {
                    return Err(game_err!("modern opening: duplicate source"));
                }
                opening.banks.insert(asset.source, asset.admit(root, original)?);
            }
        }
        match opening.banks.len() {
            0 => Ok(None),
            12 => Ok(Some(opening)),
            _ => Err(game_err!("modern opening: scene group incomplete")),
        }
    }

    /// `None` 表示整組的像素檢查曾失敗。上層可以畫同一格原版圖。
    pub(crate) fn frame(&mut self, source: OpeningSource, index: usize) -> Result<Option<Arc<Picture>>> {
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
        let picture = if bank.verified.contains(&frame.sha256) {
            decode_png_picture(&frame.bytes)
        } else {
            decode_opaque_png_picture(&frame.bytes)
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
        Ok(Some(picture))
    }

    pub(crate) fn take_notices(&mut self) -> Vec<String> {
        std::mem::take(&mut self.notices)
    }
}

#[cfg(test)]
#[path = "modern_opening_test.rs"]
mod tests;
