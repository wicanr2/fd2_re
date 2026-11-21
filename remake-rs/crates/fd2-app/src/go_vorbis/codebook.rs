//! jfreymuth/vorbis v1.0.2 的 `bitreader.go`、`huffman.go`、`codebook.go`。授權見 [`super`] 模組文件。
//!
//! 整數運算照 Go 的無號環繞（`wrapping_*`）；Go 對 32 位元以上的位移得 0，這裡以 [`shl32`] 重現。索引越界與
//! 除以零照 Go 會中止（Go 是 panic）。

use super::VorbisError;

/// Go `x << s`（`uint32`）：`s >= 32` 時為 0。
pub(super) fn shl32(x: u32, s: u32) -> u32 {
    if s >= 32 { 0 } else { x << s }
}

// ---------------- bitreader.go ----------------

pub(super) struct BitReader<'a> {
    data: &'a [u8],
    position: usize,
    bit_offset: u32,
    eof: bool,
}

impl<'a> BitReader<'a> {
    pub(super) fn new(data: &'a [u8]) -> BitReader<'a> {
        BitReader { data, position: 0, bit_offset: 0, eof: false }
    }

    #[allow(dead_code)]
    pub(super) fn eof(&self) -> bool {
        self.eof
    }

    pub(super) fn read1(&mut self) -> u32 {
        if self.position >= self.data.len() {
            self.eof = true;
            return 0;
        }
        let result = u32::from(self.data[self.position] & (1 << self.bit_offset) != 0);
        if self.bit_offset < 7 {
            self.bit_offset += 1;
        } else {
            self.bit_offset = 0;
            self.position += 1;
        }
        result
    }

    fn read(&mut self, mut n: u32, bits: u32) -> u32 {
        if n > bits {
            panic!("invalid argument");
        }
        let mut result: u32 = 0;
        let mut written: u32 = 0;
        let size = n;
        while n > 0 {
            if self.position >= self.data.len() {
                self.eof = true;
                return 0;
            }
            result |= shl32(u32::from(self.data[self.position] >> self.bit_offset), written);
            written += 8 - self.bit_offset;
            if n < 8 - self.bit_offset {
                self.bit_offset += n;
                break;
            }
            n -= 8 - self.bit_offset;
            self.bit_offset = 0;
            self.position += 1;
        }
        result & !shl32(0xFFFF_FFFF, size)
    }

    pub(super) fn read8(&mut self, n: u32) -> u8 {
        self.read(n, 8) as u8
    }

    pub(super) fn read16(&mut self, n: u32) -> u16 {
        self.read(n, 16) as u16
    }

    pub(super) fn read32(&mut self, n: u32) -> u32 {
        self.read(n, 32)
    }

    pub(super) fn read_bool(&mut self) -> bool {
        self.read8(1) == 1
    }
}

// ---------------- huffman.go ----------------

/// Go `huffmanCode`：節點陣列，奇數值是葉（`entry*2 + 1`）。
pub(super) struct HuffmanCode(Vec<u32>);

impl HuffmanCode {
    pub(super) fn lookup(&self, r: &mut BitReader<'_>) -> u32 {
        let mut i: u32 = 0;
        while i & 1 == 0 {
            i = self.0[(i + r.read1()) as usize];
        }
        i >> 1
    }
}

struct HuffmanBuilder {
    code: Vec<u32>,
    min_length: Vec<u8>,
}

impl HuffmanBuilder {
    fn new(size: u32) -> HuffmanBuilder {
        HuffmanBuilder { code: vec![0; size as usize], min_length: vec![0; (size / 2) as usize] }
    }

    fn put_entry(&mut self, entry: u32, length: u8) {
        self.put(0, entry, length.wrapping_sub(1));
    }

