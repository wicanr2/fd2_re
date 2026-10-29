#!/usr/bin/env python3
"""獨立核對古金故事框的完整PNG、普通START收據與拒收反例。"""
import argparse
import copy
import hashlib
import json
import math
from pathlib import Path
from PIL import Image
from verify_fd2_hd_transient_expiry import GOLD_REFERENCE_SHA,gold_reference,require

def sha(p):return hashlib.sha256(p.read_bytes()).hexdigest()
def frame_pixels(picture,rect):
    x0,y0,x1,y1=rect;w,h=x1-x0,y1-y0
    reference=gold_reference();pixels=picture.load();wanted=reference.load()
    def axis(v,size,ref):return v if v<18 else ref-size+v if v>=size-18 else 30
    checked,different=0,0
    for y in range(max(0,-y0),min(h,1200-y0)):
        for x in range(max(0,-x0),min(w,1920-x0)):
            if 18<=x<w-18 and 18<=y<h-18:continue
            a=pixels[x+x0,y+y0];b=wanted[axis(x,w,1860),axis(y,h,516)]
            different+=any(abs(i-j)>1 for i,j in zip(a,b));checked+=1
    return checked,different

def validate(path):
    d=json.loads(path.read_text());rows=d['rows'];normal=d['status']=='PASS-GOLD-DIALOGUE-NORMAL-START'
    require(normal or d['status']=='PASS-GOLD-STORY-GAME','收據類型不同')
    require(d['normal_player_verified'] is normal,'受控／普通證據分級不同')
    if normal:
        require(d['state_injection'] is False,'普通來源含狀態注入')
        require([r['label'] for r in rows]==['hd-hd','original-hd','original-original','hd-original','hd-hd-restored'],'普通四組控制不足')
        require([(r['art'],r['ui']) for r in rows]==[(True,'hd'),(False,'hd'),(False,'original'),(True,'original'),(True,'hd')],'普通控制組合不同')
    else:
        expected=[]
        for control in ('FFEC','FFEE','FFED','FFEF'):
            for phase in ('opening','closing'):
                expected.extend((control,phase,i,True,True,False) for i in range(5))
            expected.extend((control,phase,0,True,True,False) for phase in ('typing','scroll'))
            expected.extend((control,'steady',0,art,ui,False) for art in (False,True) for ui in (False,True))
        for control in ('FFEC','FFED'):
            expected.extend((control,'steady',0,art,ui,True) for art in (False,True) for ui in (False,True))
            expected.append((control,'opening',0,True,True,True))
        require([tuple(r[k] for k in ('control','phase','index','art','ui','legacy')) for r in rows]==expected,'完整故事74案例不符')
        require(all(r['gameplay_and_clocks_unchanged'] is True for r in rows),'故事規則或時序改變')
    total,captures=0,[]
    for r in rows:
        if 'capture' not in r:continue
        p=path.parent/r['capture'];require(p.parent==path.parent,'截圖路徑越界')
        with Image.open(p) as im:
            require(im.size==(1920,1200),'非完整HD畫布');im=im.convert('RGBA')
            require(all(high-low>=64 for low,high in im.getextrema()[:3]),'截圖缺少實際遊戲內容')
            require(sha(p)==r['sha256'] if normal else hashlib.sha256(im.tobytes()).hexdigest()==r['final_sha256'],'PNG與收據雜湊不同')
            if normal:
                s=r['state'];require(s['hd_theme'] is r['art'] and s['interface_mode']==r['ui'],'普通切換狀態不同')
                require(s['dialogue_native'] is True and s['dialogue_control'] in ('FFED','FFEF'),'父王上框來源不同')
                rect=[30,12,1890,528];gold=r['ui']=='hd'
                require(s.get('hd_gold_dialogue_rects')==[rect] if gold else not s.get('hd_gold_dialogue_rects'),'操作收據框位不同')
            else:
                y=2 if r['control'] in ('FFED','FFEF') else 112
                rect=[30,y*6,1890,y*6+516];gold=r['ui']
                if r['legacy']:
                    by=4 if y==2 else 198;scale=(r['index']+1)/7 if r['phase']=='opening' else 1
                    top=(by+(1-scale)*99)*3
                    rect=[30,math.floor(top+.5),1890,math.floor(top+594*scale+.5)]
                elif r['phase']=='opening':
                    cols,rs=[(4,2),(8,3),(12,4),(16,5),(19,5)][r['index']]
                    rect=[30,y*6,30+(cols*16+6)*6,y*6+(rs*16+6)*6]
            count,bad=frame_pixels(im,rect)
            require(bad==0 if gold else bad>100,'實際PNG古金框錯誤或原版介面誤畫金框')
            total+=count if gold else 0
        captures.append(dict(file=p.name,sha256=sha(p)))
    require(len(captures)==(5 if normal else 14),'實際截圖數量不足')
    return dict(status='PASS-GOLD-DIALOGUE-INDEPENDENT',normal_player_verified=normal,cases=len(rows),gold_png_pixels=total,captures=captures)

def main():
    a=argparse.ArgumentParser(description=__doc__);a.add_argument('receipt',type=Path);a.add_argument('--out',type=Path,required=True);a.add_argument('--self-test',action='store_true');args=a.parse_args()
    result=validate(args.receipt)
    result.update(tool_sha256=sha(Path(__file__)),receipt_sha256=sha(args.receipt),reference_sha256=GOLD_REFERENCE_SHA)
    if args.self_test:
        d=json.loads(args.receipt.read_text());temp=args.receipt.parent/'negative-gold-receipt.json'
        require(not temp.exists(),'反例輸出已存在')
        mutants=[lambda x:x['rows'].pop(),lambda x:x.update(normal_player_verified=not x['normal_player_verified']),lambda x:next(r for r in x['rows'] if 'capture' in r).__setitem__('capture','missing.png')]
        if d['status']=='PASS-GOLD-DIALOGUE-NORMAL-START':mutants.append(lambda x:x['rows'][0].update(ui='original'))
        else:mutants.append(lambda x:x['rows'][0].update(control='FFEF'))
        try:
            for mutate in mutants:
                c=copy.deepcopy(d);mutate(c);temp.write_text(json.dumps(c))
                try:validate(temp)
                except (ValueError,FileNotFoundError):continue
                raise ValueError('古金框反例未被拒收')
            black=args.receipt.parent/'negative-gold-frame.png'
            require(not black.exists(),'PNG反例輸出已存在')
            try:
                im=Image.new('RGBA',(1920,1200),(0,0,0,255));im.save(black)
                c=copy.deepcopy(d);row=next(r for r in c['rows'] if 'capture' in r)
                row.update(capture=black.name,sha256=sha(black),final_sha256=hashlib.sha256(im.tobytes()).hexdigest())
                temp.write_text(json.dumps(c))
                try:validate(temp)
                except ValueError:pass
                else:raise ValueError('實際黑圖反例未被拒收')
            finally:black.unlink(missing_ok=True)
        finally:temp.unlink(missing_ok=True)
        result['rejected_mutants']=len(mutants)+1
    require(not args.out.exists(),'禁止覆寫核對收據');args.out.write_text(json.dumps(result,ensure_ascii=False,indent=2)+'\n');print(json.dumps({k:v for k,v in result.items() if k!='captures'}))
if __name__=='__main__':main()
