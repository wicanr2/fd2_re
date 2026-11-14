//! Go `native_physical_scene_test.go` 中不經 `Draw`、不需要原版 oracle 錄製的測試。

use super::*;

/// `battle::Unit`／`battle::State`／`fdother::Frame` 有私有欄位，不能用結構更新語法；先 `default()` 再一次寫入欄位。
macro_rules! build {
    ($t:ty { $($f:ident : $v:expr),* $(,)? }) => {{
        let mut x = <$t>::default();
        ($(x.$f,)*) = ($($v,)*);
        x
    }};
}

/// Go `physicalSceneTestGame`（`native_physical_scene_test.go:25`）。Go 的單位指標在 Rust 是 `st.units` 的索引
/// （0 攻方索爾、1 守方敵軍）。`native_command_scene.json` 取自受版控的 `remake/assets`。
pub(crate) fn physical_scene_test_game() -> Option<Game> {
    let scene_table = crate::test_support::remake_asset("data/native_command_scene.json")?;
    let actor = build!(battle::Unit {
        name: String::from("索爾"),
        camp: battle::OWN,
        on_field: true,
        has_native_record_byte8: true,
        hp: 80,
        max_hp: 80,
        battle_fig: 0,
        has_battle_fig: true,
        has_native_record_byte5: true,
        atk_min: 1,
        atk_max: 1,
        native_record_byte6: 2,
        has_native_record_byte6: true,
        native_record_race: 1,
        has_native_record_race: true,
        native_record_class: 1,
        has_native_record_class: true,
        dir: 2
    });
    let target = build!(battle::Unit {
        name: String::from("敵軍"),
        camp: battle::ENEMY,
        on_field: true,
        x: 1,
        native_record_byte8: 78,
        has_native_record_byte8: true,
        hp: 80,
        max_hp: 80,
        battle_fig: 78,
        has_battle_fig: true,
        has_native_record_byte5: true,
        has_native_record_byte6: true,
        native_record_race: 1,
        has_native_record_race: true,
        native_record_class: 2,
        has_native_record_class: true
    });
    let st = build!(battle::State {
        w: 2,
        h: 1,
        units: vec![actor, target],
        has_native_map_view_state: true,
        native_map_range_mode: 1,
        has_native_map_range_mode_state: true,
        native_tile_blit_modes: vec![255, 0]
    });
    Some(Game {
        st: Some(st),
        m: Some(MapData {
            w: 2,
            h: 1,
            tiles: vec![0, 0],
            native_terrain_control: vec![0, 0, 55, 0],
            ..Default::default()
        }),
        handler_chapter: 7,
        locale_id: String::from("zh-Hant"),
        native_ui_palette: load_native_ui_palette(),
        sel: Some(0),
        moved: true,
        cur_x: 1,
        rng: Some(battle::GoMathRand::new(73)),
        native_rng_state: 17791,
        native_command_scene: Some(battle::load_native_command_scene_table(&scene_table).expect("場景表")),
        ..Default::default()
    })
}

/// Go `requirePhysicalScenePack`：分離素材包要有 FIGANI_001。
pub(crate) fn require_physical_scene_pack() -> bool {
    crate::test_support::input("FD2_ASSET_PACK", "animations/FIGANI_001/animation.json").is_some()
}

/// Go：`native_physical_scene_test.go:61`
#[test]
fn test_native_physical_scene_matches_chapter8_normal_oracle_selection() {
    if !require_physical_scene_pack() {
        return;
    }
    let Some(mut g) = physical_scene_test_game() else {
        return;
    };
    let before = g.st.as_ref().unwrap().units.clone();
    let scene = g.prepare_native_physical_scene(Some(0), 1).unwrap().expect("有原生視圖時要建場景");
    let background = scene.background.as_ref().unwrap();
    let pedestal = scene.pedestal.as_ref().unwrap();
    assert!(
        scene.selection.base_bg == 55
            && scene.selection.tai == 55
            && !scene.selection.has_separate_backgrounds
            && (background.width, background.height) == (320, 100)
            && (pedestal.width, pedestal.height) == (154, 42),
        "original ch08 BG55/TAI55 tuple changed: {:?}",
        scene.selection
    );
    assert_eq!(g.st.as_ref().unwrap().units, before, "scene preparation changed battle input");
    assert_eq!(g.native_rng_state, 17791);
    // D 組補充：前奏 9 幀是索引色畫格（GPU 貼圖在 R5），本體資源已預檢。
    assert_eq!(scene.prelude_images.len(), 9);
    assert!(scene.prelude_images.iter().all(|f| (f.width, f.height, f.pix.len()) == (320, 200, 64000)));
    assert!(scene.body_resources.as_ref().is_some_and(|r| r.base.len() == 64000 && r.attack.is_some()));
}

