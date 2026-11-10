import {BarChart3,CheckCircle2,Clock3,Plus,Trash2} from 'lucide-react';
import {useEffect,useState} from 'react';
import {api,dateTime,Empty,Field,Form,Header,Modal,msg,perform} from './ui';
import './assessments.css';

export const assessmentPath='/knowledge/assessment';
type Kind='survey'|'quiz';
type Question={kind:'rating'|'nps'|'choice'|'truefalse';text:string;scale?:number;options?:string[];answer?:number};
type QuestionSummary={count:number;median:number|null;answers:{value:number;count:number;share:number}[]};
type SurveyAnalytics={submitted:number;nps?:{count:number;promoters:number;passives:number;detractors:number;score:number|null};questions:QuestionSummary[]};
type Result={id:string;name:string;score:number;note?:string;at:string;choices?:number[]};
type Item={id:string;kind:Kind;title:string;session:string;status:string;minutes:number;questions?:Question[];results?:Result[];analytics?:SurveyAnalytics;eligible?:boolean;done?:boolean;score?:number;started?:boolean;endsAt?:string};
type Session={id:string;title:string;status:string};
const name={survey:'استبيان',quiz:'اختبار'};
const states:Record<string,string>={draft:'مسودة',open:'يستقبل المشاركات',closed:'منتهٍ'};
const href=(id:string)=>assessmentPath+'?id='+encodeURIComponent(id);
const newQuestion=(kind:Kind):Question=>kind==='survey'?{kind:'rating',text:''}:{kind:'choice',text:'',options:['',''],answer:-1};
const options=(q:Question)=>q.kind==='truefalse'?['صح','غلط']:(q.options||[]);

const recommendation='ما مدى احتمالية أن توصي بهذه الجلسة لزميل في العمل؟';
const maxRating=(question:Question)=>question.kind==='nps'||question.scale===10?10:5;
const isRating=(q:Question)=>q.kind==='rating'||q.kind==='nps';

export function RatingCard({name,label,value,min=1,max=5,anchors}:{name:string;label:string;value?:number;min?:number;max?:number;anchors?:[string,string]}){
 return <fieldset className="rating-card"><legend>{label}</legend>
  <div className={'rating-options'+(max>5?' rating-options-wide':'')} dir="ltr">
   {Array.from({length:max-min+1},(_,i)=>i+min).map(n=><label key={n}><input type="radio" name={name} value={n} defaultChecked={value===n} required/><span>{n}</span></label>)}
  </div>
  {anchors&&<div className="rating-anchors"><span>{min} · {anchors[0]}</span><span>{max} · {anchors[1]}</span></div>}
 </fieldset>;
}

