import assert from 'node:assert/strict';
import {randomBytes} from 'node:crypto';

const base = process.env.TEST_ORIGIN || 'http://127.0.0.1:18093';
const adminEmail = process.env.TEST_ADMIN_EMAIL;
const password = process.env.TEST_ADMIN_PASSWORD;
assert(adminEmail && password, 'Set TEST_ADMIN_EMAIL and TEST_ADMIN_PASSWORD for a disposable database.');
const run = randomBytes(4).toString('hex');

async function request(path, {method = 'GET', cookie, body, status = 200, headers = {}} = {}) {
  if (cookie) headers.Cookie = cookie;
  if (method !== 'GET') headers.Origin ||= base;
  if (body && !(body instanceof FormData)) {
    headers['Content-Type'] = 'application/json';
    body = JSON.stringify(body);
  }
  const response = await fetch(base + path, {method, body, headers});
  const text = await response.text();
  assert.equal(response.status, status, method + ' ' + path + ' ' + text.slice(0, 340));
  return {data: response.headers.get('content-type')?.includes('application/json') ? JSON.parse(text) : text,
    cookie: response.headers.get('set-cookie')?.split(';')[0]};
}
const action = (cookie, name, data, status = 200) => request('/api/action', {method: 'POST', cookie, body: {action: name, data}, status});
const login = (email, status = 200) => request('/api/login', {method: 'POST', body: {email, password}, status});
const read = async (area, cookie) => (await request('/api/' + area, {cookie})).data;

assert.equal((await read('public')).registration, true);
await request('/health');
const page = await request('/');
const script = page.data.match(/src="([^"]+\.js)"/)?.[1];
assert(script, 'The app must serve its built frontend');
await request(script);
await request('/api/portal', {status: 401});
const admin = (await login(adminEmail)).cookie;
const department = (await read('portal', admin)).departments[0].id;

async function register(label) {
  const employee = label + '-' + run;
  const email = employee + '@deraah.test';
  await request('/api/register', {method: 'POST', body: {name: label, title: 'متخصص تدريب', employee, email, department, password}, status: 201});
  await login(email, 403);
  const user = (await read('portal', admin)).users.find(u => u.email === email);
  assert.equal(user.status, 'pending');
  assert(!('secret' in user));
  await action(admin, 'system.status', {id: user.id, status: 'active'});
  return {id: user.id, cookie: (await login(email)).cookie};
}

const trainer = await register('trainer');
const panel = await register('panel');
const employee = await register('employee');
await action(admin, 'system.roles', {id: panel.id, trainer: 'reviewer', voice: 'none', grants: {}});
assert((await read('portal', panel.cookie)).permissions['trainer.evaluate']);
await request('/api/voice', {cookie: panel.cookie, status: 403});
await action(employee.cookie, 'system.registration', {open: false}, 403);
await request('/api/action', {method: 'POST', cookie: admin, headers: {Origin: 'http://example.net'}, body: {action: 'system.registration', data: {open: false}}, status: 403});

const date = new Intl.DateTimeFormat('sv-SE', {timeZone: 'Asia/Riyadh'}).format(new Date());
const campaign = {title: 'QA ' + run, goal: 'Practice teaching', policy: 'Show a practical example', opens: date, deadline: date, status: 'open',
  panel: [panel.id], rubric: [{id: 'content', label: 'Content', weight: 100}],
  questions: [{id: 'example', label: 'Which example will you use?', required: true}]};
