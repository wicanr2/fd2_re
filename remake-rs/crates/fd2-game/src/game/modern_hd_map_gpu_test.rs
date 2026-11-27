//! 世界圖重用：沿用的背景地形層與整張沿用，結果都要和全新合成逐位元組相同。

use super::*;

fn solid(width: i64, height: i64, rgba: [u8; 4]) -> Arc<Picture> {
    Arc::new(Picture::from_nrgba(width, height, &rgba.repeat((width * height) as usize)))
}

fn gradient(width: i64, height: i64, seed: u8) -> Arc<Picture> {
    let mut pix = Vec::with_capacity((width * height * 4) as usize);
    for y in 0..height {
        for x in 0..width {
            let a = if (x + y) % 7 == 0 { 0 } else { 255 };
            pix.extend_from_slice(&[(x as u8).wrapping_add(seed), (y as u8).wrapping_mul(3), seed, a]);
        }
    }
    Arc::new(Picture::from_nrgba(width, height, &pix))
}

fn terrain(seed: u8) -> Vec<ModernHDTerrainDraw> {
    (0..6)
        .map(|i| ModernHDTerrainDraw {
            image: gradient(144, 144, seed.wrapping_add(i as u8)),
            rect: rect(24 + i * 144, 24, 24 + (i + 1) * 144, 168),
            frame: i,
            work_offset: 0,
            scale_to_rect: false,
        })
        .collect()
}

fn frame(terrain: Vec<ModernHDTerrainDraw>, unit_frame: i64, x: i64) -> ModernHDMapFrame {
    ModernHDMapFrame {
        background: Some(gradient(320, 200, 9)),
        ui: Some(solid(320, 200, [0, 0, 0, 0])),
        units: vec![ModernHDMapUnitDraw {
            group: 3,
            frame: unit_frame,
            rect: rect(x, 200, x + 144, 344),
            work_offset: 0,
            palette_band: unit_frame == 1,
        }],
        foreground: vec![ModernHDMapForegroundDraw {
            image: gradient(24, 24, 77),
            rect: rect(300, 300, 444, 444),
            source_frame: 0,
            work_offset: 0,
            blit_mode: 0,
            hd_image: None,
        }],
        terrain,
        ..Default::default()
    }
}

#[test]
fn reused_world_matches_fresh_composition() {
    let theme = ModernStoryPortraitSet {
        has_hd: true,
        map_sprites_hd: [(3, vec![Some(gradient(144, 144, 1)), Some(gradient(144, 144, 2))])].into(),
        ..Default::default()
    };
    let ground = terrain(10);
    let mut with_range = frame(ground.clone(), 0, 120);
    with_range.range = Some(gradient(320, 200, 40));
    let mut tinted = frame(ground.clone(), 1, 120);
    tinted.palette_add6 = 20;
    let sequence: Vec<(ModernHDMapFrame, Option<(&str, i64)>)> = vec![
        (frame(ground.clone(), 0, 120), None),
        (frame(ground.clone(), 0, 120), None),
        (frame(ground.clone(), 1, 120), None),
        (frame(ground.clone(), 1, 168), None),
        (with_range, None),
        (frame(terrain(30), 1, 168), None),
        (frame(ground.clone(), 0, 120), Some(("第一章", -1))),
        (frame(ground.clone(), 0, 120), Some(("第一章", 3))),
        (tinted, None),
        (frame(ground, 0, 120), None),
    ];
    let mut reused = ModernHDMapGPU::default();
    let mut previous: Option<(Arc<Picture>, usize)> = None;
    for (index, (input, banner)) in sequence.iter().enumerate() {
        let ui = reused.compose(input, &theme, *banner).unwrap();
        let mut fresh = ModernHDMapGPU::default();
        if let Some((name, step)) = banner
            && *step >= 0
        {
            // 橫幅馬賽克取自同名第一次的完整畫面；全新合成器先以同一畫面建立來源。
            fresh.compose(input, &theme, Some((name, -1))).unwrap();
        }
        let expected_ui = fresh.compose(input, &theme, *banner).unwrap();
        assert_eq!(ui, expected_ui, "frame {index}");
        let world = reused.world.clone().unwrap();
        assert_eq!(world.pix, fresh.world.as_ref().unwrap().pix, "frame {index}");
        if let Some((last, last_index)) = &previous
            && sequence[*last_index] == sequence[index]
        {
            assert!(Arc::ptr_eq(last, &world), "identical input {index} recomposed");
        }
        previous = Some((world, index));
    }
}

#[test]
fn failed_composition_does_not_reuse_stale_world() {
    let theme = ModernStoryPortraitSet {
        has_hd: true,
        map_sprites_hd: [(3, vec![Some(gradient(144, 144, 1)), None])].into(),
        ..Default::default()
    };
    let mut renderer = ModernHDMapGPU::default();
    let good = frame(terrain(10), 0, 120);
    renderer.compose(&good, &theme, None).unwrap();
    let before = renderer.world.clone().unwrap();
    assert!(renderer.compose(&frame(terrain(10), 1, 120), &theme, None).is_err());
    assert!(Arc::ptr_eq(&before, renderer.world.as_ref().unwrap()));
    let mut bad_overlay = good.clone();
    bad_overlay.hud.push(ModernHDMapOverlayDraw::default());
    assert!(renderer.compose(&bad_overlay, &theme, None).is_err());
    // 失敗後同一份好輸入要重新合成（上一筆鍵已清除），內容仍與原圖相同。
    renderer.compose(&good, &theme, None).unwrap();
    assert_eq!(renderer.world.as_ref().unwrap().pix, before.pix);
}
