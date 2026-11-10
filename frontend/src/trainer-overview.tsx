import {ScheduleCalendar,type CalendarEvent} from './schedule';
import {Roadmap} from './roadmap';
import {ArrowUpLeft,Award,BarChart3,ClipboardList,Plus,UsersRound} from 'lucide-react';
import {Header,dateTime,Empty} from './ui';

export function TrainerOverview({campaigns,applications,sessions,calendar,manage,create,navigate}:{
 campaigns:number;
 applications:{id:string;topic:string;status:string;appointment:string;score:number|null;decision:string;reviewCount:number;reviewComplete?:boolean}[];
 sessions:{feedbackCount?:number}[];
 calendar:CalendarEvent[];
 manage:boolean;
 create?:()=>void;
 navigate:(tab:string)=>void;
}) {
 const accepted=applications.filter(a=>a.status==='accepted').length;
 const metrics=[
  {value:campaigns,label:'فرص تقديم مفتوحة',detail:'مشاركة الخبرة بين الإدارات',Icon:ClipboardList},
  {value:applications.length,label:manage?'طلبات التقديم':'طلباتي',detail:manage?'الطلبات المسجلة في البرنامج':'الطلبات التي قدّمتها',Icon:UsersRound},
  {value:accepted,label:'طلبات مقبولة',detail:'موضوعات قابلة للتحويل إلى جلسات',Icon:Award},
  {value:sessions.reduce((count,s)=>count+(s.feedbackCount||0),0),label:manage?'تقييمات الحضور':'تقييمات جلساتي',detail:'ملاحظات تبني خطة التطوير',Icon:BarChart3},
 ];
 const stages=[
  ['بانتظار تحديد موعد اللجنة',applications.filter(a=>a.status==='submitted'&&!a.appointment).length],
  ['لها موعد لجنة',applications.filter(a=>a.status==='submitted'&&a.appointment).length],
  ['مقبولة',accepted],
  ['غير مقبولة',applications.filter(a=>a.status==='declined').length],
 ] as const;
 return <div className="trainer-overview">
  <Header label="التدريب والتطوير / برنامج نقل المعرفة" title="نظرة عامة" subtitle="معرفة من داخل الشركة. أثر يمتد لكل الإدارات."
   action={create&&<button className="btn primary" onClick={create}><Plus size={17}/> فتح طلب تقديم</button>}/>
  <div className="hero">
   <div>
    <div className="hero-tag"><span/> برنامج سفراء المعرفة</div>
    <h2>خبراتهم تستحق<br/><em>أن تصل للجميع.</em></h2>
    <p>اكتشف الخبراء داخل إداراتك، وشارك المعرفة العملية التي يستفيد منها الزملاء في شغلهم.</p>
    <button onClick={()=>navigate(manage?'applications':'campaigns')}>{manage?'استعرض رحلة المرشحين':'عرض فرص التقديم'} <ArrowUpLeft size={18}/></button>
   </div>
   <div className="hero-seal"><Award size={46}/><b>معرفة • تمكين • أثر</b><span>KNOWLEDGE AMBASSADORS</span></div>
  </div>
  <ScheduleCalendar events={calendar}/>
  <Roadmap title="خريطة رحلة سفراء المعرفة" goalLabel="الهدف" goal="تحويل الخبرة العملية داخل درعه إلى جلسات يتعلم منها الموظفون."
   valueLabel="القيمة المضافة" value="تنتقل الخبرة بين الإدارات، ويقيس فريق التدريب الفائدة بتقييم الحضور والاختبار البعدي."
   steps={[
    {title:'قدّم موضوعك',detail:'افتح الرابط المشترك واختر إدارتك واكتب فكرة الموضوع ونقاط الشرح.'},
    {title:'اعرض الفكرة للجنة',detail:'تراجع اللجنة الموضوع وتقيّمه بالمعايير المعلنة قبل اتخاذ القرار.'},
    {title:'قدّم الجلسة',detail:'يتفق الفريق مع المقبولين على الموعد والمحتوى، ثم تُتاح الجلسة للموظفين.'},
    {title:'قِس الأثر',detail:'تقييم الحضور والاختبار البعدي يوضحان ما استفاد منه الزملاء وما يحتاج تطويراً.'},
   ]}/>
  <div className="stats">{metrics.map(({value,label,detail,Icon},i)=><div className="stat" key={label}>
   <div><small>{label}</small><strong>{value}</strong><p>{detail}</p></div>
   <span className={'stat-icon c'+i}><Icon size={23}/></span>
  </div>)}</div>
  <div>
   {manage?<section className="card"><div className="section"><Header title="جاهزية المرشحين" subtitle="توزيع الطلبات حسب المرحلة الحالية"/></div>
    <div className="bars">{stages.map(([label,count])=><div key={label}>
     <div className="row between"><span>{label}</span><b>{count}</b></div>
     <progress aria-label={label} value={count} max={applications.length||1}/>
    </div>)}</div>
   </section>:<section className="card section"><Header title="جاهزية طلباتي" subtitle="تابع مرحلة طلبك وموعد اللجنة ونتيجة تقييمها." action={<button className="btn outline" onClick={()=>navigate('calendar')}>عرض التقويم</button>}/>
    {applications.length?applications.map(a=><article className="item" key={a.id}><h3>{a.topic}</h3><div className="meta-grid"><span><b>المرحلة الحالية</b>{a.status==='accepted'?'مقبول · تنسيق الجلسة':a.status==='declined'?'غير مقبول':a.reviewComplete?'بانتظار القرار':a.appointment?'موعد اللجنة محدد':'بانتظار تحديد موعد اللجنة'}</span><span><b>موعد اللجنة</b>{a.appointment?dateTime(a.appointment):'لم يتحدد بعد'}</span><span><b>درجة اللجنة</b>{a.score==null?(a.reviewComplete?'اكتمل التقييم · بانتظار القرار':'لم يكتمل التقييم'):a.score+' / 100'}</span></div>{a.decision&&<p>{a.decision}</p>}</article>):<Empty>هنا تظهر مرحلة طلبك وموعد اللجنة بعد التقديم.</Empty>}
   </section>}
  </div>
 </div>;
}
