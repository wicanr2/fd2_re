//! Go `native_physical_body_test.go` 中不經 `Draw`、不需要原版 oracle 錄製的測試。

use super::super::native_physical_scene::native_physical_scene_test::{
    physical_scene_test_game, require_physical_scene_pack,
};
use super::*;

/// Go：`native_physical_body_test.go:30`。明示的 renderer 診斷：Roll 與 counter 資格不由這個 fixture 驗收。
#[test]
fn test_native_physical_body_multi_strike_counter_and_pulse() {
    if !require_physical_scene_pack() {
        return;
    }
    let Some(mut g) = physical_scene_test_game() else {
        return;
    };
    let mut scene = g.prepare_native_physical_scene(Some(0), 1).unwrap().unwrap();
    let target_fig = g.st.as_ref().unwrap().units[1].battle_fig;
    let counter = figani::load_separated_resource(&separated_asset_path("animations"), target_fig * 3 + 1).unwrap();
    let counter_sounds = g.native_physical_sound_bank(&counter).unwrap();
    {
        let r = scene.body_resources.as_mut().unwrap();
        r.counter = Some(counter);
        r.counter_sounds = Arc::new(counter_sounds);
    }
    let strike = |damage: i64, crit: bool, status: bool, defender_hp: i64| {
        let mut s = battle::NativePhysicalStrike::default();
        (s.roll.damage, s.roll.crit, s.roll.status, s.defender_hp) = (damage, crit, status, defender_hp);
        s
    };
    let mut result = battle::NativePhysicalAttackResult::default();
    (result.attack_strikes, result.counter, result.counter_strikes) = (
        vec![strike(7, true, true, 73), strike(3, false, false, 70)],
        Some(battle::AttackResult::default()),
        vec![strike(9, false, false, 71)],
    );
    g.attach_native_physical_body(Some(&mut scene), &result).unwrap();
    let jobs = &scene.body.as_ref().unwrap().jobs;
    let (mut negative_hit, mut positive_hit, mut green, mut white, mut restores) = (false, false, false, false, 0);
    for (i, job) in jobs.iter().enumerate() {
        if let Some(present) = job.present.as_ref() {
            if present.opaque_fill == 33 {
                negative_hit = negative_hit || present.dx == -14;
                positive_hit = positive_hit || present.dx == 14;
            }
            job.indexed().unwrap();
        }
        if job.palette0.is_some_and(|c| c.g == 130) {
            green = true;
            assert!(
                job.wait_millis == 20.0
                    && i + 3 < jobs.len()
                    && jobs[i + 1].palette0.unwrap().g == 0
                    && jobs[i + 2].palette0.unwrap().r == 255
                    && jobs[i + 2].wait_millis == 20.0
                    && jobs[i + 3].wait_millis >= 94.0,
                "status20→black→crit20→black40+BIOS sequence changed"
            );
        }
        white = white || job.palette0.is_some_and(|c| c.r == 255);
        if !job.pixels.is_empty() {
            restores += 1;
        }
    }
    assert!(
        negative_hit && positive_hit && green && white && restores == 1 && scene.prelude.len() == 9,
        "full main/counter lost raw side, pulses or single final restore"
    );
    let (mut actor_record, mut target_record) = (scene.actor_record.clone(), scene.target_record.clone());
    actor_record[64..66].copy_from_slice(&71u16.to_le_bytes());
    target_record[64..66].copy_from_slice(&70u16.to_le_bytes());
    let mut want = g.native_physical_base(&mut scene, &actor_record, &target_record).unwrap();
    let r = scene.body_resources.as_ref().unwrap();
    let mut platform = r.platform.clone();
    (platform.x, platform.y) = (164, 157);
    platform.blit(&mut want, 320, -1).unwrap();
    for idle in [r.target_idle.as_ref().unwrap(), r.actor_idle.as_ref().unwrap()] {
        idle.frames[0].blit_at(&mut want, 320).unwrap();
    }
    let jobs = &scene.body.as_ref().unwrap().jobs;
    assert_eq!(jobs[jobs.len() - 1].pixels, want, "counter final restore did not preserve both HP panels");
    let units = &g.st.as_ref().unwrap().units;
    assert!(
        units[0].hp == 80 && units[1].hp == 80 && g.native_rng_state == 17791,
        "renderer changed live units or RNG"
    );
}

/// D 組補充：`stepNativePhysicalBodyMillis` 每個有畫面的工作都要等呈現回報（`drawn`）才累積時間，cue 只播一次；
/// 全部走完後再等 caller 的六個 BIOS tick（`returnWaitMillis`）。沒有原生地圖素材時直接收尾。
#[test]
fn step_native_physical_body_millis_waits_for_presents_and_return_ticks() {
    let present =
        |wait: f64| NativePhysicalBodyJob { pixels: vec![1; 64000], cue: -1, wait_millis: wait, ..Default::default() };
    let mut scene = NativePhysicalScene {
        body: Some(NativePhysicalBodyPlayback {
            jobs: vec![present(30.0), NativePhysicalBodyJob { cue: -1, ..Default::default() }, present(10.0)],
            last_present: 2,
            return_wait_millis: 6.0 * indexedmap::PHASE_BANNER_STEP_MILLIS,
            ..Default::default()
        }),
        ..Default::default()
    };
    let mut g = Game::default();
    // 沒有呈現回報：不前進、不累積。
    g.step_native_physical_body_millis(&mut scene, 16.0).unwrap();
    let p = scene.body.as_ref().unwrap();
    assert_eq!((p.index, p.elapsed_millis), (0, 0.0));
    // 第一格要 30 毫秒：第一次回報累積 16，第二次 32 才換格，餘數 2 帶到下一格。
    scene.body.as_mut().unwrap().drawn = true;
    g.step_native_physical_body_millis(&mut scene, 16.0).unwrap();
    assert_eq!(scene.body.as_ref().unwrap().index, 0);
    g.step_native_physical_body_millis(&mut scene, 16.0).unwrap();
    let p = scene.body.as_ref().unwrap();
    // 換格後 cue-only 工作（沒有畫面、等待 0）同一次就通過，餘數因等待為 0 歸零；停在最後一格等它的回報。
    assert_eq!((p.index, p.drawn), (2, false));
    assert_eq!(p.elapsed_millis, 0.0);
    scene.body.as_mut().unwrap().drawn = true;
    g.step_native_physical_body_millis(&mut scene, 16.0).unwrap();
    // 最後一格 10 毫秒完成，餘數 6 留著，接著在六個 BIOS tick 的返回等待裡。
    let p = scene.body.as_ref().unwrap();
    assert_eq!(p.index, 3);
    assert!((p.elapsed_millis - 6.0).abs() < 1e-9);
    assert!(p.elapsed_millis < p.return_wait_millis);
    assert!(!p.return_ramp_started);
}
