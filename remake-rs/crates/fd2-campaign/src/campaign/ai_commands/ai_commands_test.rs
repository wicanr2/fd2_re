//! Go `remake/internal/campaign/ai_commands_test.go` 的逐條移植。

use super::*;

fn write_temp(tag: &str, text: &str) -> String {
    let path = crate::test_support::temp_dir(tag).join("item.json");
    std::fs::write(&path, text).expect("寫入暫存檔");
    path.to_string_lossy().into_owned()
}

/// Go：`TestLoadAICommandSpellMapUsesK4RawOffset`
#[test]
fn load_ai_command_spell_map_uses_k4_raw_offset() {
    let row = r#"[{"id":79,"raw":"000cc201500000000000000000001803001f0004000000"},{"id":1,"raw":"0000000000000000000000000000000000000000000000"}]"#;
    let path = write_temp("ai-k4", row);
    let got = load_ai_command_spell_map(&path).unwrap();
    assert_eq!(got.get(&79), Some(&15), "item 79 command map={got:?}, want spell 15");
    assert!(!got.contains_key(&1), "physical/zero command unexpectedly mapped: {got:?}");
}

/// Go：`TestLoadAICommandSpellMapRejectsMalformedRaw`
#[test]
fn load_ai_command_spell_map_rejects_malformed_raw() {
    let path = write_temp("ai-short", r#"[{"id":1,"raw":"00"}]"#);
    assert!(load_ai_command_spell_map(&path).is_err(), "short raw record accepted");
}
