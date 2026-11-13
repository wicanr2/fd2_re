//! 移植自 `remake/internal/battle/native_command_modifier_test.go`。

use super::*;

/// Go：`TestApplyNativeCommandModifierDispatchesVerifiedBranches`
#[test]
fn apply_native_command_modifier_dispatches_verified_branches() {
    for command_id in [17, 18, 19] {
        let mut records = vec![0u8; NATIVE_RECORD_SIZE as usize];
        let result = apply_native_command_modifier(&mut records, &[0], command_id, 0)
            .unwrap_or_else(|err| panic!("command {command_id}: {err}"));
        // RNG state is allowed to remain zero only when a branch is marked;
        // this assertion mainly verifies the command-specific union below.
        assert!(
            !(result.command_id != command_id || result.rng_state == 0 && command_id == 17),
            "command {command_id} result={result:?}"
        );
        assert!(
            !(command_id < 19 && result.word_steps.len() != 1),
            "command {command_id} word steps={:?}",
            result.word_steps
        );
        assert!(!(command_id == 19 && result.pair_steps.len() != 1), "command 19 pair steps={:?}", result.pair_steps);
    }
}

/// Go：`TestApplyNativeCommandModifierFailsClosedOnUnsupportedID`
#[test]
fn apply_native_command_modifier_fails_closed_on_unsupported_id() {
    let mut records = vec![0u8; NATIVE_RECORD_SIZE as usize];
    assert!(apply_native_command_modifier(&mut records, &[0], 16, 0).is_err(), "unsupported command modifier accepted");
}
