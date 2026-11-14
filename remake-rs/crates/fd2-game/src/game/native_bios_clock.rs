//! Go 來源：`remake/cmd/fd2/native_bios_clock.go`（95 行）。
//!
//! 骨架：Go 檔的 6 個函式中產生 4 個（階段 0 清單），其餘標 `r8`，見 `PORTING.json`。

use super::*;

/// The PC/AT BIOS timer advances at PIT channel 0 frequency / 65536:
/// 1,193,182 / 65,536 ~= 18.2065 Hz. The rounded nanosecond period keeps the
/// adapter integer-only; time.Time.Sub retains Go's monotonic component.
///
/// Go 原式：`54_925_493 * time.Nanosecond`
pub(crate) const NATIVE_BIOS_TICK_PERIOD: GoDuration = GoDuration(54925493);

/// nativeBIOSClock materializes only the signed low word read by FD2 at
/// absolute 0x46c. The native animation code does not persist the daily BIOS
/// counter, so a battle-local zero origin preserves all proven delta/wrap
/// behavior without fabricating the host's time-of-day tick.
///
/// Go：`native_bios_clock.go:14`
#[derive(Debug, Clone, Copy, Default, PartialEq, Eq)]
pub(crate) struct NativeBIOSClock {
    pub(crate) last: GameInstant,
    pub(crate) remainder: GoDuration,
    pub(crate) elapsed_ticks: u64,
}

impl NativeBIOSClock {
    /// Go：`native_bios_clock.go:20`，`func (c *nativeBIOSClock) Reset()`
    pub(crate) fn reset(&mut self) {
        todo!("native_bios_clock.go:Reset")
    }

    /// Go：`native_bios_clock.go:48`，`func (c *nativeBIOSClock) Sample(now time.Time) int`
    pub(crate) fn sample(&mut self, now: GameInstant) -> i64 {
        todo!("native_bios_clock.go:Sample")
    }

    /// Current returns the last signed low word actually materialized by Sample.
    /// It does not advance time. Caller-specific indexed adapters use it as the
    /// producer-backed snapshot for native globals that compare against the most
    /// recent 0x11CAC draw.
    ///
    /// Go：`native_bios_clock.go:70`，`func (c *nativeBIOSClock) Current() (int, bool)`
    /// Go 檢查接收者是否為 nil；Rust 的 `self` 不會是 null，該分支不移植。
    pub(crate) fn current(&self) -> (i64, bool) {
        todo!("native_bios_clock.go:Current")
    }
}

impl Game {
    /// advanceNativeMapClock reproduces the one 0x1297d call at the head of one
    /// 0x11cac redraw, followed by the independent terrain/unit BIOS-word latches
    /// consumed by that same frame. Callers must invoke it from the compositor
    /// transaction, never from the generic Update cadence. A legacy or partially
    /// materialized State is left untouched.
    ///
    /// Go：`native_bios_clock.go:82`，`func (g *Game) advanceNativeMapClock(now time.Time) bool`
    /// Go 檢查接收者是否為 nil；Rust 的 `self` 不會是 null，該分支不移植。
    pub(crate) fn advance_native_map_clock(&mut self, now: GameInstant) -> bool {
        todo!("native_bios_clock.go:advanceNativeMapClock")
    }
}
