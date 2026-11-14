//! Go `*math/rand.Rand` 的介面與相容序列。沒有對應的 `battle` Go 檔。
//!
//! Go 版 `battle` 只呼叫 `(*rand.Rand).Intn`（2026-11-14 以 `grep` 確認正式程式）；測試以
//! `rand.New(rand.NewSource(seed))` 建立固定種子，少數測試另比對 `Int63()`。R2 的完成條件要求
//! 「Go 的戰鬥決定性測試以相同種子產生相同結果」，所以 [`GoMathRand`] 逐值重現 Go 1.22.12
//! `math/rand` 的 `rngSource`（607 項加法延遲費氏產生器）與 `Intn`／`Int31n`／`Int63n`／`Int31`
//! 的取樣方式。`Cargo.lock` 裡的 `rand` crate 演算法不同，不能直接拿來用。
//!
//! 這是 `rust-port-no-go-stdlib` 教訓的具名例外之一（另兩項是 `fd2_formats::go_math` 與
//! `fd2_campaign` 的 `go_filepath`），範圍見 `PORTING.md`「開工前的決定」第 1 點。
//! `rngCooked` 表由 `tools/rust_gen_gorand_cooked.sh` 從容器內 Go 的 `$GOROOT/src/math/rand/rng.go`
//! 機械轉出到 `rng/cooked.rs`，該檔頭保留 Go 的 BSD 授權聲明。本檔的演算法對照同一份
//! `rng.go`（`seedrand`、`rngSource.Seed`、`Uint64`）與 `rand.go`（`Int63n`、`Int31n`、`Intn`），
//! 黃金差分是 `remake/cmd/fd2-golden-gorand` 與 `tests/golden_gorand.rs`。
//!
//! 原版自己的亂數（`0x9014` 旋轉，`fdother.NativeRNGStep`）不經過這裡，已由
//! `fd2_formats::native_rng::native_rng_step` 提供。

// Go math/rand 的演算法轉寫。原始碼授權：
// Copyright 2009 The Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.
// （LICENSE 全文見 rng/cooked.rs 檔頭。）

mod cooked;

use cooked::RNG_COOKED;

/// Go `*rand.Rand` 在 `battle` 用到的方法。
pub trait GoRand {
    /// Go `(*rand.Rand).Intn(n)`：回傳 `[0, n)` 的整數；Go 在 `n <= 0` 時 panic。
    fn intn(&mut self, n: i64) -> i64;
}

const RNG_LEN: usize = 607;
const RNG_TAP: usize = 273;
const RNG_MASK: u64 = (1 << 63) - 1;
const INT32_MAX: i64 = (1 << 31) - 1;

/// Go `seedrand`：`x[n+1] = 48271 * x[n] mod (2**31 - 1)`。
fn seedrand(x: i32) -> i32 {
    const A: i32 = 48271;
    const Q: i32 = 44488;
    const R: i32 = 3399;
    let hi = x / Q;
    let lo = x % Q;
    let mut x = A.wrapping_mul(lo).wrapping_sub(R.wrapping_mul(hi));
    if x < 0 {
        x = x.wrapping_add(INT32_MAX as i32);
    }
    x
}

/// Go `rngSource`（`rand.NewSource` 回傳的具體型別）。
#[derive(Clone)]
pub struct GoSource {
    tap: usize,
    feed: usize,
    vec: [i64; RNG_LEN],
}

impl GoSource {
    /// Go `rand.NewSource(seed)`。
    pub fn new(seed: i64) -> Self {
        let mut s = Self { tap: 0, feed: 0, vec: [0; RNG_LEN] };
        s.seed(seed);
        s
    }

    /// Go `(*rngSource).Seed`。
    pub fn seed(&mut self, seed: i64) {
        self.tap = 0;
        self.feed = RNG_LEN - RNG_TAP;
        let mut seed = seed % INT32_MAX;
        if seed < 0 {
            seed += INT32_MAX;
        }
        if seed == 0 {
            seed = 89482311;
        }
        let mut x = seed as i32;
        for i in -20..RNG_LEN as i64 {
            x = seedrand(x);
            if i >= 0 {
                let mut u = (x as i64) << 40;
                x = seedrand(x);
                u ^= (x as i64) << 20;
                x = seedrand(x);
                u ^= x as i64;
                u ^= RNG_COOKED[i as usize];
                self.vec[i as usize] = u;
            }
        }
    }

