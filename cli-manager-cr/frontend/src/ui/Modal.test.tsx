// 공통 모달: Esc 와 바깥 클릭으로 닫히고, 카드 안 클릭은 닫지 않는다(spec/ui/dialogs.md §1).
import { describe, expect, it, vi } from 'vitest';
import { fireEvent, render, screen } from '@testing-library/react';
import { Modal } from './Modal';

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
});
