// Exercise custom choices, keyboard interaction and modal closing in real WebKit.
import {connect} from './webkit-client.mjs';
import {readFile,writeFile} from 'node:fs/promises';
import assert from 'node:assert/strict';
const base=process.argv[2];if(!base?.startsWith('/tmp/mut-'))throw Error('Use isolated fixtures');
const {vault}=JSON.parse(await readFile(base+'/result.json','utf8'));
const c=await connect();const q=JSON.stringify;
const api=(name,args=[])=>c.asyncEvaluate(`window.go.app.App.${name}(${args.map(q).join(',')})`);
const click=selector=>c.evaluate(`document.querySelector(${q(selector)}).click()`);
const textClick=text=>c.evaluate(`[...document.querySelectorAll('button')].find(e=>e.textContent.trim()===${q(text)}).click()`);
const fill=(selector,value)=>c.evaluate(`(()=>{const e=document.querySelector(${q(selector)});e.value=${q(value)};e.dispatchEvent(new Event('input',{bubbles:true}));})()`);
const key=(selector,key)=>c.evaluate(`document.querySelector(${q(selector)}).dispatchEvent(new KeyboardEvent('keydown',{key:${q(key)},bubbles:true,cancelable:true}))`);
let owned=false;
const oldPaths=await c.evaluate('localStorage.getItem("mut.paths")');
const oldAppearance=await c.evaluate('localStorage.getItem("mut.appearance")');
try{
 const defaults=await api('GetDefaultPaths');assert(defaults.library.startsWith(base+'/'));owned=true;
 await c.wait('document.querySelector("#archive-key")');await fill('input[aria-label="Папка библиотеки"]',vault);await fill('#archive-key','native-test-password');await click('.login-page button[type=submit]');await c.wait('document.querySelector(".library-layout")');
 await textClick('Настройки');await c.wait('!!document.querySelector("dialog[open]")');await textClick('Чтение');
 assert.equal(await c.evaluate('document.querySelectorAll(".settings-layout select").length'),0);
 await click('#font-size');await c.wait('document.querySelector("#font-size-options")');
 assert(await c.evaluate(`(()=>{const m=document.querySelector('#font-size-options').getBoundingClientRect(),d=document.querySelector('dialog').getBoundingClientRect();return m.top>=d.top&&m.bottom<=d.bottom&&m.left>=0&&m.right<=innerWidth})()`),'Choice popover within dialog and viewport');
 await key('#font-size-options','End');await key('#font-size-options','Enter');
 await c.wait('document.querySelector("#font-size").textContent.includes("28")');
 assert.equal(await c.evaluate('JSON.parse(localStorage.getItem("mut.appearance")).fontSize'),28);
 await click('#line-height');await c.wait('document.querySelector("#line-height-options")');
 await key('#line-height-options','Escape');await c.wait('document.querySelector("#line-height").getAttribute("aria-expanded")==="false"');
 assert(await c.evaluate('!!document.querySelector("dialog[open]")'),'Escape should close just the popover');
 await click('#font-size');await c.wait('document.querySelector("#font-size-options")');
 await c.wait('Number(getComputedStyle(document.querySelector("#font-size-options")).opacity)>.99');
 const dim=await c.evaluate('({width:innerWidth,height:innerHeight})');
 const snap=await c.command('Page.snapshotRect',{x:0,y:0,...dim,coordinateSystem:'Viewport'});await writeFile(base+'/screenshots/settings-choices.png',Buffer.from(snap.dataURL.split(',')[1],'base64'));
 await key('#font-size-options','Escape');await c.wait('document.querySelector("#font-size").getAttribute("aria-expanded")==="false"');
 await click('dialog header [aria-label="Закрыть"]');
 const closing=await c.evaluate(`(()=>{const d=document.querySelector('dialog');return {exists:!!d,closing:d?.classList.contains('closing'),inert:d?.inert,reduced:matchMedia('(prefers-reduced-motion: reduce)').matches}})()`);
 if(!closing.reduced)assert(closing.exists&&closing.closing&&closing.inert,'Settings must animate while blocking input');
 await c.wait('!document.querySelector("dialog")');
 await textClick('Настройки');await c.wait('!!document.querySelector("dialog[open]")');await textClick('Хранилище');
 assert(await c.evaluate('[...document.querySelectorAll(".password-form input")].every(e=>e.autocomplete==="off")'));
 await api('Lock');await c.wait('!document.querySelector("dialog") && !!document.querySelector("#archive-key")');
 console.log('PASS styled choice bounds, keyboard selection/persistence, Escape, animated close, immediate close on lock, password autofill disabled');
}catch(e){console.error(e.message);process.exitCode=1}
finally{
 if(owned)await api('Lock').catch(()=>{});
 for(const [name,value] of [['mut.paths',oldPaths],['mut.appearance',oldAppearance]])await c.evaluate(value===null?`localStorage.removeItem(${q(name)})`:`localStorage.setItem(${q(name)},${q(value)})`);
 c.close();
}
