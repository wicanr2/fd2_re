//! Go 來源：`remake/cmd/fd2/assets.go`（202 行）。
//!
//! assets.go — 資產路徑解析層(打包前置,doc38 §6.5)。
//!
//! 唯讀資產(assets/...)五層查找,先中先贏,不混層:
//!  1. XDG 使用者覆蓋層 $XDG_DATA_HOME/fd2_re/assets/...
//!     (doc38「使用者資料優先」原則:玩家編輯過的版本 / 版權衍生素材如 sprites·music·portraits·
//!     maps 皆放這裡,AppImage 只打包 scenarios/story 等原創內容,故這些目錄實務上只在此層出現)
//!  2. AppImage 唯讀基底 $APPDIR/assets/...(有設 APPDIR 才查;AppImage 執行期自動設)
//!  3. macOS app bundle 的 Contents/Resources 相對
//!  4. 執行檔所在目錄 相對(playfix #2:雙擊 / 從別的 cwd 用絕對路徑啟動 ./fd2-linux 時,
//!     cwd 不一定是 remake/,資產解不到會靜默跳過開場動畫直接進戰場;開發模式 cwd=remake/
//!     時這層與第4層同值,行為不變)
//!  5. cwd 與祖先目錄(開發模式；供 `go test` 在 cmd/fd2 package cwd 回到 remake/)
//!
//! 可寫檔(存檔/設定)一律走 $XDG_DATA_HOME/fd2_re/,不再用 cwd(唯讀 mount 內無法寫入)。
//!
//! 骨架：Go 檔的 7 個函式中產生 6 個（階段 0 清單），其餘標 `r8`，見 `PORTING.json`。
//!
//! 路徑以 `std::path` 處理（教訓 `rust-port-no-go-stdlib`）：Go `filepath.Join` 會做詞法 Clean，
//! 這裡只略過空的組成段，不消去 `..`；本檔的呼叫端只傳固定的相對路徑，結果相同。
//!
//! 本檔另有 Rust 新增的畫面轉換工具（`picture_from_*`、調色盤轉換、PNG 解碼），取代 Go 的
//! `ebiten.NewImageFromImage` 與 `image.Decode`；A 組的載入函式與其他組都可以用。

use super::*;
use std::path::{Path, PathBuf};

/// Go：`assets.go:29`
///
/// Go 會在執行期改寫這個套件層變數，Rust 以 `RwLock` 包住。
pub(crate) static EXE_DIR_CACHED: LazyLock<RwLock<String>> = LazyLock::new(|| RwLock::new(Default::default()));

/// Go：`assets.go:30`
///
/// Go 會在執行期改寫這個套件層變數，Rust 以 `RwLock` 包住。
pub(crate) static EXE_DIR_LOOKED: LazyLock<RwLock<bool>> = LazyLock::new(|| RwLock::new(Default::default()));

/// Go `filepath.Join`（不做詞法 Clean，見模組文件）。
pub(crate) fn join_path(parts: &[&str]) -> String {
    let mut out = PathBuf::new();
    for part in parts {
        if !part.is_empty() {
            out.push(part);
        }
    }
    out.to_string_lossy().into_owned()
}

/// exeDir 回傳執行檔所在目錄(符號連結已解),取不到則回傳 ""。
///
/// Go：`assets.go:33`，`func exeDir() string`
pub(crate) fn exe_dir() -> String {
    if *EXE_DIR_LOOKED.read().unwrap() {
        return EXE_DIR_CACHED.read().unwrap().clone();
    }
    *EXE_DIR_LOOKED.write().unwrap() = true;
    let Ok(mut p) = std::env::current_exe() else {
        return String::new();
    };
    if let Ok(rp) = std::fs::canonicalize(&p) {
        p = rp;
    }
    let dir = p.parent().map(|d| d.to_string_lossy().into_owned()).unwrap_or_else(|| String::from("."));
    *EXE_DIR_CACHED.write().unwrap() = dir.clone();
    dir
}

/// Go：`assets.go:49`
///
/// Go 會在執行期改寫這個套件層變數，Rust 以 `RwLock` 包住。
pub(crate) static USER_DATA_DIR_CACHED: LazyLock<RwLock<String>> = LazyLock::new(|| RwLock::new(Default::default()));

