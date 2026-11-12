// 오류 경계 고정 테스트: 자식이 throw 해도 안내가 보이고, 대화상자 경계는 나머지를 살린다.
import { describe, it, expect, vi, beforeEach, afterEach } from 'vitest';
import { render, fireEvent, screen } from '@testing-library/react';
import { ErrorBoundary } from './ErrorBoundary';

function Boom({ msg }: { msg: string }): never {
  throw new Error(msg);
}

describe('ErrorBoundary', () => {
  beforeEach(() => {
    // React 가 오류를 콘솔에 남기므로 테스트 출력을 조용히 한다.
    vi.spyOn(console, 'error').mockImplementation(() => undefined);
  });
  afterEach(() => {
    vi.restoreAllMocks();
  });

  it('페이지 경계: 자식 오류 시 안내·메시지·다시 불러오기를 보인다', () => {
    render(
      <ErrorBoundary>
        <Boom msg="Cannot read properties of undefined (reading 'toUpperCase')" />
      </ErrorBoundary>,
    );
    expect(screen.getByText('화면 오류가 발생했습니다')).toBeTruthy();
    expect(screen.getByText(/Cannot read properties of undefined/)).toBeTruthy();
    expect(screen.getByRole('button', { name: '다시 불러오기' })).toBeTruthy();
  });

  it('오류가 없으면 자식을 그대로 보인다', () => {
    render(
      <ErrorBoundary>
        <p>정상 화면</p>
      </ErrorBoundary>,
    );
    expect(screen.getByText('정상 화면')).toBeTruthy();
    expect(screen.queryByText('화면 오류가 발생했습니다')).toBeNull();
  });

  it('대화상자 경계: 안내를 보이고 [닫기] 로 onDismiss 를 부른다', () => {
    const onDismiss = vi.fn();
    render(
      <ErrorBoundary variant="dialog" label="설정" onDismiss={onDismiss}>
        <Boom msg="설정 렌더링 실패" />
      </ErrorBoundary>,
    );
    expect(screen.getByText('설정 오류')).toBeTruthy();
    expect(screen.getByText('설정 렌더링 실패')).toBeTruthy();
    fireEvent.click(screen.getByRole('button', { name: '닫기' }));
    expect(onDismiss).toHaveBeenCalledTimes(1);
  });

  it('대화상자 경계는 형제 트리를 지우지 않는다', () => {
    render(
      <div>
        <p>다른 화면</p>
        <ErrorBoundary variant="dialog" label="서버 가져오기">
          <Boom msg="가져오기 실패" />
        </ErrorBoundary>
      </div>,
    );
    expect(screen.getByText('다른 화면')).toBeTruthy();
    expect(screen.getByText('가져오기 실패')).toBeTruthy();
  });
});
