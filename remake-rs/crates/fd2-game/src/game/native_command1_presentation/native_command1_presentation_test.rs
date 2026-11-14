//! D 組：第 1 指令演出的測試。Go 來源：`native_command1_presentation_test.go`。
//!
//! Go 的演出工作與計畫保存 `*battle.Unit`，測試的單位不在 `g.st` 裡；Rust 的單位是 `st.units` 的索引，
//! 所以夾具把建立計畫的 `State` 放進 `g.st`（行動者索引 0、目標索引 1）。Go 以閉包改寫區域變數
//! `callback`，Rust 的延續要 `Send`，改用 `Arc<AtomicI64>`。本檔的夾具也給其他指令演出測試共用。

use super::*;
use std::sync::Arc as StdArc;
use std::sync::atomic::{AtomicI64, Ordering};

/// 行動者索引。
pub(crate) const ACTOR: UnitIdx = 0;
/// 目標索引。
pub(crate) const TARGET: UnitIdx = 1;

/// Go 各指令測試共用的 `nativeCommandNPresentationTestPlan` 夾具（`native_command1_presentation_test.go:11` 等）：
/// 36 筆指令書只填 `record`，行動者在 (0,0)、HP 20，目標 class 5 在 (1,0)，`W: 2, H: 1`。
pub(crate) fn command_test_state(
    record: battle::NativeCommandRecord,
    (actor_camp, actor_mp): (battle::Camp, i64),
    (target_camp, target_hp): (battle::Camp, i64),
) -> battle::State {
    let mut book = vec![battle::NativeCommandRecord::default(); battle::NATIVE_COMMAND_RECORD_COUNT as usize];
    for (id, entry) in book.iter_mut().enumerate() {
        entry.id = id as i64;
    }
    let index = record.id as usize;
    book[index] = record;
    let mut actor = battle::Unit::default();
    (actor.camp, actor.x, actor.y, actor.hp, actor.mp, actor.on_field) = (actor_camp, 0, 0, 20, actor_mp, true);
    let mut target = battle::Unit::default();
    (target.camp, target.class_id, target.x, target.y, target.hp, target.on_field) =
        (target_camp, 5, 1, 0, target_hp, true);
    let mut state = battle::State::default();
    (state.w, state.h) = (2, 1);
    state.units = vec![actor, target];
    state.native_composition_event_bytes = vec![0; 2];
    state.native_command_book = book;
    state
}

/// 多數指令測試的計畫：我方 MP 5 對敵方 HP 103，抗性 `{5: 10}`，RNG 3，`PlanNativeCommandDamage`。
pub(crate) fn command_damage_test_plan(
    command_id: i64,
    record: battle::NativeCommandRecord,
) -> (battle::State, battle::NativeCommandDamagePlan) {
    let state = command_test_state(record, (battle::OWN, 5), (battle::ENEMY, 103));
    let plan = state.plan_native_command_damage(Some(ACTOR), Some(TARGET), command_id, &resist(), 3).expect("plan");
    (state, plan)
}

/// Go 的 `map[int]int{5: 10}`。
pub(crate) fn resist() -> BTreeMap<i64, i64> {
    BTreeMap::from([(5, 10)])
}

/// 指令書一筆（Go 的複合字面值，`TargetCode` 都是 0）。
pub(crate) fn command_record(
    id: i64,
    damage: i64,
    hit: i64,
    selection_mode: i64,
    effect_mode: i64,
    mp_cost: i64,
) -> battle::NativeCommandRecord {
    battle::NativeCommandRecord { id, damage, hit, selection_mode, effect_mode, mp_cost, target_code: 0 }
}

/// Go 的 `make([]*ebiten.Image, n)`：n 個 nil 畫格。
pub(crate) fn nil_frames(n: usize) -> Vec<Option<Arc<Picture>>> {
    vec![None; n]
}

/// Go 的 `[]battlepresent.NativeCommand0ActorFrame{{PublishMP: true}}`。
pub(crate) fn publish_mp_spec() -> Vec<battlepresent::NativeCommand0ActorFrame> {
    vec![battlepresent::NativeCommand0ActorFrame { publish_mp: true, ..Default::default() }]
}

/// 延續型別：收結果切片。
pub(crate) type ResultsCallback<T> = Box<dyn FnOnce(&mut Game, Vec<T>) + Send>;

/// 記錄延續收到幾筆結果（Go 的 `callback = len(results)`）。
pub(crate) fn callback_counter<T: 'static>() -> (StdArc<AtomicI64>, ResultsCallback<T>) {
    let counter = StdArc::new(AtomicI64::new(0));
    let seen = counter.clone();
    (counter, Box::new(move |_: &mut Game, results: Vec<T>| seen.store(results.len() as i64, Ordering::SeqCst)))
}

