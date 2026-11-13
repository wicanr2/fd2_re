//! 移植自 `remake/internal/battle/native_command26_test.go`。

use super::*;

/// Go 測試的 `countingZeroSource`：`Int63()` 永遠回 0 並計數。Go `(*Rand).Intn(n)` 對這種來源
/// 每次只取一次 `Int63` 並回傳 0（`Int31n` 的拒絕取樣不會重抽，因為 0 不超過上限），所以這裡
/// 以「每次 `intn` 計一次、回 0」重現同樣的序列與次數。
struct CountingZeroSource {
    draws: i64,
}

impl GoRand for CountingZeroSource {
    fn intn(&mut self, n: i64) -> i64 {
        assert!(n > 0, "invalid argument to Intn");
        self.draws += 1;
        0
    }
}

fn native_command_application_book(id: i64) -> Vec<NativeCommandRecord> {
    let mut book: Vec<NativeCommandRecord> =
        (0..36).map(|i| NativeCommandRecord { id: i, ..Default::default() }).collect();
    book[id as usize] =
        NativeCommandRecord { id, selection_mode: 1, effect_mode: 0, mp_cost: 2, target_code: 0, ..Default::default() };
    book
}

/// Go：`passingNativeApplicationRNG`
fn passing_native_application_rng() -> GoMathRand {
    for seed in 0..1000i64 {
        let mut r = GoMathRand::new(seed);
        if r.intn(100) < 50 {
            return GoMathRand::new(seed);
        }
    }
    panic!("no deterministic application RNG seed found");
}

fn application_state(target: Unit, id: i64) -> State {
    State {
        w: 2,
        h: 1,
        units: vec![Unit { camp: OWN, on_field: true, hp: 20, mp: 5, x: 0, y: 0, ..Default::default() }, target],
        native_composition_event_bytes: vec![0; 2],
        native_command_book: native_command_application_book(id),
        ..Default::default()
    }
}

fn enemy_target() -> Unit {
    Unit { camp: ENEMY, class_id: 2, on_field: true, hp: 20, x: 1, y: 0, ..Default::default() }
}

/// Go：`TestExecuteNativeCommandApplicationUsesThreeRNGDrawsAndNativeDamage`
#[test]
fn execute_native_command_application_uses_three_rng_draws_and_native_damage() {
    let mut st = application_state(enemy_target(), 26);
    let mut rng = passing_native_application_rng();
    let got = st.execute_native_command_application(Some(0), Some(1), 26, Some(&mut rng));
    assert!(
        matches!(&got, Ok(g) if g.len() == 1 && g[0].applied && g[0].damage == 9 && (2..=5).contains(&g[0].duration)),
        "application = {got:?}"
    );
    let got = got.unwrap();
    let (actor, target) = (&st.units[0], &st.units[1]);
    assert!(
        target.hp == 11 && target.native_transient[3] == got[0].duration && actor.mp == 3 && actor.acted,
        "post application actor={actor:?} target={target:?}"
    );
}

/// Go：`TestExecuteNativeCommandApplicationConsumesGateDamageMarkerDraws`
#[test]
fn execute_native_command_application_consumes_gate_damage_marker_draws() {
    let mut st = application_state(enemy_target(), 22);
    let mut source = CountingZeroSource { draws: 0 };
    let got = st.execute_native_command_application(Some(0), Some(1), 22, Some(&mut source));
    assert!(matches!(&got, Ok(g) if g.len() == 1 && g[0].applied), "application = {got:?}");
    let got = got.unwrap();
    assert!(
        source.draws == 3 && got[0].damage == 9 && got[0].duration == 2,
        "draws={} result={:?}",
        source.draws,
        got[0]
    );
}

/// Go：`TestExecuteNativeCommandApplicationKeepsRawGateButConsumesSuccessfulCommand`
#[test]
fn execute_native_command_application_keeps_raw_gate_but_consumes_successful_command() {
    let target = Unit { native_transient: [0, 0, 0, 4, 0, 0], ..enemy_target() };
    let mut st = application_state(target, 26);
    let mut rng = GoMathRand::new(1);
    let got = st.execute_native_command_application(Some(0), Some(1), 26, Some(&mut rng));
    let (actor, target) = (&st.units[0], &st.units[1]);
    assert!(
        matches!(&got, Ok(g) if g.len() == 1 && !g[0].applied)
            && target.hp == 20
            && target.native_transient[3] == 4
            && actor.mp == 3
            && actor.acted,
        "existing raw interval gate = {got:?}, actor={actor:?} target={target:?}"
    );
}

/// Go：`TestExecuteNativeCommandApplicationSupportsRecoveredIDTwentyTwo`
#[test]
fn execute_native_command_application_supports_recovered_id_twenty_two() {
    let mut st = application_state(enemy_target(), 22);
    let mut rng = passing_native_application_rng();
    let got = st.execute_native_command_application(Some(0), Some(1), 22, Some(&mut rng));
    let (actor, target) = (&st.units[0], &st.units[1]);
    assert!(
        matches!(&got, Ok(g) if g.len() == 1 && g[0].applied && g[0].offset == 0x27
            && target.native_transient[5] == g[0].duration)
            && target.hp == 11,
        "ID22 application = {got:?} actor={actor:?} target={target:?}"
    );
}

/// Go：`TestExecuteNativeCommandApplicationRejectsUnknownIDBeforeMutation`
///
/// Go 以空 `State` 呼叫且 actor 不在 `Units`；Rust 依模組文件第 6 點先 push 再傳索引。
#[test]
fn execute_native_command_application_rejects_unknown_id_before_mutation() {
    let mut st = State { units: vec![Unit { mp: 5, ..Default::default() }], ..Default::default() };
    let mut rng = GoMathRand::new(1);
    let err = st.execute_native_command_application(Some(0), None, 25, Some(&mut rng));
    let actor = &st.units[0];
    assert!(
        err.is_err() && actor.mp == 5 && !actor.acted,
        "unknown application ID must fail closed: actor={actor:?} err={err:?}"
    );
}

/// Go：`TestExecuteNativeCommandApplicationPreflightsBeforeMPDebit`
#[test]
fn execute_native_command_application_preflights_before_mp_debit() {
    let target = Unit { hp: -1, ..enemy_target() };
    let mut st = application_state(target, 22);
    let mut rng = GoMathRand::new(1);
    assert!(
        st.execute_native_command_application(Some(0), Some(1), 22, Some(&mut rng)).is_err(),
        "invalid target HP state accepted"
    );
    let (actor, target) = (&st.units[0], &st.units[1]);
    assert!(
        actor.mp == 5 && !actor.acted && target.hp == -1 && target.native_transient[5] == 0,
        "failed preflight mutated actor={actor:?} target={target:?}"
    );
}
