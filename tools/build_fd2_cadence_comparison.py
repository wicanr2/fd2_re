#!/usr/bin/env python3
"""合成DOSBox-X與重製的相近開場片段，僅作本機呈現診斷。"""
import argparse
import hashlib
import json
from pathlib import Path
import subprocess


def main():
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument('--original', type=Path, required=True)
    parser.add_argument('--remake', type=Path, required=True)
    parser.add_argument('--output', type=Path, required=True)
    args = parser.parse_args()
    assert Path('/.dockerenv').exists() and not args.output.exists()
    font = '/usr/share/fonts/opentype/noto/NotoSerifCJK-Regular.ttc'
    filters = (
        '[0:v]fps=60,scale=640:400:flags=neighbor,setsar=1[a];'
        '[1:v]fps=60,scale=640:400:flags=neighbor,setsar=1[b];'
        '[a][b]hstack,pad=1280:480:0:40:color=0x071a19,'
        f"drawtext=fontfile={font}:text='DOSBox-X 原版（12000 cycles）':x=16:y=8:fontsize=22:fontcolor=white,"
        f"drawtext=fontfile={font}:text='重製版：獨立音畫擷取':x=656:y=8:fontsize=22:fontcolor=white,"
        f"drawtext=fontfile={font}:text='相近開場片段，未逐幀同步；音軌省略。僅作呈現診斷，不是正式原版對拍。':x=16:y=448:fontsize=20:fontcolor=white[v]")
    subprocess.run(['ffmpeg', '-hide_banner', '-loglevel', 'warning', '-ss', '9',
                    '-i', str(args.original), '-ss', '19', '-i', str(args.remake),
                    '-t', '12', '-filter_complex_threads', '1', '-filter_complex', filters,
                    '-map', '[v]', '-an', '-threads', '2', '-c:v', 'libx264',
                    '-preset', 'veryfast', '-crf', '19', '-pix_fmt', 'yuv420p',
                    '-movflags', '+faststart', str(args.output)], check=True)
    subprocess.run(['ffmpeg', '-v', 'error', '-threads', '1', '-i', str(args.output),
                    '-f', 'null', '-'], check=True)
    probe = json.loads(subprocess.check_output(['ffprobe', '-v', 'error', '-show_format',
                                               '-show_streams', '-of', 'json', str(args.output)]))
    assert abs(float(probe['format']['duration']) - 12) < .1
    assert (probe['streams'][0]['width'], probe['streams'][0]['height']) == (1280, 480)
    report = {'schema_version': 1, 'status': 'DECODE-PASSED-REVIEW-REQUIRED',
              'public_distribution': False, 'original_runner': 'DOSBox-X auxiliary',
              'comparison': '相近的首個大型騎士登場片段；不同起始時點，未逐幀同步。',
              'original_start': 9, 'remake_start': 19, 'duration': 12,
              'original_sha256': hashlib.file_digest(args.original.open('rb'), 'sha256').hexdigest(),
              'remake_sha256': hashlib.file_digest(args.remake.open('rb'), 'sha256').hexdigest(),
              'sha256': hashlib.file_digest(args.output.open('rb'), 'sha256').hexdigest(),
              'bytes': args.output.stat().st_size, 'audio': 'omitted for visual diagnosis',
              'limits': 'cycles、啟動與軟體繪圖負擔不同；只比較錄影與呈現，不宣稱硬體時間或dosgolem parity。',
              'probe': probe}
    args.output.with_suffix('.json').write_text(json.dumps(report, ensure_ascii=False, indent=2) + '\n')
    subprocess.run(['ffmpeg', '-v', 'error', '-threads', '1', '-ss', '3', '-i', str(args.output),
                    '-frames:v', '1', '-filter_threads', '1', '-threads', '1',
                    str(args.output.with_suffix('.png'))], check=True)
    print(json.dumps({k: v for k, v in report.items() if k != 'probe'}, ensure_ascii=False))


if __name__ == '__main__':
    main()
