use std::fmt::Write as _;
use std::path::PathBuf;

use sha2::{Digest, Sha256};

use super::*;
use crate::game::native_map_frame_input::native_map_frame_input_test::complete_native_map_frame_fixture;
use crate::test_support::temp_dir;

fn png16_sprite(side: usize, alpha: u16, faint_mark: bool) -> Vec<u8> {
    let mut pixels = vec![0; side * side * 8];
    let centre = 8 * (side * 4 + side / 2 - 1);
    pixels[centre + 6..centre + 8].copy_from_slice(&alpha.to_be_bytes());
    if faint_mark {
        pixels[7] = 1;
    }
    let mut raw = Vec::new();
    let mut encoder = png::Encoder::new(&mut raw, side as u32, side as u32);
    encoder.set_depth(png::BitDepth::Sixteen);
    encoder.set_color(png::ColorType::Rgba);
    encoder.write_header().unwrap().write_image_data(&pixels).unwrap();
    raw
}

#[test]
fn map_sprite_checks_true_sixteen_bit_alpha() {
    assert!(decode_sprite(&png16_sprite(144, 65535, false), 144).is_some());
    assert!(decode_sprite(&png16_sprite(144, 65534, false), 144).is_none());
    assert!(decode_sprite(&png16_sprite(144, 65535, true), 144).is_none());
    assert!(decode_sprite(&png16_sprite(24, 65534, false), 24).is_none());
    assert!(decode_sprite(&png16_sprite(24, 65535, false), 24).is_some());
    assert!(decode_sprite(&png16_sprite(24, 65535, false), 144).is_none());
}