export function AssessmentManager({kind,sessions,refreshParent}:{kind:Kind;sessions:Session[];refreshParent:()=>Promise<void>}){
 const [data,setData]=useState<{items:Item[];template:Question[]}>(),[error,setError]=useState(''),[dialog,setDialog]=useState<{mode:string;item?:Item}>();
 const refresh=async()=>setData(await api<{items:Item[];template:Question[]}>('/api/trainer/assessments'));
 const items=data?.items.filter(item=>item.kind===kind);
 useEffect(()=>{void refresh().catch(e=>setError(msg(e)))},[kind]);
 const update=async(action:string,body:unknown)=>{await perform(action,body);await Promise.all([refresh(),refreshParent()]);};
 const close=()=>setDialog(undefined);
 const item=dialog?.item;
 return <section className="card section" data-guide={kind==='survey'?'knowledge-surveys':'knowledge-quizzes'}>
  <Header title={kind==='survey'?'استبيانات الجلسات':'الاختبارات البعدية'} subtitle="جهّز الأداة، شارك رابطها عند إتاحتها، واربطها بجلسة وقت ما تحتاج." action={<button className="btn primary" disabled={!data} onClick={()=>setDialog({mode:'edit'})}>إنشاء {name[kind]}</button>}/>
  {error&&<p className="error">{error}</p>}
  <div className="assessment-grid">{items?.map(a=><article className="item assessment-card" key={a.id}>
   <div className="row between"><h3>{a.title}</h3><span>{states[a.status]}</span></div>
   <p>{sessions.find(s=>s.id===a.session)?.title||'أداة مستقلة'} · {a.questions?.length||0} أسئلة · {a.results?.length||0} مشاركات</p>
   <div className="actions">
    <button className="btn outline" onClick={()=>setDialog({mode:'edit',item:a})}>{a.status==='draft'?'تعديل':'العنوان وربط الجلسة'}</button>
    {a.status==='draft'&&<button className="btn primary" onClick={()=>setDialog({mode:'open',item:a})}>إتاحة {name[kind]}</button>}
    {a.status==='open'&&<button className="btn primary" onClick={()=>setDialog({mode:'link',item:a})}>رابط المشاركة</button>}
    {a.status==='open'&&<button className="btn outline" onClick={()=>setDialog({mode:'closed',item:a})}>إنهاء {name[kind]}</button>}
    <button className="btn outline" onClick={()=>setDialog({mode:'results',item:a})}><BarChart3 size={16}/>{kind==='survey'?'لوحة النتائج':'النتائج'}</button>
   </div>
  </article>)}</div>
  {!items?.length&&<Empty>{items?'لا توجد أدوات تقييم هنا بعد.':'جاري التحميل…'}</Empty>}
  {dialog?.mode==='edit'&&<Editor kind={kind} item={item} sessions={sessions} template={data?.template||[]} close={close} save={body=>update('trainer.assessmentSave',body)}/>}
  {item&&dialog?.mode==='link'&&<Share item={item} close={close}/>}
  {item&&dialog?.mode==='results'&&<Results item={item} close={close}/>}
  {item&&(dialog?.mode==='open'||dialog?.mode==='closed')&&<Modal title={(dialog.mode==='open'?'إتاحة ':'إنهاء ')+name[kind]} onClose={close}>
   <Form label={dialog.mode==='open'?'إتاحة المشاركة':'إيقاف استقبال الردود'} onSave={async()=>{await update('trainer.assessmentStatus',{id:item.id,status:dialog.mode});if(dialog.mode==='open')setDialog({mode:'link',item});else close()}}>
    <p>{dialog.mode==='open'?'هيبقى الرابط جاهز للمشاركة مع الموظفين.':'تُحفظ النتائج ويمكن الرجوع لها بعد الإنهاء.'}</p>
   </Form></Modal>}
 </section>;
}
function Share({item,close}:{item:Item;close:()=>void}){
 const url=new URL(href(item.id),location.origin).href;
 const [status,setStatus]=useState('');
 return <Modal title="رابط المشاركة" onClose={close}>
  <p>يقدر الموظف يفتح الرابط ويسجل دخوله للمشاركة.</p>
  <Field label="الرابط"><input type="url" value={url} readOnly dir="ltr" onFocus={e=>e.target.select()}/></Field>
  <div className="actions"><button className="btn primary" onClick={async()=>{try{await navigator.clipboard.writeText(url);setStatus('تم نسخ الرابط')}catch{setStatus('حدّد الرابط في الحقل وانسخه')}}}>نسخ الرابط</button><a className="btn outline" href={url} target="_blank" rel="noreferrer">فتح الرابط</a></div>
  <p role="status">{status}</p>
 </Modal>;
}
function Results({item,close}:{item:Item;close:()=>void}){
 const rows=item.results||[];
 return <Modal title={(item.kind==='survey'?'لوحة نتائج الاستبيان':'نتائج الاختبار')+' · '+item.title} onClose={close}>
  {item.kind==='survey'&&item.analytics&&<SurveyDashboard item={item} analytics={item.analytics}/>}
  {rows.length?<>
   {item.kind==='quiz'&&<p>{rows.length} مشاركات · المتوسط {(rows.reduce((sum,row)=>sum+row.score,0)/rows.length).toFixed(1)} / 100</p>}
   <div className="table-wrap assessment-respondents"><table><thead><tr><th>المشارك</th><th>{item.kind==='quiz'?'الدرجة':'الإجابات'}</th><th>الملاحظة</th></tr></thead><tbody>{rows.map(r=><tr key={r.id}><td>{r.name}</td><td>{item.kind==='quiz'?r.score+' / 100':<details><summary>عرض الإجابات</summary><ol>{item.questions?.map((q,i)=><li key={i}>{q.text} <bdi>{r.choices?.[i]??'—'}</bdi></li>)}</ol></details>}</td><td>{r.note||'—'}</td></tr>)}</tbody></table></div>
  </>:<Empty>لا توجد نتائج حتى الآن.</Empty>}
 </Modal>;
}
function SurveyDashboard({item,analytics}:{item:Item;analytics:SurveyAnalytics}){
 const nps=analytics.nps;
 const groups=nps?[
  {name:'المروّجون',range:'9–10',count:nps.promoters,tone:'promoter'},
  {name:'المحايدون',range:'7–8',count:nps.passives,tone:'passive'},
  {name:'المنتقدون',range:'0–6',count:nps.detractors,tone:'detractor'},
 ]:[];
 const number=(value:number)=>value.toLocaleString('ar-SA',{maximumFractionDigits:1});
 const percentage=(count:number)=>nps?.count?number(100*count/nps.count)+'٪':'—';
 return <div className="survey-dashboard">
  <div className="survey-metrics">
   <article><span>الردود المكتملة</span><strong>{number(analytics.submitted)}</strong></article>
   {nps&&<article><span>مؤشر التوصية · NPS</span><strong>{nps.score==null?'—':<bdi>{nps.score>0?'+':''}{number(nps.score)}</bdi>}</strong><small>{number(nps.count)} إجابات على سؤال التوصية</small></article>}
  </div>
  {nps&&<section className="survey-result-panel">
   <h3>توزيع التوصية</h3>
   <div className="nps-groups">{groups.map(g=><div className={'nps-group '+g.tone} key={g.name}><b>{g.name} <bdi>{g.range}</bdi></b><strong>{number(g.count)}</strong><span>{percentage(g.count)}</span></div>)}</div>
   {nps.count>0&&<div className="nps-bar" aria-hidden="true">{groups.map(g=><i className={g.tone} key={g.name} style={{flex:g.count}}/>)}</div>}
   <p className="muted">نسبة المروّجين ناقص نسبة المنتقدين. يشمل المقام كل إجابات سؤال التوصية، بما فيها إجابات المحايدين.</p>
  </section>}
  {!nps&&<p className="muted">يظهر NPS في الاستبيانات التي تحتوي على سؤال توصية من 0 إلى 10.</p>}
  <section className="survey-question-results">
   <h3>توزيع الإجابات لكل سؤال</h3>
   {analytics.questions.map((itemSummary,i)=>{
    const q=item.questions?.[i];if(!q)return null;
    return <article className="survey-result-panel" key={i}>
     <header><h4>{i+1}. {q.text}</h4><span>{q.kind==='nps'?'0–10':'1–'+maxRating(q)} · {number(itemSummary.count)} إجابة{itemSummary.median!=null?' · الوسيط '+number(itemSummary.median):''}</span></header>
     {itemSummary.count>0?<ol className="survey-distribution">{itemSummary.answers.map(answer=><li key={answer.value}>
      <b>{answer.value}</b><div className="survey-bar" aria-hidden="true"><i style={{width:answer.share+'%'}}/></div><span>{number(answer.count)} · {number(answer.share)}٪</span>
     </li>)}</ol>:<p className="muted">لا توجد إجابات لهذا السؤال بعد.</p>}
    </article>;
   })}
  </section>
  <details className="survey-method">
   <summary>طريقة قراءة النتائج والمراجع</summary>
   <p>توزيع كل سؤال يعرض العدد والنسبة من إجابات هذا السؤال. الوسيط هو منتصف الإجابات بعد ترتيبها. مؤشر التوصية يحسب من سؤال التوصية وحده. الرغبة في التطبيق تعكس رأي المشارك بعد الجلسة، ويقيس الاختبار البعدي المعرفة التي غطتها أسئلته.</p>
   <ul><li><a href="https://www.netpromotersystem.com/about/measuring-your-net-promoter-score/" target="_blank" rel="noreferrer">Bain · حساب NPS</a></li>
   <li><a href="https://www.cdc.gov/training-development/php/about/evaluate-training-measuring-effectiveness.html" target="_blank" rel="noreferrer">CDC · قياس فاعلية التدريب</a></li></ul>
  </details>
 </div>;
}
function Editor({kind,item,sessions,template,close,save}:{kind:Kind;item?:Item;sessions:Session[];template:Question[];close:()=>void;save:(body:unknown)=>Promise<void>}){
 const [questions,setQuestions]=useState<Question[]>(item?.questions||(kind==='survey'&&template.length?template:[newQuestion(kind)]));
 const [timed,setTimed]=useState((item?.minutes??0)>0);
 const locked=!!item&&item.status!=='draft';
 const patch=(i:number,change:Partial<Question>)=>setQuestions(qs=>qs.map((q,j)=>j===i?{...q,...change}:q));
 const removeOption=(q:Question,i:number,index:number)=>{
  const answer=q.answer??-1;
  patch(i,{options:options(q).filter((_,j)=>j!==index),answer:answer===index?-1:answer>index?answer-1:answer});
 };
 return <Modal title={(item?'تعديل ':'إنشاء ')+name[kind]} onClose={close}>
  <div className="assessment-editor">
   <Form after={close} onSave={f=>save({id:item?.id||'',kind,title:f.get('title'),session:f.get('session'),minutes:locked?item?.minutes:timed?Number(f.get('minutes')):0,questions})}>
    <section className="assessment-settings">
     <h3>الإعدادات</h3>
     <div className="assessment-setup-fields">
      <Field label="العنوان"><input name="title" required maxLength={160} defaultValue={item?.title}/></Field>
      <Field label="الجلسة المرتبطة (اختياري)"><select name="session" defaultValue={item?.session||''} aria-readonly={locked&&!!item?.session}>
       <option value="" disabled={locked&&!!item?.session}>بدون جلسة</option>{sessions.map(s=><option value={s.id} key={s.id} disabled={s.id!==item?.session&&(locked&&!!item?.session||s.status==='closed'&&item?.status!=='closed')}>{s.title}</option>)}
      </select></Field>
     </div>
     {kind==='quiz'&&<div className="assessment-timer">
      <div><Clock3 size={19} aria-hidden="true"/><div><strong>وقت الاختبار</strong><p>يبدأ العداد بعد الضغط على «ابدأ الاختبار».</p></div></div>
      <label className="check"><input type="checkbox" checked={timed} disabled={locked} onChange={e=>setTimed(e.target.checked)}/> تحديد وقت</label>
      {timed&&<Field label="المدة بالدقائق"><input type="number" name="minutes" min={1} max={240} required disabled={locked} defaultValue={item?.minutes||15}/></Field>}
     </div>}
    </section>
    {locked?<p className="muted">بعد الإتاحة تظل الأسئلة والمدة ثابتة لحفظ اتساق النتائج.</p>:<>
     <div className="assessment-questions-heading"><h3>الأسئلة</h3><span>{questions.length} سؤال</span></div>
     {questions.map((q,i)=><article className="assessment-question" key={i}>
      <header><span className="assessment-question-number">{i+1}</span><strong>السؤال {i+1}</strong>
       <button type="button" className="icon" aria-label={'حذف السؤال '+(i+1)} disabled={questions.length<=1} onClick={()=>setQuestions(qs=>qs.filter((_,j)=>i!==j))}><Trash2 size={17}/></button>
      </header>
      <div className="assessment-question-fields">
       <Field label="نص السؤال"><textarea required rows={2} maxLength={300} value={q.text} onChange={e=>patch(i,{text:e.target.value})}/></Field>
       {kind==='quiz'?<Field label="نوع السؤال"><select value={q.kind} onChange={e=>patch(i,{kind:e.target.value as Question['kind'],options:['',''],answer:-1})}><option value="choice">اختيار من متعدد</option><option value="truefalse">صح أو غلط</option></select></Field>
       :<Field label="مقياس الإجابة"><select value={q.kind==='nps'?'nps':String(maxRating(q))} onChange={e=>{
        const value=e.target.value;
        patch(i,value==='nps'?{kind:'nps',scale:undefined,text:recommendation}:{kind:'rating',scale:Number(value)});
       }}><option value="5">من 1 إلى 5</option><option value="10">من 1 إلى 10</option><option value="nps" disabled={q.kind!=='nps'&&questions.some(question=>question.kind==='nps')}>توصية NPS · من 0 إلى 10</option></select></Field>}
      </div>
      {kind==='survey'&&q.kind==='nps'&&<p className="muted">خصص هذا السؤال لقياس احتمالية التوصية بالجلسة لزميل. يُحسب المؤشر من إجاباته فقط.</p>}
      {kind==='quiz'&&<>
       {q.kind==='choice'&&<div className="assessment-choice-list">
        <h4>خيارات الإجابة</h4>
        {options(q).map((o,j)=><div className="assessment-choice-row" key={j}><span>{j+1}</span><input aria-label={'الخيار '+(j+1)+' للسؤال '+(i+1)} required maxLength={180} value={o} placeholder={'اكتب الخيار '+(j+1)} onChange={e=>patch(i,{options:options(q).map((v,k)=>j===k?e.target.value:v)})}/><button type="button" className="icon" aria-label={'حذف الخيار '+(j+1)} disabled={options(q).length<=2} onClick={()=>removeOption(q,i,j)}>×</button></div>)}
        <button type="button" className="btn outline" disabled={options(q).length>=6} onClick={()=>patch(i,{options:[...options(q),'']})}><Plus size={16}/> إضافة خيار</button>
       </div>}
       <fieldset className="assessment-key"><legend><CheckCircle2 size={16} aria-hidden="true"/> مفتاح الإجابة</legend>
        <p>حدد إجابة صحيحة واحدة للتصحيح التلقائي.</p>
        <div>{options(q).map((o,j)=><label className="check" key={j}><input type="radio" name={'answer-key-'+i} required checked={q.answer===j} onChange={()=>patch(i,{answer:j})}/>{o||'الخيار '+(j+1)}</label>)}</div>
       </fieldset>
      </>}
     </article>)}
     <button type="button" className="btn outline" disabled={questions.length>=40} onClick={()=>setQuestions(qs=>[...qs,newQuestion(kind)])}><Plus size={16}/> إضافة سؤال</button>
    </>}
   </Form>
  </div>
 </Modal>;
}

