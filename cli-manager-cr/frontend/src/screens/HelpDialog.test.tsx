// 도움말 창(D155): 왼쪽 위 버튼으로 열기, 검색과 강조, 설정에서 열고 닫으면 설정으로 돌아가기.
import { afterEach, beforeEach, describe, expect, it } from 'vitest';
import { cleanup, fireEvent, render, screen } from '@testing-library/react';
import { HelpDialog } from './HelpDialog';
import { useUi } from '../state/ui';
import { useLang } from '../i18n';

beforeEach(() => {
  localStorage.removeItem('dariterm.helpSort');
  localStorage.removeItem('dariterm.helpSize');
  window.innerWidth = 1024;
  window.innerHeight = 768;
  useLang.getState().setLang('ko');
  useUi.setState({ help: { open: true, topic: null, returnTo: null }, settings: { open: false, section: 'appearance' } });
});

afterEach(() => cleanup());

describe('도움말 창', () => {
  it('처음에는 첫 항목을 보이고, 목록에서 고르면 그 항목을 보인다', () => {
    render(<HelpDialog />);
    expect(screen.getByRole('article', { name: '처음 실행' })).toBeTruthy();
    fireEvent.click(screen.getByRole('button', { name: /^매크로$/ }));
    expect(screen.getByRole('article', { name: '매크로' })).toBeTruthy();
  });

  it('검색하면 맞은 항목만 목록에 남고 맞은 개수와 강조가 보인다. Enter 는 첫 결과를 연다', () => {
    const { container } = render(<HelpDialog />);
    const box = screen.getByRole('textbox', { name: /도움말 검색/ });
    fireEvent.change(box, { target: { value: '프록시 명령' } });
    expect(screen.getByRole('status').textContent).toMatch(/개 항목/);
    fireEvent.keyDown(box, { key: 'Enter' });
    expect(screen.getByRole('article', { name: '프록시와 프록시 명령' })).toBeTruthy();
    expect(container.querySelectorAll('.help-body mark').length).toBeGreaterThan(0);
    fireEvent.change(box, { target: { value: '없는낱말xyz' } });
    expect(screen.getByRole('status').textContent).toContain('없는낱말xyz');
  });

  it('항목 id 를 주고 열면 그 항목부터 보인다', () => {
    useUi.setState({ help: { open: true, topic: 'login-script', returnTo: null } });
    render(<HelpDialog />);
    expect(screen.getByRole('article', { name: '로그인 스크립트' })).toBeTruthy();
  });
});

describe('도움말 창 — 정렬·질문 카드·그림 (D156~D158)', () => {
  const listTitles = () => [...document.querySelectorAll('.help-list .help-item-title')].map((x) => x.textContent);

  it('기본은 기능 순서, [가나다순]을 누르면 가나다순으로 바뀌고 다음에 열 때도 기억한다', () => {
    render(<HelpDialog />);
    expect(screen.getByRole('button', { name: '기능 순서' }).getAttribute('aria-pressed')).toBe('true');
    expect(listTitles()[0]).toBe('처음 실행');
    fireEvent.click(screen.getByRole('button', { name: '가나다순' }));
    expect(screen.getByRole('button', { name: '가나다순' }).getAttribute('aria-pressed')).toBe('true');
    const sorted = listTitles();
    expect(sorted[0]).toBe('개인용·업무용과 업데이트');
    expect(sorted.length).toBe(28);
    cleanup();
    render(<HelpDialog />);
    expect(listTitles()).toEqual(sorted);
    fireEvent.click(screen.getByRole('button', { name: '기능 순서' }));
    expect(listTitles()[0]).toBe('처음 실행');
  });

  it('검색 중에는 정렬 단추 대신 결과 개수가 보이고, 결과는 맞은 정도 순이다', () => {
    render(<HelpDialog />);
    fireEvent.change(screen.getByRole('textbox', { name: /도움말 검색/ }), { target: { value: '프록시 명령' } });
    expect(screen.queryByRole('group', { name: '목록 순서' })).toBeNull();
    expect(listTitles()[0]).toBe('프록시와 프록시 명령');
  });

  it('영어에서는 이름순 단추가 A–Z 이고 ABC 순으로 바뀐다', () => {
    useLang.getState().setLang('en');
    render(<HelpDialog />);
    fireEvent.click(screen.getByRole('button', { name: 'A–Z' }));
    const t = listTitles();
    expect(t[0]!.toLowerCase().startsWith('a')).toBe(true);
  });

  it('자주 묻는 질문은 질문마다 카드로 나뉜다', () => {
    useUi.setState({ help: { open: true, topic: 'faq', returnTo: null } });
    const { container } = render(<HelpDialog />);
    const cards = container.querySelectorAll('.help-body .help-qa');
    expect(cards.length).toBeGreaterThanOrEqual(9);
    expect(cards[0].querySelector('.help-q')!.textContent).toContain('연결이 자꾸 끊겨요');
    expect(cards[0].querySelector('.help-a')!.textContent).toContain('다시 연결');
  });

  it('그림이 있는 항목은 줄인 그림과 설명을 보인다', () => {
    useUi.setState({ help: { open: true, topic: 'proxy', returnTo: null } });
    const { container } = render(<HelpDialog />);
    const img = container.querySelector('.help-fig img') as HTMLImageElement;
    expect(img).toBeTruthy();
    expect(img.getAttribute('src')).toMatch(/ko\/proxy\.png/);
    expect(img.alt).toBe('서버 편집 > 고급 — 프록시 명령');
    expect(container.querySelector('.help-fig figcaption')!.textContent).toBe('서버 편집 > 고급 — 프록시 명령');
  });
});

