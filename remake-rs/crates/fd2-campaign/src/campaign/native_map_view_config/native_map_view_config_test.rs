//! Go `remake/internal/campaign/native_map_view_config_test.go` 的逐條移植。

use super::*;

fn view(camera_x: i64, camera_y: i64, cursor_x: i64, cursor_y: i64, vx: i64, vy: i64) -> NativeMapViewConfig {
    NativeMapViewConfig {
        camera_x,
        camera_y,
        cursor_x,
        cursor_y,
        visible_cursor_x: vx,
        visible_cursor_y: vy,
        range_mode: None,
    }
}

/// Go：`TestNativeMapViewConfigEntryContract`
///
/// 釘住「節點常數」與「執行期狀態」是兩套契約。執行期的可見游標不被夾（原版八個寫入端都不檢查
/// 範圍，走行捲動會把它寫到視窗外）；節點常數是進場即繪的靜止視圖，游標框與指令環立刻消費它，
/// 所以要求視窗內、而且與 cursor-camera 一致。
#[test]
fn native_map_view_config_entry_contract() {
    const W: i64 = 24;
    const H: i64 = 24;
    let good = view(1, 13, 8, 17, 7, 4);
    if let Err(err) = good.validate(W, H) {
        panic!("拒絕了合法的入口視圖：{err}");
    }
    for (name, v) in [
        ("可見游標出界", view(1, 13, 8, 17, 13, 4)),
        ("可見游標為負", view(1, 13, 8, 12, 7, -1)),
        ("可見游標與 cursor-camera 不一致", view(1, 13, 8, 17, 7, 3)),
        ("鏡頭越界", view(1, H - 7, 8, 18, 7, 1)),
        ("游標出場", view(1, 13, W, 17, W - 1, 4)),
    ] {
        assert!(v.validate(W, H).is_err(), "{name}：入口視圖 {v:?} 被接受了");
    }
    assert!(good.validate(12, 24).is_err(), "接受了小於 13×8 視窗的場地");
}

/// Go：`TestVersionedNodeViewsSatisfyTheEntryContract`
///
/// 讓受版控的節點常數自己驗一次。
#[test]
fn versioned_node_views_satisfy_the_entry_contract() {
    let Some(path) = crate::test_support::remake_asset("scenarios/campaign_full.json") else {
        return;
    };
    let data = load(&path).unwrap();
    let mut checked = 0;
    for (id, node) in &data.nodes {
        let Some(view) = node.native_map_view else {
            continue;
        };
        checked += 1;
        assert!(
            view.visible_cursor_x == view.cursor_x - view.camera_x
                && view.visible_cursor_y == view.cursor_y - view.camera_y
                && (0..13).contains(&view.visible_cursor_x)
                && (0..8).contains(&view.visible_cursor_y),
            "節點 {id} 的入口視圖不符契約：{view:?}"
        );
    }
    assert!(checked != 0, "沒有節點帶 native_map_view，這個檢查等於沒跑");
}
