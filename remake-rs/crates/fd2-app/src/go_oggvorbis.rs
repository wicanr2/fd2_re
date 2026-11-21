//! `github.com/jfreymuth/oggvorbis` v1.0.5 的逐行移植（R5 階段 4，代號 r5v；`rust-port-no-go-stdlib` 的具名例外）。
//!
//! Ogg 容器讀取（`ogg.go`、`crc.go`）與 `Reader`（`reader.go`）。Go 版 BGM 的結尾行為由這裡決定：
//! `NewReader` 以最後一頁的 granule 當長度（`LastPosition`），解到最後一個封包時依所在頁的 granule 截掉多出的
//! 樣本；Ebiten 解碼時的 `SetPosition(0)` 讓位置改從 0 自行計數（docs/goal/121「階段 4 結果：r5o」的結尾分析）。
//! 這些都照 Go 的狀態機重現，不另依 Vorbis I 規格處理起始偏移。
//!
//! 只移植 `cmd/fd2` 經 Ebiten 用到的 API：`NewReader`、`Read`、`SetPosition`、`Length`、`Position`、`Channels`、
//! `SampleRate`，以及單元測試用的 `ReadAll`、`GetFormat`、`GetLength`。來源固定是記憶體中的位元組
//! （Go `bytes.Reader`）；`seekable == false` 對應 Go 測試裡包一層、不實作 `io.Seeker` 的讀取器。
//!
//! 原授權（`third_party/jfreymuth-oggvorbis-v1.0.5-LICENSE` 為原檔）：
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

use std::sync::{Arc, OnceLock};

use crate::go_io::{self, BytesReader, IoError, Read, Seek, Whence};
use crate::go_vorbis::{self, Bitrate, VorbisError};

impl From<VorbisError> for IoError {
    fn from(e: VorbisError) -> IoError {
        IoError::Other(e.to_string())
    }
}

/// Go `noEOF`。
fn no_eof(err: IoError) -> IoError {
    if err == IoError::Eof { IoError::UnexpectedEof } else { err }
}

// ---------------- crc.go ----------------

fn crc_table() -> &'static [u32; 256] {
    static TABLE: OnceLock<[u32; 256]> = OnceLock::new();
    TABLE.get_or_init(|| {
        let mut t = [0u32; 256];
        for (i, slot) in t.iter_mut().enumerate() {
            let mut r = (i as u32) << 24;
            for _ in 0..8 {
                if r & 0x8000_0000 != 0 {
                    r = (r << 1) ^ 0x04c1_1db7;
                } else {
                    r <<= 1;
                }
            }
            *slot = r;
        }
        t
    })
}

fn crc_update(mut crc: u32, data: &[u8]) -> u32 {
    let t = crc_table();
    for &b in data {
        crc = (crc << 8) ^ t[usize::from((crc >> 24) as u8 ^ b)];
    }
    crc
}

// ---------------- ogg.go ----------------

const HEADER_FLAG_CONTINUED_PACKET: u8 = 1;
#[allow(dead_code)]
const HEADER_FLAG_BEGINNING_OF_STREAM: u8 = 2;
const HEADER_FLAG_END_OF_STREAM: u8 = 4;
const CAPTURE_PATTERN: [u8; 4] = *b"OggS";
const MAX_PAGE_SIZE: i64 = 27 + 255 + 255 * 255;

/// Go `oggReader` 的 `source`／`seeker`／`buffer`（`oggReader.Read` 先讀 `buffer` 再讀來源）。
struct OggIo {
    source: BytesReader,
    seekable: bool,
    buffer: Vec<u8>,
}

impl Read for OggIo {
    fn read(&mut self, p: &mut [u8]) -> (usize, Option<IoError>) {
        if !self.buffer.is_empty() {
            let n = self.buffer.len().min(p.len());
            p[..n].copy_from_slice(&self.buffer[..n]);
            self.buffer.drain(..n);
            return (n, None);
        }
        self.source.read(p)
    }
}

