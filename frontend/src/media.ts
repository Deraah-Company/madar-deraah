export const videoSources: Partial<Record<'login' | 'promo' | 'knowledge' | 'pulse' | 'family', string>> = {
 login: '/media/05_Madar_Deraah_Welcome_Promo_1080p.mp4',
 promo: '/media/02_Madar_Deraah_Portal_Intro_1080p.mp4',
 knowledge: '/media/03_Knowledge_Ambassadors_Guide_1080p.mp4',
 pulse: '/media/04_Deraah_Pulse_Guide_1080p.mp4',
};
const guideClipCounts: Record<string, number> = {
 portal: 3, settings: 1, knowledge: 3, pulse: 3, family: 2,
 'knowledge-campaigns': 2, 'knowledge-applications': 2, 'knowledge-sessions': 2,
 'knowledge-report': 1, 'knowledge-learn': 1, 'pulse-plans': 2, 'pulse-campaigns': 2, 'pulse-learn': 1,
};
export const guideAudio = Object.fromEntries(Object.entries(guideClipCounts).map(([section, count]) => [
 section,
 Array.from({length: count}, (_, index) => `/guide/audio/${section}-${String(index + 1).padStart(2, '0')}.mp3`),
]));

export function visitGuide(section: string) {
 window.dispatchEvent(new CustomEvent('deraah-guide', {detail: section}));
}