    /// Go `(*rngSource).Uint64`。
    pub fn uint64(&mut self) -> u64 {
        self.tap = if self.tap == 0 { RNG_LEN - 1 } else { self.tap - 1 };
        self.feed = if self.feed == 0 { RNG_LEN - 1 } else { self.feed - 1 };
        let x = self.vec[self.feed].wrapping_add(self.vec[self.tap]);
        self.vec[self.feed] = x;
        x as u64
    }

    /// Go `(*rngSource).Int63`。
    pub fn int63(&mut self) -> i64 {
        (self.uint64() & RNG_MASK) as i64
    }
}

impl std::fmt::Debug for GoSource {
    fn fmt(&self, f: &mut std::fmt::Formatter<'_>) -> std::fmt::Result {
        f.debug_struct("GoSource").field("tap", &self.tap).field("feed", &self.feed).finish_non_exhaustive()
    }
}

/// Go `rand.New(rand.NewSource(seed))`：以 [`GoSource`] 為來源的 `*rand.Rand`。
///
/// 只移植 `battle` 與其測試用到的取樣方法；Go `Rand` 的 `readVal`／`readPos`（`Read` 專用）不移植。
#[derive(Clone, Debug)]
pub struct GoMathRand {
    src: GoSource,
}

impl GoMathRand {
    /// Go `rand.New(rand.NewSource(seed))`。
    pub fn new(seed: i64) -> Self {
        Self { src: GoSource::new(seed) }
    }

    /// Go `(*Rand).Int63`。
    pub fn int63(&mut self) -> i64 {
        self.src.int63()
    }

    /// Go `(*Rand).Int31`。
    pub fn int31(&mut self) -> i32 {
        (self.int63() >> 32) as i32
    }

    /// Go `(*Rand).Int63n`；Go 在 `n <= 0` 時 panic，訊息相同。
    pub fn int63n(&mut self, n: i64) -> i64 {
        assert!(n > 0, "invalid argument to Int63n");
        if n & (n - 1) == 0 {
            return self.int63() & (n - 1);
        }
        let max = ((1u64 << 63) - 1 - (1u64 << 63) % (n as u64)) as i64;
        let mut v = self.int63();
        while v > max {
            v = self.int63();
        }
        v % n
    }

    /// Go `(*Rand).Int31n`；Go 在 `n <= 0` 時 panic，訊息相同。
    pub fn int31n(&mut self, n: i32) -> i32 {
        assert!(n > 0, "invalid argument to Int31n");
        if n & (n - 1) == 0 {
            return self.int31() & (n - 1);
        }
        let max = ((1u32 << 31) - 1 - (1u32 << 31) % (n as u32)) as i32;
        let mut v = self.int31();
        while v > max {
            v = self.int31();
        }
        v % n
    }
}

impl GoRand for GoMathRand {
    /// Go `(*Rand).Intn`：`n` 不超過 `1<<31-1` 時走 `Int31n`，否則走 `Int63n`。
    fn intn(&mut self, n: i64) -> i64 {
        assert!(n > 0, "invalid argument to Intn");
        if n <= INT32_MAX {
            return self.int31n(n as i32) as i64;
        }
        self.int63n(n)
    }
}

#[cfg(test)]
mod tests {
    use super::*;

    /// Go 1.22.12 `rand.New(rand.NewSource(1))` 的前幾個值（`fd2-golden-gorand` 黃金檔的子集，
    /// 這裡只做沒有黃金檔時的煙霧測試）。
    #[test]
    fn seed_one_matches_go() {
        let mut r = GoMathRand::new(1);
        assert_eq!(r.int63(), 5577006791947779410);
        assert_eq!(r.int63(), 8674665223082153551);
        let mut r = GoMathRand::new(1);
        assert_eq!(r.intn(100), 81);
        assert_eq!(r.intn(100), 87);
        assert_eq!(r.intn(100), 47);
    }

    #[test]
    #[should_panic(expected = "invalid argument to Intn")]
    fn intn_rejects_zero() {
        GoMathRand::new(1).intn(0);
    }
}
