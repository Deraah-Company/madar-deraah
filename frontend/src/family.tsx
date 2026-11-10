import React, {useEffect,useRef,useState} from 'react';
import {api, Badge, date, Empty, Field, Form, Header, MediaSlot, Modal, msg, perform, type Portal} from './ui';
import {ProgramWorkspace,ToolOverview} from './workspace';

type Media = {id:string;name:string};
type Comment = {id:string;name:string;text:string;at:string};
type Bulletin = {
 id:string;title:string;summary:string;at:string;hidden:boolean;
 pdf:Media|null;video:Media|null;comments:Comment[];
};
type FamilyData = {intro:Media|null;current:string;bulletins:Bulletin[]};
type Editor = {kind:'intro'} | {kind:'bulletin';item?:Bulletin};
const familyParams=new URLSearchParams(location.search);
const mediaURL = (file:Media) => '/files/family/'+file.id;

export function Family({portal}:{portal:Portal}) {
 const [data,setData]=useState<FamilyData>();
 const [error,setError]=useState('');
 const [selected,setSelected]=useState(familyParams.get('issue')||'');
 const [tab,setTab]=useState(familyParams.get('tab')==='current'?'current':'home');
 const [editor,setEditor]=useState<Editor>();
 const commentInput=useRef<HTMLTextAreaElement>(null);
 const [confirm,setConfirm]=useState<{title:string;body:string;action:string;payload:unknown}>();
 const permissions=portal.permissions;
 const manage=permissions['family.manage'];
 const moderate=permissions['family.moderate'];
 const refresh=async()=>setData(await api<FamilyData>('/api/family'));
 useEffect(()=>{refresh().catch(e=>setError(msg(e)))},[]);
 const act=async(action:string,payload:unknown)=>{
  setError('');
  await perform(action,payload);
  await refresh();
 };
 const change=async(action:string,payload:unknown)=>{
  try {await act(action,payload)} catch(e){setError(msg(e))}
 };
 if(!data)return error?<p className="error">{error}</p>:<Empty>جاري تحميل نشرات Deraah Family…</Empty>;
 const current=data.bulletins.find(b=>b.id===data.current);
 const selectedIssue=data.bulletins.find(b=>b.id===selected);
 const shownTab=tab==='current'&&selectedIssue&&selectedIssue.id!==current?.id?'archive':tab;
 const open=shownTab==='archive'?selectedIssue:current;
 const archive=data.bulletins.filter(b=>b.id!==data.current);
 const navigate=(target:string)=>{
  setTab(target);setSelected('');window.scrollTo(0,0);
 };
 return <ProgramWorkspace program="family" title="Deraah Family" tab={shownTab} navigate={navigate}
  tabs={ [['home','نظرة عامة'],['current','النشرة الحالية'],['archive','أرشيف النشرات']] }>
  {shownTab==='home'&&<><section className="program-hero family-hero" id="family-home">
   <div className="eyebrow">نشرات درعه</div>
   <h1 lang="en">Deraah Family</h1>
   <p>أخبار درعه ومبادراتها وتجارب زملائنا، في نشرة واحدة نقرأها ونتشارك الرأي فيها.</p>
   <div className="actions">
    <button className="btn" onClick={()=>navigate('archive')}>أرشيف النشرات</button>
    {manage&&<><button className="btn" onClick={()=>setEditor({kind:'bulletin'})}>إضافة نشرة</button><button className="btn" onClick={()=>setEditor({kind:'intro'})}>إدارة فيديو الترحيب</button></>}
   </div>
  </section>
  <ToolOverview program="family"/>
  <section className="card section">
   {data.intro?<><Header title="تعرّف على Deraah Family"/><video className="family-video" src={mediaURL(data.intro)} controls playsInline preload="metadata" aria-label="فيديو الترحيب بـ Deraah Family"/></>:<MediaSlot slot="family" title="تعرّف على Deraah Family" description="طريقة تصفح النشرات والتعليقات والأرشيف."/>}
  </section>
  </>}
  {error&&<p className="error" role="alert">{error}</p>}
  {(shownTab==='current'||shownTab==='archive'&&open)&&<section className="card section" id="family-current" data-guide="family-current">
   {open?<article key={open.id}>
    <Header label={open.id===data.current?'النشرة الحالية':open.hidden?'نشرة مخفية':'من الأرشيف'} title={open.title} subtitle={date(open.at)}
     action={shownTab==='archive'?<button className="btn" onClick={()=>navigate('archive')}>العودة للأرشيف</button>:undefined}/>
    {open.summary&&<p className="preserve family-summary">{open.summary}</p>}
    {manage&&<div className="actions family-tools">
     <button className="btn" onClick={()=>setEditor({kind:'bulletin',item:open})}>تعديل النشرة</button>
     {!open.hidden&&open.id!==data.current&&<button className="btn" onClick={()=>void change('family.feature',{id:open.id})}>تعيين كنشرة حالية</button>}
     <button className="btn" onClick={()=>void change('family.hide',{id:open.id,hidden:!open.hidden})}>{open.hidden?'إظهار النشرة':'إخفاء النشرة'}</button>
     <button className="btn danger" onClick={()=>setConfirm({title:'حذف النشرة',body:'سيتم حذف النشرة وملفاتها وتعليقاتها.',action:'family.delete',payload:{id:open.id}})}>حذف النشرة</button>
    </div>}
    {open.pdf&&<div className="family-document">
     <iframe src={mediaURL(open.pdf)} title={'ملف PDF · '+open.title}/>
     <a className="btn" href={mediaURL(open.pdf)} target="_blank" rel="noreferrer">افتح ملف PDF في تبويب جديد ↗</a>
    </div>}
    {open.video&&<video className="family-video" src={mediaURL(open.video)} controls playsInline preload="metadata" aria-label={open.title}/>}
    <section className="family-comments">
     <Header title="التعليقات" subtitle={open.comments.length?'عدد التعليقات '+open.comments.length:undefined}/>
     {open.comments.map(c=><div className="item" key={c.id}>
      <div className="row between"><div><b>{c.name}</b><small>{date(c.at)}</small></div>
       {moderate&&<button className="link danger" onClick={()=>setConfirm({title:'حذف تعليق',body:'سيتم حذف التعليق من النشرة.',action:'family.removeComment',payload:{id:open.id,comment:c.id}})}>حذف التعليق</button>}
      </div><p className="preserve">{c.text}</p>
     </div>)}
     {!open.comments.length&&<p className="muted">لسه مفيش تعليقات.</p>}
     {permissions['family.comment']&&!open.hidden&&<Form label="إرسال التعليق" onSave={async(f)=>{
      await act('family.comment',{id:open.id,text:f.get('comment')});
      if(commentInput.current)commentInput.current.value='';
     }}><Field label="اكتب تعليقك"><textarea ref={commentInput} name="comment" rows={3} maxLength={2000} required/></Field></Form>}
    </section>
   </article>:<><Header title="النشرة الحالية"/><Empty>تظهر هنا أول نشرة بعد نشرها.</Empty></>}
  </section>}
  {shownTab==='archive'&&!open&&<section className="card section" id="family-archive" data-guide="family-archive">
   <Header title="النشرات السابقة" subtitle={manage?'تظهر لك النشرات المخفية لإدارتها.':undefined}/>
   {archive.length?<div className="family-archive">{archive.map(b=><article className="item" key={b.id}>
    <h3>{b.title}</h3><small>{date(b.at)}</small><p>{b.summary}</p>
    <div className="actions">{b.hidden&&<Badge tone="wait">مخفية</Badge>}{b.pdf&&<Badge>PDF</Badge>}{b.video&&<Badge>فيديو</Badge>}</div>
    <button className="btn" onClick={()=>{setSelected(b.id);window.scrollTo(0,0)}}>عرض النشرة</button>
   </article>)}</div>:<Empty>لا توجد نشرات سابقة.</Empty>}
  </section>}
  {editor&&<FamilyEditor editor={editor} intro={data.intro} close={()=>setEditor(undefined)} save={async f=>{
   await api('/api/family/save',{method:'POST',body:f});await refresh();
  }}/>}
  {confirm&&<Modal title={confirm.title} onClose={()=>setConfirm(undefined)}><Form label={confirm.title} after={()=>setConfirm(undefined)} onSave={()=>act(confirm.action,confirm.payload)}><p>{confirm.body}</p></Form></Modal>}
 </ProgramWorkspace>;
}

