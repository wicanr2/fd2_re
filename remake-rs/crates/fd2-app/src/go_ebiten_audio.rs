//! Ebiten v2.6.6 `audio/vorbis`、`audio/wav` 與 `audio/internal/convert` 的逐行移植（R5 階段 4，代號 r5v；
//! `rust-port-no-go-stdlib` 的具名例外）。
//!
//! Go 版 `cmd/fd2` 在 `audio.NewContext(44100)` 下以 `vorbis.DecodeWithSampleRate`、`wav.DecodeWithSampleRate`
//! 解碼，結果是 44.1 kHz、雙聲道、16 位元小端的位元組串流。這條鏈有幾個會改變樣本的細節，都照 Go 重現：
//!
//! - 浮點轉 16 位元發生兩次：oggvorbis 的浮點樣本（已夾到 ±1）先以 `int16(f * 32767)` 轉成整數
//!   （[`convert`] 的 `f32Reader`），重取樣時再除以 32767 回到 `f64`，算完夾到 ±1 後再 `int16(x * 32767)`。
//! - 重取樣（[`convert::Resampling`]）是 ±8 個來源樣本的加窗 sinc，cos 以 65536 格查表（`fastCos01`）。
//!   查表用 `fd2_formats::go_math::cos`（Go `math.Cos`）建，讓各平台得到同一張表。
//! - 長度：OGG 以 oggvorbis 的 `Length()`（最後一頁的 granule）× 聲道 × 2 定總位元組數，解出的樣本不足時
//!   重取樣器補零；所以 BGM 循環點有 Go 的靜音（使用者 2026-11-16 定案保留）。WAV 以 `data` 宣告長度定長。
//! - `decode` 先讀 65536 位元組再 `Seek(0)`，讓 oggvorbis 經 `SetPosition(0)` 從 0 重新計數。
//!
//! 讀取大小不影響輸出：`Resampling` 每次以 16384 位元組的固定區塊向來源取樣本，輸出幀只依幀號計算；
//! oggvorbis 的樣本序列與每次要求的長度無關。所以呼叫端可以用任意大小讀（`go_io::read_all` 固定 16 KiB），
//! 結果與 Go 的 `io.ReadAll` 及 `audio.NewInfiniteLoop` 第一圈相同。`InfiniteLoop` 每圈結尾 `Seek(0)` 後
//! 重新經 `SetPosition(0)` 解碼，與第一圈逐位元組相同（`tools/rust_r5o_audio_pcm.sh` 以 Go 的兩圈雜湊核對），
//! 外殼因此以 Bevy 的 `PlaybackSettings::LOOP` 重播同一串樣本，不移植 `InfiniteLoop`。
//!
//! 與 Go 的刻意差異：取樣率為 0 時 Go 的 `Resampling.Length()` 是 `int64(+Inf)`，讀取時位置變負而中止；
//! 這裡在建立串流時回報錯誤 `sample rate is zero`（Rust 新增的守門，素材不會遇到）。
//!
//! 原授權（`third_party/ebiten-v2.6.6-LICENSE` 為 Apache License 2.0 全文）。本檔是 Go 原始碼
//! `audio/vorbis/vorbis.go`、`audio/wav/decode.go`、`audio/internal/convert/{resampling,float32,stereo16}.go`
//! 的 Rust 改寫，改動是語言轉換、把 `io.ReadSeeker` 換成 [`crate::go_io`] 的契約型別，以及上述取樣率守門：
//!
//! ```text
//! Copyright 2015 Hajime Hoshi
//!
//! Licensed under the Apache License, Version 2.0 (the "License");
//! you may not use this file except in compliance with the License.
//! You may obtain a copy of the License at
//!
//!     http://www.apache.org/licenses/LICENSE-2.0
//!
//! Unless required by applicable law or agreed to in writing, software
//! distributed under the License is distributed on an "AS IS" BASIS,
//! WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
//! See the License for the specific language governing permissions and
//! limitations under the License.
//! ```

use crate::go_io::{IoError, ReadSeek, Seek, Whence};

/// Ebiten 的 `Stream`（`vorbis.Stream`、`wav.Stream`）：解碼後的位元組串流與總長度。
pub struct Stream {
    inner: Box<dyn ReadSeek>,
    size: i64,
}

impl Stream {
    /// Go `Length()`：解碼後的總位元組數。
    pub fn length(&self) -> i64 {
        self.size
    }
}

