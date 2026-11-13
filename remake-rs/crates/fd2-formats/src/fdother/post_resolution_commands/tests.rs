//! 移植自 `post_resolution_commands_test.go`。

use crate::fdother::*;

// Go: TestParseNativePostResolutionCommandsUsesThreeByteRecords
#[test]
fn parse_native_post_resolution_commands_uses_three_byte_records() {
    let raw = [0u8, 0x34, 0x12, 1, 0xb5, 0x00, 3, 0x00, 0xa0];
    let commands = parse_native_post_resolution_commands(&raw, 3).unwrap();
    let want = [
        NativePostResolutionCommand { kind: 0, payload: 0x1234 },
        NativePostResolutionCommand { kind: 1, payload: 0xb5 },
        NativePostResolutionCommand { kind: 3, payload: 0xa000 },
    ];
    assert_eq!(commands, want);
}

// Go: TestParseNativePostResolutionCommandsRejectsTruncation
#[test]
fn parse_native_post_resolution_commands_rejects_truncation() {
    assert!(parse_native_post_resolution_commands(&[0u8; 5], 2).is_err(), "short command stream must fail closed");
}
