//! 移植自 `native_hotel_route_test.go`。

use crate::fdother::*;

// Go: TestResolveNativeHotelServiceRoutePreservesRawBranches
#[test]
fn resolve_native_hotel_service_route_preserves_raw_branches() {
    for (selector, primary, secondary) in [(0, 0x2ffa5, 0), (1, 0x30012, 0), (2, 0x301f4, 0), (3, 0x19953, 0x197e5)] {
        let got = resolve_native_hotel_service_route(selector);
        assert_eq!(
            got,
            Some(NativeHotelServiceRoute { selector, resource_id: 13, primary, secondary }),
            "route({selector})"
        );
    }
    assert_eq!(resolve_native_hotel_service_route(4), None, "invalid selector accepted");
}
