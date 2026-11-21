// 탭 끌기 화면 나누기 안내의 표시·지우기 (D101·D102, spec/ui/shell.md §2.3, 수용 G8·G8a).
// Shell 의 작업 영역과 같은 구조(영역 안에 탭 막대·전파를 막는 탭·내용·안내 상자)를 작은 화면으로 만든다.
// 영역은 (100,50)~(500,350), 너비 400 이라 가장자리 띠는 100px(x ≥ 400), 가운데는 x = 300 이다.
import { afterEach, describe, expect, it, vi } from 'vitest';
import { act, cleanup, createEvent, fireEvent, render, screen } from '@testing-library/react';
import { useSplitDropHint } from './splitDropHint';

const MIME = 'application/x-dariterm-tab';

type Rect = { left: number; top: number; width: number; height: number };
const AREA: Rect = { left: 100, top: 50, width: 400, height: 300 };

function Harness({ enabled = true, onSplit }: { enabled?: boolean; onSplit?: (id: string) => void }) {
  const hint = useSplitDropHint(enabled, MIME, '.tabbar');
  return (
    <main data-testid="zone" {...hint.zone}>
      <div className="tabbar">
        <div
          data-testid="tab"
          onDragOver={(e) => e.preventDefault()}
          onDrop={(e) => {
            // TabBar 의 탭처럼 놓기 전파를 막는다.
            e.preventDefault();
            e.stopPropagation();
          }}
        >
          tab
        </div>
      </div>
      <div data-testid="content">content</div>
      {hint.show && (
        <div
          data-testid="hint"
          onDragOver={(e) => e.preventDefault()}
          onDrop={(e) => {
            const id = e.dataTransfer.getData(MIME);
            hint.hide();
            if (id) onSplit?.(id);
          }}
        >
          hint
        </div>
      )}
    </main>
  );
}

function setup(props: { enabled?: boolean; onSplit?: (id: string) => void } = {}, rect: Rect = AREA) {
  render(<Harness {...props} />);
  const zone = screen.getByTestId('zone');
  zone.getBoundingClientRect = () =>
    ({
      left: rect.left,
      top: rect.top,
      width: rect.width,
      height: rect.height,
      right: rect.left + rect.width,
      bottom: rect.top + rect.height,
      x: rect.left,
      y: rect.top,
      toJSON: () => ({}),
    }) as DOMRect;
  return zone;
}

function transfer(types: string[] = [MIME], id = 'tab-1') {
  return { types, getData: (k: string) => (k === MIME ? id : '') };
}

/** jsdom 은 끌기 이벤트에 좌표를 넣어 주지 않아 직접 붙인다. */
function drag(
  kind: 'dragOver' | 'dragLeave' | 'drop' | 'dragEnd',
  el: Element,
  x = 0,
  y = 200,
  types: string[] = [MIME],
) {
  const ev = createEvent[kind](el, { dataTransfer: transfer(types) });
  Object.defineProperty(ev, 'clientX', { value: x });
  Object.defineProperty(ev, 'clientY', { value: y });
  fireEvent(el, ev);
}

const shown = () => screen.queryByTestId('hint') !== null;
const content = () => screen.getByTestId('content');
/** 창 처리기가 미룬 지우기(setTimeout 0)를 실행한다. */
const flush = () => act(() => new Promise<void>((r) => setTimeout(r, 0)));

afterEach(cleanup);

