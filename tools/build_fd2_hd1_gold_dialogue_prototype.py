#!/usr/bin/env python3
"""以SVG呈現HD1古金色對話框原型，來源遊戲PNG保持原檔。"""
import argparse
import base64
import hashlib
import json
import struct
from pathlib import Path

def sha(p):return hashlib.sha256(p.read_bytes()).hexdigest()
def frame_svg(w,h):
    if w<40 or h<40:raise ValueError('frame smaller than fixed corners')
    return f'''<svg xmlns="http://www.w3.org/2000/svg" width="{w}" height="{h}">
<path fill="#302719" fill-rule="evenodd" d="M0 0H{w}V{h}H0Z M18 18V{h-18}H{w-18}V18Z"/>
<rect x="3" y="3" width="{w-6}" height="{h-6}" fill="none" stroke="#b99a58" stroke-width="5"/>
<path d="M6 7H{w-7} M6 7V{h-7}" fill="none" stroke="#e6d2a0" stroke-width="2"/>
<path d="M{w-7} 7V{h-7}H6" fill="none" stroke="#70532e" stroke-width="3"/>
<rect x="14" y="14" width="{w-28}" height="{h-28}" fill="none" stroke="#b99a58" stroke-width="2"/>
<rect x="17" y="17" width="{w-34}" height="{h-34}" fill="none" stroke="#171a25" stroke-width="2"/></svg>'''
def main():
    p=argparse.ArgumentParser(description=__doc__);p.add_argument('--source',type=Path);p.add_argument('--frame-size',type=int,nargs=2);p.add_argument('--out',type=Path,required=True);a=p.parse_args()
    if a.frame_size:
        if a.source or a.out.with_suffix('.svg').exists():raise ValueError('fresh frame-only output required')
        a.out.with_suffix('.svg').write_text(frame_svg(*a.frame_size));return
    if not a.source:p.error('--source required without --frame-size')
    for suffix in ('.svg','.html','.json'):
        if a.out.with_suffix(suffix).exists():raise ValueError('fresh output required')
    header=a.source.read_bytes()[:24]
    if header[:8]!=b'\x89PNG\r\n\x1a\n' or struct.unpack('>II',header[16:24])!=(1920,1200):raise ValueError('expected full1920x1200 PNG source')
    url='data:image/png;base64,'+base64.b64encode(a.source.read_bytes()).decode()
    # Source is a full Game.Draw reference. Only the native frame's 18px
    # border band is covered; head, text, interior and world remain its image.
    ring='''<path fill="#302719" fill-rule="evenodd" d="M30 672H1890V1188H30Z M48 690V1170H1872V690Z"/>
<rect x="33" y="675" width="1854" height="510" fill="none" stroke="#b99a58" stroke-width="5"/>
<path d="M36 679H1883 M36 679V1181" fill="none" stroke="#e6d2a0" stroke-width="2"/>
<path d="M1883 679V1181H36" fill="none" stroke="#70532e" stroke-width="3"/>
<rect x="44" y="686" width="1832" height="488" fill="none" stroke="#b99a58" stroke-width="2"/>
<rect x="47" y="689" width="1826" height="482" fill="none" stroke="#171a25" stroke-width="2"/>'''
    svg=f'''<svg xmlns="http://www.w3.org/2000/svg" xmlns:xlink="http://www.w3.org/1999/xlink" width="1000" height="1410" viewBox="0 0 1000 1410">
<rect width="1000" height="1410" fill="#101827"/>
<g font-family="DejaVu Sans" fill="#e6d2a0" font-size="23"><text x="20" y="38">CURRENT HD1 / 3px border</text><text x="20" y="724">HD1 PROTOTYPE / antique gold frame</text></g>
<g transform="translate(20 60) scale(.5)"><image width="1920" height="1200" xlink:href="{url}"/></g>
<g transform="translate(20 746) scale(.5)"><image width="1920" height="1200" xlink:href="{url}"/>{ring}</g>
<text x="20" y="1380" font-family="DejaVu Sans" font-size="17" fill="#b5c2d6">STYLE PROTOTYPE / SVG frame overlay only / not a new runtime capture</text></svg>'''
    a.out.with_suffix('.svg').write_text(svg)
    a.out.with_suffix('.html').write_text('<!doctype html><html lang="zh-Hant"><meta charset="utf-8"><title>HD1古金色對話框原型</title><body style="margin:0;background:#101827;color:#e6d2a0;font-family:sans-serif"><p style="margin:20px">上：現行HD細框。下：古金色框線原型。遊戲圖、頭像、文字與原框位保留；這是SVG疊圖，尚未接入遊戲。</p>'+svg+'</body></html>')
    a.out.with_suffix('.json').write_text(json.dumps(dict(status='PROTOTYPE-NOT-RUNTIME',source=str(a.source),source_sha256=sha(a.source),tool_sha256=sha(Path(__file__)),svg_sha256=sha(a.out.with_suffix('.svg')),palette=dict(gold='#b99a58',highlight='#e6d2a0',shadow='#70532e',outer='#302719',inner='#171a25'),native_rect=[5,112,315,198],border_band_physical_pixels=18,body_and_text_unchanged=True),ensure_ascii=False,indent=2)+'\n')
    print(a.out.with_suffix('.html'))
if __name__=='__main__':main()
