//! GPU 模式的顯示清單（R5 階段 5，`r5g`；Rust 新增，沒有對應的 Go 檔）。
//!
//! 依 `docs/goal/121-goal-rust-r5-original-art-shell-20261115.md`「架構」：CPU 合成是預設與驗證基準，GPU 模式在原版美術下
//! 的輸出必須與 CPU 畫格逐像素相同。做法是把畫到**畫面目標**（`Game::draw` 收到的 640×400 畫布）的畫布操作錄成本模組的
//! [`DisplayList`]，交給 `fd2-app` 的 GPU 合成器執行；畫面以外的離屏畫布（320×200 storyView、文字遮罩等）照舊在 CPU 合成，
//! 錄製時把它們當成來源貼圖快照下來。
//!
//! # 為什麼 GPU 端只做整數運算
//!
//! [`super`] 的 `DrawImage` 有兩段浮點：幾何（`adjustDestinationPixel` 與來源索引 `floor((c-x0)*n/(x1-x0))`）和顏色
//! （`f2u(u2f(v)*k)`）。WGSL 的 f32 除法只保證 2.5 ULP、實作可以把乘加融合、可以把次正規數清成零，同一段式子在 GPU 上
//! 不保證與 CPU 逐位元相同。兩段都只依賴一軸座標或一個通道值，所以錄製時在 CPU 以 [`super`] 原本的程式算好：
//!
//! - 每個目標欄、列對到的來源索引（`xs`、`ys` 兩張一維表，沒有覆蓋記 [`NONE`]）；
//! - `ColorScale` 不是恆等時，四個通道各 256 項的查表（打包成 256 個 `u32`）。
//!
//! GPU 每個像素只做查表、整數 source-over `min(255, s + mul255(d*(255-sa)))` 與覆蓋寫入，結果由構造保證與 CPU 相同。
//!
//! # 錄製的範圍與失敗即關閉
//!
//! - [`record`] 建立一個 `pix` 為空的畫面目標，執行期間對它的 `Fill`／`Clear`／`Set`／`WritePixels`／`DrawImage`／
//!   `SubImage` 寫入都錄成操作，不寫 CPU 像素。
//! - 畫面目標當成 `DrawImage` 的來源時（Rust 借用規則下只會發生在畫到別的畫布），先以 [`DisplayList::replay`] 在 CPU 重建
//!   目前內容再取樣，計入 [`DisplayList::target_reads`]。
//! - `ReadPixels` 讀畫面目標、直接存取 `pix`（空向量會越界）、或把目標換成別的 `Picture`，都會 panic，不輸出猜測的畫格。
//!   `fd2-game` 裡直接存取 `.pix` 的位置由 `display_list_test.rs` 的 `direct_pix_sites_are_reviewed` 盤點。
//! - 錄製不能巢狀；錄製期間 panic 時守衛會清掉錄製狀態。
//!
//! # 傳輸格式
//!
//! [`DisplayList::encode`]／[`DisplayList::decode`] 是驗證工具在行程之間傳清單用的小端序二進位格式（`fd2-game` 的畫格擷取
//! 測試把清單送給 `fd2-app` 的 `r5g_gpu_exec`，見 `PORTING.md`「R5 階段 5：r5g」）。

use super::*;
use std::cell::RefCell;
use std::io::{Read, Write};

/// 一維表裡「這一欄（列）沒有覆蓋」。
pub const NONE: u32 = u32::MAX;
/// 覆蓋寫入固定顏色（`Fill`、`Clear`、`Set`）。
pub const OP_FILL: u32 = 0;
/// 覆蓋寫入來源像素、1:1（`WritePixels`）。
pub const OP_COPY: u32 = 1;
/// 最近鄰取樣、`ColorScale` 查表、source-over（`DrawImage`）。
pub const OP_DRAW: u32 = 2;
/// [`Op`] 的 `u32` 個數（GPU 端的結構同序）。
pub const OP_WORDS: usize = 12;

/// 一個操作。矩形 `[x0,x1)×[y0,y1)` 已裁在畫面內。
#[derive(Debug, Clone, Copy, Default, PartialEq, Eq)]
pub struct Op {
    pub kind: u32,
    pub x0: u32,
    pub y0: u32,
    pub x1: u32,
    pub y1: u32,
    /// `OP_FILL` 的顏色（`r | g<<8 | b<<16 | a<<24`，預乘）。
    pub color: u32,
    /// 來源快照在 [`DisplayList::src`] 的起點（像素）與列寬。
    pub src_off: u32,
    pub src_w: u32,
    /// `OP_DRAW`：`xs`、`ys` 表在 [`DisplayList::tables`] 的起點；表長是矩形寬、高，值是快照內的座標或 [`NONE`]。
    pub xs_off: u32,
    pub ys_off: u32,
    /// `OP_DRAW`：`ColorScale` 查表在 [`DisplayList::luts`] 的起點（256 項）；恆等時是 [`NONE`]。
    pub lut_off: u32,
    pub reserved: u32,
}