/// userDataDir 回傳(並確保存在)可寫使用者資料夾:$XDG_DATA_HOME/fd2_re/,預設 ~/.local/share/fd2_re/。
///
/// Go：`assets.go:52`，`func userDataDir() string`
pub(crate) fn user_data_dir() -> String {
    {
        let cached = USER_DATA_DIR_CACHED.read().unwrap();
        if !cached.is_empty() {
            return cached.clone();
        }
    }
    let mut base = std::env::var("XDG_DATA_HOME").unwrap_or_default();
    if base.is_empty() {
        // Go `os.UserHomeDir()`：Unix 讀 $HOME，失敗時退回 "."。
        let home = std::env::var("HOME").ok().filter(|h| !h.is_empty()).unwrap_or_else(|| String::from("."));
        base = join_path(&[&home, ".local", "share"]);
    }
    let dir = join_path(&[&base, "fd2_re"]);
    let _ = std::fs::create_dir_all(&dir);
    *USER_DATA_DIR_CACHED.write().unwrap() = dir.clone();
    dir
}

/// userDataPath 回傳可寫檔的完整路徑(存檔/設定/編輯器輸出)。
///
/// Go：`assets.go:71`，`func userDataPath(name string) string`
pub(crate) fn user_data_path(name: &str) -> String {
    join_path(&[&user_data_dir(), name])
}

/// Go：`assets.go:75`，`func fileExists(p string) bool`
pub(crate) fn file_exists(p: &str) -> bool {
    std::fs::metadata(p).is_ok()
}

/// macBundleResourceDir 回傳標準 .app bundle 的 Resources 目錄候選。
/// 只辨識 Contents/MacOS 這個結構；實際資產仍須由呼叫端以存在性檢查承認。
///
/// Go：`assets.go:82`，`func macBundleResourceDir() string`
pub(crate) fn mac_bundle_resource_dir() -> String {
    let d = exe_dir();
    let d = Path::new(&d);
    if d.as_os_str().is_empty() || d.file_name().is_none_or(|n| n != "MacOS") {
        return String::new();
    }
    let Some(contents) = d.parent() else {
        return String::new();
    };
    if contents.file_name().is_none_or(|n| n != "Contents") {
        return String::new();
    }
    contents.join("Resources").to_string_lossy().into_owned()
}

/// assetPath 解析一個唯讀資產相對路徑(如 "assets/map.json" 或 "assets/maps/map3" 目錄)。
/// 絕對路徑(系統字型等)原樣回傳。五層都沒有 → 回傳未改寫的 cwd 相對路徑,
/// 呼叫端的 os.ReadFile 自然得到「檔案不存在」錯誤,行為與改動前一致。
///
/// Go：`assets.go:97`，`func assetPath(rel string) string`
pub(crate) fn asset_path(rel: &str) -> String {
    if Path::new(rel).is_absolute() {
        return rel.to_string();
    }
    let p = join_path(&[&user_data_dir(), rel]);
    if file_exists(&p) {
        return p;
    }
    if let Ok(appdir) = std::env::var("APPDIR")
        && !appdir.is_empty()
    {
        let p = join_path(&[&appdir, rel]);
        if file_exists(&p) {
            return p;
        }
    }
    let resources = mac_bundle_resource_dir();
    if !resources.is_empty() {
        let p = join_path(&[&resources, rel]);
        if file_exists(&p) {
            return p;
        }
    }
    let d = exe_dir();
    if !d.is_empty() {
        let p = join_path(&[&d, rel]);
        if file_exists(&p) {
            return p;
        }
    }
    if let Ok(cwd) = std::env::current_dir() {
        let mut dir = cwd;
        for _ in 0..5 {
            let p = dir.join(rel).to_string_lossy().into_owned();
            if file_exists(&p) {
                return p;
            }
            match dir.parent() {
                Some(parent) if parent != dir => dir = parent.to_path_buf(),
                _ => break,
            }
        }
    }
    rel.to_string()
}

// ---------------- Rust 新增：格式化 ----------------

/// Go `fmt.Sprintf("%v", []int{...})`：`[1 2 3]`。
pub(crate) fn go_int_slice(v: &[i64]) -> String {
    let parts: Vec<String> = v.iter().map(|x| x.to_string()).collect();
    format!("[{}]", parts.join(" "))
}

// ---------------- Rust 新增：畫面轉換 ----------------