    fn put(&mut self, index: u32, entry: u32, length: u8) -> bool {
        let i = index as usize;
        if length < self.min_length[i / 2] {
            return false;
        }
        if length == 0 {
            if self.code[i] == 0 {
                self.code[i] = entry.wrapping_mul(2).wrapping_add(1);
                return true;
            }
            if self.code[i + 1] == 0 {
                self.code[i + 1] = entry.wrapping_mul(2).wrapping_add(1);
                self.min_length[i / 2] = 1;
                return true;
            }
            self.min_length[i / 2] = 1;
            return false;
        }
        if self.code[i] & 1 == 0 {
            if self.code[i] == 0 {
                self.code[i] = self.find_empty(index + 2);
            }
            if self.put(self.code[i], entry, length - 1) {
                return true;
            }
        }
        if self.code[i + 1] & 1 == 0 {
            if self.code[i + 1] == 0 {
                self.code[i + 1] = self.find_empty(index + 2);
            }
            if self.put(self.code[i + 1], entry, length - 1) {
                return true;
            }
        }
        self.min_length[i / 2] = length.wrapping_add(1);
        false
    }

    fn find_empty(&self, mut index: u32) -> u32 {
        while self.code[index as usize] != 0 {
            index += 2;
        }
        index
    }
}

// ---------------- codebook.go ----------------

const CODEBOOK_PATTERN: u32 = 0x564342; // "BCV"

pub(super) struct Codebook {
    pub(super) dimensions: u32,
    entries: HuffmanCode,
    values: Vec<f32>,
}

impl Codebook {
    pub(super) fn read_from(r: &mut BitReader<'_>) -> Result<Codebook, VorbisError> {
        if r.read32(24) != CODEBOOK_PATTERN {
            return Err(VorbisError::Decoding);
        }
        let dimensions = r.read32(16);
        let num_entries = r.read32(24);
        let mut entries = HuffmanBuilder::new(num_entries.wrapping_mul(2).wrapping_sub(2));
        let ordered = r.read_bool();
        if !ordered {
            let sparse = r.read_bool();
            for i in 0..num_entries {
                if !sparse || r.read_bool() {
                    let len = r.read8(5) + 1;
                    entries.put_entry(i, len);
                }
            }
        } else {
            let mut current_entry: u32 = 0;
            let mut current_length: u8 = r.read8(5) + 1;
            while current_entry < num_entries {
                let num = r.read32(ilog(i64::from(num_entries - current_entry)));
                let mut i = current_entry;
                while i < current_entry.wrapping_add(num) {
                    entries.put_entry(i, current_length);
                    i += 1;
                }
                current_entry = current_entry.wrapping_add(num);
                current_length = current_length.wrapping_add(1);
            }
        }
        let mut book = Codebook { dimensions, entries: HuffmanCode(entries.code), values: Vec::new() };

        let lookup_type = r.read8(4);
        if lookup_type == 0 {
            return Ok(book);
        }
        if lookup_type > 2 {
            return Err(VorbisError::Decoding);
        }
        let minimum_value = float32_unpack(r.read32(32));
        let delta_value = float32_unpack(r.read32(32));
        let value_bits = u32::from(r.read8(4)) + 1;
        let sequence_p = r.read_bool();
        let count = if lookup_type == 1 {
            lookup1_values(i64::from(num_entries), dimensions)
        } else {
            i64::from(num_entries).wrapping_mul(i64::from(dimensions))
        };
        let mut multiplicands = vec![0u32; usize::try_from(count).expect("makeslice: len out of range")];
        for m in &mut multiplicands {
            *m = r.read32(value_bits);
        }
        book.values = vec![0f32; num_entries.wrapping_mul(dimensions) as usize];
        let len_mult = multiplicands.len() as i64;
        for entry in 0..i64::from(num_entries) {
            let index = (entry * i64::from(dimensions)) as usize;
            if lookup_type == 1 {
                let mut last = 0f32;
                let mut index_divisor: i64 = 1;
                for i in 0..dimensions as usize {
                    let multiplicand_offset = ((entry / index_divisor) % len_mult) as usize;
                    book.values[index + i] =
                        multiplicands[multiplicand_offset] as f32 * delta_value + minimum_value + last;
                    if sequence_p {
                        last = book.values[index + i];
                    }
                    index_divisor = index_divisor.wrapping_mul(len_mult);
                }
            } else {
                let mut last = 0f32;
                for i in 0..dimensions as usize {
                    book.values[index + i] = multiplicands[index + i] as f32 * delta_value + minimum_value + last;
                    if sequence_p {
                        last = book.values[index + i];
                    }
                }
            }
        }
        Ok(book)
    }

    pub(super) fn decode_scalar(&self, r: &mut BitReader<'_>) -> u32 {
        self.entries.lookup(r)
    }

