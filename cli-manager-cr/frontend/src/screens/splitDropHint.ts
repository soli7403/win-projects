// 탭을 끄는 동안 작업 영역 오른쪽 절반에 보이는 "여기에 놓으면 좌우로 나눕니다" 안내의 표시 상태.
// 근거: spec/ui/shell.md §2.3(나누기), spec/acceptance/02-shell-sidebar.md G8.
import { useEffect, useState, type DragEvent } from 'react';

/** 안내를 켜는 오른쪽 가장자리 띠의 너비: 영역 너비의 이 비율, 단 아래 최소값(px)보다 좁지 않게 (D102). */
export const SPLIT_EDGE_RATIO = 0.1;
export const SPLIT_EDGE_MIN_PX = 100;

export interface SplitDropHint {
  /** 안내를 보일지 */
  show: boolean;
  hide: () => void;
  /** 작업 영역(안내가 덮는 영역 전체)에 다는 처리기 */
  zone: {
    onDragOver: (e: DragEvent<HTMLElement>) => void;
    onDragLeave: (e: DragEvent<HTMLElement>) => void;
    onDrop: () => void;
  };
}

/**
 * mime 을 가진 끌기가 작업 영역의 오른쪽 가장자리 띠에 들어오면 안내를 보인다.
 * 한번 보이면 포인터가 오른쪽 절반(안내 상자 자리)에 있는 동안 유지하고, 왼쪽 절반으로 가면 지운다.
 * ignore(CSS 선택자)에 맞는 요소 위에서는 켜지 않는다(탭 막대: 탭 순서 바꾸기를 가리지 않도록).
 * enabled 가 false 면(이미 나뉘어 있거나 탭이 1개) 보이지 않는다.
 *
 * 끌기는 놓기 말고도 여러 방식으로 끝난다(Esc 취소, 놓을 수 없는 곳·창 밖에서 놓기,
 * 전파를 막는 자식 위에 놓기). 어느 경우든 지워지도록 창에서 dragend·drop 을 직접 듣는다.
 */
export function useSplitDropHint(enabled: boolean, mime: string, ignore?: string): SplitDropHint {
  const [show, setShow] = useState(false);

  useEffect(() => {
    if (!show) return;
    let timer: ReturnType<typeof setTimeout> | undefined;
    // 캡처 단계라 자식이 전파를 막아도 받는다. 곧바로 지우면 안내 상자가 자기 drop 을
    // 처리하기 전에 화면에서 빠질 수 있어 이벤트 처리가 끝난 뒤로 미룬다.
    const clear = () => {
      clearTimeout(timer);
      timer = setTimeout(() => setShow(false), 0);
    };
    window.addEventListener('dragend', clear, true);
    window.addEventListener('drop', clear, true);
    return () => {
      clearTimeout(timer);
      window.removeEventListener('dragend', clear, true);
      window.removeEventListener('drop', clear, true);
    };
  }, [show]);

  return {
    show: show && enabled,
    hide: () => setShow(false),
    zone: {
      onDragOver: (e) => {
        if (!enabled || !Array.from(e.dataTransfer.types).includes(mime)) return;
        e.preventDefault();
        const r = e.currentTarget.getBoundingClientRect();
        if (show) {
          // 안내 상자는 오른쪽 절반을 덮는다. 그 밖(왼쪽 절반)으로 나가면 지운다.
          if (e.clientX < r.left + r.width / 2) setShow(false);
          return;
        }
        const edge = Math.max(SPLIT_EDGE_MIN_PX, r.width * SPLIT_EDGE_RATIO);
        if (e.clientX < r.right - edge) return;
        if (ignore && (e.target as Element).closest?.(ignore)) return;
        setShow(true);
      },
      // 자식 사이를 오갈 때도 dragleave 가 올라오므로, 포인터가 영역 밖으로 나갔을 때만 지운다.
      // relatedTarget 은 웹뷰 엔진에 따라 비어 있어 좌표로 판단한다.
      onDragLeave: (e) => {
        const r = e.currentTarget.getBoundingClientRect();
        if (e.clientX <= r.left || e.clientX >= r.right || e.clientY <= r.top || e.clientY >= r.bottom) setShow(false);
      },
      onDrop: () => setShow(false),
    },
  };
}
