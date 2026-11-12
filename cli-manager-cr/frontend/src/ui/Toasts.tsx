// 토스트 묶음. 근거: spec/ui/shell.md §5, spec/ui/tokens.md §3(토스트).
import { useApp } from '../state/app';

export function Toasts() {
  const toasts = useApp((s) => s.toasts);
  if (toasts.length === 0) return null;
  return (
    <div className="toasts" aria-live="polite">
      {toasts.map((t) => (
        <div key={t.id} className={`toast ${t.kind === 'error' ? 'error' : t.kind === 'ok' ? 'ok' : ''}`}>
          {t.text}
        </div>
      ))}
    </div>
  );
}
