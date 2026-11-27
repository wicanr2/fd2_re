//! Go 來源：`remake/cmd/fd2/modern_hd_map_gpu.go`（608 行）。
//!
//! 骨架：Go 檔的 17 個函式中產生 0 個（階段 0 清單），其餘標 `r8`，見 `PORTING.json`。
//! HD 檔：型別只保留第一章忠實模式執行到的欄位，其餘欄位與函式在 R6。

use super::*;

/// Go：`modern_hd_map_gpu.go:14`
///
/// HD 型別：只保留第一章忠實模式下執行到的函式用到的欄位（120「階段 1 骨架的對應規則」），其餘欄位在 R6 補上。
#[derive(Debug, Clone, Default, PartialEq, Eq)]
pub(crate) struct ModernHDMapGPU {
    pub(crate) theme: Option<ModernStoryPortraitSet>,
    pub(crate) banner_name: String,
    pub(crate) quality: HdTextureQuality,
    pub(crate) world: Option<Arc<Picture>>,
    banner: Option<Arc<Picture>>,
    /// 背景＋地形層與它的輸入。輸入逐像素相同時沿用，每幀只疊其餘圖層。
    base: Option<(ModernHDMapBaseKey, Arc<Picture>)>,
    /// 上一張完整世界圖的全部輸入。相同時沿用同一個 `world`，外殼也就不重傳貼圖。
    last: Option<ModernHDMapFrameKey>,
}

#[derive(Debug, Clone, PartialEq, Eq)]
struct ModernHDMapBaseKey {
    background: Arc<Picture>,
    terrain: Vec<ModernHDTerrainDraw>,
    clip: Rectangle,
}

#[derive(Debug, Clone, PartialEq, Eq)]
struct ModernHDMapFrameKey {
    frame: ModernHDMapFrame,
    units: Vec<(Arc<Picture>, Rectangle)>,
    banner: Option<(String, i64)>,
}

pub(crate) const HD_MAP_WIDTH: i64 = 1920;
pub(crate) const HD_MAP_HEIGHT: i64 = 1200;

impl ModernHDMapGPU {
    pub(crate) fn clear_banner(&mut self) {
        self.banner_name.clear();
        self.banner = None;
    }

