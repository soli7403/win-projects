// DariTerm 로고 마크: 두 화면(PC·서버)을 아치 다리가 잇는 모양. 앱 아이콘(build/appicon.png)과 같은 그림.
// 원본은 docs/brand/dariterm-icon.svg. 근거: spec/ui/tokens.md §5.
import { useId } from 'react';

export function LogoMark({ size = 18 }: { size?: number }) {
  // 한 화면에 여러 개가 있어도 그라디언트 id 가 겹치지 않게 한다.
  const gid = `dari-bg-${useId().replace(/:/g, '')}`;
  return (
    <svg className="logo-mark" width={size} height={size} viewBox="0 0 256 256" aria-hidden="true">
      <defs>
        <linearGradient id={gid} x1="0" y1="0" x2="1" y2="1">
          <stop offset="0" stopColor="#22417a" />
          <stop offset="1" stopColor="#0f1a33" />
        </linearGradient>
      </defs>
      <rect width="256" height="256" rx="56" fill={`url(#${gid})`} />
      <path d="M72 118 Q128 34 184 118" fill="none" stroke="#4f8cff" strokeWidth="18" strokeLinecap="round" />
      <rect x="28" y="116" width="90" height="80" rx="16" fill="#0f1a33" stroke="#ffffff" strokeWidth="13" />
      <path d="M50 140 L66 155 L50 170" fill="none" stroke="#7fb0ff" strokeWidth="10" strokeLinecap="round" strokeLinejoin="round" />
      <line x1="76" y1="172" x2="98" y2="172" stroke="#ffffff" strokeWidth="10" strokeLinecap="round" />
      <rect x="138" y="116" width="90" height="80" rx="16" fill="#0f1a33" stroke="#ffffff" strokeWidth="13" />
      <path d="M160 140 L176 155 L160 170" fill="none" stroke="#7fb0ff" strokeWidth="10" strokeLinecap="round" strokeLinejoin="round" />
      <line x1="186" y1="172" x2="208" y2="172" stroke="#ffffff" strokeWidth="10" strokeLinecap="round" />
    </svg>
  );
}