/// Go：`native_physical_scene_test.go:79`
#[test]
fn test_native_physical_terrain_uses_mutable_cell_and_rejects_malformed_grid() {
    let Some(mut g) = physical_scene_test_game() else {
        return;
    };
    g.m.as_mut().unwrap().native_terrain_control.extend_from_slice(&[0, 2, 5, 0]);
    let st = g.st.as_mut().unwrap();
    st.has_native_map_event_grid = true;
    st.native_map_event_grid = vec![0; 12];
    (st.native_map_event_grid[0], st.native_map_event_grid[2]) = (2, 1);
    crate::test_support::put16(&mut st.native_map_event_grid, 4, 0xfc01);
    assert_eq!(
        g.native_physical_terrain_control(Some(0)).unwrap(),
        [0, 2, 5, 0],
        "scene read initial tile or high bits"
    );
    g.st.as_mut().unwrap().native_map_event_grid[0] = 3;
    assert!(g.native_physical_terrain_control(Some(0)).is_err(), "bad map header fell back to initial tiles");
    g.st.as_mut().unwrap().native_map_event_grid.truncate(4);
    assert!(g.native_physical_terrain_control(Some(0)).is_err(), "truncated mutable map fell back to initial tiles");
}

/// Go：`native_physical_scene_test.go:100`
#[test]
fn test_native_physical_surface_retains_opaque_index_zero() {
    let mut palette: ColorPalette = (0..256).map(|i| ColorRgba { r: i as u8, a: 255, ..Default::default() }).collect();
    palette[0] = ColorRgba { r: 17, g: 23, b: 31, a: 255 };
    let mut frame = build!(fdother::Frame { width: 2, height: 1, indexed: vec![0, 1], mask: vec![255, 0] });
    let img = native_physical_surface_rgba(&frame, &palette).unwrap().unwrap();
    assert_eq!(img.pix, vec![17, 23, 31, 255, 0, 0, 0, 0], "palette zero or retained span changed");
    frame.mask[1] = 1;
    assert!(native_physical_surface_rgba(&frame, &palette).is_err(), "invalid mask accepted");
}

/// Go：`native_physical_scene_test.go:156` 的 player 分支（mode11 分支經 `executeNativeAIMode11Physical`，`r8`）。
#[test]
fn test_native_physical_scene_missing_raw_name_stops_before_settlement() {
    if !require_physical_scene_pack() {
        return;
    }
    let Some(mut g) = physical_scene_test_game() else {
        return;
    };
    g.ensure_native_attack_presentation(0, 78).unwrap();
    g.st.as_mut().unwrap().units[1].has_native_record_byte8 = false;
    let before = g.st.as_ref().unwrap().units.clone();
    g.confirm();
    assert!(g.load_err.contains("raw record byte +8"), "missing panel provenance changed transaction: {}", g.load_err);
    assert!(g.atk.is_none() && g.native_rng_state == 17791);
    assert_eq!(g.st.as_ref().unwrap().units, before);
    assert_eq!(
        g.rng.as_mut().unwrap().int63(),
        battle::GoMathRand::new(73).int63(),
        "missing panel provenance consumed RNG"
    );
}

/// Go：`native_physical_scene_test.go:183` 的 player 分支。
#[test]
fn test_native_physical_scene_return_dac_stops_before_settlement() {
    if !require_physical_scene_pack() {
        return;
    }
    for failure in ["busy", "baseline"] {
        let Some(mut g) = physical_scene_test_game() else {
            return;
        };
        g.native_map_assets = Some(NativeMapAssets { palette_dac: vec![0; 768], ..Default::default() });
        if failure == "busy" {
            g.native_palette_ramp = Some(NativePaletteRampJob::default());
        } else {
            g.native_map_assets.as_mut().unwrap().palette_dac = Vec::new();
        }
        g.ensure_native_attack_presentation(0, 78).unwrap();
        let before = g.st.as_ref().unwrap().units.clone();
        g.confirm();
        assert!(g.load_err.contains("DAC owner or baseline unavailable"), "{failure}: DAC預檢改寫交易：{}", g.load_err);
        assert!(g.atk.is_none() && g.native_rng_state == 17791);
        assert_eq!(g.st.as_ref().unwrap().units, before);
        assert_eq!(g.rng.as_mut().unwrap().int63(), battle::GoMathRand::new(73).int63(), "DAC預檢消耗Go RNG");
    }
}