impl Op {
    fn words(&self) -> [u32; OP_WORDS] {
        [
            self.kind,
            self.x0,
            self.y0,
            self.x1,
            self.y1,
            self.color,
            self.src_off,
            self.src_w,
            self.xs_off,
            self.ys_off,
            self.lut_off,
            self.reserved,
        ]
    }

    fn from_words(w: [u32; OP_WORDS]) -> Op {
        Op {
            kind: w[0],
            x0: w[1],
            y0: w[2],
            x1: w[3],
            y1: w[4],
            color: w[5],
            src_off: w[6],
            src_w: w[7],
            xs_off: w[8],
            ys_off: w[9],
            lut_off: w[10],
            reserved: w[11],
        }
    }
}

/// 一張畫面的顯示清單：從全透明開始，依序套用 `ops`。
#[derive(Debug, Clone, Default, PartialEq, Eq)]
pub struct DisplayList {
    pub width: u32,
    pub height: u32,
    pub ops: Vec<Op>,
    /// 來源快照（預乘 RGBA8，逐像素 4 位元組）。
    pub src: Vec<u8>,
    pub tables: Vec<u32>,
    pub luts: Vec<u32>,
    /// 畫面目標被當成來源、以 CPU 重建的次數。
    pub target_reads: u32,
    /// 來源快照重複、沿用先前快照的次數（只是統計）。
    pub src_reused: u32,
}

fn pack(px: [u8; 4]) -> u32 {
    u32::from_le_bytes(px)
}

/// llvmpipe 8 位元混色（與 [`super`] 的 `draw_quad` 同式）：`min(255, s + mul255(d*(255-sa)))`，四個通道同式。
fn over(s: [u8; 4], d: &mut [u8]) {
    let inv = 255 - s[3] as u32;
    for (dv, sv) in d.iter_mut().zip(s) {
        *dv = (sv as u32 + mul255(*dv as u32 * inv)).min(255) as u8;
    }
}

impl DisplayList {
    /// 依序在 CPU 套用全部操作，回傳預乘 RGBA8（GPU 合成器的參考實作，也用來重建被當成來源的畫面目標）。
    pub fn replay(&self) -> Vec<u8> {
        let mut dst = vec![0u8; (self.width * self.height * 4) as usize];
        self.replay_into(&mut dst, self.ops.len());
        dst
    }

    fn replay_into(&self, dst: &mut [u8], n: usize) {
        let w = self.width as usize;
        for op in &self.ops[..n] {
            for y in op.y0..op.y1 {
                let row = (y as usize) * w;
                let sy = match op.kind {
                    OP_DRAW => self.tables[(op.ys_off + (y - op.y0)) as usize],
                    _ => y - op.y0,
                };
                if sy == NONE {
                    continue;
                }
                for x in op.x0..op.x1 {
                    let at = (row + x as usize) * 4;
                    match op.kind {
                        OP_FILL => dst[at..at + 4].copy_from_slice(&op.color.to_le_bytes()),
                        OP_COPY => {
                            let s = ((op.src_off + sy * op.src_w + (x - op.x0)) * 4) as usize;
                            let px: [u8; 4] = self.src[s..s + 4].try_into().unwrap();
                            dst[at..at + 4].copy_from_slice(&px);
                        }
                        OP_DRAW => {
                            let sx = self.tables[(op.xs_off + (x - op.x0)) as usize];
                            if sx == NONE {
                                continue;
                            }
                            let s = ((op.src_off + sy * op.src_w + sx) * 4) as usize;
                            let mut px: [u8; 4] = self.src[s..s + 4].try_into().unwrap();
                            if op.lut_off != NONE {
                                for (ch, v) in px.iter_mut().enumerate() {
                                    *v = (self.luts[(op.lut_off + *v as u32) as usize] >> (8 * ch)) as u8;
                                }
                            }
                            over(px, &mut dst[at..at + 4]);
                        }
                        other => panic!("顯示清單：未知的操作 {other}"),
                    }
                }
            }
        }
    }

