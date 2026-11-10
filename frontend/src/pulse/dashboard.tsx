import {ArrowUpLeft,ClipboardList,Download,MessageSquare,Plus,Printer,Users} from 'lucide-react';
import {Roadmap} from '../roadmap';
import {PublishedScore} from './score';
import {api,useLocale,useResource,Loading,ErrorBox,Empty,Panel,Badge,task} from './shared';
import type {Survey,Analytics,Driver} from './types';

export function SurveySelect({surveys,selected,setSelected}:{surveys:Survey[];selected:string;setSelected:(s:string)=>void}){const{tx,local}=useLocale();return <select className="cycle-select" aria-label={tx('الاستبيان','Survey')} value={selected} onChange={e=>setSelected(e.target.value)}>{!surveys.length&&<option>{tx('لا توجد استبيانات','No surveys')}</option>}{surveys.map(s=><option value={s.id} key={s.id}>{local(s.title)}</option>)}</select>}
export function PageHeading({title,sub,children}:{title:string;sub:string;children?:React.ReactNode}){return <div className="page-heading"><div><h1>{title}</h1><p>{sub}</p></div><div className="heading-actions">{children}</div></div>}
export function DriverBars({drivers}:{drivers:Driver[]}){const{tx,local}=useLocale();return drivers.length?<div className="driver-bars">{drivers.map((d,i)=><div className="driver-bar" key={d.id}><div><span><b>{String(i+1).padStart(2,'0')}</b>{local(d.name)}</span><strong>{d.average?.toFixed(2)??'—'}<small>/5</small></strong></div><div className="progress-track"><i className={d.average!==null&&d.average<3?'low':''} style={{width:(d.average??0)/5*100+'%'}}/></div></div>)}</div>:<Empty title={tx('النتائج لم تُنشر بعد','Results are not released yet')} body={tx('تظهر المحاور بعد إغلاق الاستبيان ووصول المشاركات.','Drivers appear after the survey closes with responses.')}/>}
function Radar({drivers}:{drivers:Driver[]}){const{local,tx}=useLocale();const ds=drivers.filter(d=>d.average!==null);if(ds.length<3)return <Empty title={tx('يلزم 3 محاور متاحة لعرض الرسم','At least 3 available drivers are needed')}/>;const center=150,r=110;const point=(i:number,f:number)=>{const ang=(i*2*Math.PI/ds.length)-Math.PI/2;return[center+Math.cos(ang)*r*f,center+Math.sin(ang)*r*f]};return <div className="radar-wrap"><svg viewBox="0 0 300 300" role="img" aria-label={tx('رسم مستويات المحاور','Driver score radar')}>{[.2,.4,.6,.8,1].map(f=><polygon key={f} points={ds.map((_,i)=>point(i,f).join(',')).join(' ')} fill="none" stroke="#e8e3eb"/>)}{ds.map((_,i)=>{const[x,y]=point(i,1);const[lx,ly]=point(i,1.18);return <g key={i}><line x1={center} y1={center} x2={x} y2={y} stroke="#e8e3eb"/><text x={lx} y={ly+4} textAnchor="middle" fontSize="11" fill="#7c5973">{i+1}</text></g>})}<polygon points={ds.map((d,i)=>point(i,d.average!/5).join(',')).join(' ')} fill="#a21c7440" stroke="#93115e" strokeWidth="2"/>{ds.map((d,i)=>{const[x,y]=point(i,d.average!/5);return <circle key={i} cx={x} cy={y} r="3.5" fill="#a21c74"/>})}</svg><div className="radar-legend">{ds.map((d,i)=><span key={d.id}><b>{i+1}</b>{local(d.name)}</span>)}</div></div>}

export function PulseRoadmap(){
 const {tx}=useLocale();
 return <Roadmap title={tx('خريطة رحلة نبض درعه','Deraah Pulse roadmap')}
  goalLabel={tx('الهدف','Purpose')} goal={tx('قياس تجربة الموظفين ومعرفة الجوانب التي تستحق الأولوية في التحسين.','Measure the employee experience and identify what needs attention.')}
  valueLabel={tx('القيمة المضافة','Value')} value={tx('نتائج موحدة توضح نقاط القوة والفرص، وتساعد على اختيار إجراءات قابلة للمتابعة والقياس.','A shared view of strengths and opportunities that guides measurable actions.')}
  steps={[
   {title:tx('جهّز الاستبيان','Prepare'),detail:tx('اختر المحاور وعدّل الأسئلة وحدد فترة الاستقبال.','Choose drivers, edit questions and set the response window.')},
   {title:tx('شارك الرابط','Share'),detail:tx('أرسل رابطاً واحداً للكل. يشارك الموظف بدون حساب أو اسم.','Send one link. Employees respond without an account or name.')},
   {title:tx('اقرأ النتائج','Review'),detail:tx('بعد الإغلاق تظهر النتائج العامة، والـeNPS وتقييم المحاور.','After closing, review the overall score, eNPS and drivers.')},
   {title:tx('تابع التحسين','Act'),detail:tx('حدد الإجراء والمسؤول والموعد وراجع تقدمه.','Assign an action, owner and due date, then follow progress.')},
  ]}/>;
}

