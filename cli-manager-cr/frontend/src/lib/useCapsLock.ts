// Caps Lock 이 켜져 있는지 알려 주는 훅. 비밀번호 입력 화면에서 쓴다 (D97).
// 브라우저는 Caps Lock 상태를 따로 물어볼 방법이 없고, 키·마우스 이벤트에 실려 올 때만 알 수 있다.
// 그래서 키를 누르고 뗄 때와 마우스를 움직이거나 누를 때 상태를 읽는다(첫 이벤트 전에는 "꺼짐"으로 본다).
import { useEffect, useState } from 'react';

type ModifierEvent = { getModifierState?: (key: string) => boolean };

/** 이벤트에서 Caps Lock 상태를 읽는다. 알 수 없으면 null. */
export function capsLockOf(e: ModifierEvent): boolean | null {
  return typeof e.getModifierState === 'function' ? e.getModifierState('CapsLock') : null;
}

const EVENTS = ['keydown', 'keyup', 'mousedown', 'mousemove'] as const;

/** enabled 가 참인 동안 창의 키·마우스 이벤트로 Caps Lock 상태를 따라간다. */
export function useCapsLock(enabled = true): boolean {
  const [on, setOn] = useState(false);
  useEffect(() => {
    if (!enabled) {
      setOn(false);
      return;
    }
    const read = (e: Event) => {
      const state = capsLockOf(e as unknown as ModifierEvent);
      if (state !== null) setOn(state);
    };
    // 창을 벗어난 동안 바뀐 상태는 알 수 없으므로 모른다고 보고 끈다. 돌아와서 첫 이벤트에 다시 읽는다.
    const reset = () => setOn(false);
    for (const name of EVENTS) window.addEventListener(name, read, true);
    window.addEventListener('blur', reset);
    return () => {
      for (const name of EVENTS) window.removeEventListener(name, read, true);
      window.removeEventListener('blur', reset);
    };
  }, [enabled]);
  return on;
}
