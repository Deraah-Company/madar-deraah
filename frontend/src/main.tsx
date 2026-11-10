import {AssessmentPage,assessmentPath} from './assessments';
import {NotificationCenter} from './schedule';
import {PublicTrainer,trainerApplyPath} from './trainer-public';
import React, {useCallback, useEffect, useState} from 'react';
import {createRoot} from 'react-dom/client';
import {ArrowUpLeft, BarChart3, BriefcaseBusiness, CircleHelp, House, LogOut, MessageCircle, Settings, ShieldCheck, Target} from 'lucide-react';
import {Admin} from './admin';
import {Family} from './family';
import {Trainer} from './trainer';
import {PublicSurvey, Voice} from './voice';
import {Tour, type GuideStep} from './tour';
import {api, Avatar, Field, Form, Header, MediaSlot, Modal, msg, organization, perform, Portal, post, profile, ProfileFields, readProfile, type Dept} from './ui';
import './styles.css';

const area = location.pathname === '/knowledge' ? 'knowledge' : (location.pathname === '/pulse' || location.pathname === '/pulse/history') ? 'pulse' : location.pathname === '/family' ? 'family' : 'portal';
const isLogin = location.pathname === '/login';
function afterLoginPath(){
 const next=new URLSearchParams(location.search).get('next');
 if(!next)return '/';
 try{const url=new URL(next,location.origin);return url.origin===location.origin&&url.pathname===assessmentPath?url.pathname+url.search:'/'}catch{return '/'}
}
const isSettings = location.pathname === '/settings';
const tourSteps:Record<string,GuideStep[]> = {
 portal:[
  ['أهلاً بك في مدار درعه','من هنا تبدأ. بحسابك الواحد تقدر تدخل البرامج المتاحة لك.','welcome'],
  ['اختر برنامجك','شارك خبراتك في سفراء المعرفة، تابع خطط التحسين في نبض درعه، واقرأ نشرات Deraah Family.','program-voice','programs'],
  ['ملفك الشخصي','اضغط على صورتك لتعديل بياناتك، أو افتح سفراء المعرفة لمتابعة لوحة الصدارة.','profile'],
 ],
 settings:[['إعدادات البوابة','من هنا يدير مسؤول النظام الحسابات والموافقات وصلاحيات البرامج.','settings-tabs']],
 knowledge:[
  ['شارك خبرتك','شاهد فرص التقديم المفتوحة لكل الإدارات. راجع المواعيد والمعايير قبل رفع مادتك.','knowledge-tab-campaigns'],
  ['لجنة التقديم','بعد التقديم تابع الموعد والقرار. اللجنة تقيّم عرضك بمعايير معلنة.','knowledge-tab-applications'],
  ['من الخبرة إلى التطبيق','بعد الجلسة يشارك الحضور تقييمهم ويختبرون ما تعلموه. تظهر النتائج في لوحة الصدارة.','knowledge-tab-sessions'],
 ],
 pulse:[
  ['صوتك يساعدنا نحدد الأولويات','عندما يُرسل لك رابط الاستبيان، تقدر تجاوب من غير تسجيل دخول.','pulse-hero'],
  ['شاهد النتائج المنشورة','بعد انتهاء الجولة، ستظهر هنا النتائج الإجمالية التي تنشرها الإدارة.','pulse-results','pulse-tab-results'],
  ['تابع التحسينات','ستجد المحاور المختارة والخطوات التي بدأت والمرحلة الحالية لكل خطوة.','pulse-tab-plans'],
 ],
 family:[
  ['النشرة الحالية','اقرأ ملف النشرة أو شغّل فيديوها من نفس الصفحة.','family-tab-current'],
  ['التعليقات والأرشيف','شارك رأيك تحت النشرة وافتح تبويب الأرشيف لاستعراض الإصدارات السابقة.','family-tab-archive'],
 ],
 'knowledge-campaigns':[
  ['فرص التقديم','راجع الهدف والموعد وأسماء اللجنة في بطاقة الفرصة.','knowledge-campaign-card','knowledge-campaigns'],
  ['إرسال الطلب','افتح نموذج التقديم واكتب فكرة الشرح وأرفق المادة إذا كانت جاهزة.','knowledge-apply','knowledge-campaigns'],
 ],
 'knowledge-applications':[
  ['طلبات التقديم','تابع طلبك وموعد اللجنة من هنا.','knowledge-application','knowledge-applications'],
  ['القرار','بعد انتهاء التقييم يظهر القرار والدرجة. مدير البرنامج يقدر يقارن الطلبات.','knowledge-application','knowledge-applications'],
 ],
 'knowledge-sessions':[
  ['الجلسات','شوف موضوع الجلسة والموعد واسم السفير.','knowledge-session','knowledge-sessions'],
  ['بعد الجلسة','لما يتاح التقييم، قيّم الشرح وجاوب على الاختبار البعدي.','knowledge-session','knowledge-sessions'],
 ],
 'knowledge-report':[['مشاركة الإدارات','التقرير يوضح عدد المتقدمين والمقبولين من كل إدارة. اختر جولة بعينها من القائمة.','knowledge-report']],
 'knowledge-learn':[['الفيديو التعليمي','هنا مكان فيديو يوضح خطوات البرنامج من التقديم لحد الجلسة.','knowledge-media']],
 'pulse-plans':[
  ['من النتائج إلى التنفيذ','كل بطاقة توضح خطوة تحسين والمسؤول عنها والمرحلة الحالية.','pulse-plan-card','pulse-plans'],
  ['تابع التقدم','تظهر نسبة التنفيذ إذا حددتها الإدارة، والمرحلة تظل واضحة في الحالتين.','pulse-plan-progress','pulse-plan-card','pulse-plans'],
 ],
 'pulse-campaigns':[
  ['جولات الاستبيان','من هنا يجهّز المسؤول جولة منتصف العام أو نهاية العام ويشارك رابطها.','pulse-survey-card','pulse-campaigns'],
  ['نشر النتائج','بعد إقفال الجولة، يختار المسؤول إتاحة النتائج المجمّعة للموظفين.','pulse-publish','pulse-campaigns'],
 ],
 'pulse-learn':[['الفيديو التعليمي','هنا مكان فيديو يوضح المشاركة في الاستبيان ومتابعة النتائج والتحسينات.','pulse-media']],
};

