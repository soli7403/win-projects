# 도움말 창 크기 조절과 그림 보기 (+ 공통 모달의 Esc 는 맨 위 겹만)

**작성**: 2026-11-27 · **상태**: 코드·자동 테스트·Edge 화면 확인 완료(D159). 앱 화면에서의 확인은 수동 검증으로 남음(§H) · **대상**: cli-manager-cr(DariTerm) · **Known issue**: 없음
**앞 문서**: [2026-11-27-18-help-sort-faq-images.md](./2026-11-27-18-help-sort-faq-images.md) — **이 문서는 -17·-18 을 적용한 상태를 기준으로 쓴다.**
**명세**: [spec/ui/help.md](../../spec/ui/help.md) §1·§2·§6, [spec/ui/dialogs.md](../../spec/ui/dialogs.md) §1
**첨부 파일**: 없음(그림은 -18 의 폴더 그대로 쓴다)

---

## 0. 한 줄 요약

사용자 요청 두 가지:

1. "도움말 창의 크기를 조절 가능하게" → 카드 **오른쪽 아래 모서리 손잡이**로 크기 조절. 놓을 때 기억하고, 두 번 누르면 기본 크기.
2. "그림을 선택하면 그림이 잘려서 보이는데, 전체 보게 할 방법이 없다" → 그림을 누르면 **앱 창 전체를 덮는 그림 보기**. 처음에는 **그림 전체가 화면에 맞게**, [실제 크기]로 원래 크기 + 가로·세로 스크롤.

이를 위해 공통 `Modal` 을 고쳤다: **Esc 는 맨 위 겹 하나만 닫는다**(전에는 열린 모달이 모두 닫혔다 — 그림 보기에서 Esc 를 누르면 도움말까지 닫혔을 것이다). `Modal` 에 `className`·`style` 도 더했다.

## 1. 원인 — 그림이 잘려 보인 까닭

-18 의 그림은 누르면 `.help-fig.zoom` 이 되어 본문 **안에서** `max-width/max-height` 를 풀었다. 그런데 그림을 감싼 단추(`.help-fig-btn`)는 `max-width: 100%`(본문 폭)이고 `overflow: auto` 라, 본문보다 넓은 그림은 단추 안에서 가로 스크롤해야 했고 본문 자체도 세로로 스크롤되어 **한눈에 전체를 볼 방법이 없었다**. 도움말 창 크기도 820px × 70vh 고정이라 넓힐 수 없었다.

## 2. 결정

| 항목 | 결정 | 까닭 |
|---|---|---|
| 크기 조절 방식 | 자체 손잡이(포인터 끌기) | CSS `resize: both` 는 카드가 가로 가운데 정렬이라 손잡이가 마우스의 절반 속도로 움직인다. 자체 손잡이는 너비를 끈 거리의 **2배**로 바꿔 손잡이가 마우스를 따라온다 |
| 손잡이 자리·모양 | 카드 오른쪽 아래 18×18, 빗금 두 줄, `nwse-resize` | 흔한 모양 |
| 크기 범위 | 최소 600×380. 최대 (앱 창 너비 − 40) × (앱 창 높이 × 0.92 − 16). 창이 더 작으면 최소 | 목록(250px)+본문이 읽히는 최소. 카드 위 여백(8vh)을 빼고 화면을 넘지 않게 |
| 기본 크기 | 820 × 80vh(전 70vh 본문 + 머리글보다 조금 큼) | |
| 기억 | 놓을 때 `localStorage['dariterm.helpSize'] = {"w","h"}`(정수). 열 때 지금 앱 창에 맞게 잘라 씀. 끄는 동안은 저장 안 함 | 보기 편의라 store(동기화·백업)에 두지 않는다(D156 의 정렬과 같은 방식) |
| 되돌리기 | 손잡이 두 번 누르기 → 기본 크기 + 저장값 지움 | |
| 키보드 | 손잡이(tabIndex 0, role separator)에 포커스 뒤 방향키 20px, Shift 60px | 마우스 없이도 조절 |
| 그림 보기 | `body` 포털, `position: fixed; inset: 0; z-index: 200`, 어두운 바탕(0.82). 위 줄: 설명·[실제 크기]/[화면에 맞추기]·X | 도움말 카드 크기·스크롤에 갇히지 않게. 그림은 화면에 맞춰 전체가 보인다 |
| 실제 크기 | `.actual` → 무대(stage)가 `overflow: auto`, 그림 `max-*: none` | 원래 크기를 끝까지 스크롤해 볼 수 있다 |
| 닫기 | Esc · X · 어두운 곳(무대의 빈 곳 포함) 누르기. **그림 보기만** 닫는다 | 사용자는 도움말로 돌아가 계속 읽는다 |
| 보기 겹의 자리 | `<figure>` 안, **단추의 형제**로 | 포털 안의 클릭도 React 트리로는 부모로 올라간다. 단추 안에 두면 X 를 누르자마자 단추의 onClick 이 다시 연다(코드 주석으로 남김) |
| Esc 쌓임 | 공통 `Modal` 에 `useEscLayer(onEsc)` — 열린 겹 중 **처음 그려진 번호가 가장 큰 겹**만 Esc 를 받는다 | effect 등록 순서로 정하면 부모·자식이 한꺼번에 열릴 때(자식 effect 가 먼저) 순서가 뒤집힌다 — 테스트에서 실제로 실패해서 렌더 단계 번호(`useState(() => ++seq)`, 부모가 먼저)로 바꿨다 |
| onClose 가 다시 그릴 때마다 바뀌어도 | ref 로 들고 등록은 처음 한 번 | 다시 등록하면 순서가 맨 위로 바뀐다 |
| `Modal` 의 포커스 되돌리기 | effect 의존성을 `[onClose]` → `[]` | 전에는 onClose 가 바뀔 때마다 포커스를 잡았다 놓았다 했다(Esc 처리와 같은 effect 에 있었음). 처음 열 때 잡고 닫을 때 되돌리는 게 맞다 |
| 문구 | `help.imgShrink` 없앰(쓰는 곳 없음), `help.imgActual`·`help.imgFit`·`help.resize` 추가 × 10개 언어 | |

## 3. 공통 모달 — `frontend/src/ui/Modal.tsx`

영향: `Modal` 을 쓰는 화면 20곳. 바뀌는 동작은 **두 모달이 겹쳐 있을 때 Esc 가 맨 위만 닫는 것** 하나다(한 개만 열려 있으면 전과 같다). 새 props `className`·`style` 은 선택이다.

