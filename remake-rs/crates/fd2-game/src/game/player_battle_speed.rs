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

impl Game {
    /// Go：`player_battle_speed.go:13`，`func (g *Game) battleAnimationActive() bool`
    ///
    /// R5 階段 4（r5w）移植：外殼的 `SetTPS` 讀 `effectivePlayerSpeed`。Go 檢查接收者是否為 nil；Rust 的 `self` 不會是 null。
    pub(crate) fn battle_animation_active(&self) -> bool {
        self.atk.is_some()
            || self.native_heal_presentation.is_some()
            || self.native_modifier_presentation.is_some()
            || self.native_ai_command_modifier.is_some()
            || self.native_ai_item_presentation.is_some()
            || self.native_cmd0_presentation.is_some()
            || self.native_cmd1_presentation.is_some()
            || self.native_cmd2_presentation.is_some()
            || self.native_cmd3_presentation.is_some()
            || self.native_cmd5_presentation.is_some()
            || self.native_cmd6_presentation.is_some()
            || self.native_cmd7_presentation.is_some()
            || self.native_cmd8_presentation.is_some()
            || self.native_cmd9_player.is_some()
            || self.native_cmd9_ai_presentation.is_some()
            || self.native_cmd1012.is_some()
            || self.native_cmd24_presentation.is_some()
            || self.native_cmd29_presentation.is_some()
            || self.native_cmd32_presentation.is_some()
            || self.native_cmd33_presentation.is_some()
            || self.native_cmd34_presentation.is_some()
            || self.native_cmd35_presentation.is_some()
            || self.native_ch28_post_present.as_ref().is_some_and(|j| j.kind == "action" || j.kind == "status")
    }

    /// Go：`player_battle_speed.go:25`，`func (g *Game) effectivePlayerSpeed() int`
    pub(crate) fn effective_player_speed(&self) -> i64 {
        let speed = normalize_battle_animation_speed(self.player_battle_speed);
        if speed != 0 && self.battle_animation_active() {
            return speed;
        }
        normalize_player_speed(self.player_speed)
    }

    /// Go：`player_battle_speed.go:35`，`func (g *Game) syncPlayerClockSpeedAt(now time.Time)`
    pub(crate) fn sync_player_clock_speed_at(&mut self, now: GameInstant) {
        let speed = self.effective_player_speed();
        if self.player_clock.speed != speed {
            self.player_clock.sample(now);
            self.player_clock.speed = speed;
        }
    }
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

    fn at(sec: i64) -> GameInstant {
        GameInstant(sec * 1_000_000_000)
    }

    /// Go `player_battle_speed_test.go:9` `TestBattleSpeedClockEntersLeavesWithoutJump`（R5 階段 4 r5w）。
    #[test]
    fn test_battle_speed_clock_enters_leaves_without_jump() {
        let start = 100;
        let mut g = Game { player_battle_speed: 2, ..Default::default() };
        g.set_player_speed_at(3, at(start));
        g.atk = Some(AtkAnim::default());
        g.sync_player_clock_speed_at(at(start + 1));
        assert!(
            g.player_clock.game == at(start + 3) && g.effective_player_speed() == 2,
            "battle entry jumped clock or multiplied speeds"
        );
        g.set_player_speed_at(4, at(start + 2));
        assert!(
            g.player_clock.game == at(start + 5) && g.player_clock.speed == 2,
            "F8 overrode independent battle speed"
        );
        g.atk = None;
        g.sync_player_clock_speed_at(at(start + 3));
        assert!(
            g.player_clock.game == at(start + 7) && g.player_clock.speed == 4,
            "battle exit jumped or failed to restore"
        );
        assert_eq!(g.player_clock.sample(at(start + 4)), at(start + 11));
    }

    /// Go `player_battle_speed_test.go:57` `TestBattleSpeedFollowsAndCoversAllPresentationOwners`（R5 階段 4 r5w）。
    #[test]
    fn test_battle_speed_follows_and_covers_all_presentation_owners() {
        let owners: Vec<fn(&mut Game)> = vec![
            |g| g.atk = Some(Default::default()),
            |g| g.native_heal_presentation = Some(Default::default()),
            |g| g.native_modifier_presentation = Some(Default::default()),
            |g| g.native_ai_command_modifier = Some(Default::default()),
            |g| g.native_ai_item_presentation = Some(Default::default()),
            |g| g.native_cmd0_presentation = Some(Default::default()),
            |g| g.native_cmd1_presentation = Some(Default::default()),
            |g| g.native_cmd2_presentation = Some(Default::default()),
            |g| g.native_cmd3_presentation = Some(Default::default()),
            |g| g.native_cmd5_presentation = Some(Default::default()),
            |g| g.native_cmd6_presentation = Some(Default::default()),
            |g| g.native_cmd7_presentation = Some(Default::default()),
            |g| g.native_cmd8_presentation = Some(Default::default()),
            |g| g.native_cmd9_player = Some(Default::default()),
            |g| g.native_cmd9_ai_presentation = Some(Default::default()),
            |g| g.native_cmd1012 = Some(Default::default()),
            |g| g.native_cmd24_presentation = Some(Default::default()),
            |g| g.native_cmd29_presentation = Some(Default::default()),
            |g| g.native_cmd32_presentation = Some(Default::default()),
            |g| g.native_cmd33_presentation = Some(Default::default()),
            |g| g.native_cmd34_presentation = Some(Default::default()),
            |g| g.native_cmd35_presentation = Some(Default::default()),
            |g| {
                g.native_ch28_post_present =
                    Some(NativeCh28PostPresentJob { kind: "action".into(), ..Default::default() })
            },
            |g| {
                g.native_ch28_post_present =
                    Some(NativeCh28PostPresentJob { kind: "status".into(), ..Default::default() })
            },
        ];
        for general in 1..=4 {
            for independent in 0..=4 {
                for (i, owner) in owners.iter().enumerate() {
                    let mut g = Game { player_speed: general, player_battle_speed: independent, ..Default::default() };
                    assert_eq!(g.effective_player_speed(), general, "map speed changed");
                    owner(&mut g);
                    let want = if independent == 0 { general } else { independent };
                    assert_eq!(g.effective_player_speed(), want, "owner{i} speed mismatch");
                }
            }
        }
    }

    /// Go `player_battle_speed_test.go:94` `TestBattleSpeedLeavesScriptPresentationAtGeneralRate`（R5 階段 4 r5w）。
    #[test]
    fn test_battle_speed_leaves_script_presentation_at_general_rate() {
        let g = Game {
            player_speed: 1,
            player_battle_speed: 4,
            native_ch28_post_present: Some(NativeCh28PostPresentJob { kind: "script".into(), ..Default::default() }),
            ..Default::default()
        };
        assert_eq!(g.effective_player_speed(), 1, "battle preference changed story presentation speed");
    }
}
