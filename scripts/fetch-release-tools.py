#!/usr/bin/env python3
"""Download pinned, SHA-256 checked build inputs without installing system tools."""
import hashlib,json,pathlib,sys,urllib.request
root=pathlib.Path(__file__).resolve().parent.parent
cache=root/'build/release-tools';cache.mkdir(parents=True,exist_ok=True)
lock=json.loads((root/'packaging/tools.lock.json').read_text())
for name in sys.argv[1:] or lock:
 item=lock[name];dest=cache/item['file']
 if dest.exists() and hashlib.sha256(dest.read_bytes()).hexdigest()==item['sha256']:
  print('Cached:',name,flush=True);continue
 temp=dest.with_suffix(dest.suffix+'.part')
 print('Downloading:',name,flush=True)
 request=urllib.request.Request(item['url'],headers={'User-Agent':'mut-release-builder/1'})
 with urllib.request.urlopen(request,timeout=60) as response,temp.open('wb') as f:
  while chunk:=response.read(1<<20):f.write(chunk)
 if hashlib.sha256(temp.read_bytes()).hexdigest()!=item['sha256']:
  temp.unlink();raise SystemExit('SHA-256 mismatch for '+name+'; update the pin only after verifying upstream')
 temp.replace(dest);dest.chmod(0o755 if name in ['linuxdeploy','appimagetool','runtime'] else 0o644)
 print('Verified:',name,flush=True)
