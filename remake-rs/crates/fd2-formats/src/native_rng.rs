//! 原版 `0x4e893` 的 16 位元狀態轉移。
//!
//! 移植自 Go `remake/internal/fdother/native_rng.go`。與存檔外層 `0x4dbd8` 用的是同一個轉移式；
//! FIGANI 與成長流程共用，因此獨立成模組。

/// `state = rol16(state + 0x9014, 3)`。原版呼叫端取除以 4 的餘數當原始標記來源，不取商。
pub fn native_rng_step(state: u16) -> u16 {
    state.wrapping_add(0x9014).rotate_left(3)
}

/// 成長流程呼叫端在 idiv 4 之後產生的標記位元組：EDX 餘數 + 2。
pub fn native_rng_marker(state: u16) -> u8 {
    (native_rng_step(state) % 4 + 2) as u8
}

#[cfg(test)]
mod tests {
    use super::*;

    // Go: TestNativeRNGStepUses16BitAddAndRotate
    #[test]
    fn step_uses_16bit_add_and_rotate() {
        assert_eq!(native_rng_step(0), 0x80a4);
        assert_eq!(native_rng_step(0x1234), 0x1245);
    }

    // Go: TestNativeRNGMarkerUsesRemainderNotQuotient
    #[test]
    fn marker_uses_remainder_not_quotient() {
        assert_eq!(native_rng_marker(0), 2);
        assert_eq!(native_rng_marker(0x1234), 3);
    }
}
