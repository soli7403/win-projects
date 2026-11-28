// 공통 모달: Esc 와 바깥 클릭으로 닫히고, 카드 안 클릭은 닫지 않는다(spec/ui/dialogs.md §1).
// 겹쳐 있으면 Esc 는 맨 위 겹만 닫는다(D159).
import { afterEach, describe, expect, it, vi } from 'vitest';
import { cleanup, fireEvent, render, screen } from '@testing-library/react';
import { useState } from 'react';
import { Modal, useEscLayer } from './Modal';

afterEach(() => cleanup());

describe('공통 모달', () => {
  it('Esc 로 닫힌다', () => {
    const onClose = vi.fn();
    render(<Modal title="테스트" onClose={onClose}><p>본문</p></Modal>);
    fireEvent.keyDown(window, { key: 'Escape' });
    expect(onClose).toHaveBeenCalledTimes(1);
  });

  it('바깥(오버레이) 클릭으로 닫히고 카드 안 클릭은 닫지 않는다', () => {
    const onClose = vi.fn();
    const { container } = render(<Modal title="테스트" onClose={onClose}><p>본문</p></Modal>);
    fireEvent.mouseDown(screen.getByText('본문'));
    expect(onClose).not.toHaveBeenCalled();
    fireEvent.mouseDown(container.querySelector('.overlay')!);
    expect(onClose).toHaveBeenCalledTimes(1);
  });

  it('className·style 을 카드에 붙인다', () => {
    render(<Modal title="테스트" onClose={() => undefined} className="help-modal" style={{ width: 700 }}><p>본문</p></Modal>);
    const card = screen.getByRole('dialog');
    expect(card.className).toBe('modal help-modal');
    expect(card.style.width).toBe('700px');
  });
});

describe('Esc 쌓임 (D159)', () => {
  function Layer({ onEsc }: { onEsc: () => void }) {
    useEscLayer(onEsc);
    return null;
  }

  it('겹쳐 있으면 Esc 는 맨 위 겹만 닫고, 그 겹이 닫히면 다음 Esc 가 아래 모달을 닫는다', () => {
    const outer = vi.fn();
    const inner = vi.fn();
    function Both() {
      const [top, setTop] = useState(true);
      return (
        <Modal title="바깥" onClose={outer}>
          {top && <Layer onEsc={() => { inner(); setTop(false); }} />}
        </Modal>
      );
    }
    render(<Both />);
    fireEvent.keyDown(window, { key: 'Escape' });
    expect(inner).toHaveBeenCalledTimes(1);
    expect(outer).not.toHaveBeenCalled();
    fireEvent.keyDown(window, { key: 'Escape' });
    expect(outer).toHaveBeenCalledTimes(1);
    expect(inner).toHaveBeenCalledTimes(1);
  });

  it('다시 그려져 onClose 가 바뀌어도 쌓인 순서는 그대로다', () => {
    const outer = vi.fn();
    const inner = vi.fn();
    function Both({ n }: { n: number }) {
      // 바깥 모달의 onClose 를 그릴 때마다 새 함수로 준다
      return (
        <Modal title={`바깥 ${n}`} onClose={() => outer(n)}>
          <Layer onEsc={inner} />
        </Modal>
      );
    }
    const { rerender } = render(<Both n={1} />);
    rerender(<Both n={2} />);
    fireEvent.keyDown(window, { key: 'Escape' });
    expect(inner).toHaveBeenCalledTimes(1);
    expect(outer).not.toHaveBeenCalled();
  });
});
