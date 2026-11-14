//! Go `remake/internal/editorcanonical/compile_test.go` 的逐條移植。

use super::*;
use serde_json::Value;

fn bundle_root() -> Option<String> {
    crate::test_support::remake_asset("editor-canonical")
}

fn legacy_path() -> Option<String> {
    crate::test_support::remake_asset("scenarios/campaign_full.json")
}

/// Go：`TestCompiledCampaignEqualsLegacyDocument`
///
/// 編譯結果要和 legacy 原檔在 JSON 值域上完全相同。這是整條編輯器路徑唯一有意義
/// 的驗收：只驗「編得出東西」或「節點數一樣」都會放過欄位遺失，而遺失的欄位要玩到
/// 那個節點才會發作。
#[test]
fn compiled_campaign_equals_legacy_document() {
    let (Some(root), Some(legacy_path)) = (bundle_root(), legacy_path()) else {
        return;
    };
    let compiled = compile_campaign_json(&root).unwrap_or_else(|err| panic!("編譯 canonical 戰役: {err}"));
    let legacy = std::fs::read(&legacy_path).unwrap_or_else(|err| panic!("讀 legacy 戰役: {err}"));
    let from_canonical: Value = serde_json::from_slice(&compiled).unwrap_or_else(|err| panic!("解析編譯結果: {err}"));
    let from_legacy: Value = serde_json::from_slice(&legacy).unwrap_or_else(|err| panic!("解析 legacy 戰役: {err}"));
    if from_canonical == from_legacy {
        return;
    }
    // 不相等時把差異指到具體節點與欄位；只說「不相等」的失敗訊息沒辦法動手修。
    let mut errors = Vec::new();
    let empty = serde_json::Map::new();
    let left = from_canonical.as_object().unwrap_or(&empty);
    let right = from_legacy.as_object().unwrap_or(&empty);
    for (key, want) in right {
        if left.get(key) == Some(want) {
            continue;
        }
        if key != "nodes" {
            errors.push(format!("頂層欄位 {key:?} 不同"));
            continue;
        }
        let left_nodes = left.get(key).and_then(Value::as_object).unwrap_or(&empty);
        let right_nodes = want.as_object().unwrap_or(&empty);
        let mut reported = 0;
        for (node, value) in right_nodes {
            if left_nodes.get(node) == Some(value) {
                continue;
            }
            reported += 1;
            if reported > 5 {
                errors.push(format!("還有更多節點不同，共 {} 個", right_nodes.len()));
                break;
            }
            errors.push(format!("節點 {node:?} 不同：\n編譯={:?}\nlegacy={value}", left_nodes.get(node)));
        }
    }
    for key in left.keys() {
        if !right.contains_key(key) {
            errors.push(format!("編譯結果多出頂層欄位 {key:?}"));
        }
    }
    panic!("{}", errors.join("\n"));
}

/// Go：`TestCompiledCampaignDecodesAndMatchesLegacyGraph`
///
/// 編譯結果要能走完 campaign 自己那道轉場驗證，而不是只有 JSON 長得像；順帶守住
/// canonical 與 legacy 不漂移——任何一邊被單獨改到，就會出現「編輯器看到的」與
/// 「遊戲跑的」不是同一份，而那種差異要玩到那個節點才會發作。
#[test]
fn compiled_campaign_decodes_and_matches_legacy_graph() {
    let (Some(root), Some(legacy_path)) = (bundle_root(), legacy_path()) else {
        return;
    };
    let compiled = compile_campaign_json(&root).unwrap_or_else(|err| panic!("編譯 canonical 戰役: {err}"));
    let from_canonical =
        crate::campaign::decode(&compiled).unwrap_or_else(|err| panic!("編譯結果過不了戰役驗證: {err}"));
    let from_legacy = crate::campaign::load(&legacy_path).unwrap_or_else(|err| panic!("讀 legacy 戰役: {err}"));
    assert_eq!(
        from_canonical.start, from_legacy.start,
        "起點 {:?}，legacy 是 {:?}",
        from_canonical.start, from_legacy.start
    );
    assert_eq!(
        from_canonical.nodes.len(),
        from_legacy.nodes.len(),
        "節點數 {}，legacy 是 {}",
        from_canonical.nodes.len(),
        from_legacy.nodes.len()
    );
    if from_canonical != from_legacy {
        for (id, want) in &from_legacy.nodes {
            let got = from_canonical.nodes.get(id).unwrap_or_else(|| panic!("編譯結果缺節點 {id:?}"));
            assert!(got == want, "節點 {id:?} 的型別化內容不同");
        }
        panic!("戰役圖不同，但逐節點比對沒找到差異；改了 legacy 就要重跑 tools/export_editor_canonical.py");
    }
}

/// Go：`TestCompileRefusesBrokenBundles`
///
/// 壞掉的 bundle 要說得出壞在哪，不能編出一份少了東西卻看起來正常的戰役。
#[test]
fn compile_refuses_broken_bundles() {
    let Some(root) = bundle_root() else {
        return;
    };
    let source = std::fs::read(std::path::Path::new(&root).join("campaign").join("campaign_full.json"))
        .unwrap_or_else(|err| panic!("讀 canonical 戰役: {err}"));
    let _document: Value = serde_json::from_slice(&source).unwrap_or_else(|err| panic!("解析 canonical 戰役: {err}"));

    type Mutate = fn(&mut Value);
    let cases: [(&str, Mutate); 5] = [
        ("schema 版本不對", |d| d["schema_version"] = serde_json::json!(2)),
        ("kind 不對", |d| d["kind"] = serde_json::json!("scenario")),
        ("沒有節點", |d| d["nodes"] = serde_json::json!([])),
        ("節點少了 legacy_key", |d| {
            d["nodes"][0]["extensions"].as_object_mut().unwrap().remove("legacy_key");
        }),
        ("node_id 重複", |d| {
            let first = d["nodes"][0]["node_id"].clone();
            d["nodes"][1]["node_id"] = first;
        }),
    ];
    for (name, mutate) in cases {
        let mut copied: Value = serde_json::from_slice(&source).unwrap_or_else(|err| panic!("{name}：複製文件: {err}"));
        mutate(&mut copied);
        let root = crate::test_support::temp_dir("compile-broken");
        std::fs::create_dir_all(root.join("campaign")).unwrap_or_else(|err| panic!("{name}：建立目錄: {err}"));
        let encoded = serde_json::to_vec(&copied).unwrap_or_else(|err| panic!("{name}：編碼文件: {err}"));
        std::fs::write(root.join("campaign").join("campaign_full.json"), encoded)
            .unwrap_or_else(|err| panic!("{name}：寫入文件: {err}"));
        assert!(compile_campaign_json(&root.to_string_lossy()).is_err(), "{name}：壞掉的 bundle 編出了結果");
    }

    assert!(compile_campaign_json("").is_err(), "空路徑編出了結果");
    let empty = crate::test_support::temp_dir("compile-empty");
    assert!(compile_campaign_json(&empty.to_string_lossy()).is_err(), "空目錄編出了結果");
}

/// Go：`TestCompileAcceptsTheShippedBundle`
///
/// 正對照：完整 bundle 在同一個流程下必須成功。沒有這一條，上面那些「壞掉會失敗」
/// 也可能只是因為流程本身根本不會成功。
#[test]
fn compile_accepts_the_shipped_bundle() {
    let Some(root) = bundle_root() else {
        return;
    };
    let compiled = compile_campaign_json(&root).unwrap_or_else(|err| panic!("完整 bundle 編不出結果: {err}"));
    assert!(!compiled.is_empty(), "編譯結果是空的");
}
