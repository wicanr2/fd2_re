//! Go 來源：`remake/internal/ending/dato_frame.go`（20 行）。

use super::*;

/// RenderDATOFrameAt executes the opaque 0x4e8af paste only. Native callers
/// provide the staging destination (typically staging+[0x53c67]); the helper
/// keeps that offset explicit so it cannot silently become a guessed anchor.
///
/// Go：`dato_frame.go:12`，`func RenderDATOFrameAt(dst []byte, frame dato.Frame, stride, destinationOffset int) error`
pub fn render_dato_frame_at(dst: &mut [u8], frame: &dato::Frame, stride: i64, destination_offset: i64) -> Result<()> {
    if dst.len() as i64 != BYTES {
        return Err(campaign_err!("ending: DATO destination must be {BYTES} bytes"));
    }
    frame.blit_at_offset(dst, stride, destination_offset).map_err(|err| {
        CampaignError::from(err).wrap(format!("ending: DATO frame at {}: ", battle::go_sharp_x(destination_offset)))
    })
}