/// 讀計數。
pub(crate) fn count(counter: &StdArc<AtomicI64>) -> i64 {
    counter.load(Ordering::SeqCst)
}

/// `g.st.units[index]`。
pub(crate) fn unit(g: &Game, index: UnitIdx) -> &battle::Unit {
    &g.st.as_ref().expect("st").units[index]
}

fn test_plan() -> (battle::State, battle::NativeCommandDamagePlan) {
    command_damage_test_plan(1, command_record(1, 90, 100, 1, 0, 2))
}

/// Go：`native_command1_presentation_test.go:28` `TestNativeCommand1PresentationPublishesOnlyAfterDraw`
#[test]
fn test_native_command1_presentation_publishes_only_after_draw() {
    let (state, plan) = test_plan();
    let (counter, then) = callback_counter();
    let (actor, target) = (state.units[ACTOR].clone(), state.units[TARGET].clone());
    let mut g = Game { native_rng_state: plan.rng_before, st: Some(state), ..Default::default() };
    let mut job = NativeCommand1PresentationJob {
        actor: Some(ACTOR),
        plan: Some(plan.clone()),
        prelude: nil_frames(1),
        actor_black: nil_frames(1),
        actor_pulse: nil_frames(1),
        actor_specs: publish_mp_spec(),
        tail: nil_frames(1),
        actor_mp_before: actor.mp,
        actor_acted_before: actor.acted,
        target_hp_before: vec![target.hp],
        then: Some(then),
        ..Default::default()
    };
    for stage in 1..=plan.damage_stages {
        job.targets.push(NativeCommand1TargetFrame { target_index: 0, hp_stage: stage, ..Default::default() });
    }
    g.native_cmd1_presentation = Some(job);
    g.step_native_command1_presentation();
    let (a, t) = (unit(&g, ACTOR), unit(&g, TARGET));
    assert!(a.mp == plan.mp_before && !a.acted && t.hp == plan.results[0].hp_before, "未繪製的第 1 指令改變了戰鬥狀態");
    while let Some(job) = g.native_cmd1_presentation.as_mut() {
        job.drawn = true;
        g.step_native_command1_presentation();
    }
    let (a, t) = (unit(&g, ACTOR), unit(&g, TARGET));
    assert!(
        a.mp == plan.mp_after
            && a.acted
            && t.hp == plan.results[0].hp_after
            && g.native_rng_state == plan.rng_after
            && count(&counter) == 1,
        "第 1 指令交易未完成：mp={} acted={} hp={} rng={:#x} callback={}",
        a.mp,
        a.acted,
        t.hp,
        g.native_rng_state,
        count(&counter)
    );
}

/// Go：`native_command1_presentation_test.go:56` `TestNativeCommand1PresentationRollsBackRuntimeFailure`
#[test]
fn test_native_command1_presentation_rolls_back_runtime_failure() {
    let (state, plan) = test_plan();
    let (actor, target) = (state.units[ACTOR].clone(), state.units[TARGET].clone());
    let mut g = Game { native_rng_state: plan.rng_before, st: Some(state), ..Default::default() };
    g.native_cmd1_presentation = Some(NativeCommand1PresentationJob {
        actor: Some(ACTOR),
        plan: Some(plan.clone()),
        phase: NATIVE_COMMAND1_ACTOR,
        actor_black: nil_frames(1),
        actor_pulse: nil_frames(1),
        actor_specs: publish_mp_spec(),
        targets: vec![NativeCommand1TargetFrame { target_index: 0, hp_stage: 2, ..Default::default() }],
        tail: nil_frames(1),
        actor_mp_before: actor.mp,
        actor_acted_before: actor.acted,
        target_hp_before: vec![target.hp],
        drawn: true,
        ..Default::default()
    });
    g.step_native_command1_presentation();
    assert_eq!(unit(&g, ACTOR).mp, plan.mp_after, "MP 標記未發布");
    g.native_cmd1_presentation.as_mut().expect("job").drawn = true;
    g.step_native_command1_presentation();
    let (a, t) = (unit(&g, ACTOR), unit(&g, TARGET));
    assert!(
        g.native_cmd1_presentation.is_none()
            && a.mp == plan.mp_before
            && !a.acted
            && t.hp == plan.results[0].hp_before
            && g.native_rng_state == plan.rng_before
            && !g.load_err.is_empty(),
        "失敗回復不完整：mp={} acted={} hp={} rng={:#x} err={:?}",
        a.mp,
        a.acted,
        t.hp,
        g.native_rng_state,
        g.load_err
    );
}
