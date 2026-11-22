//! 劇情走行與移鏡的私有投影，沿 Go `modern_hd_story_motion.go`。

use super::modern_hd_terrain::{SCALE, TerrainColourContext, TerrainMap, TerrainRole};
use super::*;

const TILE: i64 = fdicon::NATIVE_SIZE;
const HD_TILE: i64 = TILE * SCALE as i64;

fn hd_pixel(offset: f64) -> i64 {
    (offset * SCALE as f64).round() as i64
}

fn translate(area: Rectangle, delta: Point) -> Rectangle {
    rect(area.min.x + delta.x, area.min.y + delta.y, area.max.x + delta.x, area.max.y + delta.y)
}

impl Game {
    pub(crate) fn project_hd_story_motion(
        &self,
        frame: &mut ModernHDMapFrame,
        input: &indexedmap::FrameInput,
        seed: Vec<u8>,
        palette: &[ColorRgba],
        art: Option<(&TerrainMap, &TerrainColourContext)>,
        sprites: &BTreeMap<i64, Vec<Option<Arc<Picture>>>>,
    ) -> Result<()> {
        if input.units.len() != self.story_actors.len() || input.foreground_units.len() != self.story_actors.len() {
            return Err(game_err!("modern HD story: visual roster provenance mismatch"));
        }
        let dx = self.cam_x - (input.camera_x * TILE) as f64;
        let dy = self.cam_y - (input.camera_y * TILE) as f64;
        let shift = Point { x: -hd_pixel(dx), y: -hd_pixel(dy) };
        let viewport = modern_hd_map_viewport();
        let cols = (input.map_width - input.camera_x).min(14);
        let rows = (input.cells.len() as i64 / input.map_width - input.camera_y).min(9);
        let parallax = StoryParallax::from_input(input, cols, rows, dx, dy)?;
        let work = story_terrain_work(input, seed, cols, rows, dx, dy, parallax.as_ref())?;
        let backdrop = match parallax {
            Some(parallax) => parallax.backdrop(input, cols, rows, palette)?,
            None => native_backdrop(&work, cols, rows, palette),
        };
        let backdrop_rect =
            rect(viewport.min.x, viewport.min.y, viewport.min.x + cols * HD_TILE, viewport.min.y + rows * HD_TILE);
        let mut terrain = vec![ModernHDTerrainDraw {
            image: Arc::new(backdrop),
            rect: translate(backdrop_rect, shift),
            frame: 0,
            work_offset: NATIVE_MAP_STEADY_BASE,
            scale_to_rect: true,
        }];
        if let Some((map, colours)) = art {
            let region = TerrainRegion {
                terrain_base: NATIVE_MAP_STEADY_BASE,
                viewport_base: NATIVE_MAP_STEADY_BASE,
                camera_y: input.camera_y,
                cols,
                rows,
            };
            let clip = translate(viewport, Point { x: -shift.x, y: -shift.y });
            let tiles = build_modern_hd_terrain_region_plan(input, region, clip, map, colours)?;
            terrain.extend(tiles.into_iter().map(|mut tile| {
                tile.rect = translate(tile.rect, shift);
                tile
            }));
        }

        let mut walking = vec![false; self.story_actors.len()];
        for job in &self.story_walks {
            let actor = walking
                .get_mut(job.actor as usize)
                .ok_or_else(|| game_err!("modern HD story: invalid walking actor"))?;
            *actor = true;
        }
        let mut input = input.clone();
        let mut units = Vec::new();
        for (index, &moves) in walking.iter().enumerate() {
            let raw = &input.units[index];
            if moves {
                if let Some(unit) = self.project_walking_story_actor(
                    &self.story_actors[index],
                    raw.slot,
                    &mut input.foreground_units[index],
                    (input.unit_bank, input.selector_cache),
                    sprites,
                    viewport,
                )? {
                    units.push(unit);
                }
            } else {
                let draws = build_modern_hd_map_unit_region_plan(
                    &input,
                    std::slice::from_ref(raw),
                    NATIVE_MAP_STEADY_BASE,
                    sprites,
                    13,
                    9,
                )?;
                units.extend(draws.into_iter().filter_map(|mut unit| {
                    unit.rect = translate(unit.rect, shift);
                    (!rect_empty(&rect_intersect(unit.rect, viewport))).then_some(unit)
                }));
            }
        }
        let mut foreground =
            build_modern_hd_map_foreground_region_plan(&input, NATIVE_MAP_STEADY_BASE, palette, 13, 9)?;
        for draw in &mut foreground {
            draw.rect = translate(draw.rect, shift);
            if let Some((map, colours)) = art
                && let Ok(id) = u16::try_from(draw.source_frame)
                && map.frames.get(&id).is_some_and(|tile| tile.role.supports(TerrainRole::Foreground))
            {
                draw.hd_image = map.shade(id, colours, draw.blit_mode == 0xff)?;
            }
        }
        frame.terrain = terrain;
        frame.units = units;
        frame.foreground = foreground;
        frame.terrain_seed = work;
        Ok(())
    }