    /// 檢查索引都在範圍內（`decode` 之後、送進 GPU 之前呼叫；GPU 端不做越界檢查以外的防護）。
    pub fn validate(&self) -> Result<(), String> {
        if self.width == 0 || self.height == 0 || self.width > 16384 || self.height > 16384 {
            return Err(format!("顯示清單：畫面大小 {}×{}", self.width, self.height));
        }
        if !self.src.len().is_multiple_of(4) {
            return Err("顯示清單：來源長度不是 4 的倍數".into());
        }
        let src_px = (self.src.len() / 4) as u64;
        for (i, op) in self.ops.iter().enumerate() {
            let bad = |m: &str| Err(format!("顯示清單第 {i} 個操作：{m}（{op:?}）"));
            if op.x0 > op.x1 || op.y0 > op.y1 || op.x1 > self.width || op.y1 > self.height {
                return bad("矩形超出畫面");
            }
            let (w, h) = ((op.x1 - op.x0) as u64, (op.y1 - op.y0) as u64);
            match op.kind {
                OP_FILL => {}
                OP_COPY => {
                    if op.src_w as u64 != w || (op.src_off as u64 + w * h) > src_px {
                        return bad("來源超出範圍");
                    }
                }
                OP_DRAW => {
                    let table = |off: u32, n: u64| -> Option<&[u32]> {
                        self.tables.get(off as usize..(off as u64 + n) as usize)
                    };
                    let (Some(xs), Some(ys)) = (table(op.xs_off, w), table(op.ys_off, h)) else {
                        return bad("索引表超出範圍");
                    };
                    let max = |t: &[u32]| t.iter().copied().filter(|&v| v != NONE).max();
                    if let (Some(mx), Some(my)) = (max(xs), max(ys))
                        && (mx >= op.src_w || op.src_off as u64 + my as u64 * op.src_w as u64 + mx as u64 >= src_px)
                    {
                        return bad("來源超出範圍");
                    }
                    if op.lut_off != NONE && op.lut_off as usize + 256 > self.luts.len() {
                        return bad("查表超出範圍");
                    }
                }
                other => return bad(&format!("未知的操作 {other}")),
            }
        }
        Ok(())
    }

    /// 寫成傳輸格式：`"FD2DL001"`，再依序是 `width, height, ops 數, src 位元組數, tables 數, luts 數,
    /// target_reads, src_reused`（各 `u32`），之後是 ops（每個 [`OP_WORDS`] 個 `u32`）、src、tables、luts；全為小端序。
    pub fn encode(&self, w: &mut impl Write) -> std::io::Result<()> {
        let mut head = Vec::with_capacity(8 + 8 * 4);
        head.extend_from_slice(b"FD2DL001");
        for v in [
            self.width,
            self.height,
            self.ops.len() as u32,
            self.src.len() as u32,
            self.tables.len() as u32,
            self.luts.len() as u32,
            self.target_reads,
            self.src_reused,
        ] {
            head.extend_from_slice(&v.to_le_bytes());
        }
        w.write_all(&head)?;
        let mut words = Vec::with_capacity(self.ops.len() * OP_WORDS * 4);
        for op in &self.ops {
            for v in op.words() {
                words.extend_from_slice(&v.to_le_bytes());
            }
        }
        w.write_all(&words)?;
        w.write_all(&self.src)?;
        w.write_all(&u32s_to_bytes(&self.tables))?;
        w.write_all(&u32s_to_bytes(&self.luts))
    }

