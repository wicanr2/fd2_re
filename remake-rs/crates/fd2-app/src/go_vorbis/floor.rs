//! jfreymuth/vorbis v1.0.2 的 `floor0.go`、`floor1.go`。授權見 [`super`] 模組文件。
//!
//! floor1 的整數運算照 Go 的 `uint32` 環繞。`sort.Sort` 不是穩定排序，這裡用穩定排序；兩者只在 X 座標
//! 重複時可能不同，Vorbis I 規格要求 X 座標互不相同。
//!
//! floor0 照 Go 的式子移植（包括 `q = 1 / 4` 在 Go 是整數常數除法、值為 0，以及 `mapResult` 的
//! `rate*i/2*n`）。三角函數用 `fd2_formats::go_math`（Go `math.Cos`）；`math.Exp`、`math.Atan` 用 Rust 的
//! `f64::exp`、`f64::atan`，末位可能與 Go 不同。分離素材包與 `remake/assets` 的 OGG 都只用 floor1
//! （`r5o_audio_pcm` 的 `floor_types` 欄），floor0 沒有逐樣本驗證。

use fd2_formats::go_math;

use super::codebook::{BitReader, Codebook, ilog};
use super::inverse_db_table::INVERSE_DB_TABLE;

/// Go `floor` 介面的兩種實作。
pub(super) enum Floor {
    Floor0(Floor0),
    Floor1(Floor1),
}

/// Go `Decode` 回傳的 `interface{}`；`None` 即 Go 的 `nil`。
pub(super) enum FloorPayload {
    Floor0 { amplitude: u32, coefficients: Vec<f32> },
    Floor1(Vec<u32>),
}

impl Floor {
    pub(super) fn type_number(&self) -> u16 {
        match self {
            Floor::Floor0(_) => 0,
            Floor::Floor1(_) => 1,
        }
    }

    pub(super) fn decode(&self, r: &mut BitReader<'_>, books: &[Codebook], n: u32) -> Option<FloorPayload> {
        match self {
            Floor::Floor0(f) => f.decode(r, books, n),
            Floor::Floor1(f) => f.decode(r, books, n),
        }
    }

    pub(super) fn apply(&mut self, out: &mut [f32], data: &FloorPayload) {
        match (self, data) {
            (Floor::Floor0(f), FloorPayload::Floor0 { amplitude, coefficients }) => {
                f.apply(out, *amplitude, coefficients)
            }
            (Floor::Floor1(f), FloorPayload::Floor1(y)) => f.apply(out, y),
            // Go 的型別斷言失敗會 panic；同一個 floor 只會收到自己 Decode 的結果。
            _ => panic!("interface conversion: floor data type mismatch"),
        }
    }
}

// ---------------- floor0.go ----------------

pub(super) struct Floor0 {
    order: u8,
    rate: u16,
    bark_map_size: u16,
    amplitude_bits: u8,
    amplitude_offset: u8,
    book_list: Vec<u8>,
}

impl Floor0 {
    pub(super) fn read_from(r: &mut BitReader<'_>) -> Floor0 {
        let order = r.read8(8);
        let rate = r.read16(16);
        let bark_map_size = r.read16(16);
        let amplitude_bits = r.read8(6);
        let amplitude_offset = r.read8(8);
        let n = usize::from(r.read8(4)) + 1;
        let book_list = (0..n).map(|_| r.read8(8)).collect();
        Floor0 { order, rate, bark_map_size, amplitude_bits, amplitude_offset, book_list }
    }

    fn decode(&self, r: &mut BitReader<'_>, books: &[Codebook], _n: u32) -> Option<FloorPayload> {
        let amplitude = r.read32(u32::from(self.amplitude_bits));
        if amplitude == 0 {
            return None;
        }
        let book_number = r.read8(ilog(self.book_list.len() as i64));
        let book = &books[usize::from(self.book_list[usize::from(book_number)])];
        let mut coefficients = vec![0f32; usize::from(self.order)];
        let mut i = 0usize;
        let mut last = 0f32;
        loop {
            let temp_vector = book.decode_vector(r);
            for &c in temp_vector {
                coefficients[i] = c + last;
                i += 1;
                if i >= coefficients.len() {
                    return Some(FloorPayload::Floor0 { amplitude, coefficients });
                }
            }
            last = temp_vector[temp_vector.len().wrapping_sub(1)];
        }
    }