describe('그림 보기 (D159)', () => {
  const ALT = '서버 편집 > 고급 — 프록시 명령';
  const openViewer = () => {
    useUi.setState({ help: { open: true, topic: 'proxy', returnTo: null } });
    const r = render(<HelpDialog />);
    fireEvent.click(screen.getByRole('button', { name: ALT }));
    return r;
  };
  const viewer = () => document.body.querySelector('.help-viewer') as HTMLElement | null;

  it('그림을 누르면 화면 전체 보기가 열리고, 처음에는 그림 전체가 보이게 맞춘다', () => {
    openViewer();
    expect(viewer()).toBeTruthy();
    expect(screen.getByRole('dialog', { name: ALT })).toBe(viewer());
    expect(viewer()!.querySelector('.help-viewer-stage')!.classList.contains('actual')).toBe(false);
    expect((viewer()!.querySelector('.help-viewer-stage img') as HTMLImageElement).getAttribute('src')).toMatch(/ko\/proxy\.png/);
  });

  it('[실제 크기] ↔ [화면에 맞추기], 그림을 눌러도 바뀐다', () => {
    openViewer();
    const stage = () => viewer()!.querySelector('.help-viewer-stage')!;
    fireEvent.click(screen.getByRole('button', { name: '실제 크기' }));
    expect(stage().classList.contains('actual')).toBe(true);
    fireEvent.click(screen.getByRole('button', { name: '화면에 맞추기' }));
    expect(stage().classList.contains('actual')).toBe(false);
    fireEvent.click(stage().querySelector('img')!);
    expect(stage().classList.contains('actual')).toBe(true);
  });

  it('Esc 는 그림 보기만 닫고 도움말은 그대로, 다음 Esc 가 도움말을 닫는다', () => {
    openViewer();
    fireEvent.keyDown(window, { key: 'Escape' });
    expect(viewer()).toBeNull();
    expect(useUi.getState().help.open).toBe(true);
    expect(screen.getByRole('article', { name: '프록시와 프록시 명령' })).toBeTruthy();
    fireEvent.keyDown(window, { key: 'Escape' });
    expect(useUi.getState().help.open).toBe(false);
  });

  it('X 와 바깥(어두운 곳) 누르기로 닫히고, 닫은 뒤 다시 열리지 않는다', () => {
    openViewer();
    fireEvent.click(viewer()!.querySelector('.help-viewer-bar .icon-btn')!);
    expect(viewer()).toBeNull();
    fireEvent.click(screen.getByRole('button', { name: ALT }));
    fireEvent.mouseDown(viewer()!);
    expect(viewer()).toBeNull();
    expect(useUi.getState().help.open).toBe(true);
  });
});

