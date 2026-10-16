// Run "prepare", restart the same isolated app process, then run "verify".
import {connect} from './webkit-client.mjs';
import {readFile,writeFile} from 'node:fs/promises';
import assert from 'node:assert/strict';
const [mode,base]=process.argv.slice(2);
if(!base?.startsWith('/tmp/mut-'))throw Error('Use /tmp/mut-* test fixtures');
const {restored}=JSON.parse(await readFile(base+'/result.json','utf8'));
const c=await connect();const q=JSON.stringify;
async function fill(selector,value){await c.evaluate(`(()=>{const e=document.querySelector(${q(selector)});e.value=${q(value)};e.dispatchEvent(new Event('input',{bubbles:true}));e.dispatchEvent(new Event('change',{bubbles:true}));})()`)}
try {
 const defaults=await c.asyncEvaluate("window.go.app.App.GetDefaultPaths()");
 assert(defaults.library.startsWith(base+"/"),"Inspector must target isolated fixtures");
 if(mode==='prepare'){
  await c.asyncEvaluate('window.go.app.App.Lock()');
  await c.wait('document.querySelector("#archive-key")');
  await writeFile(base+'/old-paths.json',JSON.stringify(await c.evaluate('localStorage.getItem("mut.paths")')));
  await fill('input[aria-label="Папка библиотеки"]',restored);
  assert.equal(await c.evaluate('JSON.parse(localStorage.getItem("mut.paths")).library'),restored);
  console.log('PASS remembered path prepared; restart isolated app now');
 } else if(mode==='verify'){
  await c.wait('document.querySelector("#archive-key")');
  assert.equal(await c.evaluate(`document.querySelector(${q('input[aria-label="Папка библиотеки"]')}).value`),restored);
  await fill('#archive-key','native-test-password');
  await c.evaluate('document.querySelector("button[type=submit]").click()');
  await c.wait('document.querySelectorAll(".book-item").length===3');
  const books=await c.asyncEvaluate('window.go.app.App.ListBooks()');
  assert(books.every(b=>b.bookmarks.length===1));
  const pdf=books.find(b=>b.format==='PDF');assert.equal(pdf.position.locator,'v1:pdf:2');
  await c.evaluate(`[...document.querySelectorAll('.book-item')].find(e=>e.getAttribute('aria-label')?.startsWith(${q('Открыть '+pdf.title+' · ')})).click()`);
  await c.wait('document.querySelector(".textLayer")?.textContent.includes("page two")');
  console.log('PASS real process restart: remembered directory, locked startup, PDF resumes on page two, bookmarks retained');
  await c.asyncEvaluate('window.go.app.App.Lock()');
  const old=JSON.parse(await readFile(base+'/old-paths.json','utf8'));
  await c.evaluate(old===null?'localStorage.removeItem("mut.paths")':`localStorage.setItem("mut.paths",${q(old)})`);
 } else throw Error('mode: prepare|verify');
}finally{c.close()}