    fn apply(&self, out: &mut [f32], amplitude: u32, coefficients: &[f32]) {
        let n = out.len() as u32;
        let mut i: u32 = 0;
        let order = self.order;
        while i < n {
            let mapi = self.map_result(i, n);
            let w = std::f64::consts::PI * mapi as f64 / f64::from(self.bark_map_size);
            let cosw = go_math::cos(w);
            let (mut p, mut q);
            if order % 2 == 1 {
                p = 1.0 - cosw * cosw;
                for j in 0..=(i64::from(order.wrapping_sub(3)) / 2) {
                    let tmp = go_math::cos(f64::from(coefficients[(2 * j + 1) as usize])) - cosw;
                    p *= 4.0 * tmp * tmp;
                }
                // Go 原式 `q = 1 / 4`：整數常數除法，值為 0。
                q = 0.0;
                for j in 0..=(i64::from(order.wrapping_sub(1)) / 2) {
                    let tmp = go_math::cos(f64::from(coefficients[(2 * j) as usize])) - cosw;
                    q *= 4.0 * tmp * tmp;
                }
            } else {
                p = (1.0 - cosw * cosw) / 2.0;
                for j in 0..=(i64::from(order.wrapping_sub(2)) / 2) {
                    let tmp = go_math::cos(f64::from(coefficients[(2 * j + 1) as usize])) - cosw;
                    p *= 4.0 * tmp * tmp;
                }
                q = (1.0 + cosw * cosw) / 2.0;
                for j in 0..=(i64::from(order.wrapping_sub(2)) / 2) {
                    let tmp = go_math::cos(f64::from(coefficients[(2 * j) as usize])) - cosw;
                    q *= 4.0 * tmp * tmp;
                }
            }
            let bits_max = ((1u64 << self.amplitude_bits) - 1) as f64;
            let offset = f64::from(self.amplitude_offset);
            let linear_floor_value =
                (0.11512925 * (f64::from(amplitude) * offset / (bits_max * (p + q).sqrt()) - offset)).exp();
            while self.map_result(i, n) == mapi {
                out[i as usize] *= linear_floor_value as f32;
                i += 1;
            }
        }
    }

    fn map_result(&self, i: u32, n: u32) -> i64 {
        if i >= n {
            return -1;
        }
        let rate = f64::from(self.rate);
        let b = (bark(rate * f64::from(i) / 2.0 * f64::from(n)) * f64::from(self.bark_map_size) / bark(0.5 * rate))
            .floor() as i64;
        if b > i64::from(self.bark_map_size) - 1 {
            return i64::from(self.bark_map_size) - 1;
        }
        b
    }
}

fn bark(x: f64) -> f64 {
    13.1 * (0.00074 * x).atan() + 2.24 * (0.0000000185 * x * x).atan() + 0.0001 * x
}

// ---------------- floor1.go ----------------

struct Floor1Class {
    dimension: u8,
    subclass: u8,
    masterbook: u8,
    subclass_books: Vec<u8>,
}

pub(super) struct Floor1 {
    partition_class_list: Vec<u8>,
    classes: Vec<Floor1Class>,
    multiplier: u8,
    #[allow(dead_code)]
    rangebits: u8,
    x_list: Vec<u32>,
    sort: Vec<u32>,
    step2: Vec<bool>,
    final_y: Vec<u32>,
}

const RANGES: [u32; 4] = [256, 128, 86, 64];

impl Floor1 {
    pub(super) fn read_from(r: &mut BitReader<'_>) -> Floor1 {
        let n = usize::from(r.read8(5));
        let mut partition_class_list = Vec::with_capacity(n);
        let mut maximum_class: u8 = 0;
        for _ in 0..n {
            let class = r.read8(4);
            partition_class_list.push(class);
            if class > maximum_class {
                maximum_class = class;
            }
        }
        let mut classes = Vec::with_capacity(usize::from(maximum_class) + 1);
        for _ in 0..=maximum_class {
            let dimension = r.read8(3) + 1;
            let subclass = r.read8(2);
            let masterbook = if subclass != 0 { r.read8(8) } else { 0 };
            let subclass_books = (0..1usize << subclass).map(|_| r.read8(8).wrapping_sub(1)).collect();
            classes.push(Floor1Class { dimension, subclass, masterbook, subclass_books });
        }
        let multiplier = r.read8(2) + 1;
        let rangebits = r.read8(4);
        let mut x_list = vec![0, 1u32 << rangebits];
        for &class in &partition_class_list {
            for _ in 0..classes[usize::from(class)].dimension {
                x_list.push(r.read32(u32::from(rangebits)));
            }
        }
        let mut sort: Vec<u32> = (0..x_list.len() as u32).collect();
        sort.sort_by_key(|&i| x_list[i as usize]);
        let len = x_list.len();
        Floor1 {
            partition_class_list,
            classes,
            multiplier,
            rangebits,
            x_list,
            sort,
            step2: vec![false; len],
            final_y: vec![0; len],
        }
    }

    fn decode(&self, r: &mut BitReader<'_>, books: &[Codebook], _n: u32) -> Option<FloorPayload> {
        if !r.read_bool() {
            return None;
        }
        let range = RANGES[usize::from(self.multiplier) - 1];
        let mut y: Vec<u32> = Vec::with_capacity(self.x_list.len());
        let bits = ilog(i64::from(range) - 1);
        y.push(r.read32(bits));
        y.push(r.read32(bits));
        for &class_index in &self.partition_class_list {
            let class = &self.classes[usize::from(class_index)];
            let cdim = class.dimension;
            let cbits = u32::from(class.subclass);
            let csub = (1u32 << cbits) - 1;
            let mut cval: u32 = 0;
            if cbits > 0 {
                cval = books[usize::from(class.masterbook)].decode_scalar(r);
            }
            for _ in 0..cdim {
                let book = class.subclass_books[(cval & csub) as usize];
                cval >>= cbits;
                if book != 0xFF {
                    y.push(books[usize::from(book)].decode_scalar(r));
                } else {
                    y.push(0);
                }
            }
        }
        Some(FloorPayload::Floor1(y))
    }