impl crate::go_io::Read for Stream {
    fn read(&mut self, p: &mut [u8]) -> (usize, Option<IoError>) {
        self.inner.read(p)
    }
}

impl Seek for Stream {
    fn seek(&mut self, offset: i64, whence: Whence) -> Result<i64, IoError> {
        self.inner.seek(offset, whence)
    }
}

/// Go 在 amd64 的 `int16(float)`：先截成 32 位元整數（`CVTTSS2SL`／`CVTTSD2SL`，超出範圍與 NaN 得
/// `0x80000000`），再取低 16 位元。呼叫端的值都已夾到 ±32767，這裡只是把轉換寫明。
fn go_float_to_i16(x: f64) -> i16 {
    let i = if x.is_nan() || x >= 2_147_483_648.0 || x < -2_147_483_648.0 { i32::MIN } else { x as i32 };
    i as i16
}

pub mod convert {
    //! Ebiten v2.6.6 `audio/internal/convert`：`Resampling`、`NewReaderFromFloat32Reader`、`Stereo16`。

    use std::collections::BTreeMap;
    use std::sync::OnceLock;

    use fd2_formats::go_math;

    use super::go_float_to_i16;
    use crate::go_io::{IoError, Read, ReadSeek, Seek, Whence};

    // ---------------- resampling.go ----------------

