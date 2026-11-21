//! `github.com/jfreymuth/vorbis` v1.0.2 的逐行移植（R5 階段 4，代號 r5v；`rust-port-no-go-stdlib` 的具名例外）。
//!
//! Go 版 `cmd/fd2` 經 Ebiten v2.6.6 `audio/vorbis` → `github.com/jfreymuth/oggvorbis` v1.0.5 → 本套件解 OGG。
//! 使用者 2026-11-16 定案音訊採 C 案：解碼器與重取樣都照 Go 移植，目標與 Go 逐樣本相同（docs/goal/121
//! 「階段 4 使用者定案」）。lewton 與本套件的浮點運算順序不同，轉 16 位元時約 0.07% 的樣本差 1，所以不能沿用
//! lewton；本模組照 Go 原始碼的運算順序與整數環繞重寫，不是依 Vorbis I 規格另寫一份解碼器，與規格不同之處
//! （例如殘差型別 2 只看第一個聲道的 `doNotDecode`）也照 Go。
//!
//! 檔案對應：`vorbis.go`、`header.go`、`setup.go`、`decode.go` 在本檔；`bitreader.go`、`huffman.go`、
//! `codebook.go` 在 [`codebook`]；`floor0.go`、`floor1.go` 在 [`floor`]；`residue.go` 在 [`residue`]；
//! `imdct.go`、`window.go` 在 [`imdct`]；`inversedbtable.go` 在 [`inverse_db_table`]。
//! 只移植 `cmd/fd2` 用到的 API（`ReadHeader`、`DecodeInto`、`Clear`、`BufferSize` 等）。
//!
//! 原授權（`third_party/jfreymuth-vorbis-v1.0.2-LICENSE` 為原檔）：
//!
//! ```text
//! MIT License
//!
//! Copyright (c) 2016 Johann Freymuth
//!
//! Permission is hereby granted, free of charge, to any person obtaining a copy
//! of this software and associated documentation files (the "Software"), to deal
//! in the Software without restriction, including without limitation the rights
//! to use, copy, modify, merge, publish, distribute, sublicense, and/or sell
//! copies of the Software, and to permit persons to whom the Software is
//! furnished to do so, subject to the following conditions:
//!
//! The above copyright notice and this permission notice shall be included in all
//! copies or substantial portions of the Software.
//!
//! THE SOFTWARE IS PROVIDED "AS IS", WITHOUT WARRANTY OF ANY KIND, EXPRESS OR
//! IMPLIED, INCLUDING BUT NOT LIMITED TO THE WARRANTIES OF MERCHANTABILITY,
//! FITNESS FOR A PARTICULAR PURPOSE AND NONINFRINGEMENT. IN NO EVENT SHALL THE
//! AUTHORS OR COPYRIGHT HOLDERS BE LIABLE FOR ANY CLAIM, DAMAGES OR OTHER
//! LIABILITY, WHETHER IN AN ACTION OF CONTRACT, TORT OR OTHERWISE, ARISING FROM,
//! OUT OF OR IN CONNECTION WITH THE SOFTWARE OR THE USE OR OTHER DEALINGS IN THE
//! SOFTWARE.
//! ```

mod codebook;
mod floor;
mod imdct;
mod inverse_db_table;
mod residue;

/// Go `lookup1Values`（比對工具以 Go 的對照表核對，見其文件）。
pub use codebook::lookup1_values;
use codebook::{BitReader, Codebook, ilog};
use floor::{Floor, Floor0, Floor1, FloorPayload};
use imdct::{ImdctLookup, WindowType, apply_window, generate_imdct_lookup, imdct, make_window};
use residue::Residue;

/// 本套件的錯誤（文字照 Go）。
#[derive(Debug, Clone, PartialEq, Eq)]
pub enum VorbisError {
    /// `vorbis: decoding error`。
    Decoding,
    /// `vorbis: invalid header`。
    InvalidHeader,
    /// `vorbis: unknown header type`。
    UnknownHeaderType,
    /// `vorbis: missing headers`。
    MissingHeaders,
    /// `vorbis: buffer too short`。
    BufferTooShort,
}

