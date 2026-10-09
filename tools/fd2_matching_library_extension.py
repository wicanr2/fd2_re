#!/usr/bin/env python3
"""由固定SDK重建三個完整函式及amodf整個CODE；只補既有保留函式庫來源。"""
import argparse
import copy
import hashlib
import json
import os
from pathlib import Path
import re
import subprocess
from types import SimpleNamespace

from fd2_matching_library_link import (INPUT_SHA,LIBRARIES,WLINK_SHA,WDIS_SHA,omf_sections,
    map_segments,plan as native_plan,run as native_run,sha,validate_function)

EXTRA_MEMBERS=('ioalloc','setvbuf','unlink')


def group_section(raw):
    sections=omf_sections(raw)
    at=0
    # 完整群組僅支援無import／fixup的自足CODE；其他OMF形式一律拒收。
    while at<len(raw):
        kind=raw[at];length=int.from_bytes(raw[at+1:at+3],'little')
        if kind not in (0x80,0x88,0x96,0x98,0x99,0x90,0x91,0xa0,0xa1,0x8a,0x8b):
            raise ValueError('SDK群組含未支援的import／fixup／資料紀錄')
        at+=3+length
    if len(sections)!=1 or not sections[0]['flags']&32 or sections[0]['relocation_count']:
        raise ValueError('SDK群組不是完整唯一無重定位CODE')
    section=sections[0]
    if section['size']!=302 or {(p['name'],p['offset']) for p in section['publics']}!={('__ModF',0),('__ZBuf2F',144)}:
        raise ValueError('固定amodf物件的完整CODE或公開入口不符')
    return section


def group_plan(args,image,original,library,ledger,c_addresses):
    obj=args.objects/'amodf.o';raw=obj.read_bytes();position=library.find(raw)
    if position<0 or position!=library.rfind(raw):raise ValueError('SDK群組物件不是LIB內唯一完整序列')
    section=group_section(raw);by_name={}
    for f in image['functions']:by_name.setdefault(f['inventory']['ida_analysis_name'],[]).append(f)
    if len(by_name.get('__ModF',[]))!=1:raise ValueError('SDK群組公開入口沒有唯一IDA定位')
    start=int(by_name['__ModF'][0]['inventory']['start'],16);end=start+section['size']
    functions=sorted([f for f in image['functions'] if start<=int(f['inventory']['start'],16)<end],key=lambda f:int(f['inventory']['start'],16))
    cursor=start;code=bytearray()
    for f in functions:
        inv=f['inventory'];address=inv['start']
        if int(address,16)!=cursor or inv['classification']['value']!='runtime' or 'library' not in inv['ida_function_flags']:
            raise ValueError('SDK群組不是連續完整的既有runtime邊界')
        if ledger.get(address)!='retained_library' or address in c_addresses:
            raise ValueError('SDK群組不能替代C或未分類code')
        data=validate_function(f,original);code.extend(data);cursor+=len(data)
    if cursor!=end or bytes(code)!=section['data']:raise ValueError('SDK整個CODE與完整IDA群組不相同')
    for public in section['publics']:
        matches=by_name.get(public['name'],[])
        if len(matches)!=1 or int(matches[0]['inventory']['start'],16)!=start+public['offset']:
            raise ValueError('SDK群組公開入口與IDA函式邊界不符')
    offset=int(functions[0]['chunks'][0]['instructions'][0]['file_offset'],16)
    if original[offset:offset+section['size']]!=section['data']:
        raise ValueError('SDK群組原檔CODE含修正或不連續映射')
    callers=[]
    for owner in image['functions']:
        if start<=int(owner['inventory']['start'],16)<end:continue
        ins=[i for c in owner['chunks'] for i in c['instructions']]
        for n,i in enumerate(ins):
            b=bytes.fromhex(i['loaded_bytes'])
            if len(b)==5 and b[0] in (0xe8,0xe9) and start<=int(i['ida_linear_address'],16)+5+int.from_bytes(b[1:],'little',signed=True)<end:
                callers.append({'original_owner':owner['inventory']['start'],'call':i['ida_linear_address'],
                    'instructions':ins[max(0,n-2):n+3],'confidence':'已證實'})
    if not callers:raise ValueError('SDK完整群組缺少外部consumer證據')
    return {'object':obj,'section':section,'functions':functions,'start':start,'file_offset':offset,
        'library_file_offset':position,'object_sha256':sha(obj),'object_size':len(raw),'caller_windows':callers}


