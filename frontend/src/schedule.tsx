import {useEffect,useRef,useState} from 'react';
import {Bell,CalendarDays,ChevronLeft,ChevronRight,Clock3} from 'lucide-react';
import {api,Header,Modal,perform,msg} from './ui';
import './schedule.css';

export type CalendarEvent={id:string;kind:'interview'|'lecture'|'draft';title:string;detail:string;at:string};
type Notice={id:string;kind:'interview'|'lecture'|'bulletin';title:string;detail:string;at:string;href:string;popup:boolean;read:boolean};
const labels={interview:'مقابلة اللجنة',lecture:'محاضرة معلنة',draft:'محاضرة قيد الإعداد'};
const weekdays=['الأحد','الاثنين','الثلاثاء','الأربعاء','الخميس','الجمعة','السبت'];
const weekdayInitials=['ح','ن','ث','ر','خ','ج','س'];
const colors={
 interview:['#873456','#713b72','#91425c'],
 lecture:['#1d6a65','#245f83','#345f45'],
 draft:['#78511d','#746145','#77503f'],
};
const riyadh='Asia/Riyadh';
const today=new Intl.DateTimeFormat('sv-SE',{timeZone:riyadh}).format(new Date());
const showDay=(value:string)=>new Intl.DateTimeFormat('ar-SA',{dateStyle:'full',calendar:'gregory',timeZone:riyadh}).format(new Date(value.slice(0,10)+'T12:00:00+03:00'));
const monthName=new Intl.DateTimeFormat('ar-SA',{month:'long',year:'numeric',calendar:'gregory',timeZone:'UTC'});
const shortDate=new Intl.DateTimeFormat('ar-SA',{day:'numeric',month:'long',calendar:'gregory',timeZone:'UTC'});
const currentTime=new Intl.DateTimeFormat('en-GB',{hour:'2-digit',minute:'2-digit',hourCycle:'h23',timeZone:riyadh});
const dayKey=(date:Date)=>date.toISOString().slice(0,10);
const calendarDate=(day:string)=>new Date(day+'T00:00:00Z');
function shiftDay(day:string,offset:number){
 const date=calendarDate(day);
 date.setUTCDate(date.getUTCDate()+offset);
 return dayKey(date);
}

type CalendarMode='month'|'week';
type ScheduledEvent=CalendarEvent&{color:string};
const eventTime=(event:CalendarEvent)=>event.at.slice(11,16);

