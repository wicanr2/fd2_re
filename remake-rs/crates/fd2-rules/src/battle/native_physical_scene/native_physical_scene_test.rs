//! 移植自 `remake/internal/battle/native_physical_scene_test.go`。

use super::*;

fn physical_scene_unit(side: u8, figure: i64, race: u8, class: u8) -> Unit {
    Unit {
        has_native_record_byte6: true,
        native_record_byte6: side,
        has_battle_fig: true,
        battle_fig: figure,
        has_native_record_race: true,
        native_record_race: race,
        has_native_record_class: true,
        native_record_class: class,
        ..Default::default()
    }
}

fn sel(
    primary_bg: u8,
    secondary_bg: u8,
    base_bg: u8,
    tai: u8,
    has_separate_backgrounds: bool,
) -> NativePhysicalSceneSelection {
    NativePhysicalSceneSelection { primary_bg, secondary_bg, base_bg, tai, has_separate_backgrounds }
}

/// Go：`TestNativePhysicalSceneChapter12NormalTuple`
#[test]
fn native_physical_scene_chapter12_normal_tuple() {
    let actor = physical_scene_unit(2, 4, 1, 3);
    let target = physical_scene_unit(0, 102, 8, 2);
    let (before_actor, before_target) = (actor.clone(), target.clone());
    let got = select_native_physical_scene(0, Some(&actor), Some(&target), [0, 2, 5, 0], [0, 0, 50, 0], 0);
    let want = sel(50, 5, 50, 5, false);
    assert!(matches!(got, Ok(g) if g == want), "normal raw tuple = {got:?}, want {want:?}");
    assert!(actor == before_actor && target == before_target, "scene selection modified raw units");
}

/// Go：`TestNativePhysicalSceneSidesAndPreludeBase`
#[test]
fn native_physical_scene_sides_and_prelude_base() {
    for side in [0u8, 1, 2, 255] {
        for flag in [0u8, 1, 2] {
            let (actor, target) = (physical_scene_unit(side, 4, 1, 3), physical_scene_unit(2, 102, 8, 2));
            let got = select_native_physical_scene(3, Some(&actor), Some(&target), [0, 0, 5, 0], [0, 0, 50, 0], flag);
            let mut want = sel(5, 50, 5, 50, flag != 0);
            if side != 0 {
                (want.primary_bg, want.secondary_bg, want.tai) = (50, 5, 5);
                if flag == 0 {
                    want.base_bg = 50;
                }
            }
            assert!(matches!(got, Ok(g) if g == want), "raw side={side} flag={flag}: {got:?}, want {want:?}");
        }
    }
}

/// Go：`TestNativePhysicalSceneGateAndZeroInitialTAI`
#[test]
fn native_physical_scene_gate_and_zero_initial_tai() {
    let cases = [
        (27i64, 3u8, 19u8, true),
        (29, 4, 18, true),
        (29, 5, 18, true),
        (29, 3, 18, false),
        (29, 6, 18, false),
        (28, 4, 19, false),
    ];
    for (figure, race, class, gate) in cases {
        for initial in [0u8, 3] {
            let mut actor = physical_scene_unit(2, figure, race, class);
            let target = physical_scene_unit(0, 102, 8, 2);
            let got =
                select_native_physical_scene(initial, Some(&actor), Some(&target), [0, 0, 5, 0], [0, 0, 50, 0], 0);
            let (mut want_bg, mut want_tai) = (5u8, 5u8);
            if gate {
                want_tai = initial;
                if initial != 0 {
                    want_bg = initial;
                }
            }
            assert!(
                matches!(got, Ok(g) if g.primary_bg == 50 && g.secondary_bg == want_bg && g.tai == want_tai),
                "gate case={:?} initial={initial}: {got:?}, want BG={want_bg} TAI={want_tai}",
                (figure, race, class, gate)
            );
            // 同一 gate 放在 primary，仍須排除 figure28，initial0時取地形。
            actor.native_record_byte6 = 0;
            let got =
                select_native_physical_scene(initial, Some(&actor), Some(&target), [0, 0, 5, 0], [0, 0, 50, 0], 0);
            assert!(
                matches!(got, Ok(g) if g.primary_bg == want_bg && g.secondary_bg == 50 && g.tai == 50),
                "primary gate case={:?} initial={initial}: {got:?}",
                (figure, race, class, gate)
            );
        }
    }
}

/// Go：`TestNativePhysicalSceneMissingRawProvenanceRejects`
#[test]
fn native_physical_scene_missing_raw_provenance_rejects() {
    let mutations: [fn(&mut Unit); 6] = [
        |u| u.has_native_record_byte6 = false,
        |u| u.has_battle_fig = false,
        |u| u.has_native_record_race = false,
        |u| u.has_native_record_class = false,
        |u| u.battle_fig = -1,
        |u| u.battle_fig = 256,
    ];
    for mutate in mutations {
        for change_actor in [false, true] {
            let (mut actor, mut target) = (physical_scene_unit(2, 4, 1, 3), physical_scene_unit(0, 102, 8, 2));
            if change_actor {
                mutate(&mut actor);
            } else {
                mutate(&mut target);
            }
            assert!(
                select_native_physical_scene(0, Some(&actor), Some(&target), [0; 4], [0; 4], 0).is_err(),
                "missing raw input accepted"
            );
        }
    }
    let (actor, target) = (physical_scene_unit(2, 4, 1, 3), physical_scene_unit(0, 102, 8, 2));
    assert!(
        select_native_physical_scene(4, Some(&actor), Some(&target), [0; 4], [0; 4], 0).is_err(),
        "initial outside verified chapter table accepted"
    );
}
