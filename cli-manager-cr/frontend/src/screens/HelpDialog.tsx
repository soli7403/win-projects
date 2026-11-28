// 도움말 창: 왼쪽은 검색 칸과 항목 목록, 오른쪽은 고른 항목의 설명. 검색어는 본문에서 강조한다.
// 근거: spec/ui/help.md (D155). 본문은 locales/help/<언어>.md(lib/help.ts).
import { Fragment, useMemo, useRef, useState, type KeyboardEvent as ReactKeyboardEvent, type MouseEvent as ReactMouseEvent, type PointerEvent as ReactPointerEvent, type ReactNode } from 'react';
import { createPortal } from 'react-dom';
import { CircleHelp, Search, X } from 'lucide-react';
import { Modal, useEscLayer } from '../ui/Modal';
import {
  clampHelpSize,
  helpFor,
  helpImage,
  loadHelpSize,
  loadHelpSort,
  markParts,
  saveHelpSize,
  saveHelpSort,
  searchHelp,
  searchTerms,
  sortTopics,
  type HelpBlock,
  type HelpSize,
  type HelpSort,
  type HelpTopic,
} from '../lib/help';
import { useUi } from '../state/ui';
import { t, useLang } from '../i18n';

/** 글자 안의 **굵게**·`코드` 를 그리고, 검색 낱말을 <mark> 로 감싼다. */
function Inline({ text, terms }: { text: string; terms: readonly string[] }) {
  const marked = (s: string) => markParts(s, terms).map(([part, hit], i) => (hit ? <mark key={i}>{part}</mark> : <Fragment key={i}>{part}</Fragment>));
  const out: ReactNode[] = [];
  text.split(/(\*\*[^*]+\*\*|`[^`]+`)/).forEach((seg, i) => {
    if (!seg) return;
    if (seg.startsWith('**') && seg.endsWith('**')) out.push(<b key={i}>{marked(seg.slice(2, -2))}</b>);
    else if (seg.startsWith('`') && seg.endsWith('`')) out.push(<code key={i}>{marked(seg.slice(1, -1))}</code>);
    else out.push(<Fragment key={i}>{marked(seg)}</Fragment>);
  });
  return <>{out}</>;
}

/**
 * 그림 보기(D159): 앱 창 전체를 덮는 겹. 처음에는 그림 **전체가 화면에 들어가게** 줄여 보이고,
 * [실제 크기]를 누르면(또는 그림을 누르면) 원래 크기로 — 넘치는 부분은 가로·세로로 스크롤해 끝까지 볼 수 있다.
 * Esc·X·바깥(어두운 곳) 누르기는 이 겹만 닫는다(도움말 창은 그대로, Modal 의 Esc 쌓임).
 * body 에 포털로 그려 도움말 카드의 크기·스크롤에 갇히지 않는다.
 */
function ImageViewer({ url, alt, onClose }: { url: string; alt: string; onClose: () => void }) {
  const [actual, setActual] = useState(false);
  useEscLayer(onClose);
  const outside = (e: ReactMouseEvent) => {
    if (e.target === e.currentTarget) onClose();
  };
  return createPortal(
    <div className="help-viewer" role="dialog" aria-modal="true" aria-label={alt || t('help.title')} onMouseDown={outside}>
      <div className="help-viewer-bar">
        <span className="help-viewer-title">{alt}</span>
        <button type="button" className="btn" aria-pressed={actual} onClick={() => setActual((a) => !a)}>
          {actual ? t('help.imgFit') : t('help.imgActual')}
        </button>
        <button type="button" className="icon-btn" onClick={onClose} aria-label={t('common.close')} title={t('common.close')}>
          <X size={18} />
        </button>
      </div>
      <div className={`help-viewer-stage${actual ? ' actual' : ''}`} onMouseDown={outside}>
        <img src={url} alt={alt} title={actual ? t('help.imgFit') : t('help.imgActual')} onClick={() => setActual((a) => !a)} />
      </div>
    </div>,
    document.body,
  );
}

/** 도움말 그림(본문 안에서는 줄인 크기). 누르면 그림 보기(ImageViewer)를 연다. 그림 파일이 없으면 아무것도 그리지 않는다. */
function Figure({ src, alt, lang, terms }: { src: string; alt: string; lang: string; terms: readonly string[] }) {
  const [open, setOpen] = useState(false);
  const url = helpImage(lang, src);
  if (!url) return null;
  // 보기 겹은 단추 밖(형제)에 둔다: 포털 안의 클릭도 React 트리로는 부모에게 올라가므로, 단추 안에 두면 닫자마자 다시 열린다.
  return (
    <figure className="help-fig">
      <button type="button" className="help-fig-btn" onClick={() => setOpen(true)} title={t('help.imgZoom')} aria-label={alt}>
        <img src={url} alt={alt} loading="lazy" />
      </button>
      {alt && <figcaption className="faint small"><Inline text={alt} terms={terms} /></figcaption>}
      {open && <ImageViewer url={url} alt={alt} onClose={() => setOpen(false)} />}
    </figure>
  );
}

/**
 * 크기 조절 손잡이(D159): 도움말 카드 오른쪽 아래 모서리. 끌면 크기가 바뀌고(놓을 때 저장), 두 번 누르면 기본 크기.
 * 카드는 가로 가운데·위쪽 고정이라 너비는 끈 거리의 2배, 높이는 끈 거리만큼 바꾼다(손잡이가 마우스를 따라온다).
 * 키보드: 포커스 뒤 방향키 20px(Shift 는 60px).
 */
function ResizeHandle({ size, onSize }: { size: HelpSize | null; onSize: (s: HelpSize | null, save: boolean) => void }) {
  const drag = useRef<{ x: number; y: number; w: number; h: number } | null>(null);
  const cardOf = (el: Element) => el.closest('.modal')?.getBoundingClientRect();
  const view = () => ({ w: window.innerWidth, h: window.innerHeight });
  const onDown = (e: ReactPointerEvent<HTMLDivElement>) => {
    const r = cardOf(e.currentTarget);
    if (!r) return;
    e.preventDefault();
    e.currentTarget.setPointerCapture?.(e.pointerId);
    drag.current = { x: e.clientX, y: e.clientY, w: r.width, h: r.height };
  };
  const onMove = (e: ReactPointerEvent<HTMLDivElement>) => {
    const d = drag.current;
    if (!d) return;
    const v = view();
    onSize(clampHelpSize({ w: d.w + 2 * (e.clientX - d.x), h: d.h + (e.clientY - d.y) }, v.w, v.h), false);
  };
  const onUp = (e: ReactPointerEvent<HTMLDivElement>) => {
    if (!drag.current) return;
    drag.current = null;
    e.currentTarget.releasePointerCapture?.(e.pointerId);
    onSize(size, true);
  };
  const onKey = (e: ReactKeyboardEvent<HTMLDivElement>) => {
    const step = e.shiftKey ? 60 : 20;
    const dx = e.key === 'ArrowRight' ? step : e.key === 'ArrowLeft' ? -step : 0;
    const dy = e.key === 'ArrowDown' ? step : e.key === 'ArrowUp' ? -step : 0;
    if (!dx && !dy) return;
    e.preventDefault();
    const r = cardOf(e.currentTarget);
    const base = size ?? { w: r?.width ?? 820, h: r?.height ?? 600 };
    const v = view();
    onSize(clampHelpSize({ w: base.w + dx, h: base.h + dy }, v.w, v.h), true);
  };
  return (
    <div
      className="help-resize"
      role="separator"
      aria-orientation="vertical"
      tabIndex={0}
      aria-label={t('help.resize')}
      title={t('help.resize')}
      onPointerDown={onDown}
      onPointerMove={onMove}
      onPointerUp={onUp}
      onPointerCancel={onUp}
      onDoubleClick={() => onSize(null, true)}
      onKeyDown={onKey}
    />
  );
}

function Block({ b, terms, lang }: { b: HelpBlock; terms: readonly string[]; lang: string }) {
  switch (b.kind) {
    case 'qa':
      return (
        <section className="help-qa">
          <div className="help-q"><span className="help-q-mark" aria-hidden="true">Q</span><Inline text={b.q} terms={terms} /></div>
          {b.a && <div className="help-a"><span className="help-q-mark a" aria-hidden="true">A</span><span><Inline text={b.a} terms={terms} /></span></div>}
        </section>
      );
    case 'img':
      return <Figure src={b.src} alt={b.alt} lang={lang} terms={terms} />;
    case 'h':
      return <h4><Inline text={b.text} terms={terms} /></h4>;
    case 'p':
      return <p><Inline text={b.text} terms={terms} /></p>;
    case 'note':
      return <div className="notice info"><Inline text={b.text} terms={terms} /></div>;
    case 'code':
      return <pre className="mono selectable">{markParts(b.text, terms).map(([part, hit], i) => (hit ? <mark key={i}>{part}</mark> : <Fragment key={i}>{part}</Fragment>))}</pre>;
    case 'table':
      return (
        <table className="help-table">
          <thead><tr>{b.head.map((c, i) => <th key={i}><Inline text={c} terms={terms} /></th>)}</tr></thead>
          <tbody>{b.rows.map((r, i) => <tr key={i}>{r.map((c, j) => <td key={j}><Inline text={c} terms={terms} /></td>)}</tr>)}</tbody>
        </table>
      );
    default: {
      const items = b.items.map((it, i) => (
        <li key={i}>
          <Inline text={it.text} terms={terms} />
          {it.sub.length > 0 && <ul>{it.sub.map((s, j) => <li key={j}><Inline text={s} terms={terms} /></li>)}</ul>}
        </li>
      ));
      return b.kind === 'ol' ? <ol>{items}</ol> : <ul>{items}</ul>;
    }
  }
}

export function HelpBody({ topic, terms, lang = 'en' }: { topic: HelpTopic; terms: readonly string[]; lang?: string }) {
  return (
    <article className="help-body selectable" aria-label={topic.title}>
      <h3><Inline text={topic.title} terms={terms} /></h3>
      {topic.blocks.map((b, i) => <Block key={i} b={b} terms={terms} lang={lang} />)}
    </article>
  );
}

export function HelpDialog() {
  const lang = useLang((s) => s.lang);
  const startTopic = useUi((s) => s.help.topic);
  const close = useUi((s) => s.closeHelp);
  // bodyLang: 실제로 보이는 본문의 언어(대체 언어면 en). 이름순 정렬과 그림 고르기에 쓴다.
  const { topics, fallback, lang: bodyLang } = useMemo(() => helpFor(lang), [lang]);
  const [query, setQuery] = useState('');
  const [picked, setPicked] = useState<string | null>(startTopic);
  const [sort, setSortState] = useState<HelpSort>(loadHelpSort);
  // 창 크기(D159): 저장된 값을 지금 창에 맞게 잘라 쓴다. null 이면 CSS 기본 크기.
  const [size, setSizeState] = useState<HelpSize | null>(() => {
    const s = loadHelpSize();
    return s ? clampHelpSize(s, window.innerWidth, window.innerHeight) : null;
  });
  const setSize = (s: HelpSize | null, save: boolean) => {
    setSizeState(s);
    if (save) saveHelpSize(s);
  };
  const setSort = (s: HelpSort) => {
    setSortState(s);
    saveHelpSort(s);
  };
  const terms = useMemo(() => searchTerms(query), [query]);
  const hits = useMemo(() => searchHelp(topics, query), [topics, query]);
  const ordered = useMemo(() => sortTopics(topics, sort, bodyLang), [topics, sort, bodyLang]);
  const searching = terms.length > 0;
  // 검색 중이면 맞은 항목 중에서(맞은 정도 순), 아니면 고른 순서의 전체에서 고른다. 고른 것이 목록에 없으면 첫 항목.
  const listed = searching ? hits.map((h) => h.topic) : ordered;
  const current = listed.find((x) => x.id === picked) ?? listed[0];

  return (
    <Modal title={t('help.title')} icon={CircleHelp} size="lg" onClose={close} className="help-modal" style={size ? { width: size.w, height: size.h } : undefined}>
      <div className="help-wrap">
        <div className="help-side">
          <div style={{ position: 'relative' }}>
            <Search size={13} style={{ position: 'absolute', left: 8, top: 9, color: 'var(--fg-3)' }} />
            <input
              className="input"
              style={{ paddingLeft: 26 }}
              autoFocus
              placeholder={t('help.search')}
              aria-label={t('help.search')}
              value={query}
              onChange={(e) => setQuery(e.target.value)}
              onKeyDown={(e) => {
                if (e.key === 'Enter' && hits[0]) setPicked(hits[0].topic.id);
              }}
            />
          </div>
          {searching ? (
            <div className="faint small" role="status">
              {hits.length > 0 ? t('help.found', { count: hits.length }) : t('help.noResult', { query: query.trim() })}
            </div>
          ) : (
            <div className="help-sort" role="group" aria-label={t('help.sort')}>
              {(['feature', 'name'] as const).map((s) => (
                <button key={s} type="button" className={`help-sort-btn${sort === s ? ' on' : ''}`} aria-pressed={sort === s} onClick={() => setSort(s)}>
                  {t(s === 'feature' ? 'help.sortFeature' : 'help.sortName')}
                </button>
              ))}
            </div>
          )}
          <nav className="help-list" aria-label={t('help.topics')}>
            {(searching ? hits : ordered.map((topic) => ({ topic, snippet: '' }))).map(({ topic, snippet }) => (
              <button
                key={topic.id}
                type="button"
                className={`help-item${current?.id === topic.id ? ' on' : ''}`}
                aria-current={current?.id === topic.id ? 'true' : undefined}
                onClick={() => setPicked(topic.id)}
              >
                <span className="help-item-title"><Inline text={topic.title} terms={terms} /></span>
                {snippet && <span className="help-item-snip faint small"><Inline text={snippet} terms={terms} /></span>}
              </button>
            ))}
          </nav>
        </div>
        <div className="help-main">
          {fallback && <div className="notice info small">{t('help.fallback')}</div>}
          {current ? <HelpBody topic={current} terms={terms} lang={bodyLang} /> : <div className="faint">{t('help.noResult', { query: query.trim() })}</div>}
        </div>
      </div>
      <ResizeHandle size={size} onSize={setSize} />
    </Modal>
  );
}
