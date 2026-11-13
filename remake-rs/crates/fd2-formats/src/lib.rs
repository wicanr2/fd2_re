//! 《炎龍騎士團 2》原版檔案格式。
//!
//! 每個模組對應凍結 Go 版 `remake/internal/` 的一個格式套件，行為以 Go 版的單元測試與
//! 黃金差分資料驗證；原版位址與證據等級沿用 Go 版註解及 `docs/data/`。

pub mod afm;
pub mod archive;
pub mod dato;
pub mod fdicon;
pub mod fdother;
pub mod fdtxt;
pub mod figani;
pub mod keyboard;
pub mod native_rng;
pub mod save;
