// 터미널 색 테마. 근거: spec/ui/tokens.md §6 (값 그대로).
export interface TermTheme {
  name: string;
  background: string;
  foreground: string;
  cursor: string;
  selectionBackground: string;
  black: string; red: string; green: string; yellow: string; blue: string; magenta: string; cyan: string; white: string;
  brightBlack: string; brightRed: string; brightGreen: string; brightYellow: string; brightBlue: string; brightMagenta: string; brightCyan: string; brightWhite: string;
}

// 순서: 배경, 글자, 커서, 선택 배경, 8색, 밝은 8색.
const ROWS: [string, string[]][] = [
  ['DariTerm Dark', ['#0d1117', '#d6dde6', '#7dd3fc', '#2f4a6b', '#1b2029', '#f87171', '#4ade80', '#facc15', '#60a5fa', '#c084fc', '#22d3ee', '#d6dde6', '#5b6573', '#fca5a5', '#86efac', '#fde68a', '#93c5fd', '#d8b4fe', '#67e8f9', '#ffffff']],
  ['DariTerm Light', ['#ffffff', '#1f2937', '#2563eb', '#bfdbfe', '#1f2937', '#dc2626', '#15803d', '#a16207', '#1d4ed8', '#9333ea', '#0e7490', '#e5e7eb', '#6b7280', '#ef4444', '#16a34a', '#ca8a04', '#2563eb', '#a855f7', '#0891b2', '#f9fafb']],
  ['PuTTY Classic', ['#000000', '#bbbbbb', '#00ff00', '#555555', '#000000', '#bb0000', '#00bb00', '#bbbb00', '#0000bb', '#bb00bb', '#00bbbb', '#bbbbbb', '#555555', '#ff5555', '#55ff55', '#ffff55', '#5555ff', '#ff55ff', '#55ffff', '#ffffff']],
  ['Solarized Dark', ['#002b36', '#93a1a1', '#93a1a1', '#073642', '#073642', '#dc322f', '#859900', '#b58900', '#268bd2', '#d33682', '#2aa198', '#eee8d5', '#586e75', '#cb4b16', '#93a1a1', '#839496', '#839496', '#6c71c4', '#93a1a1', '#fdf6e3']],
  ['Monokai', ['#272822', '#f8f8f2', '#f8f8f0', '#49483e', '#272822', '#f92672', '#a6e22e', '#f4bf75', '#66d9ef', '#ae81ff', '#a1efe4', '#f8f8f2', '#75715e', '#f92672', '#a6e22e', '#f4bf75', '#66d9ef', '#ae81ff', '#a1efe4', '#f9f8f5']],
  ['Nord', ['#2e3440', '#d8dee9', '#d8dee9', '#434c5e', '#3b4252', '#bf616a', '#a3be8c', '#ebcb8b', '#81a1c1', '#b48ead', '#88c0d0', '#e5e9f0', '#4c566a', '#bf616a', '#a3be8c', '#ebcb8b', '#81a1c1', '#b48ead', '#8fbcbb', '#eceff4']],
];

const keys = ['background', 'foreground', 'cursor', 'selectionBackground', 'black', 'red', 'green', 'yellow', 'blue', 'magenta', 'cyan', 'white', 'brightBlack', 'brightRed', 'brightGreen', 'brightYellow', 'brightBlue', 'brightMagenta', 'brightCyan', 'brightWhite'] as const;

export const TERM_THEMES: TermTheme[] = ROWS.map(([name, vals]) => {
  const t = { name } as TermTheme;
  keys.forEach((k, i) => {
    (t as unknown as Record<string, string>)[k] = vals[i];
  });
  return t;
});

/** 이름이 없으면 DariTerm Dark. 저장 값이 원본 앱 접두어 `BrightTerm ` 으로 시작하면 `DariTerm ` 으로 읽는다(store.md §4). */
export function resolveTermTheme(name: string | undefined): TermTheme {
  const fixed = (name ?? '').replace(/^BrightTerm /, 'DariTerm ');
  return TERM_THEMES.find((t) => t.name === fixed) ?? TERM_THEMES[0];
}
