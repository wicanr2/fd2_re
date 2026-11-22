//! CPU 畫布：凍結 Go 版原版美術模式用到的 Ebiten v2.6.6 繪圖語意子集（R5 階段 1，Rust 新增，沒有對應的 Go 檔）。
//!
//! 依 `docs/goal/121-goal-rust-r5-original-art-shell-20261115.md`「架構」：繪圖在 CPU 產生 640×400 畫格，
//! 畫格位元組必須與 Go 版在 Xvfb／Mesa llvmpipe 下 `ReadPixels` 讀回的值逐位元組相同。
//!
//! # 為什麼放在 `game` 底下
//!
//! [`Picture`] 是 R4 定下的 `*ebiten.Image` 對應型別，`game` 裡每個繪圖函式的簽名都收 `&mut Picture`／
//! `&Arc<Picture>`；畫布操作寫成 `Picture` 的方法，移植時 `screen.DrawImage(img, op)` 一對一寫成
//! `screen.draw_image(&img, &op)`。和 `support.rs` 一樣是手寫、沒有 Go 檔對應的模組。`fd2-app` 只需要讀
//! 畫格位元組（`Picture` 的欄位是 `pub`），不需要畫布操作；之後的 GPU 模式以本模組的輸出當黃金。
//!
//! # 像素格式
//!
//! `Picture::pix` 存 **預乘 alpha** 的 RGBA8，即 Ebiten `ReadPixels` 的輸出。Go 的 `NewImageFromImage`
//! 在建立時就預乘（`*image.Paletted` 經 `color.RGBAModel`，`*image.NRGBA` 經 `image/draw`），之後的
//! 混色也都在預乘空間；照存才能逐位元組比對。
//!
//! # 子集與公式（實測出處）
//!
//! | 操作 | 規則 | 出處 |
//! |---|---|---|
//! | `NewImage` | 全透明 | Ebiten |
//! | `Fill`、`Clear` | 覆蓋（BlendCopy），不疊合；寫入 `color.RGBA` 原值，含 R/G/B 大於 A | 階段 1 `fill_values`、`fill_whole`、`clear` |
//! | `SubImage(r)` | 與父影像相交；座標仍是父影像座標；畫上去時裁切在矩形內 | 階段 1 `subimage_*` |
//! | `Set` | 寫入 `clr.RGBA()>>8`（`color.NRGBA` 先預乘） | 階段 1 `set_values` |
//! | `WritePixels`／`ReadPixels` | 原樣 | 階段 1 `write_read_roundtrip` |
//! | `NewImageFromImage(*image.Paletted)` | `color.RGBAModel.Convert(palette[i])` | 階段 1 `new_from_paletted` |
//! | `NewImageFromImage(*image.NRGBA)` | `uint8((v*a*0x101/0xff)>>8)` | 階段 1 `new_from_nrgba` |
//! | `DrawImage` 幾何 | 只有平移與縮放（`GeoM.Translate`／`Scale`）；頂點經 Ebiten `adjustDestinationPixel` 對齊，像素中心在 `[x0,x1)` 內才覆蓋，來源索引 `floor((c-x0)*n/(x1-x0))`，全程 float32 | 階段 1 取樣 89 組 |
//! | `DrawImage` 顏色 | 來源每通道 `f2u(u2f(v)*k)`（`k` 是 `ColorScale`），再 source-over `min(255, s+mul255(d*(255-sa)))`，四個通道（含 alpha）同式 | 階段 0 30 組、階段 1 `over_translucent_dst` |
//!
//! 每一步都明寫 `f32`，Rust 不會把乘加融合成 FMA。這是 llvmpipe 的行為；換 GPU 驅動可能不同，CPU 畫格是
//! 之後 GPU 模式的黃金。子集以外的操作（旋轉、負縮放、`ColorM`、線性濾鏡、其他混色模式）不實作，
//! 用到時 panic，不以近似值輸出。測試向量在 `canvas/ebiten_vectors.json`（`tools/rust_r5_canvas_vectors.sh`）。
//!
//! # GPU 模式（R5 階段 5，`r5g`）
//!
//! [`display_list::record`] 執行期間，畫到畫面目標的操作不寫 CPU 像素，改錄成顯示清單（幾何與 `ColorScale` 仍由本模組
//! 以同一段程式算好），由 `fd2-app` 在 GPU 上以整數運算執行。沒有錄製時本模組的行為與位元組完全不變。

