//! 移植自 `remake/internal/ending/montage_tail_player_test.go`。

use super::*;
use crate::ending::montage_tail::montage_tail_test::{tail, tail_visual_paths};
use crate::test_support::input;

fn shared(c: IndexedCompositor) -> SharedCompositor {
    std::sync::Arc::new(std::sync::Mutex::new(c))
}

/// Go `loadOriginalMontageTailPlayer`。
pub(crate) fn load_original_montage_tail_player() -> Option<(MontageTailPlayer, MontageTailAssets)> {
    for name in ["FDOTHER.DAT", "TAI.DAT", "BG.DAT"] {
        input("FD2_FLAME2_DIR", name)?;
    }
    let archive = input("FD2_FLAME2_DIR", "FDOTHER.DAT")?;
    let tail = tail()?;
    let visual = tail_visual_paths()?;
    let assets = load_montage_tail_assets_archive(&tail, &archive.to_string_lossy()).unwrap();
    let sets = load_montage_tail_visual_sets(&tail, &visual).unwrap();
    let player = new_montage_tail_player(&tail, &assets, &sets, Some(shared(new_indexed_compositor()))).unwrap();
    Some((player, assets))
}

fn vga(p: &MontageTailPlayer) -> (Vec<u8>, [u8; 768]) {
    let c = p.compositor.as_ref().unwrap().lock().unwrap();
    (c.vga.clone(), c.palette)
}

#[test]
fn test_montage_tail_player_uses_all_twenty_source_transactions_then_holds_final() {
    let Some((mut player, assets)) = load_original_montage_tail_player() else { return };
    for (index, set) in player.visual_sets.iter().enumerate() {
        for (name, animation) in [
            ("record0 base", &set.record0_figani_base),
            ("record0 aux", &set.record0_figani_aux),
            ("record1 base", &set.record1_figani_base),
            ("record1 aux", &set.record1_figani_aux),
        ] {
            let animation = animation.as_ref().unwrap();
            assert!(
                animation.header_byte1 == 0 && animation.header_byte2 as usize == animation.frames.len(),
                "tail entry {index} {name} header={}/{} frames={}",
                animation.header_byte1,
                animation.header_byte2,
                animation.frames.len()
            );
        }
    }
    let mut seen_segments = BTreeSet::new();
    let mut hashes = 0;
    let mut positive_waits = 0;
    let mut steps = 0;
    while steps < 4096 && !player.ready() {
        let (before_segment, before_phase) = (player.segment, player.phase);
        player
            .step()
            .unwrap_or_else(|err| panic!("tail step {steps} segment={before_segment} phase={}: {err}", before_phase.0));
        if before_phase != MONTAGE_TAIL_PHASE_INTRO && before_phase != MONTAGE_TAIL_PHASE_FINAL {
            seen_segments.insert(before_segment);
        }
        if player.delay_ticks > 0 {
            positive_waits += 1;
        }
        hashes += 1;
        steps += 1;
    }
    assert!(hashes > 0 && positive_waits > 0, "tail intro did not expose its source-proven wait");
    assert!(
        player.ready() && player.segment == 20 && seen_segments.len() == 20 && positive_waits >= 60,
        "tail completion ready={} segment={} seen={} waits={positive_waits}",
        player.ready(),
        player.segment,
        seen_segments.len()
    );
    assert!(hashes >= 100, "tail produced only {hashes} indexed presentations");
    let mut final_c = new_indexed_compositor();
    final_c.palette.copy_from_slice(&assets.loop_palette);
    final_c.baseline.copy_from_slice(&assets.loop_palette);
    final_c.baseline_known = true;
    assets.present_final(Some(&mut final_c)).unwrap();
    let (got_vga, got_palette) = vga(&player);
    assert!(
        got_vga == final_c.vga && got_palette == final_c.palette,
        "tail player did not finish on the source-derived FDOTHER#59 image"
    );
    let before = got_vga;
    let err = player.step();
    assert!(err.is_err() && vga(&player).0 == before, "completed tail accepted another step or mutated terminal frame");
}

#[test]
fn test_montage_tail_player_is_deterministic_for_original_assets() {
    let Some((mut first, _)) = load_original_montage_tail_player() else { return };
    let Some((mut second, _)) = load_original_montage_tail_player() else { return };
    let mut steps = 0;
    while steps < 4096 && (!first.ready() || !second.ready()) {
        assert!(
            first.ready() == second.ready()
                && first.phase == second.phase
                && first.segment == second.segment
                && first.frame == second.frame,
            "players diverged before step {steps}"
        );
        first.step().unwrap();
        second.step().unwrap();
        assert!(
            first.delay_ticks == second.delay_ticks && vga(&first) == vga(&second),
            "players diverged after step {steps}"
        );
        steps += 1;
    }
    assert!(first.ready() && second.ready(), "deterministic players exceeded bounded completion budget");
}

#[test]
fn test_montage_tail_player_fails_closed_before_partial_playback() {
    let tail = MontageTail {
        r#loop: MontageTailLoop { count: 1, ..Default::default() },
        raw_tables: MontageTailRawTable { record0_byte7: vec![1], record1_byte7: vec![2], global540_ff: vec![3] },
        ..Default::default()
    };
    let player =
        new_montage_tail_player(&tail, &MontageTailAssets::default(), &[], Some(shared(new_indexed_compositor())));
    assert!(player.is_err(), "incomplete tail produced a player");
}

fn solid_tail_fdother_frame(colour: u8) -> fdother::Frame {
    let mut f = fdother::Frame::default();
    (f.width, f.height) = (1, 1);
    f.pixels = vec![1, 0, 1, 0, 0, colour];
    f
}