    fn cos_table() -> &'static [f64] {
        static TABLE: OnceLock<Vec<f64>> = OnceLock::new();
        TABLE.get_or_init(|| {
            (0..65536).map(|i| go_math::cos(f64::from(i) * std::f64::consts::PI / 2.0 / 65536.0)).collect()
        })
    }

    pub(crate) fn fast_cos01(x: f64) -> f64 {
        let x = x.abs();
        let cos_table = cos_table();
        let len = cos_table.len() as i64;
        let mut i = (4.0 * len as f64 * x) as i64;
        if 4 * len < i {
            i %= 4 * len;
        }
        let mut sign = 1.0;
        if i < len {
        } else if i < len * 2 {
            i = len * 2 - i;
            sign = -1.0;
        } else if i < len * 3 {
            i -= len * 2;
            sign = -1.0;
        } else {
            i = len * 4 - i;
        }
        if i == len {
            return 0.0;
        }
        sign * cos_table[i as usize]
    }

    fn fast_sin01(x: f64) -> f64 {
        fast_cos01(x - 0.25)
    }

    fn sinc01(x: f64) -> f64 {
        if x.abs() < 1e-8 {
            return 1.0;
        }
        fast_sin01(x) / (x * 2.0 * std::f64::consts::PI)
    }

    const RESAMPLING_BUFFER_SIZE: i64 = 4096;

    /// Go `Resampling`。
    pub struct Resampling {
        source: Box<dyn ReadSeek>,
        size: i64,
        from: i64,
        to: i64,
        pos: i64,
        src_block: i64,
        src_buf: BTreeMap<i64, (Vec<f64>, Vec<f64>)>,
        lru_src_blocks: Vec<i64>,
    }

    impl Resampling {
        /// Go `NewResampling(source, size, from, to)`。
        pub fn new(source: Box<dyn ReadSeek>, size: i64, from: i64, to: i64) -> Resampling {
            Resampling {
                source,
                size,
                from,
                to,
                pos: 0,
                src_block: -1,
                src_buf: BTreeMap::new(),
                lru_src_blocks: Vec::new(),
            }
        }

        /// Go `Length()`。
        pub fn length(&self) -> i64 {
            let s = (self.size as f64 * self.to as f64 / self.from as f64) as i64;
            s / 4 * 4
        }

        fn src(&mut self, i: i64) -> Result<(f64, f64), IoError> {
            if i < 0 {
                return Ok((0.0, 0.0));
            }
            if self.size / 4 <= i {
                return Ok((0.0, 0.0));
            }
            let next_pos = i / RESAMPLING_BUFFER_SIZE;
            if !self.src_buf.contains_key(&next_pos) {
                if self.src_block + 1 != next_pos {
                    self.source.seek(next_pos * RESAMPLING_BUFFER_SIZE * 4, Whence::Start)?;
                }
                let mut buf = vec![0u8; (RESAMPLING_BUFFER_SIZE * 4) as usize];
                let mut c = 0usize;
                while c < buf.len() {
                    let (n, err) = self.source.read(&mut buf[c..]);
                    c += n;
                    match err {
                        None => {}
                        Some(IoError::Eof) => break,
                        Some(e) => return Err(e),
                    }
                }
                buf.truncate(c);
                let mut sl = vec![0f64; RESAMPLING_BUFFER_SIZE as usize];
                let mut sr = vec![0f64; RESAMPLING_BUFFER_SIZE as usize];
                for k in 0..buf.len() / 4 {
                    sl[k] = f64::from(i16::from_le_bytes([buf[4 * k], buf[4 * k + 1]])) / 32767.0;
                    sr[k] = f64::from(i16::from_le_bytes([buf[4 * k + 2], buf[4 * k + 3]])) / 32767.0;
                }
                self.src_block = next_pos;
                self.src_buf.insert(self.src_block, (sl, sr));
                // To keep srcBufL/R not too big, let's remove the least used buffers.
                if self.lru_src_blocks.len() >= 4 {
                    let p = self.lru_src_blocks.remove(0);
                    self.src_buf.remove(&p);
                }
                self.lru_src_blocks.push(self.src_block);
            } else {
                self.src_block = next_pos;
                let idx = self.lru_src_blocks.iter().position(|&p| p == self.src_block).expect("not reach");
                self.lru_src_blocks.remove(idx);
                self.lru_src_blocks.push(self.src_block);
            }
            let ii = (i % RESAMPLING_BUFFER_SIZE) as usize;
            let (l, r) = &self.src_buf[&self.src_block];
            Ok((l[ii], r[ii]))
        }

        // 夾值照 Go 的 if 寫法（NaN 原樣保留）。
        #[allow(clippy::manual_clamp)]
        fn at(&mut self, t: i64) -> Result<(f64, f64), IoError> {
            let window_size = 8.0;
            let t_in_src = t as f64 * self.from as f64 / self.to as f64;
            let mut start_n = (t_in_src - window_size) as i64;
            if start_n < 0 {
                start_n = 0;
            }
            if self.size / 4 <= start_n {
                start_n = self.size / 4 - 1;
            }
            let mut end_n = (t_in_src + window_size) as i64;
            if self.size / 4 <= end_n {
                end_n = self.size / 4 - 1;
            }
            let (mut lv, mut rv) = (0.0f64, 0.0f64);
            for n in start_n..=end_n {
                let (src_l, src_r) = self.src(n)?;
                let d = t_in_src - n as f64;
                let w = 0.5 + 0.5 * fast_cos01(d / (window_size * 2.0 + 1.0));
                let s = sinc01(d / 2.0) * w;
                lv += src_l * s;
                rv += src_r * s;
            }
            if lv < -1.0 {
                lv = -1.0;
            }
            if lv > 1.0 {
                lv = 1.0;
            }
            if rv < -1.0 {
                rv = -1.0;
            }
            if rv > 1.0 {
                rv = 1.0;
            }
            Ok((lv, rv))
        }
    }

    impl Read for Resampling {
        fn read(&mut self, b: &mut [u8]) -> (usize, Option<IoError>) {
            if self.pos == self.length() {
                return (0, Some(IoError::Eof));
            }
            let mut n = (b.len() / 4 * 4) as i64;
            if self.length() - self.pos <= n {
                n = self.length() - self.pos;
            }
            for i in 0..(n / 4).max(0) {
                let (l, r) = match self.at(self.pos / 4 + i) {
                    Ok(v) => v,
                    Err(e) => return (0, Some(e)),
                };
                let l16 = go_float_to_i16(l * 32767.0);
                let r16 = go_float_to_i16(r * 32767.0);
                let k = 4 * i as usize;
                b[k..k + 2].copy_from_slice(&l16.to_le_bytes());
                b[k + 2..k + 4].copy_from_slice(&r16.to_le_bytes());
            }
            self.pos += n;
            // Go 回傳 `int(n)`；`n` 只在取樣率異常時為負，`Resampling` 的建立端已擋下（見模組文件）。
            (usize::try_from(n).expect("negative read"), None)
        }
    }

    impl Seek for Resampling {
        fn seek(&mut self, offset: i64, whence: Whence) -> Result<i64, IoError> {
            match whence {
                Whence::Start => self.pos = offset,
                Whence::Current => self.pos += offset,
                Whence::End => self.pos += self.length() + offset,
            }
            if self.pos < 0 {
                self.pos = 0;
            }
            if self.length() <= self.pos {
                self.pos = self.length();
            }
            Ok(self.pos)
        }
    }

    // ---------------- float32.go ----------------

    /// Go `Float32Reader`。
    pub trait Float32Reader: Send {
        fn read_f32(&mut self, p: &mut [f32]) -> (usize, Option<IoError>);
    }

    impl Float32Reader for crate::go_oggvorbis::Reader {
        fn read_f32(&mut self, p: &mut [f32]) -> (usize, Option<IoError>) {
            self.read(p)
        }
    }

    /// Go `f32Reader`（`NewReaderFromFloat32Reader`）的狀態；來源由呼叫端每次傳入（Go 的 `decoded` 同時持有
    /// 來源與這個讀取器，兩者指向同一個 `*oggvorbis.Reader`）。
    #[derive(Default)]
    pub struct F32Reader {
        eof: bool,
        has_remain: bool,
        remain: u8,
        fbuf: Vec<f32>,
    }

    impl F32Reader {
        pub fn read<R: Float32Reader + ?Sized>(&mut self, r: &mut R, buf: &mut [u8]) -> (usize, Option<IoError>) {
            if self.eof {
                return (0, Some(IoError::Eof));
            }
            if buf.is_empty() {
                return (0, None);
            }
            if self.has_remain {
                buf[0] = self.remain;
                self.has_remain = false;
                return (1, None);
            }

            let l = (buf.len() / 2).max(1);
            if self.fbuf.len() < l {
                self.fbuf = vec![0.0; l];
            }

            let (n, err) = r.read_f32(&mut self.fbuf[..l]);
            let err = match err {
                None => None,
                Some(IoError::Eof) => {
                    self.eof = true;
                    Some(IoError::Eof)
                }
                Some(e) => return (0, Some(e)),
            };

            let mut tmp = [0u8; 2];
            let use_tmp = buf.len() == 1 && n > 0;
            let b: &mut [u8] = if use_tmp { &mut tmp } else { &mut *buf };
            for i in 0..n {
                let f = self.fbuf[i];
                let s = go_float_to_i16(f64::from(f * 32767.0));
                b[2 * i] = s as u8;
                b[2 * i + 1] = (s >> 8) as u8;
            }

            if use_tmp {
                buf[0] = tmp[0];
                self.remain = tmp[1];
                self.has_remain = true;
                return (1, err);
            }
            (n * 2, err)
        }
    }

    // ---------------- stereo16.go ----------------

    /// Go `Stereo16`。
    pub struct Stereo16 {
        source: Box<dyn ReadSeek>,
        mono: bool,
        eight: bool,
        buf: Vec<u8>,
    }

    impl Stereo16 {
        pub fn new(source: Box<dyn ReadSeek>, mono: bool, eight: bool) -> Stereo16 {
            Stereo16 { source, mono, eight, buf: Vec::new() }
        }
    }

    impl Read for Stereo16 {
        fn read(&mut self, b: &mut [u8]) -> (usize, Option<IoError>) {
            let mut l = b.len();
            if self.mono {
                l /= 2;
            }
            if self.eight {
                l /= 2;
            }
            if self.buf.len() < l {
                self.buf = vec![0; l];
            }
            let (mut n, err) = self.source.read(&mut self.buf[..l]);
            if let Some(e) = &err
                && *e != IoError::Eof
            {
                return (0, err);
            }
            let s = &self.buf;
            match (self.mono, self.eight) {
                (true, true) => {
                    for i in 0..n {
                        let v = (i32::from(s[i]) * 0x101 - (1 << 15)) as i16;
                        b[4 * i] = v as u8;
                        b[4 * i + 1] = (v >> 8) as u8;
                        b[4 * i + 2] = v as u8;
                        b[4 * i + 3] = (v >> 8) as u8;
                    }
                }
                (true, false) => {
                    for i in 0..n / 2 {
                        b[4 * i] = s[2 * i];
                        b[4 * i + 1] = s[2 * i + 1];
                        b[4 * i + 2] = s[2 * i];
                        b[4 * i + 3] = s[2 * i + 1];
                    }
                }
                (false, true) => {
                    for i in 0..n / 2 {
                        let v0 = (i32::from(s[2 * i]) * 0x101 - (1 << 15)) as i16;
                        let v1 = (i32::from(s[2 * i + 1]) * 0x101 - (1 << 15)) as i16;
                        b[4 * i] = v0 as u8;
                        b[4 * i + 1] = (v0 >> 8) as u8;
                        b[4 * i + 2] = v1 as u8;
                        b[4 * i + 3] = (v1 >> 8) as u8;
                    }
                }
                (false, false) => {}
            }
            if self.mono {
                n *= 2;
            }
            if self.eight {
                n *= 2;
            }
            (n, err)
        }
    }

    impl Seek for Stereo16 {
        fn seek(&mut self, mut offset: i64, whence: Whence) -> Result<i64, IoError> {
            if self.mono {
                offset /= 2;
            }
            if self.eight {
                offset /= 2;
            }
            self.source.seek(offset, whence)
        }
    }
}

