import React,{useState} from 'react';
import {Avatar,Badge,Empty,Field,Form,Header,Modal,ProfileFields,metricLabels,programs,profile,readProfile,msg,type Act,type Board,type Portal,type Program,type User} from './ui';

const accountStatus:Record<string,string>={active:'نشط',pending:'بانتظار الموافقة',suspended:'موقوف'};
export function BoardEditor({board,act,onClose}:{board:Board;act:Act;onClose:()=>void}){
 return <Modal title="إعدادات لوحة المنافسة" onClose={onClose}><Form after={onClose} onSave={f=>act('trainer.board',{title:f.get('title'),visible:f.has('visible'),weights:Object.fromEntries(['panel','survey','quiz','manager'].map(k=>[k,Number(f.get(k))]))})}><Field label="عنوان اللوحة"><input name="title" required defaultValue={board.title}/></Field><label className="check"><input type="checkbox" name="visible" defaultChecked={board.visible}/>إظهار اللوحة للموظفين</label><p className="muted">اكتب صفر للبند اللي عايز تستبعده. مجموع الأوزان 100.</p><div className="form-grid">{['panel','survey','quiz','manager'].map(k=><Field key={k} label={metricLabels[k]}><input name={k} type="number" min="0" max="100" required defaultValue={board.weights[k]||0}/></Field>)}</div></Form></Modal>
}
export function Admin({data,act}:{data:Portal;act:Act}){
 const [tab,setTab]=useState('accounts'),[filter,setFilter]=useState(''),[status,setStatus]=useState('all'),[selected,setSelected]=useState<User>(),[mode,setMode]=useState('profile'),[error,setError]=useState(''),[saving,setSaving]=useState(false);
 const run=async(action:string,payload:unknown)=>{setSaving(true);setError('');try{await act(action,payload)}catch(e){setError(msg(e))}finally{setSaving(false)}};
 const users=(data.users||[]).filter(u=>(status==='all'||u.status===status)&&[u.name,u.email,u.employee].join(' ').toLowerCase().includes(filter.toLowerCase())).sort((a,b)=>Number(b.status==='pending')-Number(a.status==='pending')||a.name.localeCompare(b.name,'ar'));
 const open=(u:User,m:string)=>{setSelected(u);setMode(m)};
 return <section className="card section" id="system"><Header label="مكتب مدير النظام" title="إدارة البوابة" subtitle="الحسابات والموافقات وصلاحيات البرامج من مكان واحد."/>
  <div className="tabset" data-guide="settings-tabs">{[['accounts','الحسابات والموافقات'],['roles','الصلاحيات'],['programs','إتاحة البرامج'],['departments','الإدارات'],['audit','سجل العمليات']].map(([k,v])=><button key={k} className={tab===k?'selected':''} onClick={()=>setTab(k)}>{v}</button>)}</div>
  {error&&<p className="error" role="alert">{error}</p>}
  {tab==='accounts'&&<DemoData installed={Boolean(data.demoInstalled)} act={act}/>}
  {['accounts','roles'].includes(tab)&&<><div className="toolbar"><input placeholder="بحث بالاسم أو البريد أو الرقم" aria-label="بحث الحسابات" value={filter} onChange={e=>setFilter(e.target.value)}/><select aria-label="حالة الحساب" value={status} onChange={e=>setStatus(e.target.value)}><option value="all">كل الحالات</option>{Object.entries(accountStatus).map(([k,v])=><option key={k} value={k}>{v}</option>)}</select>{tab==='accounts'&&<button className="btn outline" disabled={saving} onClick={()=>run('system.registration',{open:!data.registration})}>{data.registration?'إقفال التسجيل':'فتح التسجيل'}</button>}<Badge tone={data.registration?'good':''}>{data.registration?'التسجيل مفتوح':'التسجيل مغلق'}</Badge></div>
  {users.length?<div className="table-wrap"><table><thead><tr><th>الموظف</th><th>الحالة</th><th>الإجراءات</th></tr></thead><tbody>{users.map(u=><tr key={u.id}><td><div className="person"><Avatar name={u.name} photo={u.photo}/><div><b>{u.name}</b><small>{u.email} · {u.employee}</small></div></div></td><td><Badge tone={u.status==='active'?'good':u.status==='pending'?'wait':'bad'}>{accountStatus[u.status]}</Badge>{u.system&&<Badge>مدير النظام</Badge>}</td><td><div className="actions">{tab==='roles'?<button className="link" onClick={()=>open(u,'roles')}>توزيع الصلاحيات</button>:<><button className="link" onClick={()=>open(u,'profile')}>عرض / تعديل</button><button className="link" onClick={()=>open(u,'password')}>كلمة المرور</button>{u.id!==data.me.id&&<><button className="link" onClick={()=>open(u,u.status==='active'?'suspend':'activate')}>{u.status==='pending'?'موافقة':u.status==='active'?'إيقاف':'إعادة التفعيل'}</button><button className="link danger" onClick={()=>open(u,'delete')}>حذف</button></>}</>}</div></td></tr>)}</tbody></table></div>:<Empty>لا توجد حسابات مطابقة.</Empty>}</>}
  {tab==='programs'&&<div><p className="muted">البرنامج المغلق يختفي من البوابة ويتوقف الرابط المباشر. تقدر تفتحه تاني بنفس البيانات.</p>{(Object.keys(programs) as Program[]).map(key=><div className="item row between" key={key}><div><b>{programs[key]}</b> <Badge tone={data.programs[key]?'good':'wait'}>{data.programs[key]?'مفتوح':'مغلق'}</Badge></div><button className="btn" disabled={saving} onClick={()=>void run('system.program',{program:key,open:!data.programs[key]})}>{data.programs[key]?'إقفال البرنامج':'فتح البرنامج'}</button></div>)}</div>}
  {tab==='departments'&&<><Form label="إضافة إدارة" onSave={f=>act('system.department',{name:f.get('name')})}><Field label="اسم الإدارة الجديدة"><input name="name" required maxLength={80}/></Field></Form><div className="dept-grid">{data.departments.map(d=><form key={d.id} onSubmit={e=>{e.preventDefault();const f=new FormData(e.currentTarget);void run('system.department',{id:d.id,name:f.get('name')})}}><input name="name" aria-label={d.name} defaultValue={d.name} required maxLength={80}/><button className="btn outline" disabled={saving}>حفظ</button></form>)}</div></>}
  {tab==='audit'&&<div className="table-wrap"><table><thead><tr><th>الوقت</th><th>المستخدم</th><th>الإجراء</th></tr></thead><tbody>{[...(data.audit||[])].reverse().map((a,i)=><tr key={i}><td>{new Date(a.at).toLocaleString('ar-SA',{calendar:'gregory',timeZone:'Asia/Riyadh'})}</td><td>{a.by}</td><td>{auditLabel(a.action)}</td></tr>)}</tbody></table>{!(data.audit||[]).length&&<Empty>لا توجد عمليات مسجلة.</Empty>}</div>}
  {selected&&<Modal title={`${modeLabel(mode)} · ${selected.name}`} onClose={()=>setSelected(undefined)}>
  {mode==='profile'&&<Form after={()=>setSelected(undefined)} onSave={async f=>act('system.profile',{id:selected.id,profile:await readProfile(f,profile(selected))})}><ProfileFields value={profile(selected)} departments={data.departments}/></Form>}
  {mode==='password'&&<Form after={()=>setSelected(undefined)} onSave={f=>act('system.password',{id:selected.id,password:f.get('password')})}><p className="muted">تغيير كلمة المرور ينهي جلسات الحساب الحالية.</p><Field label="كلمة المرور الجديدة"><input type="password" name="password" minLength={12} required autoComplete="new-password"/></Field></Form>}
  {mode==='roles'&&<PermissionEditor key={selected.id} user={selected} self={selected.id===data.me.id} values={data.userPermissions?.[selected.id]||{}} labels={data.permissionsCatalog||{}} save={payload=>act('system.permissions',payload)} done={()=>setSelected(undefined)}/>}
  {['activate','suspend','delete'].includes(mode)&&<Form label={modeLabel(mode)} after={()=>setSelected(undefined)} onSave={()=>mode==='delete'?act('system.delete',{id:selected.id}):act('system.status',{id:selected.id,status:mode==='activate'?'active':'suspended'})}><p>{mode==='delete'?'سيُحذف الحساب وتُنهى جلساته. تُحتفظ بسجلات البرامج اللازمة للتقارير دون ملف الحساب.':mode==='activate'?'سيتمكن الموظف من تسجيل الدخول حسب الصلاحيات المحددة.':'يتوقف دخول الموظف وتُنهي المنصة جلساته الحالية.'}</p></Form>}
  </Modal>}
 </section>
}
function PermissionEditor({user,self,values,labels,save,done}:{user:User;self:boolean;values:Record<string,boolean>;labels:Record<string,string>;save:(payload:unknown)=>Promise<void>;done:()=>void}){
 const [grants,setGrants]=useState(values),[system,setSystem]=useState(user.system);
 const toggle=(key:string,label:string,checked:boolean,change:(value:boolean)=>void,disabled=false)=><label className="permission-row" key={key}>
  <span>{label}</span><span className="toggle-track" dir="ltr"><input type="checkbox" role="switch" checked={checked} disabled={disabled} onChange={e=>change(e.target.checked)}/><span aria-hidden="true"/></span>
 </label>;
 return <Form label="حفظ الصلاحيات" after={done} onSave={()=>save({id:user.id,grants,system})}>
  {toggle('system','إدارة النظام بالكامل',system,setSystem,self)}
  <p className="muted">فعّل دخول القسم أولاً، ثم اختر الإجراءات المطلوبة للحساب.</p>
  {(Object.keys(programs) as Program[]).map(program=><fieldset className="permission-group" key={program}>
   <legend>{program==='family'?'النشرة · Deraah Family':programs[program]}</legend>
   {Object.entries(labels).filter(([key])=>key.startsWith(program+'.')).sort(([a],[b])=>Number(b.endsWith('.view'))-Number(a.endsWith('.view'))).map(([key,label])=>{
    const unavailable=!key.endsWith('.view')&&!grants[program+'.view'];
    return toggle(key,label,system||(!unavailable&&!!grants[key]),enabled=>setGrants(old=>({...old,[key]:enabled})),system||unavailable);
   })}
  </fieldset>)}
 </Form>;
}
function DemoData({installed,act}:{installed:boolean;act:Act}){
 const [open,setOpen]=useState(false);
 return <section className="item demo-data">
  <h3>البيانات التجريبية</h3>
  {installed?<><p role="status">تمت إضافة 10 حسابات تجريبية. ابحث عن «تجريبي» في الحسابات أدناه.</p>
   <p>لتجربة الاختبار البعدي، سجّل الدخول بـ <bdi>demo07@deraah.test</bdi> وكلمة المرور التي اخترتها، وافتح «الجلسات والاختبارات» في سفراء المعرفة.</p>
   <p>تجد نتائج العشر مشاركات في نبض درعه، ورابط المشاركة اليدوية في «جولات الاستبيان». افتح «أرشيف النشرات» في Family لتصفح الأعداد التجريبية.</p>
  </>:<><p>أضف 10 حسابات، وجولة استبيان بنتائج، وترشيحات وجلسة باختبار، ونشرتين للتجربة في البرامج الثلاثة.</p>
   {open?<Form label="إضافة البيانات" after={()=>setOpen(false)} onSave={f=>act('system.demo',{password:f.get('password')})}>
    <p>اختر كلمة مرور للحسابات التجريبية فقط. ستظهر السجلات الموسومة «تجريبي» في البرامج حسب صلاحيات المستخدمين.</p>
    <Field label="كلمة مرور الحسابات التجريبية"><input name="password" type="password" minLength={12} maxLength={256} required autoComplete="new-password"/></Field>
   </Form>:<button type="button" className="btn outline" onClick={()=>setOpen(true)}>إضافة البيانات التجريبية</button>}
  </>}
 </section>;
}
function modeLabel(k:string){return ({profile:'ملف الحساب',roles:'توزيع الصلاحيات',password:'تغيير كلمة المرور',activate:'تفعيل الحساب',suspend:'إيقاف الحساب',delete:'حذف الحساب'} as Record<string,string>)[k]||k}
function auditLabel(k:string){return ({register:'طلب تسجيل','system.demo':'إضافة البيانات التجريبية', 'system.profile':'تعديل ملف موظف','system.status':'تغيير حالة حساب','system.roles':'تحديث الصلاحيات','system.password':'تغيير كلمة المرور','system.delete':'حذف حساب','system.department':'تحديث إدارة','system.registration':'تغيير إتاحة التسجيل','system.program':'تغيير إتاحة برنامج','family.save':'حفظ نشرة','family.intro':'تحديث فيديو الترحيب','family.hide':'تغيير إتاحة نشرة','family.feature':'تعيين النشرة الحالية','family.delete':'حذف نشرة','family.removeComment':'حذف تعليق','trainer.campaign':'تحديث طلب تقديم','trainer.apply':'طلب ترشيح','trainer.appointment':'تحديد موعد لجنة','trainer.evaluate':'تقييم اللجنة','trainer.decide':'قرار طلب ترشيح','trainer.session':'تحديث جلسة','trainer.rate':'تقييم السفير','trainer.board':'تعديل لوحة المنافسة','voice.survey':'تعديل جولة استبيان','voice.publish':'تغيير نشر النتائج','voice.plan':'تحديث خطة تحسين','voice.planDelete':'حذف خطوة تحسين'} as Record<string,string>)[k]||k}
