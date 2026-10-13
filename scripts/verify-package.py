#!/usr/bin/env python3
"""Inspect our deb/tar archives without installing or extracting system files."""
import hashlib,io,json,pathlib,tarfile
root=pathlib.Path(__file__).resolve().parent.parent
version=json.loads((root/'frontend/package.json').read_text())['version']
out=root/'dist';deb=out/f'mut-{version}-amd64.deb';raw=deb.read_bytes()
assert raw[:8]==b'!<arch>\n';pos=8;members={}
while pos<len(raw):
 header=raw[pos:pos+60];assert header[-2:]==b'`\n';pos+=60
 name=header[:16].decode().strip().rstrip('/');size=int(header[48:58]);members[name]=raw[pos:pos+size];pos+=size+(size%2)
assert set(members)=={'debian-binary','control.tar.gz','data.tar.gz'}
assert members['debian-binary']==b'2.0\n'
with tarfile.open(fileobj=io.BytesIO(members['control.tar.gz']),mode='r:gz') as t:
 control=t.extractfile('./control').read().decode();assert 'Package: mut\n' in control and 'Architecture: amd64\n' in control
sources=[io.BytesIO(members['data.tar.gz']),io.BytesIO((out/f'mut-{version}-linux-amd64.tar.gz').read_bytes())]
for source in sources:
 with tarfile.open(fileobj=source,mode='r:gz') as t:
  files={m.name.removeprefix('./'):m for m in t.getmembers()}
  for name,m in files.items():
   assert not name.startswith('/') and '..' not in pathlib.PurePosixPath(name).parts
   assert not m.issym() and not m.islnk()
  for binary in ['mut','vaultctl']:
   m=files['usr/bin/'+binary];assert m.mode&0o111
   assert hashlib.sha256(t.extractfile(m).read()).digest()==hashlib.sha256((root/'build/bin'/binary).read_bytes()).digest()
  for name in ['usr/share/applications/mut.desktop','usr/share/doc/mut/LICENSE','usr/share/doc/mut/RECOVERY.md','usr/share/doc/mut/format-v1.md','usr/share/doc/mut/licenses/go/Go-LICENSE']:
   assert name in files
print('PASS deb structure, both archives, exact binary hashes, licenses and docs')
