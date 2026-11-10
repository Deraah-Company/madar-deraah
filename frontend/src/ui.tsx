import React,{useEffect,useRef,useState} from 'react';
import {videoSources} from './media';

export const organization='إدارة التدريب والتطوير التنظيمي';
export const programs={trainer:'سفراء المعرفة',voice:'نبض درعه',family:'Deraah Family'};
export type Program=keyof typeof programs;

export type Profile={name:string;email:string;title:string;employee:string;department:string;bio:string;photo:string};
export type User=Profile&{id:string;status:string;system:boolean;trainer:string;voice:string;family?:string;grants:Record<string,boolean>|null};
export type Dept={id:string;name:string};
export type Board={visible:boolean;title:string;weights:Record<string,number>};
export type Ranking={id:string;name:string;photo:string;department:string;scores:Record<string,number|null>};
export type Portal={me:User;programs:Record<Program,boolean>;permissions:Record<string,boolean>;departments:Dept[];users?:User[];registration?:boolean;demoInstalled?:boolean;userPermissions?:Record<string,Record<string,boolean>>;permissionsCatalog?:Record<string,string>;audit?:{at:string;by:string;action:string;target:string}[];overview?:Record<string,number>};
export type Act=(action:string,data:unknown)=>Promise<void>;

export async function api<T>(url:string,options:RequestInit={}):Promise<T>{
 const r=await fetch(url,options);const data=await r.json();if(!r.ok){const e=new Error(data.error||'تعذر الاتصال');Object.assign(e,{status:r.status});throw e}return data;
}
export const post=<T,>(url:string,data:unknown,extra:RequestInit={})=>api<T>(url,{method:'POST',headers:{'Content-Type':'application/json'},body:JSON.stringify(data),...extra});
export const perform=(action:string,data:unknown)=>post('/api/action',{action,data});
export const msg=(e:unknown)=>e instanceof Error?e.message:'تعذر إتمام العملية';
export const date=(value:string)=>value?new Intl.DateTimeFormat('ar-SA',{calendar:'gregory',year:'numeric',month:'short',day:'numeric',timeZone:'Asia/Riyadh'}).format(new Date(value)):'—';
export const dateTime=(value:string)=>value?`${date(value.slice(0,10))} · ${value.slice(11,16)} بتوقيت الرياض`:'—';
export const day=()=>new Intl.DateTimeFormat('sv-SE',{timeZone:'Asia/Riyadh'}).format(new Date());
export const key=()=>crypto.randomUUID();
export const profile=(a?:Profile):Profile=>a?{name:a.name,email:a.email,title:a.title,employee:a.employee,department:a.department,bio:a.bio,photo:a.photo}:{name:'',email:'',title:'',employee:'',department:'',bio:'',photo:''};

