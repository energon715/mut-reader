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
 assert 'libwebkit2gtk-4.1-0 (>= 2.54.0)' in control
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

# Inspect the actual AppImage, including its payload rather than just the staging directory.
import subprocess,tempfile
appimage=out/f'mut-{version}-linux-x86_64.AppImage'
if appimage.exists():
 with tempfile.TemporaryDirectory(prefix='mut-appimage-check-') as temp:
  subprocess.run([str(appimage),'--appimage-extract'],cwd=temp,stdout=subprocess.DEVNULL,check=True)
  app=pathlib.Path(temp)/'squashfs-root'
  assert (app/'AppRun').is_file() and not (app/'AppRun').is_symlink()
  for name in ['mut','vaultctl']:
   binary=app/'usr/bin'/name
   assert binary.read_bytes()[:5]==b'\x7fELF\x02'
   assert subprocess.check_output(['go','tool','buildid',str(binary)]).strip()==subprocess.check_output(['go','tool','buildid',str(root/'build/bin'/name)]).strip(), 'Wrong packaged Go binary: '+name
  for name in ['WebKitWebProcess','WebKitNetworkProcess','WebKitGPUProcess']:
   assert (app/'usr/lib/x86_64-linux-gnu/webkit2gtk-4.1'/name).is_file()
  for name in ['usr/lib/libmut-webkit-relocate.so','usr/share/glib-2.0/schemas/gschemas.compiled','usr/share/doc/mut/licenses/system/packages.json']:
   assert (app/name).is_file()
  runtime=json.loads((app/'usr/share/doc/mut/licenses/system/packages.json').read_text())
  assert any(p['package']=='libwebkit2gtk-4.1-0' and tuple(map(int,p['version'].split('-',1)[0].split('.')[:2])) >= (2,54) for p in runtime)
  assert b'WEBKIT_FORCE_SANDBOX=1' in (app/'AppRun').read_bytes()
  assert b'DISABLE_SANDBOX' not in (app/'AppRun').read_bytes()
  cli=subprocess.check_output([str(app/'AppRun'),'--vaultctl','--help'],text=True)
  assert 'offline management' in cli
 print('PASS actual AppImage payload, x86-64 Go build IDs, matching WebKit helpers, sandbox launcher, licenses and CLI')
for line in (out/'SHA256SUMS').read_text().splitlines():
 checksum,name=line.split('  ',1)
 assert pathlib.Path(name).name==name
 assert hashlib.sha256((out/name).read_bytes()).hexdigest()==checksum
print('PASS release checksums')

sources=out/f'mut-{version}-runtime-sources.tar.gz'
if sources.exists():
 import re
 hashes={};descriptions=[];source_inventory=None
 with tarfile.open(sources,mode='r|gz') as t:
  for member in t:
   assert member.isfile() and not member.name.startswith('/') and '..' not in pathlib.PurePosixPath(member.name).parts
   stream=t.extractfile(member);name=pathlib.PurePosixPath(member.name).name
   if name.endswith('.dsc') or name=='runtime-packages.json':
    content=stream.read();digest=hashlib.sha256(content).hexdigest()
    if name.endswith('.dsc'):descriptions.append(content.decode())
    else:source_inventory=json.loads(content)
   else:digest=hashlib.file_digest(stream,'sha256').hexdigest()
   hashes[name]=(digest,member.size)
 for description in descriptions:
  match=re.search(r'Checksums-Sha256:\n((?:[ \t].*\n)+)',description);assert match
  for line in match[1].splitlines():
   digest,size,name=line.split();assert hashes[name]==(digest,int(size)), 'Source checksum mismatch: '+name
 expected=json.loads((root/'build/appimage/mut.AppDir/usr/share/doc/mut/licenses/system/packages.json').read_text())
 assert {(p['package'],p['version']) for p in source_inventory}=={(p['package'],p['version']) for p in expected}
 assert len(descriptions)==len({(p['source'],p['sourceVersion']) for p in expected})
 print('PASS corresponding runtime source inventory and all Debian source checksums')
