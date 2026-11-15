//! 每個顯示幀要跑幾次 `Update`（R5 階段 4，r5w）。
//!
//! Adapted from github.com/hajimehoshi/ebiten/v2 v2.6.6 `internal/clock/clock.go`（`calcCountFromTPS`）與
//! `internal/ui/context.go`（`updateFrameImpl` 的「第一次 Draw 之前至少 Update 一次」）。Copyright 2017 The Ebiten
//! Authors，Apache License 2.0（同 Go 版 `remake/cmd/fd2/ebiten_screen_filter.go` 的處理）。改動：時間由呼叫端傳入
//! （單調時鐘的奈秒），狀態放在結構裡而不是套件變數。

const SECOND: i64 = 1_000_000_000;

/// Ebiten 的 TPS 時鐘。
#[derive(Debug, Clone)]
pub struct EbitenClock {
    last_system_time: i64,
    prev_tps: i64,
    update_called: bool,
}

impl EbitenClock {
    /// `now` 是起點（Ebiten `clock.init` 的 `lastSystemTime = now()`）。
    pub fn new(now: i64) -> EbitenClock {
        EbitenClock { last_system_time: now, prev_tps: 0, update_called: false }
    }

    /// 本顯示幀的 `Update` 次數（`clock.UpdateFrame` 加上 `updateFrameImpl` 第一幀至少一次）。
    pub fn update_count(&mut self, tps: i64, now: i64) -> u32 {
        let mut count = self.calc_count_from_tps(tps, now);
        if !self.update_called && count == 0 {
            count = 1;
            self.update_called = true;
        }
        count
    }

    fn calc_count_from_tps(&mut self, tps: i64, now: i64) -> u32 {
        if tps <= 0 {
            return 0;
        }
        let diff = now - self.last_system_time;
        if diff < 0 {
            return 0;
        }
        let mut count: i64 = 0;
        let mut sync_with_system_clock = false;
        // Use either 5 ticks or 5/60 sec in the case when TPS is too big like 300 (#1444).
        if diff > (SECOND * 5 / tps).max(SECOND * 5 / 60) || self.prev_tps != tps {
            sync_with_system_clock = true;
        } else {
            count = diff * tps / SECOND;
        }
        self.prev_tps = tps;
        if count == 0 && SECOND / tps / 2 < diff {
            count = 1;
        }
        if count == 2 && SECOND / tps * 3 / 2 > diff {
            count = 1;
        }
        if sync_with_system_clock {
            self.last_system_time = now;
        } else {
            self.last_system_time += count * SECOND / tps;
        }
        count as u32
    }
}

#[cfg(test)]
mod tests {
    use super::*;

    #[test]
    fn steady_sixty_fps_runs_one_update_per_frame() {
        let mut c = EbitenClock::new(0);
        // 第一幀：TPS 剛設定（prevTPS 不同）→ 與系統時鐘同步，count 0；第一次 Draw 前補一次。
        assert_eq!(c.update_count(60, 16_666_667), 1);
        let mut total = 0;
        for i in 2..=61 {
            total += c.update_count(60, i * 16_666_667);
        }
        assert!((59..=61).contains(&total), "{total}");
    }

    #[test]
    fn slow_frames_catch_up_and_long_stalls_resync() {
        let mut c = EbitenClock::new(0);
        c.update_count(60, 1);
        assert_eq!(c.update_count(60, 1 + 50_000_000), 3, "50ms ≈ 3 個 tick");
        assert_eq!(
            c.update_count(60, 1 + 50_000_000 + SECOND),
            1,
            "超過 5/60 秒：與系統時鐘同步，不補跑（穩定化後 1 次）"
        );
        assert_eq!(c.update_count(120, 2 * SECOND), 1, "TPS 改變：同步（穩定化後 1 次）");
    }
}
