//! Ebiten v2.6.6 `audio/internal/convert`（`resampling_test.go`、`float32_test.go`）與 `audio/vorbis`
//! （`vorbis_test.go`）的逐條移植，另加 Rust 新增的 WAV 與循環測試。測試檔 `test_mono.ogg`、`test_tooshort.ogg`
//! 是 Ebiten 的原檔（Apache 2.0，複本在 `third_party/testdata/`）。

use std::sync::Arc;

use fd2_formats::go_math;

use super::convert::{F32Reader, Float32Reader, Resampling};
use super::*;
use crate::go_io::{self, BytesReader, Read};
use crate::go_oggvorbis;

const TEST_MONO_OGG: &[u8] = include_bytes!("../../third_party/testdata/test_mono.ogg");
const TEST_TOOSHORT_OGG: &[u8] = include_bytes!("../../third_party/testdata/test_tooshort.ogg");

// ---------------- resampling_test.go ----------------

fn sound_at(time_in_second: f64) -> f64 {
    const FREQ: f64 = 220.0;
    let amp = [1.0, 0.8, 0.6, 0.4, 0.2];
    let mut v = 0.0;
    for (j, a) in amp.iter().enumerate() {
        v += a * go_math::sin(2.0 * std::f64::consts::PI * time_in_second * FREQ * (j + 1) as f64);
    }
    v.clamp(-1.0, 1.0)
}

fn new_sound_bytes(sample_rate: usize) -> Vec<u8> {
    let mut b = vec![0u8; sample_rate * 4]; // 1 second
    for i in 0..b.len() / 4 {
        let v = sound_at(i as f64 / sample_rate as f64);
        let v16 = (v * 32767.0) as i16;
        b[4 * i..4 * i + 2].copy_from_slice(&v16.to_le_bytes());
        b[4 * i + 2..4 * i + 4].copy_from_slice(&v16.to_le_bytes());
    }
    b
}

#[test]
fn resampling() {
    for (from, to) in [(44100usize, 48000usize), (48000, 44100)] {
        let in_b = new_sound_bytes(from);
        let size = in_b.len() as i64;
        let mut out_s = Resampling::new(Box::new(BytesReader::new(Arc::from(in_b))), size, from as i64, to as i64);
        let (got_b, err) = go_io::read_all(&mut out_s);
        assert!(err.is_none(), "{err:?}");
        let want_b = new_sound_bytes(to);
        assert_eq!(got_b.len(), want_b.len(), "len(gotB) == {} but len(wantB) == {}", got_b.len(), want_b.len());
        for i in 0..got_b.len() / 2 {
            let got = f64::from(i16::from_le_bytes([got_b[2 * i], got_b[2 * i + 1]])) / 32767.0;
            let want = f64::from(i16::from_le_bytes([want_b[2 * i], want_b[2 * i + 1]])) / 32767.0;
            assert!((got - want).abs() <= 0.025, "sample rate: {to}, index: {i}: got: {got}, want: {want}");
        }
    }
}

// ---------------- float32_test.go ----------------

struct VecF32Reader {
    data: Vec<f32>,
    pos: usize,
}

impl Float32Reader for VecF32Reader {
    fn read_f32(&mut self, buf: &mut [f32]) -> (usize, Option<go_io::IoError>) {
        if self.pos == self.data.len() {
            return (0, Some(go_io::IoError::Eof));
        }
        let n = buf.len().min(self.data.len() - self.pos);
        buf[..n].copy_from_slice(&self.data[self.pos..self.pos + n]);
        self.pos += n;
        (n, None)
    }
}

