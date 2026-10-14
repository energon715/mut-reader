#!/usr/bin/env python3
"""Launch an isolated package smoke test with a distribution's filesystem."""
import argparse,os,pathlib,subprocess,time
root=pathlib.Path(__file__).resolve().parent.parent
p=argparse.ArgumentParser();p.add_argument('--rootfs',required=True);p.add_argument('--appdir');p.add_argument('--binary');p.add_argument('--fixtures',required=True);p.add_argument('--inspector',action='store_true');args=p.parse_args()
fixtures=pathlib.Path(args.fixtures)
if not str(fixtures).startswith('/tmp/mut-'):raise SystemExit('Use dedicated /tmp/mut-* fixtures')
fixtures.mkdir(parents=True,exist_ok=True)
for name in ['cache','data','runtime']: (fixtures/name).mkdir(exist_ok=True)
(fixtures/'runtime').chmod(0o700)
cmd=['bwrap','--unshare-user','--uid',str(os.getuid()),'--gid',str(os.getgid()),'--bind',str(pathlib.Path(args.rootfs).resolve()),'/', '--proc','/proc','--dev-bind','/dev','/dev','--ro-bind','/sys','/sys','--bind',str(fixtures),str(fixtures),'--ro-bind','/tmp/.X11-unix','/tmp/.X11-unix','--dir',f'/run/user/{os.getuid()}/at-spi','--chdir','/', '--setenv','PATH','/usr/local/sbin:/usr/local/bin:/usr/sbin:/usr/bin:/sbin:/bin','--setenv','MUT_VAULT_PATH',str(fixtures/'default'),'--setenv','XDG_CACHE_HOME',str(fixtures/'cache'),'--setenv','XDG_DATA_HOME',str(fixtures/'data'),'--setenv','XDG_RUNTIME_DIR',str(fixtures/'runtime'),'--setenv','GDK_BACKEND','x11','--setenv','DBUS_SESSION_BUS_ADDRESS','unix:path='+str(fixtures/'runtime/session-bus'),'--setenv','NO_AT_BRIDGE','1']
if args.inspector:cmd += ['--setenv','WEBKIT_INSPECTOR_HTTP_SERVER','127.0.0.1:9223']
else:cmd += ['--unsetenv','WEBKIT_INSPECTOR_HTTP_SERVER']
if args.appdir:
 cmd += ['--ro-bind',str(pathlib.Path(args.appdir).resolve()),'/opt/mut','--','/opt/mut/AppRun']
elif args.binary:
 cmd += ['--',args.binary]
else:raise SystemExit('Choose --appdir or --binary')
bus=fixtures/'runtime/session-bus'
bus.unlink(missing_ok=True)
daemon=subprocess.Popen(['dbus-daemon','--session','--nofork','--address=unix:path='+str(bus)],stdout=subprocess.DEVNULL,stderr=subprocess.DEVNULL)
try:
 for _ in range(100):
  if bus.exists():break
  if daemon.poll() is not None:raise SystemExit('Private test D-Bus failed to start')
  time.sleep(0.02)
 subprocess.run(cmd,check=True)
finally:
 daemon.terminate();daemon.wait(timeout=5)
 bus.unlink(missing_ok=True)
