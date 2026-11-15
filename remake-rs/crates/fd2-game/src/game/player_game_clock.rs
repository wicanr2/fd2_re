//! Go 來源：`remake/cmd/fd2/player_game_clock.go`（50 行）。
//!
//! 骨架：Go 檔的 5 個函式中產生 3 個（階段 0 清單），其餘標 `r8`，見 `PORTING.json`。
//!
//! # 虛擬時鐘（R4 A 組）
//!
//! Go 的 `gameNow()` 是 `playerClock.sample(time.Now())`：牆鐘差乘遊戲速度。Rust 把牆鐘換成
//! `Game::virtual_now`，由驅動端每個邏輯幀（一次 `update`）推進一次；`player_clock` 照 Go 乘速度。
//! 推進規則與理由見 `PORTING.md`「虛擬時鐘」與 [`VirtualFrameClock`]。

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
    if !(1..=4).contains(&speed) {
        return 1;
    }
    speed
}

impl PlayerGameClock {
    /// Go：`player_game_clock.go:17`，`func (c *playerGameClock) sample(now time.Time) time.Time`
    pub(crate) fn sample(&mut self, now: GameInstant) -> GameInstant {
        if self.wall.is_zero() {
            self.wall = now;
            self.game = now;
            return self.game;
        }
        let delta = now - self.wall;
        if delta.0 < 0 {
            return self.game;
        }
        self.game = self.game + GoDuration(delta.0.wrapping_mul(normalize_player_speed(self.speed)));
        self.wall = now;
        self.game
    }
}

impl Game {
    /// Go：`player_game_clock.go:31`，`func (g *Game) gameNow() time.Time`
    /// Go 讀 `time.Now()`；Rust 讀驅動端推進的 `virtual_now`（見本檔模組文件）。
    pub(crate) fn game_now(&mut self) -> GameInstant {
        let now = self.virtual_now;
        self.player_clock.sample(now)
    }

    /// Go：`player_game_clock.go:35`，`func (g *Game) setPlayerSpeedAt(speed int, now time.Time)`
    ///
    /// R5 階段 4（r5w）移植，只給 `player_battle_speed.go` 的測試用；F8 的 `cyclePlayerSpeed` 仍是 `r8`。
    pub(crate) fn set_player_speed_at(&mut self, speed: i64, now: GameInstant) {
        self.player_clock.sample(now);
        self.player_speed = normalize_player_speed(speed);
        self.player_clock.speed = self.effective_player_speed();
    }
}

/// Rust 新增（沒有 Go 對應）：Ebiten 主迴圈的基準更新率。Go `main.go:13475` 與
/// `modern_hd_presentation.go:83` 以 `ebiten.SetTPS(60 * speed)` 設定每秒 `Update` 次數。
pub(crate) const VIRTUAL_TICKS_PER_SECOND: i64 = 60;

/// Rust 新增：虛擬時鐘的起點。Go 的 `time.Now()` 永遠不是零值，而 `playerGameClock.sample`、
/// 演出工作的 `started.IsZero()` 都把零值當成「尚未開始」，所以第一個邏輯幀不能落在 0。
/// 取 1 秒只是為了避開零值，不代表任何主機時間。
pub(crate) const VIRTUAL_CLOCK_ORIGIN: GameInstant = GameInstant(1_000_000_000);

/// Rust 新增：驅動端持有的虛擬牆鐘，取代 Go 主迴圈的牆鐘與 `SetTPS`。
///
/// Go 的每次 `Update` 間隔是 `1s / (60 × speed)` 的牆鐘（`SetTPS(60 * effectivePlayerSpeed())`），
/// `gameNow()` 再把牆鐘差乘 `speed`；所以穩態下每次 `Update` 的遊戲時間恰好前進 1/60 秒，與速度無關，
/// 速度只改變每秒跑幾次 `Update`。Rust 照這個關係：每個邏輯幀牆鐘前進 `1e9 / (60 × speed)` 奈秒，
/// 速度乘法仍由 [`PlayerGameClock::sample`] 照 Go 做。除不盡的部分以 `carry`（單位 1/(60×speed)
/// 奈秒）累積，速度固定時每 `60 × speed` 幀牆鐘剛好前進 1 秒、遊戲時間前進 `speed` 秒。
#[derive(Debug, Clone, Copy, PartialEq, Eq)]
pub(crate) struct VirtualFrameClock {
    /// 目前的虛擬牆鐘（交給 `Game::virtual_now`）。
    pub(crate) now: GameInstant,
    /// 已推進的邏輯幀數。
    pub(crate) frames: u64,
    carry: i64,
}

impl Default for VirtualFrameClock {
    fn default() -> Self {
        Self { now: VIRTUAL_CLOCK_ORIGIN, frames: 0, carry: 0 }
    }
}

impl VirtualFrameClock {
    /// 推進一個邏輯幀。`speed` 是 Go `SetTPS` 用的速度（`effectivePlayerSpeed()`，驅動端在沒有
    /// 該函式前讀 `normalize_player_speed(game.player_clock.speed)`）；越界值照 Go 視為 1。
    /// 速度改變時，未滿 1 奈秒的累積餘數沿用新分母，誤差小於 1 奈秒。
    pub(crate) fn advance(&mut self, speed: i64) -> GameInstant {
        let denominator = VIRTUAL_TICKS_PER_SECOND * normalize_player_speed(speed);
        let numerator = GoDuration::SECOND.0 + self.carry;
        let step = numerator / denominator;
        self.carry = numerator % denominator;
        self.now = self.now + GoDuration(step);
        self.frames += 1;
        self.now
    }
}

impl Game {
    /// Rust 新增：驅動端在每次 `update` 前呼叫，把 `virtual_now` 推進一個邏輯幀。速度取
    /// `player_clock.speed`（Go `syncPlayerClockSpeedAt` 寫入、`SetTPS` 讀的同一個值）。
    pub(crate) fn advance_virtual_frame(&mut self, clock: &mut VirtualFrameClock) -> GameInstant {
        self.virtual_now = clock.advance(self.player_clock.speed);
        self.virtual_now
    }
}

#[cfg(test)]
#[path = "player_game_clock/player_game_clock_test.rs"]
mod player_game_clock_test;
