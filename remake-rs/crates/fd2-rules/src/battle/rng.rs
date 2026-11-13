//! Go `*math/rand.Rand` 的介面。沒有對應的 Go 檔。
//!
//! Go 版 `battle` 只呼叫 `(*rand.Rand).Intn`（2026-11-14 以 `grep` 確認正式程式）；測試以
//! `rand.New(rand.NewSource(seed))` 建立固定種子。R2 的完成條件要求「Go 的戰鬥決定性測試以相同
//! 種子產生相同結果」，所以實作 `GoRand` 的型別必須逐值重現 Go `math/rand` 的 `rngSource`
//! （607 項加法延遲費氏產生器與 `Intn` 的取樣方式）。`Cargo.lock` 裡的 `rand` crate 演算法不同，
//! 不能直接拿來用。序列實作在本檔，由階段 2 的 A 組移植，範圍與黃金差分見 `PORTING.md`「開工前的決定」。
//!
//! 原版自己的亂數（`0x9014` 旋轉，`fdother.NativeRNGStep`）不經過這裡，已由
//! `fd2_formats::native_rng::native_rng_step` 提供。

/// Go `*rand.Rand` 在 `battle` 用到的方法。
pub trait GoRand {
    /// Go `(*rand.Rand).Intn(n)`：回傳 `[0, n)` 的整數；Go 在 `n <= 0` 時 panic。
    fn intn(&mut self, n: i64) -> i64;
}
