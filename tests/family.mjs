import assert from 'node:assert/strict';
import {randomBytes} from 'node:crypto';

const base = process.env.TEST_ORIGIN || 'http://127.0.0.1:18095';
const email = process.env.TEST_ADMIN_EMAIL;
const password = process.env.TEST_ADMIN_PASSWORD;
assert(email && password, 'Use a disposable database and set TEST_ADMIN_EMAIL and TEST_ADMIN_PASSWORD.');

async function call(path, {cookie, data, status = 200, headers = {}} = {}) {
  if (cookie) headers.Cookie = cookie;
  let body = data;
  if (data) {
    headers.Origin = base;
    if (!(data instanceof FormData)) {
      headers['Content-Type'] = 'application/json';
      body = JSON.stringify(data);
    }
  }
  const response = await fetch(base + path, {method: data ? 'POST' : 'GET', headers, body});
  assert.equal(response.status, status, path + ' ' + (response.status === status ? '' : await response.text()));
  const content = response.headers.get('content-type')?.includes('application/json') ? await response.json() : await response.text();
  return {content, headers: response.headers, cookie: response.headers.get('set-cookie')?.split(';')[0]};
}
const act = (cookie, action, data, status) => call('/api/action', {cookie, data:{action,data}, status});
const portal = async cookie => (await call('/api/portal',{cookie})).content;
const family = async cookie => (await call('/api/family',{cookie})).content;
const save = (cookie,data,status) => call('/api/family/save',{cookie,data,status});
const filePath = id => '/files/family/'+id;
const admin = (await call('/api/login',{data:{email,password}})).cookie;
const employee = 'family-' + randomBytes(4).toString('hex');
const employeeEmail = employee+'@deraah.test';
const department = (await portal(admin)).departments[0].id;
await call('/api/register',{data:{email:employeeEmail,password,name:'موظف الاختبار',title:'تجربة',employee,department},status:201});
const staffID = (await portal(admin)).users.find(user=>user.email===employeeEmail).id;
await act(admin,'system.status',{id:staffID,status:'active'});
const staff = (await call('/api/login',{data:{email:employeeEmail,password}})).cookie;
assert((await portal(staff)).permissions['family.comment']);
assert.deepEqual((await family(staff)).bulletins,[]);
await call('/family');
await call('/api/family',{status:401});

const pdf = () => new Blob(['%PDF-1.4\n%%EOF\n'],{type:'application/pdf'});
const mp4 = () => new Blob([Buffer.from([0,0,0,16,102,116,121,112,105,115,111,109,0,0,0,0])],{type:'video/mp4'});
const first = new FormData();
first.set('title','النشرة الأولى');
first.set('summary','النشرة الورقية والمقطع');
first.set('pdf',pdf(),'issue.pdf');
first.set('video',mp4(),'issue.mp4');
await save(staff,first,403);
await save(admin,first);
let firstIssue = (await family(staff)).bulletins[0];
assert.equal(firstIssue.title,'النشرة الأولى');
const pdfPath = filePath(firstIssue.pdf.id);
const videoPath = filePath(firstIssue.video.id);
assert.equal((await call(pdfPath,{cookie:staff})).headers.get('x-frame-options'),'SAMEORIGIN');
assert.match((await call(videoPath,{cookie:staff,headers:{Range:'bytes=0-3'},status:206})).headers.get('content-range'),/^bytes 0-3\//);
await call(pdfPath,{status:401});

const welcome = new FormData();
welcome.set('mode','intro');
welcome.set('video',mp4(),'intro.mp4');
await save(admin,welcome);
assert.equal((await family(staff)).intro.name,'intro.mp4');
const second = new FormData();
second.set('title','النشرة الثانية');
second.set('video',mp4(),'second.mp4');
await save(admin,second);
const latest = (await family(staff)).current;
assert.notEqual(latest,firstIssue.id);
await act(admin,'family.feature',{id:firstIssue.id});
assert.equal((await family(staff)).current,firstIssue.id);

const issue = {id:firstIssue.id};
await act(staff,'family.comment',{...issue,text:'أعجبتني النشرة'});
firstIssue = (await family(staff)).bulletins.find(b=>b.id===firstIssue.id);
assert.equal(firstIssue.comments[0].name,'موظف الاختبار');
assert(!('Author' in firstIssue.comments[0]));
const comment = {...issue,comment:firstIssue.comments[0].id};
await act(staff,'family.removeComment',comment,403);
await act(admin,'family.removeComment',comment);
assert.equal((await family(staff)).bulletins.find(b=>b.id===issue.id).comments.length,0);

await act(admin,'family.hide',{...issue,hidden:true});
assert.equal((await family(staff)).current,latest);
assert(!(await family(staff)).bulletins.some(b=>b.id===issue.id));
await call(pdfPath,{cookie:staff,status:404});
await act(staff,'family.comment',{...issue,text:'مخفية'},404);
await act(admin,'family.hide',{...issue,hidden:false});
await call(pdfPath,{cookie:staff});

await act(staff,'system.program',{program:'family',open:false},403);
for (const [program, route] of [['family','family'],['trainer','trainer'],['voice','voice']]) {
  await act(admin,'system.program',{program,open:false});
  assert.equal((await portal(staff)).programs[program],false);
  assert.equal((await call('/api/public')).content.programs[program],false);
  await call('/api/'+route,{cookie:staff,status:404});
  if (program === 'family') {
    await call(pdfPath,{cookie:staff,status:404});
    await save(admin,first,404);
  }
  if (program === 'voice') await call('/api/survey/submit',{data:{token:'unused',ratings:{},enps:10},status:404});
  await act(admin,'system.program',{program,open:true});
  await call('/api/'+route,{cookie:staff});
}
assert.equal((await family(staff)).bulletins.length,2);
await act(admin,'system.roles',{id:staffID,trainer:'employee',voice:'employee',family:'none',grants:{}});
await call('/api/family',{cookie:staff,status:403});
await call(pdfPath,{cookie:staff,status:403});
await act(admin,'family.delete',issue);
assert(!(await family(admin)).bulletins.some(b=>b.id===issue.id));
await call(pdfPath,{cookie:admin,status:404});
console.log('Verified Family uploads, archive, comments, permissions, file access, and program switches.');