def prepare(args):
    if sha(args.original)!=INPUT_SHA or sha(args.baseline)!=INPUT_SHA or sha(args.library)!=LIBRARIES['9.5'][0]:
        raise ValueError('原版、基準或SDK版本不符')
    if sha(args.wlink)!=WLINK_SHA or sha(args.wdis)!=WDIS_SHA:raise ValueError('原生工具版本不符')
    image=json.loads(args.evidence.read_text());base=json.loads(args.receipt.read_text());original=args.original.read_bytes()
    if image['tool']['version']!='9.4' or image['input']!=base['input'] or image['input']['sha256']!=INPUT_SHA:
        raise ValueError('IDA／基準／原版不是相同固定輸入')
    if base['output_sha256']!=INPUT_SHA or not base['whole_file_equal'] or base['decompilation_complete']:
        raise ValueError('基準收據或完成層級不符')
    ledger={f['ida_linear_address']:f['source_kind'] for f in base['functions']}
    c_addresses={s['ida_linear_address'] for s in base['restored_spans']}
    group=group_plan(args,image,original,args.library.read_bytes(),ledger,c_addresses)
    native_args=SimpleNamespace(objects=args.objects,library=args.library,evidence=args.evidence,original=args.original,
        output=args.output/'native',wlink=args.wlink,wdis=args.wdis,members=EXTRA_MEMBERS)
    _,plans=native_plan(native_args)
    for p in plans:
        inv=p['function']['inventory']
        if ledger.get(inv['start'])!='retained_library' or inv['start'] in c_addresses:
            raise ValueError('SDK擴充只能加入既有保留函式庫')
        offset=int(p['function']['chunks'][0]['instructions'][0]['file_offset'],16)
        if original[offset:offset+inv['size']]!=p['expected']:
            raise ValueError('本擴充尚未支援原檔CODE內的LE修正')
    return image,base,group,native_args,plans


def link_group(args,group):
    out=args.output/'amodf';out.mkdir();start=group['start'];size=group['section']['size'];origin=start
    # OFFSET只定對齊基準；RAW起點由實際第一個CODE區段決定。
    script='format raw bin\noption nodefaultlibs,offset='+hex(origin&~255)+',map='+str(out/'link.map')+',start=__ModF\n'
    script+='name '+str(out/'linked.raw')+'\nfile '+str(group['object'])+'\norder clname CODE offset='+hex(start)+'\n'
    (out/'link.lnk').write_text(script,encoding='ascii')
    result=subprocess.run([str(args.wlink),'@'+str(out/'link.lnk')],capture_output=True,text=True,timeout=20)
    (out/'link.log').write_text(result.stdout+result.stderr)
    if result.returncode or 'Error!' in result.stdout+result.stderr:raise ValueError('SDK整個群組原生連結失敗')
    text=(out/'link.map').read_text();segments=map_segments(text);nonempty=[s for s in segments if s['size']]
    if len(nonempty)!=1 or nonempty[0]['class']!='CODE' or nonempty[0]['address']!=start or nonempty[0]['size']!=size:
        raise ValueError('SDK群組原生map沒有保留全部唯一CODE')
    for public in group['section']['publics']:
        if not re.search(r'^'+format(start+public['offset'],'08x')+r'[*+]?\s+'+re.escape(public['name'])+r'\s*$',text,re.M|re.I):
            raise ValueError('SDK群組原生map公開入口不符')
    raw=(out/'linked.raw').read_bytes();prefix=start-origin
    if len(raw)!=prefix+size or any(raw[:prefix]):raise ValueError('SDK群組RAW容器前綴或長度不符')
    candidate=raw[prefix:]
    if candidate!=group['section']['data']:raise ValueError('SDK原生整個CODE與原廠物件不符')
    (out/'candidate.bin').write_bytes(candidate)
    subprocess.run([str(args.wdis),'-l='+str(out/'original.dis'),str(group['object'])],capture_output=True,text=True,check=True,timeout=20)
    dis=(out/'original.dis').read_text();publics=sorted(group['section']['publics'],key=lambda p:p['offset'])
    offsets=[p['offset'] for p in publics]+[size]
    if 'No disassembly errors' not in dis or re.findall(r'Routine Size: (\d+) bytes',dis)!=[str(b-a) for a,b in zip(offsets,offsets[1:])]:
        raise ValueError('WDIS未涵蓋SDK整個群組')
    return candidate,segments,prefix


