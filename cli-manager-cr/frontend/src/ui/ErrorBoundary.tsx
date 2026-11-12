// 렌더링 오류 경계. 자식이 던진 오류를 잡아 화면 전체가 사라지지 않게 한다.
// - page: 앱 최상위. "화면 오류가 발생했습니다" + 오류 메시지 + [다시 불러오기].
// - dialog: 대화상자 하나. 그 대화상자만 안내로 바꾸고 [닫기](onDismiss) 로 닫으면 나머지 앱은 그대로 살아 있다.
import { Component, type ErrorInfo, type ReactNode } from 'react';

interface Props {
  children: ReactNode;
  variant?: 'page' | 'dialog';
  /** dialog 변형에서 보일 이름(예: "설정"). */
  label?: string;
  /** dialog 변형의 [닫기] 동작(보통 그 대화상자를 닫는다). */
  onDismiss?: () => void;
}

interface State {
  error: Error | null;
}

export function reloadApp(): void {
  window.location.reload();
}

export class ErrorBoundary extends Component<Props, State> {
  state: State = { error: null };

  static getDerivedStateFromError(error: Error): State {
    return { error };
  }

  componentDidCatch(error: Error, info: ErrorInfo): void {
    // 진단용 기록. 화면에는 메시지만 보인다.
    console.error('[화면 오류]', this.props.label ?? 'app', error, info.componentStack);
  }

  render(): ReactNode {
    const { error } = this.state;
    if (!error) return this.props.children;
    const message = error.message || String(error);
    const variant = this.props.variant ?? 'page';

    if (variant === 'dialog') {
      return (
        <div className="overlay" role="alertdialog" aria-label="대화상자 오류">
          <div className="modal sm" style={{ padding: 0 }}>
            <div className="modal-head">
              <span className="title">{this.props.label ? `${this.props.label} 오류` : '대화상자 오류'}</span>
            </div>
            <div className="modal-body">
              <div>이 화면에서 오류가 발생했습니다. 다른 기능은 그대로 쓸 수 있습니다.</div>
              <div className="faint selectable" style={{ fontSize: 11.5, wordBreak: 'break-all' }}>{message}</div>
            </div>
            <div className="modal-foot">
              <button className="btn" type="button" onClick={reloadApp}>다시 불러오기</button>
              <button className="btn primary" type="button" onClick={() => this.props.onDismiss?.()}>닫기</button>
            </div>
          </div>
        </div>
      );
    }

    return (
      <div
        role="alert"
        style={{
          minHeight: '100vh', display: 'flex', alignItems: 'center', justifyContent: 'center',
          background: 'var(--bg)', color: 'var(--fg)', padding: 16,
        }}
      >
        <div style={{ maxWidth: 480, display: 'flex', flexDirection: 'column', gap: 12, textAlign: 'center' }}>
          <div style={{ fontSize: 16, fontWeight: 700 }}>화면 오류가 발생했습니다</div>
          <div style={{ color: 'var(--fg-2)', lineHeight: 1.6 }}>
            화면을 그리는 중 문제가 생겼습니다. 다시 불러오면 대부분 해결됩니다.
          </div>
          <div className="faint selectable" style={{ fontSize: 11.5, wordBreak: 'break-all' }}>{message}</div>
          <div>
            <button className="btn primary" type="button" onClick={reloadApp}>다시 불러오기</button>
          </div>
        </div>
      </div>
    );
  }
}
