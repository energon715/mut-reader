#!/usr/bin/env python3
"""Collect exact module versions and available license notices into build/licenses."""
import json, pathlib, shutil, subprocess
root=pathlib.Path(__file__).resolve().parent.parent
out=root/'build/licenses'
if out.exists():shutil.rmtree(out)
out.mkdir(parents=True)
raw=subprocess.check_output(['go','list','-m','-json','all'],cwd=root,text=True)
decoder=json.JSONDecoder();modules=[]
while raw.strip():
 value,pos=decoder.raw_decode(raw.lstrip());raw=raw.lstrip()[pos:];modules.append(value)
used=set(subprocess.check_output(["go","list","-tags","webkit2_41","-deps","-f","{{if .Module}}{{.Module.Path}}{{end}}",".","./cmd/vaultctl"],cwd=root,text=True).split())
modules=[m for m in modules if m["Path"] in used]
missing=[]
for m in modules:
 if m.get('Main'):continue
 folder=pathlib.Path(m.get('Dir','/nonexistent'))
 files=[p for p in folder.glob('*') if p.is_file() and p.name.upper().startswith(('LICENSE','COPYING','NOTICE'))]
 name=m['Path'].replace('/','_')+'@'+m.get('Version','unknown')
 dst=out/name;dst.mkdir(exist_ok=True)
 for p in files:shutil.copy2(p,dst/p.name)
 if not files:missing.append(m['Path'])
(out/'go-modules.json').write_text(json.dumps([{k:m.get(k) for k in ['Path','Version','Sum','GoVersion']} for m in modules],indent=2)+'\n')
lock=json.loads((root/'frontend/package-lock.json').read_text());packages=[]
for key,pkg in lock['packages'].items():
 if not key:continue
 packages.append({'path':key,'version':pkg.get('version'),'license':pkg.get('license'),'dev':pkg.get('dev',False),'integrity':pkg.get('integrity')})
 if key in ['node_modules/pdfjs-dist','node_modules/svelte']:
  for p in (root/'frontend'/key).glob('*'):
   if p.is_file() and p.name.upper().startswith(('LICENSE','NOTICE')):shutil.copy2(p,out/(pathlib.Path(key).name+'-'+p.name))
(out/'frontend-packages.json').write_text(json.dumps(packages,indent=2)+'\n')
shutil.copy2(pathlib.Path(subprocess.check_output(['go','env','GOROOT'],text=True).strip())/'LICENSE',out/'Go-LICENSE')
(out/'NOTICE.txt').write_text('mut is MPL-2.0. PDF.js is Apache-2.0. Svelte is MIT. Fonts are SIL OFL 1.1.\nGo and frontend dependency versions are included in JSON.\nGTK/WebKit libraries are dynamically linked. DEB uses system libraries; AppImage includes runtime notices and versions in licenses/system.\n')
if missing:raise SystemExit('Missing root license notices: '+', '.join(missing))
print('Collected licenses for',len(modules)-1,'Go modules and frontend runtime')
