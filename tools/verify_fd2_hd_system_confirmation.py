#!/usr/bin/env python3
"""核對五caller完整矩陣、目前Go、實際DATO75像素與四組控制。"""
import argparse
from collections import Counter
from copy import deepcopy
import hashlib
import json
from pathlib import Path
import tempfile

from PIL import Image,ImageOps

ROOT=Path(__file__).resolve().parents[1]
def sha(p):return hashlib.sha256(p.read_bytes()).hexdigest()
def read(p):return json.loads(p.read_text())
def key(r):return tuple(r[k] for k in ('caller','phase','frame','choice','pulse','accepted','art_hd','ui_hd','locale'))
def required():
    rows=[]
    for caller in range(5):
        for art in (False,True):
            for ui in (False,True):
                for choice in range(2):
                    for pulse in range(2):rows.append((caller,1,0,choice,pulse,False,art,ui,'zh-Hant'))
        for phase,indices in ((0,[0,5,6,9]),(2,[0,3]),(4,[-2]),(3,[0,4,5])):
            for index in indices:rows.append((caller,phase,index,0,0,False,True,True,'zh-Hant'))
        rows.append((caller,4,-2,0,0,True,True,True,'zh-Hant'))
        if caller==0:
            for locale in ('zh-Hans','en','ja'):
                for choice in range(2):
                    for pulse in range(2):rows.append((0,1,0,choice,pulse,False,True,True,locale))
    return Counter(rows)

def verify(report,folder,catalog,pack,original):
    if report.get('status')!='PASS-SYSTEM-CONFIRMATION-HD-GPU' or report.get('catalog_sha256')!=sha(catalog):raise ValueError('清冊或GPU狀態不同')
    if report.get('caller_count')!=5 or report.get('display_modes')!=4 or report.get('save_bytes_unchanged',0)<=0:raise ValueError('caller、四組控制或保存缺項')
    rows=report.get('rows',[])
    if Counter(map(key,rows))!=required():raise ValueError('完整147個來源矩陣缺項或重複')
    if {r['caller'] for r in rows if r.get('f2_f11_roundtrip')}!=set(range(5)):raise ValueError('途中F2／F11往返缺項')
    if any(r.get('full_rgba_pixels')!=1920*1200 or r.get('rgba_tolerance')!=2 or r.get('source_state_unchanged') is not True for r in rows):raise ValueError('完整RGBA或原生狀態核對缺項')
    files=sorted(list((ROOT/'remake/cmd').rglob('*.go'))+list((ROOT/'remake/internal').rglob('*.go')),key=lambda p:p.as_posix())
    tree=hashlib.sha256(''.join(f'{p.relative_to(ROOT/"remake").as_posix()}\0{sha(p)}\n' for p in files).encode()).hexdigest()
    if report.get('source_tree_sha256')!=tree or report.get('source_file_count')!=len(files):raise ValueError('Go來源指紋不同')
    asset=next(a for a in read(catalog)['assets'] if a['role']=='story_portrait_hd_frame' and a['speaker_id']==75)
    if sha(pack/asset['file'])!=asset['sha256']:raise ValueError('DATO75 HD來源SHA不同')
    hd=ImageOps.mirror(Image.open(pack/asset['file']).convert('RGBA'))
    native=ImageOps.mirror(Image.open(original/'portraits/DATO_075_m0.png').convert('RGBA')).resize((480,480),Image.Resampling.NEAREST)
    captures=[r for r in rows if r.get('file')]
    if len(captures)!=15 or len({r['file'] for r in captures})!=15:raise ValueError('完整畫面PNG缺項')
    pixels=0
    for r in captures:
        name=r['file']
        if Path(name).name!=name or sha(folder/name)!=r['sha256']:raise ValueError('PNG路徑或SHA不同')
        image=Image.open(folder/name).convert('RGBA')
        if image.size!=(1920,1200):raise ValueError('不是完整遊戲畫面')
        if image.crop((48,690,528,1170)).tobytes()!=(hd if r['art_hd'] else native).tobytes():raise ValueError('實際頭像與美術主題圖源不同')
        pixels+=480*480
    return dict(status='PASS-SYSTEM-CONFIRMATION-INDEPENDENT',matrix_rows=len(rows),pngs=len(captures),portrait_pixels=pixels,source_tree_sha256=tree,source_file_count=len(files))

