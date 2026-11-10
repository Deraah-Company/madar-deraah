import {useId} from 'react';
import './roadmap.css';

export function Roadmap({title,goalLabel,goal,valueLabel,value,steps}:{
 title:string;goalLabel:string;goal:string;valueLabel:string;value:string;steps:{title:string;detail:string}[];
}){
 const id=useId();
 return <section className="program-roadmap" aria-labelledby={id}>
  <h2 id={id}>{title}</h2>
  <dl className="roadmap-purpose">
   <div><dt>{goalLabel}</dt><dd>{goal}</dd></div>
   <div><dt>{valueLabel}</dt><dd>{value}</dd></div>
  </dl>
  <ol className="roadmap-steps">
   {steps.map(({title,detail},i)=><li key={title}>
    <span className="roadmap-number" aria-hidden="true">{String(i+1).padStart(2,'0')}</span>
    <h3>{title}</h3><p>{detail}</p>
   </li>)}
  </ol>
 </section>;
}
