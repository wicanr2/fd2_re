//! Go `math.Sin`／`math.Cos` 的逐行移植（`rust-port-no-go-stdlib` 的具名例外）。
//!
//! Rust 的 `f64::sin`／`f64::cos` 呼叫各平台的數學函式庫（glibc、Apple libm、Windows CRT、wasm 的
//! libm 移植），IEEE 754 不要求它們精確捨入，末位可能隨平台不同。Go 的 `math.Sin`／`math.Cos` 是純 Go
//! 實作，每個平台結果相同。`figani` 指令 6（`0x26E39`）把三角函數的結果截成整數座標，末位差異落在
//! 整數邊界上就會改變座標，所以這裡照 Go 1.22.12 `src/math/sin.go` 逐行重現，讓桌面、網頁與手機
//! 得到同一組座標。
//!
//! 只移植 `|x| < 2^29`（Go 的 `reduceThreshold`）的化簡路徑。超出範圍時 Go 改用 `trig_reduce.go` 的
//! Payne-Hanek 化簡，本模組不移植，呼叫端傳入這種值會直接中止。
//!
//! 黃金差分：`remake/cmd/fd2-golden-figani`（`cases_native.go`）輸出指令 6 的座標，Rust 端逐值比對。

// 轉寫自 Go 1.22.12 src/math/sin.go。原始碼授權：
// Copyright 2011 The Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.
//
// Go 原始碼註明：常數與演算法來自 Cephes Math Library Release 2.8（June, 2000），
// Copyright 1984, 1987, 1989, 1992, 2000 by Stephen L. Moshier，
// http://netlib.sandia.gov/cephes/cmath/sin.c。

/// Go `reduceThreshold`：`1 << 29`。
const REDUCE_THRESHOLD: f64 = (1u64 << 29) as f64;

// 常數照抄 Go 原始碼的十進位字面值（同樣四捨五入到最近的 f64）。
#[allow(clippy::excessive_precision)]
const PI4A: f64 = 7.85398125648498535156e-1;
#[allow(clippy::excessive_precision)]
const PI4B: f64 = 3.77489470793079817668e-8;
#[allow(clippy::excessive_precision)]
const PI4C: f64 = 2.69515142907905952645e-15;
#[allow(clippy::excessive_precision)]
const SIN: [f64; 6] = [
    1.58962301576546568060e-10,
    -2.50507477628578072866e-8,
    2.75573136213857245213e-6,
    -1.98412698295895385996e-4,
    8.33333333332211858878e-3,
    -1.66666666666666307295e-1,
];
#[allow(clippy::excessive_precision)]
const COS: [f64; 6] = [
    -1.13585365213876817300e-11,
    2.08757008419747316778e-9,
    -2.75573141792967388112e-7,
    2.48015872888517045348e-5,
    -1.38888888888730564116e-3,
    4.16666666666665929218e-2,
];

/// Go `sin.go` 在 `x < reduceThreshold` 時的化簡：回傳八分圓編號 `j`（0..7）與化簡後的角度 `z`。
fn reduce_pi4(x: f64) -> (u64, f64) {
    assert!(x < REDUCE_THRESHOLD, "go_math: |x| >= 2^29 需要 Payne-Hanek 化簡，本移植不支援");
    // Go 的 `4 / Pi` 是精確常數再捨入；2/π 的 f64 常數乘 2 是精確運算，得到同一個值。
    let mut j = (x * (2.0 * std::f64::consts::FRAC_2_PI)) as u64;
    let mut y = j as f64;
    if j & 1 == 1 {
        j += 1;
        y += 1.0;
    }
    j &= 7;
    (j, ((x - y * PI4A) - y * PI4B) - y * PI4C)
}

fn sin_poly(z: f64, zz: f64) -> f64 {
    z + z * zz * ((((((SIN[0] * zz) + SIN[1]) * zz + SIN[2]) * zz + SIN[3]) * zz + SIN[4]) * zz + SIN[5])
}

fn cos_poly(zz: f64) -> f64 {
    1.0 - 0.5 * zz + zz * zz * ((((((COS[0] * zz) + COS[1]) * zz + COS[2]) * zz + COS[3]) * zz + COS[4]) * zz + COS[5])
}

/// Go `math.Cos(x)`。`|x| >= 2^29` 時中止（見模組文件）。
pub fn cos(x: f64) -> f64 {
    if x.is_nan() || x.is_infinite() {
        return f64::NAN;
    }
    let mut sign = false;
    let x = x.abs();
    let (mut j, z) = reduce_pi4(x);
    if j > 3 {
        j -= 4;
        sign = !sign;
    }
    if j > 1 {
        sign = !sign;
    }
    let zz = z * z;
    let y = if j == 1 || j == 2 { sin_poly(z, zz) } else { cos_poly(zz) };
    if sign { -y } else { y }
}

/// Go `math.Sin(x)`。`|x| >= 2^29` 時中止（見模組文件）。
pub fn sin(x: f64) -> f64 {
    if x == 0.0 || x.is_nan() {
        return x;
    }
    if x.is_infinite() {
        return f64::NAN;
    }
    let mut sign = false;
    let mut x = x;
    if x < 0.0 {
        x = -x;
        sign = true;
    }
    let (mut j, z) = reduce_pi4(x);
    if j > 3 {
        sign = !sign;
        j -= 4;
    }
    let zz = z * z;
    let y = if j == 1 || j == 2 { cos_poly(zz) } else { sin_poly(z, zz) };
    if sign { -y } else { y }
}

#[cfg(test)]
mod tests {
    use super::*;

    /// 特殊值照 Go `math.Sin`／`math.Cos` 的文件。
    #[test]
    fn special_values_follow_go() {
        assert_eq!(sin(0.0).to_bits(), 0.0f64.to_bits());
        assert_eq!(sin(-0.0).to_bits(), (-0.0f64).to_bits());
        assert!(sin(f64::NAN).is_nan() && sin(f64::INFINITY).is_nan() && sin(f64::NEG_INFINITY).is_nan());
        assert!(cos(f64::NAN).is_nan() && cos(f64::INFINITY).is_nan());
        assert_eq!(cos(0.0), 1.0);
    }

    /// 和平台 libm 的差距不超過 1 ulp（這裡只檢查移植沒有寫錯，逐位元的對照由 Go 黃金差分負責）。
    #[test]
    fn close_to_platform_libm() {
        for i in -2000..=2000 {
            let x = i as f64 * 0.01;
            for (go, std) in [(sin(x), x.sin()), (cos(x), x.cos())] {
                let ulps = (go.to_bits() as i64 - std.to_bits() as i64).abs();
                assert!(ulps <= 1 || (go - std).abs() < 1e-300, "x={x} go={go} std={std}");
            }
        }
    }

    #[test]
    #[should_panic(expected = "Payne-Hanek")]
    fn large_argument_is_rejected() {
        sin(REDUCE_THRESHOLD);
    }
}
