import React from 'react';
import {Archive, BarChart3, CalendarDays, ClipboardList, LayoutGrid, Newspaper, PlayCircle, UsersRound} from 'lucide-react';
import {Header} from './ui';

const overviews = {
 knowledge: {
  goal: 'تحويل الخبرات العملية من فرق درعه إلى جلسات داخلية يقدر الزملاء يطبقونها في شغلهم.',
  use: 'قدّم موضوعك من «طلبات التقديم». تراجع اللجنة الموضوع، وبعد القبول تُتاح الجلسة للموظفين مع استبيان حضور واختبار بعدي.',
  value: 'تنتقل الخبرة بين الإدارات، وتعرف إدارة التدريب من التقييم والاختبار مدى استفادة المشاركين وأين يحتاج المحتوى إلى تطوير.',
 },
 pulse: {
  goal: 'فهم تجربة الموظفين وتحديد أولويات تحسين بيئة العمل.',
  use: 'يشارك الموظفون عبر رابط استبيان مجهول. تُعرض النتائج المجمعة بعد النشر، وتتابع الإدارة المسؤول عن كل خطوة ونسبة تنفيذها.',
  value: 'يتحول رأي الموظفين إلى خطوات تحسين يمكن متابعتها، ويقدر الموظفون يشوفون التقدم في الخطوات المنشورة.',
 },
 family: {
  goal: 'جمع أخبار درعه ومبادرات الفرق في نشرة داخلية واحدة.',
  use: 'افتح النشرة الحالية لقراءة ملفها أو مشاهدة الفيديو المرفق. تقدر ترجع للأعداد من الأرشيف وتشارك رأيك بالتعليقات حسب صلاحيتك.',
  value: 'توصل الأخبار للفرق في مكان واحد، وتظل الأعداد السابقة سهلة الرجوع، وتقدر الإدارة تسمع مقترحات الزملاء على كل نشرة.',
 },
};

export function ToolOverview({program}: {program: keyof typeof overviews}) {
 const {goal, use, value} = overviews[program];
 return <section className="card section tool-overview" aria-label="الهدف والقيمة المضافة">
  <Header title="عن الأداة"/>
  <dl>
   <div><dt>الهدف</dt><dd>{goal}</dd></div>
   <div><dt>طريقة الاستخدام</dt><dd>{use}</dd></div>
   <div><dt>القيمة المضافة</dt><dd>{value}</dd></div>
  </dl>
 </section>;
}

const icons = {
 home: LayoutGrid, results: BarChart3, campaigns: ClipboardList, applications: UsersRound,
 sessions: CalendarDays, report: BarChart3, plans: ClipboardList, learn: PlayCircle,
 current: Newspaper, archive: Archive,
};

export function ProgramWorkspace({program, title, tabs, tab, navigate, children}: {
 program: 'knowledge' | 'pulse' | 'family';
 title: string;
 tabs: string[][];
 tab: string;
 navigate: (tab: string) => void;
 children: React.ReactNode;
}) {
 return <div className={`program-workspace workspace-${program}`}>
  <aside className="workspace-sidebar">
   <nav aria-label={`أقسام ${title}`}>
    {tabs.map(([key,label]) => {
     const Icon = icons[key as keyof typeof icons] || LayoutGrid;
     return <button key={key} data-guide={`${program}-tab-${key}`} aria-current={tab === key ? 'page' : undefined} onClick={() => navigate(key)}>
      <Icon size={19} aria-hidden="true"/><span>{label}</span>
     </button>;
    })}
   </nav>
  </aside>
  <div className="workspace-content">
   {children}
  </div>
 </div>;
}