pub mod vorbis {
    //! Ebiten v2.6.6 `audio/vorbis`。

    use std::sync::Arc;

    use super::Stream;
    use super::convert::{F32Reader, Resampling, Stereo16};
    use crate::go_io::{IoError, Read, Seek, Whence};
    use crate::go_oggvorbis;

    /// Go `decoded`。
    pub struct Decoded {
        total_bytes: i64,
        pos_in_bytes: i64,
        decoder: go_oggvorbis::Reader,
        decoderr: Option<F32Reader>,
    }

    impl Decoded {
        /// 診斷用：設定標頭的 floor 型別號。
        pub fn floor_types(&self) -> Vec<u16> {
            self.decoder.floor_types()
        }
    }

    impl Read for Decoded {
        fn read(&mut self, b: &mut [u8]) -> (usize, Option<IoError>) {
            let decoderr = self.decoderr.get_or_insert_with(F32Reader::default);
            let mut l = self.total_bytes - self.pos_in_bytes;
            if l > b.len() as i64 {
                l = b.len() as i64;
            }
            if l < 0 {
                return (0, Some(IoError::Eof));
            }
            let l = l as usize;
            let (n, err) = loop {
                let (n, err) = decoderr.read(&mut self.decoder, &mut b[..l]);
                if let Some(e) = &err
                    && *e != IoError::Eof
                {
                    return (0, err);
                }
                if n == 0 && l > 0 && err != Some(IoError::Eof) {
                    // When l is too small, decoder's Read might return 0 for a while. Let's retry.
                    continue;
                }
                break (n, err);
            };
            self.pos_in_bytes += n as i64;
            if self.pos_in_bytes == self.total_bytes || err == Some(IoError::Eof) {
                return (n, Some(IoError::Eof));
            }
            (n, None)
        }
    }

