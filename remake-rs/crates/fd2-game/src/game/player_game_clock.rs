//! Go 來源：`remake/cmd/fd2/player_game_clock.go`（50 行）。
//!
//! 骨架：Go 檔的 5 個函式中產生 3 個（階段 0 清單），其餘標 `r8`，見 `PORTING.json`。

use super::*;

/// Go：`player_game_clock.go:5`
#[derive(Debug, Clone, Copy, Default, PartialEq, Eq)]
pub(crate) struct PlayerGameClock {
    pub(crate) wall: GameInstant,
    pub(crate) game: GameInstant,
    pub(crate) speed: i64,
}

/// Go：`player_game_clock.go:10`，`func normalizePlayerSpeed(speed int) int`
pub(crate) fn normalize_player_speed(speed: i64) -> i64 {
    todo!("player_game_clock.go:normalizePlayerSpeed")
}

impl PlayerGameClock {
    /// Go：`player_game_clock.go:17`，`func (c *playerGameClock) sample(now time.Time) time.Time`
    pub(crate) fn sample(&mut self, now: GameInstant) -> GameInstant {
        todo!("player_game_clock.go:sample")
    }
}

impl Game {
    /// Go：`player_game_clock.go:31`，`func (g *Game) gameNow() time.Time`
    pub(crate) fn game_now(&mut self) -> GameInstant {
        todo!("player_game_clock.go:gameNow")
    }
}
