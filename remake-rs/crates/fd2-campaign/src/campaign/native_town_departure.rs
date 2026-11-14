//! Go 來源：`remake/internal/campaign/native_town_departure.go`（48 行）。

use super::*;

/// NativeTownDepartureGeometry retains the original 1/128-pixel source space.
///
/// Go：`native_town_departure.go:9`
#[derive(Debug, Clone, Copy, Default, PartialEq, Eq)]
pub struct NativeTownDepartureGeometry {
    pub center: go_image::Point,
    pub spacing: i64,
}

/// Go：`native_town_departure.go:14`，`func NativeTownDepartureGeometryFor(variant, selection, step int) (NativeTownDepartureGeometry, error)`
pub fn native_town_departure_geometry_for(
    variant: i64,
    selection: i64,
    step: i64,
) -> Result<NativeTownDepartureGeometry> {
    todo!("campaign/native_town_departure.go:NativeTownDepartureGeometryFor")
}

/// ComposeNativeTownDepartureFrame follows sub_2D093→sub_2FB9F. Coordinates
/// remain in 1/128 source pixels; signed IDIV truncates before multiplication.
/// Evidence: docs/data/ida/fd2_town_departure_20261005.json.
///
/// Go：`native_town_departure.go:28`，`func ComposeNativeTownDepartureFrame(source []byte, variant, selection, step int) ([]byte, error)`
pub fn compose_native_town_departure_frame(source: &[u8], variant: i64, selection: i64, step: i64) -> Result<Vec<u8>> {
    todo!("campaign/native_town_departure.go:ComposeNativeTownDepartureFrame")
}
