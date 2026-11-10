import {api,useResource,useLocale} from './shared';
import type {Metric,Text} from './types';

export function PublishedScore({legacyOnly=false}:{legacyOnly?:boolean}) {
 const {local,tx}=useLocale();
 const {value}=useResource(()=>legacyOnly?Promise.resolve(null):api<{title:Text;overall:Metric}|null>('/overview'),[legacyOnly]);
 const {value:legacy}=useResource(async()=>{
  const response=await fetch('/api/voice');
  if(!response.ok) return null;
  const data=await response.json() as {surveys:{title:string;published:boolean;results?:{enps:number}}[]};
  return data.surveys.filter(s=>s.published&&s.results).at(-1);
 },[]);
 const score=value?.overall.enps??legacy?.results?.enps;
 if(legacyOnly&&score==null)return null;
 return <section className="panel padded pulse-about" aria-label="eNPS" data-guide="pulse-results">
  <h2>{tx('مؤشر التوصية بالعمل','Employee Net Promoter Score')} · eNPS</h2>
  {score!=null?<><p>{value?local(value.title):legacy?.title}</p><strong className="pulse-score">{score>0?'+':''}{score}</strong><small> {tx('من −100 إلى +100','From −100 to +100')}</small></>:<p>{tx('سيظهر المؤشر بعد نشر نتيجة الاستبيان.','The score appears when a result is published.')}</p>}
 </section>;
}