impl OggIo {
    fn seek(&mut self, offset: i64, whence: Whence) -> Result<i64, IoError> {
        // 只在 Go 的 `r.seeker != nil` 分支呼叫。
        self.source.seek(offset, whence)
    }
}

#[derive(Default, Clone)]
struct Page {
    // pageHeader
    capture_pattern: [u8; 4],
    stream_structure_version: u8,
    header_type_flag: u8,
    absolute_granule_position: i64,
    #[allow(dead_code)]
    stream_serial_number: u32,
    #[allow(dead_code)]
    page_sequence_number: u32,
    page_checksum: u32,
    page_segments: u8,

    header_checksum: u32,
    packet_count: usize,
    packet_sizes: Vec<usize>,
    total_size: usize,
    #[allow(dead_code)]
    needs_continue: bool,
    packets: Vec<Vec<u8>>,
}

impl Page {
    fn is_last(&self) -> bool {
        self.header_type_flag & HEADER_FLAG_END_OF_STREAM != 0
    }

    fn read<R: Read + ?Sized>(&mut self, r: &mut R) -> Result<(), IoError> {
        self.read_header(r)?;
        self.read_content(r)
    }

    fn read_header<R: Read + ?Sized>(&mut self, r: &mut R) -> Result<(), IoError> {
        let mut data = [0u8; 27];
        let (_, err) = go_io::read_full(r, &mut data);
        if let Some(e) = err {
            return Err(e);
        }
        let le32 = |at: usize| u32::from_le_bytes([data[at], data[at + 1], data[at + 2], data[at + 3]]);
        self.capture_pattern = [data[0], data[1], data[2], data[3]];
        self.stream_structure_version = data[4];
        self.header_type_flag = data[5];
        self.absolute_granule_position = i64::from_le_bytes(data[6..14].try_into().unwrap());
        self.stream_serial_number = le32(14);
        self.page_sequence_number = le32(18);
        self.page_checksum = le32(22);
        self.page_segments = data[26];
        if self.capture_pattern != CAPTURE_PATTERN {
            return Err(IoError::other("ogg: missing capture pattern"));
        }
        if self.stream_structure_version != 0 {
            return Err(IoError::other("ogg: unsupported version"));
        }
        let mut segment_table = vec![0u8; usize::from(self.page_segments)];
        let (_, err) = go_io::read_full(r, &mut segment_table);
        if let Some(e) = err {
            return Err(no_eof(e));
        }
        data[22..26].fill(0);
        self.header_checksum = crc_update(0, &data);
        self.header_checksum = crc_update(self.header_checksum, &segment_table);

        let mut size = 0usize;
        self.total_size = 0;
        self.packet_count = 0;
        self.packet_sizes = Vec::new();
        for &s in &segment_table {
            size += usize::from(s);
            self.total_size += usize::from(s);
            if s < 0xFF {
                self.packet_count += 1;
                self.packet_sizes.push(size);
                size = 0;
            }
        }
        // Go：`segmentTable[p.PageSegments-1]`，片段數為 0 時越界 panic。
        self.needs_continue = segment_table[usize::from(self.page_segments.wrapping_sub(1))] == 0xFF;
        Ok(())
    }

    fn read_content<R: Read + ?Sized>(&mut self, r: &mut R) -> Result<(), IoError> {
        let mut content = vec![0u8; self.total_size];
        let (_, err) = go_io::read_full(r, &mut content);
        if let Some(e) = err {
            return Err(no_eof(e));
        }
        let checksum = crc_update(self.header_checksum, &content);
        if checksum != self.page_checksum {
            return Err(IoError::other("ogg: wrong checksum"));
        }
        let mut packets = Vec::with_capacity(self.packet_count + 1);
        let mut offset = 0usize;
        for &size in &self.packet_sizes {
            packets.push(content[offset..offset + size].to_vec());
            offset += size;
        }
        packets.push(content[offset..].to_vec());
        self.packets = packets;
        Ok(())
    }
}