def main():
    p=argparse.ArgumentParser(description=__doc__)
    for k in ('receipt','catalog','pack','original','out'):p.add_argument('--'+k,type=Path,required=True)
    p.add_argument('--overview-out',type=Path)
    p.add_argument('--font',type=Path)
    p.add_argument('--self-test',action='store_true');a=p.parse_args()
    if a.out.exists():raise ValueError('不得覆寫核對報告')
    d=read(a.receipt);result=verify(d,a.receipt.parent,a.catalog,a.pack,a.original);rejected=[]
    if a.self_test:
        for fault in ('missing-row','duplicate-row','wrong-mode','wrong-phase','wrong-code','wrong-catalog','wrong-png','missing-save','wrong-tolerance','false-state'):
            bad=deepcopy(d)
            if fault=='missing-row':bad['rows'].pop()
            elif fault=='duplicate-row':bad['rows'][-1]=deepcopy(bad['rows'][0])
            elif fault=='wrong-mode':bad['rows'][0]['ui_hd']=not bad['rows'][0]['ui_hd']
            elif fault=='wrong-phase':bad['rows'][0]['phase']=99
            elif fault=='wrong-code':bad['source_tree_sha256']='wrong'
            elif fault=='wrong-catalog':bad['catalog_sha256']='wrong'
            elif fault=='wrong-png':next(r for r in bad['rows'] if r.get('file'))['sha256']='wrong'
            elif fault=='missing-save':bad['save_bytes_unchanged']=0
            elif fault=='wrong-tolerance':bad['rows'][0]['rgba_tolerance']=3
            elif fault=='false-state':bad['rows'][0]['source_state_unchanged']=False
            try:verify(bad,a.receipt.parent,a.catalog,a.pack,a.original)
            except (ValueError,KeyError):rejected.append(fault)
            else:raise ValueError('反例未拒收：'+fault)
        with tempfile.TemporaryDirectory(dir=a.out.parent) as temp:
            folder=Path(temp);bad=deepcopy(d);rows=[r for r in bad['rows'] if r.get('file')]
            changed=next(r for r in rows if r['art_hd'])
            for r in rows:
                if r is not changed:(folder/r['file']).symlink_to((a.receipt.parent/r['file']).resolve())
            Image.new('RGB',(1920,1200)).save(folder/changed['file']);changed['sha256']=sha(folder/changed['file'])
            try:verify(bad,folder,a.catalog,a.pack,a.original)
            except ValueError as error:
                if '實際頭像' not in str(error):raise
                rejected.append('actual-black-png')
            else:raise ValueError('黑畫面未拒收')
    if a.overview_out is not None:
        from PIL import ImageFont
        from build_fd2_hd_map_overview import bounded_text
        if a.overview_out.exists() or a.font is None:raise ValueError('總覽需全新輸出與明示字型')
        fonts={size:ImageFont.truetype(str(a.font),size,index=3) for size in (18,24,34)}
        if 'TC' not in fonts[24].getname()[0]:raise ValueError('總覽須使用Noto繁中字型')
        canvas=Image.new('RGB',(1352,1136),'#131b24')
        labels=[bounded_text(canvas,'HD1 系統確認與獨立切換',(24,22,1304,48),fonts[34],'#f3d799'),bounded_text(canvas,'同一受控世界與原生問句，介面與美術分開選擇',(24,80,1304,26),fonts[18],'#cad4df')]
        for i,(art,ui) in enumerate(((False,False),(False,True),(True,False),(True,True))):
            row=next(r for r in d['rows'] if r.get('file') and r['caller']==0 and r['locale']=='zh-Hant' and r['art_hd']==art and r['ui_hd']==ui)
            x,y=24+(i%2)*664,126+(i//2)*488
            label=f'美術：{"HD" if art else "原版"}　介面：{"HD" if ui else "原版"}'
            labels.append(bounded_text(canvas,label,(x+12,y+8,616,32),fonts[24],'#f0dfb6'))
            image=Image.open(a.receipt.parent/row['file']).convert('RGB')
            canvas.paste(image.resize((640,400),Image.Resampling.LANCZOS),(x,y+48))
        labels.append(bounded_text(canvas,'完整遊戲畫面縮圖；受控章節測具，造型與一般操作由使用者確認。',(24,1090,1304,28),fonts[18],'#c5d4df'))
        canvas.save(a.overview_out,optimize=True)
        result['overview']=dict(file=a.overview_out.name,sha256=sha(a.overview_out),size=list(canvas.size),labels=labels,font_sha256=sha(a.font))
    result['rejected_mutants']=rejected
    result['inputs']=dict(receipt_sha256=sha(a.receipt),catalog_sha256=sha(a.catalog),tool_sha256=sha(Path(__file__)))
    a.out.write_text(json.dumps(result,ensure_ascii=False,indent=2)+'\n');print(json.dumps(result,ensure_ascii=False))

if __name__=='__main__':main()
