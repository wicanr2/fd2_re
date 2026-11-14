//! Go 來源：`remake/cmd/fd2/player_game_clock_test.go`，以及虛擬時鐘（Rust 新增）的測試。

use super::*;

fn unix(sec: i64) -> GameInstant {
    GameInstant(sec * GoDuration::SECOND.0)
}

/// Go `setPlayerSpeedAt` 是 `r8` 函式（第一章沒有執行到，骨架沒有產生）。測試照它的三行本體：
/// 先 `sample(now)`，再寫入速度；沒有戰鬥演出時 `effectivePlayerSpeed()` 等於
/// `normalizePlayerSpeed(playerSpeed)`（`player_battle_speed.go:25`）。
fn set_player_speed_at(g: &mut Game, speed: i64, now: GameInstant) {
    g.player_clock.sample(now);
    g.player_speed = normalize_player_speed(speed);
    g.player_clock.speed = normalize_player_speed(g.player_speed);
}

/// Go：`player_game_clock_test.go:8` `TestPlayerGameClockChangesSpeedWithoutJump`
#[test]
fn test_player_game_clock_changes_speed_without_jump() {
    let start = unix(1000);
    let sec = GoDuration::SECOND;
    let mut g = Game::default();
    set_player_speed_at(&mut g, 1, start);
    assert_eq!(g.player_clock.sample(start + sec), start + sec);
    set_player_speed_at(&mut g, 4, start + GoDuration(2 * sec.0));
    assert_eq!(g.player_clock.game, start + GoDuration(2 * sec.0), "speed change jumped time");
    assert_eq!(g.player_clock.sample(start + GoDuration(3 * sec.0)), start + GoDuration(6 * sec.0));
    set_player_speed_at(&mut g, 1, start + GoDuration(3 * sec.0));
    assert_eq!(g.player_clock.sample(start + GoDuration(4 * sec.0)), start + GoDuration(7 * sec.0));
    assert_eq!(g.player_clock.sample(start), start + GoDuration(7 * sec.0), "wall regression rewound gameplay");
    for bad in [-1, 0, 5, 1_000_000] {
        assert_eq!(normalize_player_speed(bad), 1, "invalid speed accepted");
    }
}

#[test]
fn virtual_frame_clock_starts_off_zero_and_spans_one_second_per_sixty_frames() {
    let mut clock = VirtualFrameClock::default();
    assert!(!clock.now.is_zero());
    let first = clock.advance(1);
    assert!(first.0 == VIRTUAL_CLOCK_ORIGIN.0 + 16_666_666 || first.0 == VIRTUAL_CLOCK_ORIGIN.0 + 16_666_667);
    for _ in 1..60 {
        clock.advance(1);
    }
    assert_eq!(clock.frames, 60);
    assert_eq!(clock.now, VIRTUAL_CLOCK_ORIGIN + GoDuration::SECOND);
}

#[test]
fn virtual_frame_clock_keeps_one_sixtieth_game_second_per_frame_at_any_speed() {
    for speed in 1..=4 {
        let mut g = Game::default();
        g.player_clock.speed = speed;
        let mut clock = VirtualFrameClock::default();
        g.advance_virtual_frame(&mut clock);
        let start = g.game_now();
        let frames = VIRTUAL_TICKS_PER_SECOND * speed;
        let mut prev = start;
        for _ in 0..frames {
            g.advance_virtual_frame(&mut clock);
            let now = g.game_now();
            let step = (now - prev).0;
            // 單幀的遊戲時間 = 牆鐘步長 × speed；牆鐘步長是 floor 或 ceil(1e9 / (60×speed))。
            let lo = (GoDuration::SECOND.0 / frames) * speed;
            assert!(step >= lo && step <= lo + speed, "speed {speed} step {step}");
            prev = now;
        }
        // 速度固定時 60×speed 幀的牆鐘恰為 1 秒，遊戲時間恰為 speed 秒。
        assert_eq!(prev - start, GoDuration(speed * GoDuration::SECOND.0), "speed {speed}");
    }
}

#[test]
fn virtual_frame_clock_treats_invalid_speed_as_one() {
    let mut a = VirtualFrameClock::default();
    let mut b = VirtualFrameClock::default();
    for _ in 0..7 {
        assert_eq!(a.advance(0), b.advance(1));
    }
}

#[test]
fn virtual_clock_drives_bios_ticks_deterministically() {
    // 60 個邏輯幀（1 遊戲秒）跨過 floor(1e9 / 54925493) = 18 個 BIOS tick。
    let mut g = Game::default();
    let mut clock = VirtualFrameClock::default();
    let mut bios = NativeBIOSClock::default();
    g.advance_virtual_frame(&mut clock);
    let now = g.game_now();
    assert_eq!(bios.sample(now), 0);
    let mut last = 0;
    for _ in 0..60 {
        g.advance_virtual_frame(&mut clock);
        let now = g.game_now();
        last = bios.sample(now);
    }
    assert_eq!(last, 18);
}