export function AssessmentPage(){
 const id=new URLSearchParams(location.search).get('id')||'';
 const [item,setItem]=useState<Item>(),[error,setError]=useState(''),[now,setNow]=useState(Date.now());
 const refresh=async()=>{
  try{setItem(await api<Item>('/api/trainer/assessment?id='+encodeURIComponent(id)));setError('')}
  catch(e){if((e as {status?:number}).status===401)location.replace('/login?next='+encodeURIComponent(href(id)));else setError(msg(e))}
 };
 useEffect(()=>{void refresh()},[id]);
 useEffect(()=>{if(!item?.endsAt)return;const timer=window.setInterval(()=>setNow(Date.now()),1000);return()=>window.clearInterval(timer)},[item?.endsAt]);
 const remaining=item?.endsAt?Math.max(0,Math.ceil((Date.parse(item.endsAt)-now)/1000)):null;
 const save=async(action:string,body:unknown)=>{await perform(action,{id,...body as object});await refresh()};
 return <main className="shell assessment-page"><section className="card section">
  <Header title={item?.title||'تقييم سفراء المعرفة'} action={<a className="btn outline" href="/knowledge">سفراء المعرفة</a>}/>
  {error?<p className="error">{error}</p>:!item?<Empty>جاري التحميل…</Empty>:item.done?<p className="notice" role="status">تم تسجيل مشاركتك.{item.kind==='quiz'&&' نتيجتك '+item.score+' / 100'}</p>:!item.eligible||item.status!=='open'?<Empty>المشاركة غير متاحة الآن لهذا الحساب.</Empty>:item.kind==='quiz'&&!item.started?<Form label="ابدأ الاختبار" onSave={()=>save('trainer.assessmentStart',{})}><p>{item.minutes?'المدة '+item.minutes+' دقيقة. يبدأ الوقت عند الضغط على زر البدء.':'الاختبار بدون وقت محدد.'}</p></Form>:remaining===0?<p className="error">انتهى وقت الاختبار.</p>:<>
   {remaining!==null&&<p className="notice assessment-clock">الوقت المتبقي <bdi>{Math.floor(remaining/60)}:{String(remaining%60).padStart(2,'0')}</bdi></p>}
   <Form label="إرسال المشاركة" onSave={f=>save('trainer.assessmentSubmit',{choices:item.questions?.map((_,i)=>{const value=f.get('q'+i);return value===null?null:Number(value)}),note:f.get('note')||''})}>
    {item.questions?.map((q,i)=>isRating(q)?<RatingCard key={i} name={'q'+i} label={(i+1)+'. '+q.text} min={q.kind==='nps'?0:1} max={maxRating(q)} anchors={q.kind==='nps'?['غير محتمل إطلاقاً','محتمل جداً']:maxRating(q)===5?['لا أوافق بشدة','أوافق بشدة']:['أقل تقييم','أعلى تقييم']}/>:<fieldset className="rating-card" key={i}><legend>{i+1}. {q.text}</legend><div className="assessment-options">{options(q).map((o,j)=><label className="check" key={j}><input required type="radio" name={'q'+i} value={j}/>{o}</label>)}</div></fieldset>)}
    {item.kind==='survey'&&<Field label="ملاحظاتك (اختياري)"><textarea name="note" rows={3} maxLength={1000}/></Field>}
   </Form>
  </>}
 </section></main>;
}