    fn project_walking_story_actor(
        &self,
        actor: &battle::Unit,
        slot: i64,
        foreground: &mut fdicon::NativeForegroundLayerEntry,
        (bank, cache): (Option<&fdicon::Bank>, Option<&fdicon::NativeSelectorCache>),
        sprites: &BTreeMap<i64, Vec<Option<Arc<Picture>>>>,
        viewport: Rectangle,
    ) -> Result<Option<ModernHDMapUnitDraw>> {
        foreground.inactive = !actor.on_field || !actor.alive();
        if foreground.inactive {
            return Ok(None);
        }
        let px = actor.x as f64 * TILE as f64 + actor.off_x;
        let py = actor.y as f64 * TILE as f64 + actor.off_y;
        if !px.is_finite() || !py.is_finite() || !(0..4).contains(&actor.dir) {
            return Err(game_err!("modern HD story: invalid visual placement"));
        }
        foreground.x = (px / TILE as f64).floor() as i64;
        foreground.y = (py / TILE as f64).floor() as i64;
        foreground.pose = actor.dir as u8;
        foreground.motion_offset = 1;
        let x = viewport.min.x.saturating_add(hd_pixel(px - self.cam_x));
        let y = viewport.min.y.saturating_add(hd_pixel(py - self.cam_y - 6.0));
        let area = rect(x, y, x.saturating_add(HD_TILE), y.saturating_add(HD_TILE));
        if rect_empty(&rect_intersect(area, viewport)) {
            return Ok(None);
        }
        let (Some(bank), Some(cache)) = (bank, cache) else {
            return Err(game_err!("modern HD story: missing unit bank or selector cache"));
        };
        let group = cache.key_for_slot(slot)?;
        let cycle = (self.frame / 8) % 3;
        let sprite = bank.sprite_for(group, actor.dir, cycle)?;
        if !sprite.mask.iter().any(|&value| value != 0) {
            return Ok(None);
        }
        let pictures = sprites.get(&group).ok_or_else(|| game_err!("modern HD story: missing group {group}"))?;
        let frame = if pictures.len() == 3 { cycle } else { actor.dir * 3 + cycle };
        if !pictures
            .get(frame as usize)
            .and_then(Option::as_ref)
            .is_some_and(|p| (p.width, p.height) == (HD_TILE, HD_TILE))
        {
            return Err(game_err!("modern HD story: invalid group {group} frame {frame}"));
        }
        Ok(Some(ModernHDMapUnitDraw { group, frame, rect: area, palette_band: actor.acted, ..Default::default() }))
    }
}

fn story_terrain_work(
    input: &indexedmap::FrameInput,
    mut work: Vec<u8>,
    cols: i64,
    rows: i64,
    dx: f64,
    dy: f64,
    parallax: Option<&StoryParallax<'_>>,
) -> Result<Vec<u8>> {
    let base = NATIVE_MAP_STEADY_BASE;
    let stride = fdicon::NATIVE_MAP_STRIDE;
    let bank = input.terrain_bank.ok_or_else(|| game_err!("modern HD story: missing terrain bank"))?;
    if let Some(aux) = input.chapter_aux {
        fdother::blit_native_chapter_aux_viewport(&mut work[base as usize..], stride, aux, input.chapter_aux_phase)?;
    }
    let underlay = work.clone();
    for y in 0..rows * TILE {
        let sy = ((y as f64 - dy).floor() as i64).clamp(0, 191);
        for x in 0..cols * TILE {
            let sx = ((x as f64 - dx).floor() as i64).clamp(0, 311);
            let value = match parallax {
                Some(source) => source.at((x as usize) * SCALE, (y as usize) * SCALE),
                None => underlay[(base + sy * stride + sx) as usize],
            };
            work[(base + y * stride + x) as usize] = value;
        }
    }
    bank.blit_native_terrain_region(
        &mut work,
        stride,
        base % stride,
        base / stride,
        input.map_width,
        &input.cells,
        input.controls,
        input.camera_x,
        input.camera_y,
        cols,
        rows,
        input.flip,
        input.terrain_cycle,
        input.lut,
    )?;
    Ok(work)
}

