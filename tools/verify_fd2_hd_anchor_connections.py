#!/usr/bin/env python3
"""核對已編製HD圖層的明示來源定位點；不推定姿勢或原版parity。"""
import argparse
import hashlib
import json
import math
from pathlib import Path

from PIL import Image


def sha(path):
    return hashlib.sha256(path.read_bytes()).hexdigest()


def main():
    p = argparse.ArgumentParser(description=__doc__)
    for name in ['compiled', 'recipe', 'pack', 'connections', 'report']:
        p.add_argument('--'+name, type=Path, required=True)
    args = p.parse_args()
    if args.report.exists():
        raise SystemExit('不覆寫既有收據')
    root = Path(__file__).resolve().parents[1]
    compiled = json.loads(args.compiled.read_text())
    recipe = json.loads(args.recipe.read_text())
    connections = json.loads(args.connections.read_text())
    assert compiled['status'] == 'PASS-COMPILED-CHARACTER-CANDIDATE'
    assert compiled['recipe_sha256'] == sha(args.recipe)
    assert connections['schema_version'] == 1 and connections['connections']
    checks = []
    for connection in connections['connections']:
        frame = next(f for f in compiled['frames'] if
                     (f['resource'], f['frame']) == (connection['resource'], connection['frame']))
        assert Path(frame['file']).name == frame['file']
        assert sha(args.pack/frame['file']) == frame['sha256'], '成品PNG SHA不同'
        points = []
        for endpoint in [connection['left'], connection['right']]:
            pose, point = endpoint['pose'], endpoint['point']
            assert len(point) == 2 and all(isinstance(v, (int, float)) and math.isfinite(v) for v in point)
            source = recipe['sources'][pose]
            path = (root/source['file']).resolve()
            assert path.is_relative_to(root) and sha(path) == source['sha256']
            assert compiled['sources'][pose] == source
            with Image.open(path) as image:
                assert image.mode == 'RGBA' and 0 <= point[0] < image.width and 0 <= point[1] < image.height
                neighbors = [image.getpixel((x, y))[3]
                             for x in {math.floor(point[0]), math.ceil(point[0])}
                             for y in {math.floor(point[1]), math.ceil(point[1])}
                             if 0 <= x < image.width and 0 <= y < image.height]
                assert max(neighbors) >= 192, '定位點沒有明示高覆蓋圖源'
            layer = next(v for v in frame.get('layers', [frame]) if v['pose'] == pose)
            assert layer['source_anchor'] == source['anchor']
            points.append([layer['origin'][c]+point[c]*layer['scale_xy'][c] for c in range(2)])
        maximum = connection['maximum_distance_hd']
        assert isinstance(maximum, (int, float)) and 0 < maximum <= 12
        distance = math.dist(*points)
        if distance > maximum:
            raise SystemExit(f"來源定位點未連接：{frame['resource']}/{frame['frame']} distance={distance:.3f} limit={maximum}")
        checks.append({'resource': frame['resource'], 'frame': frame['frame'],
                       'left_hd': points[0], 'right_hd': points[1],
                       'distance_hd_pixels': distance, 'maximum_distance_hd': maximum})
    report = {'status': 'PASS-HD-AUTHORED-ANCHOR-CONNECTIONS',
              'compiled_sha256': sha(args.compiled), 'recipe_sha256': sha(args.recipe),
              'connections_sha256': sha(args.connections), 'tool_sha256': sha(Path(__file__)),
              'checks': checks,
              'scope': '只核對人工選定的原RGBA高覆蓋點與已編製仿射位置、成品SHA；不推定人體自然度、最終GPU、原版位置或逐幀parity。'}
    args.report.write_text(json.dumps(report, ensure_ascii=False, indent=2)+'\n')
    print(json.dumps({'status': report['status'], 'checked': len(checks),
                      'maximum_hd_pixels': max(v['distance_hd_pixels'] for v in checks)}, ensure_ascii=False))


if __name__ == '__main__':
    main()
