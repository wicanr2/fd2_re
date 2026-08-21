package main

import "time"

// The PC/AT BIOS timer advances at PIT channel 0 frequency / 65536:
// 1,193,182 / 65,536 ~= 18.2065 Hz. The rounded nanosecond period keeps the
// adapter integer-only; time.Time.Sub retains Go's monotonic component.
const nativeBIOSTickPeriod = 54_925_493 * time.Nanosecond

// nativeBIOSClock materializes only the signed low word read by FD2 at
// absolute 0x46c. The native animation code does not persist the daily BIOS
// counter, so a battle-local zero origin preserves all proven delta/wrap
// behavior without fabricating the host's time-of-day tick.
type nativeBIOSClock struct {
	last         time.Time
	remainder    time.Duration
	elapsedTicks uint64
}

func (c *nativeBIOSClock) Reset() {
	*c = nativeBIOSClock{}
}

// Seed installs a captured signed BIOS low word at an explicit host-time
// boundary. Title CONTINUE uses this to preserve main's 0x25D83 sample while
// allowing later 0x11CAC redraws to observe elapsed ticks.
func (c *nativeBIOSClock) Seed(rawTick int, now time.Time) bool {
	if c == nil || now.IsZero() || rawTick < -0x8000 || rawTick > 0x7fff {
		return false
	}
	*c = nativeBIOSClock{
		last:         now,
		elapsedTicks: uint64(uint16(int16(rawTick))),
	}
	return true
}

func (c *nativeBIOSClock) Sample(now time.Time) int {
	if c.last.IsZero() {
		c.last = now
		return int(int16(uint16(c.elapsedTicks)))
	}
	elapsed := now.Sub(c.last)
	if elapsed < 0 {
		return int(int16(uint16(c.elapsedTicks)))
	}
	c.last = now
	c.remainder += elapsed
	if ticks := c.remainder / nativeBIOSTickPeriod; ticks > 0 {
		c.elapsedTicks += uint64(ticks)
		c.remainder -= ticks * nativeBIOSTickPeriod
	}
	return int(int16(uint16(c.elapsedTicks)))
}

// Current returns the last signed low word actually materialized by Sample.
// It does not advance time. Caller-specific indexed adapters use it as the
// producer-backed snapshot for native globals that compare against the most
// recent 0x11CAC draw.
func (c *nativeBIOSClock) Current() (int, bool) {
	if c == nil || c.last.IsZero() {
		return 0, false
	}
	return int(int16(uint16(c.elapsedTicks))), true
}

// advanceNativeMapClock reproduces the one 0x1297d call at the head of one
// 0x11cac redraw, followed by the independent terrain/unit BIOS-word latches
// consumed by that same frame. Callers must invoke it from the compositor
// transaction, never from the generic Update cadence. A legacy or partially
// materialized State is left untouched.
func (g *Game) advanceNativeMapClock(now time.Time) bool {
	if g == nil || g.st == nil ||
		!g.st.HasNativeMapCycleState ||
		!g.st.HasNativeTerrainPhaseState ||
		!g.st.HasNativeMapBinaryTimingState ||
		!g.st.HasNativeMapViewState {
		return false
	}
	rawTick := g.nativeMapClock.Sample(now)
	return g.st.AdvanceNativeMapPresentationCycles(rawTick) &&
		g.st.AdvanceNativeTerrainPhase(rawTick, -1) &&
		g.st.AdvanceNativeTerrainFlip(rawTick) &&
		g.st.AdvanceNativeUnitPixelShift(rawTick)
}
