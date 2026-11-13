//! 移植自 `remake/internal/battle/native_death_presentation_flag_test.go`。

use super::*;

/// Go：`TestNativeDeathPresentationFlagPreservesWriterAndStaysOutOfSaves`
#[test]
fn native_death_presentation_flag_preserves_writer_and_stays_out_of_saves() {
    let mut unit =
        Unit { hp: 10, max_hp: 10, native_record_byte5: 0x80, has_native_record_byte5: true, ..Default::default() };
    unit.apply_hp_damage(3);
    assert!(!unit.native_death_presentation_flag().1, "nonlethal damage created a corpse");
    unit.apply_hp_damage(7);
    unit.apply_hp_damage(9);
    let (flag, valid) = unit.native_death_presentation_flag();
    assert!(
        valid && flag == 0x80 && unit.native_record_byte5 == 1,
        "death writer flag was guessed from the already inactive record"
    );
    let with = serde_json::to_vec(&unit).unwrap();
    let loaded: Unit = serde_json::from_slice(&with).unwrap();
    assert!(!loaded.native_death_presentation_flag().1, "presentation transition was serialized");
    unit.clear_native_death_presentation_flag();
    let without = serde_json::to_vec(&unit).unwrap();
    assert!(
        with == without && unit.hp == 0 && unit.native_record_byte5 == 1,
        "clearing presentation metadata changed the save or death writer"
    );
    unit.restore_native_hp();
    let hp = unit.hp;
    unit.apply_hp_damage(hp);
    unit.restore_native_hp();
    unit.hp = 0; // a later unrelated HP write must not expose the previous corpse flag
    assert!(!unit.native_death_presentation_flag().1, "revival retained a previous death");
}
