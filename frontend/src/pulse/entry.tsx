import {Tour,type GuideStep} from '../tour';
import {NotificationCenter} from '../schedule';
import {useEffect,useState} from 'react';
import {createRoot} from 'react-dom/client';
import {AdminApp} from './admin';
import {PublicSurvey} from './public';
import {Locale,api,Loading,Toasts} from './shared';
import type {Lang,User} from './types';
import {PulseRoadmap} from './dashboard';
import {PublishedScore} from './score';
import './styles.css';

const guide:GuideStep[]=[
 ['صوتك يساعدنا نحدد الأولويات','شارك من الرابط المرسل لك بدون تسجيل دخول.','pulse-hero'],
 ['شاهد مؤشر التوصية بالعمل','ستجد هنا المؤشر عندما تنشر الإدارة النتيجة.','pulse-results'],
 ['تابع خطط التحسين','تابع الخطوات المنشورة لتحسين تجربة العمل.','pulse-tab-plans','pulse-history','pulse-hero'],
];
function Pulse() {
 const [lang,setLang]=useState<Lang>(()=>localStorage.getItem('voice-language')==='en'?'en':'ar');
 const [guideOpen,setGuideOpen]=useState(()=>new URLSearchParams(location.search).get('tour')==='1'||sessionStorage.getItem('deraah-tour')==='on'&&!sessionStorage.getItem('deraah-tour-pulse'));
 const closeGuide=()=>{sessionStorage.setItem('deraah-tour-pulse','seen');setGuideOpen(false)};
 const [user,setUser]=useState<User|null>();
 const token=location.pathname.startsWith('/pulse/r/')?location.pathname.slice('/pulse/r/'.length):'';
 const tx=(ar:string,en:string)=>lang==='ar'?ar:en;
 const toggle=()=>setLang(value=>value==='ar'?'en':'ar');
 useEffect(()=>{
  document.documentElement.lang=lang;
  document.documentElement.dir=lang==='ar'?'rtl':'ltr';
  localStorage.setItem('voice-language',lang);
 },[lang]);
 useEffect(()=>{
  if(!token) api<User>('/auth/me').then(setUser).catch(()=>setUser(null));
 },[token]);
 return <Locale.Provider value={{lang,tx,local:t=>t[lang]}}>
  {token?<PublicSurvey token={token} toggle={toggle}/>
  :user===undefined?<Loading/>
  :user===null?<main className="public-container"><a className="btn primary" href="/login">{tx('الدخول إلى مدار درعه','Sign in to Madar Deraah')}</a></main>
  :!user.can_edit&&!user.can_results&&!user.can_plan&&!user.can_review&&!user.can_publish?
    <div className="admin-app"><header className="topbar"><a href="/"><img width="105" src="/deraah-logo-v2.png" alt="درعه"/></a><div className="top-actions"><NotificationCenter/><a className="top-button" href="/">{tx('الرئيسية','Home')}</a></div></header><main className="public-container">
     <h1 data-guide="pulse-hero">{tx('نبض درعه','Deraah Pulse')}</h1>
     <p>{tx('نقيس تجربة الموظفين ونحوّل النتائج إلى خطط تحسين. تقدر تشارك من رابط الاستبيان المرسل لك بدون تسجيل دخول.','We measure employee experience and turn the results into action plans. Open your invitation to participate anonymously.')}</p>
     <PulseRoadmap/>
     <PublishedScore/>
     <button className="btn ghost" onClick={()=>setGuideOpen(true)}>{tx('الجولة التعليمية','Guided tour')}</button>
     <a className="btn ghost" data-guide="pulse-history" href="/pulse/history">{tx('النتائج وخطط التحسين المنشورة سابقاً','Previously published results and plans')}</a>
    </main></div>
  :<AdminApp user={user} toggle={toggle} onTour={()=>setGuideOpen(true)} onLogout={()=>location.assign('/login')}/>}
  {user&&!token&&guideOpen&&<Tour section="pulse" slides={guide} onClose={closeGuide} onSkipAll={()=>{sessionStorage.setItem('deraah-tour','off');closeGuide()}}/>}
  <Toasts/>
 </Locale.Provider>;
}

createRoot(document.getElementById('root')!).render(<Pulse/>);
