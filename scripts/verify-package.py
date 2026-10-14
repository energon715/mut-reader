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
