import {useState} from 'react';
import {Plus,Copy,ClipboardList,Pencil,ArrowUpLeft,Play,Square,Trash2,Save,Link2,ShieldCheck} from 'lucide-react';
import {api,useLocale,Empty,Panel,Badge,Field,Modal,task,toast,formatDate} from './shared';
import {PageHeading} from './dashboard';
import type {Survey,Category} from './types';
const uid=()=>crypto.randomUUID();
function cloneSurvey(s?:Survey):Survey{return{id:'',title:{ar:'',en:''},description:{ar:'نستمع لتجربتك ونحوّل آراءك إلى خطوات تحسين.',en:'We listen to your experience and turn feedback into action.'},definition:s?structuredClone(s.definition):{categories:[]},status:'draft',expected_count:0,excellent:50,good:10,opens_at:null,closes_at:null,response_count:0,created_at:''}}
const riyadhTime=(iso:string|null)=>iso?new Date(new Date(iso).getTime()+3*3600000).toISOString().slice(0,16):'';
const fromRiyadh=(value:string)=>value?new Date(value+'+03:00').toISOString():null;

export function SurveysPage({surveys,canEdit,canPublish,onRefresh,onSelect}:{surveys:Survey[];canEdit:boolean;canPublish:boolean;onRefresh:()=>void;onSelect:(s:string)=>void}){
 const{lang,tx,local}=useLocale();
 const [editing,setEditing]=useState<Survey|null>(null),[transition,setTransition]=useState<{id:string;status:'published'|'closed'}|null>(null);
 const [busy,setBusy]=useState(false),[filter,setFilter]=useState('all'),[query,setQuery]=useState(''),[link,setLink]=useState('');
 const list=surveys.filter(s=>(filter==='all'||s.status===filter)&&local(s.title).toLowerCase().includes(query.trim().toLowerCase()));
 const showLink=(id:string)=>task(async()=>{
  const value=await api<{path:string}>('/surveys/'+id+'/link','POST',{});
  setLink(new URL(value.path,location.origin).href);
 },lang);
 return <>
  <PageHeading title={tx('الاستبيانات','Surveys')} sub={tx('جهّز الاستبيان وانشر رابط مشاركة واحداً لجميع الموظفين.','Prepare a survey and share one link with all employees.')}>
   {canEdit&&<>
    <button className="btn ghost" disabled={busy} onClick={()=>{setBusy(true);void task(async()=>{
     const demo=await api<{id:string}>('/surveys/demo','POST',{});onRefresh();onSelect(demo.id);
    },lang).finally(()=>setBusy(false))}}>{tx('استبيان تجريبي · 10 مشاركات','Demo survey · 10 responses')}</button>
    <button className="btn primary" onClick={()=>setEditing(cloneSurvey(surveys[0]))}><Plus size={18}/>{tx('إنشاء استبيان','Create survey')}</button>
   </>}
  </PageHeading>
  <div className="filter-bar">
   <div className="segmented">{[['all',tx('الكل','All')],['draft',tx('مسودات','Drafts')],['published',tx('مفتوحة','Open')],['closed',tx('مغلقة','Closed')]].map(([id,label])=><button className={filter===id?'active':''} onClick={()=>setFilter(id)} key={id}>{label}</button>)}</div>
   <input value={query} onChange={e=>setQuery(e.target.value)} placeholder={tx('ابحث عن استبيان…','Find a survey…')}/>
  </div>
  {list.length?<div className="survey-grid">{list.map(s=><article className="survey-card" key={s.id}>
   <div className="survey-card-top"><span className="round-icon"><ClipboardList size={24}/></span><Badge status={s.status}/></div>
   <h2>{local(s.title)}</h2><p>{local(s.description)}</p>
   <div className="survey-card-stats">
    <div><strong>{s.definition.categories.length}</strong><small>{tx('محاور','Drivers')}</small></div>
    <div><strong>{s.definition.categories.reduce((total,c)=>total+c.questions.length,0)}</strong><small>{tx('عبارة','Items')}</small></div>
    <div><strong>{s.response_count}</strong><small>{tx('مشاركة','Responses')}</small></div>
   </div>
   <div className="cycle-date"><small>{formatDate(s.created_at,lang)}</small></div>
   <div className="survey-card-actions">
    <button className="btn ghost small" onClick={()=>onSelect(s.id)}>{tx('النتائج','Results')}<ArrowUpLeft size={15}/></button>
    <div className="button-row">
     {canPublish&&s.id!=='pulse-demo-v1'&&s.status==='closed'&&s.response_count>0&&<button className="btn ghost small" onClick={()=>void task(async()=>{
      await api('/surveys/'+s.id+'/visibility','PATCH',{visible:!s.employee_visible});onRefresh();
     },lang)}>{s.employee_visible?tx('إخفاء النتيجة عن الموظفين','Hide score from employees'):tx('نشر النتيجة للموظفين','Publish score to employees')}</button>}
     {canEdit&&<>
      <button className="icon-btn" aria-label={tx('نسخ الاستبيان','Duplicate survey')} onClick={()=>setEditing(cloneSurvey(s))}><Copy size={16}/></button>
      {s.status==='draft'&&<>
       <button className="icon-btn" aria-label={tx('تعديل الاستبيان','Edit survey')} onClick={()=>setEditing(s)}><Pencil size={16}/></button>
       <button className="btn primary small" onClick={()=>setTransition({id:s.id,status:'published'})}><Play size={15}/>{tx('نشر','Publish')}</button>
      </>}
      {s.status==='published'&&<>
       <button className="btn primary small" onClick={()=>void showLink(s.id)}><Link2 size={16}/>{tx('رابط المشاركة','Participation link')}</button>
       <button className="btn soft small" onClick={()=>setTransition({id:s.id,status:'closed'})}><Square size={15}/>{tx('إغلاق الاستبيان','Close survey')}</button>
      </>}
     </>}
    </div>
   </div>
  </article>)}</div>:<Panel><Empty title={tx('لا توجد استبيانات','No surveys found')}/></Panel>}
  {editing&&<SurveyBuilder value={editing} onClose={()=>setEditing(null)} onSaved={()=>{setEditing(null);onRefresh()}}/>}
  {transition&&<Modal title={transition.status==='published'?tx('نشر الاستبيان','Publish survey'):tx('إغلاق الاستبيان','Close survey')} onClose={()=>setTransition(null)}>
   <p>{transition.status==='published'?tx('بعد النشر تقدر تنسخ رابط المشاركة الموحد من بطاقة الاستبيان.','After publishing, copy the shared participation link from the survey card.'):tx('يتوقف استقبال المشاركات وتظهر النتائج في الرئيسية.','Responses close and the results appear on the overview.')}</p>
   <button className="btn primary" disabled={busy} onClick={()=>{setBusy(true);void task(async()=>{
    await api('/surveys/'+transition.id+'/status','POST',{status:transition.status});
    const {id,status}=transition;setTransition(null);onRefresh();
    if(status==='published')await showLink(id);
   },lang).finally(()=>setBusy(false))}}>{tx('تأكيد','Confirm')}</button>
  </Modal>}
  {link&&<Modal title={tx('رابط المشاركة لجميع الموظفين','Participation link for all employees')} onClose={()=>setLink('')}>
   <Field label={tx('رابط الاستبيان','Survey link')}><input dir="ltr" readOnly value={link} onFocus={e=>e.target.select()}/></Field>
   {['localhost','127.0.0.1','[::1]'].includes(location.hostname)&&<p className="muted">{tx('هذا عنوان جهازك. انسخ الرابط من عنوان المنصة المتاح للموظفين عند تشغيلها على خادم الشبكة.','This is your local address. Share the link from the address employees can reach.')}</p>}
   <div className="button-row">
    <button className="btn primary" onClick={()=>void task(async()=>{await navigator.clipboard.writeText(link);toast(tx('تم نسخ الرابط','Link copied'))},lang)}><Copy size={16}/>{tx('نسخ رابط المشاركة','Copy participation link')}</button>
    <a className="btn ghost" href={link} target="_blank" rel="noreferrer">{tx('فتح الاستبيان','Open survey')}</a>
   </div>
  </Modal>}
 </>;
}
function SurveyBuilder({value,onClose,onSaved}:{value:Survey;onClose:()=>void;onSaved:()=>void}){const{lang,tx,local}=useLocale();const[v,setV]=useState(value);const[busy,setBusy]=useState(false);const cats=v.definition.categories;function updateCat(index:number,c:Category){setV(v=>({...v,definition:{categories:v.definition.categories.map((x,i)=>i===index?c:x)}}))}return <Modal title={v.id?tx('تعديل مسودة الاستبيان','Edit survey draft'):tx('إنشاء استبيان استبيان','Create survey')} sub={tx('راجع النص العربي والإنجليزي قبل النشر','Review Arabic and English wording before publication')} onClose={onClose} wide><form onInvalid={e=>{const details=(e.target as HTMLElement).closest('details');if(details)details.open=true}} onSubmit={e=>{e.preventDefault();setBusy(true);task(async()=>{await api(v.id?'/surveys/'+v.id:'/surveys',v.id?'PUT':'POST',v);toast(tx('تم حفظ المسودة','Draft saved'));onSaved()},lang).finally(()=>setBusy(false))}}><div className="builder-grid"><section><div className="two-columns"><Field label={tx('عنوان الاستبيان بالعربية','Cycle title in Arabic')}><input required value={v.title.ar} maxLength={180} onChange={e=>setV({...v,title:{...v.title,ar:e.target.value}})}/></Field><Field label={tx('عنوان الاستبيان بالإنجليزية','Cycle title in English')}><input dir="ltr" required value={v.title.en} maxLength={180} onChange={e=>setV({...v,title:{...v.title,en:e.target.value}})}/></Field></div><div className="two-columns"><Field label={tx('وصف عربي','Arabic description')}><textarea required rows={2} value={v.description.ar} onChange={e=>setV({...v,description:{...v.description,ar:e.target.value}})}/></Field><Field label={tx('وصف إنجليزي','English description')}><textarea dir="ltr" required rows={2} value={v.description.en} onChange={e=>setV({...v,description:{...v.description,en:e.target.value}})}/></Field></div><div className="section-heading"><div><h3>{tx('بنك الأسئلة','Question bank')}</h3><p>{cats.length} {tx('محاور','drivers')} · {cats.reduce((n,c)=>n+c.questions.length,0)} {tx('عبارة','items')}</p></div><button className="btn ghost small" type="button" disabled={cats.length>=20} onClick={()=>setV({...v,definition:{categories:[...cats,{id:uid(),name:{ar:'محور جديد',en:'New driver'},track:'all',questions:[{id:uid(),text:{ar:'',en:''},required:true}]}]}})}><Plus size={15}/>{tx('إضافة محور','Add driver')}</button></div>{cats.map((c,index)=><details className="category-editor" key={c.id} open={cats.length===1?true:undefined}><summary><span className="category-index">{String(index+1).padStart(2,'0')}</span><strong>{local(c.name)||tx('محور جديد','New driver')}</strong><small>{c.questions.length} {tx('عبارات','items')}</small></summary><div className="category-content"><div className="two-columns"><Field label={tx('اسم المحور عربي','Driver name — Arabic')}><input required value={c.name.ar} onChange={e=>updateCat(index,{...c,name:{...c.name,ar:e.target.value}})}/></Field><Field label={tx('اسم المحور إنجليزي','Driver name — English')}><input dir="ltr" required value={c.name.en} onChange={e=>updateCat(index,{...c,name:{...c.name,en:e.target.value}})}/></Field></div>{c.questions.map((q,qi)=><div className="question-editor" key={q.id}><div className="question-editor-top"><strong>{tx('العبارة','Item')} {qi+1}</strong><label className="check-label small"><input type="checkbox" checked={q.required} onChange={e=>updateCat(index,{...c,questions:c.questions.map(x=>x.id===q.id?{...x,required:e.target.checked}:x)})}/>{tx('إلزامي','Required')}</label><button className="icon-btn danger" type="button" disabled={c.questions.length===1} aria-label={tx('حذف العبارة','Remove item')} onClick={()=>updateCat(index,{...c,questions:c.questions.filter(x=>x.id!==q.id)})}><Trash2 size={15}/></button></div><div className="two-columns"><Field label={tx('النص العربي','Arabic text')}><textarea required rows={2} value={q.text.ar} onChange={e=>updateCat(index,{...c,questions:c.questions.map(x=>x.id===q.id?{...x,text:{...x.text,ar:e.target.value}}:x)})}/></Field><Field label={tx('النص الإنجليزي','English text')}><textarea dir="ltr" required rows={2} value={q.text.en} onChange={e=>updateCat(index,{...c,questions:c.questions.map(x=>x.id===q.id?{...x,text:{...x.text,en:e.target.value}}:x)})}/></Field></div></div>)}<div className="button-row"><button className="btn ghost small" type="button" disabled={c.questions.length>=12} onClick={()=>updateCat(index,{...c,questions:[...c.questions,{id:uid(),text:{ar:'',en:''},required:true}]})}><Plus size={14}/>{tx('إضافة عبارة','Add item')}</button><button className="btn danger-ghost small" type="button" disabled={cats.length===1} onClick={()=>setV({...v,definition:{categories:cats.filter(x=>x.id!==c.id)}})}><Trash2 size={14}/>{tx('حذف المحور','Remove driver')}</button></div></div></details>)}</section><aside className="builder-settings"><h3>{tx('إعدادات الاستبيان','Cycle settings')}</h3><Field label={tx('عدد الموظفين المستهدفين','Target audience count')}><input type="number" min={0} max={10000000} value={v.expected_count} onChange={e=>setV({...v,expected_count:Number(e.target.value)})}/></Field><Field label={tx('فتح المشاركة — توقيت الرياض','Opens — Riyadh time')}><input type="datetime-local" value={riyadhTime(v.opens_at)} onChange={e=>setV({...v,opens_at:fromRiyadh(e.target.value)})}/></Field><Field label={tx('نهاية المشاركة — توقيت الرياض','Closes — Riyadh time')}><input type="datetime-local" value={riyadhTime(v.closes_at)} onChange={e=>setV({...v,closes_at:fromRiyadh(e.target.value)})}/></Field><hr/><h4>{tx('حدود التصنيف الداخلية','Internal interpretation bands')}</h4><p className="small muted">{tx('إعدادات داخلية للمقارنة وليست معيارًا عالميًا معتمدًا.','Internal interpretation settings, not validated universal benchmarks.')}</p><Field label={tx('بداية التصنيف الجيد','Good band starts at')}><input type="number" min={-100} max={99} value={v.good} onChange={e=>setV({...v,good:Number(e.target.value)})}/></Field><Field label={tx('حد الإشارة الإيجابية القوية','Strong positive signal above')}><input type="number" min={v.good+1} max={100} value={v.excellent} onChange={e=>setV({...v,excellent:Number(e.target.value)})}/></Field><div className="privacy-note compact"><ShieldCheck size={18}/><p>{tx('الحفظ ينشئ مسودة. النشر خطوة مستقلة تثبت إعدادات الاستبيان.','Saving creates a draft. Publishing is a separate step that locks cycle settings.')}</p></div></aside></div><div className="modal-sticky-actions"><button className="btn primary" disabled={busy||cats.length===0}><Save size={16}/>{tx(busy?'جاري الحفظ…':'حفظ المسودة',busy?'Saving…':'Save draft')}</button><button type="button" className="btn ghost" onClick={onClose}>{tx('إلغاء','Cancel')}</button></div></form></Modal>}