    #[allow(clippy::needless_range_loop)]
    fn apply(&mut self, out: &mut [f32], y: &[u32]) {
        let n = out.len() as u32;
        let range = RANGES[usize::from(self.multiplier) - 1];

        self.step2[0] = true;
        self.step2[1] = true;
        self.final_y[0] = y[0];
        self.final_y[1] = y[1];

        for i in 2..self.x_list.len() {
            let low = low_neighbor(&self.x_list, i);
            let high = high_neighbor(&self.x_list, i);
            let predicted = render_point(
                self.x_list[low],
                self.final_y[low],
                self.x_list[high],
                self.final_y[high],
                self.x_list[i],
            );
            let val = y[i];

            let high_room = range.wrapping_sub(predicted);
            let low_room = predicted;
            let room = if high_room < low_room { high_room.wrapping_mul(2) } else { low_room.wrapping_mul(2) };

            if val == 0 {
                self.step2[i] = false;
                self.final_y[i] = predicted;
            } else {
                self.step2[low] = true;
                self.step2[high] = true;
                self.step2[i] = true;
                if val >= room {
                    if high_room > low_room {
                        self.final_y[i] = val.wrapping_sub(low_room).wrapping_add(predicted);
                    } else {
                        self.final_y[i] = predicted.wrapping_sub(val).wrapping_add(high_room).wrapping_sub(1);
                    }
                } else if val % 2 == 1 {
                    self.final_y[i] = predicted.wrapping_sub(val.wrapping_add(1) / 2);
                } else {
                    self.final_y[i] = predicted.wrapping_add(val / 2);
                }
            }
        }

        let mult = u32::from(self.multiplier);
        let (mut hx, mut lx): (u32, u32) = (0, 0);
        let mut ly = self.final_y[0].wrapping_mul(mult);
        let mut hy: u32 = 0;
        for j in 1..self.final_y.len() {
            let i = self.sort[j] as usize;
            if self.step2[i] {
                hy = self.final_y[i].wrapping_mul(mult);
                hx = self.x_list[i];
                render_line(lx, ly, hx, hy, out);
                lx = hx;
                ly = hy;
            }
        }

        if hx < n {
            for i in hx..n {
                out[i as usize] *= INVERSE_DB_TABLE[hy as usize];
            }
        }
    }
}

fn low_neighbor(v: &[u32], index: usize) -> usize {
    let val = v[index];
    let mut best = 0;
    let mut max: u32 = 0;
    for (i, &x) in v.iter().enumerate().take(index).skip(1) {
        if x >= val {
            continue;
        }
        if x > max {
            best = i;
            max = x;
        }
    }
    best
}

fn high_neighbor(v: &[u32], index: usize) -> usize {
    let val = v[index];
    let mut best = 0;
    let mut min: u32 = 0xffff_ffff;
    for (i, &x) in v.iter().enumerate().take(index).skip(1) {
        if x <= val {
            continue;
        }
        if x < min {
            best = i;
            min = x;
        }
    }
    best
}

fn render_point(x0: u32, y0: u32, x1: u32, y1: u32, x: u32) -> u32 {
    let dy = i64::from(y1) - i64::from(y0);
    let adx = x1.wrapping_sub(x0);
    let mut ady = y1.wrapping_sub(y0);
    if dy < 0 {
        ady = (-dy) as u32;
    }
    let err = ady.wrapping_mul(x.wrapping_sub(x0));
    let off = err / adx;
    if dy < 0 { y0.wrapping_sub(off) } else { y0.wrapping_add(off) }
}

fn render_line(x0: u32, y0: u32, x1: u32, y1: u32, v: &mut [f32]) {
    let dy = i64::from(y1) - i64::from(y0);
    let adx = x1.wrapping_sub(x0);
    let mut ady = y1.wrapping_sub(y0);
    if dy < 0 {
        ady = (-dy) as u32;
    }
    let base = dy / i64::from(adx);
    let mut y = y0;
    let mut err: u32 = 0;
    let sy = if dy < 0 { base - 1 } else { base + 1 };

    let mut abs_base = base as u32;
    if base < 0 {
        abs_base = abs_base.wrapping_neg();
    }
    ady = ady.wrapping_sub(abs_base.wrapping_mul(adx));

    v[x0 as usize] *= INVERSE_DB_TABLE[y as usize];
    let mut x = x0.wrapping_add(1);
    while x < x1 {
        err = err.wrapping_add(ady);
        if err >= adx {
            err -= adx;
            y = (i64::from(y) + sy) as u32;
        } else {
            y = (i64::from(y) + base) as u32;
        }
        v[x as usize] *= INVERSE_DB_TABLE[y as usize];
        x += 1;
    }
}