    pub(crate) fn compose(
        &mut self,
        frame: &ModernHDMapFrame,
        theme: &ModernStoryPortraitSet,
        banner: Option<(&str, i64)>,
    ) -> Result<Option<Arc<Picture>>> {
        let (Some(background), Some(ui)) = (&frame.background, &frame.ui) else {
            return Err(game_err!("modern HD map: missing background or UI"));
        };
        if frame.palette_add6 > 63 || [background, ui].iter().any(|p| (p.width, p.height) != (320, 200)) {
            return Err(game_err!("modern HD map: invalid plane size or palette addition"));
        }
        let clip = if rect_empty(&frame.viewport) { modern_hd_map_viewport() } else { frame.viewport };
        if let Some(range) = &frame.range
            && (range.width, range.height) != (320, 200)
        {
            return Err(game_err!("modern HD map: invalid range plane size"));
        }
        let units = frame
            .units
            .iter()
            .map(|unit| {
                let image = theme
                    .map_sprites_hd
                    .get(&unit.group)
                    .and_then(|frames| frames.get(unit.frame as usize))
                    .and_then(Option::as_ref)
                    .ok_or_else(|| game_err!("modern HD map: missing unit picture"))?;
                Ok((image.clone(), theme.map_sprite_visual_rect(unit.group, unit.frame, unit.rect)))
            })
            .collect::<Result<Vec<_>>>()?;
        let logical_ui = (!frame.overlays_above_ui).then(|| ui.clone());
        let key = ModernHDMapFrameKey {
            frame: frame.clone(),
            units,
            banner: banner.map(|(name, step)| (name.to_owned(), step)),
        };
        if self.world.is_some() && self.last.as_ref() == Some(&key) {
            return Ok(logical_ui);
        }
        self.last = None;
        let base_key = ModernHDMapBaseKey { background: background.clone(), terrain: frame.terrain.clone(), clip };
        let mut world = match &self.base {
            Some((cached, base)) if *cached == base_key => (**base).clone(),
            _ => {
                let mut base = Picture::new_image(HD_MAP_WIDTH, HD_MAP_HEIGHT);
                draw_layer(&mut base, background, world_rect(), world_rect(), false);
                draw_terrain(&mut base, &frame.terrain, clip);
                let world = base.clone();
                self.base = Some((base_key, Arc::new(base)));
                world
            }
        };
        if let Some(range) = &frame.range {
            draw_layer(&mut world, range, world_rect(), world_rect(), false);
        }
        for ((image, area), unit) in key.units.iter().zip(&frame.units) {
            draw_layer(&mut world, image, *area, clip, unit.palette_band);
        }
        for foreground in &frame.foreground {
            let image = foreground.hd_image.as_ref().unwrap_or(&foreground.image);
            draw_layer(&mut world, image, foreground.rect, clip, false);
        }
        if !frame.overlays_above_ui {
            draw_patches(&mut world, &frame.overlays, clip)?;
        }
        if let Some((name, step)) = banner {
            if self.banner_name != name || self.banner.is_none() {
                self.banner_name = name.to_owned();
                self.banner = Some(Arc::new(world));
            }
            let source = self.banner.as_ref().unwrap();
            world = if step < 0 { (**source).clone() } else { mosaic_world(source, step) };
        } else {
            self.clear_banner();
        }
        if frame.overlays_above_ui {
            draw_layer(&mut world, ui, world_rect(), world_rect(), false);
            draw_patches(&mut world, &frame.hud, clip)?;
            draw_patches(&mut world, &frame.overlays, clip)?;
        } else {
            draw_patches(&mut world, &frame.hud, clip)?;
        }
        if frame.palette_add6 != 0 {
            let addition = f32::from(frame.palette_add6) / 63.0;
            let mut pixels = world.read_pixels().to_vec();
            for p in pixels.as_chunks_mut::<4>().0 {
                for i in 0..3 {
                    p[i] = (f32::from(p[i]) + addition * f32::from(p[3])).min(f32::from(p[3])).round_ties_even() as u8;
                }
            }
            world.write_pixels(&pixels);
        }
        self.world = Some(Arc::new(world));
        self.last = Some(key);
        Ok(logical_ui)
    }
}

pub(crate) fn compose_poster_terrain(base: &Picture, terrain: &[ModernHDTerrainDraw], clip: Rectangle) -> Picture {
    let mut world = Picture::new_image(HD_MAP_WIDTH, HD_MAP_HEIGHT);
    draw_layer(&mut world, base, world_rect(), world_rect(), false);
    draw_terrain(&mut world, terrain, clip);
    world
}

fn draw_terrain(world: &mut Picture, layers: &[ModernHDTerrainDraw], clip: Rectangle) {
    for terrain in layers {
        let rect = if terrain.scale_to_rect {
            terrain.rect
        } else {
            at_size(terrain.rect.min, terrain.image.width, terrain.image.height)
        };
        draw_layer(world, &terrain.image, rect, clip, false);
    }
}

fn world_rect() -> Rectangle {
    rect(0, 0, HD_MAP_WIDTH, HD_MAP_HEIGHT)
}

fn at_size(origin: Point, width: i64, height: i64) -> Rectangle {
    rect(origin.x, origin.y, origin.x + width, origin.y + height)
}

fn draw_layer(world: &mut Picture, image: &Picture, area: Rectangle, clip: Rectangle, dim: bool) {
    let mut op = DrawImageOptions::default();
    op.geo_m.scale(
        (area.max.x - area.min.x) as f64 / image.width as f64,
        (area.max.y - area.min.y) as f64 / image.height as f64,
    );
    op.geo_m.translate(area.min.x as f64, area.min.y as f64);
    if dim {
        op.color_scale.scale(0.55, 0.55, 0.6, 1.0);
    }
    world.sub_image_mut(clip).draw_image(image, &op);
}