#[test]
fn float32_reader() {
    let in1: Vec<f32> = (0..256).map(|i| go_math::sin(f64::from(i)) as f32).collect();
    let in2: Vec<f32> = (0..65536).map(|i| go_math::cos(f64::from(i)) as f32).collect();
    let cases = [(&in1, 1usize), (&in1, 2), (&in1, 3), (&in1, 1024), (&in2, 1), (&in2, 4096)];
    for (i, (input, n)) in cases.into_iter().enumerate() {
        let mut src = VecF32Reader { data: input.clone(), pos: 0 };
        let mut r = F32Reader::default();
        let mut got = Vec::new();
        loop {
            let mut buf = vec![0u8; n];
            let (read, err) = r.read(&mut src, &mut buf);
            if let Some(e) = err {
                if read == 0 && e == go_io::IoError::Eof {
                    break;
                }
                panic!("{e}");
            }
            got.extend_from_slice(&buf[..read]);
        }
        let mut want = vec![0u8; input.len() * 2];
        for (k, &f) in input.iter().enumerate() {
            let s = (f * 32767.0) as i16;
            want[2 * k..2 * k + 2].copy_from_slice(&s.to_le_bytes());
        }
        assert_eq!(got, want, "case: {i}");
    }
}

// ---------------- vorbis_test.go ----------------

#[test]
fn mono() {
    let s = vorbis::decode_with_sample_rate(44100, Arc::from(TEST_MONO_OGG)).unwrap();
    let r = go_oggvorbis::Reader::new(Arc::from(TEST_MONO_OGG), true).unwrap();
    // Stream decoded by audio/vorbis.DecodeWithSampleRate() is always 16bit stereo.
    let got = s.length();
    // On the other hand, the original vorbis package is monoral.
    // As Length() represents the number of samples,
    // this needs to be doubled by 2 (= bytes in 16bits).
    let want = r.length() * 2 * 2;
    assert_eq!(got, want, "s.Length(): got: {got}, want: {want}");
}

#[test]
fn too_short() {
    let s = vorbis::decode_with_sample_rate(44100, Arc::from(TEST_TOOSHORT_OGG)).unwrap();
    assert_eq!(s.length(), 79424);
}

#[test]
fn non_seeker() {
    let s = vorbis::decode_with_sample_rate_with(44100, Arc::from(TEST_TOOSHORT_OGG), false).unwrap();
    assert_eq!(s.length(), 0);
}

// ---------------- Rust 新增 ----------------

/// 8 位元單聲道、取樣率 `rate` 的最小 WAV。
pub(crate) fn wav8(rate: u32, data: &[u8]) -> Vec<u8> {
    let mut out = Vec::new();
    out.extend_from_slice(b"RIFF");
    out.extend_from_slice(&(36 + data.len() as u32).to_le_bytes());
    out.extend_from_slice(b"WAVEfmt ");
    out.extend_from_slice(&16u32.to_le_bytes());
    out.extend_from_slice(&1u16.to_le_bytes());
    out.extend_from_slice(&1u16.to_le_bytes());
    out.extend_from_slice(&rate.to_le_bytes());
    out.extend_from_slice(&rate.to_le_bytes());
    out.extend_from_slice(&1u16.to_le_bytes());
    out.extend_from_slice(&8u16.to_le_bytes());
    out.extend_from_slice(b"data");
    out.extend_from_slice(&(data.len() as u32).to_le_bytes());
    out.extend_from_slice(data);
    out
}

fn read_all_i16(mut s: Stream) -> Vec<i16> {
    let (b, err) = go_io::read_all(&mut s);
    assert!(err.is_none(), "{err:?}");
    b.as_chunks::<2>().0.iter().map(|c| i16::from_le_bytes(*c)).collect()
}

/// 8 位元無號樣本照 `convert.Stereo16` 轉成 `int16(int(b)*0x101 - 32768)`，單聲道複製成雙聲道；44.1 kHz 不重取樣。
#[test]
fn wav8_mono_44k_is_stereo16_without_resampling() {
    let s = wav::decode_with_sample_rate(44100, Arc::from(wav8(44100, &[0, 128, 255]))).unwrap();
    assert_eq!(s.length(), 12);
    assert_eq!(read_all_i16(s), [-32768, -32768, 128, 128, 32767, 32767]);
}