describe('도움말 창 크기 조절 (D159)', () => {
  afterEach(() => localStorage.removeItem('dariterm.helpSize'));
  const card = () => screen.getByRole('dialog', { name: '도움말' }) as HTMLElement;
  const handle = () => screen.getByRole('separator', { name: /창 크기 조절/ });

  it('처음에는 기본 크기(인라인 크기 없음)', () => {
    render(<HelpDialog />);
    expect(card().style.width).toBe('');
    expect(card().classList.contains('help-modal')).toBe(true);
  });

  it('손잡이를 끌면 크기가 바뀌고 놓을 때 저장하며, 다음에 열 때 그 크기로 연다', () => {
    window.innerWidth = 1600;
    window.innerHeight = 1000;
    render(<HelpDialog />);
    // jsdom 은 크기가 0 이라 카드 크기를 흉내 낸다
    card().getBoundingClientRect = () => ({ width: 820, height: 600, top: 0, left: 0, right: 820, bottom: 600, x: 0, y: 0, toJSON: () => ({}) });
    fireEvent.pointerDown(handle(), { clientX: 100, clientY: 100, pointerId: 1 });
    fireEvent.pointerMove(handle(), { clientX: 150, clientY: 140, pointerId: 1 });
    expect(card().style.width).toBe('920px'); // 가운데 정렬이라 2배
    expect(card().style.height).toBe('640px');
    expect(localStorage.getItem('dariterm.helpSize')).toBeNull(); // 끄는 동안은 저장하지 않는다
    fireEvent.pointerUp(handle(), { clientX: 150, clientY: 140, pointerId: 1 });
    expect(JSON.parse(localStorage.getItem('dariterm.helpSize')!)).toEqual({ w: 920, h: 640 });
    cleanup();
    render(<HelpDialog />);
    expect(card().style.width).toBe('920px');
  });

  it('최소 600×380 보다 작게는 줄지 않는다', () => {
    render(<HelpDialog />);
    card().getBoundingClientRect = () => ({ width: 820, height: 600, top: 0, left: 0, right: 820, bottom: 600, x: 0, y: 0, toJSON: () => ({}) });
    fireEvent.pointerDown(handle(), { clientX: 500, clientY: 500, pointerId: 1 });
    fireEvent.pointerMove(handle(), { clientX: 0, clientY: 0, pointerId: 1 });
    expect(card().style.width).toBe('600px');
    expect(card().style.height).toBe('380px');
  });

  it('두 번 누르면 기본 크기로 돌아가고 저장값을 지운다', () => {
    localStorage.setItem('dariterm.helpSize', JSON.stringify({ w: 900, h: 600 }));
    render(<HelpDialog />);
    expect(card().style.width).toBe('900px');
    fireEvent.doubleClick(handle());
    expect(card().style.width).toBe('');
    expect(localStorage.getItem('dariterm.helpSize')).toBeNull();
  });

  it('방향키로도 조절한다(20px, Shift 60px)', () => {
    localStorage.setItem('dariterm.helpSize', JSON.stringify({ w: 900, h: 600 }));
    window.innerWidth = 1600;
    window.innerHeight = 1000;
    render(<HelpDialog />);
    fireEvent.keyDown(handle(), { key: 'ArrowRight' });
    expect(card().style.width).toBe('920px');
    fireEvent.keyDown(handle(), { key: 'ArrowDown', shiftKey: true });
    expect(card().style.height).toBe('660px');
    expect(JSON.parse(localStorage.getItem('dariterm.helpSize')!)).toEqual({ w: 920, h: 660 });
  });
});

describe('설정과 도움말 오가기', () => {
  it('설정에서 연 도움말을 닫으면 그 설정 구역으로 돌아간다', () => {
    useUi.setState({ help: { open: false, topic: null, returnTo: null }, settings: { open: true, section: 'rules' } });
    useUi.getState().openHelp('triggers', 'rules');
    expect(useUi.getState().settings.open).toBe(false);
    expect(useUi.getState().help).toEqual({ open: true, topic: 'triggers', returnTo: 'rules' });
    useUi.getState().closeHelp();
    expect(useUi.getState().settings).toEqual({ open: true, section: 'rules' });
    expect(useUi.getState().help.open).toBe(false);
  });

  it('왼쪽 위에서 연 도움말을 닫으면 설정은 그대로다', () => {
    useUi.getState().openHelp();
    useUi.getState().closeHelp();
    expect(useUi.getState().settings.open).toBe(false);
  });
});