fn draw_patches(world: &mut Picture, patches: &[ModernHDMapOverlayDraw], clip: Rectangle) -> Result<()> {
    for patch in patches {
        let Some(image) = &patch.image else { return Err(game_err!("modern HD map: missing overlay picture")) };
        let area = if patch.scale_to_rect { patch.rect } else { at_size(patch.rect.min, image.width, image.height) };
        draw_layer(world, image, area, clip, false);
    }
    Ok(())
}

fn mosaic_world(source: &Picture, step: i64) -> Picture {
    let block = (indexedmap::phase_banner_block(step) * 6) as usize;
    let dim = (indexedmap::phase_banner_dim(step) as f32 / 63.0 * 255.0).round_ties_even() as u8;
    let original = source.read_pixels().as_chunks::<4>().0;
    let mut pixels = original.to_vec();
    for y in 24..1176 {
        let row = y * HD_MAP_WIDTH as usize;
        let base = (24 + (y - 24) / block * block) * HD_MAP_WIDTH as usize;
        for x in 24..1896 {
            let at = base + 24 + (x - 24) / block * block;
            let [r, g, b, a] = original[at];
            pixels[row + x] = [r.saturating_sub(dim), g.saturating_sub(dim), b.saturating_sub(dim), a];
        }
    }
    let mut world = Picture::new_image(HD_MAP_WIDTH, HD_MAP_HEIGHT);
    world.write_pixels(pixels.as_flattened());
    world
}

impl Game {
    pub(crate) fn draw_modern_hd_map_frame(&mut self, screen: &mut Picture) -> Result<()> {
        if self.has_hd_story_map_source() {
            let frame = self.prepare_hd_story_map().inspect_err(|_| screen.clear())?;
            self.modern_hd_map_frame = Some(frame);
        } else {
            self.modern_hd_map_composing = true;
            let result = self.compose_native_map_frame();
            self.modern_hd_map_composing = false;
            result?;
        }
        let step = self.phase_banner_step();
        let timed_dialogue = self.modern_hd_timed_dialogue_eligible();
        let frame = self.modern_hd_map_frame.as_ref().ok_or_else(|| game_err!("modern HD map: missing frame"))?;
        let frame = if timed_dialogue {
            let cells = &self.native_class_ui.as_ref().unwrap().dialogue;
            let control = &self.native_dialogue_layout.as_ref().unwrap().control;
            self.native_level_up_dialogue
                .as_mut()
                .unwrap()
                .freeze_hd_world(frame, cells, control)
                .inspect_err(|_| screen.clear())?
        } else {
            frame
        };
        let theme = self.modern_story_portraits.as_ref().ok_or_else(|| game_err!("modern HD map: missing theme"))?;
        let banner = (self.banner_t > 0 && !self.banner.is_empty()).then_some((self.banner.as_str(), step));
        let renderer = self.modern_hd_map_gpu.get_or_insert_with(ModernHDMapGPU::default);
        let ui = renderer.compose(frame, theme, banner)?;
        screen.clear();
        if !self.modern_hd_separate_world {
            let mut op = DrawImageOptions::default();
            op.geo_m.scale(LOGICAL_W as f64 / HD_MAP_WIDTH as f64, LOGICAL_H as f64 / HD_MAP_HEIGHT as f64);
            screen.draw_image(renderer.world.as_ref().unwrap(), &op);
        }
        if let Some(ui) = ui {
            let mut op = DrawImageOptions::default();
            op.geo_m.scale(2.0, 2.0);
            screen.draw_image(&ui, &op);
        }
        self.modern_hd_world_visible = true;
        self.modern_hd_map_presented_frame = self.frame;
        self.modern_hd_map_presented = Some(frame.clone());
        Ok(())
    }
}

#[cfg(test)]
#[path = "modern_hd_map_gpu_test.rs"]
mod tests;