    pub(super) fn decode_vector(&self, r: &mut BitReader<'_>) -> &[f32] {
        let index = self.entries.lookup(r).wrapping_mul(self.dimensions);
        &self.values[index as usize..index.wrapping_add(self.dimensions) as usize]
    }
}

/// Go `ilog(x int) uint`：`x` 的二進位位數，`x <= 0` 時為 0。
pub(super) fn ilog(mut x: i64) -> u32 {
    let mut r = 0;
    while x > 0 {
        r += 1;
        x >>= 1;
    }
    r
}

/// Go `int(math.Floor(math.Pow(float64(entries), 1/float64(dim))))`。
///
/// Go 的 `math.Pow` 在 amd64 經組語版 `Exp`／`Log` 計算，dim ≥ 3 的完全次方會落在整數之下而比整數根少 1
/// （例如 dim 3、entries 27 得 2）；Rust 的 `powf` 呼叫平台數學函式庫（glibc 得 3），結果不同。這裡以整數根
/// 計算再套用 [`GO_POW_FLOOR_SHORT`]，不依賴平台數學函式庫。表由產生 Go 黃金的主機（Go 1.22.12、
/// linux/amd64、CPU 有 FMA）窮舉 dim 1..24、`r^dim ≤ 2^24` 的完全次方及其 ±1 取得，與整數根不同的 231 組
/// 全部是「完全次方時少 1」；`tools/rust_r5o_audio_pcm.sh` 以 Go 輸出的對照表
/// （`remake/cmd/fd2-r5v-audio-loop -lookup1`）逐值核對，不同時該腳本失敗。
/// dim 0 或 entries < 1 時沿用 `powf`（Go 的 `1/0 = +Inf` 與 0 次方邊界，結果不在小數末位上）。
pub fn lookup1_values(entries: i64, dim: u32) -> i64 {
    if dim == 0 || entries < 1 {
        return go_f64_to_i64((entries as f64).powf(1.0 / f64::from(dim)).floor());
    }
    // 整數根：最大的 r 使 r^dim ≤ entries。
    let fits = |r: i64| {
        let mut p = 1i64;
        for _ in 0..dim {
            match p.checked_mul(r) {
                Some(v) if v <= entries => p = v,
                _ => return false,
            }
        }
        true
    };
    let mut r = ((entries as f64).powf(1.0 / f64::from(dim)).floor() as i64).max(1);
    while !fits(r) {
        r -= 1;
    }
    while fits(r + 1) {
        r += 1;
    }
    if u32::try_from(entries).is_ok_and(|e| GO_POW_FLOOR_SHORT.binary_search(&(dim, e)).is_ok()) {
        r -= 1;
    }
    r
}

/// Go 的 `lookup1Values` 比整數根少 1 的 (dim, entries)，依 (dim, entries) 排序。來源見 [`lookup1_values`]。
#[rustfmt::skip]
const GO_POW_FLOOR_SHORT: [(u32, u32); 231] = [
    (3, 8), (3, 27), (3, 64), (3, 512), (3, 729), (3, 1000), (3, 1728), (3, 2197),
    (3, 2744), (3, 3375), (3, 4096), (3, 4913), (3, 5832), (3, 6859), (3, 8000), (3, 21952),
    (3, 24389), (3, 27000), (3, 29791), (3, 32768), (3, 39304), (3, 42875), (3, 50653), (3, 54872),
    (3, 59319), (3, 68921), (3, 85184), (3, 91125), (3, 103823), (3, 110592), (3, 125000), (3, 132651),
    (3, 140608), (3, 148877), (3, 166375), (3, 175616), (3, 185193), (3, 195112), (3, 205379), (3, 216000),
    (3, 226981), (3, 238328), (3, 250047), (3, 262144), (3, 274625), (3, 287496), (3, 300763), (3, 314432),
    (3, 328509), (3, 343000), (3, 357911), (3, 373248), (3, 389017), (3, 405224), (3, 421875), (3, 493039),
    (3, 512000), (3, 531441), (3, 551368), (3, 571787), (3, 592704), (3, 636056), (3, 681472), (3, 704969),
    (3, 753571), (3, 804357), (3, 830584), (3, 857375), (3, 884736), (3, 941192), (3, 970299), (3, 1000000),
    (3, 1030301), (3, 1061208), (3, 1124864), (3, 1157625), (3, 1191016), (3, 1225043), (3, 1295029), (3, 1331000),
    (3, 1367631), (3, 1404928), (3, 1481544), (3, 1520875), (3, 1560896), (3, 1643032), (3, 1685159), (3, 1728000),
    (3, 1815848), (3, 1860867), (3, 1906624), (3, 1953125), (3, 2048383), (3, 2097152), (3, 2146689), (3, 2197000),
    (3, 2248091), (3, 2299968), (3, 2406104), (3, 2460375), (3, 2515456), (3, 2571353), (3, 2628072), (3, 2685619),
    (3, 2744000), (3, 2863288), (3, 2924207), (3, 2985984), (3, 3048625), (3, 3176523), (3, 3241792), (3, 3307949),
    (3, 3375000), (3, 3581577), (3, 3652264), (3, 3796416), (3, 3869893), (3, 3944312), (3, 4096000), (3, 4251528),
    (3, 4410944), (3, 4492125), (3, 4574296), (3, 4657463), (3, 4741632), (3, 4826809), (3, 4913000), (3, 5000211),
    (3, 5088448), (3, 5177717), (3, 5359375), (3, 5451776), (3, 5545233), (3, 5735339), (3, 5929741), (3, 6028568),
    (3, 6128487), (3, 6331625), (3, 6434856), (3, 6539203), (3, 6644672), (3, 6859000), (3, 7529536), (3, 8000000),
    (3, 8242408), (3, 8365427), (3, 8489664), (3, 8869743), (3, 8998912), (3, 9129329), (3, 9261000), (3, 9393931),
    (3, 9528128), (3, 9800344), (3, 9938375), (3, 10077696), (3, 10218313), (3, 10360232), (3, 10648000), (3, 10793861),
    (3, 10941048), (3, 11089567), (3, 11239424), (3, 11390625), (3, 11543176), (3, 11697083), (3, 12008989), (3, 12167000),
    (3, 12326391), (3, 12487168), (3, 13144256), (3, 13312053), (3, 13481272), (3, 13651919), (3, 13824000), (3, 13997521),
    (3, 14172488), (3, 14348907), (3, 14706125), (3, 15069223), (3, 15252992), (3, 15438249), (3, 15625000), (3, 15813251),
    (3, 16003008), (3, 16194277), (3, 16581375), (3, 16777216), (4, 4096), (4, 38416), (4, 65536), (4, 104976),
    (4, 130321), (4, 160000), (4, 390625), (4, 614656), (4, 1500625), (4, 1874161), (4, 2085136), (4, 2313441),
    (4, 3748096), (4, 4100625), (4, 4879681), (4, 5308416), (4, 5764801), (4, 11316496), (4, 12960000), (4, 15752961),
    (4, 16777216), (5, 32768), (6, 64), (6, 729), (6, 4096), (6, 262144), (6, 531441), (6, 1000000),
    (6, 2985984), (6, 4826809), (6, 7529536), (6, 11390625), (6, 16777216), (7, 823543), (7, 2097152), (7, 10000000),
    (8, 16777216), (9, 19683), (9, 10077696), (12, 4096), (12, 531441), (12, 16777216), (24, 16777216),
];

/// Go 在 amd64 的 `int64(float64)`（`CVTTSD2SQ`）：超出範圍與 NaN 得 `math.MinInt64`。
pub(super) fn go_f64_to_i64(f: f64) -> i64 {
    const TWO_63: f64 = (1u64 << 63) as f64;
    if f.is_nan() || f >= TWO_63 || f < -TWO_63 { i64::MIN } else { f as i64 }
}

/// Go `float32Unpack`：`float32(math.Ldexp(mantissa, int(exponent)-788))`。指數範圍 −788..235，`2^k` 是正規數，
/// 乘法精確；轉 `f32` 與 Go 一樣四捨五入。
pub(super) fn float32_unpack(x: u32) -> f32 {
    let mut mantissa = f64::from(x & 0x1f_ffff);
    if x & 0x8000_0000 != 0 {
        mantissa = -mantissa;
    }
    let exponent = ((x & 0x7fe0_0000) >> 21) as i64;
    let k = exponent - 788;
    (mantissa * f64::from_bits(((k + 1023) as u64) << 52)) as f32
}