impl std::fmt::Display for VorbisError {
    fn fmt(&self, f: &mut std::fmt::Formatter<'_>) -> std::fmt::Result {
        f.write_str(match self {
            VorbisError::Decoding => "vorbis: decoding error",
            VorbisError::InvalidHeader => "vorbis: invalid header",
            VorbisError::UnknownHeaderType => "vorbis: unknown header type",
            VorbisError::MissingHeaders => "vorbis: missing headers",
            VorbisError::BufferTooShort => "vorbis: buffer too short",
        })
    }
}

impl std::error::Error for VorbisError {}

/// Go `Bitrate`。
#[derive(Debug, Clone, Copy, Default, PartialEq, Eq)]
pub struct Bitrate {
    pub nominal: i64,
    pub minimum: i64,
    pub maximum: i64,
}

/// Go `CommentHeader`。
#[derive(Debug, Clone, Default, PartialEq, Eq)]
pub struct CommentHeader {
    pub vendor: Vec<u8>,
    pub comments: Vec<Vec<u8>>,
}

struct Mapping {
    coupling_steps: u16,
    angle: Vec<u8>,
    magnitude: Vec<u8>,
    mux: Vec<u8>,
    submaps: Vec<(u8, u8)>,
}

struct Mode {
    blockflag: u8,
    mapping: u8,
}

struct FloorData {
    floor: usize,
    data: Option<FloorPayload>,
    no_residue: bool,
}

const HEADER_TYPE_IDENTIFICATION: u8 = 1;
const HEADER_TYPE_COMMENT: u8 = 3;
const HEADER_TYPE_SETUP: u8 = 5;

/// Go `Decoder`。
#[derive(Default)]
pub struct Decoder {
    header_read: bool,
    setup_read: bool,

    sample_rate: i64,
    channels: usize,
    pub bitrate: Bitrate,
    blocksize: [usize; 2],
    pub comment_header: CommentHeader,

    codebooks: Vec<Codebook>,
    floors: Vec<Floor>,
    residues: Vec<Residue>,
    mappings: Vec<Mapping>,
    modes: Vec<Mode>,

    overlap: Vec<f32>,
    has_overlap: bool,
    overlap_short: bool,

    windows: [Vec<f32>; 2],
    lookup: [Option<ImdctLookup>; 2],
    residue_buffer: Vec<Vec<f32>>,
    floor_buffer: Vec<FloorData>,
    raw_buffer: Vec<Vec<f32>>,
}

/// Go `IsHeader`。
pub fn is_header(packet: &[u8]) -> bool {
    packet.len() > 6 && packet[0] & 1 == 1 && &packet[1..7] == b"vorbis"
}

impl Decoder {
    pub fn new() -> Decoder {
        Decoder::default()
    }

    /// Go `SampleRate()`（Go `int`）。
    pub fn sample_rate(&self) -> i64 {
        self.sample_rate
    }

    pub fn channels(&self) -> usize {
        self.channels
    }

    /// Go `BufferSize()`：一個封包最多解出的值數（已乘聲道數）。
    pub fn buffer_size(&self) -> usize {
        self.blocksize[1] / 2 * self.channels
    }

    /// 設定標頭裡各 floor 的型別號（診斷用：記錄素材只用到 floor1）。
    pub fn floor_types(&self) -> Vec<u16> {
        self.floors.iter().map(Floor::type_number).collect()
    }

    /// Go `ReadHeader`。
    pub fn read_header(&mut self, header: &[u8]) -> Result<(), VorbisError> {
        if !is_header(header) {
            return Err(VorbisError::InvalidHeader);
        }
        let header_type = header[0];
        let header = &header[7..];
        match header_type {
            HEADER_TYPE_IDENTIFICATION => {
                self.read_identification_header(header)?;
                self.header_read = true;
            }
            HEADER_TYPE_COMMENT => return self.read_comment_header(header),
            HEADER_TYPE_SETUP => {
                self.read_setup_header(header)?;
                self.overlap = vec![0.0; self.blocksize[1] * self.channels];
                self.setup_read = true;
            }
            _ => return Err(VorbisError::UnknownHeaderType),
        }
        Ok(())
    }

    pub fn headers_read(&self) -> bool {
        self.header_read && self.setup_read
    }

