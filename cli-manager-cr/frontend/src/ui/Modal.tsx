// 공통 모달. 근거: spec/ui/dialogs.md §1. Esc·바깥 클릭으로 닫는다(카드 안 클릭은 닫지 않음).
import { useEffect, useRef, type ReactNode } from 'react';
import type { LucideIcon } from 'lucide-react';
import { X } from 'lucide-react';
import { t } from '../i18n';

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
}: ModalProps) {
  const prevFocus = useRef<Element | null>(null);

  useEffect(() => {
    prevFocus.current = document.activeElement;
    const onKey = (e: KeyboardEvent) => {
      if (e.key === 'Escape') {
        // Esc 는 모달이 먼저 가로채고 아래로 전달하지 않는다.
        e.preventDefault();
        e.stopPropagation();
        onClose();
      }
    };
    window.addEventListener('keydown', onKey, true);
    return () => {
      window.removeEventListener('keydown', onKey, true);
      const el = prevFocus.current as HTMLElement | null;
      if (el && typeof el.focus === 'function' && document.contains(el)) el.focus();
    };
  }, [onClose]);

  const cls = size === 'sm' ? 'modal sm' : size === 'lg' ? 'modal lg' : 'modal';
  return (
    <div
      className="overlay"
      onMouseDown={(e) => {
        if (dismissOnOverlay && e.target === e.currentTarget) onClose();
      }}
    >
      <div className={cls} role="dialog" aria-modal="true" aria-labelledby={labelledBy} aria-label={labelledBy ? undefined : title}>
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