    impl Seek for Decoded {
        fn seek(&mut self, offset: i64, whence: Whence) -> Result<i64, IoError> {
            let mut next = match whence {
                Whence::Start => offset,
                Whence::Current => self.pos_in_bytes + offset,
                Whence::End => self.total_bytes + offset,
            };
            // pos should be always even
            next = next / 2 * 2;
            self.pos_in_bytes = next;
            self.decoder.set_position(next / self.decoder.channels() as i64 / 2)?;
            self.decoderr = None;
            Ok(next)
        }
    }

    /// Go `decode(in)`：回傳 `decoded`、聲道數與原始取樣率。`seekable` 對應 Go 的 `in.(io.Seeker)`。
    fn decode(src: Arc<[u8]>, seekable: bool) -> Result<(Decoded, usize, i64), IoError> {
        let r = go_oggvorbis::Reader::new(src, seekable)?;
        let channels = r.channels();
        let sample_rate = r.sample_rate();
        let mut d =
            Decoded { total_bytes: r.length() * channels as i64 * 2, pos_in_bytes: 0, decoder: r, decoderr: None };
        if seekable {
            let mut probe = vec![0u8; 65536];
            if let (_, Some(e)) = d.read(&mut probe)
                && e != IoError::Eof
            {
                return Err(e);
            }
            d.seek(0, Whence::Start)?;
        }
        Ok((d, channels, sample_rate))
    }

    fn channel_error(n: usize) -> IoError {
        IoError::Other(format!("vorbis: number of channels must be 1 or 2 but was {n}"))
    }