describe('켜기 (D102)', () => {
  it('오른쪽 가장자리 띠에 들어가면 보인다', () => {
    setup();
    drag('dragOver', content(), 450);
    expect(shown()).toBe(true);
  });

  it('띠의 경계(오른쪽 끝 − 100px)에서부터 보인다', () => {
    setup();
    drag('dragOver', content(), 399);
    expect(shown()).toBe(false);
    drag('dragOver', content(), 400);
    expect(shown()).toBe(true);
  });

  it('왼쪽 절반이나 오른쪽 절반의 띠 밖에서는 보이지 않는다', () => {
    setup();
    drag('dragOver', content(), 200);
    drag('dragOver', content(), 350);
    expect(shown()).toBe(false);
  });

  it('영역이 넓으면 띠는 너비의 10% 다', () => {
    setup({}, { left: 0, top: 50, width: 2000, height: 300 });
    drag('dragOver', content(), 1799);
    expect(shown()).toBe(false);
    drag('dragOver', content(), 1800);
    expect(shown()).toBe(true);
  });

  it('탭 막대 위에서는 가장자리라도 보이지 않는다', () => {
    setup();
    drag('dragOver', screen.getByTestId('tab'), 480);
    expect(shown()).toBe(false);
  });

  it('보인 뒤에는 오른쪽 절반 안에서 띠 밖으로 나와도 유지된다', () => {
    setup();
    drag('dragOver', content(), 450);
    drag('dragOver', screen.getByTestId('hint'), 350);
    drag('dragOver', screen.getByTestId('hint'), 300);
    expect(shown()).toBe(true);
  });

  it('보인 뒤 왼쪽 절반으로 가면 지워지고, 다시 띠에 들어가면 보인다', () => {
    setup();
    drag('dragOver', content(), 450);
    drag('dragOver', content(), 250);
    expect(shown()).toBe(false);
    drag('dragOver', content(), 350);
    expect(shown()).toBe(false);
    drag('dragOver', content(), 450);
    expect(shown()).toBe(true);
  });

  it('탭이 아닌 것을 끌면 보이지 않는다', () => {
    setup();
    drag('dragOver', content(), 450, 200, ['Files']);
    expect(shown()).toBe(false);
  });

  it('나눌 수 없는 상태(enabled=false)면 보이지 않는다', () => {
    setup({ enabled: false });
    drag('dragOver', content(), 450);
    expect(shown()).toBe(false);
  });
});

describe('지우기 (D101)', () => {
  it('끌기를 취소하면(dragend 만 온다) 지워진다', async () => {
    setup();
    drag('dragOver', content(), 450);
    drag('dragEnd', screen.getByTestId('tab'));
    await flush();
    expect(shown()).toBe(false);
  });

  it('전파를 막는 자식(탭) 위에 놓아도 지워진다', async () => {
    setup();
    drag('dragOver', content(), 450);
    drag('drop', screen.getByTestId('tab'), 150, 60);
    await flush();
    expect(shown()).toBe(false);
  });

  it('영역 밖으로 나가면 지워진다 (자식에서 난 dragleave 라도)', () => {
    setup();
    drag('dragOver', content(), 450);
    drag('dragLeave', content(), 520, 200);
    expect(shown()).toBe(false);
  });

  it('창 밖으로 나가면(좌표 0,0) 지워진다', () => {
    setup();
    drag('dragOver', content(), 450);
    drag('dragLeave', screen.getByTestId('hint'), 0, 0);
    expect(shown()).toBe(false);
  });

  it('영역 안에서 자식 사이를 오가는 dragleave 로는 지워지지 않는다', () => {
    setup();
    drag('dragOver', content(), 450);
    drag('dragLeave', content(), 420, 200);
    drag('dragLeave', screen.getByTestId('hint'), 300, 200);
    expect(shown()).toBe(true);
  });

  it('안내 상자에 놓으면 나누기가 실행되고 지워진다', async () => {
    const onSplit = vi.fn();
    setup({ onSplit });
    drag('dragOver', content(), 450);
    drag('drop', screen.getByTestId('hint'), 450);
    expect(onSplit).toHaveBeenCalledWith('tab-1');
    await flush();
    expect(shown()).toBe(false);
  });

  it('지워진 뒤 다시 끌면 다시 보이고, 다시 지워진다', async () => {
    setup();
    drag('dragOver', content(), 450);
    drag('dragEnd', screen.getByTestId('tab'));
    await flush();
    expect(shown()).toBe(false);
    drag('dragOver', content(), 450);
    expect(shown()).toBe(true);
    drag('dragEnd', screen.getByTestId('tab'));
    await flush();
    expect(shown()).toBe(false);
  });
});
