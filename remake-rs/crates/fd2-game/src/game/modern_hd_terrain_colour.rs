//! 凍結 Go 版 `modern_hd_terrain.go` 的 HD 地形調色。

use std::collections::{HashMap, VecDeque};

use sha2::{Digest, Sha256};

use super::*;

type Rgb = [u8; 3];
type Palette = [Rgb; 256];
type DigestBytes = [u8; 32];

pub(crate) struct TerrainColourContext {
    base: Palette,
    mapped: Palette,
    raw: Palette,
    key: DigestBytes,
    basis: DigestBytes,
}

impl TerrainColourContext {
    pub(crate) fn new(map_id: u8, base: &[ColorRgba], current: &[ColorRgba], lut: &[u8]) -> Result<Self> {
        if base.len() != 256 || current.len() != 256 || lut.len() != 256 {
            return Err(game_err!("modern HD terrain: incomplete palette or LUT"));
        }
        let rgb = |colour: ColorRgba| [colour.r, colour.g, colour.b];
        let base: Palette = std::array::from_fn(|i| rgb(base[i]));
        let raw: Palette = std::array::from_fn(|i| rgb(current[i]));
        let mapped = std::array::from_fn(|i| raw[lut[i] as usize]);
        let mut key = Sha256::new();
        key.update([map_id]);
        for i in 0..256 {
            key.update(base[i]);
            key.update(mapped[i]);
            key.update(raw[i]);
        }
        let basis = Sha256::digest(base.as_flattened()).into();
        Ok(Self { base, mapped, raw, key: key.finalize().into(), basis })
    }
}

#[derive(Debug, Default)]
pub(super) struct TerrainShades {
    recent: VecDeque<((DigestBytes, bool), Arc<Picture>)>,
    basis: Option<DigestBytes>,
    weights: HashMap<Rgb, ColourWeights>,
}

#[derive(Debug)]
struct ColourWeights {
    indices: [u8; 4],
    weights: [f64; 4],
}

impl ColourWeights {
    fn new(rgb: Rgb, colours: &[u8], base: &Palette) -> Self {
        let mut distances = [f64::INFINITY; 4];
        let mut indices = [0; 4];
        for &index in colours {
            let [r, g, b] = std::array::from_fn(|i| f64::from(rgb[i]) - f64::from(base[index as usize][i]));
            let distance = r * r + g * g + b * b;
            // Equal distances keep the native colour list's order.
            if let Some(slot) = distances.iter().position(|&other| distance < other) {
                distances.copy_within(slot..3, slot + 1);
                indices.copy_within(slot..3, slot + 1);
                distances[slot] = distance;
                indices[slot] = index;
            }
        }
        let weights = if distances[0] == 0.0 {
            distances.map(|distance| if distance == 0.0 { 1.0 } else { 0.0 })
        } else {
            distances.map(|distance| 1.0 / distance)
        };
        let total: f64 = weights.iter().sum();
        let weights = if total == 0.0 { weights } else { weights.map(|weight| weight / total) };
        Self { indices, weights }
    }

    fn apply(&self, rgb: Rgb, base: &Palette, target: &Palette) -> Rgb {
        let mut values = rgb.map(f64::from);
        for (&index, &weight) in self.indices.iter().zip(&self.weights) {
            let index = index as usize;
            for (channel, value) in values.iter_mut().enumerate() {
                *value += weight * (f64::from(target[index][channel]) - f64::from(base[index][channel]));
            }
        }
        values.map(|value| value.clamp(0.0, 255.0).round() as u8)
    }
}

impl TerrainMap {
    pub(crate) fn shade(&self, id: u16, ctx: &TerrainColourContext, raw: bool) -> Result<Option<Arc<Picture>>> {
        let Some(frame) = self.frames.get(&id) else { return Ok(None) };
        let colours = &self.colours[&id];
        let target = if raw { &ctx.raw } else { &ctx.mapped };
        if colours.iter().all(|&i| target[i as usize] == ctx.base[i as usize]) {
            return Ok(Some(frame.image.clone()));
        }

        let mut cache = self.shades.lock().map_err(|_| game_err!("modern HD terrain: shade cache poisoned"))?;
        let shades = cache.entry(id).or_default();
        let key = (ctx.key, raw);
        if let Some((_, image)) = shades.recent.iter().find(|(cached, _)| *cached == key) {
            return Ok(Some(image.clone()));
        }
        if shades.basis != Some(ctx.basis) {
            shades.weights.clear();
            shades.basis = Some(ctx.basis);
        }
        let collapse = [[0; 3], [255; 3]].into_iter().find(|&rgb| colours.iter().all(|&i| target[i as usize] == rgb));
        let mut changed = HashMap::new();
        let mut pixels = frame.image.read_pixels().to_vec();
        for pixel in pixels.as_chunks_mut::<4>().0 {
            let [r, g, b, alpha] = *pixel;
            if alpha == 0 {
                continue;
            }
            let rgb = [r, g, b];
            let mapped = collapse.unwrap_or_else(|| {
                *changed.entry(rgb).or_insert_with(|| {
                    shades
                        .weights
                        .entry(rgb)
                        .or_insert_with(|| ColourWeights::new(rgb, colours, &ctx.base))
                        .apply(rgb, &ctx.base, target)
                })
            });
            pixel[..3].copy_from_slice(&mapped);
        }
        let mut image = Picture::new_image(frame.image.width, frame.image.height);
        image.write_pixels(&pixels);
        let image = Arc::new(image);
        if shades.recent.len() == 3 {
            shades.recent.pop_front();
        }
        shades.recent.push_back((key, image.clone()));
        Ok(Some(image))
    }
}
