//! 移植自 `separated_frame_bank_test.go`。四組「與原版相同」與四組「缺素材即拒收」測試的
//! 本體相同，以共用函式表達。

use super::{TempDir, original, pack};
use crate::fdother::separated_frame_bank::{
    NATIVE_CH20_SKY_KEY_FRAME_COUNT, NATIVE_ENDING_PREFIX_FRAME_COUNT, NATIVE_EVENT61_FRAME_COUNT,
    NATIVE_PENDING_CODE1_FRAME_COUNT,
};
use crate::fdother::*;

fn matches_original(rel: &str, resource: i64, count: Option<i64>, load: fn(&std::path::Path) -> Result<Vec<Frame>>) {
    let Some(archive) = original("FDOTHER.DAT") else { return };
    let Some(root) = pack(rel, "bank.json") else { return };
    let want = decode_resource(&archive, resource).unwrap();
    let got = load(&root).unwrap();
    assert!(
        got.len() == want.len() && count.is_none_or(|c| got.len() as i64 == c),
        "frame count={} want={}",
        got.len(),
        want.len()
    );
    for (index, (g, w)) in got.iter().zip(&want).enumerate() {
        let (want_indexed, want_mask) = w.indexed_layers().unwrap();
        assert!(
            g.x == w.x
                && g.y == w.y
                && g.width == w.width
                && g.height == w.height
                && g.indexed == want_indexed
                && g.mask == want_mask,
            "{rel} frame {index} differs"
        );
    }
}

fn fails_closed(load: fn(&std::path::Path) -> Result<Vec<Frame>>) {
    let dir = TempDir::new("frame-bank");
    assert!(load(dir.path()).is_err(), "incomplete pack was accepted");
}

#[test]
fn separated_event61_frames_match_original() {
    // Go 版只比對長度與原版相同；常數另外確認。
    assert_eq!(NATIVE_EVENT61_FRAME_COUNT, 59);
    matches_original("animations/fdother_045_event61", 45, None, |r| load_separated_event61_frames(r));
}

#[test]
fn separated_event61_frames_fail_closed_without_pack() {
    fails_closed(|r| load_separated_event61_frames(r));
}

#[test]
fn separated_ending_prefix_frames_match_original() {
    matches_original("animations/fdother_054_ending_prefix", 54, Some(NATIVE_ENDING_PREFIX_FRAME_COUNT), |r| {
        load_separated_ending_prefix_frames(r)
    });
}

#[test]
fn separated_ending_prefix_frames_fail_closed_without_pack() {
    fails_closed(|r| load_separated_ending_prefix_frames(r));
}

#[test]
fn separated_ch20_sky_key_frames_match_original() {
    matches_original("animations/fdother_034_ch20_sky_key", 34, Some(NATIVE_CH20_SKY_KEY_FRAME_COUNT), |r| {
        load_separated_ch20_sky_key_frames(r)
    });
}

#[test]
fn separated_ch20_sky_key_frames_fail_closed_without_pack() {
    fails_closed(|r| load_separated_ch20_sky_key_frames(r));
}

#[test]
fn separated_pending_code1_frames_match_original() {
    matches_original("animations/fdother_079_pending_code1", 79, Some(NATIVE_PENDING_CODE1_FRAME_COUNT), |r| {
        load_separated_pending_code1_frames(r)
    });
}

#[test]
fn separated_pending_code1_frames_fail_closed_without_pack() {
    fails_closed(|r| load_separated_pending_code1_frames(r));
}