function Brand(){return <a className="brand" href="/" aria-label="مدار درعه"><img src="/deraah-logo-v2.png" alt="درعه"/><span>مدار درعه<small className="brand-org">{organization}</small></span></a>}

function Login({onLogin}:{onLogin:()=>Promise<void>}){
 type PublicConfig={registration:boolean;departments:Dept[];programs:Portal['programs']};
 const [config,setConfig]=useState<PublicConfig>(),[register,setRegister]=useState(false),[sent,setSent]=useState(false),[error,setError]=useState('');
 useEffect(()=>{api<PublicConfig>('/api/public').then(setConfig).catch(e=>setError(msg(e)))},[]);
 return <main className="auth">
  <section className="auth-hero">
   <img className="auth-logo" src="/deraah-logo-v2.png" alt="درعه"/>
   <div>
    <div className="eyebrow">{organization}</div>
    <h1>حيّاك في <em>مدار درعه.</em></h1>
    <p>برامج ومبادرات درعه في مكان واحد.</p>
    <MediaSlot slot="login" title="أهلًا بك في مدار درعه" description="فيديو الترحيب"
     poster="/media/madar-welcome-poster.jpg"/>
   </div>
   <small className="auth-credit">درعه · {organization}</small>
  </section>
  <section className="auth-side"><div className="card auth-card">
   <img className="auth-logo" src="/deraah-logo-v2.png" alt="درعه"/>
   <div className="auth-icon"><BriefcaseBusiness aria-hidden="true"/></div>
   <h2>{register?'طلب حساب جديد':'أهلاً بك من جديد'}</h2>
   <p className="auth-subtitle">{register?'يراجع مدير النظام طلبك قبل تفعيل الحساب.':'سجّل دخولك إلى مدار درعه'}</p>
   {error&&<p className="error">{error}</p>}
   {sent&&<p className="success" role="status">تم إرسال طلب التسجيل. تقدر تسجّل دخولك بعد موافقة مدير النظام.</p>}
   {register&&config?.registration?<Form label="إرسال طلب التسجيل" onSave={async f=>{await post('/api/register',{...await readProfile(f,profile()),password:f.get('password')});setRegister(false);setSent(true)}}>
    <ProfileFields value={profile()} departments={config.departments}/><Field label="كلمة المرور"><input name="password" type="password" minLength={12} maxLength={256} required autoComplete="new-password"/><small>12 حرفاً على الأقل</small></Field>
   </Form>:<Form label="تسجيل الدخول" onSave={async f=>{await post('/api/login',{email:f.get('email'),password:f.get('password')});for(const k of Object.keys(tourSteps))sessionStorage.removeItem('deraah-tour-'+k);sessionStorage.setItem('deraah-tour',f.has('tour')?'on':'off');if(isLogin)location.replace(afterLoginPath());else await onLogin()}}>
    <Field label="البريد الإلكتروني"><input name="email" type="email" required autoComplete="username" dir="ltr"/></Field><Field label="كلمة المرور"><input name="password" type="password" required autoComplete="current-password"/></Field>
    <label className="check"><input name="tour" type="checkbox" defaultChecked/> أريد جولة تعليمية</label>
   </Form>}
   {config?.registration&&<>
    <div className="auth-divider">{register?'لديك حساب؟':'للموظفين'}</div>
    <button className="btn auth-toggle" onClick={()=>{setRegister(!register);setSent(false)}}>{register?'العودة لتسجيل الدخول':'إنشاء حساب موظف'} <ArrowUpLeft size={15} aria-hidden="true"/></button>
   </>}
   <small className="auth-credit"><ShieldCheck size={14} aria-hidden="true"/>{organization}</small>
  </div></section>
 </main>;
}

