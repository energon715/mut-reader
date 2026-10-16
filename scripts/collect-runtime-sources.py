#!/usr/bin/env python3
"""Ship corresponding Debian sources alongside the AppImage runtime notices."""
import gzip,json,os,pathlib,subprocess,tarfile
root=pathlib.Path(__file__).resolve().parent.parent
inventory=root/'build/appimage/mut.AppDir/usr/share/doc/mut/licenses/system/packages.json'
packages=json.loads(inventory.read_text())
folder=root/'build/runtime-sources';folder.mkdir(parents=True,exist_ok=True)
# Only the disposable build environment's APT configuration is changed.
for source in pathlib.Path('/etc/apt/sources.list.d').glob('*.sources'):
 text=source.read_text().replace('Types: deb\n','Types: deb deb-src\n');source.write_text(text)
subprocess.run(['apt-get','-o','APT::Sandbox::User=root','update'],check=True)
sources=sorted({(p['source'],p['sourceVersion']) for p in packages})
for name,version in sources:
 print('Corresponding source:',name,version,flush=True)
 subprocess.run(['apt-get','-o','APT::Sandbox::User=root','source','--download-only',name+'='+version],cwd=folder,check=True)
(folder/'runtime-packages.json').write_text(json.dumps(packages,indent=2)+'\n')
(folder/'README.txt').write_text('Corresponding Debian sources for the bundled AppImage libraries and data.\nExtract source packages with dpkg-source -x NAME.dsc. Relocation hook source is in the AppImage licenses/system directory and the mut repository packaging/webkit-relocate.c.\n')
version=json.loads((root/'frontend/package.json').read_text())['version']
out=root/'dist'/f'mut-{version}-runtime-sources.tar.gz'
epoch=int(os.environ.get('SOURCE_DATE_EPOCH','0'))
with out.open('wb') as dest,gzip.GzipFile(fileobj=dest,mode='wb',mtime=epoch,filename='') as gz,tarfile.open(fileobj=gz,mode='w') as t:
 for p in sorted(folder.iterdir()):
  if not p.is_file():continue
  i=t.gettarinfo(p,arcname='runtime-sources/'+p.name);i.uid=i.gid=0;i.uname=i.gname='root';i.mtime=epoch
  with p.open('rb') as f:t.addfile(i,f)
print(out,flush=True)