use super::*;

#[path = "canvas/display_list.rs"]
pub mod display_list;

// ---------------- 矩形 ----------------

/// Go `image.Rect(x0, y0, x1, y1)`（會把顛倒的座標換正）。
pub const fn rect(x0: i64, y0: i64, x1: i64, y1: i64) -> Rectangle {
    let (x0, x1) = if x0 > x1 { (x1, x0) } else { (x0, x1) };
    let (y0, y1) = if y0 > y1 { (y1, y0) } else { (y0, y1) };
    Rectangle { min: Point { x: x0, y: y0 }, max: Point { x: x1, y: y1 } }
}

/// Go `r.Intersect(s)`：沒有交集時回傳 `image.ZR`（`r.Empty()` 是 `font.rs` 的 `rect_empty`）。
pub fn rect_intersect(r: Rectangle, s: Rectangle) -> Rectangle {
    let out = Rectangle {
        min: Point { x: r.min.x.max(s.min.x), y: r.min.y.max(s.min.y) },
        max: Point { x: r.max.x.min(s.max.x), y: r.max.y.min(s.max.y) },
    };
    if rect_empty(&out) { Rectangle::default() } else { out }
}

// ---------------- 數值換算（llvmpipe） ----------------

/// llvmpipe 取樣 8 位元貼圖的換算：`float32(v/256) * float32(256/255)`（階段 0）。
pub(crate) fn u2f(v: u8) -> f32 {
    let a: f32 = v as f32 / 256.0;
    let k: f32 = 256.0f32 / 255.0f32;
    a * k
}

/// llvmpipe 著色器輸出寫回 8 位元：x≤0 為 0、x≥1 為 255，其餘取 `x*(255/256) + 32768` 的 float32 位元低 8 位
/// （float32 加法以最近偶數取整，階段 0）。
pub(crate) fn f2u(x: f32) -> u8 {
    // Go 寫 `!(x > 0)`：NaN 也是 0。
    if x.is_nan() || x <= 0.0 {
        return 0;
    }
    if x >= 1.0 {
        return 255;
    }
    let k: f32 = 255.0f32 / 256.0f32;
    let y: f32 = x * k;
    let z: f32 = y + 32768.0f32;
    (z.to_bits() & 0xff) as u8
}

/// llvmpipe 8 位元混色的乘法：`x/255 ≈ (x + (x>>8) + 0x80) >> 8`（階段 0）。
pub(crate) fn mul255(x: u32) -> u32 {
    (x + (x >> 8) + 0x80) >> 8
}

/// Go `color.NRGBA.RGBA()` 高 8 位元、也是 `image/draw` 的 `drawNRGBASrc`：`uint8((v*a*0x101/0xff)>>8)`。
pub(crate) fn premultiply_channel(v: u8, a: u8) -> u8 {
    let sa = a as u32 * 0x101;
    ((v as u32 * sa / 0xff) >> 8) as u8
}

impl ColorNrgba {
    /// Go `color.RGBAModel.Convert(c)`：預乘成 `color.RGBA`。
    pub fn premultiplied(self) -> ColorRgba {
        ColorRgba {
            r: premultiply_channel(self.r, self.a),
            g: premultiply_channel(self.g, self.a),
            b: premultiply_channel(self.b, self.a),
            a: self.a,
        }
    }
}

// ---------------- GeoM 與 ColorScale ----------------

/// Ebiten `ebiten.GeoM`。和 Ebiten 一樣存 `a-1`、`d-1`（零值即單位矩陣），送進 GPU 時換成 float32
/// （`elements32`：`float32(a-1)+1`）。子集只有平移與縮放。
#[derive(Debug, Clone, Copy, Default, PartialEq)]
pub struct GeoM {
    a_1: f64,
    b: f64,
    c: f64,
    d_1: f64,
    tx: f64,
    ty: f64,
}

impl GeoM {
    /// Ebiten `GeoM.Translate`（`geom.go:119`）。
    pub fn translate(&mut self, tx: f64, ty: f64) {
        self.tx += tx;
        self.ty += ty;
    }