    /// 讀傳輸格式。檔頭在 EOF 處結束時回 `Ok(None)`（對方正常關閉）。
    pub fn decode(r: &mut impl Read) -> Result<Option<DisplayList>, String> {
        let mut magic = [0u8; 8];
        match read_exact_or_eof(r, &mut magic)? {
            false => return Ok(None),
            true if &magic != b"FD2DL001" => return Err(format!("顯示清單：檔頭 {magic:?}")),
            true => {}
        }
        let mut head = [0u8; 32];
        r.read_exact(&mut head).map_err(|e| format!("顯示清單檔頭：{e}"))?;
        let h: Vec<u32> = head.as_chunks::<4>().0.iter().map(|b| u32::from_le_bytes(*b)).collect();
        let (n_ops, src_len, n_tables, n_luts) = (h[2] as usize, h[3] as usize, h[4] as usize, h[5] as usize);
        if n_ops > 1 << 22 || src_len > 1 << 30 || n_tables > 1 << 26 || n_luts > 1 << 26 {
            return Err(format!("顯示清單：大小不合理 {h:?}"));
        }
        let read = |r: &mut dyn Read, n: usize| -> Result<Vec<u8>, String> {
            let mut buf = vec![0u8; n];
            r.read_exact(&mut buf).map_err(|e| format!("顯示清單本體：{e}"))?;
            Ok(buf)
        };
        let ops_raw = read(r, n_ops * OP_WORDS * 4)?;
        let ops = ops_raw
            .as_chunks::<{ OP_WORDS * 4 }>()
            .0
            .iter()
            .map(|c| {
                let mut w = [0u32; OP_WORDS];
                for (v, b) in w.iter_mut().zip(c.as_chunks::<4>().0) {
                    *v = u32::from_le_bytes(*b);
                }
                Op::from_words(w)
            })
            .collect();
        let src = read(r, src_len)?;
        let tables = bytes_to_u32s(&read(r, n_tables * 4)?);
        let luts = bytes_to_u32s(&read(r, n_luts * 4)?);
        let dl =
            DisplayList { width: h[0], height: h[1], ops, src, tables, luts, target_reads: h[6], src_reused: h[7] };
        dl.validate()?;
        Ok(Some(dl))
    }
}

/// `u32` 陣列 → 小端序位元組（GPU 上傳與傳輸格式共用）。
pub fn u32s_to_bytes(v: &[u32]) -> Vec<u8> {
    let mut out = Vec::with_capacity(v.len() * 4);
    for x in v {
        out.extend_from_slice(&x.to_le_bytes());
    }
    out
}

fn bytes_to_u32s(b: &[u8]) -> Vec<u32> {
    b.as_chunks::<4>().0.iter().map(|c| u32::from_le_bytes(*c)).collect()
}

fn read_exact_or_eof(r: &mut impl Read, buf: &mut [u8]) -> Result<bool, String> {
    let mut got = 0;
    while got < buf.len() {
        match r.read(&mut buf[got..]) {
            Ok(0) if got == 0 => return Ok(false),
            Ok(0) => return Err("顯示清單：檔頭中途結束".into()),
            Ok(n) => got += n,
            Err(e) if e.kind() == std::io::ErrorKind::Interrupted => {}
            Err(e) => return Err(format!("顯示清單：{e}")),
        }
    }
    Ok(true)
}

// ---------------- 錄製 ----------------

/// 先前的來源快照（同一個來源、同一塊範圍、內容相同時沿用）。
struct Snapshot {
    addr: usize,
    region: (i64, i64, i64, i64),
    off: u32,
}

struct Recorder {
    target: usize,
    list: DisplayList,
    snapshots: Vec<Snapshot>,
}

thread_local! {
    static RECORDER: RefCell<Option<Recorder>> = const { RefCell::new(None) };
}

fn addr(p: &Picture) -> usize {
    p as *const Picture as usize
}

/// `p` 是否是錄製中的畫面目標。
pub(crate) fn is_target(p: &Picture) -> bool {
    RECORDER.with(|r| r.borrow().as_ref().is_some_and(|rec| rec.target == addr(p)))
}

struct Guard;

impl Drop for Guard {
    fn drop(&mut self) {
        RECORDER.with(|r| r.borrow_mut().take());
    }
}

/// 在 `f` 執行期間把畫到畫面目標（`width×height`、從全透明開始）的操作錄成顯示清單。
pub fn record(width: i64, height: i64, f: impl FnOnce(&mut Picture)) -> DisplayList {
    assert!(width > 0 && height > 0, "顯示清單：畫面大小 {width}×{height}");
    let mut target = Picture { width, height, pix: Vec::new() };
    RECORDER.with(|r| {
        let mut r = r.borrow_mut();
        assert!(r.is_none(), "顯示清單不能巢狀錄製");
        *r = Some(Recorder {
            target: addr(&target),
            list: DisplayList { width: width as u32, height: height as u32, ..Default::default() },
            snapshots: Vec::new(),
        });
    });
    let guard = Guard;
    f(&mut target);
    let rec = RECORDER.with(|r| r.borrow_mut().take()).expect("錄製狀態遺失");
    drop(guard);
    assert!(
        target.pix.is_empty() && (target.width, target.height) == (width, height),
        "畫面目標在錄製期間被直接改寫（pix 長度 {}、大小 {}×{}）；GPU 模式不能錄這種寫入",
        target.pix.len(),
        target.width,
        target.height
    );
    rec.list
}

