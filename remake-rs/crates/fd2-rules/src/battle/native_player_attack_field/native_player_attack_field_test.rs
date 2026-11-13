//! 移植自 `remake/internal/battle/native_player_attack_field_test.go`。

use super::*;

/// Go：`TestNativePlayerAttackTargetFieldMarksWeaponDiamond`——0x18f6a：mode=row+0xc 當 flood 預算、
/// radius=row+0xb 排除內圈，落在 0x4dbfc 基底旗標上。射程 1..2 的武器：距離 0 的自己格 0xff，
/// 1..2 有預算值。
///
/// Go 的 `u` 與第二個空單位不在 `State.Units`；Rust 依模組文件第 6 點先 push 再傳索引（這個函式只讀
/// 傳入的單位與地圖旗標）。
#[test]
fn native_player_attack_target_field_marks_weapon_diamond() {
    const W: i64 = 7;
    const H: i64 = 7;
    let mut s = State { w: W, h: H, native_composition_event_bytes: vec![0; (W * H) as usize], ..Default::default() };
    let size = NATIVE_ITEM_EFFECT_ROW_SIZE as usize;
    let mut rows = vec![0u8; 0x40 * size];
    rows[0x34 * size + 0x0b] = 1;
    rows[0x34 * size + 0x0c] = 2;
    s.bind_native_future_item_rows(&rows).unwrap();
    s.units.push(Unit {
        x: 3,
        y: 3,
        inventory: vec![0x34, 0xa4],
        native_inventory_flags: vec![0x40, 0x40],
        ..Default::default()
    });
    let field = s.native_player_attack_target_field(Some(0)).unwrap();
    for y in 0..H {
        for x in 0..W {
            let d = (x - 3).abs() + (y - 3).abs();
            let got = field[(y * W + x) as usize];
            if d == 0 || d > 2 {
                assert_eq!(got, 0xff, "({x},{y}) d={d} got {got:#x} want 0xff");
            } else {
                assert_eq!(got, (2 - d) as u8, "({x},{y}) d={d} got {got:#x} want {:#x}", 2 - d);
            }
        }
    }
    s.units.push(Unit { x: 3, y: 3, ..Default::default() });
    assert!(s.native_player_attack_target_field(Some(1)).is_err(), "沒有已裝備武器仍回傳射程");
}