/// Go `oggReader`。
struct OggReader {
    io: OggIo,
    current_page: Page,
    #[allow(dead_code)]
    last_page_position: i64,
    packet_index: usize,
    ready: bool,
    last_packet: bool,
}

impl OggReader {
    fn next_packet(&mut self) -> Result<Vec<u8>, IoError> {
        if !self.ready {
            self.current_page.read(&mut self.io)?;
            self.packet_index = 0;
            if self.current_page.header_type_flag & HEADER_FLAG_CONTINUED_PACKET != 0 {
                self.packet_index = 1;
            }
            self.ready = true;
        }
        if self.packet_index == self.current_page.packet_count {
            let rest = self.current_page.packets[self.current_page.packet_count].clone();
            if self.current_page.absolute_granule_position != -1 {
                self.last_page_position = self.current_page.absolute_granule_position;
            }
            self.current_page.read(&mut self.io)?;
            if !rest.is_empty() {
                let mut joined = rest;
                joined.extend_from_slice(&self.current_page.packets[0]);
                self.current_page.packets[0] = joined;
            }
            self.packet_index = 0;
            return self.next_packet();
        }
        let packet = self.current_page.packets[self.packet_index].clone();
        self.packet_index += 1;
        if self.packet_index == self.current_page.packet_count && self.current_page.is_last() {
            self.last_packet = true;
        }
        Ok(packet)
    }

    fn restore(&mut self) -> Result<(), IoError> {
        let mut b = vec![0u8; 1024];
        loop {
            let (n, err) = go_io::read_at_least(&mut self.io, &mut b, 4);
            if let Some(e) = err {
                return Err(e);
            }
            match b[..n].windows(4).position(|w| w == CAPTURE_PATTERN) {
                None => self.io.buffer = b[n - 3..n].to_vec(),
                Some(i) => {
                    self.io.buffer = b[i..n].to_vec();
                    self.ready = false;
                    return Ok(());
                }
            }
        }
    }

    fn last_position(&mut self) -> Result<i64, IoError> {
        if self.io.seek(-MAX_PAGE_SIZE, Whence::End).is_err() {
            let _ = self.io.seek(0, Whence::Start);
        }
        self.io.buffer.clear();
        self.restore()?;
        let mut p = Page::default();
        let mut result = 0i64;
        loop {
            match p.read_header(&mut self.io) {
                Err(IoError::Eof) => return Ok(result),
                Err(e) => return Err(e),
                Ok(()) => {}
            }
            result = p.absolute_granule_position;
            if p.is_last() {
                return Ok(result);
            }
            let _ = self.io.seek(p.total_size as i64 - self.io.buffer.len() as i64, Whence::Current);
            self.io.buffer.clear();
        }
    }

    fn seek_page_before(&mut self, pos: i64) -> Result<i64, IoError> {
        let _ = self.io.seek(0, Whence::Start);
        self.io.buffer.clear();
        self.last_packet = false;
        let mut p = Page::default();
        let mut last_offset = 0i64;
        let mut last_pos = 0i64;
        loop {
            let offset = self.io.seek(0, Whence::Current).unwrap_or(0);
            p.read_header(&mut self.io)?;
            if p.absolute_granule_position > pos {
                let _ = self.io.seek(last_offset, Whence::Start);
                self.current_page.read(&mut self.io)?;
                if last_pos > 0 {
                    self.packet_index = self.current_page.packet_count.wrapping_sub(1);
                    self.ready = true;
                } else {
                    self.ready = false;
                }
                return Ok(last_pos);
            }
            if p.is_last() {
                return Err(IoError::UnexpectedEof);
            }
            last_offset = offset;
            last_pos = p.absolute_granule_position;
            let _ = self.io.seek(p.total_size as i64 - self.io.buffer.len() as i64, Whence::Current);
            self.io.buffer.clear();
        }
    }
}

