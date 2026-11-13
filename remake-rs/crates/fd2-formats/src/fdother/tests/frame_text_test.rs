//! 移植自 `frame_text_test.go`。

use crate::fdother::*;
use std::sync::{Arc, Mutex};

#[test]
fn text_placeholder_validates_and_preserves_destination() {
    let mut source = Frame::separated(1, 2, 2, 2, vec![8, 9, 10, 11], vec![255, 255, 255, 255]);
    let seen = Arc::new(Mutex::new((-1i64, -1i64)));
    let sink = Arc::clone(&seen);
    let placeholder = source.text_placeholder(move |px, py| *sink.lock().unwrap() = (px, py)).unwrap();
    let mut dst = vec![77u8; 160];
    placeholder.blit_at(&mut dst, 10, 23, -1).unwrap();
    assert!(
        *seen.lock().unwrap() == (4, 4) && dst == vec![77u8; 160],
        "placeholder changed source coordinates or background"
    );
    assert_eq!(source.indexed, [8, 9, 10, 11], "placeholder mutated source");
    source.mask[0] = 33;
    assert!(source.text_placeholder(|_, _| {}).is_err(), "invalid source mask accepted");
}
