//! Go 來源：`remake/internal/campaign/native_continue_handoff.go`（110 行）。

use super::*;

/// MaterializeNativeContinueInteractiveBoundary records the second, verified
/// selector write performed by the native CONTINUE loader immediately before
/// returning to the shared 0x117E7 controller.  The opening redraw owns mode 0;
/// the controller owns the later interactive mode.  No opening redraw, clock
/// sample, scenario setup, or unit construction is performed here.
///
/// Go：`native_continue_handoff.go:15`，`func MaterializeNativeContinueInteractiveBoundary( state *battle.State, input fdsave.ContinueRuntimeInput, ) error`
pub fn materialize_native_continue_interactive_boundary(
    state: Option<&mut battle::State>,
    input: &save::ContinueRuntimeInput,
) -> Result<()> {
    todo!("campaign/native_continue_handoff.go:MaterializeNativeContinueInteractiveBoundary")
}

/// ValidateNativeContinueBattleHandoff is the fail-closed boundary between
/// the four typed CONTINUE adapters and the player-facing battle controller.
/// It validates only fields already proven by the native save/resource traces;
/// it does not infer a chapter, call Scenario.Setup, or consume the unresolved
/// owner list as if it were a gameplay result.
///
/// Go：`native_continue_handoff.go:53`，`func ValidateNativeContinueBattleHandoff( input fdsave.ContinueRuntimeInput, state *battle.State, scenario *battle.Scenario, ) error`
pub fn validate_native_continue_battle_handoff(
    input: &save::ContinueRuntimeInput,
    state: Option<&battle::State>,
    scenario: Option<&battle::Scenario>,
) -> Result<()> {
    todo!("campaign/native_continue_handoff.go:ValidateNativeContinueBattleHandoff")
}