await action(admin, 'trainer.campaign', campaign);
const campaignID = (await read('trainer', trainer.cookie)).campaigns.find(c => c.title === campaign.title).id;
const file = new FormData();
file.set('file', new Blob(['%PDF-1.4\n%%EOF'], {type: 'application/pdf'}), 'sample.pdf');
const upload = await request('/api/upload', {method: 'POST', body: file, cookie: trainer.cookie, status: 201});
const applicationFields = {campaign: campaignID, topic: 'Teaching', outline: 'Example and exercise', material: upload.data.id};
await action(trainer.cookie, 'trainer.apply', applicationFields, 400);
await action(trainer.cookie, 'trainer.apply', {...applicationFields, answers: {example: 'A real scenario'}});
const application = (await read('trainer', admin)).applications.find(a => a.campaign === campaignID);
assert.equal(application.answers.example, 'A real scenario');
await request('/files/' + upload.data.id, {cookie: employee.cookie, status: 403});
await request('/files/' + upload.data.id, {cookie: panel.cookie});
await action(admin, 'trainer.decide', {id: application.id, status: 'accepted', note: 'Chosen'}, 400);
await action(employee.cookie, 'trainer.evaluate', {id: application.id, ratings: {content: 5}}, 403);
await action(admin, 'trainer.appointment', {id: application.id, appointment: date + 'T12:00'});
await action(panel.cookie, 'trainer.evaluate', {id: application.id, ratings: {content: 4}, note: 'Good case'});
assert.equal((await read('trainer', employee.cookie)).ranking.find(r => r.id === trainer.id).scores.panel, 80);
assert(!('ranking' in await read('portal', employee.cookie)));
await action(admin, 'trainer.decide', {id: application.id, status: 'accepted', note: 'Chosen'});
assert((await read('trainer', admin)).departments.some(d => d.campaign === campaignID && d.applied === 1 && d.accepted === 1));

const session = {application: application.id, title: 'Session ' + run, objective: 'Practice one skill', at: date + 'T13:00', duration: 30, status: 'open',
  quiz: [{question: 'Choose one', options: ['A', 'B'], answer: 0}]};
await action(admin, 'trainer.session', session);
const listed = (await read('trainer', employee.cookie)).sessions.find(s => s.title === session.title);
assert(!('answer' in listed.quiz[0]));
const id = listed.id;
await action(employee.cookie, 'trainer.feedback', {id, ratings: [5, 5, 4, 4]});
await action(employee.cookie, 'trainer.feedback', {id, ratings: [5, 5, 4, 4]}, 400);
await action(employee.cookie, 'trainer.quiz', {id, answers: [0]});
await action(admin, 'trainer.rate', {id, score: 100, note: 'Strong practical lesson'});
await action(admin, 'trainer.session', {...session, id, status: 'closed'});
assert.deepEqual((await read('trainer', trainer.cookie)).sessions.find(s => s.id === id).feedbackDimensions, [100, 100, 80, 80]);
const row = (await read('trainer', employee.cookie)).ranking.find(r => r.id === trainer.id);
assert.deepEqual(row.scores, {panel: 80, survey: 90, quiz: 100, manager: 100, combined: 89.5});
const board = (await read('trainer', admin)).board;
await action(admin, 'trainer.board', {...board, visible: false});
assert.deepEqual((await read('trainer', employee.cookie)).ranking, []);
await action(admin, 'trainer.board', board);

const voice = await read('voice', admin);
assert.equal(voice.surveys.length, 2);
const semi = voice.surveys.find(s => s.cycle === 'semi');
assert(semi.questions.length > 0);
const survey = {id: semi.id, title: semi.title, benefit: semi.benefit, questions: semi.questions};
await action(admin, 'voice.survey', {...survey, status: 'open'});
await request('/api/survey?token=' + semi.token);
const ratings = Object.fromEntries(semi.questions.map(q => [q.id, 4]));
await request('/api/survey/submit', {method: 'POST', body: {token: semi.token, ratings, enps: 9}, status: 201});
await action(admin, 'voice.survey', {...survey, status: 'closed'});
assert.deepEqual((await read('voice', employee.cookie)).surveys, []);
await action(admin, 'voice.publish', {id: semi.id, published: true});
await action(admin, 'voice.plan', {survey: semi.id, theme: semi.questions[0].theme, action: 'Review the process', owner: 'HR', stage: 'working', progress: 50, published: true});
const published = await read('voice', employee.cookie);
assert.equal(published.surveys[0].results.count, 1);
assert.equal(published.plans[0].progress, 50);
assert(!JSON.stringify(published).includes(semi.token));

await action(admin, 'system.status', {id: employee.id, status: 'suspended'});
await request('/api/portal', {cookie: employee.cookie, status: 401});
await action(admin, 'system.registration', {open: false});
assert.equal((await read('public')).registration, false);
await action(admin, 'system.registration', {open: true});
console.log('Verified accounts, permissions, applications, material access, committee, sessions, leaderboard, anonymous surveys, and publication.');