    /// Go `DecodeWithoutResampling(bytes.NewReader(src))`。
    pub fn decode_without_resampling(src: Arc<[u8]>) -> Result<Stream, IoError> {
        decode_without_resampling_with(src, true)
    }

    /// 同上；`seekable == false` 對應 Go 測試的非 `io.Seeker` 來源。
    pub fn decode_without_resampling_with(src: Arc<[u8]>, seekable: bool) -> Result<Stream, IoError> {
        let (decoded, channels, _) = decode(src, seekable)?;
        if channels != 1 && channels != 2 {
            return Err(channel_error(channels));
        }
        let mut size = decoded.total_bytes;
        let inner: Box<dyn crate::go_io::ReadSeek> = if channels == 1 {
            size *= 2;
            Box::new(Stereo16::new(Box::new(decoded), true, false))
        } else {
            Box::new(decoded)
        };
        Ok(Stream { inner, size })
    }

    /// Go `DecodeWithSampleRate(sampleRate, bytes.NewReader(src))`。
    pub fn decode_with_sample_rate(sample_rate: i64, src: Arc<[u8]>) -> Result<Stream, IoError> {
        decode_with_sample_rate_with(sample_rate, src, true)
    }

    /// 同上；`seekable == false` 對應 Go 測試的非 `io.Seeker` 來源。
    pub fn decode_with_sample_rate_with(sample_rate: i64, src: Arc<[u8]>, seekable: bool) -> Result<Stream, IoError> {
        let (decoded, channels, orig_sample_rate) = decode(src, seekable)?;
        if channels != 1 && channels != 2 {
            return Err(channel_error(channels));
        }
        let mut size = decoded.total_bytes;
        let mut inner: Box<dyn crate::go_io::ReadSeek> = if channels == 1 {
            size *= 2;
            Box::new(Stereo16::new(Box::new(decoded), true, false))
        } else {
            Box::new(decoded)
        };
        if orig_sample_rate != sample_rate {
            if orig_sample_rate == 0 {
                return Err(IoError::other("sample rate is zero"));
            }
            let r = Resampling::new(inner, size, orig_sample_rate, sample_rate);
            size = r.length();
            inner = Box::new(r);
        }
        Ok(Stream { inner, size })
    }

    /// 只解標頭與前 65536 位元組時的 floor 型別（診斷用）。
    pub fn floor_types(src: Arc<[u8]>) -> Result<Vec<u16>, IoError> {
        Ok(decode(src, true)?.0.floor_types())
    }
}

pub mod wav {
    //! Ebiten v2.6.6 `audio/wav`。

    use std::sync::Arc;

    use super::Stream;
    use super::convert::{Resampling, Stereo16};
    use crate::go_io::{self, BytesReader, IoError, Read, ReadSeek, Seek, Whence};

    /// Go `stream`。
    struct WavStream {
        src: BytesReader,
        header_size: i64,
        data_size: i64,
        remaining: i64,
    }

    impl Read for WavStream {
        fn read(&mut self, p: &mut [u8]) -> (usize, Option<IoError>) {
            if self.remaining <= 0 {
                return (0, Some(IoError::Eof));
            }
            let len = if self.remaining < p.len() as i64 { self.remaining as usize } else { p.len() };
            let (n, err) = self.src.read(&mut p[..len]);
            self.remaining -= n as i64;
            (n, err)
        }
    }

    impl Seek for WavStream {
        fn seek(&mut self, mut offset: i64, mut whence: Whence) -> Result<i64, IoError> {
            match whence {
                Whence::Start => offset += self.header_size,
                Whence::Current => {}
                Whence::End => {
                    offset += self.header_size + self.data_size;
                    whence = Whence::Start;
                }
            }
            let n = self.src.seek(offset, whence)?;
            if n - self.header_size < 0 {
                return Err(IoError::other("wav: invalid offset"));
            }
            self.remaining = self.data_size - (n - self.header_size);
            // There could be a tail in wav file.
            if self.remaining < 0 {
                self.remaining = 0;
                return Ok(self.data_size);
            }
            Ok(n - self.header_size)
        }
    }

    fn invalid_header() -> IoError {
        IoError::other("wav: invalid header")
    }