// ---------------- reader.go ----------------

/// Go `Reader`。`buffer` 是 `original_buffer[buf_start..buf_end]`（Go 的切片）。
pub struct Reader {
    r: OggReader,
    dec: go_vorbis::Decoder,

    position: i64,
    buf_start: usize,
    buf_end: usize,
    original_buffer: Vec<f32>,
    to_skip: usize,

    length: i64,
}

impl Reader {
    /// Go `NewReader(bytes.NewReader(data))`；`seekable == false` 時如同來源不實作 `io.Seeker`。
    pub fn new(data: Arc<[u8]>, seekable: bool) -> Result<Reader, IoError> {
        let mut r = Reader {
            r: OggReader {
                io: OggIo { source: BytesReader::new(data), seekable, buffer: Vec::new() },
                current_page: Page::default(),
                last_page_position: 0,
                packet_index: 0,
                ready: false,
                last_packet: false,
            },
            dec: go_vorbis::Decoder::new(),
            position: 0,
            buf_start: 0,
            buf_end: 0,
            original_buffer: Vec::new(),
            to_skip: 0,
            length: 0,
        };
        r.init()?;
        Ok(r)
    }

    fn buffer_len(&self) -> usize {
        self.buf_end - self.buf_start
    }

    fn init(&mut self) -> Result<(), IoError> {
        if self.r.io.seekable {
            if let Ok(length) = self.r.last_position() {
                self.length = length;
            }
            self.r.io.buffer.clear();
            let _ = self.r.io.seek(0, Whence::Start);
        }

        // read headers
        for _ in 0..3 {
            let packet = self.r.next_packet().map_err(no_eof)?;
            self.dec.read_header(&packet)?;
        }
        self.original_buffer = vec![0.0; self.dec.buffer_size()];

        // read the first two packets, because the data might not start at position zero, see:
        // https://xiph.org/vorbis/doc/Vorbis_I_spec.html#x1-132000A.2
        self.fill_buffer()?;
        if self.r.packet_index == self.r.current_page.packet_count {
            let offset = self.r.current_page.absolute_granule_position - self.position;
            if offset < 0 {
                // Go：`r.buffer = r.buffer[-offset:]`（以值數而非幀數切）。
                let cut = usize::try_from(-offset).expect("slice bounds out of range");
                assert!(cut <= self.buffer_len(), "slice bounds out of range");
                self.buf_start += cut;
            }
            self.position = self.r.current_page.absolute_granule_position;
        }
        Ok(())
    }

    pub fn sample_rate(&self) -> i64 {
        self.dec.sample_rate()
    }

    pub fn channels(&self) -> usize {
        self.dec.channels()
    }

    pub fn bitrate(&self) -> Bitrate {
        self.dec.bitrate
    }

    /// 設定標頭裡各 floor 的型別號（診斷用）。
    pub fn floor_types(&self) -> Vec<u16> {
        self.dec.floor_types()
    }

    /// Go `Position()`。
    pub fn position(&self) -> i64 {
        let ch = self.channels() as i64;
        self.position - self.buffer_len() as i64 / ch + self.to_skip as i64 / ch
    }

    /// Go `Length()`：樣本（幀）數，0 表示未知。
    pub fn length(&self) -> i64 {
        self.length
    }

    /// Go `SetPosition`。
    pub fn set_position(&mut self, pos: i64) -> Result<(), IoError> {
        if !self.r.io.seekable {
            return Err(IoError::other("oggvorbis: reader is not seekable"));
        }
        self.buf_start = 0;
        self.buf_end = 0;
        if pos >= self.length {
            self.r.last_packet = true;
            self.position = self.length;
            return Ok(());
        }
        let start = self.r.seek_page_before(pos)?;
        // Go 忽略這兩個呼叫的錯誤。
        let packet = self.r.next_packet().unwrap_or_default();
        self.dec.clear();
        let mut ob = std::mem::take(&mut self.original_buffer);
        let _ = self.dec.decode_into(&packet, &mut ob);
        self.original_buffer = ob;
        self.position = start;
        self.to_skip = usize::try_from((pos - start) * self.channels() as i64).expect("negative skip");
        Ok(())
    }

