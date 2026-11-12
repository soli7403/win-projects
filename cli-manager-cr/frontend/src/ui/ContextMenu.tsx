// 컨텍스트 메뉴 호스트. 근거: spec/ui/dialogs.md §1.3.
// 화면 밖으로 넘치면 가장자리에서 8px 안쪽으로 당기고, 바깥 누름·창 포커스 잃음·Esc·항목 선택으로 닫는다.
import { useEffect, useLayoutEffect, useRef, useState } from 'react';
import { useUi } from '../state/ui';

export function ContextMenuHost() {
  const menu = useUi((s) => s.menu);
  const close = useUi((s) => s.closeMenu);
  const ref = useRef<HTMLDivElement>(null);
  const [pos, setPos] = useState<{ left: number; top: number } | null>(null);

  useLayoutEffect(() => {
    if (!menu || !ref.current) {
      setPos(null);
      return;
    }
    const { offsetWidth: w, offsetHeight: h } = ref.current;
    const left = Math.max(8, Math.min(menu.x, window.innerWidth - w - 8));
    const top = Math.max(8, Math.min(menu.y, window.innerHeight - h - 8));
    setPos({ left, top });
  }, [menu]);

  useEffect(() => {
    if (!menu) return;
    const onDown = (e: MouseEvent) => {
      if (ref.current && !ref.current.contains(e.target as Node)) close();
    };
    const onKey = (e: KeyboardEvent) => {
      if (e.key === 'Escape') {
        e.stopPropagation();
        close();
      }
    };
    const onBlur = () => close();
    document.addEventListener('mousedown', onDown, true);
    window.addEventListener('keydown', onKey, true);
    window.addEventListener('blur', onBlur);
    return () => {
      document.removeEventListener('mousedown', onDown, true);
      window.removeEventListener('keydown', onKey, true);
      window.removeEventListener('blur', onBlur);
    };
  }, [menu, close]);

  if (!menu) return null;
  return (
    <div
      ref={ref}
      className="ctx-menu"
      role="menu"
      style={{ left: pos?.left ?? menu.x, top: pos?.top ?? menu.y, visibility: pos ? 'visible' : 'hidden' }}
      onContextMenu={(e) => e.preventDefault()}
    >
      {menu.items.map((it, i) =>
        it.separator ? (
          <div key={`sep-${i}`} className="ctx-sep" />
        ) : (
          <button
            key={`${it.label}-${i}`}
            role="menuitem"
            className={`ctx-item${it.danger ? ' danger' : ''}`}
            disabled={it.disabled}
            onClick={() => {
              close();
              it.onSelect?.();
            }}
          >
            <span>{it.label}</span>
            {it.shortcut && <span className="kbd">{it.shortcut}</span>}
          </button>
        ),
      )}
    </div>
  );
}