    /// Ebiten `GeoM.Scale`（`geom.go:102`）。
    pub fn scale(&mut self, x: f64, y: f64) {
        let a = (self.a_1 + 1.0) * x;
        let b = self.b * x;
        let tx = self.tx * x;
        let c = self.c * y;
        let d = (self.d_1 + 1.0) * y;
        let ty = self.ty * y;
        self.a_1 = a - 1.0;
        self.b = b;
        self.c = c;
        self.d_1 = d - 1.0;
        self.tx = tx;
        self.ty = ty;
    }

    /// Ebiten `GeoM.Element(i, j)`。
    pub fn element(&self, i: usize, j: usize) -> f64 {
        match (i, j) {
            (0, 0) => self.a_1 + 1.0,
            (0, 1) => self.b,
            (0, 2) => self.tx,
            (1, 0) => self.c,
            (1, 1) => self.d_1 + 1.0,
            (1, 2) => self.ty,
            _ => panic!("ebiten: i or j is out of index"),
        }
    }

    /// Ebiten `geom.go:59` elements32。
    fn elements32(&self) -> (f32, f32, f32, f32, f32, f32) {
        (self.a_1 as f32 + 1.0, self.b as f32, self.c as f32, self.d_1 as f32 + 1.0, self.tx as f32, self.ty as f32)
    }
}

/// Ebiten `ebiten.ColorScale`（預乘空間的每通道係數）。和 Ebiten 一樣以 float32 存 `係數-1`。
#[derive(Debug, Clone, Copy, Default, PartialEq)]
pub struct ColorScale {
    r_1: f32,
    g_1: f32,
    b_1: f32,
    a_1: f32,
}

impl ColorScale {
    /// Ebiten `ColorScale.Scale`（`colorscale.go`）：`c_1 = (c_1+1)*k - 1`，float32。
    pub fn scale(&mut self, r: f32, g: f32, b: f32, a: f32) {
        let f = |c_1: f32, k: f32| -> f32 {
            let m: f32 = (c_1 + 1.0) * k;
            m - 1.0
        };
        self.r_1 = f(self.r_1, r);
        self.g_1 = f(self.g_1, g);
        self.b_1 = f(self.b_1, b);
        self.a_1 = f(self.a_1, a);
    }

    /// Ebiten `ColorScale.ScaleWithColor`：係數是 `clr.RGBA()/0xffff`（預乘值）。
    pub fn scale_with_color(&mut self, c: ColorRgba) {
        let k = |v: u8| -> f32 { (v as u32 * 0x101) as f32 / 65535.0f32 };
        self.scale(k(c.r), k(c.g), k(c.b), k(c.a));
    }

    /// Ebiten `ColorScale.apply(1, 1, 1, 1)`（`ColorM` 是單位矩陣時 `colorMToScale` 給 1）。
    fn factors(&self) -> [f32; 4] {
        [(self.r_1 + 1.0) * 1.0, (self.g_1 + 1.0) * 1.0, (self.b_1 + 1.0) * 1.0, (self.a_1 + 1.0) * 1.0]
    }
}

/// Ebiten `ebiten.DrawImageOptions` 的子集：預設 source-over、最近鄰、`ColorM` 單位矩陣。
#[derive(Debug, Clone, Copy, Default, PartialEq)]
pub struct DrawImageOptions {
    pub geo_m: GeoM,
    pub color_scale: ColorScale,
}

/// Ebiten `internal/graphics/vertex.go:113` adjustDestinationPixel：目標頂點對齊到約 1/3 像素，避開像素中心。
pub(super) fn adjust_destination_pixel(x: f32) -> f32 {
    let mut ix = (x as i64) as f32; // Go `float32(int(x))`：向零截斷
    if x < 0.0 && x != ix {
        ix -= 1.0;
    }
    let frac: f32 = x - ix;
    if frac < 3.0 / 16.0 {
        ix
    } else if frac < 8.0 / 16.0 {
        ix + 5.0 / 16.0
    } else if frac < 13.0 / 16.0 {
        ix + 11.0 / 16.0
    } else {
        ix + 1.0 // Go 寫 16.0/16.0
    }
}

