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
    if !(0..NATIVE_TOWN_VARIANT_COUNT).contains(&variant) || !(0..6).contains(&selection) || !(1..=10).contains(&step) {
        return Err(campaign_err!("campaign: native town departure geometry unavailable"));
    }
    let (v, s) = (variant as usize, selection as usize);
    // Go 整數除法向零截斷，與 Rust `/` 相同。
    Ok(NativeTownDepartureGeometry {
        center: go_image::Point {
            x: 128 * (160 + (NATIVE_TOWN_SELECTION_X[v][s] - 150) * step / 10),
            y: 128 * (100 + (NATIVE_TOWN_SELECTION_Y[v][s] - 100) * step / 10),
        },
        spacing: 128 - 9 * step,
    })
}

/// ComposeNativeTownDepartureFrame follows sub_2D093→sub_2FB9F. Coordinates
/// remain in 1/128 source pixels; signed IDIV truncates before multiplication.
/// Evidence: docs/data/ida/fd2_town_departure_20261005.json.
///
/// Go：`native_town_departure.go:28`，`func ComposeNativeTownDepartureFrame(source []byte, variant, selection, step int) ([]byte, error)`
pub fn compose_native_town_departure_frame(source: &[u8], variant: i64, selection: i64, step: i64) -> Result<Vec<u8>> {
    let geometry = native_town_departure_geometry_for(variant, selection, step);
    let geometry = match geometry {
        Ok(g) if source.len() as i64 == NATIVE_TOWN_WIDTH * NATIVE_TOWN_HEIGHT => g,
        _ => return Err(campaign_err!("campaign: native town departure source or step unavailable")),
    };
    let (spacing, cx, cy) = (geometry.spacing, geometry.center.x, geometry.center.y);
    let mut frame = vec![0u8; (NATIVE_TOWN_WIDTH * NATIVE_TOWN_HEIGHT) as usize];
    for y in 0..NATIVE_TOWN_HEIGHT {
        let sy = cy - 100 * spacing + y * spacing;
        if !(0..NATIVE_TOWN_HEIGHT * 128).contains(&sy) {
            continue;
        }
        for x in 0..NATIVE_TOWN_WIDTH {
            let sx = cx - 160 * spacing + x * spacing;
            if (0..NATIVE_TOWN_WIDTH * 128).contains(&sx) {
                frame[(y * NATIVE_TOWN_WIDTH + x) as usize] =
                    source[((sy / 128) * NATIVE_TOWN_WIDTH + sx / 128) as usize];
            }
        }
    }
    Ok(frame)
}

#[cfg(test)]
pub(crate) mod native_town_departure_test;
