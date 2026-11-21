//! HD1 地形清冊、按需解碼與原版 FDSHAP 遮罩核對。

use std::collections::VecDeque;
use std::io::Cursor;
use std::path::{Path, PathBuf};
use std::sync::Weak;

use serde::Deserialize;

use super::*;

const NATIVE_SIDE: usize = 24;
const SCALE: usize = 6;
const HD_SIDE: usize = NATIVE_SIDE * SCALE;
const MAPS_KEPT: usize = 2;

#[derive(Debug, Clone, Copy, PartialEq, Eq, Deserialize)]
#[serde(rename_all = "lowercase")]
pub(crate) enum TerrainRole {
    Terrain,
    Foreground,
    Both,
}

impl TerrainRole {
    pub(crate) fn supports(self, role: Self) -> bool {
        self == Self::Both || self == role
    }
}

#[derive(Deserialize)]
#[serde(tag = "role")]
enum CatalogEntry {
    #[serde(rename = "native_terrain_hd_set")]
    Terrain(Box<TerrainAsset>),
    #[serde(other)]
    Other,
}

#[derive(Deserialize)]
struct TerrainAsset {
    map_id: u8,
    status: String,
    consumer_contract: String,
    alpha_contract: String,
    cycle_policy: String,
    width: usize,
    height: usize,
    frame_count: usize,
    frame_ids: Vec<u16>,
    files: Vec<String>,
    frame_sha256: Vec<String>,
    semantics: Vec<TerrainRole>,
}

#[derive(Debug)]
struct FrameSource {
    path: PathBuf,
    sha: String,
    role: TerrainRole,
}

#[derive(Debug, Default)]
struct MapSource {
    frames: BTreeMap<u16, FrameSource>,
    native: Option<NativeMasks>,
    rejected: bool,
}

#[derive(Debug)]
struct NativeMasks {
    bank: Weak<fdicon::Bank>,
    colours: Arc<BTreeMap<u16, Vec<u8>>>,
}

#[derive(Debug, Clone)]
pub(crate) struct TerrainFrame {
    pub(crate) role: TerrainRole,
    pub(crate) image: Arc<Picture>,
}

#[derive(Debug)]
pub(crate) struct TerrainMap {
    pub(crate) frames: BTreeMap<u16, TerrainFrame>,
    pub(crate) colours: Arc<BTreeMap<u16, Vec<u8>>>,
}

#[derive(Debug, Default)]
pub(crate) struct ModernTerrain {
    maps: BTreeMap<u8, MapSource>,
    recent: VecDeque<(u8, Arc<TerrainMap>)>,
    notices: Vec<String>,
}

impl ModernTerrain {
    pub(crate) fn load(catalog: &Path, root: &Path) -> Result<Self> {
        let mut terrain = Self::default();
        for entry in read_theme_catalog(catalog)? {
            if let CatalogEntry::Terrain(asset) = entry {
                terrain.admit(*asset, root)?;
            }
        }
        Ok(terrain)
    }

    fn admit(&mut self, asset: TerrainAsset, root: &Path) -> Result<()> {
        let count = asset.frame_count;
        if asset.map_id >= 33
            || !(1..=1024).contains(&count)
            || asset.frame_ids.len() != count
            || asset.files.len() != count
            || asset.frame_sha256.len() != count
            || asset.semantics.len() != count
            || (asset.width, asset.height) != (HD_SIDE, HD_SIDE)
            || !matches!(asset.status.as_str(), "runtime_candidate" | "runtime_ready")
            || asset.consumer_contract != "fdshap_native_hd_sparse_v1"
            || asset.alpha_contract != "native_mask_nearest_6"
            || asset.cycle_policy != "native_selector"
        {
            return Err(game_err!("modern HD terrain: invalid catalog contract for map {}", asset.map_id));
        }
        let previous = self.maps.get(&asset.map_id);
        let mut frames = BTreeMap::new();
        for (((id, name), sha), role) in
            asset.frame_ids.into_iter().zip(asset.files).zip(asset.frame_sha256).zip(asset.semantics)
        {
            if id >= 1024 || frames.contains_key(&id) || previous.is_some_and(|map| map.frames.contains_key(&id)) {
                return Err(game_err!("modern HD terrain: invalid or duplicate map {} frame {id}", asset.map_id));
            }
            if theme_file_extension(&name) != Some("png") {
                return Err(game_err!("modern HD terrain: unsafe file path for map {} frame {id}", asset.map_id));
            }
            let path = root.join(name);
            let raw = read_verified_theme_file(&path, &sha)?;
            let reader = png::Decoder::new(Cursor::new(raw)).read_info().map_err(|err| {
                game_err!("modern HD terrain map {} frame {id}: invalid PNG header: {err}", asset.map_id)
            })?;
            if (reader.info().width as usize, reader.info().height as usize) != (HD_SIDE, HD_SIDE) {
                return Err(game_err!("modern HD terrain map {} frame {id}: PNG geometry changed", asset.map_id));
            }
            frames.insert(id, FrameSource { path, sha, role });
        }
        self.maps.entry(asset.map_id).or_default().frames.extend(frames);
        Ok(())
    }

