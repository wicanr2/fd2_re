//! jfreymuth/vorbis v1.0.2 的 `decode_test.go`、`imdct_test.go` 移植，另加 Rust 新增的單元測試。
//!
//! `TestDecode` 原本讀 `testdata/test.gob`（Go `encoding/gob` 存的標頭與封包）；gob 是 Go 專用格式，這裡改從
//! 同一段音訊的 `oggvorbis/testdata/test.ogg` 拆出封包（兩者的參考輸出 `test.raw` 逐位元組相同），
//! 比對方式與容差照 Go。

use super::codebook::{BitReader, float32_unpack, ilog, lookup1_values, shl32};
use super::imdct::{generate_imdct_lookup, imdct};
use super::inverse_db_table::INVERSE_DB_TABLE;
use super::*;

const TEST_OGG: &[u8] = include_bytes!("../../third_party/testdata/test.ogg");
const TEST_RAW: &[u8] = include_bytes!("../../third_party/testdata/test.raw");

/// 依 Ogg 頁的片段表拆出封包（只供測試；正式路徑在 `go_oggvorbis`）。
fn ogg_packets(data: &[u8]) -> Vec<Vec<u8>> {
    let (mut pos, mut packets, mut cur) = (0usize, Vec::new(), Vec::new());
    while pos + 27 <= data.len() {
        assert_eq!(&data[pos..pos + 4], b"OggS");
        let segments = usize::from(data[pos + 26]);
        let table = &data[pos + 27..pos + 27 + segments];
        let mut body = pos + 27 + segments;
        for &s in table {
            cur.extend_from_slice(&data[body..body + usize::from(s)]);
            body += usize::from(s);
            if s < 255 {
                packets.push(std::mem::take(&mut cur));
            }
        }
        pos = body;
    }
    packets
}

fn reference() -> Vec<f32> {
    TEST_RAW.as_chunks::<4>().0.iter().map(|c| f32::from_le_bytes(*c)).collect()
}

/// Go `equal(a, b, tolerance)`（照抄，含其不對稱的寫法）。
fn equal(a: f32, b: f32, tolerance: f32) -> bool {
    (a > b && a - b < tolerance) || b - a < tolerance
}

#[test]
fn decode() {
    let packets = ogg_packets(TEST_OGG);
    let mut dec = Decoder::new();
    for header in &packets[..3] {
        dec.read_header(header).unwrap();
    }
    assert_eq!(dec.sample_rate(), 44100);
    assert_eq!(dec.channels(), 1);

    let reference = reference();
    let mut pos = 0usize;
    let mut out = vec![0f32; dec.buffer_size()];
    for packet in &packets[3..] {
        let mut n = dec.decode_into(packet, &mut out).unwrap();
        if pos + n > reference.len() {
            // it's ok for the decoded data to be longer, since only an integral number of packets can be decoded
            n = reference.len() - pos;
        }
        for i in 0..n {
            assert!(
                equal(out[i], reference[pos + i], 0.00002),
                "different values at sample {i} ({} != {})",
                out[i],
                reference[pos + i]
            );
        }
        pos += n;
    }
    assert!(pos >= reference.len(), "not enough samples were decoded ({pos}, expected at least {})", reference.len());
}

/// Go `imdctSlow`。
fn imdct_slow(input: &[f32]) -> Vec<f32> {
    let n = input.len();
    let fn_ = n as f32;
    let mut out = vec![0f32; 2 * n];
    for (i, o) in out.iter_mut().enumerate() {
        let mut sum = 0f32;
        let fi = i as f32;
        for (k, &x) in input.iter().enumerate() {
            let fk = k as f32;
            let arg = (std::f32::consts::PI / fn_) * (fi + 0.5 + fn_ / 2.0) * (fk + 0.5);
            sum += x * fd2_formats::go_math::cos(f64::from(arg)) as f32;
        }
        *o = sum;
    }
    out
}

/// Go `TestIMDCT`：Go 用未設種子的 `rand.Float32()`；這裡用固定的線性同餘序列產生 [0, 1) 的值，容差照 Go。
#[test]
fn imdct_matches_slow_reference() {
    const BLOCKSIZE: usize = 256;
    let mut seed: u64 = 0x2545_F491_4F6C_DD1D;
    let mut data: Vec<f32> = (0..BLOCKSIZE / 2)
        .map(|_| {
            seed = seed.wrapping_mul(6364136223846793005).wrapping_add(1442695040888963407);
            (seed >> 40) as f32 / (1u64 << 24) as f32
        })
        .collect();
    let reference = imdct_slow(&data);
    let lookup = generate_imdct_lookup(BLOCKSIZE);
    let mut result = vec![0f32; BLOCKSIZE];
    imdct(&lookup, &mut data, &mut result);
    for (i, (&r, &e)) in result.iter().zip(&reference).enumerate() {
        assert!(r - e < 1.002 && e - r < 1.002, "different values at index {i} ({r} != {e})");
    }
}