export function Dashboard({surveys,selected,setSelected,go,canEdit,canResults,canPublish,refresh}:{surveys:Survey[];selected:string;setSelected:(s:string)=>void;go:(s:string)=>void;canEdit:boolean;canResults:boolean;canPublish:boolean;refresh:()=>void}){
 const {tx,local,lang}=useLocale();
 const {value:a,error,reload}=useResource(()=>selected&&canResults?api<Analytics>('/surveys/'+selected+'/analytics'):Promise.resolve(null),[selected,canResults]);
 const v=surveys.find(s=>s.id===selected);
 const stats=[
  {Icon:ClipboardList,label:tx('الاستبيانات','Surveys'),number:surveys.length},
  {Icon:MessageSquare,label:tx('إجمالي المشاركات','Total responses'),number:surveys.reduce((total,s)=>total+s.response_count,0)},
  {Icon:Users,label:tx('تستقبل المشاركات','Collecting responses'),number:surveys.filter(s=>s.status==='published').length},
 ];
 const ready=Boolean(a&&!a.overall.suppressed);
 const score=a?.overall.enps;
 const distribution=[
  {label:tx('مروجون 9–10','Promoters 9–10'),count:a?.overall.promoters,className:'promoter'},
  {label:tx('محايدون 7–8','Passives 7–8'),count:a?.overall.passives,className:'passive'},
  {label:tx('منتقدون 0–6','Detractors 0–6'),count:a?.overall.detractors,className:'detractor'},
 ];
 const percent=(count:number|null|undefined)=>a?.overall.count?Math.round((count??0)/a.overall.count*100):0;
 const exportCSV=()=>task(async()=>{
  const r=await fetch('/pulse/api/surveys/'+selected+'/export');
  if(!r.ok)throw Error('results_suppressed');
  const url=URL.createObjectURL(await r.blob());
  const link=document.createElement('a');link.href=url;link.download='deraah-pulse-results.csv';link.click();
  setTimeout(()=>URL.revokeObjectURL(url),1000);
 },lang);
 return <>
  <section className="dashboard-hero" data-guide="pulse-hero"><div><span className="eyebrow">{tx('نبض المؤسسة','THE PULSE OF OUR PEOPLE')}</span>
   <h1>{tx('كل صوت يفتح','Every voice opens')}<br/><em>{tx('مساحة للتغيير.','a space for change.')}</em></h1>
   <p>{tx('رؤية أوضح لتجربة موظفينا، وخطوات عملية تصنع أثرًا.','Clearer insights into the employee experience, with practical next steps.')}</p>
   {canEdit&&<button className="btn primary" onClick={()=>go('surveys')}><Plus size={17}/>{tx('إدارة الاستبيانات وروابط المشاركة','Manage surveys and participation links')}<ArrowUpLeft size={16}/></button>}
  </div></section>
  <PulseRoadmap/>
  <div className="overview-stats pulse-three-stats">{stats.map(({Icon,label,number},i)=><div className="stat-card" key={label}><div><span>{label}</span><strong>{number}</strong></div><span className={'stat-icon c'+i}><Icon size={22}/></span></div>)}</div>
  {canResults&&<>
   {!surveys.some(s=>s.status==='closed'&&s.response_count>0)&&<PublishedScore legacyOnly/>}
   <PageHeading title={tx('النتائج والتحليلات','Results & insights')} sub={tx('اختر استبياناً لقراءة النتائج العامة ومتابعة عدد المشاركات.','Select a survey to see its overall results and participation.')}>
    <SurveySelect surveys={surveys} selected={selected} setSelected={setSelected}/>
    {ready&&<><button className="btn ghost" onClick={exportCSV}><Download size={16}/>CSV</button><button className="btn ghost" onClick={()=>window.print()}><Printer size={16}/>{tx('طباعة / PDF','Print / PDF')}</button></>}
   </PageHeading>
   {error?<ErrorBox code={error} retry={reload}/>:a===undefined?<Loading/>:!a||!v?<Panel><Empty title={tx('اختر استبياناً','Select a survey')}/></Panel>:<>
    <div className="dashboard-grid">
     <Panel title={tx('مؤشر التوصية بالعمل','Employee Net Promoter Score')} sub={local(v.title)} action={<Badge status={v.status}/>}>
      <div data-guide="pulse-results">
      <div className="score-overview"><div className="enps-ring" style={{'--score':score==null?0:(score+100)/2} as React.CSSProperties}><div><span>eNPS</span><strong>{score==null?'—':(score>0?'+':'')+score}</strong><small>−100 / +100</small></div></div>
      <div className="score-explainer"><p>{a.release==='closed_only'?tx('تظهر النتائج بعد إغلاق الاستبيان.','Results appear after the survey closes.'):ready?tx('نسبة المروجين ناقص نسبة المنتقدين.','Promoter percentage minus detractor percentage.'):tx('لم تصل مشاركات لهذا الاستبيان.','This survey has no responses.')}</p></div></div>
      {canPublish&&v.id!=='pulse-demo-v1'&&v.status==='closed'&&score!=null&&<div className="enps-sharing">
       <button className="btn ghost" role="switch" aria-checked={Boolean(v.employee_visible)} onClick={()=>task(async()=>{await api('/surveys/'+v.id+'/visibility','PATCH',{visible:!v.employee_visible});refresh()},lang)}>
        {v.employee_visible?tx('المؤشر ظاهر للموظفين','Score visible to employees'):tx('المؤشر مخفي عن الموظفين','Score hidden from employees')}
       </button>
      </div>}</div>
     </Panel>
     <Panel title={tx('تقدم المشاركة','Participation progress')}>
      <div className="participation"><div><strong>{a.response_count}</strong><span>{tx('مشاركة مكتملة','Completed responses')}</span></div>
       {a.expected_count>0&&<><span className="participation-percent">{Math.round(a.response_count/a.expected_count*100)}%</span>
       <div className="progress-track"><i style={{width:Math.min(100,a.response_count/a.expected_count*100)+'%'}}/></div></>}
      </div>
     </Panel>
    </div>
    {ready&&<>
     <Panel title={tx('توزيع التوصية','Recommendation distribution')}>
      <div className="distribution">{distribution.filter(x=>x.count).map(x=><div key={x.className} className={x.className} style={{flex:x.count??0}} title={x.label}>{percent(x.count)}%</div>)}</div>
      <div className="distribution-legend">{distribution.map(x=><span key={x.className}><i className={x.className}/>{x.label} · {x.count??0}</span>)}</div>
     </Panel>
     <div className="two-columns">
      <Panel title={tx('تقييم المحاور من 5','Driver scores out of 5')}><DriverBars drivers={a.drivers}/></Panel>
      <Panel title={tx('خريطة تجربة الموظف','Employee experience radar')}><Radar drivers={a.drivers}/></Panel>
     </div>
     <Panel title={tx('نسبة الموافقة الإيجابية','Favorable response rate')} sub={tx('نسبة الدرجات 4 و5 من الإجابات لكل محور.','Scores of 4 or 5 as a share of responses for each driver.')}>
      <div className="table-scroll"><table><thead><tr><th>{tx('المحور','Driver')}</th><th>{tx('المتوسط','Average')}</th><th>{tx('الموافقة','Favorable')}</th><th>{tx('المشاركون','Respondents')}</th></tr></thead>
       <tbody>{a.drivers.map(d=><tr key={d.id}><td>{local(d.name)}</td><td>{d.average?.toFixed(2)??'—'}</td><td>{d.favorable==null?'—':d.favorable+'%'}</td><td>{d.respondents??'—'}</td></tr>)}</tbody>
      </table></div>
     </Panel>
     <Panel title={tx('أبرز الكلمات في التعليقات المعتمدة','Words from approved comments')} sub={tx('وصف للتكرار في التعليقات المعتمدة.','Descriptive frequencies from approved comments.')}>
      {a.themes.length?<div className="theme-chips">{a.themes.map(t=><span key={t.word}>{t.word}<b>{t.count}</b></span>)}</div>:<Empty title={tx('لا توجد تعليقات معتمدة حتى الآن','No approved comments yet')}/>}
     </Panel>
    </>}
   </>}
  </>}
  <p className="pulse-about" data-guide="pulse-history"><a href="/pulse/history">{tx('النتائج السابقة في مدار درعه','Previous Madar Deraah results')}</a></p>
 </>;
}
