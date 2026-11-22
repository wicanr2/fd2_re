//! 對應凍結 Go 版地圖人物的清冊、透明度及落腳點核對。

use std::path::Path;

use serde::{Deserialize, Deserializer};

use super::modern_hd_terrain::ModernTerrainArt;
use super::*;

#[derive(Deserialize)]
#[serde(tag = "role")]
enum CatalogEntry {
    #[serde(rename = "map_sprite_set")]
    Parent(SpriteSet),
    #[serde(rename = "map_sprite_hd_set")]
    Hd(SpriteSet),
    #[serde(other)]
    Other,
}

#[derive(Deserialize)]
struct SpriteSet {
    source_group: i64,
    status: String,
    consumer_contract: String,
    frame_count: usize,
    width: u32,
    height: u32,
    alpha_contract: String,
    #[serde(default)]
    layout_policy: String,
    cycle_policy: String,
    files: Vec<String>,
    frame_sha256: Vec<String>,
    #[serde(default, deserialize_with = "visual_scale")]
    visual_scale_percent: u8,
}

fn visual_scale<'de, D: Deserializer<'de>>(value: D) -> std::result::Result<u8, D::Error> {
    match u8::deserialize(value)? {
        scale @ (100 | 115) => Ok(scale),
        _ => Err(serde::de::Error::custom("unsupported map sprite scale")),
    }
}

impl SpriteSet {
    fn validate(&self, hd: bool) -> Result<()> {
        let (side, alpha, walk, still) = if hd {
            (144, "coverage", "fdicon_hd_map_sprite_12x144_v1", "fdicon_hd_static_sprite_3x144_v1")
        } else {
            (24, "binary", "fdicon_map_sprite_12x24_v1", "fdicon_static_sprite_3x24_v1")
        };
        let contract = match (self.consumer_contract.as_str(), self.frame_count, self.cycle_policy.as_str()) {
            (name, 12, "three_distinct_cycles" | "source_exact_repeats") => name == walk,
            (name, 3, "static_three_phase") => name == still,
            _ => false,
        };
        if !contract
            || !fdicon::native_source_group_has_pixels(self.source_group)
            || !matches!(self.status.as_str(), "runtime_candidate" | "runtime_ready")
            || (self.width, self.height) != (side, side)
            || self.alpha_contract != alpha
            || self.files.len() != self.frame_count
            || self.frame_sha256.len() != self.frame_count
            || (hd && self.layout_policy != "contain_22px_cell_feet_v1")
        {
            return Err(game_err!("modern map sprite {} violates the frame contract", self.source_group));
        }
        let unique = self.frame_sha256.iter().collect::<BTreeSet<_>>().len();
        if (unique < self.frame_count) != (self.cycle_policy == "source_exact_repeats") {
            return Err(game_err!("modern map sprite {} violates the phase repetition contract", self.source_group));
        }
        Ok(())
    }
}

#[derive(Clone)]
struct SpriteFrame {
    picture: Arc<Picture>,
    ink: Rectangle,
}

fn decode_sprite(raw: &[u8], side: u32) -> Option<SpriteFrame> {
    let mut min = Point { x: i64::from(side), y: i64::from(side) };
    let mut max = Point::default();
    let (mut transparent, mut opaque, mut partial) = (false, false, false);
    let picture = decode_sprite_png(raw, side, |x, y, alpha| {
        transparent |= alpha == 0;
        opaque |= alpha == u16::MAX;
        partial |= alpha != 0 && alpha != u16::MAX;
        if alpha != 0 {
            min.x = min.x.min(x);
            min.y = min.y.min(y);
            max.x = max.x.max(x + 1);
            max.y = max.y.max(y + 1);
        }
    })?;
    let ink = Rectangle { min, max };
    if side == 24 && partial {
        return None;
    }
    if side == 144
        && (!transparent
            || !opaque
            || rect_empty(&ink)
            || max.x - min.x > 132
            || max.y - min.y > 132
            || !(142..=146).contains(&(min.x + max.x)))
    {
        return None;
    }
    Some(SpriteFrame { picture: Arc::new(picture), ink })
}