function App(){
 const [data,setData]=useState<Portal|null>(),[error,setError]=useState(''),[profileOpen,setProfileOpen]=useState(false),[tour,setTour]=useState(''),[guideSection,setGuideSection]=useState<string>(isSettings?'settings':area);
 const program = area === 'knowledge' ? 'trainer' : area === 'pulse' ? 'voice' : area;
 const tourAllowed = Boolean(data && !isLogin && (isSettings ? data.me.system
  : program === 'portal' || (data.programs[program] && data.permissions[program+'.view'])));
 const refresh=useCallback(async()=>{try{setData(await api<Portal>('/api/portal'));setError('')}catch(e){if((e as {status?:number}).status===401)setData(null);else setError(msg(e))}},[]);
 useEffect(()=>{void refresh()},[refresh]);
 useEffect(()=>{
  const autoStart = new URLSearchParams(location.search).get('tour') === '1' || (sessionStorage.getItem('deraah-tour') === 'on' && !sessionStorage.getItem('deraah-tour-'+guideSection));
  setTour(current => tourAllowed && (autoStart || current === guideSection) ? guideSection : '');
 },[tourAllowed,guideSection]);
 useEffect(()=>{if(data===null&&location.pathname==='/')location.replace('/login');if(data&&isLogin)location.replace(afterLoginPath())},[data]);
 useEffect(()=>{
  const listener=(event:Event)=>{const section=String((event as CustomEvent).detail);setGuideSection(tourSteps[section]?section:area)};
  window.addEventListener('deraah-guide',listener);
  return ()=>window.removeEventListener('deraah-guide',listener);
 },[]);
 const closeTour=()=>{sessionStorage.setItem('deraah-tour-'+tour,'seen');setTour('')};
 const skipAllTours=()=>{sessionStorage.setItem('deraah-tour','off');closeTour()};
 const act=async(action:string,payload:unknown)=>{await perform(action,payload);await refresh()};
 if(data===undefined)return <main className="loading"><Brand/><p>{error||'جاري تحميل مدار درعه…'}</p>{error&&<button className="btn" onClick={()=>void refresh()}>إعادة المحاولة</button>}</main>;
 if(!data)return <Login onLogin={refresh}/>;
 if(isLogin)return null;
 const modules=[
  {href:'/knowledge',program:'trainer',key:'trainer.view',Icon:Target,title:'سفراء المعرفة',subtitle:'شارك خبرتك، قدّم للجنة، وساعد زملاءك يطبّقون ما تعلموه.',className:'knowledge'},
  {href:'/pulse',program:'voice',key:'voice.view',Icon:MessageCircle,title:'نبض درعه',subtitle:'تابع نتائج الاستبيانات وخطط التحسين في بيئة العمل.',className:'pulse'},
  {href:'/family',program:'family',key:'family.view',Icon:BarChart3,title:'Deraah Family',subtitle:'اقرأ النشرة الحالية والنشرات السابقة وشارك رأيك.',className:'family'},
 ] as const;
 const available=modules.filter(m=>data.programs[m.program]);
 const active=available.find(m=>m.program===program);
 return <><header className="site-head">
  <Brand/>
  <div className="account-actions">
   <NotificationCenter/>
   {(area!=='portal'||isSettings)&&<a className="head-action head-home" href="/"><House size={17} aria-hidden="true"/>الرئيسية</a>}
   {data.me.system&&!isSettings&&<a className="head-action" href="/settings" aria-label="الإعدادات" title="الإعدادات"><Settings size={18} aria-hidden="true"/></a>}
   {tourAllowed&&<button className="head-action" aria-label="جولة" title="جولة" onClick={()=>setTour(guideSection)}><CircleHelp size={18} aria-hidden="true"/></button>}
   <button className="head-action head-account" data-guide="profile" aria-label="ملفي الشخصي" title={data.me.name} onClick={()=>setProfileOpen(true)}><Avatar name={data.me.name} photo={data.me.photo}/></button>
   <button className="head-action" aria-label="خروج" title="خروج" onClick={()=>{void post('/api/logout',{}).then(()=>location.assign('/login')).catch(e=>setError(msg(e)))}}><LogOut size={18} aria-hidden="true"/></button>
  </div>
 </header>
 <main className={area==='portal'?'shell':'shell workspace-shell'}>
  {error&&<p className="error">{error}</p>}
  {isSettings?data.me.system?<Admin data={data} act={act}/>:<section className="card section"><Header title="الإعدادات متاحة لمدير النظام"/><a className="btn" href="/">العودة للبوابة</a></section>:area==='portal'?<>
   <section className="welcome" data-guide="welcome">
    <div className="eyebrow">إدارة التدريب والتطوير التنظيمي</div>
    <h1>أهلاً، {data.me.name}</h1>
    <p>اختر البرنامج اللي يناسبك وتابع آخر المستجدات من مكان واحد.</p>
   </section>
   <section className="card program-panel" aria-label="البرامج" data-guide="programs">
    {available.length>0?<div className="program-grid">{available.map(m=>{
     const enabled=data.permissions[m.key];
     const target='program-'+m.program;
     const body=<>
      <span className="module-icon"><m.Icon size={24} aria-hidden="true"/></span>
      <h2>{m.title}</h2>
      <span className="program-status">{enabled?'متاح':'حسب الصلاحيات'}</span>
      <p>{m.subtitle}</p>
     </>;
     return enabled
      ? <a key={m.key} href={m.href} data-guide={target} className={'program '+m.className}>{body}</a>
      : <article key={m.key} data-guide={target} className={'program disabled '+m.className}>{body}</article>;
    })}</div>:<p className="notice">البرامج مغلقة حالياً.</p>}
   </section>
   {data.me.system&&data.overview&&<div className="stats overview">{Object.entries({campaigns:'فرص التقديم',applications:'طلبات المشاركة',sessions:'الجلسات',surveys:'جولات نبض درعه',bulletins:'نشرات Deraah Family'}).map(([key,label])=><div key={key}><small>{label}</small><strong>{data.overview?.[key]||0}</strong></div>)}</div>}
   <section className="card section portal-intro">
    <Header title="تعرّف على مدار درعه"/>
    <MediaSlot slot="promo" title="دليل برامج مدار درعه" description="الفيديو التعريفي للبوابة"/>
   </section>
  </>:active&&data.permissions[active.key]?(area==='knowledge'?<Trainer portal={data}/>:area==='pulse'?<Voice portal={data}/>:<Family portal={data}/>):<section className="card section"><Header title={active?'البرنامج غير متاح لحسابك':'البرنامج مغلق حالياً'} subtitle={active?'تواصل مع مدير النظام إذا كنت تحتاج صلاحية الدخول.':undefined}/><a className="btn" href="/">العودة للبوابة</a></section>}
 </main>
 {profileOpen&&<Modal title="ملفي الشخصي" onClose={()=>setProfileOpen(false)}><div className="profile-intro"><Avatar name={data.me.name} photo={data.me.photo} large/><div><strong>{data.me.name}</strong><p>{data.me.bio}</p></div></div><Form onSave={async f=>act('profile.update',await readProfile(f,data.me))} after={()=>setProfileOpen(false)}><ProfileFields value={data.me} departments={data.departments} employeeLocked/></Form></Modal>}
 {tourAllowed&&tour&&<Tour section={tour} slides={tourSteps[tour]} key={tour} onClose={closeTour} onSkipAll={skipAllTours}/>}
 </>;
}

createRoot(document.getElementById('root')!).render(location.pathname==='/survey'?<PublicSurvey/>:location.pathname===trainerApplyPath?<PublicTrainer/>:location.pathname===assessmentPath?<AssessmentPage/>:<App/>);