/// 一軸的目標座標 → 來源座標：`dst_range` 內每個目標座標對到的來源索引（`None` 是沒有覆蓋）。
/// `x0`、`x1` 是對齊後的頂點，`n` 是來源長度，`u0` 是來源起點。
fn axis_map(dst_lo: i64, dst_hi: i64, x0: f32, x1: f32, u0: i64, n: i64) -> Vec<Option<i64>> {
    let mut out = Vec::with_capacity((dst_hi - dst_lo).max(0) as usize);
    let nf = n as f32;
    let span: f32 = x1 - x0;
    for p in dst_lo..dst_hi {
        let c: f32 = p as f32 + 0.5;
        if !(c >= x0 && c < x1) {
            out.push(None);
            continue;
        }
        let rel: f32 = c - x0;
        let num: f32 = rel * nf;
        let u: f32 = num / span;
        let idx = u.floor() as i64;
        // 對齊後的頂點讓覆蓋的像素中心不會落在 x1 上；索引仍越界時（實測沒有出現）Ebiten 會取到圖集裡的
        // 鄰格，這裡當作沒有覆蓋，不猜鄰格內容。
        out.push(if (0..n).contains(&idx) { Some(u0 + idx) } else { None });
    }
    out
}

// ---------------- 影像 ----------------

/// 唯讀的 `SubImage`：父影像與範圍（父影像座標）。
#[derive(Debug, Clone, Copy)]
pub struct SubImage<'a> {
    pub pic: &'a Picture,
    pub bounds: Rectangle,
}

/// 可寫的 `SubImage`：畫上去時裁切在 `bounds` 內，座標仍是父影像座標。
#[derive(Debug)]
pub struct SubImageMut<'a> {
    pub pic: &'a mut Picture,
    pub bounds: Rectangle,
}

impl Picture {
    /// Ebiten `ebiten.NewImage(w, h)`：全透明。寬高必須為正（Ebiten 會 panic）。
    pub fn new_image(width: i64, height: i64) -> Picture {
        assert!(width > 0, "ebiten: width at NewImage must be positive but {width}");
        assert!(height > 0, "ebiten: height at NewImage must be positive but {height}");
        Picture { width, height, pix: vec![0; (width * height * 4) as usize] }
    }

    /// Ebiten `ebiten.NewImageFromImage(*image.Paletted)`（即 `assets.rs` 的 `picture_from_indexed`）。
    pub fn from_indexed(frame: &IndexedFrame) -> Picture {
        picture_from_indexed(frame)
    }

    /// Ebiten `ebiten.NewImageFromImage(*image.NRGBA)`：`pix` 是不預乘的 RGBA8，依 `image/draw` 預乘。
    pub fn from_nrgba(width: i64, height: i64, pix: &[u8]) -> Picture {
        let mut out = Picture::new_image(width, height);
        for (d, s) in out.pix.as_chunks_mut::<4>().0.iter_mut().zip(pix.as_chunks::<4>().0) {
            let a = s[3];
            d.copy_from_slice(&[
                premultiply_channel(s[0], a),
                premultiply_channel(s[1], a),
                premultiply_channel(s[2], a),
                a,
            ]);
        }
        out
    }

    /// Go `img.Bounds()`（Rust 的畫布左上一律是 (0, 0)）。
    pub fn bounds(&self) -> Rectangle {
        rect(0, 0, self.width, self.height)
    }

    /// Ebiten `img.Clear()`。
    pub fn clear(&mut self) {
        self.fill(ColorRgba::default());
    }

    /// Ebiten `img.Fill(clr)`：覆蓋整張。
    pub fn fill(&mut self, c: ColorRgba) {
        let b = self.bounds();
        fill_rect(self, b, c);
    }

    /// Ebiten `img.Set(x, y, clr)`（`color.RGBA`；`color.NRGBA` 先以 [`ColorNrgba::premultiplied`] 換算）。
    /// 範圍外不寫。
    pub fn set(&mut self, x: i64, y: i64, c: ColorRgba) {
        if x < 0 || y < 0 || x >= self.width || y >= self.height {
            return;
        }
        if display_list::is_target(self) {
            display_list::record_fill(rect(x, y, x + 1, y + 1), c);
            return;
        }
        let at = ((y * self.width + x) * 4) as usize;
        self.pix[at..at + 4].copy_from_slice(&[c.r, c.g, c.b, c.a]);
    }