/// `fdother::Palette`（Go `color.Palette` 的 `color.NRGBA` 元素）→ [`ColorPalette`]：照 Go
/// `color.NRGBA.RGBA()` 預乘 alpha 後取高 8 位元（VGA 調色盤全是不透明色，值不變）。
pub(crate) fn color_palette_from_nrgba(palette: &[fdother::Nrgba]) -> ColorPalette {
    palette
        .iter()
        .map(|c| {
            let (r, g, b, a) = c.rgba();
            ColorRgba { r: (r >> 8) as u8, g: (g >> 8) as u8, b: (b >> 8) as u8, a: (a >> 8) as u8 }
        })
        .collect()
}

/// [`ColorPalette`] → `fdother::Palette`（解除預乘；不透明色原值）。
pub(crate) fn nrgba_palette_from_color(palette: &[ColorRgba]) -> fdother::Palette {
    palette
        .iter()
        .map(|c| {
            let (r, g, b) = unpremultiply(*c);
            fdother::Nrgba { r, g, b, a: c.a }
        })
        .collect()
}

fn unpremultiply(c: ColorRgba) -> (u8, u8, u8) {
    match c.a {
        0 => (0, 0, 0),
        255 => (c.r, c.g, c.b),
        a => {
            let f = |v: u8| ((v as u32 * 255 + a as u32 / 2) / a as u32).min(255) as u8;
            (f(c.r), f(c.g), f(c.b))
        }
    }
}

/// Go `ebiten.NewImageFromImage(*image.Paletted)`：每像素存 `color.RGBAModel.Convert(palette[i])`
/// （[`ColorRgba`] 已預乘，原值寫入；R5 階段 1 起 [`Picture`] 存預乘值）。
///
/// 範圍外的索引：Ebiten `imageToBytes` 會 panic；這裡沿用 R4 寫成透明黑（KNOWN_DIFFERENCES
/// `indexed-out-of-palette-transparent`，正式素材的調色盤都是 256 色，不會出現）。
pub(crate) fn picture_from_indexed(frame: &IndexedFrame) -> Picture {
    let mut pix = vec![0u8; frame.pix.len() * 4];
    for (i, &index) in frame.pix.iter().enumerate() {
        if let Some(&c) = frame.palette.get(index as usize) {
            pix[i * 4..i * 4 + 4].copy_from_slice(&[c.r, c.g, c.b, c.a]);
        }
    }
    Picture { width: frame.width, height: frame.height, pix }
}

/// Go `ebiten.NewImageFromImage(*image.Paletted)`（`fdother::PalettedImage`，調色盤是 `color.NRGBA`）：
/// 每像素 `color.RGBAModel.Convert` 預乘。
pub(crate) fn picture_from_paletted(image: &fdother::PalettedImage) -> Picture {
    let (w, h) = (image.width.max(0), image.height.max(0));
    let mut pix = vec![0u8; (w * h * 4) as usize];
    for y in 0..h {
        for x in 0..w {
            let index = image.color_index_at(x, y) as usize;
            if let Some(c) = image.palette.get(index) {
                let at = ((y * w + x) * 4) as usize;
                let p = ColorNrgba { r: c.r, g: c.g, b: c.b, a: c.a }.premultiplied();
                pix[at..at + 4].copy_from_slice(&[p.r, p.g, p.b, p.a]);
            }
        }
    }
    Picture { width: w, height: h, pix }
}

/// Go `img.SubImage(image.Rect(x, y, x+w, y+h))`：與原圖相交的部分複製成新畫面（Go 共用像素，
/// Rust 複製；`fd2-game` 不寫入圖塊）。
pub(crate) fn crop_picture(src: &Picture, x: i64, y: i64, w: i64, h: i64) -> Picture {
    let (x0, y0) = (x.max(0), y.max(0));
    let (x1, y1) = ((x + w).min(src.width), (y + h).min(src.height));
    if x1 <= x0 || y1 <= y0 {
        return Picture::default();
    }
    let (cw, ch) = (x1 - x0, y1 - y0);
    let mut pix = Vec::with_capacity((cw * ch * 4) as usize);
    for row in y0..y1 {
        let start = ((row * src.width + x0) * 4) as usize;
        pix.extend_from_slice(&src.pix[start..start + (cw * 4) as usize]);
    }
    Picture { width: cw, height: ch, pix }
}