    /// Go `DecodeInto`：解一個封包寫進 `buffer`，回傳寫入的值數（Go 回傳 `buffer[:n]`）。
    pub fn decode_into(&mut self, packet: &[u8], buffer: &mut [f32]) -> Result<usize, VorbisError> {
        if !self.headers_read() {
            return Err(VorbisError::MissingHeaders);
        }
        if buffer.len() < self.buffer_size() {
            return Err(VorbisError::BufferTooShort);
        }
        self.decode_packet(&mut BitReader::new(packet), buffer)
    }

    /// Go `Clear`：兩個不相鄰的封包之間呼叫。
    pub fn clear(&mut self) {
        self.has_overlap = false;
    }

    // ---------------- header.go ----------------

    fn read_identification_header(&mut self, h: &[u8]) -> Result<(), VorbisError> {
        if h.len() <= 22 {
            return Err(VorbisError::Decoding);
        }
        let le32 = |at: usize| u32::from_le_bytes([h[at], h[at + 1], h[at + 2], h[at + 3]]);
        if le32(0) != 0 {
            return Err(VorbisError::Decoding);
        }
        self.channels = usize::from(h[4]);
        self.sample_rate = i64::from(le32(5));
        self.bitrate.maximum = i64::from(le32(9));
        self.bitrate.nominal = i64::from(le32(13));
        self.bitrate.minimum = i64::from(le32(17));
        self.blocksize[0] = 1 << (h[21] & 0x0F);
        self.blocksize[1] = 1 << (h[21] >> 4);
        if h[22] & 1 == 0 {
            return Err(VorbisError::Decoding);
        }
        Ok(())
    }

    /// Go 以 `recover()` 接住越界 panic，但具名回傳值 `err` 在 `defer` 前已求值，所以越界時仍回傳 `nil`；
    /// 這裡同樣不回報錯誤，只在越界處停下（已讀到的欄位保留）。
    fn read_comment_header(&mut self, h: &[u8]) -> Result<(), VorbisError> {
        let le32 = |b: &[u8]| b.get(..4).map(|b| u32::from_le_bytes([b[0], b[1], b[2], b[3]]));
        let mut h = h;
        let Some(vendor_len) = le32(h) else { return Ok(()) };
        h = &h[4..];
        let Some(vendor) = h.get(..vendor_len as usize) else { return Ok(()) };
        self.comment_header.vendor = vendor.to_vec();
        h = &h[vendor_len as usize..];
        let Some(num_comments) = le32(h) else { return Ok(()) };
        self.comment_header.comments = vec![Vec::new(); num_comments as usize];
        h = &h[4..];
        for i in 0..num_comments as usize {
            let Some(comment_len) = le32(h) else { return Ok(()) };
            h = &h[4..];
            let Some(comment) = h.get(..comment_len as usize) else { return Ok(()) };
            self.comment_header.comments[i] = comment.to_vec();
            h = &h[comment_len as usize..];
        }
        Ok(())
    }

    // ---------------- setup.go ----------------

