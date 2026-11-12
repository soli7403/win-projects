// 단축키 표(화면 표시용). 근거: spec/behavior/shortcuts.md §4.
// 시작 화면은 처음 12개, 설정 화면은 전체를 보여 준다.

export interface ShortcutItem {
  name: string;
  keys: string;
}

export const SHORTCUT_LABELS: ShortcutItem[] = [
  { name: '빠른 접속 / 명령', keys: 'Ctrl+K' },
  { name: '새 탭 (빠른 접속)', keys: 'Ctrl+Shift+T' },
  { name: '새 로컬 터미널', keys: 'Ctrl+Alt+T' },
  { name: '탭 닫기', keys: 'Ctrl+Shift+W' },
  { name: '오른쪽 분할 (같은 서버)', keys: 'Ctrl+Shift+D' },
  { name: '아래 분할 (같은 서버)', keys: 'Ctrl+Shift+E' },
  { name: '패널 이동', keys: 'Alt+방향키' },
  { name: '패널 크게 보기', keys: 'Ctrl+Shift+Enter' },
  { name: '탭 이동', keys: 'Ctrl+Tab / Ctrl+1~9' },
  { name: '붙여넣기 (이미지·파일 자동 업로드)', keys: 'Ctrl+V' },
  { name: '클립보드 이미지 → 서버', keys: 'Ctrl+Alt+V' },
  { name: '복사', keys: 'Ctrl+Shift+C / 선택' },
  { name: '터미널에서 찾기', keys: 'Ctrl+Shift+F' },
  { name: 'SFTP 패널', keys: 'Ctrl+Shift+S' },
  { name: '동시 입력(브로드캐스트)', keys: 'Ctrl+Shift+B' },
  { name: '서버 목록 표시/숨김', keys: 'Ctrl+Shift+L' },
  { name: '글자 크기', keys: 'Ctrl+= / Ctrl+- / Ctrl+0' },
];

/** 시작 화면에 보이는 처음 12개. */
export const START_SHORTCUTS: ShortcutItem[] = SHORTCUT_LABELS.slice(0, 12);