fn read_sprites(assets: &BTreeMap<i64, SpriteSet>, root: &Path) -> Result<BTreeMap<i64, Vec<SpriteFrame>>> {
    let mut shared = BTreeMap::<&str, SpriteFrame>::new();
    let mut groups = BTreeMap::new();
    for (&group, asset) in assets {
        let mut frames = Vec::with_capacity(asset.frame_count);
        for (index, (name, sha)) in asset.files.iter().zip(&asset.frame_sha256).enumerate() {
            if theme_file_extension(name) != Some("png") {
                return Err(game_err!("modern map sprite {group} frame {index}: unsafe PNG path"));
            }
            let raw = read_verified_theme_file(&root.join(name), sha)?;
            let frame = if let Some(frame) = shared.get(sha.as_str()) {
                frame.clone()
            } else {
                let frame = decode_sprite(&raw, asset.width).ok_or_else(|| {
                    game_err!("modern map sprite {group} frame {index}: invalid PNG geometry or alpha")
                })?;
                shared.insert(sha, frame.clone());
                frame
            };
            frames.push(frame);
        }
        groups.insert(group, frames);
    }
    Ok(groups)
}

fn pictures(groups: BTreeMap<i64, Vec<SpriteFrame>>) -> BTreeMap<i64, Vec<Option<Arc<Picture>>>> {
    groups.into_iter().map(|(group, frames)| (group, frames.into_iter().map(|f| Some(f.picture)).collect())).collect()
}

impl ModernStoryPortraitSet {
    /// Admit the entire map-art portion of the candidate theme before installing any of it.
    pub(crate) fn load_map_art(&mut self, catalog: &Path, root: &Path) -> Result<()> {
        let mut parents = BTreeMap::new();
        let mut children = BTreeMap::new();
        for entry in read_theme_catalog(catalog)? {
            let (target, asset, hd) = match entry {
                CatalogEntry::Parent(asset) => (&mut parents, asset, false),
                CatalogEntry::Hd(asset) => (&mut children, asset, true),
                CatalogEntry::Other => continue,
            };
            asset.validate(hd)?;
            let group = asset.source_group;
            if target.insert(group, asset).is_some() {
                return Err(game_err!("modern map sprite {group}: duplicate catalog group"));
            }
        }
        let expected = (0..fdicon::NATIVE_SOURCE_GROUP_COUNT).filter(|&id| fdicon::native_source_group_has_pixels(id));
        if !parents.keys().copied().eq(expected.clone()) || !children.keys().copied().eq(expected) {
            return Err(game_err!("modern map sprite: incomplete parent or HD collection"));
        }
        let base = read_sprites(&parents, root)?;
        let hd = read_sprites(&children, root)?;
        let mut feet = BTreeMap::new();
        for (&id, child) in &children {
            let parent = &parents[&id];
            if (parent.frame_count, &parent.cycle_policy) != (child.frame_count, &child.cycle_policy) {
                return Err(game_err!("modern map sprite {id}: parent cycle contract differs"));
            }
            let mut group_feet = Vec::with_capacity(parent.frame_count);
            for i in 0..child.frame_count {
                let parent_ink = base[&id][i].ink;
                let foot = parent_ink.max.y * 6;
                if rect_empty(&parent_ink) || !(0..=1).contains(&(foot - hd[&id][i].ink.max.y)) {
                    return Err(game_err!("modern map sprite {id} frame {i}: registered foot differs"));
                }
                if (0..i).any(|j| {
                    (child.frame_sha256[i] == child.frame_sha256[j])
                        != (parent.frame_sha256[i] == parent.frame_sha256[j])
                }) {
                    return Err(game_err!("modern map sprite {id} frame {i}: phase repetition differs"));
                }
                group_feet.push(foot);
            }
            if child.visual_scale_percent == 115 {
                feet.insert(id, group_feet);
            }
        }
        let terrain_hd = ModernTerrainArt::load(catalog, root)?;
        self.map_sprites = pictures(base);
        self.map_sprites_hd = pictures(hd);
        self.map_sprite_feet = feet;
        self.terrain_hd = Some(terrain_hd);
        Ok(())
    }

    pub(crate) fn map_sprite_visual_rect(&self, group: i64, frame: i64, area: Rectangle) -> Rectangle {
        let Some(foot) = self.map_sprite_feet.get(&group).and_then(|feet| feet.get(frame as usize)) else {
            return area;
        };
        if rect_empty(&area) {
            return area;
        }
        let (w, h) = (area.max.x - area.min.x, area.max.y - area.min.y);
        let width = (w as f64 * 115.0 / 100.0).round_ties_even() as i64;
        let height = (h as f64 * 115.0 / 100.0).round_ties_even() as i64;
        let x = ((area.min.x + area.max.x - width) as f64 / 2.0).round_ties_even() as i64;
        let y = ((area.min.y + foot) as f64 - *foot as f64 * height as f64 / h as f64).round_ties_even() as i64;
        rect(x, y, x + width, y + height)
    }
}

#[cfg(test)]
#[path = "modern_hd_map_sprite_assets_test.rs"]
mod tests;