    fn read_setup_header(&mut self, header: &[u8]) -> Result<(), VorbisError> {
        let mut r = BitReader::new(header);

        // CODEBOOKS
        let n = usize::from(r.read16(8)) + 1;
        self.codebooks = Vec::with_capacity(n);
        for _ in 0..n {
            self.codebooks.push(Codebook::read_from(&mut r)?);
        }

        // TIME DOMAIN TRANSFORMS
        let transform_count = r.read8(6) + 1;
        for _ in 0..transform_count {
            if r.read16(16) != 0 {
                return Err(VorbisError::Decoding);
            }
        }

        // FLOORS
        let n = usize::from(r.read8(6)) + 1;
        self.floors = Vec::with_capacity(n);
        for _ in 0..n {
            match r.read16(16) {
                0 => self.floors.push(Floor::Floor0(Floor0::read_from(&mut r))),
                1 => self.floors.push(Floor::Floor1(Floor1::read_from(&mut r))),
                _ => return Err(VorbisError::Decoding),
            }
        }

        // RESIDUES
        let n = usize::from(r.read8(6)) + 1;
        self.residues = Vec::with_capacity(n);
        for _ in 0..n {
            self.residues.push(Residue::read_from(&mut r)?);
        }

        // MAPPINGS
        let n = usize::from(r.read8(6)) + 1;
        self.mappings = Vec::with_capacity(n);
        let channel_bits = ilog(self.channels as i64 - 1);
        for _ in 0..n {
            if r.read16(16) != 0 {
                return Err(VorbisError::Decoding);
            }
            let submap_count = if r.read_bool() { usize::from(r.read8(4)) + 1 } else { 1 };
            let mut m = Mapping {
                coupling_steps: 0,
                angle: Vec::new(),
                magnitude: Vec::new(),
                mux: Vec::new(),
                submaps: vec![(0, 0); submap_count],
            };
            if r.read_bool() {
                m.coupling_steps = r.read16(8) + 1;
                m.magnitude = vec![0; usize::from(m.coupling_steps)];
                m.angle = vec![0; usize::from(m.coupling_steps)];
                for i in 0..usize::from(m.coupling_steps) {
                    m.magnitude[i] = r.read8(channel_bits);
                    m.angle[i] = r.read8(channel_bits);
                }
            }
            if r.read8(2) != 0 {
                return Err(VorbisError::Decoding);
            }
            m.mux = vec![0; self.channels];
            if m.submaps.len() > 1 {
                for mux in &mut m.mux {
                    *mux = r.read8(4);
                }
            }
            for submap in &mut m.submaps {
                r.read8(8);
                submap.0 = r.read8(8);
                submap.1 = r.read8(8);
            }
            self.mappings.push(m);
        }

        // MODES
        let n = usize::from(r.read8(6)) + 1;
        self.modes = Vec::with_capacity(n);
        for _ in 0..n {
            let blockflag = r.read8(1);
            if r.read16(16) != 0 {
                return Err(VorbisError::Decoding);
            }
            if r.read16(16) != 0 {
                return Err(VorbisError::Decoding);
            }
            let mapping = r.read8(8);
            self.modes.push(Mode { blockflag, mapping });
        }

        if !r.read_bool() {
            return Err(VorbisError::Decoding);
        }
        self.init_lookup();
        Ok(())
    }

    fn init_lookup(&mut self) {
        self.windows = [make_window(self.blocksize[0]), make_window(self.blocksize[1])];
        self.lookup = [Some(generate_imdct_lookup(self.blocksize[0])), Some(generate_imdct_lookup(self.blocksize[1]))];
        self.residue_buffer = vec![vec![0.0; self.blocksize[1] / 2]; self.channels];
        self.raw_buffer = vec![vec![0.0; self.blocksize[1]]; self.channels];
    }

    // ---------------- decode.go ----------------

