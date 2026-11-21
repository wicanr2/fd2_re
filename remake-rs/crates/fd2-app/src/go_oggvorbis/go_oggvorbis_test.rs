//! jfreymuth/oggvorbis v1.0.5 的 `reader_test.go` 逐條移植。測試檔是該套件的 `testdata/`（MIT，複本在
//! `third_party/testdata/`）。Go 以「包一層、不實作 `io.Seeker`」測非定位來源，這裡以 `seekable == false` 表示。

use super::*;

const TEST_OGG: &[u8] = include_bytes!("../../third_party/testdata/test.ogg");
const TEST_RAW: &[u8] = include_bytes!("../../third_party/testdata/test.raw");
const LONG_OGG: &[u8] = include_bytes!("../../third_party/testdata/long.ogg");
const EOF_ISSUE_OGG: &[u8] = include_bytes!("../../third_party/testdata/eof_issue.ogg");

fn arc(b: &[u8]) -> Arc<[u8]> {
    Arc::from(b)
}

/// Go `equal`（照抄）。
fn equal(a: f32, b: f32, tolerance: f32) -> bool {
    (a > b && a - b < tolerance) || b - a < tolerance
}

fn compare(a: &[f32], e: &[f32], tolerance: f32) {
    assert_eq!(a.len(), e.len(), "length of output is {}, expected {}", a.len(), e.len());
    for (i, (&x, &y)) in a.iter().zip(e).enumerate() {
        assert!(equal(x, y, tolerance), "different values at index {i} ({x} != {y})");
    }
}

fn test(data: &[u8], reference: &[f32], tolerance: f32, wrap: bool) {
    let (dec, _) = read_all(arc(data), !wrap).expect("decoding error");
    compare(&dec, reference, tolerance);
}

#[test]
fn read() {
    let raw: Vec<f32> = TEST_RAW.as_chunks::<4>().0.iter().map(|c| f32::from_le_bytes(*c)).collect();
    test(TEST_OGG, &raw, 0.00002, false);
    test(TEST_OGG, &raw, 0.00002, true);
}

#[test]
fn format() {
    let format = get_format(arc(TEST_OGG)).unwrap();
    assert_eq!(format.sample_rate, 44100);
    assert_eq!(format.channels, 1);
}

#[test]
fn length() {
    let (length, _) = get_length(arc(TEST_OGG)).unwrap();
    assert_eq!(length, 44100);
}

#[test]
fn length_for_unexpected_eof() {
    const EXPECTED_LENGTH: i64 = 72384;
    let (length, _) = get_length(arc(EOF_ISSUE_OGG)).unwrap();
    assert_eq!(length, EXPECTED_LENGTH);
    let r = Reader::new(arc(EOF_ISSUE_OGG), true).unwrap();
    assert_eq!(r.length(), EXPECTED_LENGTH);
}

#[test]
fn seek() {
    let (data, _) = read_all(arc(LONG_OGG), true).unwrap();
    let mut r = Reader::new(arc(LONG_OGG), true).unwrap();
    assert_eq!(r.length() as usize, data.len(), "length is {}, expected {}", r.length(), data.len());

    let sections: [(i64, i64); 8] = [
        (600000, 10000),
        // test first position
        (0, 500),
        // test last position
        (881999, 1),
        // test around page boundary
        (182399, 100),
        (182400, 100),
        (182401, 100),
        // seek to an out of bounds position and then continue
        (1000000, 0),
        (200000, 1000),
    ];
    for (start, len) in sections {
        r.set_position(start).unwrap();
        if len == 0 {
            continue;
        }
        let reference = &data[start as usize..(start + len) as usize];
        let mut buf = vec![0f32; len as usize];
        let mut n = 0;
        while n < buf.len() {
            let (read, err) = r.read(&mut buf[n..]);
            assert!(err.is_none(), "{err:?}");
            n += read;
        }
        compare(&buf, reference, 0.00001);
    }

    r.set_position(882000).unwrap();
    let (_, err) = r.read(&mut []);
    assert_eq!(err, Some(IoError::Eof), "error should be EOF, but is {err:?}");
}

#[test]
fn empty() {
    let err = Reader::new(arc(&[]), true).err();
    assert_eq!(err, Some(IoError::UnexpectedEof), "error should be unexpected EOF, but is {err:?}");
}

// ---------------- Rust 新增 ----------------

/// 非定位來源不能 `SetPosition`（Go 的錯誤文字）。
#[test]
fn set_position_requires_seeker() {
    let mut r = Reader::new(arc(TEST_OGG), false).unwrap();
    assert_eq!(r.length(), 0);
    assert_eq!(r.set_position(0), Err(IoError::other("oggvorbis: reader is not seekable")));
}

/// 頁的 CRC 錯誤照 Go 回報 `ogg: wrong checksum`。
#[test]
fn wrong_checksum() {
    let mut bad = TEST_OGG.to_vec();
    bad[22] ^= 1;
    let err = Reader::new(arc(&bad), true).err();
    assert_eq!(err, Some(IoError::other("ogg: wrong checksum")));
}

/// 每次讀的長度不影響樣本序列（`go_ebiten_audio` 模組文件的前提）。例外是長度小於聲道數：Go 把 `p` 截成
/// 聲道數的倍數後為空，仍會 `fillBuffer` 而丟掉緩衝；Ebiten 的呼叫端不會傳這種長度，這裡也不測。
#[test]
fn read_sizes_do_not_change_samples() {
    let (whole, _) = read_all(arc(LONG_OGG), true).unwrap();
    for size in [2usize, 3, 7, 64, 1000, 4096, 100_000] {
        let mut r = Reader::new(arc(LONG_OGG), true).unwrap();
        let mut got = Vec::new();
        let mut buf = vec![0f32; size];
        loop {
            let (n, err) = r.read(&mut buf);
            got.extend_from_slice(&buf[..n]);
            if err.is_some() {
                break;
            }
        }
        assert_eq!(got.len(), whole.len(), "size {size}");
        assert!(got.iter().zip(&whole).all(|(a, b)| a.to_bits() == b.to_bits()), "size {size}");
    }
}
