#!/usr/bin/env python3
"""依既有80／480契約追加DATO75候選；正式登錄另驗GPU收據。"""
import argparse
import hashlib
import json
import os
from pathlib import Path
import shutil
import tempfile

from PIL import Image

ROOT = Path(__file__).resolve().parents[1]

def sha(path):
    return hashlib.sha256(path.read_bytes()).hexdigest()

def read(path):
    return json.loads(path.read_text())

def prepare(recipe_path, catalog_path, pack, original, master, out):
    r, c = read(recipe_path), read(catalog_path)
    if not Path('/.dockerenv').exists() or out.exists() or not out.parent.is_dir():
        raise ValueError('只在Docker及既有父目錄編製全新候選')
    if (r['kind'], r['speaker_id'], r['master_file']) != ('fd2_system_confirmation_portrait',75,'dato-075-system-portrait-handpainted-v1-master.png'):
        raise ValueError('固定頭像配方不同')
    if sha(catalog_path)!=r['base_catalog_sha256'] or sha(master)!=r['master_sha256']:
        raise ValueError('基底或母稿SHA不同')
    if sha(original/'portraits/DATO_075_m0.png')!=r['native_png_sha256']:
        raise ValueError('原版頭像SHA不同')
    if any(a.get('speaker_id')==75 and a['role'] in ('story_portrait_frame','story_portrait_hd_frame') for a in c['assets']):
        raise ValueError('已有DATO75登錄')
    image=Image.open(master).convert('RGBA')
    if image.size!=(1254,1254) or image.getchannel('A').getextrema()!=(255,255):
        raise ValueError('母稿形狀或透明度不同')
    names=set()
    for a in c['assets']:
        names.update(a.get('files',[]));names.update(a.get('source_files',[]))
        names.update(a[k] for k in ('file','master_file') if k in a)
    if any(Path(n).name!=n or not (pack/n).is_file() for n in names):
        raise ValueError('既有私人包缺件或路徑不安全')
    stage=Path(tempfile.mkdtemp(prefix='.system-portrait-',dir=out.parent))
    try:
        for n in sorted(names):(stage/n).symlink_to(pack.resolve()/n)
        shutil.copyfile(master,stage/r['master_file'])
        outputs=[]
        for size,role,contract in [(80,'story_portrait_frame','native_story_dialogue_rgba_overlay_v1'),(480,'story_portrait_hd_frame','native_story_dialogue_rgba_overlay_hd6_v1')]:
            name=f'dato-075-system-portrait-handpainted-v1-{size}.png'
            image.resize((size,size),Image.Resampling.LANCZOS).save(stage/name,compress_level=9)
            outputs.append(dict(file=name,sha256=sha(stage/name),size=size))
            c['assets'].append(dict(asset_id=f'modern.system_clerk.portrait.{size}.frame0',role=role,status='runtime_candidate',file=name,master_file=r['master_file'],width=size,height=size,sha256=sha(stage/name),speaker_id=75,frame=0,mouth_state='closed',consumer_contract=contract,source_refs=['asset:DATO_075_m0','remake/assets/themes/modern/system-confirmation-portrait.recipe.json','docs/knowledge-base/61-modern-theme-runtime-spec.md']))
        c['private_root']=out.relative_to(ROOT).as_posix()
        (stage/'catalog.json').write_text(json.dumps(c,ensure_ascii=False,indent=2)+'\n')
        report=dict(kind=r['kind'],status='SOURCE-VERIFIED-CANDIDATE',base_catalog_sha256=sha(catalog_path),catalog_sha256=sha(stage/'catalog.json'),master_file=r['master_file'],master_sha256=r['master_sha256'],recipe_sha256=sha(recipe_path),tool_sha256=sha(Path(__file__)),outputs=outputs)
        (stage/'portrait-manifest.json').write_text(json.dumps(report,ensure_ascii=False,indent=2)+'\n')
        stage.rename(out)
        return report
    except BaseException:
        shutil.rmtree(stage,ignore_errors=True)
        raise

def register(catalog_path,candidate,gpu_path):
    from verify_fd2_hd_system_confirmation import verify
    c,p,m,g=read(catalog_path),read(candidate/'catalog.json'),read(candidate/'portrait-manifest.json'),read(gpu_path)
    if not Path('/.dockerenv').exists() or sha(catalog_path)!=m['base_catalog_sha256'] or sha(candidate/'catalog.json')!=m['catalog_sha256'] or g.get('status')!='PASS-SYSTEM-CONFIRMATION-HD-GPU' or g.get('catalog_sha256')!=m['catalog_sha256'] or g.get('portrait_manifest_sha256')!=sha(candidate/'portrait-manifest.json'):
        raise ValueError('正式、候選或GPU收據不符')
    if p['assets'][:-2]!=c['assets'] or len(p['assets'])!=len(c['assets'])+2 or len(g.get('rows',[]))<100 or g.get('caller_count')!=5 or g.get('display_modes')!=4 or g.get('save_bytes_unchanged',0)<=0:
        raise ValueError('完整消費端、切換或保存檢查缺項')
    verify(g,gpu_path.parent,candidate/'catalog.json',candidate,Path('/original'))
    for row in m['outputs']:
        if Path(row['file']).name!=row['file'] or sha(candidate/row['file'])!=row['sha256']:
            raise ValueError('已驗頭像變動')
    if sha(candidate/m['master_file'])!=m['master_sha256']:
        raise ValueError('已驗母稿變動')
    for row in g['rows']:
        if row.get('file') and (Path(row['file']).name!=row['file'] or sha(gpu_path.parent/row['file'])!=row['sha256']):
            raise ValueError('GPU圖片變動')
    p['private_root']=c['private_root']
    with tempfile.NamedTemporaryFile('w',dir=catalog_path.parent,delete=False) as f:
        f.write(json.dumps(p,ensure_ascii=False,indent=2)+'\n');temp=Path(f.name)
    os.chmod(temp,catalog_path.stat().st_mode&0o777);temp.replace(catalog_path)
    return dict(status='REGISTERED-SYSTEM-CONFIRMATION-PORTRAIT',catalog_sha256=sha(catalog_path),gpu_sha256=sha(gpu_path))

if __name__=='__main__':
    parser=argparse.ArgumentParser(description=__doc__)
    parser.add_argument('--register',action='store_true')
    for key in ('recipe','catalog','pack','original','master','out','candidate','gpu-receipt'):
        parser.add_argument('--'+key,type=lambda p:Path(p).resolve())
    a=parser.parse_args()
    args=(a.catalog,a.candidate,a.gpu_receipt) if a.register else (a.recipe,a.catalog,a.pack,a.original,a.master,a.out)
    if any(v is None for v in args):parser.error('須明示全部輸入')
    print(json.dumps(register(*args) if a.register else prepare(*args),ensure_ascii=False))
