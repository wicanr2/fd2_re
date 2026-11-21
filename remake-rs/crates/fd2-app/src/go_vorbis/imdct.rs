//! jfreymuth/vorbis v1.0.2 的 `imdct.go`、`window.go`。授權見 [`super`] 模組文件。
//!
//! `f32` 運算照 Go 原式的運算順序逐項寫出（Go 在 amd64 `GOAMD64=v1` 不合併乘加，Rust 也不會），查表用
//! `fd2_formats::go_math` 的 Go `math.Sin`／`math.Cos`。

use fd2_formats::go_math;

use super::codebook::ilog;

pub(super) struct ImdctLookup {
    a: Vec<f32>,
    b: Vec<f32>,
    c: Vec<f32>,
}

pub(super) fn generate_imdct_lookup(n: usize) -> ImdctLookup {
    let mut l = ImdctLookup { a: vec![0.0; n / 2], b: vec![0.0; n / 2], c: vec![0.0; n / 4] };
    let fn_ = n as f64;
    let pi = std::f64::consts::PI;
    for k in 0..n / 4 {
        let fk = k as f64;
        l.a[2 * k] = go_math::cos(4.0 * fk * pi / fn_) as f32;
        l.a[2 * k + 1] = (-go_math::sin(4.0 * fk * pi / fn_)) as f32;
        l.b[2 * k] = go_math::cos((2.0 * fk + 1.0) * pi / fn_ / 2.0) as f32;
        l.b[2 * k + 1] = go_math::sin((2.0 * fk + 1.0) * pi / fn_ / 2.0) as f32;
    }
    for k in 0..n / 8 {
        let fk = k as f64;
        l.c[2 * k] = go_math::cos(2.0 * (2.0 * fk + 1.0) * pi / fn_) as f32;
        l.c[2 * k + 1] = (-go_math::sin(2.0 * (2.0 * fk + 1.0) * pi / fn_)) as f32;
    }
    l
}