export function Field({label,children}:{label:string;children:React.ReactNode}){return <label className="field"><span>{label}</span>{children}</label>}
export function Empty({children}:{children:React.ReactNode}){return <div className="empty">{children}</div>}
export function Badge({children,tone=''}:{children:React.ReactNode;tone?:string}){return <span className={'badge '+tone}>{children}</span>}
export function Avatar({name,photo,large=false}:{name:string;photo?:string;large?:boolean}){return <span className={'avatar '+(large?'large':'')}>{photo?<img src={photo} alt=""/>:name.trim().slice(0,1)||'د'}</span>}
export function Header({label,title,subtitle,action}:{label?:string;title:string;subtitle?:string;action?:React.ReactNode}){return <header className="section-head"><div>{label&&<div className="eyebrow">{label}</div>}<h2>{title}</h2>{subtitle&&<p>{subtitle}</p>}</div>{action}</header>}
export function Modal({title,children,onClose}:{title:string;children:React.ReactNode;onClose:()=>void}){
 const ref=useRef<HTMLDialogElement>(null);useEffect(()=>{const d=ref.current;d?.showModal();return()=>d?.close()},[]);
 return <dialog ref={ref} onCancel={e=>{e.preventDefault();onClose()}} className="modal"><div className="modal-head"><h2>{title}</h2><button className="icon" type="button" onClick={onClose} aria-label="إغلاق">×</button></div><div className="modal-body">{children}</div></dialog>
}
export function Form({onSave,children,label='حفظ',after}:{onSave:(form:FormData)=>Promise<unknown>;children?:React.ReactNode;label?:string;after?:()=>void}){
 const [busy,setBusy]=useState(false),[error,setError]=useState('');return <form onSubmit={async e=>{e.preventDefault();if(busy)return;const data=new FormData(e.currentTarget);setBusy(true);setError('');try{await onSave(data);after?.()}catch(e){setError(msg(e))}finally{setBusy(false)}}}>{children}{error&&<p className="error" role="alert">{error}</p>}<button className="btn primary" disabled={busy}>{busy?'جاري الحفظ…':label}</button></form>
}
export function ProfileFields({value,departments,employeeLocked=false}:{value:Profile;departments:Dept[];employeeLocked?:boolean}){
 return <div className="form-grid"><Field label="الاسم"><input name="name" required maxLength={100} defaultValue={value.name}/></Field><Field label="البريد الإلكتروني"><input type="email" name="email" required defaultValue={value.email} dir="ltr"/></Field><Field label="المسمى الوظيفي"><input name="title" required maxLength={120} defaultValue={value.title}/></Field><Field label="الرقم الوظيفي"><input name="employee" required maxLength={32} defaultValue={value.employee} readOnly={employeeLocked}/></Field><Field label="الإدارة"><select name="department" required defaultValue={value.department} disabled={employeeLocked}><option value="">اختر الإدارة</option>{departments.map(d=><option key={d.id} value={d.id}>{d.name}</option>)}</select></Field><Field label="صورة الحساب"><input type="file" name="picture" accept="image/png,image/jpeg"/><small>PNG أو JPG حتى 500 كيلوبايت</small></Field><Field label="نبذة قصيرة"><textarea name="bio" rows={2} maxLength={500} defaultValue={value.bio} placeholder="تُعرض نبذة من المسمى والإدارة عند تركها فارغة"/></Field>{value.photo&&<label className="check"><input type="checkbox" name="removePhoto"/>إزالة الصورة الحالية</label>}</div>
}
export async function readProfile(form:FormData,old:Profile):Promise<Profile>{const file=form.get('picture');let photo=form.has('removePhoto')?'':old.photo;if(file instanceof File&&file.size){if(file.size>500*1024||!['image/png','image/jpeg'].includes(file.type))throw new Error('الصورة لازم تكون PNG أو JPG بحجم 500 كيلوبايت أو أقل');photo=await new Promise<string>((resolve,reject)=>{const r=new FileReader();r.onload=()=>resolve(String(r.result));r.onerror=reject;r.readAsDataURL(file)})}return {name:String(form.get('name')||''),email:String(form.get('email')||''),title:String(form.get('title')||''),employee:String(form.get('employee')||old.employee),department:String(form.get('department')||old.department),bio:String(form.get('bio')||''),photo}}
export const metricLabels:Record<string,string>={combined:'الدرجة الإجمالية',panel:'لجنة التقديم',survey:'استبيان الجلسة',quiz:'الاختبار البعدي',manager:'تقييم مدير النظام'};
export function Rankings({board,rows,action}:{board:Board;rows:Ranking[];action?:React.ReactNode}){
 const [metric,setMetric]=useState('combined');if(!board.visible&&!action)return null;
 const ranking=rows.filter(r=>r.scores[metric]!=null).sort((a,b)=>(b.scores[metric]??0)-(a.scores[metric]??0)||a.name.localeCompare(b.name,'ar'));
 return <section className="card section"><Header label="المنافسة بين الإدارات" title={board.title} action={action}/>{!board.visible&&<p className="muted">لوحة المنافسة مخفية حالياً.</p>}<div className="tabset">{Object.entries(metricLabels).map(([k,v])=><button key={k} className={k===metric?'selected':''} onClick={()=>setMetric(k)}>{v}</button>)}</div>{ranking.length?<div className="ranking">{ranking.slice(0,10).map((r,i)=><div className="rank" key={r.id}><span className={'position '+(i===0?'gold':'')}>{i+1}</span><Avatar name={r.name} photo={r.photo}/><div className="grow"><b>{r.name}</b><small>{r.department}</small></div><strong>{r.scores[metric]} <small>/ 100</small></strong></div>)}</div>:<Empty>ستظهر النتائج بعد اكتمال التقييمات لهذا المسار.</Empty>}{metric==='combined'&&<small className="muted">تظهر الدرجة المجمعة عند اكتمال المقاييس المختارة في الأوزان.</small>}</section>
}
export function MediaSlot({title,description,slot,poster}:{title:string;description:string;slot:keyof typeof videoSources;poster?:string}){
 const source=videoSources[slot];
 if(slot==='promo'||slot==='login')return <figure className="promo-player">
  <div className="promo-screen">
   {source?<video src={source} poster={poster} controls playsInline preload="metadata" aria-label={title}/>:<div className="promo-placeholder"><small>{description}</small><strong>{title}</strong></div>}
  </div>
  {source&&<figcaption>{title}</figcaption>}
 </figure>;
 return <section className="media-slot" data-guide={slot+'-media'}>{source?<video src={source} controls playsInline preload="metadata" aria-label={title}/>:<span className="play" aria-hidden="true">▶</span>}<div><h3>{title}</h3>{!source&&<p>{description}</p>}</div></section>;
}