    /// Map assets are published only after every image and the native masks agree.
    /// `None` selects original art for an absent map or a rejected pixel decode.
    pub(crate) fn map(&mut self, map_id: u8, bank: &Arc<fdicon::Bank>) -> Result<Option<Arc<TerrainMap>>> {
        let Some(source) = self.maps.get_mut(&map_id) else { return Ok(None) };
        if source.rejected {
            return Ok(None);
        }
        let existing = self.recent.iter().position(|(id, _)| *id == map_id);
        let same_bank = source.native.as_ref().filter(|n| n.bank.as_ptr() == Arc::as_ptr(bank));
        if let Some(i) = existing
            && same_bank.is_some()
        {
            let entry = self.recent.remove(i).unwrap();
            let map = entry.1.clone();
            self.recent.push_back(entry);
            return Ok(Some(map));
        }
        let frames = if let Some(i) = existing {
            self.recent[i].1.frames.clone()
        } else {
            match decode_map(&source.frames)? {
                Ok(frames) => frames,
                Err(frame) => {
                    source.rejected = true;
                    self.notices
                        .push(format!("modern HD terrain map {map_id} frame {frame}: invalid PNG or non-binary alpha"));
                    return Ok(None);
                }
            }
        };
        let colours = match same_bank {
            Some(native) => native.colours.clone(),
            None => {
                let colours = Arc::new(validate_masks(&frames, bank)?);
                source.native = Some(NativeMasks { bank: Arc::downgrade(bank), colours: colours.clone() });
                colours
            }
        };
        let prepared = Arc::new(TerrainMap { frames, colours });
        if let Some(i) = existing {
            self.recent.remove(i);
        } else if self.recent.len() == MAPS_KEPT {
            self.recent.pop_front();
        }
        self.recent.push_back((map_id, prepared.clone()));
        Ok(Some(prepared))
    }

    pub(crate) fn take_notices(&mut self) -> Vec<String> {
        std::mem::take(&mut self.notices)
    }
}

/// 外層錯誤是檔案被刪改，內層錯誤是指定畫格的像素驗證失敗。
fn decode_map(sources: &BTreeMap<u16, FrameSource>) -> Result<std::result::Result<BTreeMap<u16, TerrainFrame>, u16>> {
    let mut by_sha = BTreeMap::new();
    let mut jobs = Vec::new();
    for (&id, source) in sources {
        // 即使 SHA 相同，不同檔名也必須通過核對。
        let bytes = read_verified_theme_file(&source.path, &source.sha)?;
        by_sha.entry(&source.sha).or_insert_with(|| {
            let index = jobs.len();
            jobs.push((id, bytes));
            index
        });
    }
    let workers = std::thread::available_parallelism().map_or(1, |n| (n.get() / 2).clamp(1, 3));
    let decode = |(id, bytes): &(u16, Vec<u8>)| {
        decode_precise_png_picture(bytes)
            .filter(|p| {
                (p.width, p.height) == (HD_SIDE as i64, HD_SIDE as i64)
                    && p.read_pixels().as_chunks::<4>().0.iter().all(|px| matches!(px[3], 0 | 255))
            })
            .map(Arc::new)
            .ok_or(*id)
    };
    let pictures: std::result::Result<Vec<_>, u16> = if workers == 1 || jobs.len() < 2 {
        jobs.iter().map(decode).collect()
    } else {
        std::thread::scope(|scope| {
            let threads: Vec<_> = jobs
                .chunks(jobs.len().div_ceil(workers))
                .map(|chunk| scope.spawn(move || chunk.iter().map(decode).collect::<Vec<_>>()))
                .collect();
            threads.into_iter().flat_map(|worker| worker.join().unwrap()).collect()
        })
    };
    Ok(pictures.map(|pictures| {
        sources
            .iter()
            .map(|(&id, source)| {
                let image = pictures[by_sha[&source.sha]].clone();
                (id, TerrainFrame { role: source.role, image })
            })
            .collect()
    }))
}

fn validate_masks(frames: &BTreeMap<u16, TerrainFrame>, bank: &fdicon::Bank) -> Result<BTreeMap<u16, Vec<u8>>> {
    let mut colours = BTreeMap::new();
    for (&id, frame) in frames {
        let sprite = bank
            .sprites
            .get(usize::from(id))
            .ok_or_else(|| game_err!("modern HD terrain frame {id}: exceeds actual native bank"))?;
        let cells = NATIVE_SIDE * NATIVE_SIDE;
        if [sprite.pixels.len(), sprite.mask.len(), sprite.remap_mask.len()] != [cells; 3] {
            return Err(game_err!("modern HD terrain frame {id}: native layers incomplete"));
        }
        let mut found = [false; 256];
        for ((&pixel, &mask), &remap) in sprite.pixels.iter().zip(&sprite.mask).zip(&sprite.remap_mask) {
            if mask > 1 || remap > 1 || (mask != 0 && remap != 0) {
                return Err(game_err!("modern HD terrain frame {id}: native masks must be binary and disjoint"));
            }
            if mask != 0 {
                found[usize::from(pixel)] = true;
            }
        }
        for (index, pixel) in frame.image.read_pixels().as_chunks::<4>().0.iter().enumerate() {
            let (x, y) = (index % HD_SIDE, index / HD_SIDE);
            let mask = sprite.mask[(y / SCALE) * NATIVE_SIDE + x / SCALE];
            if pixel[3] != mask * 255 {
                return Err(game_err!("modern HD terrain frame {id}: native alpha mismatch"));
            }
        }
        colours.insert(id, (0..=255u8).filter(|&index| found[usize::from(index)]).collect());
    }
    Ok(colours)
}

#[cfg(test)]
#[path = "modern_hd_terrain_test.rs"]
mod tests;