def run(args):
    if not Path('/.dockerenv').exists():raise ValueError('本工具只在Docker執行')
    image,base,group,native_args,plans=prepare(args)
    args.output.mkdir(parents=True,exist_ok=False)
    if args.output.stat().st_uid!=os.getuid():raise ValueError('輸出擁有權不符')
    native_run(native_args);native_report_path=args.output/'native/library-link-report.json';native=json.loads(native_report_path.read_text())
    candidate,segments,prefix=link_group(args,group)
    group_report={'member':'amodf.o','start':hex(group['start']),'size':len(candidate),'publics':group['section']['publics'],
        'object_sha256':group['object_sha256'],'object_size':group['object_size'],'library_file_offset':hex(group['library_file_offset']),
        'code_sha256':hashlib.sha256(candidate).hexdigest(),'functions':[f['inventory'] for f in group['functions']],
        'caller_windows':group['caller_windows'],'map_segments':segments,'raw_container_prefix_zero_bytes':prefix,
        'whole_code_equal':True,'original_classifications_unchanged':True}
    report={'schema_version':1,'input':image['input'],'tool':image['tool'],'evidence_sha256':sha(args.evidence),
        'library':native['library'],'native_tools':native['native_tools'],'driver_sha256':sha(Path(__file__)),
        'native_driver_sha256':native['driver_sha256'],'native_report':native,'native_report_sha256':sha(native_report_path),
        'whole_group':group_report,'classification_changed':False,'decompilation_complete':False,
        'rights':'SDK物件、完整CODE、map與EXE只留本機；公開來源雜湊、工具及有限caller證據。'}
    report_path=args.output/'sdk-extension-report.json';report_path.write_text(json.dumps(report,ensure_ascii=False,indent=2)+'\n')
    original=args.original.read_bytes();rebuilt=bytearray(args.baseline.read_bytes());spans=copy.deepcopy(base['sdk_library_spans'])
    if len({s['ida_linear_address'] for s in spans})!=len(spans):raise ValueError('既有SDK來源重複')
    seen={s['ida_linear_address'] for s in spans};new=[]
    for trial,p in zip(native['trials'],plans):
        if not trial['exact_interval_bytes']:continue
        address=trial['ida_linear_address'];offset=int(p['function']['chunks'][0]['instructions'][0]['file_offset'],16)
        data=(args.output/'native'/Path(trial['member']).stem/'candidate.bin').read_bytes()
        if original[offset:offset+trial['size']]!=data:raise ValueError('SDK完整函式不能直接重建原檔區間')
        if address in seen:continue
        rebuilt[offset:offset+len(data)]=data
        span={k:trial[k] for k in ('ida_linear_address','original_name','size','member','vendor_symbols','object_sha256','object_size','library_file_offset','code_sha256','original_classification','bindings')}
        span.update(file_offset=hex(offset),source_kind='retained_library_from_native_sdk',library_sha256=sha(args.library),
            native_source_report_sha256=sha(native_report_path),native_tool_inputs=native['native_tools'])
        spans.append(span);new.append(span);seen.add(address)
    rebuilt[group['file_offset']:group['file_offset']+len(candidate)]=candidate
    cursor=0
    for f in group['functions']:
        inv=f['inventory'];address=inv['start'];size=inv['size'];data=candidate[cursor:cursor+size]
        if address not in seen:
            span={'ida_linear_address':address,'original_name':inv['ida_analysis_name'],'size':size,'member':'amodf.o',
                'vendor_symbols':[p['name'] for p in group['section']['publics'] if p['offset']==cursor],
                'object_sha256':group['object_sha256'],'object_size':group['object_size'],'library_file_offset':hex(group['library_file_offset']),
                'code_sha256':hashlib.sha256(data).hexdigest(),'original_classification':inv['classification'],
                'file_offset':hex(group['file_offset']+cursor),'source_kind':'retained_library_from_native_sdk_group',
                'library_sha256':sha(args.library),'native_source_report_sha256':sha(report_path),
                'whole_group_start':hex(group['start']),'whole_group_size':len(candidate),'offset_in_whole_group':cursor,
                'whole_group_code_sha256':group_report['code_sha256'],'native_tool_inputs':native['native_tools']}
            spans.append(span);new.append(span);seen.add(address)
        cursor+=size
    if hashlib.sha256(rebuilt).hexdigest()!=INPUT_SHA or bytes(rebuilt)!=original:raise ValueError('SDK擴充後全檔不相同')
    (args.output/'FD2.EXE').write_bytes(rebuilt)
    receipt=copy.deepcopy(base);receipt.update(sdk_library_spans=spans,sdk_library_function_count=len(spans),sdk_library_code_bytes=sum(s['size'] for s in spans),
        sdk_library_is_subset_of_retained_library=True,classification_changed=False,
        sdk_extension_new_functions=len(new),sdk_extension_new_code_bytes=sum(s['size'] for s in new),sdk_extension_rebuilt_spans=new,
        sdk_extension_inputs={'baseline_receipt_sha256':sha(args.receipt),'source_report_sha256':sha(report_path),'driver_sha256':sha(Path(__file__))})
    (args.output/'sdk-extension-bootstrap-receipt.json').write_text(json.dumps(receipt,ensure_ascii=False,indent=2)+'\n')
    print(json.dumps({'sdk_functions':len(spans),'sdk_bytes':sum(s['size'] for s in spans),'new_source_functions':len(new),
        'new_source_bytes':sum(s['size'] for s in new),'counts':base['counts'],'classification_changed':False,'whole_file_equal':True,'decompilation_complete':False}))


def main():
    p=argparse.ArgumentParser(description=__doc__)
    for name in ('objects','library','evidence','original','baseline','receipt','output'):p.add_argument('--'+name,type=Path,required=True)
    p.add_argument('--wlink',type=Path,default=Path('/opt/watcom/binl64/wlink'))
    p.add_argument('--wdis',type=Path,default=Path('/opt/watcom/binl64/wdis'))
    run(p.parse_args())


if __name__=='__main__':main()
