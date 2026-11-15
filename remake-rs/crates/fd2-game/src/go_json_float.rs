//! Go `encoding/json` 的浮點數寫法（`rust-port-no-go-stdlib` 的第五項具名例外）。
//!
//! Go 的 `json.Marshal` 寫 `float64` 時用 `strconv.AppendFloat(f, 'f', -1, 64)`：最短可還原的位數、不用
//! 指數，所以整數值寫成 `0`、`12`；絕對值小於 `1e-6` 或不小於 `1e21` 時改用 `'e'` 格式，並把 `e-07`
//! 這類兩位數負指數清成 `e-7`。serde_json 的預設寫法在整數值後面補 `.0`，存檔裡的 `battle.Unit`
//! （`OffX`、`OffY`、`Exp`）因此和 Go 的位元組不同（#319）。
//!
//! 這裡只換掉 serde_json `Formatter` 的 `write_f64`／`write_f32`，位數仍由 Rust 標準函式庫的 `Display`
//! 與 `LowerExp` 算（兩者都輸出最短可還原表示，和 Go 的 `-1` 精度同一個定義），其餘的縮排與標點照
//! 包住的 `PrettyFormatter`。非有限值 serde_json 在呼叫 `write_f64` 前就寫成 `null`，Go 則回錯誤；這個
//! 差異已列在存檔的 `KNOWN_DIFFERENCES`（非有限小數）。

// 行為對照 Go 1.22.12 src/encoding/json/encode.go 的 floatEncoder.encode。
// 原始碼授權：
// Copyright 2010 The Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

use std::io;

use serde_json::ser::{Formatter, PrettyFormatter};

/// 縮排同 `PrettyFormatter`、浮點數照 Go `encoding/json` 寫出的 serde_json `Formatter`。
pub(crate) struct GoJsonFormatter<'a> {
    inner: PrettyFormatter<'a>,
}

impl<'a> GoJsonFormatter<'a> {
    /// 以 `indent` 縮排，對應 Go `json.MarshalIndent(v, "", indent)`。
    pub(crate) fn with_indent(indent: &'a [u8]) -> Self {
        Self { inner: PrettyFormatter::with_indent(indent) }
    }
}

/// Go `floatEncoder.encode` 的文字：`small` 與 `large` 是改用指數格式的界線（`float32` 與 `float64` 各自比較）。
fn go_float_text(abs_is_zero: bool, use_exp: bool, display: String, lower_exp: String) -> String {
    if abs_is_zero || !use_exp {
        return display;
    }
    // Rust `LowerExp` 寫 `1e21`、`1.5e-7`；Go `'e'` 寫 `1e+21`、`1.5e-07`，再把兩位數負指數的前導 0 去掉。
    // 指數格式只用在絕對值 < 1e-6（指數 ≤ -7）或 ≥ 1e21（指數 ≥ 21），所以正指數至少兩位、負指數清完不補零。
    match lower_exp.split_once('e') {
        Some((mantissa, exp)) if !exp.starts_with('-') => format!("{mantissa}e+{exp}"),
        _ => lower_exp,
    }
}

/// Go `json.Marshal` 寫 `float64` 的文字。
pub(crate) fn go_json_float64(v: f64) -> String {
    let abs = v.abs();
    go_float_text(abs == 0.0, !(1e-6..1e21).contains(&abs), format!("{v}"), format!("{v:e}"))
}

/// Go `json.Marshal` 寫 `float32` 的文字（界線以 `float32` 比較）。
pub(crate) fn go_json_float32(v: f32) -> String {
    let abs = v.abs();
    go_float_text(abs == 0.0, !(1e-6f32..1e21f32).contains(&abs), format!("{v}"), format!("{v:e}"))
}

impl Formatter for GoJsonFormatter<'_> {
    fn write_f64<W: ?Sized + io::Write>(&mut self, w: &mut W, value: f64) -> io::Result<()> {
        w.write_all(go_json_float64(value).as_bytes())
    }

    fn write_f32<W: ?Sized + io::Write>(&mut self, w: &mut W, value: f32) -> io::Result<()> {
        w.write_all(go_json_float32(value).as_bytes())
    }

    // 以下照 PrettyFormatter（serde_json 1.0.154 覆寫的九個方法）。
    fn begin_array<W: ?Sized + io::Write>(&mut self, w: &mut W) -> io::Result<()> {
        self.inner.begin_array(w)
    }

    fn end_array<W: ?Sized + io::Write>(&mut self, w: &mut W) -> io::Result<()> {
        self.inner.end_array(w)
    }

    fn begin_array_value<W: ?Sized + io::Write>(&mut self, w: &mut W, first: bool) -> io::Result<()> {
        self.inner.begin_array_value(w, first)
    }

    fn end_array_value<W: ?Sized + io::Write>(&mut self, w: &mut W) -> io::Result<()> {
        self.inner.end_array_value(w)
    }

    fn begin_object<W: ?Sized + io::Write>(&mut self, w: &mut W) -> io::Result<()> {
        self.inner.begin_object(w)
    }

    fn end_object<W: ?Sized + io::Write>(&mut self, w: &mut W) -> io::Result<()> {
        self.inner.end_object(w)
    }

    fn begin_object_key<W: ?Sized + io::Write>(&mut self, w: &mut W, first: bool) -> io::Result<()> {
        self.inner.begin_object_key(w, first)
    }

    fn begin_object_value<W: ?Sized + io::Write>(&mut self, w: &mut W) -> io::Result<()> {
        self.inner.begin_object_value(w)
    }

    fn end_object_value<W: ?Sized + io::Write>(&mut self, w: &mut W) -> io::Result<()> {
        self.inner.end_object_value(w)
    }
}

#[cfg(test)]
mod tests {
    use super::*;

    /// 期望值由 Go 1.22.12 `json.Marshal` 寫出（`fd2-go-test-local` 容器，2026-11-15）。
    #[test]
    fn float_text_matches_go_encoding_json() {
        let cases: &[(f64, &str)] = &[
            (0.0, "0"),
            (-0.0, "-0"),
            (1.0, "1"),
            (-12.0, "-12"),
            (0.1, "0.1"),
            (13.333333333333334, "13.333333333333334"),
            (1e-6, "0.000001"),
            (9.99e-7, "9.99e-7"),
            (1e-7, "1e-7"),
            (1.5e-10, "1.5e-10"),
            (1e20, "100000000000000000000"),
            (1e21, "1e+21"),
            (1.2345e100, "1.2345e+100"),
            (5e-324, "5e-324"),
            (f64::MAX, "1.7976931348623157e+308"),
        ];
        for &(v, want) in cases {
            assert_eq!(go_json_float64(v), want, "{v:?}");
        }
        assert_eq!(go_json_float32(0.5), "0.5");
        assert_eq!(go_json_float32(1e21), "1e+21");
        assert_eq!(go_json_float32(1e-7), "1e-7");
    }

    #[test]
    fn formatter_keeps_pretty_layout() {
        let v = serde_json::json!({"a": [0.0, 1.5], "b": {}});
        let mut out = Vec::new();
        let mut ser = serde_json::Serializer::with_formatter(&mut out, GoJsonFormatter::with_indent(b" "));
        serde::Serialize::serialize(&v, &mut ser).unwrap();
        assert_eq!(String::from_utf8(out).unwrap(), "{\n \"a\": [\n  0,\n  1.5\n ],\n \"b\": {}\n}");
    }
}