    /// Ebiten `img.WritePixels(pix)`：長度必須是 `4*w*h`（Ebiten 會 panic）。
    pub fn write_pixels(&mut self, pix: &[u8]) {
        if display_list::is_target(self) {
            let want = (self.width * self.height * 4) as usize;
            assert_eq!(pix.len(), want, "ebiten: len(pix) was {} but must be {want}", pix.len());
            display_list::record_copy(self.width, self.height, pix);
            return;
        }
        assert_eq!(pix.len(), self.pix.len(), "ebiten: len(pix) was {} but must be {}", pix.len(), self.pix.len());
        self.pix.copy_from_slice(pix);
    }

    /// Ebiten `img.ReadPixels`：預乘 alpha 的 RGBA8。GPU 模式錄製中讀畫面目標會 panic（失敗即關閉）。
    pub fn read_pixels(&self) -> &[u8] {
        assert!(!display_list::is_target(self), "GPU 模式錄製中不能 ReadPixels 畫面目標");
        &self.pix
    }

    /// Ebiten `img.SubImage(r)`（唯讀，當作 `DrawImage` 的來源）。
    pub fn sub_image(&self, r: Rectangle) -> SubImage<'_> {
        SubImage { pic: self, bounds: rect_intersect(r, self.bounds()) }
    }

    /// Ebiten `img.SubImage(r)`（可寫：`Fill`、`DrawImage`、`Set` 的目標）。
    pub fn sub_image_mut(&mut self, r: Rectangle) -> SubImageMut<'_> {
        let bounds = rect_intersect(r, self.bounds());
        SubImageMut { pic: self, bounds }
    }

    /// Ebiten `dst.DrawImage(src, op)`。
    pub fn draw_image(&mut self, src: &Picture, op: &DrawImageOptions) {
        let clip = self.bounds();
        draw_quad(self, clip, src, src.bounds(), op);
    }

    /// Ebiten `dst.DrawImage(src.SubImage(r), op)`。
    pub fn draw_sub_image(&mut self, src: SubImage<'_>, op: &DrawImageOptions) {
        let clip = self.bounds();
        draw_quad(self, clip, src.pic, src.bounds, op);
    }
}

impl SubImageMut<'_> {
    /// Ebiten `sub.Fill(clr)`：只覆蓋子影像範圍。
    pub fn fill(&mut self, c: ColorRgba) {
        fill_rect(self.pic, self.bounds, c);
    }

    /// Ebiten `sub.Clear()`。
    pub fn clear(&mut self) {
        self.fill(ColorRgba::default());
    }

    /// Ebiten `sub.Set(x, y, clr)`：範圍外不寫。
    pub fn set(&mut self, x: i64, y: i64, c: ColorRgba) {
        let b = self.bounds;
        if x < b.min.x || y < b.min.y || x >= b.max.x || y >= b.max.y {
            return;
        }
        self.pic.set(x, y, c);
    }

    /// Ebiten `sub.SubImage(r)`：與目前範圍相交。
    pub fn sub_image_mut(&mut self, r: Rectangle) -> SubImageMut<'_> {
        let bounds = rect_intersect(r, self.bounds);
        SubImageMut { pic: self.pic, bounds }
    }

    /// Ebiten `sub.DrawImage(src, op)`：GeoM 以父影像座標計算，只寫入子影像範圍。
    pub fn draw_image(&mut self, src: &Picture, op: &DrawImageOptions) {
        draw_quad(self.pic, self.bounds, src, src.bounds(), op);
    }

    /// Ebiten `sub.DrawImage(src.SubImage(r), op)`。
    pub fn draw_sub_image(&mut self, src: SubImage<'_>, op: &DrawImageOptions) {
        draw_quad(self.pic, self.bounds, src.pic, src.bounds, op);
    }
}