fn with_recorder<R>(f: impl FnOnce(&mut Recorder) -> R) -> R {
    RECORDER.with(|r| f(r.borrow_mut().as_mut().expect("不在錄製中")))
}

/// 畫面目標目前的內容（被當成來源時用）。
pub(crate) fn target_snapshot() -> Picture {
    with_recorder(|rec| {
        rec.list.target_reads += 1;
        let (w, h) = (rec.list.width as i64, rec.list.height as i64);
        Picture { width: w, height: h, pix: rec.list.replay() }
    })
}

fn clip_u32(r: Rectangle) -> (u32, u32, u32, u32) {
    (r.min.x as u32, r.min.y as u32, r.max.x as u32, r.max.y as u32)
}

/// `Fill`／`Set` 到畫面目標；`r` 已與畫面相交。
pub(crate) fn record_fill(r: Rectangle, c: ColorRgba) {
    if rect_empty(&r) {
        return;
    }
    let (x0, y0, x1, y1) = clip_u32(r);
    with_recorder(|rec| {
        rec.list.ops.push(Op { kind: OP_FILL, x0, y0, x1, y1, color: pack([c.r, c.g, c.b, c.a]), ..Default::default() })
    });
}

/// `WritePixels` 到畫面目標（整張）。
pub(crate) fn record_copy(width: i64, height: i64, pix: &[u8]) {
    with_recorder(|rec| {
        let src_off = (rec.list.src.len() / 4) as u32;
        rec.list.src.extend_from_slice(pix);
        rec.list.ops.push(Op {
            kind: OP_COPY,
            x0: 0,
            y0: 0,
            x1: width as u32,
            y1: height as u32,
            src_off,
            src_w: width as u32,
            ..Default::default()
        });
    });
}

/// `DrawImage` 到畫面目標：`clip` 是 `xs`／`ys` 第 0 項對應的目標座標，`lut` 是四個通道的 `ColorScale` 查表。
pub(crate) fn record_draw(
    src: &Picture,
    clip: Rectangle,
    xs: &[Option<i64>],
    ys: &[Option<i64>],
    lut: &[[u8; 256]; 4],
) {
    let span = |t: &[Option<i64>]| -> Option<(usize, usize, i64, i64)> {
        let first = t.iter().position(Option::is_some)?;
        let last = t.iter().rposition(Option::is_some)?;
        let vals = t[first..=last].iter().flatten();
        let (lo, hi) = vals.fold((i64::MAX, i64::MIN), |(lo, hi), &v| (lo.min(v), hi.max(v)));
        Some((first, last + 1, lo, hi))
    };
    let (Some((i0, i1, sx0, sx1)), Some((j0, j1, sy0, sy1))) = (span(xs), span(ys)) else {
        return;
    };
    let (sw, sh) = (sx1 - sx0 + 1, sy1 - sy0 + 1);
    let identity = (0..256).all(|v| lut.iter().all(|t| t[v] as usize == v));
    let a = addr(src);
    with_recorder(|rec| {
        let list = &mut rec.list;
        // 來源快照：只取實際取樣的範圍；同一來源、同範圍、內容相同時沿用。
        let row_bytes = (sw * 4) as usize;
        let region = (sx0, sy0, sw, sh);
        let rows = |y: i64| {
            let at = ((y * src.width + sx0) * 4) as usize;
            &src.pix[at..at + row_bytes]
        };
        let reuse = rec.snapshots.iter().find(|s| {
            s.addr == a && s.region == region && {
                let base = s.off as usize * 4;
                (0..sh).all(|dy| {
                    let at = base + dy as usize * row_bytes;
                    list.src[at..at + row_bytes] == *rows(sy0 + dy)
                })
            }
        });
        let src_off = match reuse {
            Some(s) => {
                list.src_reused += 1;
                s.off
            }
            None => {
                let off = (list.src.len() / 4) as u32;
                for y in sy0..=sy1 {
                    list.src.extend_from_slice(rows(y));
                }
                rec.snapshots.push(Snapshot { addr: a, region, off });
                off
            }
        };
        let xs_off = list.tables.len() as u32;
        list.tables.extend(xs[i0..i1].iter().map(|v| v.map_or(NONE, |v| (v - sx0) as u32)));
        let ys_off = list.tables.len() as u32;
        list.tables.extend(ys[j0..j1].iter().map(|v| v.map_or(NONE, |v| (v - sy0) as u32)));
        let lut_off = if identity {
            NONE
        } else {
            let off = list.luts.len() as u32;
            list.luts.extend((0..256).map(|v| pack([lut[0][v], lut[1][v], lut[2][v], lut[3][v]])));
            off
        };
        list.ops.push(Op {
            kind: OP_DRAW,
            x0: (clip.min.x + i0 as i64) as u32,
            y0: (clip.min.y + j0 as i64) as u32,
            x1: (clip.min.x + i1 as i64) as u32,
            y1: (clip.min.y + j1 as i64) as u32,
            color: 0,
            src_off,
            src_w: sw as u32,
            xs_off,
            ys_off,
            lut_off,
            reserved: 0,
        });
    });
}

