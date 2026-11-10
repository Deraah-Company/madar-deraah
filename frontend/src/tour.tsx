import './tour.css';
import React, {useEffect, useRef, useState} from 'react';
import {guideAudio} from './media';

export type GuideStep = [title: string, body: string, ...targets: string[]];

const guidePoses = ['12 152 286 759', '625 152 437 759', '620 969 325 761'];

export function Tour({section, slides, onClose, onSkipAll}: {
 section: string;
 slides: GuideStep[];
 onClose: () => void;
 onSkipAll: () => void;
}) {
 const [step, setStep] = useState(0);
 const dock = useRef<HTMLElement>(null);
 const audio = useRef<HTMLAudioElement>(null);
 const [title, body, ...targets] = slides[step];
 const source = guideAudio[section]?.[step];

 useEffect(() => {
  const player = audio.current;
  if (!source || !player) return;

  let active = true;
  function stopWaiting() {
   document.removeEventListener('click', resume, true);
   document.removeEventListener('keydown', resume, true);
  }
  function resume(event: Event) {
   if (!event.isTrusted) return;
   stopWaiting();
   void player?.play().catch(() => {});
  }
  void player.play().catch(error => {
   if (active && error.name === 'NotAllowedError') {
    document.addEventListener('click', resume, true);
    document.addEventListener('keydown', resume, true);
   }
  });
  return () => {
   active = false;
   stopWaiting();
   player.pause();
  };
 }, [source, step]);

 useEffect(() => {
  let highlighted: HTMLElement | null = null;
  const observer = new MutationObserver(highlight);

  function highlight() {
   const matches = targets.map(target => document.querySelector<HTMLElement>(`[data-guide="${target}"]`));
   const target = matches.find(Boolean) ?? null;
   if (target === highlighted) return;
   highlighted?.removeAttribute('data-guide-highlight');
   highlighted = target;
   if (!target) return;

   target.setAttribute('data-guide-highlight', '');
   const rect = target.getBoundingClientRect();
   const panel = dock.current?.getBoundingClientRect();
   const bottom = panel && rect.left < panel.right && rect.right > panel.left
    ? panel.top : window.innerHeight;
   if (rect.top < 12 || (rect.bottom > bottom - 12 && rect.top > 80)) {
    target.scrollIntoView({block: 'start', behavior: 'instant'});
   }
  }

  observer.observe(document.body, {childList: true, subtree: true});
  highlight();
  return () => {
   observer.disconnect();
   highlighted?.removeAttribute('data-guide-highlight');
  };
 }, [slides, step]);

 const last = step === slides.length - 1;
 const pose = step === 0 && section === 'portal' ? guidePoses[0] : last ? guidePoses[2] : guidePoses[1];

 useEffect(() => {
  function nextOnBackground(event: MouseEvent) {
   if (event.defaultPrevented || event.button !== 0 || window.getSelection()?.toString()) return;
   if (!(event.target instanceof Element) || document.querySelector('dialog[open], .modal-backdrop')) return;
   if (!event.target.matches('html, body, #root, main.shell, .admin-main, .public-container, .workspace-content, .program-grid, .welcome')) return;
   if (last) onClose();
   else setStep(step + 1);
  }
  document.addEventListener('click', nextOnBackground);
  return () => document.removeEventListener('click', nextOnBackground);
 }, [last, onClose, step]);

 return <aside className="tour-guide" aria-label="الجولة التعليمية" ref={dock}>
  <div className="tour-guide-card">
   <div className="tour-guide-kicker"><span>الجولة التعليمية</span><bdi>{step + 1} / {slides.length}</bdi></div>
   <div aria-live="polite" aria-atomic="true"><h2>{title}</h2><p>{body}</p></div>
   {source && <audio key={source} ref={audio} src={source} hidden/>}
   <div className="tour-guide-actions">
    <button className="btn primary" onClick={last ? onClose : () => setStep(step + 1)}>{last ? 'إنهاء الجولة' : 'التالي'}</button>
    <button className="btn ghost" onClick={onClose}>تخطي الجولة</button>
    <button className="btn ghost" onClick={onSkipAll}>تخطي الكل</button>
   </div>
  </div>
  <svg className="tour-guide-person" viewBox={pose} preserveAspectRatio="xMinYMax meet" aria-hidden="true">
   <image href="/guide/host.png" width="1080" height="1920"/>
  </svg>
 </aside>;
}
