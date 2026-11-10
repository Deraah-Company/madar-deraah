import {useEffect,useState} from 'react';
import {api,Badge,date,Empty,Field,Form,Header,msg,type Dept} from './ui';

export const trainerApplyPath='/knowledge/apply';

type Question={id:string;label:string;required:boolean};
type PublicCampaign={id:string;title:string;goal:string;policy:string;deadline:string;questions?:Question[]};
type PublicData={campaigns:PublicCampaign[];departments:Dept[];account:null|{name:string;department:string;canApply:boolean}};

export function TrainerApplicationFields({questions,departments,department}:{questions?:Question[];departments:Dept[];department?:string}){
 return <>
  <Field label="الإدارة"><select name="department" required defaultValue={department||''}>
   <option value="">اختر إدارتك</option>{departments.map(d=><option key={d.id} value={d.id}>{d.name}</option>)}
  </select></Field>
  <Field label="عنوان الموضوع"><input name="topic" required maxLength={160}/></Field>
  <Field label="نقاط الشرح وما سيطبقه الحضور"><textarea name="outline" required rows={4} maxLength={3000}/></Field>
  {questions?.map(q=><Field key={q.id} label={q.label}><textarea name={'extra-'+q.id} required={q.required} rows={2} maxLength={2000}/></Field>)}
  <Field label="مادة الشرح · اختياري"><input name="file" type="file" accept=".pdf,.pptx,.docx"/><small>PDF أو PPTX أو DOCX حتى 10 ميجابايت</small></Field>
 </>;
}

export function PublicTrainer(){
 const [data,setData]=useState<PublicData>(),[error,setError]=useState(''),[selected,setSelected]=useState(''),[done,setDone]=useState(false);
 useEffect(()=>{
  api<PublicData>('/api/public/knowledge').then(value=>{
   setData(value);
   const requested=new URLSearchParams(location.search).get('round');
   setSelected(value.campaigns.find(c=>c.id===requested)?.id||value.campaigns[0]?.id||'');
  }).catch(e=>setError(msg(e)));
 },[]);
 const campaign=data?.campaigns.find(c=>c.id===selected);
 return <>
  <header className="site-head">
   <a className="brand" href="/" aria-label="مدار درعه"><img src="/deraah-logo-v2.png" alt="درعه"/></a>
   <a className="head-action head-home" href="/">الرئيسية</a>
  </header>
  <main className="shell trainer-public">
   <section className="program-hero trainer-hero"><div className="eyebrow">إدارة التدريب والتطوير التنظيمي</div>
    <h1>التقديم في سفراء المعرفة</h1><p>شارك موضوعاً من خبرتك يستفيد منه زملاؤك. اختر إدارتك من النموذج وأرسل طلبك للجنة التقديم.</p>
   </section>
   <section className="card section">
    {error?<p className="error" role="alert">{error}</p>:!data?<Empty>جاري تحميل فرص التقديم…</Empty>
    :done?<div className="notice" role="status"><Header title="تم إرسال طلبك" subtitle="وصل طلبك إلى فريق سفراء المعرفة."/></div>
    :!campaign?<Empty>لا توجد جولات تقديم مفتوحة حالياً.</Empty>
    :data.account&&!data.account.canApply?<Empty>حسابك الحالي لا يملك صلاحية التقديم.</Empty>
    :<>
     <Header title="نموذج التقديم" subtitle="التقديم مفتوح لموظفي جميع الإدارات." action={<Badge tone="good">التقديم مفتوح</Badge>}/>
     {data.campaigns.length>1&&<Field label="فرصة التقديم"><select value={selected} onChange={e=>setSelected(e.target.value)}>
      {data.campaigns.map(c=><option key={c.id} value={c.id}>{c.title}</option>)}
     </select></Field>}
     <div className="item"><h3>{campaign.title}</h3><p>{campaign.goal}</p><small>آخر يوم للتقديم {date(campaign.deadline)}</small>
      <details><summary>سياسة التقديم</summary><p className="preserve">{campaign.policy}</p></details>
     </div>
     <Form key={campaign.id} label="إرسال الطلب" after={()=>setDone(true)} onSave={f=>{
      f.set('campaign',campaign.id);
      const file=f.get('file');
      if(file instanceof File&&file.size>10*1024*1024)throw new Error('حجم الملف لا يتجاوز 10 ميجابايت');
      if(file instanceof File&&!file.name)f.delete('file');
      return api('/api/public/knowledge',{method:'POST',body:f});
     }}>
      {data.account?<p>التقديم باسم <b>{data.account.name}</b></p>:<div className="form-grid">
       <Field label="الاسم"><input name="name" autoComplete="name" required maxLength={100}/></Field>
       <Field label="البريد الإلكتروني للتواصل"><input name="email" type="email" autoComplete="email" required maxLength={254} dir="ltr"/></Field>
       <Field label="الرقم الوظيفي"><input name="employee" required maxLength={32}/></Field>
      </div>}
      <TrainerApplicationFields questions={campaign.questions} departments={data.departments} department={data.account?.department}/>
     </Form>
    </>}
   </section>
  </main>
 </>;
}