export function ScheduleCalendar({events}:{events:CalendarEvent[]}){
 const [selected,setSelected]=useState(today);
 const [mode,setMode]=useState<CalendarMode>('month');
 const [opened,setOpened]=useState<ScheduledEvent>();
 const agendaRef=useRef<HTMLDivElement>(null);
 const active=calendarDate(selected);
 const monthStart=selected.slice(0,7)+'-01';
 const first=calendarDate(monthStart);
 const weekStart=shiftDay(selected,-active.getUTCDay());
 const monthGridStart=shiftDay(monthStart,-first.getUTCDay());
 const monthLength=new Date(Date.UTC(active.getUTCFullYear(),active.getUTCMonth()+1,0)).getUTCDate();
 const monthCells=Math.ceil((first.getUTCDay()+monthLength)/7)*7;
 const weekDays=Array.from({length:7},(_,i)=>shiftDay(weekStart,i));
 const monthDays=Array.from({length:monthCells},(_,i)=>shiftDay(monthGridStart,i));
 const sorted=[...events].sort((a,b)=>a.at.localeCompare(b.at));
 const grouped=new Map<string,ScheduledEvent[]>();
 for(const event of sorted){
  const key=event.at.slice(0,10);
  const dayEvents=grouped.get(key)??[];
  const palette=colors[event.kind];
  const kindIndex=dayEvents.filter(item=>item.kind===event.kind).length;
  dayEvents.push({...event,color:palette[kindIndex%palette.length]});
  grouped.set(key,dayEvents);
 }
 const agenda=grouped.get(selected)??[];
 const upcoming=sorted.find(event=>event.at.slice(0,16)>=today+'T'+currentTime.format(new Date())&&event.kind!=='draft');
 const weekHours=weekDays.flatMap(day=>(grouped.get(day)??[]).map(event=>Number(event.at.slice(11,13))));
 const startHour=Math.min(8,...weekHours);
 const endHour=Math.max(18,...weekHours);
 const hours=Array.from({length:endHour-startHour+1},(_,i)=>startHour+i);
 const range=mode==='month'?monthName.format(active):shortDate.format(calendarDate(weekDays[0]))+' – '+shortDate.format(calendarDate(weekDays[6]))+' '+active.getUTCFullYear();
 const move=(by:number)=>{
  if(mode==='week'){setSelected(shiftDay(selected,by*7));return}
  const next=dayKey(new Date(Date.UTC(active.getUTCFullYear(),active.getUTCMonth()+by,1)));
  setSelected(next.slice(0,7)===today.slice(0,7)?today:next);
 };
 const viewName=mode==='month'?'الشهر':'الأسبوع';
 const openEvent=(event:ScheduledEvent)=>{setSelected(event.at.slice(0,10));setOpened(event)};
 const eventButton=(event:ScheduledEvent)=><button key={event.id} type="button" className="calendar-event" style={{backgroundColor:event.color}} onClick={()=>openEvent(event)} aria-label={eventTime(event)+' · '+event.title+' · '+labels[event.kind]} title={event.title+' · '+labels[event.kind]}>
  <time dir="ltr">{eventTime(event)}</time><span>{event.title}</span>
 </button>;
 const dayButton=(day:string)=><button type="button" className="calendar-date" aria-label={showDay(day)+' · '+(grouped.get(day)?.length??0)+' مواعيد'} aria-pressed={day===selected} aria-current={day===today?'date':undefined} onClick={()=>setSelected(day)}>{calendarDate(day).getUTCDate()}</button>;
 return <section className="card section schedule-calendar" data-guide="knowledge-calendar">
  <Header title="تقويم المواعيد" subtitle="مقابلات اللجنة والمحاضرات بتوقيت الرياض."/>
  {upcoming&&<button type="button" className="calendar-upcoming" onClick={()=>setSelected(upcoming.at.slice(0,10))}>
   <CalendarDays size={18} aria-hidden="true"/><span><strong>أقرب موعد</strong> {upcoming.title}</span>
   <small>{shortDate.format(calendarDate(upcoming.at.slice(0,10)))} · <bdi>{eventTime(upcoming)}</bdi></small>
  </button>}
  <div className="calendar-toolbar">
   <div className="calendar-nav">
    <button type="button" className="calendar-arrow" onClick={()=>move(-1)} aria-label={viewName+' السابق'}><ChevronRight size={19}/></button>
    <button type="button" className="calendar-arrow" onClick={()=>move(1)} aria-label={viewName+' التالي'}><ChevronLeft size={19}/></button>
    <button type="button" className="calendar-today" onClick={()=>setSelected(today)}>اليوم</button>
   </div>
   <h3 aria-live="polite">{range}</h3>
   <div className="calendar-modes" role="group" aria-label="طريقة عرض التقويم">
    <button type="button" aria-pressed={mode==='month'} onClick={()=>setMode('month')}>الشهر</button>
    <button type="button" aria-pressed={mode==='week'} onClick={()=>setMode('week')}>الأسبوع</button>
   </div>
  </div>
  {mode==='month'?<div className="calendar-grid" aria-label="عرض الشهر">
   {weekdays.map((name,i)=><span className="calendar-weekday" key={name} title={name}><span className="calendar-full-day">{name}</span><span className="calendar-short-day">{weekdayInitials[i]}</span></span>)}
   {monthDays.map(day=>{
    const items=grouped.get(day)??[];
    return <div className={'calendar-day'+(day===selected?' is-selected':'')+(day===today?' is-today':'')+(day.slice(0,7)!==selected.slice(0,7)?' outside-month':'')} key={day}>
     {dayButton(day)}
     <div className="calendar-events">{items.slice(0,3).map(eventButton)}</div>
     {items.length>3&&<button className="calendar-more" type="button" aria-label={'عرض كل مواعيد '+showDay(day)} onClick={()=>{setSelected(day);agendaRef.current?.scrollIntoView({block:'start'})}}>+{items.length-3} أخرى</button>}
    </div>;
   })}
  </div>:<div className="calendar-week-scroll" role="region" aria-label="عرض الأسبوع حسب الساعة" tabIndex={0}>
   <div className="calendar-week-grid">
    <span className="calendar-weekday calendar-week-corner"><Clock3 size={16} aria-label="الوقت"/></span>
    {weekDays.map((day,i)=><div className={'calendar-weekday calendar-week-heading'+(day===today?' is-today':'')+(day===selected?' is-selected':'')} key={day}>{weekdays[i]} {dayButton(day)}</div>)}
    {hours.map(hour=>{
     const hourLabel=String(hour).padStart(2,'0');
     return <div className="calendar-hour-row" key={hour}>
      <time className="calendar-hour" dir="ltr">{hourLabel}:00</time>
      {weekDays.map(day=><div key={day} className={'calendar-hour-cell'+(day===today?' is-today':'')+(day===selected?' is-selected':'')}>
       {(grouped.get(day)??[]).filter(event=>event.at.slice(11,13)===hourLabel).map(eventButton)}
      </div>)}
     </div>;
    })}
   </div>
  </div>}
  <ul className="calendar-legend" aria-label="أنواع المواعيد">
   {(Object.keys(labels) as CalendarEvent['kind'][]).filter(kind=>kind!=='draft'||events.some(event=>event.kind==='draft')).map(kind=><li key={kind}><span style={{backgroundColor:colors[kind][0]}}/>{labels[kind]}</li>)}
  </ul>
  <div className="calendar-agenda" ref={agendaRef}>
   <h3>{showDay(selected)} <small>· {agenda.length} مواعيد</small></h3>
   {agenda.map(event=><article key={event.id} style={{borderInlineStartColor:event.color}}><time dir="ltr">{eventTime(event)}</time><div><b>{event.title}</b><small>{labels[event.kind]} · {event.detail}</small></div></article>)}
   {!agenda.length&&<p className="muted">لا توجد مواعيد في هذا اليوم.</p>}
  </div>
  {opened&&<Modal title={opened.title} onClose={()=>setOpened(undefined)}><div className="calendar-details">
   <strong>{labels[opened.kind]}</strong><p>{opened.detail}</p><p>{showDay(opened.at)} · <bdi>{eventTime(opened)}</bdi> بتوقيت الرياض</p>
  </div></Modal>}
 </section>;
}