#[test]
fn incomplete_map_art_does_not_replace_existing_images() {
    let root = temp_dir("hd-map-art");
    let catalog = root.join("catalog.json");
    let picture = Arc::new(Picture::new_image(144, 144));
    let mut set =
        ModernStoryPortraitSet { map_sprites_hd: BTreeMap::from([(0, vec![Some(picture)])]), ..Default::default() };
    let original = set.clone();
    std::fs::write(&catalog, r#"{"schema_version":1,"theme_id":"modern-handpainted-a","assets":[]}"#).unwrap();
    assert!(set.load_map_art(&catalog, &root).is_err());
    assert_eq!(set, original);
}

#[test]
#[ignore = "正式私人素材與 Go 黃金由 tools/rust_r6_opening.sh 安排"]
fn real_map_art_matches_go() {
    let env = |name| PathBuf::from(std::env::var_os(name).unwrap());
    let original = env("FD2_ASSET_PACK");
    let out = env("FD2_R6_OPENING_OUT");
    let mut theme = ModernStoryPortraitSet::default();
    theme.load_map_art(&env("FD2_TEST_OPENING_CATALOG"), &env("FD2_TEST_OPENING_PACK")).unwrap();
    assert_eq!(theme.map_sprites_hd.len(), 125);
    assert_eq!(theme.map_sprite_feet.len(), 4);
    let mut report = String::new();
    let mut count = 0;
    for (&group, frames) in &theme.map_sprites {
        for (i, parent) in frames.iter().enumerate() {
            let hd = theme.map_sprites_hd[&group][i].as_ref().unwrap();
            let area = theme.map_sprite_visual_rect(group, i as i64, rect(-15, 7, 129, 151));
            writeln!(
                report,
                "{group}/{i} {:x} {:x} {} {} {} {}",
                Sha256::digest(parent.as_ref().unwrap().read_pixels()),
                Sha256::digest(hd.read_pixels()),
                area.min.x,
                area.min.y,
                area.max.x,
                area.max.y
            )
            .unwrap();
            count += 1;
        }
    }
    assert_eq!(count, 1455);
    std::fs::write(out.join("rust-map-art.txt"), &report).unwrap();
    let expected = std::fs::read_to_string(out.join("go-map-art.txt")).unwrap();
    assert_eq!(report.lines().count(), expected.lines().count());
    for (rust, go) in report.lines().zip(expected.lines()) {
        assert_eq!(rust, go);
    }
    println!("125 組、1455 對人物像素與落腳矩形與 Go 相同，序列 SHA-256 {:x}", Sha256::digest(report.as_bytes()));

    let native_units = fdicon::load_separated_bank(original.join("sprites/fdicon")).unwrap();
    let (dac, palette) = fdother::load_separated_fdother_palette(original.join("palette"), 0).unwrap();
    let terrain_report = std::fs::read_to_string(out.join("go-terrain.txt")).unwrap();
    let mut rendered = String::new();
    for id in [0, 9] {
        let (bank, controls) = fdicon::load_separated_fdshap_bank(original.join("tilesets/fdshap"), id).unwrap();
        let prefix = format!("{id}/");
        let tile = terrain_report
            .lines()
            .find_map(|line| {
                let (index, rest) = line.strip_prefix(&prefix)?.split_once(' ')?;
                let index: usize = index.parse().ok()?;
                let flags = controls.get(index * 4)?;
                (flags & 0x1c == 0 && (rest.starts_with("terrain ") || rest.starts_with("both "))).then_some(index)
            })
            .unwrap();
        let (mut assets, mut field, state) = complete_native_map_frame_fixture();
        assets.map_index = id;
        assets.terrain = Some(Arc::new(bank));
        assets.units = Some(native_units.clone());
        assets.palette_dac = dac.clone();
        assets.palette = color_palette_from_nrgba(&palette);
        assets.chapter_aux = fdother::native_chapter_aux_surface_for(id)
            .map(|contract| fdother::load_separated_chapter_aux_surface_resource(original.join("surfaces"), &contract))
            .transpose()
            .unwrap();
        assets.controls = controls.clone();
        field.native_terrain_control = controls;
        field.tiles.fill(tile as i64);
        let mut game = Game {
            native_map_assets: Some(assets),
            m: Some(field),
            st: Some(state),
            modern_story_portraits: Some(theme.clone()),
            virtual_now: VIRTUAL_CLOCK_ORIGIN,
            ..Default::default()
        };
        assert!(game.modern_hd_map_frame_admission(false, true), "map {id}: incomplete field fixture");
        let mut screen = Picture::new_image(640, 400);
        game.draw_field(&mut screen);
        assert!(game.modern_hd_world_visible, "map {id}: {}", game.load_err);
        let frame = game.modern_hd_map_presented.as_ref().unwrap();
        assert_eq!(frame.terrain.len(), 104);
        assert_eq!(frame.units.len(), 1);
        assert!(frame.terrain.iter().all(|draw| draw.frame == tile as i64));
        let world = game.modern_hd_map_gpu.as_ref().unwrap().world.as_ref().unwrap();
        writeln!(
            rendered,
            "{id} {tile} {} {} {:x}",
            frame.terrain.len(),
            frame.units.len(),
            Sha256::digest(world.read_pixels())
        )
        .unwrap();
    }
    std::fs::write(out.join("rust-map-art-runtime.txt"), rendered).unwrap();
    verify_aligned_hd_story_map(theme, &out);
}

fn verify_aligned_hd_story_map(theme: ModernStoryPortraitSet, out: &std::path::Path) {
    let mut game = Game { virtual_now: VIRTUAL_CLOCK_ORIGIN, ..Default::default() };
    game.load_map("assets/maps/map32").unwrap();
    game.reset_battle("assets/maps/map32/map32_units.json", "");
    assert!(game.load_err.is_empty(), "{}", game.load_err);
    let source = game.st.take().unwrap();
    game.story_actors = source.units.clone();
    game.story_bg = true;
    game.has_story_native_map_view = true;
    game.story_native_map_view = battle::NativeMapViewState {
        camera_x: 0,
        camera_y: 2,
        cursor_x: 7,
        cursor_y: 5,
        visible_cursor_x: 7,
        visible_cursor_y: 3,
    };
    (game.cam_x, game.cam_y, game.cur_x, game.cur_y) = (0.0, 48.0, 7, 5);
    game.materialize_native_story_map_state(Some(&source)).unwrap();
    game.interface_mode = PLAYER_INTERFACE_ORIGINAL.into();
    game.modern_story_portraits = Some(theme);
    game.modern_hd_separate_world = true;
    let mut ui = Picture::new_image(LOGICAL_W, LOGICAL_H);
    let mut report = String::new();
    for camera_x in [0, 1] {
        game.story_native_map_view.camera_x = camera_x;
        game.story_native_map_view.visible_cursor_x = 7 - camera_x;
        game.cam_x = (camera_x * 24) as f64;
        let clock = game.native_map_clock;
        let dac = game.native_map_dac.clone();
        let work = game.native_map_work.clone();
        let vga = game.native_map_vga.clone();
        let actors = game.story_actors.clone();
        let state = game.story_native_map_state.as_ref().unwrap().clone();
        for _ in 0..2 {
            assert!(game.modern_hd_map_frame_admission(true, false));
            game.draw(&mut ui);
            assert!(game.modern_hd_world_visible, "{}", game.load_err);
            let frame = game.modern_hd_map_presented.as_ref().unwrap();
            assert!(!frame.terrain.is_empty() && frame.units.len() >= 2 && frame.range.is_none());
            assert!(frame.units.iter().any(|unit| unit.group == 48));
            assert!(game.st.is_none() && game.story_actors == actors);
            assert!(game.native_map_work == work && game.native_map_vga == vga && game.native_map_dac == dac);
            assert_eq!(game.native_map_clock, clock);
            let displayed = game.story_native_map_state.as_ref().unwrap();
            assert_eq!(displayed.native_map_view_state, state.native_map_view_state);
            assert_eq!(displayed.native_map_cycle_state, state.native_map_cycle_state);
            assert_eq!(displayed.native_map_selector_cache, state.native_map_selector_cache);
        }
        let world = game.modern_hd_map_gpu.as_ref().unwrap().world.as_ref().unwrap();
        let expected = std::fs::read(out.join(format!("go-story-map-{camera_x}.rgba"))).unwrap();
        assert_eq!(world.read_pixels().len(), expected.len());
        let max_diff = world.read_pixels().iter().zip(&expected).map(|(a, b)| u8::abs_diff(*a, *b)).max().unwrap();
        assert!(max_diff <= 1, "map32 camera {camera_x}: colour difference {max_diff}");
        let expected_ui = std::fs::read(out.join(format!("go-story-map-{camera_x}-ui.rgba"))).unwrap();
        assert_eq!(expected_ui.len(), 320 * 200 * 4);
        assert!(expected_ui.iter().all(|&byte| byte == 0) && ui.read_pixels().iter().all(|&byte| byte == 0));
        writeln!(
            report,
            "{camera_x} {max_diff} {:x} {:x}",
            Sha256::digest(&expected),
            Sha256::digest(world.read_pixels())
        )
        .unwrap();
    }
    let previous = game.modern_hd_map_frame.clone();
    let frame = previous.as_ref().unwrap().units.iter().find(|unit| unit.group == 48).unwrap();
    let slot = frame.frame as usize;
    let sprites = game.modern_story_portraits.as_mut().unwrap().map_sprites_hd.get_mut(&48).unwrap();
    let picture = sprites[slot].take();
    game.draw(&mut ui);
    assert!(!game.modern_hd_world_visible && game.load_err.contains("invalid group 48"));
    assert!(game.modern_hd_map_frame == previous);
    game.modern_story_portraits.as_mut().unwrap().map_sprites_hd.get_mut(&48).unwrap()[slot] = picture;
    game.load_err.clear();
    std::fs::write(out.join("rust-story-map.txt"), &report).unwrap();
    println!("皇城兩個固定鏡頭對照 Go：\n{report}");
    verify_hd_story_motion(&mut game, out);
}

fn verify_hd_story_motion(game: &mut Game, out: &std::path::Path) {
    use std::sync::atomic::{AtomicUsize, Ordering};

    #[derive(serde::Deserialize)]
    struct MotionCase {
        name: String,
        x: i64,
        y: i64,
        ticks: i64,
        camera_x: f64,
        camera_y: f64,
        parallax: bool,
        follow: bool,
    }
    let cases: Vec<MotionCase> =
        serde_json::from_slice(&std::fs::read(out.join("story-motion-cases.json")).unwrap()).unwrap();
    assert_eq!(cases.len(), 8);
    let asset_dir = separated_asset_path("surfaces");
    let parallax = fdother::load_separated_native_map_parallax(asset_dir).unwrap();
    let actor =
        game.story_actors.iter().position(|actor| actor.has_map_selector_key && actor.map_selector_key == 0).unwrap();
    let baseline = game.story_actors[actor].clone();
    let mut report = String::new();
    let mut hashes = String::new();
    let mut ui = Picture::new_image(LOGICAL_W, LOGICAL_H);
    for c in cases {
        game.story_actors[actor] = baseline.clone();
        game.story_actors[actor].on_field = true;
        assert!(game.story_actors[actor].set_map_placement(8, 5, 0));
        let raw = game.story_actors[actor].native_map_presentation;
        game.story_native_map_view.camera_x = 0;
        game.story_native_map_view.camera_y = 2;
        game.story_native_map_view.cursor_x = 8;
        game.story_native_map_view.visible_cursor_x = 8;
        (game.cam_x, game.cam_y, game.frame) = (c.camera_x, c.camera_y, 0);
        game.native_map_assets.as_mut().unwrap().parallax = c.parallax.then(|| parallax.clone());
        game.story_walks.clear();
        let completed = Arc::new(AtomicUsize::new(0));
        if c.ticks != 0 {
            let calls = completed.clone();
            game.story_walks.push(StoryWalkJob {
                actor: actor as i64,
                from_x: 8,
                from_y: 5,
                to_x: c.x,
                to_y: c.y,
                frames: 24,
                final_dir: 2,
                scroll_follow: c.follow,
                from_cam_y: c.camera_y,
                scroll_free: 2,
                then: Some(Box::new(move |_| {
                    calls.fetch_add(1, Ordering::Relaxed);
                })),
                ..Default::default()
            });
        }
        for _ in 0..c.ticks {
            game.frame += 1;
            game.step_story_walks();
        }
        let actor_snapshot = game.story_actors[actor].clone();
        let view = game.story_native_map_view;
        let camera = (game.cam_x, game.cam_y);
        let clock = game.native_map_clock;
        game.draw(&mut ui);
        assert!(game.modern_hd_world_visible, "{}: {}", c.name, game.load_err);
        assert_eq!(game.story_actors[actor], actor_snapshot);
        assert_eq!(game.story_actors[actor].native_map_presentation.x, raw.x);
        assert_eq!(game.story_actors[actor].native_map_presentation.y, raw.y);
        assert_eq!(game.story_native_map_view, view);
        assert_eq!((game.cam_x, game.cam_y), camera);
        assert_eq!(game.native_map_clock, clock);
        assert_eq!(completed.load(Ordering::Relaxed), 0);
        assert!(game.story_walks.first().is_none_or(|job| job.t == c.ticks));
        let frame = game.modern_hd_map_presented.as_ref().unwrap();
        let backdrop = &frame.terrain[0];
        assert!(backdrop.scale_to_rect);
        write!(
            report,
            "{} {} {} {} {} {:x}",
            c.name,
            backdrop.rect.min.x,
            backdrop.rect.min.y,
            backdrop.rect.max.x,
            backdrop.rect.max.y,
            Sha256::digest(backdrop.image.read_pixels())
        )
        .unwrap();
        for unit in &frame.units {
            write!(report, " {}/{}/{}/{}", unit.group, unit.frame, unit.rect.min.x, unit.rect.min.y).unwrap();
        }
        report.push('\n');
        let world = game.modern_hd_map_gpu.as_ref().unwrap().world.as_ref().unwrap();
        let expected = std::fs::read(out.join(format!("go-story-motion-{}.rgba", c.name))).unwrap();
        assert_eq!(world.read_pixels().len(), expected.len());
        let diff = world.read_pixels().iter().zip(&expected).map(|(a, b)| u8::abs_diff(*a, *b)).max().unwrap();
        writeln!(hashes, "{} {diff} {:x} {:x}", c.name, Sha256::digest(&expected), Sha256::digest(world.read_pixels()))
            .unwrap();
        assert!(diff <= 1, "{}: maximum pixel difference {diff}; plan {report}", c.name);
        if c.ticks != 0 {
            for _ in c.ticks..25 {
                game.step_story_walks();
            }
            assert_eq!(completed.load(Ordering::Relaxed), 1);
            assert!(game.story_walks.is_empty());
            assert_eq!(
                (game.story_actors[actor].x, game.story_actors[actor].y, game.story_actors[actor].dir),
                (c.x, c.y, 2)
            );
        }
    }
    let expected = std::fs::read_to_string(out.join("go-story-motion.txt")).unwrap();
    std::fs::write(out.join("rust-story-motion.txt"), &report).unwrap();
    assert_eq!(report, expected);
    let previous = game.modern_hd_map_frame.clone();
    game.cam_x = f64::NAN;
    game.draw(&mut ui);
    assert!(!game.modern_hd_world_visible && game.load_err.contains("camera outside drawable map"));
    assert!(game.modern_hd_map_frame == previous);
    std::fs::write(out.join("rust-story-motion-world.txt"), &hashes).unwrap();
    println!("劇情走行、移鏡與視差的八張世界畫布與 Go 比對：\n{hashes}");
}
