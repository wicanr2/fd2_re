#!/usr/bin/env python3
"""驗證第一輪 remake 的分層抽樣清冊，並計算95%完成門檻。"""

from __future__ import annotations

import argparse
import json
import math
from pathlib import Path
import sys


LAYERS = {
    "campaign_intermission": 12,
    "combat_ai": 18,
    "ui": 12,
    "save_party": 10,
    "ending_platform": 8,
}
LEVELS = {"RUNTIME-E1", "PLAYER-E2", "E1+E2", "candidate"}
RESULTS = {"pass", "pending", "fail"}


def load_registry(path: Path) -> dict:
    return json.loads(path.read_text(encoding="utf-8"))


def validate(registry: dict, root: Path) -> tuple[list[str], dict[str, int], int]:
    errors: list[str] = []
    counts = {layer: 0 for layer in LAYERS}
    seen: set[str] = set()
    samples = registry.get("samples")
    if registry.get("schema_version") != 1:
        errors.append("schema_version 必須為 1")
    if registry.get("confidence_target") != 0.95:
        errors.append("confidence_target 必須為 0.95")
    if registry.get("minimum_total") != sum(LAYERS.values()):
        errors.append(f"minimum_total 必須為 {sum(LAYERS.values())}")
    if registry.get("layer_minimums") != LAYERS:
        errors.append("layer_minimums 與檢查器固定配額不一致")
    if not isinstance(samples, list):
        return errors + ["samples 必須為陣列"], counts, 0

    for index, sample in enumerate(samples):
        where = f"samples[{index}]"
        if not isinstance(sample, dict):
            errors.append(f"{where} 必須為物件")
            continue
        sample_id = sample.get("id")
        if not isinstance(sample_id, str) or not sample_id:
            errors.append(f"{where}.id 缺失")
        elif sample_id in seen:
            errors.append(f"{where}.id 重複：{sample_id}")
        else:
            seen.add(sample_id)

        layer = sample.get("layer")
        if layer not in LAYERS:
            errors.append(f"{where}.layer 非法：{layer}")
        result = sample.get("result")
        if result not in RESULTS:
            errors.append(f"{where}.result 非法：{result}")
        level = sample.get("evidence_level")
        if level not in LEVELS:
            errors.append(f"{where}.evidence_level 非法：{level}")

        artifacts = sample.get("artifacts")
        if not isinstance(artifacts, list) or not artifacts:
            errors.append(f"{where}.artifacts 必須至少一筆")
        else:
            for artifact in artifacts:
                if not isinstance(artifact, str) or not artifact:
                    errors.append(f"{where}.artifacts 含空值")
                    continue
                if not (root / artifact).is_file():
                    errors.append(f"{where}.artifacts 不存在：{artifact}")
        if not isinstance(sample.get("limitations"), list):
            errors.append(f"{where}.limitations 必須為陣列")

        qualifies = (
            result == "pass"
            and sample.get("remake_checked") is True
            and sample.get("production_path") is True
            and sample.get("normal_input") is True
            and sample.get("route_patch") is False
            and sample.get("debug_shortcut") is False
            and level in {"RUNTIME-E1", "PLAYER-E2", "E1+E2"}
        )
        if sample.get("qualifies") is not qualifies:
            errors.append(
                f"{where}.qualifies 應為 {str(qualifies).lower()}，不得手動提高資格"
            )
        if qualifies and layer in counts:
            counts[layer] += 1

    return errors, counts, len(samples)


def main() -> int:
    parser = argparse.ArgumentParser()
    parser.add_argument(
        "registry",
        nargs="?",
        default="docs/data/verification/first-round-remake-samples.json",
    )
    parser.add_argument("--require-complete", action="store_true")
    args = parser.parse_args()
    path = Path(args.registry).resolve()
    root = Path(__file__).resolve().parents[1]
    registry = load_registry(path)
    errors, counts, registered = validate(registry, root)

    qualifying = sum(counts.values())
    complete = qualifying >= sum(LAYERS.values()) and all(
        counts[layer] >= minimum for layer, minimum in LAYERS.items()
    )
    upper_failure_rate = (
        1 - math.pow(0.05, 1 / qualifying) if qualifying else None
    )
    print(f"registered_samples={registered}")
    print(f"qualifying_samples={qualifying}/{sum(LAYERS.values())}")
    for layer, minimum in LAYERS.items():
        print(f"{layer}={counts[layer]}/{minimum}")
    if upper_failure_rate is None:
        print("zero_failure_upper_rate_95=unknown")
    else:
        print(f"zero_failure_upper_rate_95={upper_failure_rate:.6f}")
    print(f"first_round_complete={str(complete).lower()}")
    print(f"integrity_errors={len(errors)}")
    for error in errors:
        print(f"ERROR: {error}")

    if errors:
        return 1
    if args.require_complete and not complete:
        return 2
    return 0


if __name__ == "__main__":
    sys.exit(main())
