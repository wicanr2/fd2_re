//! Go 來源：`remake/cmd/fd2/player_battle_speed.go`（56 行）。
//!
//! 骨架：Go 檔的 6 個函式中產生 1 個（階段 0 清單），其餘標 `r8`，見 `PORTING.json`。

use super::*;

/// Zero keeps old settings and F8 behaviour: follow the general game speed.
///
/// Go：`player_battle_speed.go:6`，`func normalizeBattleAnimationSpeed(speed int) int`
pub(crate) fn normalize_battle_animation_speed(speed: i64) -> i64 {
    if !(0..=4).contains(&speed) {
        return 0;
    }
    speed
}

#[cfg(test)]
mod tests {
    use super::*;

    /// Go `TestBattleSpeedSettingsPreserveOtherPreferencesAndOldDefaults`（`player_battle_speed_test.go`）
    /// 的越界值部分；其餘部分經 `setBattleAnimationSpeed`／`chooseGeneralSpeed`（`r8`）。
    #[test]
    fn normalize_battle_animation_speed_rejects_out_of_range() {
        for bad in [-1, 5, 100] {
            assert_eq!(normalize_battle_animation_speed(bad), 0);
        }
        for ok in 0..=4 {
            assert_eq!(normalize_battle_animation_speed(ok), ok);
        }
    }
}
