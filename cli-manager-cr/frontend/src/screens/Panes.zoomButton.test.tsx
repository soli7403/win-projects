// 창 머리글의 크게 보기/원래대로 버튼: 두 상태의 아이콘 윤곽이 다르고, 크게 보는 중에는 켜진 모양이다 (D121, spec/ui/panes.md §2.1 7번).
import { describe, expect, it, vi } from 'vitest';
import { fireEvent, render, screen } from '@testing-library/react';
import { PaneZoomButton } from './Panes';

/** 버튼 안 아이콘이 그리는 도형의 종류와 개수(윤곽 비교용). */
function shapes(btn: HTMLElement): string {
  const svg = btn.querySelector('svg')!;
  return [...svg.children].map((c) => c.tagName.toLowerCase()).sort().join(',');
}

describe('PaneZoomButton (D121)', () => {
  it('평소: 빈 사각형 하나, 꺼진 모양, 툴팁 "최대화"', () => {
    render(<PaneZoomButton zoomed={false} onToggle={() => undefined} />);
    const btn = screen.getByRole('button', { name: '최대화' });
    expect(btn.className).not.toContain(' on');
    expect(btn.getAttribute('aria-pressed')).toBe('false');
    expect(btn.getAttribute('title')).toBe('최대화 (Ctrl+Shift+Enter)');
    expect(btn.querySelector('[data-icon="maximize"]')).toBeTruthy();
    expect(shapes(btn)).toBe('rect');
  });

  it('크게 보는 중: 겹친 사각형 두 개, 켜진 모양, 툴팁 "복원"', () => {
    render(<PaneZoomButton zoomed onToggle={() => undefined} />);
    const btn = screen.getByRole('button', { name: '최대화' });
    expect(btn.className).toContain(' on');
    expect(btn.getAttribute('aria-pressed')).toBe('true');
    expect(btn.getAttribute('title')).toBe('복원 (Ctrl+Shift+Enter)');
    expect(btn.querySelector('[data-icon="restore"]')).toBeTruthy();
    expect(shapes(btn)).toBe('path,rect');
  });

  it('두 상태의 아이콘은 서로 다른 도형이다', () => {
    const a = render(<PaneZoomButton zoomed={false} onToggle={() => undefined} />);
    const off = a.container.querySelector('svg')!.innerHTML;
    a.unmount();
    const b = render(<PaneZoomButton zoomed onToggle={() => undefined} />);
    const on = b.container.querySelector('svg')!.innerHTML;
    expect(on).not.toBe(off);
  });

  it('누르면 onToggle 을 부른다', () => {
    const onToggle = vi.fn();
    render(<PaneZoomButton zoomed={false} onToggle={onToggle} />);
    fireEvent.click(screen.getByRole('button', { name: '최대화' }));
    expect(onToggle).toHaveBeenCalledTimes(1);
  });
});