export function NotificationCenter(){
 const [items,setItems]=useState<Notice[]>([]),[open,setOpen]=useState(false),[dismissed,setDismissed]=useState<string[]>([]),[error,setError]=useState('');
 useEffect(()=>{
  let active=true;
  const refresh=()=>{if(!document.hidden)void api<Notice[]>('/api/notifications').then(rows=>{if(active)setItems(rows)}).catch(()=>{})};
  refresh();
  const interval=window.setInterval(refresh,30000);
  window.addEventListener('focus',refresh);
  return ()=>{active=false;window.clearInterval(interval);window.removeEventListener('focus',refresh)};
 },[]);
 const unread=items.filter(n=>!n.read);
 const popup=!open?unread.find(n=>n.popup&&!dismissed.includes(n.id)&&(n.kind==='bulletin'||n.at.slice(0,10)>=today)):undefined;
 const mark=async(n:Notice)=>{
  await perform('notice.read',{id:n.id});
  setItems(old=>old.map(x=>x.id===n.id?{...x,read:true}:x));
 };
 const dismiss=(n:Notice)=>{
  setDismissed(old=>[...old,n.id]);
  void mark(n).catch(e=>setError(msg(e)));
 };
 const dateLabel=(n:Notice)=>showDay(n.at)+(n.kind==='bulletin'?'':' · '+n.at.slice(11,16)+' بتوقيت الرياض');
 const row=(n:Notice)=><article className="notice-entry" key={n.id}>
  <b>{n.title}</b><small>{dateLabel(n)}</small><p>{n.detail}</p>
  <div className="notice-actions"><a className="btn" href={n.href} onClick={e=>{e.preventDefault();void mark(n).catch(()=>{}).finally(()=>location.assign(n.href))}}>عرض التفاصيل</a>
   {!n.read&&<button type="button" className="btn" onClick={()=>dismiss(n)}>تم الاطلاع</button>}</div>
 </article>;
 return <>
  <button type="button" className="notice-bell" aria-label={'الإشعارات · '+unread.length} onClick={()=>{setError('');setOpen(true)}}><Bell size={19}/>{unread.length>0&&<sup>{unread.length}</sup>}</button>
  {open&&<Modal title="الإشعارات" onClose={()=>setOpen(false)}>{items.length?items.map(row):<p>لا توجد إشعارات حالياً.</p>}{error&&<p role="alert">{error}</p>}</Modal>}
  {popup&&<Modal title={popup.kind==='interview'?'موعد مقابلتك':popup.kind==='lecture'?'محاضرة قادمة':'نشرة جديدة'} onClose={()=>dismiss(popup)}>{row(popup)}</Modal>}
 </>;
}
