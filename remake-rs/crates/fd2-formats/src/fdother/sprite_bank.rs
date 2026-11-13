//! 移植自 Go `sprite_bank.go`：FDSHAP 偶數資源的 24×24 四模式 B24 圖庫與地形控制表配對。

use super::{FdotherError, Result, read_resource_i64};
use crate::fdicon::{self, Bank};
use std::path::Path;

/// 載入一筆內容是原生 24x24 四模式 B24 排列的 LLLLLL 封存資源。FDSHAP 的偶數資源是這種
/// 形式；地圖與有證據的資源編號如何配對、相鄰控制表如何分開，都歸呼叫端負責。
pub fn decode_sprite_bank_resource(dat_path: impl AsRef<Path>, resource: i64) -> Result<Bank> {
    Ok(fdicon::parse(&read_resource_i64(dat_path.as_ref(), resource)?)?)
}

/// 載入一張地圖有證據的 FDSHAP 配對：圖庫 #2N 與相鄰的每格四位元組控制表 #2N+1。地圖
/// 索引必須來自明確的戰役／地圖綁定；圖片數量與正規化地圖資料都不得用來猜它。
pub fn decode_map_terrain_resources(dat_path: impl AsRef<Path>, map_index: i64) -> Result<(Bank, Vec<u8>)> {
    if map_index < 0 {
        return Err(FdotherError::fixed("fdother: negative FDSHAP map index"));
    }
    let dat_path = dat_path.as_ref();
    let resource = map_index.wrapping_mul(2);
    let bank = decode_sprite_bank_resource(dat_path, resource)?;
    let controls = read_resource_i64(dat_path, resource.wrapping_add(1))?;
    // 有些圖庫在基本控制紀錄之後帶有繪製端選用的尾端動畫影格（map17 是 384 張圖／330 筆
    // 紀錄）。地圖合成驗證基本圖塊索引；繪製端另外驗證衍生的影格索引。
    if controls.is_empty() || !controls.len().is_multiple_of(4) {
        return Err(FdotherError::fixed("fdother: FDSHAP image/control resource pair is inconsistent"));
    }
    Ok((bank, controls))
}