// ---------------- 外部執行器（驗證工具用） ----------------

/// 以子行程執行顯示清單的 GPU 合成器（`fd2-app` 的 `r5g_gpu_exec`）。畫格擷取測試在 GPU 模式下把每張清單送過去，
/// 取回預乘 RGBA8；`fd2-game` 本身不依賴 GPU。
///
/// 協定：子行程啟動後先寫 `"FD2GPU01"`、`u32` 長度與該長度的 UTF-8 轉接器說明（JSON）；之後每收到一份
/// [`DisplayList::encode`]，回 `"FD2PX001"`、`u32` 狀態（0 成功）、`u32` 長度與內容（成功是畫格，失敗是錯誤訊息）。
/// 關閉 stdin 即結束。
pub struct ExternalExecutor {
    child: std::process::Child,
    stdin: Option<std::io::BufWriter<std::process::ChildStdin>>,
    stdout: std::io::BufReader<std::process::ChildStdout>,
    /// 子行程回報的轉接器說明（JSON）。
    pub adapter: String,
}

fn read_frame(r: &mut impl Read, magic: &[u8; 8]) -> Result<(u32, Vec<u8>), String> {
    let mut head = [0u8; 16];
    r.read_exact(&mut head).map_err(|e| format!("GPU 執行器：讀回應：{e}"))?;
    if &head[..8] != magic {
        return Err(format!("GPU 執行器：回應檔頭 {:?}", &head[..8]));
    }
    let status = u32::from_le_bytes(head[8..12].try_into().unwrap());
    let len = u32::from_le_bytes(head[12..16].try_into().unwrap()) as usize;
    let mut body = vec![0u8; len];
    r.read_exact(&mut body).map_err(|e| format!("GPU 執行器：讀回應本體：{e}"))?;
    Ok((status, body))
}

impl ExternalExecutor {
    /// 啟動執行器並讀它的轉接器說明。
    pub fn spawn(path: &str) -> Result<ExternalExecutor, String> {
        use std::process::{Command, Stdio};
        let mut child = Command::new(path)
            .stdin(Stdio::piped())
            .stdout(Stdio::piped())
            .stderr(Stdio::inherit())
            .spawn()
            .map_err(|e| format!("啟動 GPU 執行器 {path}：{e}"))?;
        let stdin = std::io::BufWriter::with_capacity(1 << 20, child.stdin.take().unwrap());
        let mut stdout = std::io::BufReader::with_capacity(1 << 20, child.stdout.take().unwrap());
        let (status, body) = read_frame(&mut stdout, b"FD2GPU01")?;
        let adapter = String::from_utf8_lossy(&body).into_owned();
        if status != 0 {
            let _ = child.wait();
            return Err(format!("GPU 執行器無法使用：{adapter}"));
        }
        Ok(ExternalExecutor { child, stdin: Some(stdin), stdout, adapter })
    }

    /// 執行一份清單，回傳預乘 RGBA8（`width*height*4` 位元組）。
    pub fn render(&mut self, dl: &DisplayList) -> Result<Vec<u8>, String> {
        let stdin = self.stdin.as_mut().expect("GPU 執行器已關閉");
        dl.encode(stdin).and_then(|_| stdin.flush()).map_err(|e| format!("GPU 執行器：寫清單：{e}"))?;
        let (status, body) = read_frame(&mut self.stdout, b"FD2PX001")?;
        if status != 0 {
            return Err(format!("GPU 執行器：{}", String::from_utf8_lossy(&body)));
        }
        let want = (dl.width * dl.height * 4) as usize;
        if body.len() != want {
            return Err(format!("GPU 執行器：畫格 {} 位元組，應為 {want}", body.len()));
        }
        Ok(body)
    }
}

impl Drop for ExternalExecutor {
    fn drop(&mut self) {
        self.stdin.take();
        let _ = self.child.wait();
    }
}

#[cfg(test)]
#[path = "display_list_test.rs"]
mod display_list_test;