fn native_backdrop(work: &[u8], cols: i64, rows: i64, palette: &[ColorRgba]) -> Picture {
    let (width, height) = (cols * TILE, rows * TILE);
    let mut pix = Vec::with_capacity((width * height * 4) as usize);
    for y in 0..height {
        let start = (NATIVE_MAP_STEADY_BASE + y * fdicon::NATIVE_MAP_STRIDE) as usize;
        for &index in &work[start..start + width as usize] {
            let c = palette[index as usize];
            pix.extend_from_slice(&[c.r, c.g, c.b, c.a]);
        }
    }
    Picture { width, height, pix }
}

/// 原生與 HD 兩個取樣網格共用查表，避開逐像素的浮點運算。
struct StoryParallax<'a> {
    pixels: &'a [u8],
    columns: Vec<usize>,
    row_offsets: Vec<usize>,
}

impl<'a> StoryParallax<'a> {
    fn from_input(input: &indexedmap::FrameInput<'a>, cols: i64, rows: i64, dx: f64, dy: f64) -> Result<Option<Self>> {
        use fdother::{NATIVE_MAP_PARALLAX_HEIGHT as H, NATIVE_MAP_PARALLAX_WIDTH as W};
        let Some(source) = input.parallax else { return Ok(None) };
        if source.pixels.len() != (W * H) as usize {
            return Err(game_err!("modern HD story: incomplete parallax source"));
        }
        let x0 = (3 * input.camera_x + input.parallax_scroll_x / 2) as f64;
        let y0 = (2 * input.camera_y + input.parallax_scroll_y / 3) as f64;
        let columns = (0..cols * HD_TILE)
            .map(|x| ((x0 + x as f64 / SCALE as f64 - dx + dx / 8.0).floor() as i64).clamp(0, W - 1) as usize)
            .collect();
        let row_offsets = (0..rows * HD_TILE)
            .map(|y| (((y0 + y as f64 / SCALE as f64 - dy + dy / 12.0).floor() as i64).clamp(0, H - 1) * W) as usize)
            .collect();
        Ok(Some(Self { pixels: &source.pixels, columns, row_offsets }))
    }

    fn at(&self, x: usize, y: usize) -> u8 {
        self.pixels[self.columns[x] + self.row_offsets[y]]
    }

    fn backdrop(&self, input: &indexedmap::FrameInput, cols: i64, rows: i64, palette: &[ColorRgba]) -> Result<Picture> {
        let bank = input.terrain_bank.ok_or_else(|| game_err!("modern HD story: missing terrain bank"))?;
        let (width, height) = (cols * HD_TILE, rows * HD_TILE);
        let mut pix = vec![[0; 4]; (width * height) as usize];
        let side = HD_TILE as usize;
        for row in 0..rows {
            for col in 0..cols {
                let cell = input.cells[((input.camera_y + row) * input.map_width + input.camera_x + col) as usize];
                let tile = i64::from(cell.tile & 0x3ff);
                let frame = fdicon::native_terrain_frame_index(
                    tile,
                    input.controls[(tile * 4) as usize],
                    input.flip,
                    input.terrain_cycle,
                )?;
                let sprite = bank.sprite_for(frame / 12, (frame % 12) / 3, frame % 3)?;
                let raw = cell.blit_mode == 0xff;
                for y in 0..side {
                    let py = row as usize * side + y;
                    let start = py * width as usize + col as usize * side;
                    for (x, dst) in pix[start..start + side].iter_mut().enumerate() {
                        let i = y / SCALE * TILE as usize + x / SCALE;
                        let covered = sprite.mask[i] != 0;
                        let mut value = if covered { sprite.pixels[i] } else { self.at(col as usize * side + x, py) };
                        if !raw && (covered || sprite.remap_mask.get(i).is_some_and(|&mask| mask != 0)) {
                            value = input.lut[value as usize];
                        }
                        let c = palette[value as usize];
                        *dst = [c.r, c.g, c.b, c.a];
                    }
                }
            }
        }
        Ok(Picture { width, height, pix: pix.into_flattened() })
    }
}
