import {useState} from 'react';
import {ChevronDown,Download} from 'lucide-react';
import {PageHeading} from './dashboard';
import {api,useLocale,useResource,Loading,ErrorBox,Empty,Panel,Badge,task} from './shared';
import type {Survey} from './types';

export function CommentsPage({surveys}:{surveys:Survey[]}){
 const {tx,local}=useLocale();
 const [open,setOpen]=useState('');
 return <>
  <PageHeading title={tx('تعليقات الاستبيانات','Survey comments')} sub={tx('اختر الاستبيان لعرض تعليقاته وتنزيلها.','Select a survey to view or download its comments.')}/>
  <div className="comment-surveys">{surveys.map(s=><section className="panel comment-survey" key={s.id}>
   <button className="comment-survey-toggle" aria-expanded={open===s.id} onClick={()=>setOpen(old=>old===s.id?'':s.id)}>
    <ChevronDown size={18}/><strong>{local(s.title)}</strong><Badge status={s.status}/>
   </button>
   {open===s.id&&(s.status==='closed'&&s.response_count>0?<CommentList survey={s}/>:<div className="comment-survey-body"><Empty title={tx('التعليقات تتاح بعد إغلاق الاستبيان ووصول مشاركات.','Comments are available after closing a survey with responses.')}/></div>)}
  </section>)}</div>
  {!surveys.length&&<Panel><Empty title={tx('لا توجد استبيانات.','No surveys.')}/></Panel>}
 </>;
}
function CommentList({survey}:{survey:Survey}){
 const {lang,tx}=useLocale();
 const url='/surveys/'+encodeURIComponent(survey.id)+'/comments';
 const {value:comments,error,reload}=useResource(()=>api<{id:string;body:string;status:string}[]>(url),[url]);
 const moderate=(id:string,status:string)=>task(async()=>{await api('/comments/'+id,'PATCH',{status});reload()},lang);
 return <div className="comment-survey-body">
  {error?<ErrorBox code={error} retry={reload}/>:!comments?<Loading/>:comments.length===0?<Empty title={tx('لا توجد تعليقات.','No comments.')}/>:<>
   <div className="button-row"><a className="btn primary" href={'/pulse/api'+url+'?format=xlsx'} download><Download size={17}/> {tx('تنزيل Excel','Download Excel')}</a><span>{comments.length} {tx('تعليق','comments')}</span></div>
   <ol className="comment-rows">{comments.map(c=><li className="comment-card" key={c.id}>
    <Badge status={c.status}/><p>{c.body}</p><div className="button-row">
     <button disabled={c.status==='approved'} className="btn soft small" onClick={()=>moderate(c.id,'approved')}>{tx('اعتماد','Approve')}</button>
     <button disabled={c.status==='rejected'} className="btn ghost small" onClick={()=>moderate(c.id,'rejected')}>{tx('استبعاد','Exclude')}</button>
    </div>
   </li>)}</ol>
  </>}
 </div>;
}