fn fill_rect(dst: &mut Picture, r: Rectangle, c: ColorRgba) {
    // Ebiten Fill 是 BlendCopy：著色器輸出 `float32(v*0x101)/0xffff`，llvmpipe 寫回時 f2u 恰為 v
    // （256 個值都相同，見測試 `fill_float_roundtrip_is_identity`），所以直接寫位元組。
    let r = rect_intersect(r, dst.bounds());
    if display_list::is_target(dst) {
        display_list::record_fill(r, c);
        return;
    }
    for y in r.min.y..r.max.y {
        for x in r.min.x..r.max.x {
            let at = ((y * dst.width + x) * 4) as usize;
            dst.pix[at..at + 4].copy_from_slice(&[c.r, c.g, c.b, c.a]);
        }
    }
}

/// Ebiten `DrawImage` → `QuadVertices` → 最近鄰著色器 → source-over（`image.go:223`、
/// `internal/graphics/vertex.go:67`、`internal/builtinshader/shader.go`）。
fn draw_quad(dst: &mut Picture, clip: Rectangle, src: &Picture, src_r: Rectangle, op: &DrawImageOptions) {
    if rect_empty(&src_r) {
        return;
    }
    // GPU 模式：畫面目標當成來源時，先在 CPU 重建目前內容。
    let replayed;
    let src = if display_list::is_target(src) {
        replayed = display_list::target_snapshot();
        &replayed
    } else {
        src
    };
    let (a, b, c, d, tx, ty) = op.geo_m.elements32();
    assert!(b == 0.0 && c == 0.0, "fd2-game 畫布只實作平移與縮放（GeoM b={b} c={c}）");
    assert!(a >= 0.0 && d >= 0.0, "fd2-game 畫布不實作負縮放（GeoM a={a} d={d}）");
    // QuadVertices：x = sx1-sx0，ax = a*x，頂點 adjust(tx)、adjust(ax+tx)，全為 float32。
    let w: f32 = (src_r.max.x - src_r.min.x) as f32;
    let h: f32 = (src_r.max.y - src_r.min.y) as f32;
    let ax: f32 = a * w;
    let dy: f32 = d * h;
    let x0 = adjust_destination_pixel(tx);
    let x1 = adjust_destination_pixel(ax + tx);
    let y0 = adjust_destination_pixel(ty);
    let y1 = adjust_destination_pixel(dy + ty);
    let clip = rect_intersect(clip, dst.bounds());
    if rect_empty(&clip) || x1 <= x0 || y1 <= y0 {
        return;
    }
    let xs = axis_map(clip.min.x, clip.max.x, x0, x1, src_r.min.x, src_r.max.x - src_r.min.x);
    let ys = axis_map(clip.min.y, clip.max.y, y0, y1, src_r.min.y, src_r.max.y - src_r.min.y);
    // 來源每通道乘 ColorScale：f2u(u2f(v)*k)。
    let k = op.color_scale.factors();
    let mut lut = [[0u8; 256]; 4];
    for (ch, table) in lut.iter_mut().enumerate() {
        for (v, out) in table.iter_mut().enumerate() {
            let f: f32 = u2f(v as u8) * k[ch];
            *out = f2u(f);
        }
    }
    if display_list::is_target(dst) {
        display_list::record_draw(src, clip, &xs, &ys, &lut);
        return;
    }
    for (j, sy) in ys.iter().enumerate() {
        let Some(sy) = *sy else { continue };
        let y = clip.min.y + j as i64;
        for (i, sx) in xs.iter().enumerate() {
            let Some(sx) = *sx else { continue };
            let x = clip.min.x + i as i64;
            let s_at = ((sy * src.width + sx) * 4) as usize;
            let s = [
                lut[0][src.pix[s_at] as usize],
                lut[1][src.pix[s_at + 1] as usize],
                lut[2][src.pix[s_at + 2] as usize],
                lut[3][src.pix[s_at + 3] as usize],
            ];
            let inv = 255 - s[3] as u32;
            let d_at = ((y * dst.width + x) * 4) as usize;
            for (dv, sv) in dst.pix[d_at..d_at + 4].iter_mut().zip(s) {
                *dv = (sv as u32 + mul255(*dv as u32 * inv)).min(255) as u8;
            }
        }
    }
}

#[cfg(test)]
#[path = "canvas/canvas_test.rs"]
mod canvas_test;