    fn decode_packet(&mut self, r: &mut BitReader<'_>, out: &mut [f32]) -> Result<usize, VorbisError> {
        if r.read_bool() {
            return Err(VorbisError::Decoding);
        }
        let mode_number = r.read8(ilog(self.modes.len() as i64 - 1));
        let mode = &self.modes[usize::from(mode_number)];
        // decode window type
        let blocktype = usize::from(mode.blockflag);
        let long_window = mode.blockflag == 1;
        let mapping_index = usize::from(mode.mapping);
        let blocksize = self.blocksize[blocktype];
        let spectrum_size = (blocksize / 2) as u32;
        let mut window_prev = false;
        let mut window = WindowType { size: blocksize, prev: blocksize, next: blocksize };
        if long_window {
            window_prev = r.read_bool();
            let window_next = r.read_bool();
            if !window_prev {
                window.prev = self.blocksize[0];
            }
            if !window_next {
                window.next = self.blocksize[0];
            }
        }

        let channels = self.channels;
        if self.floor_buffer.is_empty() {
            self.floor_buffer = (0..channels).map(|_| FloorData { floor: 0, data: None, no_residue: true }).collect();
        }
        let spectrum = spectrum_size as usize;
        for buf in &mut self.residue_buffer {
            buf[..spectrum].fill(0.0);
        }

        let mapping = &self.mappings[mapping_index];
        // decodeFloors
        for ch in 0..self.floor_buffer.len() {
            let floor = usize::from(mapping.submaps[usize::from(mapping.mux[ch])].0);
            let data = self.floors[floor].decode(r, &self.codebooks, spectrum_size);
            let no_residue = data.is_none();
            self.floor_buffer[ch] = FloorData { floor, data, no_residue };
        }
        for i in 0..usize::from(mapping.coupling_steps) {
            let (m, a) = (usize::from(mapping.magnitude[i]), usize::from(mapping.angle[i]));
            if !self.floor_buffer[m].no_residue || !self.floor_buffer[a].no_residue {
                self.floor_buffer[m].no_residue = false;
                self.floor_buffer[a].no_residue = false;
            }
        }

        // decodeResidue
        for (i, submap) in mapping.submaps.iter().enumerate() {
            let mut do_not_decode = Vec::with_capacity(channels);
            let mut tmp: Vec<&mut [f32]> = Vec::with_capacity(channels);
            for (j, buf) in self.residue_buffer.iter_mut().enumerate().take(channels) {
                if mapping.mux[j] == i as u8 {
                    do_not_decode.push(self.floor_buffer[j].no_residue);
                    tmp.push(&mut buf[..spectrum]);
                }
            }
            self.residues[usize::from(submap.1)].decode(r, &do_not_decode, spectrum_size, &self.codebooks, &mut tmp);
        }

        // inverseCoupling
        for i in (1..=usize::from(mapping.coupling_steps)).rev() {
            let (mi, ai) = (usize::from(mapping.magnitude[i - 1]), usize::from(mapping.angle[i - 1]));
            for j in 0..spectrum {
                let mut m = self.residue_buffer[mi][j];
                let mut a = self.residue_buffer[ai][j];
                if m > 0.0 {
                    if a > 0.0 {
                        (m, a) = (m, m - a);
                    } else {
                        (a, m) = (m, m + a);
                    }
                } else if a > 0.0 {
                    (m, a) = (m, m + a);
                } else {
                    (a, m) = (m, m - a);
                }
                self.residue_buffer[mi][j] = m;
                self.residue_buffer[ai][j] = a;
            }
        }

        // applyFloor
        for ch in 0..self.residue_buffer.len() {
            let residue = &mut self.residue_buffer[ch][..spectrum];
            let fd = &self.floor_buffer[ch];
            match &fd.data {
                Some(data) => self.floors[fd.floor].apply(residue, data),
                None => residue.fill(0.0),
            }
        }

        // inverse MDCT
        let lookup = self.lookup[blocktype].as_ref().expect("init_lookup");
        for ch in 0..self.raw_buffer.len() {
            imdct(lookup, &mut self.residue_buffer[ch][..spectrum], &mut self.raw_buffer[ch][..blocksize]);
        }

        // apply window and overlap
        apply_window(&window, &self.windows, self.blocksize[1], &mut self.raw_buffer);
        let center = blocksize / 2;
        let offset = self.blocksize[1] / 4 - self.blocksize[0] / 4;
        let mut n = 0usize;
        if self.has_overlap {
            n = blocksize / 2;
            if long_window && !window_prev {
                n -= offset;
            }
            if !long_window && !self.overlap_short {
                n += offset;
            }
        }
        let raw = &self.raw_buffer;
        let overlap = &mut self.overlap;
        if long_window {
            let start = if window_prev { 0 } else { offset };
            if self.has_overlap {
                for (ch, raw) in raw.iter().enumerate() {
                    for i in 0..center - start {
                        out[i * channels + ch] = raw[start + i] + overlap[(start + i) * channels + ch];
                    }
                }
            }
            self.overlap_short = false;
        } else {
            if self.has_overlap {
                if self.overlap_short {
                    for (ch, raw) in raw.iter().enumerate() {
                        for i in 0..center {
                            out[i * channels + ch] = raw[i] + overlap[(offset + i) * channels + ch];
                        }
                    }
                } else {
                    out[..offset * channels].copy_from_slice(&overlap[..offset * channels]);
                    for (ch, raw) in raw.iter().enumerate() {
                        for i in offset..offset + center {
                            out[i * channels + ch] = raw[i - offset] + overlap[i * channels + ch];
                        }
                    }
                }
            }
            self.overlap_short = true;
        }

        if !self.has_overlap {
            n = 0;
        }
        let overlap_center = self.blocksize[1] / 4;
        let o_start = overlap_center - center / 2;
        let o_end = overlap_center + center / 2;
        overlap[..o_start * channels].fill(0.0);
        for (ch, raw) in raw.iter().enumerate() {
            for i in o_start..o_end {
                overlap[i * channels + ch] = raw[center + i - o_start];
            }
        }
        overlap[o_end * channels..].fill(0.0);
        self.has_overlap = true;

        Ok(n * channels)
    }
}

#[cfg(test)]
#[path = "go_vorbis/go_vorbis_test.rs"]
mod go_vorbis_test;