before → after:

````diff
--- a/frontend/src/ui/Modal.tsx
+++ b/frontend/src/ui/Modal.tsx
@@ -1,10 +1,44 @@
 // 공통 모달. 근거: spec/ui/dialogs.md §1. Esc·바깥 클릭으로 닫는다(카드 안 클릭은 닫지 않음).
-import { useEffect, useRef, type ReactNode } from 'react';
+import { useEffect, useRef, useState, type CSSProperties, type ReactNode } from 'react';
 import type { LucideIcon } from 'lucide-react';
 import { X } from 'lucide-react';
 import { t } from '../i18n';
 import { keyCapture } from '../lib/keys';
 
+// Esc 로 닫히는 겹(모달, 도움말 그림 보기 등)의 쌓임. Esc 는 **맨 위 겹 하나만** 닫는다(D159).
+// 전에는 모달마다 창 전체의 Esc 를 들어서, 겹쳐 있으면 Esc 한 번에 모두 닫혔다.
+// "맨 위" 는 **처음 그려진 순서가 가장 늦은** 겹이다. effect 는 자식이 부모보다 먼저 돌아서, 부모와 자식이 한꺼번에
+// 열리면 등록 순서가 뒤집힌다 — 그래서 처음 그릴 때(렌더 단계, 부모가 먼저) 번호를 받는다.
+const openLayers = new Set<number>();
+let layerSeq = 0;
+
+/**
+ * Esc 를 받는 겹으로 등록한다. 열려 있는 동안 맨 위 겹일 때만 onEsc 를 부른다.
+ * onEsc 는 ref 로 들어 다시 그려져도 다시 등록하지 않는다(번호·순서가 그대로).
+ */
+export function useEscLayer(onEsc: () => void): void {
+  const cb = useRef(onEsc);
+  cb.current = onEsc;
+  const [seq] = useState(() => ++layerSeq);
+  useEffect(() => {
+    openLayers.add(seq);
+    const onKey = (e: KeyboardEvent) => {
+      // 설정의 단축키 구역이 키를 받는 중이면 Esc 는 그쪽의 취소다 (D126).
+      if (e.key !== 'Escape' || keyCapture.active) return;
+      if (seq !== Math.max(...openLayers)) return;
+      // Esc 는 맨 위 겹이 가로채고 아래로 전달하지 않는다.
+      e.preventDefault();
+      e.stopPropagation();
+      cb.current();
+    };
+    window.addEventListener('keydown', onKey, true);
+    return () => {
+      window.removeEventListener('keydown', onKey, true);
+      openLayers.delete(seq);
+    };
+  }, [seq]);
+}
+
 interface ModalProps {
   title: string;
   icon?: LucideIcon;
@@ -18,6 +52,10 @@ interface ModalProps {
   /** 바깥 클릭으로 닫기 허용 여부(기본 허용). */
   dismissOnOverlay?: boolean;
   labelledBy?: string;
+  /** 카드(.modal)에 더할 클래스(예: 도움말의 help-modal). */
+  className?: string;
+  /** 카드(.modal)의 인라인 스타일(예: 사용자가 바꾼 크기). */
+  style?: CSSProperties;
 }
 
 export function Modal({
@@ -31,29 +69,21 @@ export function Modal({
   hideClose,
   dismissOnOverlay = true,
   labelledBy,
+  className,
+  style,
 }: ModalProps) {
   const prevFocus = useRef<Element | null>(null);
+  useEscLayer(onClose);
 
   useEffect(() => {
     prevFocus.current = document.activeElement;
-    const onKey = (e: KeyboardEvent) => {
-      // 설정의 단축키 구역이 키를 받는 중이면 Esc 는 그쪽의 취소다 (D126).
-      if (e.key === 'Escape' && !keyCapture.active) {
-        // Esc 는 모달이 먼저 가로채고 아래로 전달하지 않는다.
-        e.preventDefault();
-        e.stopPropagation();
-        onClose();
-      }
-    };
-    window.addEventListener('keydown', onKey, true);
     return () => {
-      window.removeEventListener('keydown', onKey, true);
       const el = prevFocus.current as HTMLElement | null;
       if (el && typeof el.focus === 'function' && document.contains(el)) el.focus();
     };
-  }, [onClose]);
+  }, []);
 
-  const cls = size === 'sm' ? 'modal sm' : size === 'lg' ? 'modal lg' : 'modal';
+  const cls = (size === 'sm' ? 'modal sm' : size === 'lg' ? 'modal lg' : 'modal') + (className ? ` ${className}` : '');
   return (
     <div
       className="overlay"
@@ -61,7 +91,7 @@ export function Modal({
         if (dismissOnOverlay && e.target === e.currentTarget) onClose();
       }}
     >
-      <div className={cls} role="dialog" aria-modal="true" aria-labelledby={labelledBy} aria-label={labelledBy ? undefined : title}>
+      <div className={cls} style={style} role="dialog" aria-modal="true" aria-labelledby={labelledBy} aria-label={labelledBy ? undefined : title}>
         <div className="modal-head">
           {Icon && <Icon size={18} color={iconColor} />}
           <span className="title" id={labelledBy}>{title}</span>
````

바뀐 뒤 전체:

````tsx
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
````

## 4. 크기 값 — `frontend/src/lib/help.ts`

`helpFor` 바로 앞(-18 의 `saveHelpSort` 뒤)에 넣는다.

````ts
/** 도움말 창 크기(px). null 이면 기본 크기(CSS: 820 × 80vh). D159. */
export interface HelpSize {
  w: number;
  h: number;
}

export const HELP_MIN_W = 600;
export const HELP_MIN_H = 380;
const SIZE_KEY = 'dariterm.helpSize';

/** 창에 들어가게 자른다: 최소 600×380, 최대 (창 너비 − 40) × (창 높이의 92% − 16). 창이 더 작으면 최소가 이긴다. */
export function clampHelpSize(s: HelpSize, viewW: number, viewH: number): HelpSize {
  const maxW = Math.max(HELP_MIN_W, viewW - 40);
  const maxH = Math.max(HELP_MIN_H, Math.floor(viewH * 0.92) - 16);
  const fit = (v: number, lo: number, hi: number) => Math.round(Math.min(hi, Math.max(lo, Number.isFinite(v) ? v : lo)));
  return { w: fit(s.w, HELP_MIN_W, maxW), h: fit(s.h, HELP_MIN_H, maxH) };
}

/** 마지막에 바꾼 크기(이 PC 의 화면 저장소). 없거나 깨졌으면 null(기본 크기). */
export function loadHelpSize(): HelpSize | null {
  try {
    const raw = globalThis.localStorage?.getItem(SIZE_KEY);
    if (!raw) return null;
    const v = JSON.parse(raw) as Partial<HelpSize>;
    return typeof v.w === 'number' && typeof v.h === 'number' && v.w > 0 && v.h > 0 ? { w: v.w, h: v.h } : null;
  } catch {
    return null;
  }
}

/** 크기를 저장한다. null 이면 지운다(기본 크기로). */
export function saveHelpSize(s: HelpSize | null): void {
  try {
    if (s) globalThis.localStorage?.setItem(SIZE_KEY, JSON.stringify({ w: Math.round(s.w), h: Math.round(s.h) }));
    else globalThis.localStorage?.removeItem(SIZE_KEY);
  } catch {
    /* 저장 실패는 무시 — 다음에는 기본 크기 */
  }
}
````

## 5. 도움말 창 — `frontend/src/screens/HelpDialog.tsx`

- import: `useRef`, `MouseEvent`·`KeyboardEvent`·`PointerEvent` 타입, `createPortal`(react-dom), `X`(lucide-react), `useEscLayer`(../ui/Modal), `clampHelpSize`·`loadHelpSize`·`saveHelpSize`·`HelpSize`(../lib/help).
- `Figure`: 누르면 `ImageViewer` 를 연다(`.zoom` 토글을 없앰). 보기 겹은 단추의 형제.
- 새 `ImageViewer`(포털, `useEscLayer`), 새 `ResizeHandle`.
- `HelpDialog`: `size` 상태(처음 값은 저장값을 지금 창에 맞게 자른 것), `Modal` 에 `className="help-modal"`·`style={{ width, height }}`, 카드 끝에 `<ResizeHandle />`.

before → after (앞은 -18 의 파일):

````diff
--- a/frontend/src/screens/HelpDialog.tsx
+++ b/frontend/src/screens/HelpDialog.tsx
@@ -1,18 +1,23 @@
 // 도움말 창: 왼쪽은 검색 칸과 항목 목록, 오른쪽은 고른 항목의 설명. 검색어는 본문에서 강조한다.
 // 근거: spec/ui/help.md (D155). 본문은 locales/help/<언어>.md(lib/help.ts).
-import { Fragment, useMemo, useState, type ReactNode } from 'react';
-import { CircleHelp, Search } from 'lucide-react';
-import { Modal } from '../ui/Modal';
+import { Fragment, useMemo, useRef, useState, type KeyboardEvent as ReactKeyboardEvent, type MouseEvent as ReactMouseEvent, type PointerEvent as ReactPointerEvent, type ReactNode } from 'react';
+import { createPortal } from 'react-dom';
+import { CircleHelp, Search, X } from 'lucide-react';
+import { Modal, useEscLayer } from '../ui/Modal';
 import {
+  clampHelpSize,
   helpFor,
   helpImage,
+  loadHelpSize,
   loadHelpSort,
   markParts,
+  saveHelpSize,
   saveHelpSort,
   searchHelp,
   searchTerms,
   sortTopics,
   type HelpBlock,
+  type HelpSize,
   type HelpSort,
   type HelpTopic,
 } from '../lib/help';
@@ -32,21 +37,111 @@ function Inline({ text, terms }: { text: string; terms: readonly string[] }) {
   return <>{out}</>;
 }
 
-/** 도움말 그림. 누르면 실제 크기로 펼치고 다시 누르면 접는다. 그림 파일이 없으면 아무것도 그리지 않는다. */
+/**
+ * 그림 보기(D159): 앱 창 전체를 덮는 겹. 처음에는 그림 **전체가 화면에 들어가게** 줄여 보이고,
+ * [실제 크기]를 누르면(또는 그림을 누르면) 원래 크기로 — 넘치는 부분은 가로·세로로 스크롤해 끝까지 볼 수 있다.
+ * Esc·X·바깥(어두운 곳) 누르기는 이 겹만 닫는다(도움말 창은 그대로, Modal 의 Esc 쌓임).
+ * body 에 포털로 그려 도움말 카드의 크기·스크롤에 갇히지 않는다.
+ */
+function ImageViewer({ url, alt, onClose }: { url: string; alt: string; onClose: () => void }) {
+  const [actual, setActual] = useState(false);
+  useEscLayer(onClose);
+  const outside = (e: ReactMouseEvent) => {
+    if (e.target === e.currentTarget) onClose();
+  };
+  return createPortal(
+    <div className="help-viewer" role="dialog" aria-modal="true" aria-label={alt || t('help.title')} onMouseDown={outside}>
+      <div className="help-viewer-bar">
+        <span className="help-viewer-title">{alt}</span>
+        <button type="button" className="btn" aria-pressed={actual} onClick={() => setActual((a) => !a)}>
+          {actual ? t('help.imgFit') : t('help.imgActual')}
+        </button>
+        <button type="button" className="icon-btn" onClick={onClose} aria-label={t('common.close')} title={t('common.close')}>
+          <X size={18} />
+        </button>
+      </div>
+      <div className={`help-viewer-stage${actual ? ' actual' : ''}`} onMouseDown={outside}>
+        <img src={url} alt={alt} title={actual ? t('help.imgFit') : t('help.imgActual')} onClick={() => setActual((a) => !a)} />
+      </div>
+    </div>,
+    document.body,
+  );
+}
+
+/** 도움말 그림(본문 안에서는 줄인 크기). 누르면 그림 보기(ImageViewer)를 연다. 그림 파일이 없으면 아무것도 그리지 않는다. */
 function Figure({ src, alt, lang, terms }: { src: string; alt: string; lang: string; terms: readonly string[] }) {
-  const [zoom, setZoom] = useState(false);
+  const [open, setOpen] = useState(false);
   const url = helpImage(lang, src);
   if (!url) return null;
+  // 보기 겹은 단추 밖(형제)에 둔다: 포털 안의 클릭도 React 트리로는 부모에게 올라가므로, 단추 안에 두면 닫자마자 다시 열린다.
   return (
-    <figure className={`help-fig${zoom ? ' zoom' : ''}`}>
-      <button type="button" className="help-fig-btn" onClick={() => setZoom((z) => !z)} title={zoom ? t('help.imgShrink') : t('help.imgZoom')} aria-label={alt}>
+    <figure className="help-fig">
+      <button type="button" className="help-fig-btn" onClick={() => setOpen(true)} title={t('help.imgZoom')} aria-label={alt}>
         <img src={url} alt={alt} loading="lazy" />
       </button>
       {alt && <figcaption className="faint small"><Inline text={alt} terms={terms} /></figcaption>}
+      {open && <ImageViewer url={url} alt={alt} onClose={() => setOpen(false)} />}
     </figure>
   );
 }
 
+/**
+ * 크기 조절 손잡이(D159): 도움말 카드 오른쪽 아래 모서리. 끌면 크기가 바뀌고(놓을 때 저장), 두 번 누르면 기본 크기.
+ * 카드는 가로 가운데·위쪽 고정이라 너비는 끈 거리의 2배, 높이는 끈 거리만큼 바꾼다(손잡이가 마우스를 따라온다).
+ * 키보드: 포커스 뒤 방향키 20px(Shift 는 60px).
+ */
+function ResizeHandle({ size, onSize }: { size: HelpSize | null; onSize: (s: HelpSize | null, save: boolean) => void }) {
+  const drag = useRef<{ x: number; y: number; w: number; h: number } | null>(null);
+  const cardOf = (el: Element) => el.closest('.modal')?.getBoundingClientRect();
+  const view = () => ({ w: window.innerWidth, h: window.innerHeight });
+  const onDown = (e: ReactPointerEvent<HTMLDivElement>) => {
+    const r = cardOf(e.currentTarget);
+    if (!r) return;
+    e.preventDefault();
+    e.currentTarget.setPointerCapture?.(e.pointerId);
+    drag.current = { x: e.clientX, y: e.clientY, w: r.width, h: r.height };
+  };
+  const onMove = (e: ReactPointerEvent<HTMLDivElement>) => {
+    const d = drag.current;
+    if (!d) return;
+    const v = view();
+    onSize(clampHelpSize({ w: d.w + 2 * (e.clientX - d.x), h: d.h + (e.clientY - d.y) }, v.w, v.h), false);
+  };
+  const onUp = (e: ReactPointerEvent<HTMLDivElement>) => {
+    if (!drag.current) return;
+    drag.current = null;
+    e.currentTarget.releasePointerCapture?.(e.pointerId);
+    onSize(size, true);
+  };
+  const onKey = (e: ReactKeyboardEvent<HTMLDivElement>) => {
+    const step = e.shiftKey ? 60 : 20;
+    const dx = e.key === 'ArrowRight' ? step : e.key === 'ArrowLeft' ? -step : 0;
+    const dy = e.key === 'ArrowDown' ? step : e.key === 'ArrowUp' ? -step : 0;
+    if (!dx && !dy) return;
+    e.preventDefault();
+    const r = cardOf(e.currentTarget);
+    const base = size ?? { w: r?.width ?? 820, h: r?.height ?? 600 };
+    const v = view();
+    onSize(clampHelpSize({ w: base.w + dx, h: base.h + dy }, v.w, v.h), true);
+  };
+  return (
+    <div
+      className="help-resize"
+      role="separator"
+      aria-orientation="vertical"
+      tabIndex={0}
+      aria-label={t('help.resize')}
+      title={t('help.resize')}
+      onPointerDown={onDown}
+      onPointerMove={onMove}
+      onPointerUp={onUp}
+      onPointerCancel={onUp}
+      onDoubleClick={() => onSize(null, true)}
+      onKeyDown={onKey}
+    />
+  );
+}
+
 function Block({ b, terms, lang }: { b: HelpBlock; terms: readonly string[]; lang: string }) {
   switch (b.kind) {
     case 'qa':
@@ -103,6 +198,15 @@ export function HelpDialog() {
   const [query, setQuery] = useState('');
   const [picked, setPicked] = useState<string | null>(startTopic);
   const [sort, setSortState] = useState<HelpSort>(loadHelpSort);
+  // 창 크기(D159): 저장된 값을 지금 창에 맞게 잘라 쓴다. null 이면 CSS 기본 크기.
+  const [size, setSizeState] = useState<HelpSize | null>(() => {
+    const s = loadHelpSize();
+    return s ? clampHelpSize(s, window.innerWidth, window.innerHeight) : null;
+  });
+  const setSize = (s: HelpSize | null, save: boolean) => {
+    setSizeState(s);
+    if (save) saveHelpSize(s);
+  };
   const setSort = (s: HelpSort) => {
     setSortState(s);
     saveHelpSort(s);
@@ -116,7 +220,7 @@ export function HelpDialog() {
   const current = listed.find((x) => x.id === picked) ?? listed[0];
 
   return (
-    <Modal title={t('help.title')} icon={CircleHelp} size="lg" onClose={close}>
+    <Modal title={t('help.title')} icon={CircleHelp} size="lg" onClose={close} className="help-modal" style={size ? { width: size.w, height: size.h } : undefined}>
       <div className="help-wrap">
         <div className="help-side">
           <div style={{ position: 'relative' }}>
@@ -167,6 +271,7 @@ export function HelpDialog() {
           {current ? <HelpBody topic={current} terms={terms} lang={bodyLang} /> : <div className="faint">{t('help.noResult', { query: query.trim() })}</div>}
         </div>
       </div>
+      <ResizeHandle size={size} onSize={setSize} />
     </Modal>
   );
 }
````

바뀐 뒤 전체는 부록 A.

## 6. 스타일 — `frontend/src/styles/settings.css`

before → after (앞은 -18 의 파일):

````diff
--- a/frontend/src/styles/settings.css
+++ b/frontend/src/styles/settings.css
@@ -74,7 +74,13 @@
 .set-nav-help { margin-top: auto; color: var(--fg-2); }
 
 /* 도움말 창 (D155): 왼쪽 검색·항목 목록, 오른쪽 설명 */
-.help-wrap { display: flex; min-height: 0; flex: 1; margin: -4px -18px -16px; height: 70vh; }
+/* 도움말 카드: 크기 조절(D159). 기본 820 × 80vh, 손잡이로 바꾼 크기는 인라인 width/height. 본문 높이는 카드를 따른다. */
+.modal.help-modal { position: relative; height: 80vh; min-width: 600px; min-height: 380px; max-width: calc(100vw - 40px); max-height: calc(92vh - 16px); }
+.modal.help-modal .modal-body { flex: 1; min-height: 0; overflow: hidden; }
+.help-wrap { display: flex; min-height: 0; flex: 1; margin: -4px -18px -16px; }
+.help-resize { position: absolute; right: 0; bottom: 0; width: 18px; height: 18px; cursor: nwse-resize; border-bottom-right-radius: 12px; touch-action: none;
+  background: linear-gradient(135deg, transparent 0 50%, var(--fg-3) 50% 56%, transparent 56% 68%, var(--fg-3) 68% 74%, transparent 74%); opacity: 0.7; }
+.help-resize:hover, .help-resize:focus-visible { opacity: 1; outline: none; background-color: var(--hover); }
 .help-side { width: 250px; flex: none; padding: 10px; border-right: 1px solid var(--border); display: flex; flex-direction: column; gap: 8px; min-height: 0; }
 .help-list { display: flex; flex-direction: column; gap: 2px; overflow: auto; min-height: 0; }
 .help-item { display: flex; flex-direction: column; align-items: flex-start; gap: 2px; text-align: left; padding: 6px 9px; border: 0; border-radius: 6px; background: none; color: var(--fg-2); cursor: pointer; font: inherit; }
@@ -108,13 +114,20 @@
 .help-q-mark { flex: none; width: 18px; height: 18px; border-radius: 4px; display: inline-flex; align-items: center; justify-content: center; font-size: 11px; font-weight: 700; background: var(--accent); color: #fff; }
 .help-q-mark.a { background: var(--border); color: var(--fg); }
 .help-qa code { background: var(--active); } /* 카드 바탕(--hover)과 같은 색이면 코드가 안 보인다 */
-/* 그림, D158: 보통은 본문 폭·높이 360px 안에, 누르면 실제 크기 */
+/* 그림, D158: 본문 안에서는 본문 폭·높이 360px 안에. 누르면 그림 보기(D159) */
 .help-fig { margin: 0 0 12px; }
 .help-fig-btn { display: block; padding: 0; border: 1px solid var(--border); border-radius: 6px; background: none; cursor: zoom-in; max-width: 100%; overflow: hidden; }
 .help-fig-btn img { display: block; max-width: 100%; max-height: 360px; object-fit: contain; }
-.help-fig.zoom .help-fig-btn { cursor: zoom-out; overflow: auto; }
-.help-fig.zoom .help-fig-btn img { max-width: none; max-height: none; }
 .help-fig figcaption { margin-top: 4px; }
+/* 그림 보기(D159): 앱 창 전체를 덮는다. 기본은 그림 전체가 보이게 맞추고, .actual 이면 원래 크기 + 스크롤 */
+.help-viewer { position: fixed; inset: 0; z-index: 200; display: flex; flex-direction: column; background: rgba(0, 0, 0, 0.82); animation: fade 0.12s ease-out; }
+.help-viewer-bar { display: flex; align-items: center; gap: 10px; padding: 10px 14px; color: #fff; flex: none; }
+.help-viewer-bar .icon-btn { color: #fff; }
+.help-viewer-title { flex: 1; min-width: 0; overflow: hidden; text-overflow: ellipsis; white-space: nowrap; font-size: 13px; }
+.help-viewer-stage { flex: 1; min-height: 0; display: flex; align-items: center; justify-content: center; padding: 0 16px 16px; }
+.help-viewer-stage img { max-width: 100%; max-height: 100%; object-fit: contain; border-radius: 6px; box-shadow: 0 8px 40px rgba(0, 0, 0, 0.5); cursor: zoom-in; background: var(--panel); }
+.help-viewer-stage.actual { display: block; overflow: auto; text-align: center; }
+.help-viewer-stage.actual img { max-width: none; max-height: none; cursor: zoom-out; display: inline-block; }
 
 /* 설정 왼쪽 열: 구역 목록(nav) + 그 아래 '이 구역 도움말' 버튼 (D155) */
 .set-side { display: flex; flex-direction: column; flex: none; border-right: 1px solid var(--border); }
````

- `.help-wrap` 의 `height: 70vh` 를 없애고 카드 높이를 따르게 했다(`.modal.help-modal .modal-body { flex: 1; min-height: 0; overflow: hidden; }`).
- `.modal` 공통의 `max-height: 84vh` 는 `.help-modal` 에서 `calc(92vh - 16px)` 로 넓혔다(§2 의 최대와 같게).
- `.help-fig.zoom` 규칙 두 줄은 없앴다.

## 7. 문구 (10개 언어) — `locales/<언어>.json`

`"help.imgShrink"` 줄을 **지우고** 그 자리에 세 키를 넣는다(`help.imgZoom` 은 그대로 — 본문 그림의 제목 "눌러서 크게 보기").

| 키 | ko | en | ja | zh | tw | de | fr | es | pt | ru |
|---|---|---|---|---|---|---|---|---|---|---|
| `help.imgActual` | 실제 크기 | Actual size | 実際のサイズ | 实际大小 | 實際大小 | Originalgröße | Taille réelle | Tamaño real | Tamanho real | Реальный размер |
| `help.imgFit` | 화면에 맞추기 | Fit to screen | 画面に合わせる | 适合屏幕 | 符合畫面 | An Bildschirm anpassen | Ajuster à l’écran | Ajustar a la pantalla | Ajustar à tela | По размеру экрана |
| `help.resize` | 끌어서 창 크기 조절 (두 번 누르면 원래 크기) | Drag to resize (double-click to reset) | ドラッグでサイズ変更（ダブルクリックで元に戻す） | 拖动调整大小（双击恢复） | 拖曳以調整大小（按兩下還原） | Ziehen zum Ändern der Größe (Doppelklick: zurücksetzen) | Faire glisser pour redimensionner (double-clic : taille par défaut) | Arrastra para cambiar el tamaño (doble clic: restablecer) | Arraste para redimensionar (clique duplo: restaurar) | Перетащите для изменения размера (двойной щелчок — сброс) |

## 8. 테스트

| 파일 | 테스트 | 확인 |
|---|---|---|
| `ui/Modal.test.tsx` | `className`·`style` 을 카드에 붙인다 | `modal help-modal`, `700px` |
| 〃 | 겹쳐 있으면 Esc 는 맨 위 겹만, 그 겹이 닫히면 다음 Esc 가 아래 모달 | 부모·자식을 **한꺼번에** 여는 경우(순서 뒤집힘 확인용) |
| 〃 | 다시 그려 onClose 가 바뀌어도 쌓인 순서 그대로 | rerender 뒤 Esc → 안쪽만 |
| `lib/help.followup.test.ts` | 창 크기 자르기: 그대로/최소/최대/창이 최소보다 작음/NaN | |
| 〃 | 크기 저장·읽기(반올림), 깨진 값 null, null 저장은 지움 | |
| `screens/HelpDialog.test.tsx` | 줄인 그림과 설명(-18 의 "누르면 zoom" 테스트를 바꿈) | |
| 〃 | 그림 보기: 누르면 열리고 처음에는 맞춤(`actual` 아님) | `role=dialog`, 이름 = 설명 |
| 〃 | [실제 크기] ↔ [화면에 맞추기], 그림을 눌러도 바뀜 | |
| 〃 | Esc 는 그림 보기만 닫고 도움말 그대로, 다음 Esc 가 도움말을 닫음 | |
| 〃 | X·어두운 곳으로 닫히고 다시 열리지 않음 | |
| 〃 | 처음에는 기본 크기(인라인 크기 없음) | |
| 〃 | 손잡이를 끌면 너비는 2배·높이는 그대로 바뀌고, 놓을 때 저장, 다시 열면 그 크기 | 820×600 → +50,+40 → 920×640 |
| 〃 | 최소 600×380 | |
| 〃 | 두 번 누르면 기본 크기·저장값 지움 | |
| 〃 | 방향키 20px, Shift 60px | |

`HelpDialog.test.tsx` 의 `beforeEach` 에 `localStorage.removeItem('dariterm.helpSize')`, `window.innerWidth = 1024; window.innerHeight = 768;` 을 더했다. jsdom 은 요소 크기가 0 이라 끌기 테스트에서 `getBoundingClientRect` 를 흉내 낸다.

## 9. 다른 곳에 적용하는 순서

1. -17, -18 이 적용되어 있는지 확인(-18 의 그림 폴더 `locales/help/img/` 포함).
2. §7 문구(10개 언어: `help.imgShrink` 지우고 3키 넣기).
3. `Modal.tsx` 를 §3 의 전체로 바꾼다.
4. `help.ts` 에 §4 블록.
5. `HelpDialog.tsx` 를 부록 A 로 바꾼다.
6. `settings.css` 를 §6 diff 대로.
7. 테스트(부록 B) → `npx tsc --noEmit`, `node run-vitest.mjs run`(PowerShell), `wails build`.
8. 명세 spec/ui/help.md·dialogs.md, decisions D159, CHANGELOG.

## 10. 검증 결과 (2026-11-27)

- `npx tsc --noEmit` 통과.
- vitest **66개 파일·500개 통과**(-18 뒤 486개). 처음에 Modal 의 Esc 쌓임 테스트 2개가 실패 — 쌓는 순서를 effect 등록 순으로 했더니 부모·자식이 한꺼번에 열릴 때 자식이 아래로 깔렸다. 렌더 단계 번호로 고쳐 통과.
- Edge(vite + 가짜 백엔드)로 확인:
  - 손잡이를 (120, 60) 끌었더니 카드 820×656 → 1060×716, 손잡이도 정확히 (120, 60) 움직임, 저장값 `{"w":1060,"h":716}`.
  - 그림 보기: proxy 그림 전체가 화면 가운데에 맞춰 보임, [실제 크기] 전환, Esc → 그림 보기만 닫히고 도움말은 남음.

## 11. 남은 일

- 그림 보기에서 마우스 휠 확대·끌어서 이동은 넣지 않았다(실제 크기 + 스크롤로 충분하다고 봄). 필요하면 다음에.
- 다른 모달(설정 등)의 크기 조절은 하지 않았다. `className`·`style` 과 같은 손잡이로 넓힐 수 있다.

## H. 수동 검증

| # | 하는 일 | 기대 | 결과 |
|---|---|---|---|
| 1 | 도움말 오른쪽 아래 모서리를 끌기 | 크기가 마우스를 따라 바뀐다. 목록·본문 높이도 따른다 | |
| 2 | 도움말 닫고 다시 열기, 앱 다시 실행 | 바꾼 크기 그대로 | |
| 3 | 손잡이 두 번 누르기 | 기본 크기 | |
| 4 | 앱 창을 작게 한 뒤 도움말 | 카드가 앱 창을 넘지 않는다 | |
| 5 | "프록시와 프록시 명령"의 그림 누르기 | 그림 전체가 앱 창에 맞춰 보인다 | |
| 6 | [실제 크기] → 스크롤 → [화면에 맞추기] | 원래 크기, 끝까지 스크롤, 다시 맞춤 | |
| 7 | 그림 보기에서 Esc / X / 어두운 곳 | 그림 보기만 닫히고 도움말은 그대로 | |
| 8 | 서버 편집 위에 확인 창이 뜬 상태에서 Esc(예: 저장 안 한 변경) | 확인 창만 닫힌다(전에는 둘 다 닫혔다) | |

## 체크리스트

- [x] 크기 조절 · [x] 크기 기억 · [x] 그림 보기(맞춤/실제 크기) · [x] Esc 쌓임 · [x] 문구 10개 언어 · [x] 명세 · [x] 테스트 · [x] Edge 확인 · [ ] §H

---

## 부록 A. `frontend/src/screens/HelpDialog.tsx` (바뀐 뒤 전체)

````tsx
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
````

## 부록 B. 테스트 코드

### B.1 `frontend/src/ui/Modal.test.tsx` (바뀐 뒤 전체)

````tsx
// 공통 모달: Esc 와 바깥 클릭으로 닫히고, 카드 안 클릭은 닫지 않는다(spec/ui/dialogs.md §1).
// 겹쳐 있으면 Esc 는 맨 위 겹만 닫는다(D159).
import { afterEach, describe, expect, it, vi } from 'vitest';
import { cleanup, fireEvent, render, screen } from '@testing-library/react';
import { useState } from 'react';
import { Modal, useEscLayer } from './Modal';

afterEach(() => cleanup());

describe('공통 모달', () => {
  it('Esc 로 닫힌다', () => {
    const onClose = vi.fn();
    render(<Modal title="테스트" onClose={onClose}><p>본문</p></Modal>);
    fireEvent.keyDown(window, { key: 'Escape' });
    expect(onClose).toHaveBeenCalledTimes(1);
  });

  it('바깥(오버레이) 클릭으로 닫히고 카드 안 클릭은 닫지 않는다', () => {
    const onClose = vi.fn();
    const { container } = render(<Modal title="테스트" onClose={onClose}><p>본문</p></Modal>);
    fireEvent.mouseDown(screen.getByText('본문'));
    expect(onClose).not.toHaveBeenCalled();
    fireEvent.mouseDown(container.querySelector('.overlay')!);
    expect(onClose).toHaveBeenCalledTimes(1);
  });

  it('className·style 을 카드에 붙인다', () => {
    render(<Modal title="테스트" onClose={() => undefined} className="help-modal" style={{ width: 700 }}><p>본문</p></Modal>);
    const card = screen.getByRole('dialog');
    expect(card.className).toBe('modal help-modal');
    expect(card.style.width).toBe('700px');
  });
});

describe('Esc 쌓임 (D159)', () => {
  function Layer({ onEsc }: { onEsc: () => void }) {
    useEscLayer(onEsc);
    return null;
  }

  it('겹쳐 있으면 Esc 는 맨 위 겹만 닫고, 그 겹이 닫히면 다음 Esc 가 아래 모달을 닫는다', () => {
    const outer = vi.fn();
    const inner = vi.fn();
    function Both() {
      const [top, setTop] = useState(true);
      return (
        <Modal title="바깥" onClose={outer}>
          {top && <Layer onEsc={() => { inner(); setTop(false); }} />}
        </Modal>
      );
    }
    render(<Both />);
    fireEvent.keyDown(window, { key: 'Escape' });
    expect(inner).toHaveBeenCalledTimes(1);
    expect(outer).not.toHaveBeenCalled();
    fireEvent.keyDown(window, { key: 'Escape' });
    expect(outer).toHaveBeenCalledTimes(1);
    expect(inner).toHaveBeenCalledTimes(1);
  });

  it('다시 그려져 onClose 가 바뀌어도 쌓인 순서는 그대로다', () => {
    const outer = vi.fn();
    const inner = vi.fn();
    function Both({ n }: { n: number }) {
      // 바깥 모달의 onClose 를 그릴 때마다 새 함수로 준다
      return (
        <Modal title={`바깥 ${n}`} onClose={() => outer(n)}>
          <Layer onEsc={inner} />
        </Modal>
      );
    }
    const { rerender } = render(<Both n={1} />);
    rerender(<Both n={2} />);
    fireEvent.keyDown(window, { key: 'Escape' });
    expect(inner).toHaveBeenCalledTimes(1);
    expect(outer).not.toHaveBeenCalled();
  });
});
````

### B.2 `frontend/src/lib/help.followup.test.ts` — import 와 더한 describe

import 줄:

````ts
import { clampHelpSize, helpImage, loadHelpSize, loadHelpSort, saveHelpSize, parseHelp, plainText, saveHelpSort, searchHelp, sortTopics, type HelpBlock, type HelpTopic } from './help';
````

`describe('자주 묻는 질문 카드 (D157)'` 바로 앞에:

````ts
describe('창 크기 (D159)', () => {
  afterEach(() => localStorage.removeItem('dariterm.helpSize'));

  it('창 안으로 자른다: 최소 600×380, 최대 (너비−40)×(높이×0.92−16)', () => {
    expect(clampHelpSize({ w: 900, h: 600 }, 1600, 1000)).toEqual({ w: 900, h: 600 });
    expect(clampHelpSize({ w: 100, h: 100 }, 1600, 1000)).toEqual({ w: 600, h: 380 });
    expect(clampHelpSize({ w: 5000, h: 5000 }, 1600, 1000)).toEqual({ w: 1560, h: 904 });
    // 창이 최소보다 작으면 최소가 이긴다
    expect(clampHelpSize({ w: 5000, h: 5000 }, 500, 300)).toEqual({ w: 600, h: 380 });
    expect(clampHelpSize({ w: Number.NaN, h: 500 }, 1600, 1000)).toEqual({ w: 600, h: 500 });
  });

  it('저장·읽기, 깨진 값은 null, null 저장은 지우기', () => {
    expect(loadHelpSize()).toBeNull();
    saveHelpSize({ w: 900.4, h: 600.6 });
    expect(loadHelpSize()).toEqual({ w: 900, h: 601 });
    localStorage.setItem('dariterm.helpSize', '{"w":"x","h":1}');
    expect(loadHelpSize()).toBeNull();
    localStorage.setItem('dariterm.helpSize', 'not json');
    expect(loadHelpSize()).toBeNull();
    saveHelpSize({ w: 900, h: 600 });
    saveHelpSize(null);
    expect(localStorage.getItem('dariterm.helpSize')).toBeNull();
  });
});
````

### B.3 `frontend/src/screens/HelpDialog.test.tsx` — before → after (앞은 -18 의 파일)

````diff
--- a/frontend/src/screens/HelpDialog.test.tsx
+++ b/frontend/src/screens/HelpDialog.test.tsx
@@ -7,6 +7,9 @@ import { useLang } from '../i18n';
 
 beforeEach(() => {
   localStorage.removeItem('dariterm.helpSort');
+  localStorage.removeItem('dariterm.helpSize');
+  window.innerWidth = 1024;
+  window.innerHeight = 768;
   useLang.getState().setLang('ko');
   useUi.setState({ help: { open: true, topic: null, returnTo: null }, settings: { open: false, section: 'appearance' } });
 });
@@ -83,18 +86,124 @@ describe('도움말 창 — 정렬·질문 카드·그림 (D156~D158)', () => {
     expect(cards[0].querySelector('.help-a')!.textContent).toContain('다시 연결');
   });
 
-  it('그림이 있는 항목은 그림과 설명을 보이고, 누르면 크게·다시 누르면 작게', () => {
+  it('그림이 있는 항목은 줄인 그림과 설명을 보인다', () => {
     useUi.setState({ help: { open: true, topic: 'proxy', returnTo: null } });
     const { container } = render(<HelpDialog />);
     const img = container.querySelector('.help-fig img') as HTMLImageElement;
     expect(img).toBeTruthy();
     expect(img.getAttribute('src')).toMatch(/ko\/proxy\.png/);
     expect(img.alt).toBe('서버 편집 > 고급 — 프록시 명령');
-    const fig = container.querySelector('.help-fig')!;
-    fireEvent.click(fig.querySelector('button')!);
-    expect(fig.classList.contains('zoom')).toBe(true);
-    fireEvent.click(fig.querySelector('button')!);
-    expect(fig.classList.contains('zoom')).toBe(false);
+    expect(container.querySelector('.help-fig figcaption')!.textContent).toBe('서버 편집 > 고급 — 프록시 명령');
+  });
+});
+
+describe('그림 보기 (D159)', () => {
+  const ALT = '서버 편집 > 고급 — 프록시 명령';
+  const openViewer = () => {
+    useUi.setState({ help: { open: true, topic: 'proxy', returnTo: null } });
+    const r = render(<HelpDialog />);
+    fireEvent.click(screen.getByRole('button', { name: ALT }));
+    return r;
+  };
+  const viewer = () => document.body.querySelector('.help-viewer') as HTMLElement | null;
+
+  it('그림을 누르면 화면 전체 보기가 열리고, 처음에는 그림 전체가 보이게 맞춘다', () => {
+    openViewer();
+    expect(viewer()).toBeTruthy();
+    expect(screen.getByRole('dialog', { name: ALT })).toBe(viewer());
+    expect(viewer()!.querySelector('.help-viewer-stage')!.classList.contains('actual')).toBe(false);
+    expect((viewer()!.querySelector('.help-viewer-stage img') as HTMLImageElement).getAttribute('src')).toMatch(/ko\/proxy\.png/);
+  });
+
+  it('[실제 크기] ↔ [화면에 맞추기], 그림을 눌러도 바뀐다', () => {
+    openViewer();
+    const stage = () => viewer()!.querySelector('.help-viewer-stage')!;
+    fireEvent.click(screen.getByRole('button', { name: '실제 크기' }));
+    expect(stage().classList.contains('actual')).toBe(true);
+    fireEvent.click(screen.getByRole('button', { name: '화면에 맞추기' }));
+    expect(stage().classList.contains('actual')).toBe(false);
+    fireEvent.click(stage().querySelector('img')!);
+    expect(stage().classList.contains('actual')).toBe(true);
+  });
+
+  it('Esc 는 그림 보기만 닫고 도움말은 그대로, 다음 Esc 가 도움말을 닫는다', () => {
+    openViewer();
+    fireEvent.keyDown(window, { key: 'Escape' });
+    expect(viewer()).toBeNull();
+    expect(useUi.getState().help.open).toBe(true);
+    expect(screen.getByRole('article', { name: '프록시와 프록시 명령' })).toBeTruthy();
+    fireEvent.keyDown(window, { key: 'Escape' });
+    expect(useUi.getState().help.open).toBe(false);
+  });
+
+  it('X 와 바깥(어두운 곳) 누르기로 닫히고, 닫은 뒤 다시 열리지 않는다', () => {
+    openViewer();
+    fireEvent.click(viewer()!.querySelector('.help-viewer-bar .icon-btn')!);
+    expect(viewer()).toBeNull();
+    fireEvent.click(screen.getByRole('button', { name: ALT }));
+    fireEvent.mouseDown(viewer()!);
+    expect(viewer()).toBeNull();
+    expect(useUi.getState().help.open).toBe(true);
+  });
+});
+
+describe('도움말 창 크기 조절 (D159)', () => {
+  afterEach(() => localStorage.removeItem('dariterm.helpSize'));
+  const card = () => screen.getByRole('dialog', { name: '도움말' }) as HTMLElement;
+  const handle = () => screen.getByRole('separator', { name: /창 크기 조절/ });
+
+  it('처음에는 기본 크기(인라인 크기 없음)', () => {
+    render(<HelpDialog />);
+    expect(card().style.width).toBe('');
+    expect(card().classList.contains('help-modal')).toBe(true);
+  });
+
+  it('손잡이를 끌면 크기가 바뀌고 놓을 때 저장하며, 다음에 열 때 그 크기로 연다', () => {
+    window.innerWidth = 1600;
+    window.innerHeight = 1000;
+    render(<HelpDialog />);
+    // jsdom 은 크기가 0 이라 카드 크기를 흉내 낸다
+    card().getBoundingClientRect = () => ({ width: 820, height: 600, top: 0, left: 0, right: 820, bottom: 600, x: 0, y: 0, toJSON: () => ({}) });
+    fireEvent.pointerDown(handle(), { clientX: 100, clientY: 100, pointerId: 1 });
+    fireEvent.pointerMove(handle(), { clientX: 150, clientY: 140, pointerId: 1 });
+    expect(card().style.width).toBe('920px'); // 가운데 정렬이라 2배
+    expect(card().style.height).toBe('640px');
+    expect(localStorage.getItem('dariterm.helpSize')).toBeNull(); // 끄는 동안은 저장하지 않는다
+    fireEvent.pointerUp(handle(), { clientX: 150, clientY: 140, pointerId: 1 });
+    expect(JSON.parse(localStorage.getItem('dariterm.helpSize')!)).toEqual({ w: 920, h: 640 });
+    cleanup();
+    render(<HelpDialog />);
+    expect(card().style.width).toBe('920px');
+  });
+
+  it('최소 600×380 보다 작게는 줄지 않는다', () => {
+    render(<HelpDialog />);
+    card().getBoundingClientRect = () => ({ width: 820, height: 600, top: 0, left: 0, right: 820, bottom: 600, x: 0, y: 0, toJSON: () => ({}) });
+    fireEvent.pointerDown(handle(), { clientX: 500, clientY: 500, pointerId: 1 });
+    fireEvent.pointerMove(handle(), { clientX: 0, clientY: 0, pointerId: 1 });
+    expect(card().style.width).toBe('600px');
+    expect(card().style.height).toBe('380px');
+  });
+
+  it('두 번 누르면 기본 크기로 돌아가고 저장값을 지운다', () => {
+    localStorage.setItem('dariterm.helpSize', JSON.stringify({ w: 900, h: 600 }));
+    render(<HelpDialog />);
+    expect(card().style.width).toBe('900px');
+    fireEvent.doubleClick(handle());
+    expect(card().style.width).toBe('');
+    expect(localStorage.getItem('dariterm.helpSize')).toBeNull();
+  });
+
+  it('방향키로도 조절한다(20px, Shift 60px)', () => {
+    localStorage.setItem('dariterm.helpSize', JSON.stringify({ w: 900, h: 600 }));
+    window.innerWidth = 1600;
+    window.innerHeight = 1000;
+    render(<HelpDialog />);
+    fireEvent.keyDown(handle(), { key: 'ArrowRight' });
+    expect(card().style.width).toBe('920px');
+    fireEvent.keyDown(handle(), { key: 'ArrowDown', shiftKey: true });
+    expect(card().style.height).toBe('660px');
+    expect(JSON.parse(localStorage.getItem('dariterm.helpSize')!)).toEqual({ w: 920, h: 660 });
   });
 });
 
````
