// 공통 모달. 근거: spec/ui/dialogs.md §1. Esc·바깥 클릭으로 닫는다(카드 안 클릭은 닫지 않음).
import { useEffect, useRef, useState, type CSSProperties, type ReactNode } from 'react';
import type { LucideIcon } from 'lucide-react';
import { X } from 'lucide-react';
import { t } from '../i18n';
import { keyCapture } from '../lib/keys';

// Esc 로 닫히는 겹(모달, 도움말 그림 보기 등)의 쌓임. Esc 는 **맨 위 겹 하나만** 닫는다(D159).
// 전에는 모달마다 창 전체의 Esc 를 들어서, 겹쳐 있으면 Esc 한 번에 모두 닫혔다.
// "맨 위" 는 **처음 그려진 순서가 가장 늦은** 겹이다. effect 는 자식이 부모보다 먼저 돌아서, 부모와 자식이 한꺼번에
// 열리면 등록 순서가 뒤집힌다 — 그래서 처음 그릴 때(렌더 단계, 부모가 먼저) 번호를 받는다.
const openLayers = new Set<number>();
let layerSeq = 0;

/**
 * Esc 를 받는 겹으로 등록한다. 열려 있는 동안 맨 위 겹일 때만 onEsc 를 부른다.
 * onEsc 는 ref 로 들어 다시 그려져도 다시 등록하지 않는다(번호·순서가 그대로).
 */
export function useEscLayer(onEsc: () => void): void {
  const cb = useRef(onEsc);
  cb.current = onEsc;
  const [seq] = useState(() => ++layerSeq);
  useEffect(() => {
    openLayers.add(seq);
    const onKey = (e: KeyboardEvent) => {
      // 설정의 단축키 구역이 키를 받는 중이면 Esc 는 그쪽의 취소다 (D126).
      if (e.key !== 'Escape' || keyCapture.active) return;
      if (seq !== Math.max(...openLayers)) return;
      // Esc 는 맨 위 겹이 가로채고 아래로 전달하지 않는다.
      e.preventDefault();
      e.stopPropagation();
      cb.current();
    };
    window.addEventListener('keydown', onKey, true);
    return () => {
      window.removeEventListener('keydown', onKey, true);
      openLayers.delete(seq);
    };
  }, [seq]);
}

interface ModalProps {
  title: string;
  icon?: LucideIcon;
  iconColor?: string;
  size?: 'sm' | 'md' | 'lg';
  onClose: () => void;
  children: ReactNode;
  foot?: ReactNode;
  /** 닫기 X 버튼 숨김 여부(기본 보임). */
  hideClose?: boolean;
  /** 바깥 클릭으로 닫기 허용 여부(기본 허용). */
  dismissOnOverlay?: boolean;
  labelledBy?: string;
  /** 카드(.modal)에 더할 클래스(예: 도움말의 help-modal). */
  className?: string;
  /** 카드(.modal)의 인라인 스타일(예: 사용자가 바꾼 크기). */
  style?: CSSProperties;
}

export function Modal({
  title,
  icon: Icon,
  iconColor,
  size = 'md',
  onClose,
  children,
  foot,
  hideClose,
  dismissOnOverlay = true,
  labelledBy,
  className,
  style,
}: ModalProps) {
  const prevFocus = useRef<Element | null>(null);
  useEscLayer(onClose);

  useEffect(() => {
    prevFocus.current = document.activeElement;
    return () => {
      const el = prevFocus.current as HTMLElement | null;
      if (el && typeof el.focus === 'function' && document.contains(el)) el.focus();
    };
  }, []);

  const cls = (size === 'sm' ? 'modal sm' : size === 'lg' ? 'modal lg' : 'modal') + (className ? ` ${className}` : '');
  return (
    <div
      className="overlay"
      onMouseDown={(e) => {
        if (dismissOnOverlay && e.target === e.currentTarget) onClose();
      }}
    >
      <div className={cls} style={style} role="dialog" aria-modal="true" aria-labelledby={labelledBy} aria-label={labelledBy ? undefined : title}>
        <div className="modal-head">
          {Icon && <Icon size={18} color={iconColor} />}
          <span className="title" id={labelledBy}>{title}</span>
          {!hideClose && (
            <button className="icon-btn" onClick={onClose} aria-label={t('common.close')} title={t('common.close')}>
              <X size={16} />
            </button>
          )}
        </div>
        <div className="modal-body">{children}</div>
        {foot && <div className="modal-foot">{foot}</div>}
      </div>
    </div>
  );
}