function FamilyEditor({editor,intro,close,save}:{editor:Editor;intro:Media|null;close:()=>void;save:(f:FormData)=>Promise<void>}) {
 const isIntro=editor.kind==='intro';
 const item=editor.kind==='bulletin'?editor.item:undefined;
 const oldVideo=isIntro?intro:item?.video;
 return <Modal title={isIntro?'فيديو الترحيب':item?'تعديل النشرة':'إضافة نشرة'} onClose={close}>
  <Form label={isIntro||item?'حفظ':'نشر النشرة'} after={close} onSave={f=>save(f)}>
   {isIntro?<input type="hidden" name="mode" value="intro"/>:<>
    {item&&<input type="hidden" name="id" value={item.id}/>}
    <Field label="عنوان النشرة"><input name="title" defaultValue={item?.title} maxLength={160} required/></Field>
    <Field label="وصف النشرة"><textarea name="summary" defaultValue={item?.summary} rows={3} maxLength={4000}/></Field>
    <Field label="ملف النشرة PDF · حتى 20 ميجابايت"><input type="file" name="pdf" accept=".pdf,application/pdf"/>{item?.pdf&&<small>{item.pdf.name}</small>}</Field>
    {item?.pdf&&<label className="check"><input type="checkbox" name="removePDF"/>إزالة ملف PDF الحالي</label>}
   </>}
   <Field label="فيديو MP4 · حتى 50 ميجابايت"><input type="file" name="video" accept=".mp4,video/mp4"/>{oldVideo&&<small>{oldVideo.name}</small>}</Field>
   {oldVideo&&<label className="check"><input type="checkbox" name="removeVideo"/>إزالة الفيديو الحالي</label>}
   {!isIntro&&<p className="muted">أضف PDF أو فيديو، أو الاثنين في نفس النشرة.</p>}
  </Form>
 </Modal>;
}
