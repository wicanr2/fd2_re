//! 與凍結 Go 版 `remake/cmd/fd2-golden-gorand` 的黃金差分：Go `math/rand` 相容序列。
//!
//! 由 `tools/rust_golden_gorand.sh` 產生黃金 JSON 並設定 `FD2_GOLDEN_GORAND`。設定
//! `FD2_REQUIRE_GOLDEN=1` 時，黃金檔缺少即失敗，避免把「沒跑到」當成通過。
//!
//! 比對方式：每個種子的 `Int63` 前 4 個值與 `Intn` 交錯序列、每個 (seed, n) 的 `Intn` 序列逐值相同。
//! 沒有刻意不照做的項目（`KNOWN_DIFFERENCES` 為空）。

use fd2_rules::battle::{GoMathRand, GoRand};
use serde_json::Value;

/// 刻意不照做 Go 的項目。這份差分沒有。
const KNOWN_DIFFERENCES: &[&str] = &[];

fn golden() -> Option<Value> {
    let require = std::env::var_os("FD2_REQUIRE_GOLDEN").is_some_and(|v| v == "1");
    let path = std::env::var_os("FD2_GOLDEN_GORAND");
    let Some(path) = path.filter(|p| std::path::Path::new(p).exists()) else {
        assert!(!require, "FD2_REQUIRE_GOLDEN=1 但沒有 FD2_GOLDEN_GORAND");
        eprintln!("略過：沒有 FD2_GOLDEN_GORAND");
        return None;
    };
    Some(serde_json::from_slice(&std::fs::read(path).unwrap()).unwrap())
}

fn ints(v: &Value) -> Vec<i64> {
    v.as_array().unwrap().iter().map(|x| x.as_i64().unwrap()).collect()
}

#[test]
fn golden_gorand() {
    let Some(doc) = golden() else { return };
    assert_eq!(doc["schema"], "fd2-golden-gorand/1");
    assert!(KNOWN_DIFFERENCES.is_empty());
    let ns = ints(&doc["ns"]);
    let mut checked = 0usize;
    for (index, case) in doc["mixed"].as_array().unwrap().iter().enumerate() {
        let seed = case["seed"].as_i64().unwrap();
        let mut r = GoMathRand::new(seed);
        let int63: Vec<i64> = (0..4).map(|_| r.int63()).collect();
        assert_eq!(int63, ints(&case["int63"]), "mixed case {index} seed {seed} Int63");
        let want = ints(&case["intn"]);
        for (i, &w) in want.iter().enumerate() {
            let n = ns[i % ns.len()];
            assert_eq!(r.intn(n), w, "mixed case {index} seed {seed} draw {i} Intn({n})");
        }
        checked += 1 + want.len();
    }
    for (index, case) in doc["per_n"].as_array().unwrap().iter().enumerate() {
        let (seed, n) = (case["seed"].as_i64().unwrap(), case["n"].as_i64().unwrap());
        let mut r = GoMathRand::new(seed);
        let got: Vec<i64> = (0..case["values"].as_array().unwrap().len()).map(|_| r.intn(n)).collect();
        assert_eq!(got, ints(&case["values"]), "per_n case {index} seed {seed} Intn({n})");
        checked += got.len();
    }
    eprintln!(
        "gorand 黃金差分：mixed {} 案例、per_n {} 案例、{checked} 個值相符",
        doc["mixed"].as_array().unwrap().len(),
        doc["per_n"].as_array().unwrap().len()
    );
}