#[test]
fn wav_header_errors_follow_go_messages() {
    let err = |b: &[u8]| wav::decode_without_resampling(Arc::from(b)).err().unwrap().to_string();
    let mut bad = wav8(44100, &[1]);
    bad[8..12].copy_from_slice(b"WAVX");
    assert_eq!(err(&bad), "wav: invalid header: 'WAVE' not found");
    let mut float = wav8(44100, &[1]);
    float[20] = 3; // 格式 3（IEEE float）
    assert_eq!(err(&float), "wav: format must be linear PCM");
    let mut ch = wav8(44100, &[1]);
    ch[22] = 3;
    assert_eq!(err(&ch), "wav: number of channels must be 1 or 2 but was 3");
    let mut bits = wav8(44100, &[1]);
    bits[34] = 24;
    assert_eq!(err(&bits), "wav: bits per sample must be 8 or 16 but was 24");
    assert_eq!(err(&wav8(44100, &[])[..30]), "wav: invalid header");
    assert_eq!(err(b"RIFF"), "wav: invalid header");
}

/// 11025 Hz 的 WAV 以 Ebiten 的 sinc 重取樣成 44.1 kHz：長度是 `data` 宣告長度 × 4（單聲道、8 位元）× 4
/// （取樣率），常數訊號在窗完全落在訊號內的中段維持原值。
#[test]
fn wav8_11k_resamples_to_44k_stereo() {
    let s = wav::decode_with_sample_rate(44100, Arc::from(wav8(11025, &[200; 100]))).unwrap();
    assert_eq!(s.length(), 100 * 4 * 4);
    let pcm = read_all_i16(s);
    assert_eq!(pcm.len(), 100 * 4 * 2);
    let src = f64::from((200i32 * 0x101 - 32768) as i16) / 32767.0;
    let mid = pcm[400];
    assert!((f64::from(mid) / 32767.0 - src).abs() < 0.01, "{mid}");
}

/// Go `audio.NewInfiniteLoop` 每圈結尾 `Seek(0)`：重取樣串流的第二圈與第一圈逐位元組相同。
#[test]
fn resampled_stream_second_pass_after_seek_is_identical() {
    let mut s = vorbis::decode_with_sample_rate(44100, Arc::from(TEST_MONO_OGG)).unwrap();
    let (first, err) = go_io::read_all(&mut s);
    assert!(err.is_none());
    assert_eq!(first.len() as i64, s.length());
    crate::go_io::Seek::seek(&mut s, 0, crate::go_io::Whence::Start).unwrap();
    let (second, err) = go_io::read_all(&mut s);
    assert!(err.is_none());
    assert_eq!(first, second);
}

/// 不同的讀取大小得到相同的位元組（`audio.rs` 的串流與 `decode_pcm` 依此相同）。
#[test]
fn read_sizes_do_not_change_resampled_bytes() {
    let whole = {
        let mut s = vorbis::decode_with_sample_rate(44100, Arc::from(TEST_MONO_OGG)).unwrap();
        go_io::read_all(&mut s).0
    };
    for size in [4usize, 6, 512, 4100, 65536] {
        let mut s = vorbis::decode_with_sample_rate(44100, Arc::from(TEST_MONO_OGG)).unwrap();
        let mut got = Vec::new();
        let mut buf = vec![0u8; size];
        loop {
            let (n, err) = s.read(&mut buf);
            got.extend_from_slice(&buf[..n]);
            if err.is_some() {
                break;
            }
        }
        assert_eq!(got, whole, "size {size}");
    }
}

/// `go_float_to_i16`：範圍內向零取整；超出 32 位元與 NaN 照 amd64 得 0x80000000 的低 16 位元（0）。
#[test]
fn go_float_to_i16_matches_amd64() {
    assert_eq!(go_float_to_i16(32767.0), 32767);
    assert_eq!(go_float_to_i16(-32767.9), -32767);
    assert_eq!(go_float_to_i16(0.99), 0);
    assert_eq!(go_float_to_i16(40000.0), (40000i32) as i16);
    assert_eq!(go_float_to_i16(f64::NAN), 0);
    assert_eq!(go_float_to_i16(1e20), 0);
}
