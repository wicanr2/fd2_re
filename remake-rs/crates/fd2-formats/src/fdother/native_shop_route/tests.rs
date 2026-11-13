//! 移植自 `native_shop_route_test.go`。

use crate::fdother::*;

// Go: TestResolveNativeShopServiceRoutePreservesResourceAndCallee
#[test]
fn resolve_native_shop_service_route_preserves_resource_and_callee() {
    let checks = [
        (0, 12, 0, 0x2f0b0, NativeShopServiceKind::Purchase),
        (3, 29, 1, 0x2f642, NativeShopServiceKind::Sell),
        (5, 63, 2, 0x2f883, NativeShopServiceKind::Equip),
        (7, 12, 3, 0x2f8ea, NativeShopServiceKind::Transfer),
    ];
    for (variant, resource, selector, callee, kind) in checks {
        assert_eq!(
            resolve_native_shop_service_route(variant, selector),
            Some(NativeShopServiceRoute { hub_variant: variant, resource_id: resource, selector, callee, kind }),
            "route({variant},{selector})"
        );
    }
    assert_eq!(resolve_native_shop_service_route(0, 4), None, "invalid selector accepted");
}
