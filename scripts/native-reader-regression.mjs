// Requires native-smoke + author-owned long.epub in an isolated /tmp/mut-* directory.
import {connect} from './webkit-client.mjs';
import {readFile,writeFile} from 'node:fs/promises';
import assert from 'node:assert/strict';
const base=process.argv[2]; if(!base?.startsWith('/tmp/mut-'))throw Error('Use isolated test fixtures');
const {vault}=JSON.parse(await readFile(base+'/result.json','utf8'));
const c=await connect();const q=JSON.stringify;
const api=(name,args=[])=>c.asyncEvaluate(`window.go.app.App.${name}(${args.map(q).join(',')})`);
const click=selector=>c.evaluate(`document.querySelector(${q(selector)}).click()`);
const fill=(selector,value)=>c.evaluate(`(()=>{const e=document.querySelector(${q(selector)});e.value=${q(value)};e.dispatchEvent(new Event('input',{bubbles:true}));})()`);
let owned=false;const oldPaths=await c.evaluate('localStorage.getItem("mut.paths")');
async function login(){
 await c.wait('document.querySelector("#archive-key")');
 await fill('input[aria-label="Папка библиотеки"]',vault);
 await fill('#archive-key','native-test-password');await click('.login-page button[type=submit]');
 await c.wait('document.querySelector(".book-item")');
}
async function open(title){await c.evaluate(`[...document.querySelectorAll('.book-item')].find(e=>e.getAttribute('aria-label')?.startsWith(${q('Открыть '+title+' · ')})).click()`)}
async function geometry(contentSelector, requireScroll = true){
 const result=await c.evaluate(`(()=>{const area=document.querySelector('.reading-area'),paper=document.querySelector('.reading-page'),content=document.querySelector(${q(contentSelector)}),nav=document.querySelector('.page-navigation');const p=paper.getBoundingClientRect(),r=content.getBoundingClientRect(),n=nav.getBoundingClientRect();return {paperBottom:p.bottom,contentBottom:r.bottom,scrollHeight:area.scrollHeight,clientHeight:area.clientHeight,navTop:n.top,navBottom:n.bottom,viewport:innerHeight}})()`);
 if(requireScroll)assert(result.scrollHeight>result.clientHeight+100,'Fixture must require vertical scrolling: '+contentSelector+' '+result.scrollHeight+'/'+result.clientHeight);
 assert(result.paperBottom>=result.contentBottom-1,'Content must fit within page background');
 assert(result.navBottom<=result.viewport && result.navTop>=0,'Navigation visible in viewport');
 if(result.scrollHeight<=result.clientHeight+2)return;
 await c.evaluate('document.querySelector(".reading-area").scrollTop=100000');
 await c.wait('document.querySelector(".reading-area").scrollTop>0');
 const bottom=await c.evaluate(`(()=>{const a=document.querySelector('.reading-area').getBoundingClientRect();return !!document.elementFromPoint(a.left+a.width/2,a.bottom-24)?.closest('.reading-page')})()`);
 assert(bottom,'Page background must still be under content at bottom of scroll');
}
try{
 const defaults=await api('GetDefaultPaths');assert(defaults.library.startsWith(base+'/'));owned=true;
 await login();await api('ImportBook',[base+'/long.epub']);await api('Lock');await login();
 await open('Long reader fixture');await c.wait('document.querySelector(".prose p")');
 await geometry('.prose');
 await click('.page-turn[aria-label="Следующая страница"]');
 await c.wait('document.querySelector("#reader-page").value==="2" && document.querySelector(".reading-area").scrollTop===0');
 await click('.bookmark-toggle');await c.wait('document.querySelector(".bookmark-toggle").getAttribute("aria-pressed")==="true"');
 assert(await c.evaluate('document.querySelector(".reader-feedback").textContent.includes("добавлена")'));
 assert(await c.evaluate('document.querySelector(".bookmark-link").textContent.includes("Часть 2")'));
 await click('[aria-label="Закрыть оглавление и закладки"]');
 await click('[aria-label="Предыдущая страница"]');await c.wait('document.querySelector("#reader-page").value==="1"');
 await click('.bookmarks-toggle');await click('.bookmark-link');
 await c.wait('document.querySelector("#reader-page").value==="2" && document.querySelector(".reading-area").scrollTop===0');
 await click('.bookmarks-toggle');await click('.remove-bookmark');
 await c.wait('!document.querySelector(".bookmark-link")');
 const books=await api('ListBooks');assert.equal(books.find(b=>b.title==='Long reader fixture').bookmarks.length,0);
 await click('[aria-label="Закрыть оглавление и закладки"]');
 // Invalid keyboard/empty input must not turn the current page into NaN.
 await fill('#reader-page','');await c.evaluate('document.querySelector(".page-jump").requestSubmit()');
 await c.wait('document.querySelector("#reader-page").value==="2"');
 const size=await c.evaluate('({width:innerWidth,height:innerHeight})');
 const snapshot=await c.command('Page.snapshotRect',{x:0,y:0,...size,coordinateSystem:'Viewport'});
 await writeFile(base+'/screenshots/reader-controls.png',Buffer.from(snapshot.dataURL.split(',')[1],'base64'));
 await click('.back-library');await open('book');await c.wait('document.querySelector(".textLayer")?.textContent');
 await geometry('.pdf-sheet',false);
 // PDF can also scroll when zoomed; paper and controls must remain intact.
 for(let n=0;n<10;n++)await click('[aria-label="Увеличить масштаб"]');
 await c.wait('document.querySelector("canvas").height>1000');
 await geometry('.pdf-sheet');
 console.log('PASS long EPUB/PDF backgrounds, fixed navigation, scroll reset, bookmark add/open/delete, empty page input');
}catch(e){console.error(e.message);process.exitCode=1}
finally{
 if(owned)await api('Lock').catch(()=>{});
 await c.evaluate(oldPaths===null?'localStorage.removeItem("mut.paths")':`localStorage.setItem("mut.paths",${q(oldPaths)})`);c.close();
}
