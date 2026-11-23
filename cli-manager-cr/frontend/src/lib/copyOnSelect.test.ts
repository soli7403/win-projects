// "선택하면 자동 복사"는 마우스로 고른 선택만 복사한다. 근거: spec/ui/settings.md, KI-011.
import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest';
import { attachCopyOnMouseSelect } from './copyOnSelect';

function setup(enabled = true) {
  const el = document.createElement('div');
  document.body.appendChild(el);
  const sel = { text: '' };
  const term = { hasSelection: () => sel.text !== '', getSelection: () => sel.text };
  const copy = vi.fn();
  const sub = attachCopyOnMouseSelect(el, term, () => enabled, copy);
  const down = (target: EventTarget = el) => target.dispatchEvent(new MouseEvent('mousedown', { button: 0, bubbles: true }));
  const up = () => window.dispatchEvent(new MouseEvent('mouseup', { button: 0, bubbles: true }));
  return { el, sel, copy, sub, down, up };
}

beforeEach(() => vi.useFakeTimers());
afterEach(() => {
  vi.useRealTimers();
  document.body.innerHTML = '';
});

describe('선택하면 자동 복사', () => {
  it('터미널에서 끌어 선택하고 버튼을 떼면 한 번 복사한다', () => {
    const f = setup();
    f.down();
    f.sel.text = 'abc';
    f.up();
    vi.runAllTimers();
    expect(f.copy).toHaveBeenCalledTimes(1);
  });

  it('코드가 바꾼 선택(찾기의 현재 항목)은 복사하지 않는다', () => {
    const f = setup();
    f.sel.text = 'found';
    // 마우스 누름 없이 버튼 뗌만 와도(다른 곳 클릭의 끝) 복사하지 않는다.
    f.up();
    vi.runAllTimers();
    expect(f.copy).not.toHaveBeenCalled();
  });

  it('터미널 밖에서 누른 클릭은 세지 않고, 선택이 없으면 복사하지 않는다', () => {
    const f = setup();
    const other = document.createElement('div');
    document.body.appendChild(other);
    f.sel.text = 'x';
    f.down(other);
    f.up();
    f.down();
    f.sel.text = '';
    f.up();
    vi.runAllTimers();
    expect(f.copy).not.toHaveBeenCalled();
  });

  it('설정이 꺼져 있으면 복사하지 않고, dispose 뒤에는 듣지 않는다', () => {
    const off = setup(false);
    off.down();
    off.sel.text = 'abc';
    off.up();
    vi.runAllTimers();
    expect(off.copy).not.toHaveBeenCalled();

    const on = setup(true);
    on.sub.dispose();
    on.down();
    on.sel.text = 'abc';
    on.up();
    vi.runAllTimers();
    expect(on.copy).not.toHaveBeenCalled();
  });
});
