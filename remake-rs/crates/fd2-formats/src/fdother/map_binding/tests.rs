//! 移植自 `map_binding_test.go`。

use crate::fdother::*;

// Go: TestMapIndexFromAssetPathRequiresExactMapBinding
#[test]
fn map_index_from_asset_path_requires_exact_map_binding() {
    for (path, want) in [("assets", 0), ("assets/maps/map0", 0), ("assets/maps/map32", 32)] {
        assert_eq!(map_index_from_asset_path(path), Ok(want), "{path:?}");
    }
    for path in ["assets/maps/map", "assets/maps/map01x", "assets/maps/map-1", "assets/maps/other"] {
        assert!(map_index_from_asset_path(path).is_err(), "{path:?} accepted");
    }
}