/// Go `imdct(t, in, out)`：`in` 長 `n/2`（同時當暫存區），`out` 長 `n`。
#[allow(clippy::many_single_char_names)]
pub(super) fn imdct(t: &ImdctLookup, input: &mut [f32], out: &mut [f32]) {
    let n = input.len() * 2;
    let (n2, n4, n8) = (n / 2, n / 4, n / 8);
    let n3_4 = n - n4;
    let a = &t.a;

    for j in 0..n8 {
        let a0 = a[n2 - 2 * j - 1];
        let a1 = a[n2 - 2 * j - 2];
        let a2 = a[n4 - 2 * j - 1];
        let a3 = a[n4 - 2 * j - 2];
        let a4 = a[n2 - 4 - 4 * j];
        let a5 = a[n2 - 3 - 4 * j];
        let v0 = (-input[4 * j + 3]) * a0 + (-input[4 * j + 1]) * a1;
        let v1 = (-input[4 * j + 3]) * a1 - (-input[4 * j + 1]) * a0;
        let v2 = (input[n2 - 4 * j - 4]) * a2 + (input[n2 - 4 * j - 2]) * a3;
        let v3 = (input[n2 - 4 * j - 4]) * a3 - (input[n2 - 4 * j - 2]) * a2;
        out[n4 + 2 * j + 1] = v3 + v1;
        out[n4 + 2 * j] = v2 + v0;
        out[2 * j + 1] = (v3 - v1) * a4 - (v2 - v0) * a5;
        out[2 * j] = (v2 - v0) * a4 + (v3 - v1) * a5;
    }
    let ld = ilog(n as i64) as usize - 1;
    for l in 0..ld.saturating_sub(3) {
        let k0 = n >> (l + 3);
        let k1 = 1usize << (l + 3);
        let rlim = n >> (l + 4);
        let s2lim = 1usize << (l + 2);
        for r in 0..rlim {
            let a0 = a[r * k1];
            let a1 = a[r * k1 + 1];
            let mut i0 = (n2 - 1 - 2 * r) as isize;
            let mut i1 = (n2 - 2 - 2 * r) as isize;
            let mut i2 = n2 as isize - 1 - k0 as isize - 2 * r as isize;
            let mut i3 = n2 as isize - 2 - k0 as isize - 2 * r as isize;
            let mut s2 = 0;
            while s2 < s2lim {
                let (v0, v1) = (out[i0 as usize], out[i1 as usize]);
                let (v2, v3) = (out[i2 as usize], out[i3 as usize]);
                out[i0 as usize] = v0 + v2;
                out[i1 as usize] = v1 + v3;
                out[i2 as usize] = (v0 - v2) * a0 - (v1 - v3) * a1;
                out[i3 as usize] = (v1 - v3) * a0 + (v0 - v2) * a1;
                let step = 2 * k0 as isize;
                i0 -= step;
                i1 -= step;
                i2 -= step;
                i3 -= step;
                s2 += 2;
            }
        }
    }
    for i in 0..n8 {
        let j = ((i as u32).reverse_bits() >> (32 - ld as u32 + 3)) as usize;
        if i < j {
            out.swap(4 * j, 4 * i);
            out.swap(4 * j + 1, 4 * i + 1);
            out.swap(4 * j + 2, 4 * i + 2);
            out.swap(4 * j + 3, 4 * i + 3);
        }
    }
    for k in 0..n8 {
        input[n2 - 1 - 2 * k] = out[4 * k];
        input[n2 - 2 - 2 * k] = out[4 * k + 1];
        input[n4 - 1 - 2 * k] = out[4 * k + 2];
        input[n4 - 2 - 2 * k] = out[4 * k + 3];
    }
    let (mut i0, mut i1) = (0usize, 1usize);
    let (mut i2, mut i3) = (n2 as isize - 2, n2 as isize - 1);
    for _ in 0..n8 {
        let (v0, v1) = (input[i0], input[i1]);
        let (v2, v3) = (input[i2 as usize], input[i3 as usize]);
        let c0 = t.c[i0];
        let c1 = t.c[i1];
        out[i0] = (v0 + v2 + c1 * (v0 - v2) + c0 * (v1 + v3)) / 2.0;
        out[i2 as usize] = (v0 + v2 - c1 * (v0 - v2) - c0 * (v1 + v3)) / 2.0;
        out[i1] = (v1 - v3 + c1 * (v1 + v3) - c0 * (v0 - v2)) / 2.0;
        out[i3 as usize] = (-v1 + v3 + c1 * (v1 + v3) - c0 * (v0 - v2)) / 2.0;
        i0 += 2;
        i1 += 2;
        i2 -= 2;
        i3 -= 2;
    }
    for k in 0..n4 {
        let b0 = t.b[2 * k];
        let b1 = t.b[2 * k + 1];
        let v0 = out[2 * k];
        let v1 = out[2 * k + 1];
        input[k] = v0 * b0 + v1 * b1;
        input[n2 - 1 - k] = v0 * b1 - v1 * b0;
    }
    for i in 0..n4 {
        out[i] = input[i + n4];
        out[n - i - 1] = -input[n - i - n3_4 - 1];
    }
    for i in n4..n3_4 {
        out[i] = -input[n3_4 - i - 1];
    }
}

/// Go `windowType`。
pub(super) struct WindowType {
    pub(super) size: usize,
    pub(super) prev: usize,
    pub(super) next: usize,
}

/// Go `(*Decoder).applyWindow`：`samples` 的每個聲道都已切成 `t.size` 長。
pub(super) fn apply_window(t: &WindowType, windows: &[Vec<f32>; 2], long_blocksize: usize, samples: &mut [Vec<f32>]) {
    let center = t.size / 2;
    let prev_offset = t.size / 4 - t.prev / 4;
    let next_offset = t.size / 4 - t.next / 4;
    let prev_type = usize::from(t.prev == long_blocksize);
    let next_type = usize::from(t.next == long_blocksize);
    for ch in samples.iter_mut() {
        let ch = &mut ch[..t.size];
        ch[..prev_offset].fill(0.0);
        let s = &mut ch[prev_offset..prev_offset + t.prev / 2];
        let w = &windows[prev_type][..s.len()];
        for (x, &wi) in s.iter_mut().zip(w) {
            *x *= wi;
        }
        let s = &mut ch[center + next_offset..center + next_offset + t.next / 2];
        let w = &windows[next_type][t.next / 2..][..s.len()];
        for (x, &wi) in s.iter_mut().zip(w) {
            *x *= wi;
        }
        ch[t.size - next_offset..].fill(0.0);
    }
}

pub(super) fn make_window(size: usize) -> Vec<f32> {
    (0..size).map(|i| window_func((i as f32 + 0.5) / (size / 2) as f32 * std::f32::consts::PI / 2.0)).collect()
}

fn window_func(x: f32) -> f32 {
    let sinx = go_math::sin(f64::from(x));
    go_math::sin(std::f64::consts::FRAC_PI_2 * sinx * sinx) as f32
}