    /// Go `Read(p []float32)`：回傳解出的值數（幀數 × 聲道數）與錯誤。
    #[allow(clippy::manual_clamp, clippy::collapsible_if)]
    pub fn read(&mut self, p: &mut [f32]) -> (usize, Option<IoError>) {
        if self.r.last_packet && self.buffer_len() == 0 {
            return (0, Some(IoError::Eof));
        }
        if p.is_empty() {
            return (0, None);
        }
        let ch = self.channels();
        let p_len = p.len() / ch * ch;
        let out = &mut p[..p_len];
        let mut total = 0usize;
        let mut err: Option<IoError> = None;
        'end: {
            if self.to_skip > 0 {
                if let Err(e) = self.skip() {
                    return (0, Some(e));
                }
            }
            if self.buffer_len() > 0 {
                let n = self.buffer_len().min(out.len() - total);
                out[total..total + n].copy_from_slice(&self.original_buffer[self.buf_start..self.buf_start + n]);
                self.buf_start += n;
                total += n;
            }
            while out.len() - total >= self.original_buffer.len() {
                let (n, e) = self.read_packet(Some(&mut out[total..]));
                total = add_signed(total, n);
                if let Some(e) = e {
                    err = Some(e);
                    break 'end;
                }
            }
            if total == 0 {
                if let Err(e) = self.fill_buffer() {
                    err = Some(e);
                    break 'end;
                }
                let n = self.buffer_len().min(out.len() - total);
                out[total..total + n].copy_from_slice(&self.original_buffer[self.buf_start..self.buf_start + n]);
                self.buf_start += n;
                total += n;
            }
        }
        for v in &mut out[..total] {
            if *v > 1.0 {
                *v = 1.0;
            } else if *v < -1.0 {
                *v = -1.0;
            }
        }
        (total, err)
    }

    fn fill_buffer(&mut self) -> Result<(), IoError> {
        loop {
            let (n, err) = self.read_packet(None);
            let n = usize::try_from(n).expect("slice bounds out of range");
            self.buf_start = 0;
            self.buf_end = n;
            if let Some(e) = err {
                return Err(e);
            }
            if n != 0 {
                return Ok(());
            }
        }
    }

    /// Go `read(p)`；`p == None` 表示解進 `originalBuffer`。回傳值數可以是負的（最後一個封包要丟掉的比解出的多），
    /// 與 Go 相同，由呼叫端在切片時中止。
    fn read_packet(&mut self, p: Option<&mut [f32]>) -> (isize, Option<IoError>) {
        let packet = match self.r.next_packet() {
            Ok(packet) => packet,
            Err(e) => return (0, Some(e)),
        };
        let decoded = match p {
            Some(p) => self.dec.decode_into(&packet, p),
            None => {
                let mut ob = std::mem::take(&mut self.original_buffer);
                let res = self.dec.decode_into(&packet, &mut ob);
                self.original_buffer = ob;
                res
            }
        };
        let ch = self.channels() as i64;
        let (mut n, err) = match decoded {
            Ok(n) => (n as isize, None),
            Err(e) => (0, Some(IoError::from(e))),
        };
        self.position += n as i64 / ch;
        if err.is_some() {
            return (n, err);
        }
        if self.r.last_packet {
            let discard = (self.position - self.r.current_page.absolute_granule_position) * ch;
            if discard > 0 {
                n -= discard as isize;
            }
            self.position = self.r.current_page.absolute_granule_position;
            self.length = self.position;
        }
        (n, None)
    }

    fn skip(&mut self) -> Result<(), IoError> {
        loop {
            let err = self.fill_buffer().err();
            if self.buffer_len() > self.to_skip {
                self.buf_start += self.to_skip;
                self.to_skip = 0;
                return match err {
                    Some(e) => Err(e),
                    None => Ok(()),
                };
            }
            self.to_skip -= self.buffer_len();
            if let Some(e) = err {
                return Err(e);
            }
        }
    }
}