    /// Go `decode(src)`：回傳串流與原始取樣率。
    fn decode(src: Arc<[u8]>) -> Result<(Stream, i64), IoError> {
        let mut src = BytesReader::new(src);
        let mut buf = [0u8; 12];
        let (n, err) = go_io::read_full(&mut src, &mut buf);
        if n != buf.len() {
            return Err(invalid_header());
        }
        if let Some(e) = err {
            return Err(e);
        }
        if &buf[0..4] != b"RIFF" {
            return Err(IoError::other("wav: invalid header: 'RIFF' not found"));
        }
        if &buf[8..12] != b"WAVE" {
            return Err(IoError::other("wav: invalid header: 'WAVE' not found"));
        }

        // Read chunks
        let mut data_size: i64;
        let mut header_size = buf.len() as i64;
        let mut mono = false;
        let mut bits_per_sample = 0i64;
        let mut sample_rate = 0i64;
        loop {
            let mut buf = [0u8; 8];
            let (n, err) = go_io::read_full(&mut src, &mut buf);
            if n != buf.len() {
                return Err(invalid_header());
            }
            if let Some(e) = err {
                return Err(e);
            }
            header_size += 8;
            let size = i64::from(u32::from_le_bytes([buf[4], buf[5], buf[6], buf[7]]));
            match &buf[0..4] {
                b"fmt " => {
                    // Size of 'fmt' header is usually 16, but can be more than 16.
                    if size < 16 {
                        return Err(IoError::other("wav: invalid header: maybe non-PCM file?"));
                    }
                    let mut buf = vec![0u8; size as usize];
                    let (n, err) = go_io::read_full(&mut src, &mut buf);
                    if n != buf.len() {
                        return Err(invalid_header());
                    }
                    if let Some(e) = err {
                        return Err(e);
                    }
                    let le16 = |at: usize| i64::from(u16::from_le_bytes([buf[at], buf[at + 1]]));
                    if le16(0) != 1 {
                        return Err(IoError::other("wav: format must be linear PCM"));
                    }
                    let channel_count = le16(2);
                    match channel_count {
                        1 => mono = true,
                        2 => mono = false,
                        _ => {
                            return Err(IoError::Other(format!(
                                "wav: number of channels must be 1 or 2 but was {channel_count}"
                            )));
                        }
                    }
                    bits_per_sample = le16(14);
                    if bits_per_sample != 8 && bits_per_sample != 16 {
                        return Err(IoError::Other(format!(
                            "wav: bits per sample must be 8 or 16 but was {bits_per_sample}"
                        )));
                    }
                    sample_rate = i64::from(u32::from_le_bytes([buf[4], buf[5], buf[6], buf[7]]));
                    header_size += size;
                }
                b"data" => {
                    data_size = size;
                    break;
                }
                _ => {
                    let mut buf = vec![0u8; size as usize];
                    let (n, err) = go_io::read_full(&mut src, &mut buf);
                    if n != buf.len() {
                        return Err(invalid_header());
                    }
                    if let Some(e) = err {
                        return Err(e);
                    }
                    header_size += size;
                }
            }
        }
        let stream = WavStream { src, header_size, data_size, remaining: data_size };
        let mut inner: Box<dyn ReadSeek> = Box::new(stream);
        if mono || bits_per_sample != 16 {
            inner = Box::new(Stereo16::new(inner, mono, bits_per_sample != 16));
            if mono {
                data_size *= 2;
            }
            if bits_per_sample != 16 {
                data_size *= 2;
            }
        }
        Ok((Stream { inner, size: data_size }, sample_rate))
    }

    /// Go `DecodeWithoutResampling(bytes.NewReader(src))`。
    pub fn decode_without_resampling(src: Arc<[u8]>) -> Result<Stream, IoError> {
        Ok(decode(src)?.0)
    }

    /// Go `DecodeWithSampleRate(sampleRate, bytes.NewReader(src))`。
    pub fn decode_with_sample_rate(sample_rate: i64, src: Arc<[u8]>) -> Result<Stream, IoError> {
        let (s, orig_sample_rate) = decode(src)?;
        if sample_rate == orig_sample_rate {
            return Ok(s);
        }
        if orig_sample_rate == 0 {
            return Err(IoError::other("sample rate is zero"));
        }
        let r = Resampling::new(s.inner, s.size, orig_sample_rate, sample_rate);
        let size = r.length();
        Ok(Stream { inner: Box::new(r), size })
    }
}

#[cfg(test)]
#[path = "go_ebiten_audio/go_ebiten_audio_test.rs"]
pub(crate) mod go_ebiten_audio_test;
