//! 移植自 `remake/internal/battle/native_ch28_post_present_test.go`。

use super::*;

fn native_ch28_post_presenter_fixture() -> State {
    let u = |hp: i64, x: u8, y: u8, byte5: u8| Unit {
        hp,
        native_map_presentation: NativeMapPresentationState { x, y, ..Default::default() },
        has_native_map_presentation: true,
        native_record_byte5: byte5,
        has_native_record_byte5: true,
        ..Default::default()
    };
    State { units: vec![u(0, 9, 7, 0), u(0, 23, 17, 0), u(1, 10, 8, 0), u(0, 10, 8, 1)], ..Default::default() }
}

/// Go：`TestPlanNativeCh28PostPresentationUsesExactViewportGate`
#[test]
fn plan_native_ch28_post_presentation_uses_exact_viewport_gate() {
    let mut st = native_ch28_post_presenter_fixture();
    let plan = plan_native_ch28_post_presentation(Some(&st), 10, 8).unwrap();
    assert!(
        plan.targets.len() == 2 && plan.targets[0].slot == 0 && plan.targets[1].slot == 1,
        "targets={:?} want boundary slots 0,1",
        plan.targets
    );
    st.units[1].native_map_presentation.x += 1;
    let plan = plan_native_ch28_post_presentation(Some(&st), 10, 8);
    assert!(
        matches!(&plan, Ok(p) if p.targets.len() == 1 && p.targets[0].slot == 0),
        "outside right boundary targets={plan:?}"
    );
}

/// Go：`TestNativeCh28PostPoseAndInactiveWritesFollowRawGates`
#[test]
fn native_ch28_post_pose_and_inactive_writes_follow_raw_gates() {
    let mut st = native_ch28_post_presenter_fixture();
    apply_native_ch28_post_pose_frame(Some(&mut st), 11).unwrap();
    let pose: Vec<u8> = st.units.iter().map(|u| u.native_map_presentation.pose).collect();
    assert_eq!(pose, [3, 3, 0, 0], "pose write mismatch: {:?}", st.units);
    apply_native_ch28_post_inactive_mark(Some(&mut st)).unwrap();
    let byte5: Vec<u8> = st.units.iter().map(|u| u.native_record_byte5).collect();
    assert_eq!(byte5, [1, 1, 0, 1], "inactive writes mismatch: {:?}", st.units);
}

/// Go：`TestNativeCh28PostPresenterRejectsMissingProvenanceAtomically`
#[test]
fn native_ch28_post_presenter_rejects_missing_provenance_atomically() {
    let mut st = native_ch28_post_presenter_fixture();
    st.units[2].has_native_map_presentation = false;
    assert!(plan_native_ch28_post_presentation(Some(&st), 10, 8).is_err(), "missing consumed raw fields were accepted");
    let before = st.units[0].native_map_presentation.pose;
    st.units[2].has_native_map_presentation = true;
    st.units[1].has_native_record_byte5 = false;
    assert!(apply_native_ch28_post_pose_frame(Some(&mut st), 2).is_err(), "missing pose provenance was accepted");
    assert_eq!(st.units[0].native_map_presentation.pose, before, "failed pose frame partially mutated state");
}