/// Go `total += n` 後 `p = p[n:]`：`n` 為負時 Go 在切片處 panic。
fn add_signed(total: usize, n: isize) -> usize {
    usize::try_from(total as isize + n).ok().filter(|_| n >= 0).expect("slice bounds out of range")
}

// ---------------- util.go ----------------

/// Go `Format`。
#[derive(Debug, Clone, Copy, PartialEq, Eq)]
pub struct Format {
    pub sample_rate: i64,
    pub channels: usize,
    pub bitrate: Bitrate,
}

/// Go `ReadAll`。
pub fn read_all(data: Arc<[u8]>, seekable: bool) -> Result<(Vec<f32>, Format), IoError> {
    let mut r = Reader::new(data, seekable)?;
    let format = Format { sample_rate: r.sample_rate(), channels: r.channels(), bitrate: r.bitrate() };
    if r.length() > 0 {
        let size = usize::try_from((r.length() - r.position()) * r.channels() as i64).expect("makeslice");
        let mut result = vec![0f32; size];
        let mut read = 0usize;
        loop {
            let (n, err) = r.read(&mut result[read..]);
            read += n;
            if err.is_some() || read == result.len() {
                result.truncate(read);
                return match err {
                    None | Some(IoError::Eof) => Ok((result, format)),
                    Some(e) => Err(e),
                };
            }
            if n == 0 {
                return Err(IoError::NoProgress);
            }
        }
    }
    let mut buf = vec![0f32; r.dec.buffer_size()];
    let mut result = Vec::new();
    loop {
        let (n, err) = r.read(&mut buf);
        result.extend_from_slice(&buf[..n]);
        match err {
            Some(IoError::Eof) => return Ok((result, format)),
            Some(e) => return Err(e),
            None => {}
        }
        if n == 0 {
            return Err(IoError::NoProgress);
        }
    }
}

/// Go `GetFormat`：只讀第一頁。
pub fn get_format(data: Arc<[u8]>) -> Result<Format, IoError> {
    let mut dec = go_vorbis::Decoder::new();
    let mut r = OggReader {
        io: OggIo { source: BytesReader::new(data), seekable: false, buffer: Vec::new() },
        current_page: Page::default(),
        last_page_position: 0,
        packet_index: 0,
        ready: false,
        last_packet: false,
    };
    let p = r.next_packet()?;
    dec.read_header(&p)?;
    Ok(Format { sample_rate: dec.sample_rate(), channels: dec.channels(), bitrate: dec.bitrate })
}

/// Go `GetLength`：最後一頁的 granule 與格式。
pub fn get_length(data: Arc<[u8]>) -> Result<(i64, Format), IoError> {
    let mut dec = go_vorbis::Decoder::new();
    let mut r = OggReader {
        io: OggIo { source: BytesReader::new(data), seekable: true, buffer: Vec::new() },
        current_page: Page::default(),
        last_page_position: 0,
        packet_index: 0,
        ready: false,
        last_packet: false,
    };
    let p = r.next_packet()?;
    dec.read_header(&p)?;
    let format = Format { sample_rate: dec.sample_rate(), channels: dec.channels(), bitrate: dec.bitrate };
    let length = r.last_position()?;
    Ok((length, format))
}

#[cfg(test)]
#[path = "go_oggvorbis/go_oggvorbis_test.rs"]
mod go_oggvorbis_test;
