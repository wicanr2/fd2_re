//! 移植自 `native_postbattle_gate_test.go`。

use crate::fdother::*;

// Go: TestResolveNativePostbattleRoutePreservesHubAndPreparationBranches
#[test]
fn resolve_native_postbattle_route_preserves_hub_and_preparation_branches() {
    let mut table = vec![0u8; 30];
    table[22] = 1;
    table[27] = 1;
    let checks = [
        (0, 0, 0x2fc85, false, false),
        (0, 1, 0x2e341, false, false),
        (0, 3, 0x2e341, false, false),
        (0, 4, 0x3072f, false, false),
        (0, 2, 0x318ad, true, false),
        (22, 0, 0x318ad, true, true),
        (27, 4, 0x318ad, true, true),
    ];
    for (chapter, option, callee, prep, direct) in checks {
        let got = resolve_native_postbattle_route(chapter, &table, option);
        assert_eq!(
            got,
            Some(NativePostbattleRoute { selector: option, callee, preparation: prep, direct_preparation: direct }),
            "route({chapter},{option})"
        );
    }
    for (chapter, option) in [(-1, 0), (30, 0), (0, 5)] {
        assert_eq!(resolve_native_postbattle_route(chapter, &table, option), None, "invalid route({chapter},{option})");
    }
}

// Go: TestResolveNativePostbattleOutcomePreservesLoopAndRawReturns
#[test]
fn resolve_native_postbattle_outcome_preserves_loop_and_raw_returns() {
    let mut table = vec![0u8; 30];
    table[22] = 1;
    let checks = [
        (22, 4, 0, true, 0),
        (22, 4, 1, false, 0),
        (0, 2, 0, true, 0),
        (0, 2, 1, false, 0),
        (0, 0, 0, true, 0),
        (0, 0, 1, false, 1),
        (0, 1, -1, false, 1),
        (0, 3, 7, false, 1),
        (0, 4, 1, false, 1),
    ];
    for (chapter, selector, callee_result, repeat, return_value) in checks {
        let route = resolve_native_postbattle_route(chapter, &table, selector)
            .unwrap_or_else(|| panic!("route({chapter},{selector}) unavailable"));
        assert_eq!(
            resolve_native_postbattle_outcome(route, callee_result),
            Some(NativePostbattleOutcome { repeat, return_value }),
            "outcome({chapter},{selector},{callee_result})"
        );
    }
}

// Go: TestResolveNativePostbattleOutcomeRejectsInventedRoutes
#[test]
fn resolve_native_postbattle_outcome_rejects_invented_routes() {
    let bad = [
        NativePostbattleRoute::default(),
        NativePostbattleRoute { selector: 0, callee: 0x318ad, ..Default::default() },
        NativePostbattleRoute { selector: 2, callee: 0x318ad, ..Default::default() },
        NativePostbattleRoute { selector: 0, callee: 0x2fc85, direct_preparation: true, ..Default::default() },
    ];
    for route in bad {
        let got = resolve_native_postbattle_outcome(route, 1);
        assert_eq!(got, None, "invented route {route:?} accepted as {got:?}");
    }
}
