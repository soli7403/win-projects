// 새로 만든 단순 로고 마크(프롬프트 꺾쇠 + 커서 막대). 원본 로고를 쓰지 않는다. 근거: spec/ui/tokens.md §5.
export function LogoMark({ size = 18 }: { size?: number }) {
  return (
    <svg className="logo-mark" width={size} height={size} viewBox="0 0 32 32" aria-hidden="true">
      <rect x="1" y="1" width="30" height="30" rx="8" fill="var(--accent)" />
      <path d="M9 11 L15 16 L9 21" fill="none" stroke="#fff" strokeWidth="2.6" strokeLinecap="round" strokeLinejoin="round" />
      <rect x="17" y="20" width="7" height="2.6" rx="1.3" fill="#fff" />
    </svg>
  );
}
