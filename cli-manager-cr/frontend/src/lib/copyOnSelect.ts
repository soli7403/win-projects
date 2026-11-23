// "선택하면 자동 복사"(spec/ui/settings.md: 드래그로 선택하면 즉시 클립보드에 복사).
// 사용자가 마우스로 선택을 끝냈을 때(왼쪽 버튼을 뗄 때) 한 번만 복사한다. 앱이 코드로 바꾼 선택
// (찾기의 현재 항목 등)은 복사하지 않는다 — KI-011: 찾기가 선택을 바꿀 때마다 클립보드가 덮이던 문제.
import type { IDisposable } from '@xterm/xterm';

export interface SelectionSource {
  hasSelection(): boolean;
  getSelection(): string;
}

/**
 * el 안에서 왼쪽 버튼을 누른 뒤 (어디서든) 뗐을 때 선택이 있으면 copy 를 부른다.
 * enabled() 가 거짓이면 복사하지 않는다(설정을 그때그때 읽는다).
 */
export function attachCopyOnMouseSelect(
  el: HTMLElement,
  term: SelectionSource,
  enabled: () => boolean,
  copy: () => void,
  win: Pick<Window, 'addEventListener' | 'removeEventListener'> = window,
): IDisposable {
  let pressed = false;
  const onDown = (ev: MouseEvent) => {
    if (ev.button === 0) pressed = true;
  };
  const onUp = (ev: MouseEvent) => {
    if (ev.button !== 0 || !pressed) return;
    pressed = false;
    // xterm 이 같은 mouseup 으로 선택을 마무리한 뒤에 읽는다.
    setTimeout(() => {
      if (enabled() && term.hasSelection() && term.getSelection()) copy();
    }, 0);
  };
  el.addEventListener('mousedown', onDown, true);
  win.addEventListener('mouseup', onUp, true);
  return {
    dispose() {
      el.removeEventListener('mousedown', onDown, true);
      win.removeEventListener('mouseup', onUp, true);
    },
  };
}