/// 以 320×200 等固定幾何建立 [`IndexedFrame`]（Go `image.NewPaletted` 後 `copy(Pix, pixels)`）。
pub(crate) fn indexed_frame(width: i64, height: i64, pixels: &[u8], palette: ColorPalette) -> IndexedFrame {
    let mut pix = vec![0u8; (width * height) as usize];
    let n = pix.len().min(pixels.len());
    pix[..n].copy_from_slice(&pixels[..n]);
    IndexedFrame { width, height, pix, palette }
}

/// Go `image.Decode`＋`ebiten.NewImageFromImage`：PNG 解成 RGBA8 後照 Ebiten 預乘（R5 階段 1 起 [`Picture`]
/// 存預乘值）；失敗回 `None`（Go 呼叫端失敗時略過該檔）。
///
/// Go 的 PNG 解碼器對帶 alpha 的 8 位元圖回傳 `*image.NRGBA`、對調色盤圖回傳 `color.NRGBA` 調色盤，兩者經
/// `NewImageFromImage` 都是 `uint8((v*a*0x101/0xff)>>8)`，與這裡相同；不透明的像素不變。16 位元 PNG 這裡先去掉
/// 低位元組再預乘，Go 在 16 位元預乘後才取高位元組，半透明時可能差 1（原版美術模式沒有 16 位元 PNG，未實測）。
pub(crate) fn decode_png_picture(data: &[u8]) -> Option<Picture> {
    decode_png(data, PngAlpha::Legacy)
}

pub(crate) fn decode_opaque_png_picture(data: &[u8]) -> Option<Picture> {
    decode_png(data, PngAlpha::Opaque)
}

/// 對應 Go 先預乘 16 位元通道，再寫入 8 位元 RGBA。
pub(crate) fn decode_precise_png_picture(data: &[u8]) -> Option<Picture> {
    decode_png(data, PngAlpha::Precise)
}

#[derive(PartialEq, Eq)]
enum PngAlpha {
    Legacy,
    Opaque,
    Precise,
}

fn decode_png(data: &[u8], alpha: PngAlpha) -> Option<Picture> {
    let mut decoder = png::Decoder::new(std::io::Cursor::new(data));
    let mut transformations = png::Transformations::EXPAND | png::Transformations::ALPHA;
    if alpha == PngAlpha::Legacy {
        transformations |= png::Transformations::STRIP_16;
    }
    decoder.set_transformations(transformations);
    let mut reader = decoder.read_info().ok()?;
    let mut buf = vec![0u8; reader.output_buffer_size()?];
    let info = reader.next_frame(&mut buf).ok()?;
    let (w, h) = (info.width as usize, info.height as usize);
    let mut pix = Vec::with_capacity(w * h * 4);
    if info.bit_depth == png::BitDepth::Sixteen {
        // 完全不透明需要 0xffff。先去掉低位元組會把 0xff00 誤認成不透明。
        let channels = match info.color_type {
            png::ColorType::Rgba => 4,
            png::ColorType::GrayscaleAlpha => 2,
            _ => return None,
        };
        let bytes_per_pixel = channels * 2;
        for px in buf[..info.buffer_size()].chunks_exact(bytes_per_pixel) {
            let channel = |i: usize| u32::from(u16::from_be_bytes([px[2 * i], px[2 * i + 1]]));
            let a = channel(channels - 1);
            if alpha == PngAlpha::Opaque && a != 65535 {
                return None;
            }
            let scaled = |i| ((channel(i) * a / 65535) >> 8) as u8;
            let rgb = if channels == 4 { [scaled(0), scaled(1), scaled(2)] } else { [scaled(0); 3] };
            pix.extend_from_slice(&[rgb[0], rgb[1], rgb[2], (a >> 8) as u8]);
        }
        return Some(Picture { width: w as i64, height: h as i64, pix });
    }
    match info.color_type {
        png::ColorType::Rgba => pix.extend_from_slice(&buf[..w * h * 4]),
        png::ColorType::GrayscaleAlpha => {
            for px in buf[..w * h * 2].as_chunks::<2>().0 {
                pix.extend_from_slice(&[px[0], px[0], px[0], px[1]]);
            }
        }
        _ => return None,
    }
    for px in pix.as_chunks_mut::<4>().0 {
        let a = px[3];
        if a != 255 {
            if alpha == PngAlpha::Opaque {
                return None;
            }
            for v in &mut px[..3] {
                *v = premultiply_channel(*v, a);
            }
        }
    }
    Some(Picture { width: w as i64, height: h as i64, pix })
}