fn solid_tail_figani_frame(colour: u8, raw4: u8, raw5: u8, delay: u8, raw7: u8) -> figani::Frame {
    figani::Frame {
        x: 10,
        y: 10,
        width: 1,
        height: 1,
        pixels: vec![colour],
        mask: vec![1],
        delay: i64::from(delay),
        raw_byte4: raw4,
        raw_byte5: raw5,
        raw_byte7: raw7,
    }
}

fn pair_player(entry: MontageTailEntry) -> MontageTailPlayer {
    MontageTailPlayer {
        compositor: Some(shared(new_indexed_compositor())),
        segment: 0,
        entries: vec![entry],
        ..Default::default()
    }
}

#[test]
fn test_montage_tail_native_pair_uses_raw_inner_count_z_order_and_base_scheduler() {
    let auxiliary = figani::Animation {
        header_byte2: 1,
        frames: vec![solid_tail_figani_frame(9, 0, 42, 2, 1)],
        ..Default::default()
    };
    let base = figani::Animation {
        header_byte2: 2,
        frames: vec![solid_tail_figani_frame(7, 0, 0, 1, 0), solid_tail_figani_frame(8, 0, 0, 1, 0)],
        ..Default::default()
    };
    let set = MontageTailVisualSet {
        tai: solid_tail_fdother_frame(2),
        bg: solid_tail_fdother_frame(3),
        record0_figani_aux: Some(auxiliary.clone()),
        record1_figani_base: Some(base.clone()),
        ..Default::default()
    };
    let mut p = pair_player(MontageTailEntry { record0_byte6: 2, ..Default::default() });
    for want_base in 7u8..=8 {
        let r = p.step_native_pair(&set, &auxiliary, &base, true, false);
        assert!(
            matches!(r, Ok((true, false))) && p.delay_ticks == 1 && p.sound_marker == 42,
            "pair step base={want_base} result={r:?} delay={} marker={}",
            p.delay_ticks,
            p.sound_marker
        );
        // side!=0 and raw +7 bit0=1 draws auxiliary first, then paired base.
        let got = vga(&p).0[(10 * WIDTH + 10) as usize];
        assert_eq!(got, want_base, "paired base scheduler pixel");
    }
    assert!(
        p.frame == 1 && p.inner == 0 && p.base_frame == 0,
        "pair state frame={} inner={} base={}",
        p.frame,
        p.inner,
        p.base_frame
    );
}

#[test]
fn test_montage_tail_second_pair_stops_after_last_raw_byte4_frame() {
    let auxiliary = figani::Animation {
        header_byte2: 3,
        frames: vec![
            solid_tail_figani_frame(9, 0, 0, 1, 0),
            solid_tail_figani_frame(10, 1, 0, 1, 0),
            solid_tail_figani_frame(11, 0, 0, 1, 0),
        ],
        ..Default::default()
    };
    let base = figani::Animation {
        header_byte2: 1,
        frames: vec![solid_tail_figani_frame(7, 0, 0, 1, 0)],
        ..Default::default()
    };
    let set = MontageTailVisualSet {
        tai: solid_tail_fdother_frame(2),
        bg: solid_tail_fdother_frame(3),
        ..Default::default()
    };
    let mut p = pair_player(MontageTailEntry { record1_byte6: 0, ..Default::default() });
    let r = p.step_native_pair(&set, &auxiliary, &base, false, true);
    assert!(matches!(r, Ok((true, false))), "first terminal pair step {r:?}");
    let r = p.step_native_pair(&set, &auxiliary, &base, false, true);
    assert!(matches!(r, Ok((true, true))), "last +4 terminal pair step {r:?}");
    let got = vga(&p).0[(20 * WIDTH + 24) as usize];
    assert!(
        got == 33 && p.effect_step == 4,
        "raw +4 displacement/opaque override pixel={got} effectStep={}",
        p.effect_step
    );
    assert_eq!(p.frame, 2, "terminal pair consumed frame index, want stop before frame2");
}

#[test]
fn test_montage_tail_rejects_nonzero_prelude_before_playback() {
    let tail = MontageTail {
        r#loop: MontageTailLoop { count: 1, ..Default::default() },
        raw_tables: MontageTailRawTable { record0_byte7: vec![1], record1_byte7: vec![2], global540_ff: vec![3] },
        ..Default::default()
    };
    let plans = tail.plan_visual_resources().unwrap();
    let frame = solid_tail_figani_frame(7, 0, 0, 1, 0);
    let valid = figani::Animation { header_byte2: 1, frames: vec![frame.clone()], ..Default::default() };
    let prelude = figani::Animation { header_byte1: 1, header_byte2: 1, frames: vec![frame], ..Default::default() };
    let set = MontageTailVisualSet {
        plan: plans[0],
        tai: solid_tail_fdother_frame(2),
        bg: solid_tail_fdother_frame(3),
        record0_figani_base: Some(valid.clone()),
        record0_figani_aux: Some(prelude),
        record1_figani_base: Some(valid.clone()),
        record1_figani_aux: Some(valid),
    };
    let assets = MontageTailAssets {
        loop_palette: vec![0; 768],
        intro: solid_tail_fdother_frame(1),
        loop_frames: vec![solid_tail_fdother_frame(4)],
        r#final: solid_tail_fdother_frame(5),
    };
    let player = new_montage_tail_player(&tail, &assets, &[set], Some(shared(new_indexed_compositor())));
    assert!(player.is_err(), "nonzero prelude entered tail player");
}