// ---------------- Rust 新增 ----------------

#[test]
fn inverse_db_table_endpoints() {
    assert_eq!(INVERSE_DB_TABLE.len(), 256);
    assert_eq!(INVERSE_DB_TABLE[0], 1.0649863e-07);
    assert_eq!(INVERSE_DB_TABLE[255], 1.0);
    assert!(INVERSE_DB_TABLE.windows(2).all(|w| w[0] < w[1]));
}

#[test]
fn bit_reader_reads_lsb_first_and_eof_returns_zero() {
    let data = [0b1010_1100u8, 0xFF];
    let mut r = BitReader::new(&data);
    assert_eq!(r.read8(2), 0b00);
    assert_eq!(r.read8(3), 0b011);
    assert_eq!(r.read16(5), 0b11101);
    assert_eq!(r.read32(6), 0b111111);
    assert!(!r.eof());
    assert_eq!(r.read32(1), 0);
    assert!(r.eof());
    let mut r = BitReader::new(&[0x78, 0x56, 0x34, 0x12, 0xFF]);
    assert_eq!(r.read32(32), 0x1234_5678);
}

/// Go 的 `uint32` 位移超過寬度得 0（`0xFFFFFFFF << 32` 在 `read` 的遮罩裡出現）。
#[test]
fn go_shift_semantics() {
    assert_eq!(shl32(1, 31), 0x8000_0000);
    assert_eq!(shl32(1, 32), 0);
    assert_eq!(shl32(0xFFFF_FFFF, 40), 0);
}

#[test]
fn ilog_matches_go() {
    for (x, want) in [(-1, 0), (0, 0), (1, 1), (2, 2), (3, 2), (4, 3), (7, 3), (8, 4), (255, 8)] {
        assert_eq!(ilog(x), want, "{x}");
    }
}

#[test]
fn lookup1_values_small_cases() {
    assert_eq!(lookup1_values(81, 2), 9);
    assert_eq!(lookup1_values(80, 2), 8);
    assert_eq!(lookup1_values(1, 4), 1);
    assert_eq!(lookup1_values(16, 1), 16);
    // Go 的 math.Pow 在 dim ≥ 3 的完全次方少 1（glibc 的 pow 不會）。
    assert_eq!(lookup1_values(27, 3), 2);
    assert_eq!(lookup1_values(28, 3), 3);
    assert_eq!(lookup1_values(81, 4), 3);
    assert_eq!(lookup1_values(4096, 4), 7);
    assert_eq!(lookup1_values(4095, 4), 7);
}

/// Vorbis I 規格 9.2.2 的範例與正負號、零。
#[test]
fn float32_unpack_values() {
    // 尾數 1、指數 788 → 1.0。
    assert_eq!(float32_unpack(1 | (788 << 21)), 1.0);
    assert_eq!(float32_unpack(3 | (787 << 21)), 1.5);
    assert_eq!(float32_unpack(0x8000_0000 | 1 | (788 << 21)), -1.0);
    assert_eq!(float32_unpack(0).to_bits(), 0);
    assert_eq!(float32_unpack(0x8000_0000).to_bits(), (-0.0f32).to_bits());
}

#[test]
fn header_errors_follow_go_messages() {
    let mut dec = Decoder::new();
    assert_eq!(dec.read_header(b"\x01vorbi"), Err(VorbisError::InvalidHeader));
    assert_eq!(dec.read_header(b"\x07vorbis...."), Err(VorbisError::UnknownHeaderType));
    assert_eq!(dec.read_header(b"\x01vorbis\x00\x00"), Err(VorbisError::Decoding));
    let mut buf = [0f32; 8];
    assert_eq!(dec.decode_into(&[0], &mut buf), Err(VorbisError::MissingHeaders));
    assert_eq!(VorbisError::Decoding.to_string(), "vorbis: decoding error");
}

/// 測試向量只用 floor1。
#[test]
fn test_vector_uses_floor1() {
    let packets = ogg_packets(TEST_OGG);
    let mut dec = Decoder::new();
    for header in &packets[..3] {
        dec.read_header(header).unwrap();
    }
    assert!(dec.floor_types().iter().all(|&t| t == 1));
}
