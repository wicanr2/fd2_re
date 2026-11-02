#!/usr/bin/env python3
"""只以受版控 importer 的已驗輸出建立教堂圖示修正候選，不改基底素材。"""
import argparse
import hashlib
import json
import os
from pathlib import Path
import shutil

def sha(path):
    return hashlib.sha256(path.read_bytes()).hexdigest()

def build(base, exported, out):
    if not Path('/.dockerenv').exists() or out.exists() or not out.parent.is_dir():
        raise ValueError('需要Docker與既有父目錄中的全新輸出')
    if out.parent.stat().st_uid != os.getuid():
        raise ValueError('輸出目錄擁有者不同')
    prefix = Path('ui/fdother_014_church')
    old, new = base/prefix, exported/prefix
    receipt = json.loads((exported/'codec-receipt.json').read_text())
    if receipt['status'] != 'PASS-ORIGINAL-CHURCH-RAW-SERVICE-EXPORT':
        raise ValueError('缺少實際匯出器驗證')
    previous = json.loads((old/'resource.json').read_text())
    current = json.loads((new/'resource.json').read_text())
    before = {e['index']:e for e in previous['entries']}
    after = {e['index']:e for e in current['entries']}
    if previous['source'] != current['source'] or set(before) != set(after) or len(after) != 21:
        raise ValueError('來源或21筆集合不同')
    services = set(range(3,11))
    for index in before:
        if index in services:
            expected = dict(before[index],codec='raw_indexed_opaque')
            if after[index] != expected or before[index]['codec'] != 'opaque_high_run':
                raise ValueError('只允許修正八個既有服務codec')
        elif before[index] != after[index]:
            raise ValueError('其他entry描述改變')
    expected = {Path('resource.json')} | {Path(e['frame']) for e in after.values()} | {
        Path(e['mask']) for e in after.values() if e.get('mask')}
    actual = {p.relative_to(new) for p in new.rglob('*') if p.is_file()}
    if actual != expected:
        raise ValueError('匯出檔案多項或缺項')
    changed = {Path('resource.json')} | {Path(after[i]['frame']) for i in services}
    for rel in expected:
        if rel not in changed and (old/rel).read_bytes() != (new/rel).read_bytes():
            raise ValueError('其他圖源bytes改變: '+str(rel))
        if rel in changed and sha(old/rel) == sha(new/rel):
            raise ValueError('應改檔案沒有改變: '+str(rel))
    manifest = json.loads((base/'manifest.json').read_text())
    found = set()
    for row in manifest['assets']:
        rel = Path(row['path'])
        try:
            local = rel.relative_to(prefix)
        except ValueError:
            continue
        if local in changed:
            if sha(base/rel) != row['sha256']:
                raise ValueError('基底manifest不符')
            row['sha256'] = sha(new/local)
            found.add(local)
    if found != changed:
        raise ValueError('manifest缺少修正項目')
    out.mkdir()
    for source in base.iterdir():
        if source.name in ['ui','manifest.json']:
            continue
        (out/source.name).symlink_to(source.resolve(),target_is_directory=source.is_dir())
    (out/'ui').mkdir()
    for source in (base/'ui').iterdir():
        if source.name == prefix.name:
            shutil.copytree(new,out/prefix)
        else:
            (out/'ui'/source.name).symlink_to(source.resolve(),target_is_directory=source.is_dir())
    (out/'manifest.json').write_text(json.dumps(manifest,ensure_ascii=False,indent=2)+'\n')
    rows=[{'path':str(prefix/rel),'before_sha256':sha(old/rel),'sha256':sha(out/prefix/rel)}
          for rel in sorted(changed)]
    report={'status':'PASS-CHURCH-CODEC-CANDIDATE','changed_outputs':rows,
            'unchanged_outputs':len(expected)-len(changed),'manifest_assets':len(manifest['assets']),
            'manifest_sha256':sha(out/'manifest.json'),'metadata_sha256':sha(out/prefix/'resource.json'),
            'source_resources_unchanged':True,'base_modified':False}
    (out.parent/'candidate-original-receipt.json').write_text(json.dumps(report,indent=2)+'\n')
    return report

def main():
    parser=argparse.ArgumentParser(description=__doc__)
    for name in ['base','exported','out']:
        parser.add_argument('--'+name,type=lambda s:Path(s).absolute(),required=True)
    args=parser.parse_args()
    report=build(args.base,args.exported,args.out)
    print(json.dumps({'status':report['status'],'changed_outputs':len(report['changed_outputs']),
                      'unchanged_outputs':report['unchanged_outputs'],'metadata_sha256':report['metadata_sha256']}))
if __name__=='__main__':
    main()
