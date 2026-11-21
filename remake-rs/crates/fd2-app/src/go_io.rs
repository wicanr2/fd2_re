//! Go `io.Reader`／`io.Seeker` 的呼叫契約與 `bytes.Reader`，給 [`crate::go_oggvorbis`] 與
//! [`crate::go_ebiten_audio`] 的逐行移植使用（R5 階段 4，代號 r5v）。
//!
//! 這不是 Go 標準函式庫的移植（`rust-port-no-go-stdlib`），只把解碼鏈依賴的契約寫成 Rust 型別：
//!
//! - `Read` 同時回傳讀到的位元組數與錯誤（Go `(n int, err error)`），`n > 0` 時也可以帶 `EOF`。oggvorbis 與
//!   Ebiten 的解碼鏈依這個契約決定何時停、何時補零與何時重新定位，換成 `std::io::Read` 的「錯誤時沒有資料」
//!   會改變樣本數。
//! - `bytes.Reader` 的 `Read`（讀到尾端回 `EOF`）、`Seek`（位置可以超過尾端；負位置回錯誤且不移動）。
//! - `io.ReadAtLeast`／`io.ReadFull` 的錯誤轉換（讀到一部分就遇到 `EOF` 時改成 `ErrUnexpectedEOF`）。
//!
//! 錯誤文字照 Go（`EOF`、`unexpected EOF`、`multiple Read calls return no data or error`）。

use std::sync::Arc;

/// Go `io` 的錯誤值。
#[derive(Debug, Clone, PartialEq, Eq)]
pub enum IoError {
    /// `io.EOF`。
    Eof,
    /// `io.ErrUnexpectedEOF`。
    UnexpectedEof,
    /// `io.ErrNoProgress`。
    NoProgress,
    /// 其他錯誤（`errors.New`／`fmt.Errorf` 的文字）。
    Other(String),
}

impl IoError {
    pub fn other(msg: impl Into<String>) -> IoError {
        IoError::Other(msg.into())
    }
}

impl std::fmt::Display for IoError {
    fn fmt(&self, f: &mut std::fmt::Formatter<'_>) -> std::fmt::Result {
        match self {
            IoError::Eof => f.write_str("EOF"),
            IoError::UnexpectedEof => f.write_str("unexpected EOF"),
            IoError::NoProgress => f.write_str("multiple Read calls return no data or error"),
            IoError::Other(s) => f.write_str(s),
        }
    }
}

impl std::error::Error for IoError {}

/// Go `io.SeekStart`／`io.SeekCurrent`／`io.SeekEnd`。
#[derive(Debug, Clone, Copy, PartialEq, Eq)]
pub enum Whence {
    Start,
    Current,
    End,
}

/// Go `io.Reader`：回傳 `(n, err)`。
pub trait Read {
    fn read(&mut self, p: &mut [u8]) -> (usize, Option<IoError>);
}

/// Go `io.Seeker`。
pub trait Seek {
    fn seek(&mut self, offset: i64, whence: Whence) -> Result<i64, IoError>;
}

/// Go `io.ReadSeeker`（解碼鏈要在 Bevy 的音訊執行緒上讀，所以要 `Send`）。
pub trait ReadSeek: Read + Seek + Send {}

impl<T: Read + Seek + Send> ReadSeek for T {}

impl<T: Read + ?Sized> Read for Box<T> {
    fn read(&mut self, p: &mut [u8]) -> (usize, Option<IoError>) {
        (**self).read(p)
    }
}

impl<T: Seek + ?Sized> Seek for Box<T> {
    fn seek(&mut self, offset: i64, whence: Whence) -> Result<i64, IoError> {
        (**self).seek(offset, whence)
    }
}

/// Go `bytes.Reader`（只有 `Read` 與 `Seek`）。
#[derive(Clone)]
pub struct BytesReader {
    s: Arc<[u8]>,
    i: i64,
}

impl BytesReader {
    pub fn new(s: Arc<[u8]>) -> BytesReader {
        BytesReader { s, i: 0 }
    }
}

impl Read for BytesReader {
    fn read(&mut self, b: &mut [u8]) -> (usize, Option<IoError>) {
        if self.i >= self.s.len() as i64 {
            return (0, Some(IoError::Eof));
        }
        let rest = &self.s[self.i as usize..];
        let n = rest.len().min(b.len());
        b[..n].copy_from_slice(&rest[..n]);
        self.i += n as i64;
        (n, None)
    }
}

impl Seek for BytesReader {
    fn seek(&mut self, offset: i64, whence: Whence) -> Result<i64, IoError> {
        let abs = match whence {
            Whence::Start => offset,
            Whence::Current => self.i + offset,
            Whence::End => self.s.len() as i64 + offset,
        };
        if abs < 0 {
            return Err(IoError::other("bytes.Reader.Seek: negative position"));
        }
        self.i = abs;
        Ok(abs)
    }
}

/// Go `io.ReadAtLeast`。
pub fn read_at_least<R: Read + ?Sized>(r: &mut R, buf: &mut [u8], min: usize) -> (usize, Option<IoError>) {
    if buf.len() < min {
        return (0, Some(IoError::other("short buffer")));
    }
    let (mut n, mut err) = (0usize, None);
    while n < min && err.is_none() {
        let (nn, e) = r.read(&mut buf[n..]);
        n += nn;
        err = e;
    }
    if n >= min {
        err = None;
    } else if n > 0 && err == Some(IoError::Eof) {
        err = Some(IoError::UnexpectedEof);
    }
    (n, err)
}

/// Go `io.ReadFull`。
pub fn read_full<R: Read + ?Sized>(r: &mut R, buf: &mut [u8]) -> (usize, Option<IoError>) {
    let min = buf.len();
    read_at_least(r, buf, min)
}

/// Go `io.ReadAll` 的結果：讀到 `EOF` 為止；其他錯誤連同已讀資料一起回傳。讀取大小不影響本專案解碼鏈的
/// 輸出（見 [`crate::go_ebiten_audio`] 模組文件），所以這裡固定每次讀 16 KiB，不照 Go 的倍增緩衝。
pub fn read_all<R: Read + ?Sized>(r: &mut R) -> (Vec<u8>, Option<IoError>) {
    let mut out = Vec::new();
    let mut chunk = vec![0u8; 16384];
    loop {
        let (n, err) = r.read(&mut chunk);
        out.extend_from_slice(&chunk[..n]);
        match err {
            None => {}
            Some(IoError::Eof) => return (out, None),
            Some(e) => return (out, Some(e)),
        }
    }
}
