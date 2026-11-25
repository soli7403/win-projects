# 설정에 "단축키" 구역 만들기 + 단축키마다 키 바꾸기 + 전체 켜기/끄기 (`shortcutsEnabled`·`shortcuts`) — 배경·설계·코드·검증

**작성**: 2026-11-25 · **상태**: 코드·자동 테스트·빌드 완료(D126). 앱 화면에서의 확인은 수동 검증으로 남음(§H) · **대상**: cli-manager-cr(DariTerm) · **Known issue**: 없음
**관련**: 같은 방식으로 구역을 떼어 낸 앞 문서 [2026-11-23-log-settings-section-ansi.md](./2026-11-23-log-settings-section-ansi.md)(D120, "로그" 구역). 명세 [spec/behavior/shortcuts.md](../../spec/behavior/shortcuts.md) §5(새로), [spec/ui/settings.md](../../spec/ui/settings.md) §2a·§8, [spec/data/store.md](../../spec/data/store.md) §4

> 이 문서 하나로 **무엇을 바꿨는지 → 왜 그렇게 정했는지 → 코드 전체 → 어떻게 검증하는지**를 알 수 있게 쓴다.

---

## 0. 한 줄 요약

세 가지를 했다.

1. **설정 화면에 "단축키" 구역을 새로 두고**, "정보" 맨 아래에 있던 단축키 표를 그리로 옮겼다. 구역은 9개에서 10개가 됐다("입력" 바로 다음).
2. **앱 단축키 34개의 키를 하나씩 바꿀 수 있다.** 다른 동작이 쓰던 키를 고르면 새 동작에 그 키를 주고, **먼저 쓰던 동작은 빈 값으로 저장**한다(키 없음).
3. **"단축키 사용" 토글**(기본 켜짐)로 앱 단축키 전체를 끈다. 끄면 그 키들은 앱이 가로채지 않고 터미널(서버)로 간다.

| | 바꾸기 전 | 바꾼 뒤 |
|---|---|---|
| 단축키 표의 자리 | 설정 > 정보 맨 아래(보기만) | 설정 > **단축키**(바꾸기·지우기·기본값으로) |
| 키 | 코드에 고정 | 명령마다 설정(`shortcuts`), 기본값은 전과 같다 |
| vi·tmux 와 겹칠 때 | 방법 없음(앱이 먼저 가로챔) | 그 단축키만 바꾸거나 지우기, 또는 전체 끄기(`shortcutsEnabled`) |
| 같은 키를 두 동작에 | — | 나중에 고른 동작이 갖고, 먼저 쓰던 동작은 "없음" |

예: "터미널에서 찾기"의 키 칸을 누르고 Ctrl+K 를 누르면

```
바꾸기 전:  빠른 연결 / 명령 = Ctrl+K      터미널에서 찾기 = Ctrl+Shift+F
바꾼 뒤:    빠른 연결 / 명령 = 없음        터미널에서 찾기 = Ctrl+K
저장 값:    "shortcuts": { "find": "Ctrl+KeyK", "quick": "" }
```

## 1. 사용자 요청

2026-11-25: "단축키를 정보에서 따로 빼서 로그처럼 단위 메뉴로 만들고 단축키를 각각 설정할 수 있도록 해줘. 중복 설정하면 먼저 있는 것을 빈 값으로 저장하고. 또 전체 단축키를 enable or disable 할 수 있도록 해줘. vi 를 사용하는 사람들은 단축키 충돌이 일어난다. 이전과 같이 import-plans 에 md 파일 추가해주고."

왜 겹치는가: 앱은 키 입력을 터미널보다 **먼저** 받아, 앱 단축키로 정해진 조합이면 동작을 실행하고 터미널로 넘기지 않는다([frontend/src/App.tsx](../../frontend/src/App.tsx) 의 창 단위 키 처리, [frontend/src/screens/TerminalView.tsx](../../frontend/src/screens/TerminalView.tsx) 의 터미널 키 처리). 그래서 서버 쪽 프로그램이 같은 조합을 쓰면 그 프로그램은 그 키를 받지 못한다. 예: vim 의 Ctrl+K(이중 글자 입력), 창 사이를 Alt+방향키로 옮기도록 설정한 vim·tmux, Ctrl+Tab·Ctrl+숫자를 쓰는 편집기 설정.

### 1.1 용어

| 말 | 뜻 |
|---|---|
| 앱 단축키 | 앱이 터미널보다 먼저 받아 처리하는 키(빠른 연결, 창 나누기, 탭 이동 등). 이 문서의 대상 |
| 터미널 안에서 쓰는 키 | 터미널 칸이 직접 처리하는 복사·붙여넣기 키(Ctrl+V, Ctrl+Shift+V, Shift+Insert, Ctrl+Shift+C, Ctrl+Alt+V). 대상이 아니다 |
| 명령 | 키를 붙일 수 있는 동작 하나. id 가 있다(`quick`, `find`, `tab3` …) |
| 조합 | 누른 키를 적은 글자. `Ctrl+` `Alt+` `Shift+` (이 순서) 뒤에 물리 키 이름. 예: `Ctrl+Shift+KeyT`, `Alt+ArrowUp`, `F11` |
| 물리 키 이름 | 키보드에서의 자리 이름(웹 표준의 `code`). `KeyT` 는 한글 입력기가 켜져 있어도 같은 자리의 키다 |
| 바꾼 것 표 | 설정 `shortcuts`. 기본값에서 바꾼 명령만 담는다. 값이 빈 글자면 "키 없음" |

## 2. 설계 결정

### 2.1 "단축키" 구역

| 항목 | 정한 것 | 이유 |
|---|---|---|
| 구역 이름·자리 | "단축키", **입력 바로 다음**(3번째) | 키보드에 관한 설정이 이어지게 한다. "로그"를 연결 다음에 둔 것과 같은 생각 |
| 아이콘 | 명령 키(lucide `Command`, ⌘ 모양) | 키보드 모양(`Keyboard`)은 "입력"이 쓰고 있다 |
| 내부 구역 id | `shortcuts` (`SettingsSection` 에 추가) | — |
| "정보" 구역 | 단축키 표를 뺐다. 다른 내용은 그대로 | 요청 그대로 |
| 명세의 절 번호 | `spec/ui/settings.md` 에 **§2a** 로 넣고 §3~§8 은 그대로 | 코드 주석과 다른 문서가 §4a·§6·§8 을 가리킨다(D120 과 같은 방식) |

### 2.2 무엇을 바꿀 수 있게 하나

| 항목 | 정한 것 | 이유 |
|---|---|---|
| 대상 | **앱 단축키 34개**(§2.6 표) | 터미널보다 먼저 가로채는 키가 충돌의 원인이다 |
| 터미널 안에서 쓰는 키 | **대상에서 뺐다**(바꿀 수 없고, 전체를 꺼도 동작) | 전체를 껐을 때 복사·붙여넣기까지 막히면 터미널을 쓸 수 없다. vi 의 Ctrl+V(블록 선택)는 이미 설정 > 입력 "Ctrl+V로 붙여넣기"(`ctrlVPaste`)로 끈다. 구역 아래에 보기 전용 표로 두고 그 안내를 적었다 |
| 묶여 있던 키 | 명령 하나씩으로 **풀었다**: Alt+방향키 → 위/아래/왼쪽/오른쪽 4개, Ctrl+1~9 → 1~8번째 탭 + 마지막 탭 9개, 글자 크기 → 크게/작게/처음으로 3개, Ctrl+Tab·Ctrl+Shift+Tab → 다음/이전 탭 | "각각 설정"에 맞추고, 겹침 처리를 "조합 하나 ↔ 명령 하나"로 단순하게 한다. Alt+방향키 중 하나만 vim 에 넘기는 것도 된다 |
| 빠른 연결의 키 3개 | 명령 3개로: `quick`(Ctrl+K) · `newTab`(Ctrl+Shift+T) · `quickAlt`(Ctrl+Shift+N). 동작은 같다(열기/닫기) | 명령마다 키를 하나로 두려는 것. 지금 있던 키를 없애지 않는다 |
| 대화상자·목록 안의 키 | 대상 아님(빠른 연결의 Ctrl+Enter, 서버 목록의 F2·Delete 등) | 그 화면에 포커스가 있을 때만 듣는 키라 터미널과 겹치지 않는다 |

### 2.3 저장 형식

| 항목 | 정한 것 | 이유 |
|---|---|---|
| `shortcutsEnabled` | 참/거짓, 기본 **true** | 기존 사용자는 달라지지 않는다 |
| `shortcuts` | 표(`{ "명령 id": "조합" }`), 기본 **빈 표**. **바꾼 명령만** 담는다 | 손대지 않은 명령은 나중에 기본 키를 고쳐도 새 기본값을 받는다. 파일이 작고, 무엇을 바꿨는지 한눈에 보인다 |
| 키 없음 | 값이 **빈 글자** `""` | 요청("빈 값으로 저장") 그대로. "표에 없음 = 기본 키"와 구분된다 |
| 기본 키로 되돌리면 | 표에서 그 명령을 **뺀다** | 기본 키와 같은 값을 적어 둘 까닭이 없다 |
| 조합의 글자 | `Ctrl+Shift+KeyT` 처럼 물리 키 이름 | 지금도 물리 키로 판단한다(한글 입력기와 무관, shortcuts.md §1). 화면에는 `Ctrl+Shift+T` 로 바꿔 보인다 |
| 맞추는 방법 | 조합이 **정확히 같을 때만** | 표 하나로 찾을 수 있고 겹침 판단이 분명하다 |
| 숫자 패드 | Enter · + · - · 0~9 는 본 자판의 같은 키로 본다 | 전에 되던 Ctrl+숫자패드+(글자 크게) 등을 그대로 둔다 |
| 동기화·백업 | 다른 설정과 함께 실린다 | `settings` 안의 키라 따로 할 일이 없다 |
| 파일을 손으로 고쳐 겹친 경우 | 표의 **앞 명령**이 그 키를 갖고 뒤 명령은 키 없음으로 본다. 모르는 id·글자가 아닌 값은 무시 | 읽을 때 한쪽으로 정해 두면 어느 동작이 실행될지 흔들리지 않는다 |

### 2.4 겹칠 때

| 상황 | 동작 |
|---|---|
| 다른 명령이 쓰던 조합을 고름 | 새 명령에 그 조합, **먼저 쓰던 명령은 `""` 로 저장**. 알림 '{키} 를 쓰던 "{이름}"의 단축키를 비웠습니다' |
| 묻지 않고 바꾸는 까닭 | 요청이 "먼저 있는 것을 빈 값으로 저장"이다. 잘못 눌렀으면 [기본값으로] 한 번이면 되돌아간다 |
| [기본값으로]를 눌렀는데 그 기본 키를 다른 명령이 쓰고 있음 | 같은 규칙: 되돌리는 명령이 키를 갖고, 쓰고 있던 명령이 비워진다 |
| [모두 기본값으로] | 표를 비운다(전부 기본 키) |

### 2.5 고를 수 없는 조합

| 조합 | 까닭 |
|---|---|
| Ctrl 도 Alt 도 없는 보통 키(`a`, `Shift+a`, `Enter` …) | 글자 입력을 앱이 가로채게 된다. 단 **F1~F24** 는 그냥도, Shift 와도 고를 수 있다 |
| Ctrl+V · Ctrl+Shift+V · Ctrl+Shift+C · Ctrl+Alt+V · Shift+Insert | 터미널 안에서 쓰는 복사·붙여넣기 키. 앱 단축키가 먼저 가로채면 복사·붙여넣기가 안 된다 |
| Esc(조합 키 없이) | 키 받기의 취소로 쓴다 |
| Win 키가 낀 조합 | Windows 가 쓴다. 받지 않는다 |

Ctrl+C 처럼 터미널에 꼭 필요한 키도 막지는 않았다(고르면 그 키는 서버로 가지 않는다). 사용자가 일부러 고르는 것이고, [기본값으로]로 되돌릴 수 있다.

### 2.6 명령 34개(화면에 보이는 순서)와 기본 키

| # | id | 이름 | 기본 키 | 동작 |
|---|---|---|---|---|
| 1 | `quick` | 빠른 연결 / 명령 | Ctrl+K | 빠른 연결 열기/닫기 |
| 2 | `newTab` | 새 탭 (빠른 연결) | Ctrl+Shift+T | 같음 |
| 3 | `quickAlt` | 빠른 연결 (보조 키) | Ctrl+Shift+N | 같음 |
| 4 | `quickOpen` | 빠른 연결 열기 (닫지 않음) | Ctrl+Shift+P | 늘 열기 |
| 5 | `newLocal` | 새 로컬 터미널 | Ctrl+Alt+T | |
| 6 | `closePane` | 탭 닫기 | Ctrl+Shift+W | 포커스된 창 닫기 |
| 7 | `splitRight` | 같은 서버를 오른쪽에 | Ctrl+Shift+D | |
| 8 | `splitDown` | 같은 서버를 아래에 | Ctrl+Shift+E | |
| 9 | `zoom` | 창 최대화 | Ctrl+Shift+Enter | |
| 10~13 | `focusUp` `focusDown` `focusLeft` `focusRight` | 위쪽/아래쪽/왼쪽/오른쪽 창으로 | Alt+↑ ↓ ← → | 포커스 이동 |
| 14 | `find` | 터미널에서 찾기 | Ctrl+Shift+F | |
| 15 | `sftp` | SFTP 창 | Ctrl+Shift+S | |
| 16 | `broadcast` | 일괄 입력 | Ctrl+Shift+B | |
| 17 | `sidebar` | 서버 목록 표시/숨김 | Ctrl+Shift+L | |
| 18 | `tabNext` | 다음 탭 | Ctrl+Tab | |
| 19 | `tabPrev` | 이전 탭 | Ctrl+Shift+Tab | |
| 20 | `tabLeft` | 탭을 왼쪽으로 옮기기 | Ctrl+Shift+← | |
| 21 | `tabRight` | 탭을 오른쪽으로 옮기기 | Ctrl+Shift+→ | |
| 22~29 | `tab1`~`tab8` | {n}번째 탭 | Ctrl+1~8 | |
| 30 | `tabLast` | 마지막 탭 | Ctrl+9 | |
| 31 | `fontUp` | 글자 크게 | Ctrl+= | |
| 32 | `fontDown` | 글자 작게 | Ctrl+- | |
| 33 | `fontReset` | 글자 크기 처음으로 | Ctrl+0 | |
| 34 | `fullscreen` | 창 최대화/복원 | F11 | |

### 2.7 화면

```
설정 > 단축키
┌──────────────────────────────────────────────────────────────┐
│ 단축키 사용                                            (●)   │  ← shortcutsEnabled
│ 끄면 아래 앱 단축키가 모두 동작하지 않고, 그 키는 터미널…    │
└──────────────────────────────────────────────────────────────┘
앱 단축키                                       [모두 기본값으로]
키 칸을 누른 뒤 쓸 키를 누르세요. Esc 는 취소입니다. …
┌──────────────────────────────────────────────────────────────┐
│ 빠른 연결 / 명령                    [   Ctrl+K     ]  ×  ↺  │
│ 새 탭 (빠른 연결)                   [ Ctrl+Shift+T ]  ×  ↺  │
│ 터미널에서 찾기                     [키를 누르세요…]  ×  ↺  │  ← 받는 중(강조색 테두리)
│ SFTP 창                             [    없음      ]  ×  ↺  │  ← 비워진 명령(흐린 글자)
│ …(34줄)                                                      │
└──────────────────────────────────────────────────────────────┘
터미널 안에서 쓰는 키
바꿀 수 없고, "단축키 사용"을 꺼도 동작합니다. Ctrl+V 붙여넣기는 설정 > 입력에서 끌 수 있습니다
  붙여넣기 (이미지·파일 자동 업로드)  Ctrl+V     클립보드 이미지 → 서버  Ctrl+Alt+V
  복사  Ctrl+Shift+C / 선택
```

| 항목 | 정한 것 | 이유 |
|---|---|---|
| 키 받기 | 키 칸을 누르면 "키를 누르세요…", 다음에 누른 조합을 받는다. Ctrl·Shift·Alt 만 누른 것은 기다린다 | 조합을 글자로 치게 하면 틀리기 쉽다 |
| 받기 취소 | Esc, 다른 곳 누르기, 창이 포커스를 잃음. **그동안의 Esc 는 설정 창을 닫지 않는다** | Esc 로 취소하려다 설정이 닫히면 안 된다 |
| 받는 동안 | 앱 단축키가 동작하지 않는다 | Ctrl+Shift+D 를 새 키로 주려는데 창이 나뉘면 안 된다 |
| 고를 수 없는 조합 | 오류 알림을 내고 **계속 기다린다** | 다시 키 칸을 누르지 않아도 된다 |
| × / ↺ | [키 지우기](키가 없으면 누를 수 없다) / [기본값으로](기본 키면 누를 수 없다, 툴팁에 기본 키) | — |
| 꺼져 있을 때 | 목록의 이름·키를 흐리게. **바꾸는 것은 된다** | 끈 채로 키를 정리해 두고 다시 켤 수 있다 |
| 저장 | 누르는 즉시(저장 버튼 없음), 열려 있는 터미널에도 곧바로 | 다른 설정과 같다 |

### 2.8 다른 화면의 키 표기

키가 바뀌므로, 키를 글자로 적어 두었던 곳을 지금 키를 보이도록 바꿨다. 키가 없으면(비웠거나 전체를 껐으면) 표기를 뺀다.

| 곳 | 전 | 뒤 |
|---|---|---|
| 시작 화면의 단축키 표(12줄) | 고정 글자 | 지금 키. 키가 없는 줄은 뺀다. 묶음 줄("창 이동", "탭 이동")은 묶인 명령이 모두 기본 키면 전의 표기(`Alt+방향키`, `Ctrl+Tab / Ctrl+1~9`), 하나라도 바꿨으면 키가 있는 첫 명령의 키 + ` …` |
| 시작 화면 안내 "… Ctrl+K 로 바로 연결하세요" | 고정 | 빠른 연결을 여는 키 중 지금 있는 것(`quick` → `newTab` → `quickAlt` → `quickOpen`). 하나도 없으면 문장을 보이지 않는다 |
| [로컬 터미널] 버튼 툴팁 | `Ctrl+Alt+T` | 지금 키(없으면 툴팁 없음) |
| 칸 메뉴: 찾기 · 오른쪽에 · 아래에 · SFTP 창 | 고정 | 지금 키 |
| 탭 메뉴: 일괄 입력 · 탭 닫기 | 고정 | 지금 키 |
| 탭 막대 SFTP 버튼 툴팁 | `SFTP (Ctrl+Shift+S)` | `SFTP (지금 키)` 또는 `SFTP` |
| 칸 메뉴: 복사 · 붙여넣기 · 이미지 붙여넣기 | 고정 | **그대로**(터미널 안에서 쓰는 키) |

### 2.9 바꾼 뒤의 흐름

```
키 입력
  │
  ├─ 창 단위 키 처리 (App.tsx, 터미널보다 먼저)
  │     설정 화면이 키를 받는 중(keyCapture.active) → 아무것도 안 함
  │     activeShortcut(e)
  │        ├ shortcutsEnabled 꺼짐      → 표가 비어 있음 → null ┐
  │        ├ 조합이 표에 없음            → null                  ├→ 터미널로
  │        └ 조합이 표에 있음            → 그 명령 실행, 터미널로 넘기지 않음
  │     (볼트 잠김·대화상자 조건은 전과 같다)
  │
  └─ 터미널 키 처리 (TerminalView.tsx)
        activeShortcut(e) 가 있으면 터미널이 처리하지 않음(위에서 이미 실행)
        없으면: 복사·붙여넣기 키(고정) → 그 동작 / 나머지 → 서버로 보냄
```

조합 → 명령 표는 설정이 바뀔 때만 다시 만든다([frontend/src/state/shortcuts.ts](../../frontend/src/state/shortcuts.ts) 가 `shortcutsEnabled`·`shortcuts` 값이 같으면 앞서 만든 표를 그대로 쓴다). 키를 누를 때마다 하는 일은 조합 글자 하나 만들기와 표 한 번 찾기다.

### 2.10 함께 달라진 동작

| 무엇 | 전 | 뒤 | 까닭 |
|---|---|---|---|
| **Ctrl+Shift+Tab** (이전 탭) | 동작하지 않았다 | 동작한다 | 명세(shortcuts.md §2)에는 있었으나 키 처리의 Shift 가지에 Tab 이 빠져 있었다. 표로 다시 만들면서 들어갔다 |
| **F11** | Ctrl+F11·Shift+F11 등도 창 최대화 | 조합 키 없이 누른 F11 만 | 조합을 정확히 맞추게 됐다. Shift+F11 같은 조합을 다른 명령에 줄 수 있다 |
| 웹뷰가 키를 가로챌 걱정 | — | 없다 | Wails 가 웹뷰의 브라우저 가속키를 끄고(`PutAreBrowserAcceleratorKeysEnabled(false)`) 확대도 끈다(`IsZoomControlEnabled` 기본 false). 그래서 앱 단축키를 꺼도 Ctrl+= 가 화면 확대가 되거나 하지 않는다. Wails v2.17.1 소스에서 확인 |

### 2.11 영향 범위

- **저장 형식**: `settings` 에 키 둘(`shortcutsEnabled`·`shortcuts`)이 늘었다. 옛 파일에는 없으므로 기본값(true·빈 표)으로 채워진다. 프런트는 값이 없어도 켜진 것으로 본다.
- **IPC**: 채널은 그대로. `store.get` 응답의 `settings` 에 키가 둘 늘어 계약 고정 파일 2개를 고쳤다.
- **백엔드**: 기본값 두 줄뿐이다. 단축키는 프런트에서만 처리한다.
- **기본 상태(아무것도 바꾸지 않음)**: §2.10 의 두 가지를 빼면 전과 같다.

## A. 바뀐 파일 한눈에

| 파일 | 종류 | 상태 |
|---|---|---|
| `frontend/src/lib/keys.ts` | 코드 | **다시 씀**(명령 표 + 조합 방식) |
| `frontend/src/lib/shortcuts.ts` | 코드 | **다시 씀**(시작 화면 표, 고정 키) |
| `frontend/src/state/shortcuts.ts` | 코드 | **새 파일**(설정 ↔ 키 처리·표기) |
| `frontend/src/screens/SettingsShortcuts.tsx` | 코드 | **새 파일**(단축키 구역) |
| `frontend/src/screens/Settings.tsx` | 코드 | 수정(구역 목록) |
| `frontend/src/screens/SettingsAbout.tsx` | 코드 | 수정(단축키 표를 뺌) |
| `frontend/src/screens/Welcome.tsx` | 코드 | 수정(지금 키 표기) |
| `frontend/src/screens/Panes.tsx`, `TabBar.tsx` | 코드 | 수정(메뉴·툴팁의 키 표기) |
| `frontend/src/screens/TerminalView.tsx` | 코드 | 수정(설정을 따르는 판정) |
| `frontend/src/App.tsx` | 코드 | 수정(설정을 따르는 판정, 키 받는 중 쉬기) |
| `frontend/src/ui/Modal.tsx` | 코드 | 수정(키 받는 중의 Esc) |
| `frontend/src/state/ui.ts` | 코드 | 수정(구역 id) |
| `frontend/src/ipc/types.ts`, `frontend/src/lib/defaults.ts` | 코드 | 수정(설정 2개) |
| `frontend/src/styles/settings.css` | 스타일 | 수정 |
| `internal/store/store.go` | 코드(Go) | 수정(기본값 2줄) |
| `frontend/src/lib/keys.test.ts` | 테스트 | **새 파일**(14개) |
| `frontend/src/screens/SettingsShortcuts.test.tsx` | 테스트 | **새 파일**(18개) |
| `frontend/src/screens/SettingsLog.test.tsx` | 테스트 | 수정(구역 10개) |
| `frontend/src/ipc/contract/store.get.json`, `store.get.afterNullPatch.json` | 계약 고정 파일 | 수정 |
| `locales/*.json` (10개) | 문구 | 키 31개 추가, 2개 삭제 |
| `spec/behavior/shortcuts.md` | 명세 | 수정(§5 새로) |
| `spec/ui/settings.md`, `spec/ui/shell.md`, `spec/data/store.md` | 명세 | 수정 |
| `spec/acceptance/03-panes-terminal.md`, `04-hosts-sftp-settings-sync.md` | 명세 | 수정(K8~K12, G6) |
| `docs/scenarios.md`, `docs/scenarios.en.md` | 문서 | 수정 |
| `docs/manual/user-guide.ko.md`, `user-guide.en.md` | 문서 | 수정(단축키 요약에 안내 한 문단) |
| `CHANGELOG.md`, `docs/decisions.md`(D126), `CLAUDE.md` | 문서 | 수정 |

아래의 `diff` 는 **2026-11-25 의 앞 문서들(D123-2·D125)을 적용한 상태**를 기준으로 뽑았다. 다시 쓴 파일과 새 파일은 전체를 싣는다.

## B. 코드

### `frontend/src/lib/keys.ts` — 다시 씀 (전체)

```ts
// 앱 단축키(순수 함수): 명령 표, 누른 키 → 조합 글자, 조합 → 명령 찾기, 키 바꾸기 규칙.
// 근거: spec/behavior/shortcuts.md §1~§2·§5 (D126).
// 문자 키는 e.code(물리 키)로 판단하므로 한글 입력기가 켜져 있어도 동작한다.
import type { MsgId, Vars } from '../i18n';

export type ShortcutAction =
  | 'quickToggle' | 'quickOpen' | 'newLocal' | 'closePane' | 'splitRight' | 'splitDown' | 'zoom'
  | 'find' | 'sftp' | 'broadcast' | 'sidebar' | 'tabLeft' | 'tabRight' | 'tabNext' | 'tabPrev'
  | 'tabIndex' | 'focusMove' | 'fontUp' | 'fontDown' | 'fontReset' | 'fullscreen';

export type ShortcutArg = number | 'last' | 'up' | 'down' | 'left' | 'right';

export interface KeyLike {
  code: string;
  key: string;
  ctrlKey: boolean;
  shiftKey: boolean;
  altKey: boolean;
  metaKey?: boolean;
}

/** 키를 바꿀 수 있는 명령 하나. keys 는 기본 조합(예: 'Ctrl+Shift+KeyT'). */
export interface ShortcutCommand {
  id: string;
  action: ShortcutAction;
  arg?: ShortcutArg;
  keys: string;
  /** 이름의 문구 키(D75)와 자리표시자 값. */
  label: MsgId;
  labelVars?: Vars;
}

/** 설정 `shortcuts`: 기본값에서 바꾼 명령만 담는다. 값이 빈 글자면 "키 없음". */
export type ShortcutOverrides = Record<string, string>;

/** 설정 화면에 보이는 순서. 같은 조합이 두 명령에 걸리면 앞의 것이 이긴다. */
export const SHORTCUT_COMMANDS: ShortcutCommand[] = [
  { id: 'quick', action: 'quickToggle', keys: 'Ctrl+KeyK', label: 'sc.quick' },
  { id: 'newTab', action: 'quickToggle', keys: 'Ctrl+Shift+KeyT', label: 'sc.newTab' },
  { id: 'quickAlt', action: 'quickToggle', keys: 'Ctrl+Shift+KeyN', label: 'sc.quickAlt' },
  { id: 'quickOpen', action: 'quickOpen', keys: 'Ctrl+Shift+KeyP', label: 'sc.quickOpen' },
  { id: 'newLocal', action: 'newLocal', keys: 'Ctrl+Alt+KeyT', label: 'sc.newLocal' },
  { id: 'closePane', action: 'closePane', keys: 'Ctrl+Shift+KeyW', label: 'sc.closeTab' },
  { id: 'splitRight', action: 'splitRight', keys: 'Ctrl+Shift+KeyD', label: 'sc.splitRight' },
  { id: 'splitDown', action: 'splitDown', keys: 'Ctrl+Shift+KeyE', label: 'sc.splitDown' },
  { id: 'zoom', action: 'zoom', keys: 'Ctrl+Shift+Enter', label: 'sc.zoomPane' },
  { id: 'focusUp', action: 'focusMove', arg: 'up', keys: 'Alt+ArrowUp', label: 'sc.focusUp' },
  { id: 'focusDown', action: 'focusMove', arg: 'down', keys: 'Alt+ArrowDown', label: 'sc.focusDown' },
  { id: 'focusLeft', action: 'focusMove', arg: 'left', keys: 'Alt+ArrowLeft', label: 'sc.focusLeft' },
  { id: 'focusRight', action: 'focusMove', arg: 'right', keys: 'Alt+ArrowRight', label: 'sc.focusRight' },
  { id: 'find', action: 'find', keys: 'Ctrl+Shift+KeyF', label: 'sc.find' },
  { id: 'sftp', action: 'sftp', keys: 'Ctrl+Shift+KeyS', label: 'sc.sftp' },
  { id: 'broadcast', action: 'broadcast', keys: 'Ctrl+Shift+KeyB', label: 'sc.broadcast' },
  { id: 'sidebar', action: 'sidebar', keys: 'Ctrl+Shift+KeyL', label: 'sc.sidebar' },
  { id: 'tabNext', action: 'tabNext', keys: 'Ctrl+Tab', label: 'sc.tabNext' },
  { id: 'tabPrev', action: 'tabPrev', keys: 'Ctrl+Shift+Tab', label: 'sc.tabPrev' },
  { id: 'tabLeft', action: 'tabLeft', keys: 'Ctrl+Shift+ArrowLeft', label: 'sc.tabLeft' },
  { id: 'tabRight', action: 'tabRight', keys: 'Ctrl+Shift+ArrowRight', label: 'sc.tabRight' },
  ...[1, 2, 3, 4, 5, 6, 7, 8].map((n): ShortcutCommand => (
    { id: `tab${n}`, action: 'tabIndex', arg: n - 1, keys: `Ctrl+Digit${n}`, label: 'sc.tabN', labelVars: { n } }
  )),
  { id: 'tabLast', action: 'tabIndex', arg: 'last', keys: 'Ctrl+Digit9', label: 'sc.tabLast' },
  { id: 'fontUp', action: 'fontUp', keys: 'Ctrl+Equal', label: 'sc.fontUp' },
  { id: 'fontDown', action: 'fontDown', keys: 'Ctrl+Minus', label: 'sc.fontDown' },
  { id: 'fontReset', action: 'fontReset', keys: 'Ctrl+Digit0', label: 'sc.fontReset' },
  { id: 'fullscreen', action: 'fullscreen', keys: 'F11', label: 'sc.fullscreen' },
];

/** 터미널 안에서 쓰는 고정 키(복사·붙여넣기). 앱 단축키로 고를 수 없다 (shortcuts.md §5). */
export const RESERVED_COMBOS: readonly string[] = ['Ctrl+KeyV', 'Ctrl+Shift+KeyV', 'Ctrl+Shift+KeyC', 'Ctrl+Alt+KeyV', 'Shift+Insert'];

const MODIFIER_CODES = /^(Control|Shift|Alt|Meta|OS)(Left|Right)?$/;

/** 숫자 패드의 같은 뜻 키는 본 자판 키로 본다(Ctrl+숫자패드 + 도 글자 키우기). */
function normalizeCode(code: string): string {
  if (code === 'NumpadEnter') return 'Enter';
  if (code === 'NumpadAdd') return 'Equal';
  if (code === 'NumpadSubtract') return 'Minus';
  const digit = /^Numpad([0-9])$/.exec(code);
  return digit ? `Digit${digit[1]}` : code;
}

/** 누른 키를 조합 글자로 바꾼다(예: 'Ctrl+Shift+KeyT'). 조합 키만 눌렀거나 Win 키가 끼면 빈 글자. */
export function comboOf(e: KeyLike): string {
  if (!e.code || MODIFIER_CODES.test(e.code) || e.metaKey) return '';
  return `${e.ctrlKey ? 'Ctrl+' : ''}${e.altKey ? 'Alt+' : ''}${e.shiftKey ? 'Shift+' : ''}${normalizeCode(e.code)}`;
}

const CODE_TEXT: Record<string, string> = {
  ArrowUp: '↑', ArrowDown: '↓', ArrowLeft: '←', ArrowRight: '→',
  Equal: '=', Minus: '-', Comma: ',', Period: '.', Slash: '/', Backslash: '\\', Semicolon: ';', Quote: "'",
  BracketLeft: '[', BracketRight: ']', Backquote: '`', Escape: 'Esc', Delete: 'Del', Insert: 'Ins',
  PageUp: 'PgUp', PageDown: 'PgDn', NumpadMultiply: 'Num *', NumpadDivide: 'Num /', NumpadDecimal: 'Num .',
};

/** 조합 글자를 화면 표기로 바꾼다('Ctrl+Shift+KeyT' → 'Ctrl+Shift+T'). 빈 글자는 그대로 빈 글자. */
export function comboText(combo: string): string {
  if (!combo) return '';
  const parts = combo.split('+');
  const code = parts.pop() ?? '';
  const text = CODE_TEXT[code] ?? code.replace(/^Key(?=[A-Z]$)/, '').replace(/^Digit(?=[0-9]$)/, '');
  return [...parts, text].join('+');
}

/**
 * 고를 수 없는 조합이면 까닭을 돌려준다.
 * modifier: Ctrl·Alt 없이 누른 보통 키(F1~F24 는 그냥도 된다) — 글자 입력과 겹친다.
 * reserved: 터미널의 복사·붙여넣기 키.
 */
export function comboProblem(combo: string): 'modifier' | 'reserved' | null {
  if (RESERVED_COMBOS.includes(combo)) return 'reserved';
  const parts = combo.split('+');
  const code = parts[parts.length - 1];
  if (/^F([1-9]|1[0-9]|2[0-4])$/.test(code)) return null;
  return parts.includes('Ctrl') || parts.includes('Alt') ? null : 'modifier';
}

/** 명령마다 지금 쓰는 조합(바꾼 것이 있으면 그것, 없으면 기본). 같은 조합이 겹치면 표의 앞 명령만 갖고 뒤는 빈 글자. */
export function effectiveKeys(overrides?: ShortcutOverrides | null): Record<string, string> {
  const out: Record<string, string> = {};
  const used = new Set<string>();
  for (const c of SHORTCUT_COMMANDS) {
    const saved = overrides?.[c.id];
    const combo = typeof saved === 'string' ? saved : c.keys;
    out[c.id] = combo && !used.has(combo) ? combo : '';
    if (combo) used.add(combo);
  }
  return out;
}

export type ShortcutHit = { action: ShortcutAction; arg?: ShortcutArg; id: string };

/** 조합 → 명령 표. enabled 가 false 면 비어 있다(전체 끄기). */
export function buildKeyMap(enabled: boolean, overrides?: ShortcutOverrides | null): Map<string, ShortcutHit> {
  const map = new Map<string, ShortcutHit>();
  if (!enabled) return map;
  const keys = effectiveKeys(overrides);
  for (const c of SHORTCUT_COMMANDS) {
    if (keys[c.id]) map.set(keys[c.id], { action: c.action, arg: c.arg, id: c.id });
  }
  return map;
}

const DEFAULT_MAP = buildKeyMap(true);

/** 단축키 동작을 돌려준다. 해당 없으면 null. map 을 주지 않으면 기본 키. tabIndex 의 arg 는 0부터의 번호 또는 'last'. */
export function shortcutFor(e: KeyLike, map: Map<string, ShortcutHit> = DEFAULT_MAP): ShortcutHit | null {
  const combo = comboOf(e);
  return (combo && map.get(combo)) || null;
}

/**
 * 명령 id 의 키를 combo 로 바꾼 새 설정 값을 돌려준다(combo 가 빈 글자면 키 없음).
 * 그 조합을 쓰던 다른 명령은 빈 값으로 저장한다 — displaced 가 그 명령 id 들.
 */
export function assignShortcut(overrides: ShortcutOverrides | null | undefined, id: string, combo: string): { overrides: ShortcutOverrides; displaced: string[] } {
  const next: ShortcutOverrides = {};
  const known = new Map(SHORTCUT_COMMANDS.map((c) => [c.id, c]));
  for (const [k, v] of Object.entries(overrides ?? {})) {
    if (known.has(k) && typeof v === 'string') next[k] = v;
  }
  const displaced: string[] = [];
  if (!known.has(id)) return { overrides: next, displaced };
  if (combo) {
    const cur = effectiveKeys(next);
    for (const c of SHORTCUT_COMMANDS) {
      if (c.id !== id && cur[c.id] === combo) {
        next[c.id] = '';
        displaced.push(c.id);
      }
    }
    // 겹쳐서 가려져 있던 뒤쪽 명령(파일을 손으로 고친 경우)도 같은 조합이면 비운다.
    for (const c of SHORTCUT_COMMANDS) {
      if (c.id !== id && !displaced.includes(c.id) && (next[c.id] ?? c.keys) === combo) next[c.id] = '';
    }
  }
  if (combo === known.get(id)!.keys) delete next[id];
  else next[id] = combo;
  return { overrides: next, displaced };
}

/** 설정 화면이 키를 받는 중이면 true. 그동안 앱 단축키와 모달의 Esc 닫기는 쉰다. */
export const keyCapture = { active: false };
```

전의 `keys.ts` 는 `shortcutFor` 하나가 if·switch 가지로 키를 판정했다(파일 전체가 그 함수). 지금은 위의 표에서 찾는다.

### `frontend/src/state/shortcuts.ts` — 새 파일

```ts
// 저장된 단축키 설정(shortcutsEnabled·shortcuts)을 키 처리와 화면 표기에 잇는다. 근거: spec/behavior/shortcuts.md §5 (D126).
import { useApp } from './app';
import { buildKeyMap, comboText, effectiveKeys, shortcutFor, type KeyLike, type ShortcutHit, type ShortcutOverrides } from '../lib/keys';
import type { Settings } from '../ipc/types';

interface Resolved {
  enabled: boolean;
  overrides: ShortcutOverrides | undefined;
  map: Map<string, ShortcutHit>;
  /** 명령 id → 화면 표기('Ctrl+Shift+F'). 꺼져 있거나 키가 없으면 빈 글자. */
  labels: Record<string, string>;
}

let cache: Resolved | null = null;

function resolve(settings: Settings | undefined): Resolved {
  const enabled = settings?.shortcutsEnabled !== false;
  const raw = settings?.shortcuts;
  const overrides = raw && typeof raw === 'object' && !Array.isArray(raw) ? (raw as ShortcutOverrides) : undefined;
  if (cache && cache.enabled === enabled && cache.overrides === overrides) return cache;
  const keys = effectiveKeys(overrides);
  const labels: Record<string, string> = {};
  for (const id of Object.keys(keys)) labels[id] = enabled ? comboText(keys[id]) : '';
  cache = { enabled, overrides, map: buildKeyMap(enabled, overrides), labels };
  return cache;
}

/** 지금 설정에서 이 키 입력이 앱 단축키면 그 동작, 아니면 null(꺼져 있으면 늘 null). */
export function activeShortcut(e: KeyLike): ShortcutHit | null {
  return shortcutFor(e, resolve(useApp.getState().store?.settings).map);
}

/** 명령 id → 화면 표기. 설정이 바뀌면 다시 그린다. 메뉴의 단축키 표기는 `labels.find || undefined` 로 쓴다. */
export function useShortcutLabels(): Record<string, string> {
  return useApp((s) => resolve(s.store?.settings).labels);
}
```

### `frontend/src/lib/shortcuts.ts` — 다시 씀 (전체)

```ts
// 단축키 표(화면 표시용). 근거: spec/behavior/shortcuts.md §4·§5.
// 시작 화면은 12줄(지금 쓰는 키로), 설정의 "단축키" 구역은 명령 전체(lib/keys.ts)와 아래 고정 키를 보여 준다.
import { t, type MsgId } from '../i18n';
import { SHORTCUT_COMMANDS, comboText } from './keys';

export interface ShortcutItem {
  name: MsgId;
  keys: string;
  /** 키 표기에 번역할 말이 있으면(예: 선택) 그 문구 키. */
  keysId?: MsgId;
}

/** 표시용 [이름, 키] 문자열. */
export function shortcutText(s: ShortcutItem): { name: string; keys: string } {
  return { name: t(s.name), keys: s.keysId ? t(s.keysId) : s.keys };
}

/** 터미널 안에서 쓰는 고정 키. 바꿀 수 없고 "단축키 사용"을 꺼도 동작한다. */
export const FIXED_SHORTCUTS: ShortcutItem[] = [
  { name: 'sc.paste', keys: 'Ctrl+V' },
  { name: 'sc.pasteImage', keys: 'Ctrl+Alt+V' },
  { name: 'sc.copy', keys: '', keysId: 'sc.copyKeys' },
];

/** 시작 화면의 한 줄: 명령 하나, 또는 여러 명령을 묶은 줄(묶음이 모두 기본 키면 묶음 표기를 쓴다). */
type StartRow =
  | { id: string }
  | { name: MsgId; ids: string[]; all: string; allId?: MsgId };

const TAB_IDS = ['tabNext', 'tab1', 'tab2', 'tab3', 'tab4', 'tab5', 'tab6', 'tab7', 'tab8', 'tabLast'];

const START_ROWS: StartRow[] = [
  { id: 'quick' },
  { id: 'newTab' },
  { id: 'newLocal' },
  { id: 'closePane' },
  { id: 'splitRight' },
  { id: 'splitDown' },
  { name: 'sc.movePane', ids: ['focusUp', 'focusDown', 'focusLeft', 'focusRight'], all: '', allId: 'sc.altArrows' },
  { id: 'zoom' },
  { name: 'sc.moveTab', ids: TAB_IDS, all: 'Ctrl+Tab / Ctrl+1~9' },
];

const COMMANDS = new Map(SHORTCUT_COMMANDS.map((c) => [c.id, c]));

/**
 * 시작 화면의 단축키 표. labels 는 명령 id → 지금 키 표기(state/shortcuts 의 useShortcutLabels).
 * 키가 없는 줄은 빼고(전체를 끄면 앱 단축키 줄이 모두 빠진다), 끝에 고정 키 3줄을 붙인다.
 */
export function startShortcutRows(labels: Record<string, string>): { name: string; keys: string }[] {
  const rows: { name: string; keys: string }[] = [];
  for (const row of START_ROWS) {
    if ('id' in row) {
      const cmd = COMMANDS.get(row.id);
      if (cmd && labels[row.id]) rows.push({ name: t(cmd.label, cmd.labelVars), keys: labels[row.id] });
      continue;
    }
    const set = row.ids.map((id) => labels[id]).filter(Boolean);
    if (set.length === 0) continue;
    const untouched = row.ids.every((id) => labels[id] === comboText(COMMANDS.get(id)?.keys ?? ''));
    const keys = untouched ? (row.allId ? t(row.allId) : row.all) : set.length > 1 ? `${set[0]} …` : set[0];
    rows.push({ name: t(row.name), keys });
  }
  return [...rows, ...FIXED_SHORTCUTS.map(shortcutText)];
}
```

### `frontend/src/screens/SettingsShortcuts.tsx` — 새 파일

```tsx
// 설정 · 단축키(화면 순서 3번째). "정보"에 있던 단축키 표를 떼어 낸 구역 (D126).
// 전체 켜기/끄기, 명령마다 키 바꾸기(다른 명령이 쓰던 키를 고르면 그 명령은 빈 값으로 저장), 기본값으로 되돌리기.
// 근거: spec/ui/settings.md §2a, spec/behavior/shortcuts.md §5.
import { useEffect, useState } from 'react';
import { RotateCcw, X } from 'lucide-react';
import { ToggleRow } from '../ui/Controls';
import { useApp } from '../state/app';
import { saveSettings } from '../state/settings';
import { useSetting, Group } from './Settings';
import { SHORTCUT_COMMANDS, assignShortcut, comboOf, comboProblem, comboText, effectiveKeys, keyCapture, type ShortcutOverrides } from '../lib/keys';
import { FIXED_SHORTCUTS, shortcutText } from '../lib/shortcuts';
import { t } from '../i18n';

export function SettingsShortcuts() {
  const [enabledRaw, setEnabled] = useSetting('shortcutsEnabled');
  const enabled = enabledRaw !== false;
  const saved = useApp((s) => s.store?.settings.shortcuts) as ShortcutOverrides | undefined;
  const pushToast = useApp((s) => s.pushToast);
  /** 키를 받는 중인 명령 id. */
  const [recording, setRecording] = useState<string | null>(null);
  const keys = effectiveKeys(saved);
  const changed = SHORTCUT_COMMANDS.some((c) => keys[c.id] !== c.keys);

  const nameOf = (id: string) => {
    const c = SHORTCUT_COMMANDS.find((x) => x.id === id);
    return c ? t(c.label, c.labelVars) : id;
  };

  const assign = (id: string, combo: string) => {
    const r = assignShortcut(saved, id, combo);
    void saveSettings({ shortcuts: r.overrides });
    for (const other of r.displaced) pushToast({ kind: 'info', text: t('sc.displaced', { keys: comboText(combo), name: nameOf(other) }) });
  };

  // 키를 받는 동안: 모든 키 입력을 여기서 가로챈다(앱 단축키·모달의 Esc 닫기는 keyCapture 를 보고 쉰다).
  useEffect(() => {
    if (!recording) return undefined;
    keyCapture.active = true;
    const onKey = (e: KeyboardEvent) => {
      e.preventDefault();
      e.stopPropagation();
      if (e.code === 'Escape' && !e.ctrlKey && !e.altKey && !e.shiftKey) {
        setRecording(null);
        return;
      }
      const combo = comboOf(e);
      if (!combo) return; // Ctrl·Shift·Alt 만 누른 상태: 다음 키를 기다린다
      const problem = comboProblem(combo);
      if (problem) {
        pushToast({ kind: 'error', text: problem === 'reserved' ? t('sc.errReserved', { keys: comboText(combo) }) : t('sc.errModifier') });
        return;
      }
      assign(recording, combo);
      setRecording(null);
    };
    const cancel = () => setRecording(null);
    window.addEventListener('keydown', onKey, true);
    window.addEventListener('mousedown', cancel, true);
    window.addEventListener('blur', cancel);
    return () => {
      keyCapture.active = false;
      window.removeEventListener('keydown', onKey, true);
      window.removeEventListener('mousedown', cancel, true);
      window.removeEventListener('blur', cancel);
    };
    // assign 은 saved 를 읽으므로 saved 가 바뀌면 다시 건다.
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [recording, saved]);

  return (
    <>
      <Group>
        <ToggleRow title={t('sc.enable')} desc={t('sc.enableDesc')} checked={enabled} onChange={setEnabled} />
      </Group>

      <div>
        <div className="sc-head">
          <div className="set-section-title" style={{ marginBottom: 0 }}>{t('sc.listTitle')}</div>
          <button className="btn small-btn" type="button" disabled={!changed} onClick={() => void saveSettings({ shortcuts: {} })}>
            {t('sc.resetAll')}
          </button>
        </div>
        <div className="help" style={{ marginBottom: 8 }}>{t('sc.editHint')}</div>
        <div className={`sc-edit${enabled ? '' : ' off'}`}>
          {SHORTCUT_COMMANDS.map((c) => {
            const name = t(c.label, c.labelVars);
            const text = comboText(keys[c.id]);
            const isRec = recording === c.id;
            return (
              <div className="sc-edit-row" key={c.id}>
                <span className="sc-name">{name}</span>
                <button
                  type="button"
                  className={`sc-key${isRec ? ' rec' : ''}${text ? '' : ' none'}`}
                  aria-label={`${name}: ${isRec ? t('sc.press') : text || t('sc.none')}`}
                  // 다른 곳(다른 줄 포함)을 누르면 창의 mousedown 이 먼저 받기를 끝내고, 그 뒤 이 click 이 새로 시작한다.
                  onClick={() => { if (!isRec) setRecording(c.id); }}
                >
                  {isRec ? t('sc.press') : text || t('sc.none')}
                </button>
                <button
                  type="button"
                  className="icon-btn"
                  title={t('sc.clear')}
                  aria-label={`${name}: ${t('sc.clear')}`}
                  disabled={!keys[c.id]}
                  onClick={() => assign(c.id, '')}
                >
                  <X size={14} />
                </button>
                <button
                  type="button"
                  className="icon-btn"
                  title={`${t('sc.reset')} (${comboText(c.keys)})`}
                  aria-label={`${name}: ${t('sc.reset')}`}
                  disabled={keys[c.id] === c.keys}
                  onClick={() => assign(c.id, c.keys)}
                >
                  <RotateCcw size={13} />
                </button>
              </div>
            );
          })}
        </div>
      </div>

      <div>
        <div className="set-section-title">{t('sc.fixedTitle')}</div>
        <div className="help" style={{ marginBottom: 8 }}>{t('sc.fixedDesc')}</div>
        <div className="sc-table">
          {FIXED_SHORTCUTS.map((s) => {
            const x = shortcutText(s);
            return (
              <span key={s.name} style={{ display: 'contents' }}>
                <span className="sc-name">{x.name}</span>
                <span className="kbd">{x.keys}</span>
              </span>
            );
          })}
        </div>
      </div>
    </>
  );
}
```

### `frontend/src/screens/Settings.tsx` — 수정

```diff
--- a/frontend/src/screens/Settings.tsx
+++ b/frontend/src/screens/Settings.tsx
@@ -1,6 +1,6 @@
-// 설정 화면: 크게 보기 모달, 왼쪽 구역 내비 9개 + 오른쪽 본문. 모든 변경은 즉시 저장. 근거: spec/ui/settings.md.
+// 설정 화면: 크게 보기 모달, 왼쪽 구역 내비 10개 + 오른쪽 본문. 모든 변경은 즉시 저장. 근거: spec/ui/settings.md.
 import { useState, type ReactNode, type KeyboardEvent } from 'react';
-import { Palette, Keyboard, ImageUp, Plug, ScrollText, RefreshCw, Shield, Archive, Info } from 'lucide-react';
+import { Palette, Keyboard, Command, ImageUp, Plug, ScrollText, RefreshCw, Shield, Archive, Info } from 'lucide-react';
 import { Modal } from '../ui/Modal';
 import { useUi, type SettingsSection } from '../state/ui';
 import { useApp } from '../state/app';
@@ -8,6 +8,7 @@
 import type { Settings as SettingsT } from '../ipc/types';
 import { SettingsAppearance } from './SettingsAppearance';
 import { SettingsInput } from './SettingsInput';
+import { SettingsShortcuts } from './SettingsShortcuts';
 import { SettingsFiles } from './SettingsFiles';
 import { SettingsConnection } from './SettingsConnection';
 import { SettingsLog } from './SettingsLog';
@@ -21,6 +22,7 @@
 const SECTIONS: { id: SettingsSection; label: MsgId; icon: typeof Palette }[] = [
   { id: 'appearance', label: 'settings.sec.appearance', icon: Palette },
   { id: 'input', label: 'settings.sec.input', icon: Keyboard },
+  { id: 'shortcuts', label: 'settings.sec.shortcuts', icon: Command },
   { id: 'files', label: 'settings.sec.files', icon: ImageUp },
   { id: 'connection', label: 'settings.sec.connection', icon: Plug },
   { id: 'log', label: 'settings.sec.log', icon: ScrollText },
@@ -98,6 +100,7 @@
   const body: Record<SettingsSection, ReactNode> = {
     appearance: <SettingsAppearance />,
     input: <SettingsInput />,
+    shortcuts: <SettingsShortcuts />,
     files: <SettingsFiles />,
     connection: <SettingsConnection />,
     log: <SettingsLog />,
```

### `frontend/src/state/ui.ts` — 수정

```diff
--- a/frontend/src/state/ui.ts
+++ b/frontend/src/state/ui.ts
@@ -45,7 +45,7 @@
   | { kind: 'host'; hostId: string | null; groupId: string | null }
   | { kind: 'folder'; folderId: string | null; parentId: string | null };
 
-export type SettingsSection = 'appearance' | 'input' | 'files' | 'connection' | 'log' | 'sync' | 'security' | 'backup' | 'about';
+export type SettingsSection = 'appearance' | 'input' | 'shortcuts' | 'files' | 'connection' | 'log' | 'sync' | 'security' | 'backup' | 'about';
 
 interface UiState {
   /** 설정 화면(크게 보기 모달). section 은 열 때 보일 구역(기본 모양). */
```

### `frontend/src/screens/SettingsAbout.tsx` — 수정

```diff
--- a/frontend/src/screens/SettingsAbout.tsx
+++ b/frontend/src/screens/SettingsAbout.tsx
@@ -1,10 +1,10 @@
-// 설정 8 · 정보·단축키. 근거: spec/ui/settings.md §8, spec/behavior/shortcuts.md §4, spec/acceptance G6·G7.
+// 설정 · 정보. 근거: spec/ui/settings.md §8, spec/acceptance G6·G7.
+// 단축키 표는 "단축키" 구역(SettingsShortcuts.tsx)으로 옮겼다 (D126).
 // 결정 D63(D8 대체): 참고한 프로젝트(BrightTerm v1.1.0, MIT)의 고지를 "참고한 프로젝트" 필드에 보인다.
 // 결정 D73: 사용 조건(개인 무료 · 업무용 유료) 요약과 전문, 설정된 외부 링크([업무용 라이선스 구매]·[후원하기]·[홈페이지]).
 import { useEffect, useState } from 'react';
 import { ToggleRow } from '../ui/Controls';
 import { LogoMark } from '../ui/Logo';
-import { SHORTCUT_LABELS, shortcutText } from '../lib/shortcuts';
 import { getBackend } from '../ipc/backend';
 import { useApp } from '../state/app';
 import { useUpdate } from '../state/update';
@@ -127,23 +127,6 @@
         <div className="detail-box">{info.dataDir ?? ''}</div>
       </div>
 
-      <div>
-        <div className="set-section-title" style={{ marginBottom: 8 }}>{t('about.shortcuts')}</div>
-        <div className="sc-table">
-          {SHORTCUT_LABELS.map((s) => (
-            <ShortcutRow key={s.name} {...shortcutText(s)} />
-          ))}
-        </div>
-      </div>
-    </>
-  );
-}
-
-function ShortcutRow({ name, keys }: { name: string; keys: string }) {
-  return (
-    <>
-      <span className="sc-name">{name}</span>
-      <span className="kbd">{keys}</span>
     </>
   );
 }
```

### `frontend/src/App.tsx` — 수정

```diff
--- a/frontend/src/App.tsx
+++ b/frontend/src/App.tsx
@@ -7,7 +7,8 @@
 import { useRequests, type UiRequest } from './state/requests';
 import { useWorkspace } from './state/workspace';
 import { subscribeSessionEvents, runShortcut } from './state/actions';
-import { shortcutFor } from './lib/keys';
+import { keyCapture } from './lib/keys';
+import { activeShortcut } from './state/shortcuts';
 import { ContextMenuHost } from './ui/ContextMenu';
 import { DialogHost } from './ui/DialogHost';
 import { Toasts } from './ui/Toasts';
@@ -73,8 +74,10 @@
     const stopImport = startImportSuggestion();
 
     // 단축키: 볼트 잠김·대화상자 열림 조건을 따른다(shortcuts.md §1). 빠른 접속이 열려 있으면 모두 동작.
+    // 키와 사용 여부는 설정(shortcuts·shortcutsEnabled)을 따른다. 설정 화면이 키를 받는 중에는 쉰다 (§5, D126).
     const onKey = (e: KeyboardEvent) => {
-      const action = shortcutFor(e);
+      if (keyCapture.active) return;
+      const action = activeShortcut(e);
       if (!action) return;
       const v = useApp.getState().vault;
       if (v?.initialized && !v.unlocked) return;
```

### `frontend/src/screens/TerminalView.tsx` — 수정

```diff
--- a/frontend/src/screens/TerminalView.tsx
+++ b/frontend/src/screens/TerminalView.tsx
@@ -13,7 +13,7 @@
 import { sessionsIn, findPane, type LayoutNode } from '../lib/layout';
 import { matchesDangerous, LineTracker, pasteLineCount } from '../lib/termGuard';
 import { decodeOsc52 } from '../lib/osc52';
-import { shortcutFor } from '../lib/keys';
+import { activeShortcut } from '../state/shortcuts';
 import { resolveTermTheme } from '../lib/termThemes';
 import { attachHighlighter, type Highlighter } from '../lib/termHighlight';
 import { attachFindMarks, type FindMarks } from '../lib/termFind';
@@ -398,10 +398,11 @@
   });
 
   // 키 처리: 앱 단축키는 터미널로 보내지 않는다(window 캡처에서 처리).
+  // 단축키를 껐거나 그 키를 비웠으면 앱 단축키가 아니므로 터미널(서버)로 간다 (shortcuts.md §5, D126).
   t.attachCustomKeyEventHandler((e) => {
     if (e.type !== 'keydown') return true;
     if (useUi.getState().dialogs.length > 0) return false;
-    if (shortcutFor(e)) return false;
+    if (activeShortcut(e)) return false;
     if (e.ctrlKey && e.shiftKey && e.code === 'KeyC') {
       void copySelection(t);
       return false;
```

### `frontend/src/ui/Modal.tsx` — 수정

```diff
--- a/frontend/src/ui/Modal.tsx
+++ b/frontend/src/ui/Modal.tsx
@@ -3,6 +3,7 @@
 import type { LucideIcon } from 'lucide-react';
 import { X } from 'lucide-react';
 import { t } from '../i18n';
+import { keyCapture } from '../lib/keys';
 
 interface ModalProps {
   title: string;
@@ -36,7 +37,8 @@
   useEffect(() => {
     prevFocus.current = document.activeElement;
     const onKey = (e: KeyboardEvent) => {
-      if (e.key === 'Escape') {
+      // 설정의 단축키 구역이 키를 받는 중이면 Esc 는 그쪽의 취소다 (D126).
+      if (e.key === 'Escape' && !keyCapture.active) {
         // Esc 는 모달이 먼저 가로채고 아래로 전달하지 않는다.
         e.preventDefault();
         e.stopPropagation();
```

### `frontend/src/screens/Welcome.tsx` — 수정

```diff
--- a/frontend/src/screens/Welcome.tsx
+++ b/frontend/src/screens/Welcome.tsx
@@ -1,10 +1,11 @@
 // 시작 화면(탭이 하나도 없을 때 작업 영역). 근거: spec/ui/shell.md §4.
-// 단축키 표(12개)는 behavior/shortcuts.md §4 가 필요하므로 4단계에서 채운다.
+// 단축키 표(12줄)와 안내의 키는 지금 설정의 키를 보인다(behavior/shortcuts.md §4·§5, D126).
 import { Download, Plus, Terminal, Zap } from 'lucide-react';
 import { useApp } from '../state/app';
 import { effectiveColor, recentHosts } from '../lib/hosts';
 import { sidebarSubtitle } from '../lib/tree';
-import { START_SHORTCUTS, shortcutText } from '../lib/shortcuts';
+import { startShortcutRows } from '../lib/shortcuts';
+import { useShortcutLabels } from '../state/shortcuts';
 import { LogoMark } from '../ui/Logo';
 import { openHost, runShortcut } from '../state/actions';
 import { useUi } from '../state/ui';
@@ -16,20 +17,25 @@
   const hosts = store?.hosts ?? [];
   const groups = store?.groups ?? [];
   const recent = recentHosts(hosts);
+  const labels = useShortcutLabels();
+  // 안내 문장의 키: 빠른 연결을 여는 키 중 지금 있는 것. 하나도 없으면(끄거나 비움) 문장을 보이지 않는다.
+  const quickKey = labels.quick || labels.newTab || labels.quickAlt || labels.quickOpen;
 
   return (
     <div className="welcome">
       <div className="welcome-card">
         <LogoMark size={48} />
         <h1>DariTerm</h1>
-        <p className="muted" style={{ margin: 0 }}>
-          {t('welcome.hint').split('{key}')[0]}<span className="kbd">Ctrl+K</span>{t('welcome.hint').split('{key}')[1]}
-        </p>
+        {quickKey && (
+          <p className="muted" style={{ margin: 0 }}>
+            {t('welcome.hint').split('{key}')[0]}<span className="kbd">{quickKey}</span>{t('welcome.hint').split('{key}')[1]}
+          </p>
+        )}
         <div className="welcome-actions">
           <button className="btn primary" onClick={() => useUi.getState().openQuick(null)}>
             <Zap size={15} />{t('welcome.quick')}
           </button>
-          <button className="btn" title="Ctrl+Alt+T" onClick={() => runShortcut('newLocal')}>
+          <button className="btn" title={labels.newLocal || undefined} onClick={() => runShortcut('newLocal')}>
             <Terminal size={15} />{t('host.localTerminal')}
           </button>
           <button className="btn" onClick={() => useUi.getState().openEditor({ kind: 'host', hostId: null, groupId: null })}>
@@ -58,10 +64,10 @@
           </div>
         )}
         <div className="shortcut-table" style={{ gridTemplateColumns: '1fr 1fr', width: '100%', gap: '6px 22px' }}>
-          {START_SHORTCUTS.map((s) => (
+          {startShortcutRows(labels).map((s) => (
             <div key={s.name} style={{ display: 'flex', justifyContent: 'space-between', gap: 8 }}>
-              <span className="muted">{shortcutText(s).name}</span>
-              <span className="kbd">{shortcutText(s).keys}</span>
+              <span className="muted">{s.name}</span>
+              <span className="kbd">{s.keys}</span>
             </div>
           ))}
         </div>
```

### `frontend/src/screens/Panes.tsx` — 수정

```diff
--- a/frontend/src/screens/Panes.tsx
+++ b/frontend/src/screens/Panes.tsx
@@ -15,6 +15,7 @@
 import { isFileDrag, hoverStart, hoverEnd, dropFiles } from './fileDrag';
 // 이 파일은 t 를 지역 변수 이름으로 쓰므로 번역 함수는 tr 로 가져온다(D75).
 import { relabel, t as tr } from '../i18n';
+import { useShortcutLabels } from '../state/shortcuts';
 
 const PANE_MIME = 'application/x-dariterm-pane';
 /** 사이드바 서버 끌어 놓기 MIME(에이전트 B 가 dragstart 에서 설정). */
@@ -122,6 +123,7 @@
 function PaneView({ tab, pane, total }: { tab: Tab; pane: PaneNode; total: number }) {
   const entry = useWorkspace((s) => (pane.sessionId ? s.sessions[pane.sessionId] : undefined));
   const focused = tab.focusedPane === pane.id;
+  const scLabels = useShortcutLabels();
   const info = entry?.info;
   const env = info?.env && info.env !== 'none' ? info.env : undefined;
   const color = info?.color;
@@ -223,11 +225,11 @@
     { separator: true },
     { label: tr('pane.selectAll'), disabled: !hasSession, onSelect: () => selectAllInPane(pane.id) },
     { label: tr('pane.clear'), disabled: !hasSession, onSelect: () => termHandles.get(pane.id)?.clear() },
-    { label: tr('pane.find'), shortcut: 'Ctrl+Shift+F', disabled: !hasSession, onSelect: () => window.dispatchEvent(new CustomEvent(FIND_EVENT, { detail: pane.id })) },
+    { label: tr('pane.find'), shortcut: scLabels.find || undefined, disabled: !hasSession, onSelect: () => window.dispatchEvent(new CustomEvent(FIND_EVENT, { detail: pane.id })) },
     { separator: true },
-    { label: tr('sc.splitRight'), shortcut: 'Ctrl+Shift+D', disabled: !info || !canSplit, onSelect: () => void splitSame('row') },
-    { label: tr('sc.splitDown'), shortcut: 'Ctrl+Shift+E', disabled: !info || !canSplit, onSelect: () => void splitSame('col') },
-    { label: info?.protocol === 'local' ? tr('pane.openFiles') : tr('pane.openSftp'), shortcut: 'Ctrl+Shift+S', disabled: !info?.canSftp && info?.protocol !== 'local', onSelect: () => { focusPane(pane.id); useUi.getState().setSftpOpen(groupOf(tab), true); } },
+    { label: tr('sc.splitRight'), shortcut: scLabels.splitRight || undefined, disabled: !info || !canSplit, onSelect: () => void splitSame('row') },
+    { label: tr('sc.splitDown'), shortcut: scLabels.splitDown || undefined, disabled: !info || !canSplit, onSelect: () => void splitSame('col') },
+    { label: info?.protocol === 'local' ? tr('pane.openFiles') : tr('pane.openSftp'), shortcut: scLabels.sftp || undefined, disabled: !info?.canSftp && info?.protocol !== 'local', onSelect: () => { focusPane(pane.id); useUi.getState().setSftpOpen(groupOf(tab), true); } },
     { label: tr('pane.reconnect'), disabled: !hasSession || entry?.info.state === 'connected', onSelect: () => void reconnect() },
     { separator: true },
     { label: tr('pane.close'), danger: true, onSelect: () => void closePane(pane.id) },
```

### `frontend/src/screens/TabBar.tsx` — 수정

```diff
--- a/frontend/src/screens/TabBar.tsx
+++ b/frontend/src/screens/TabBar.tsx
@@ -8,6 +8,7 @@
 import { focusActiveTerminal } from './TerminalView';
 // 이 파일은 t 를 탭 변수 이름으로 많이 쓰므로 번역 함수는 tr 로 가져온다(D75).
 import { t as tr } from '../i18n';
+import { useShortcutLabels } from '../state/shortcuts';
 
 export const TAB_MIME = 'application/x-dariterm-tab';
 
@@ -48,6 +49,7 @@
 /** 탭 막대. group 을 주면 그 그룹의 탭만 보이는 열 안 탭 막대(화면 나누기, ui/shell.md §2.3). */
 export function TabBar({ group }: { group?: 0 | 1 }) {
   const allTabs = useWorkspace((s) => s.tabs);
+  const scLabels = useShortcutLabels();
   const g: 0 | 1 = group ?? 0;
   const split = isSplit(allTabs);
   const tabs = allTabs.filter((t) => groupOf(t) === g);
@@ -105,11 +107,11 @@
         ? { label: tr('tab.toRight'), disabled: allTabs.length < 2, onSelect: () => moveTabToGroup(t.id, 1) }
         : { label: tr('tab.toLeft'), onSelect: () => moveTabToGroup(t.id, 0) },
       ...(split ? [{ label: tr('tab.merge'), onSelect: () => mergeGroups() } as MenuItem] : []),
-      { label: t.broadcast ? tr('tab.broadcastOff') : tr('tab.broadcastOn'), shortcut: 'Ctrl+Shift+B', disabled: panes < 2, onSelect: () => toggleBroadcast(t.id) },
+      { label: t.broadcast ? tr('tab.broadcastOff') : tr('tab.broadcastOn'), shortcut: scLabels.broadcast || undefined, disabled: panes < 2, onSelect: () => toggleBroadcast(t.id) },
       { label: tr('tab.detach'), disabled: panes < 2, onSelect: () => detachPanes(t.id) },
       { separator: true },
       { label: tr('tab.closeOthers'), onSelect: () => void closeOthers(t) },
-      { label: tr('tab.close'), shortcut: 'Ctrl+Shift+W', danger: true, onSelect: () => void confirmClose(t) },
+      { label: tr('tab.close'), shortcut: scLabels.closePane || undefined, danger: true, onSelect: () => void confirmClose(t) },
     ];
   };
 
@@ -242,7 +244,7 @@
       </button>
       <button
         className={`icon-btn tab-sftp${sftpOpen ? ' on' : ''}`}
-        title="SFTP (Ctrl+Shift+S)"
+        title={scLabels.sftp ? `SFTP (${scLabels.sftp})` : 'SFTP'}
         aria-label="SFTP"
         aria-pressed={sftpOpen}
         onClick={() => toggleSftp(g)}
```

### `frontend/src/ipc/types.ts` — 수정

```diff
--- a/frontend/src/ipc/types.ts
+++ b/frontend/src/ipc/types.ts
@@ -128,6 +128,10 @@
   sessionLogTimestamp: boolean;
   /** 세션 로그에 ANSI 시퀀스(색·커서 이동)와 CR 을 지우지 않고 받은 그대로 남긴다 (behavior/session.md §6, D120). */
   sessionLogAnsi: boolean;
+  /** 앱 단축키 전체를 쓸지. 끄면 그 키들은 터미널(서버)로 간다 (behavior/shortcuts.md §5, D126). */
+  shortcutsEnabled: boolean;
+  /** 기본값에서 바꾼 단축키만: 명령 id → 조합('Ctrl+Shift+KeyT'). 빈 글자는 키 없음 (behavior/shortcuts.md §5). */
+  shortcuts: Record<string, string>;
   bell: boolean;
   shiftEnterNewline: boolean;
   rememberWindow: boolean;
```

### `frontend/src/lib/defaults.ts` — 수정

```diff
--- a/frontend/src/lib/defaults.ts
+++ b/frontend/src/lib/defaults.ts
@@ -47,6 +47,8 @@
   sessionLogSyncSec: 2,
   sessionLogTimestamp: false,
   sessionLogAnsi: false,
+  shortcutsEnabled: true,
+  shortcuts: {},
   bell: false,
   shiftEnterNewline: true,
   rememberWindow: true,
```

### `frontend/src/styles/settings.css` — 수정

```diff
--- a/frontend/src/styles/settings.css
+++ b/frontend/src/styles/settings.css
@@ -43,6 +43,20 @@
 .sc-table { display: grid; grid-template-columns: 1fr auto 1fr auto; gap: 6px 14px; align-items: center; }
 .sc-table .sc-name { color: var(--fg-2); font-size: 12.5px; white-space: nowrap; }
 .sc-table .kbd { justify-self: end; }
+.sc-head { display: flex; align-items: center; justify-content: space-between; gap: 10px; margin-bottom: 6px; }
+.sc-edit { border: 1px solid var(--border); border-radius: 10px; padding: 2px 10px 2px 14px; }
+.sc-edit.off .sc-name, .sc-edit.off .sc-key { opacity: 0.5; }
+.sc-edit-row { display: grid; grid-template-columns: 1fr auto auto auto; gap: 4px; align-items: center; padding: 4px 0; }
+.sc-edit-row + .sc-edit-row { border-top: 1px solid var(--border); }
+.sc-edit-row .sc-name { color: var(--fg); font-size: 12.5px; }
+.sc-key {
+  min-width: 132px; height: 26px; padding: 0 10px; border-radius: 6px; cursor: pointer; text-align: center;
+  font-family: var(--mono); font-size: 11.5px; background: var(--panel-2); border: 1px solid var(--border); color: var(--fg);
+}
+.sc-key:hover { border-color: var(--border-strong); }
+.sc-key.none { color: var(--fg-3); font-family: inherit; }
+.sc-key.rec { border-color: var(--accent); color: var(--accent); font-family: inherit; }
+.sc-edit-row .icon-btn:disabled { opacity: 0.25; cursor: default; }
 .about-head { display: flex; gap: 14px; align-items: center; }
 .about-head .logo-mark { width: 44px; height: 44px; border-radius: 10px; }
 .about-head .name { font-size: 16px; font-weight: 700; }
```

### `internal/store/store.go` — 수정

```diff
--- a/internal/store/store.go
+++ b/internal/store/store.go
@@ -55,7 +55,9 @@
 	"sessionLogKeepDays":  30,
 	"sessionLogSyncSec":   2,
 	"sessionLogTimestamp": false,
-	"sessionLogAnsi":      false, // 세션 로그에 ANSI 시퀀스를 그대로 남긴다 (session.md §6, D120)
+	"sessionLogAnsi":      false,            // 세션 로그에 ANSI 시퀀스를 그대로 남긴다 (session.md §6, D120)
+	"shortcutsEnabled":    true,             // 앱 단축키 전체 사용 (shortcuts.md §5, D126)
+	"shortcuts":           map[string]any{}, // 기본값에서 바꾼 단축키만: 명령 id → 조합, 빈 글자는 키 없음
 	"bell":                false,
 	"shiftEnterNewline":   true,
 	"rememberWindow":      true,
```


## C. 테스트 코드

### `frontend/src/lib/keys.test.ts` — 새 파일

```ts
// 단축키 규칙(D126): 기본 키는 전과 같고, 키를 바꾸면 그 키로만 동작하며, 겹치면 먼저 쓰던 명령이 비워지고, 전체를 끄면 아무것도 걸리지 않는다.
// 근거: spec/behavior/shortcuts.md §2·§5.
import { describe, expect, it } from 'vitest';
import {
  SHORTCUT_COMMANDS, assignShortcut, buildKeyMap, comboOf, comboProblem, comboText, effectiveKeys, shortcutFor, type KeyLike,
} from './keys';

function key(init: Partial<KeyLike>): KeyLike {
  return { code: '', key: '', ctrlKey: false, shiftKey: false, altKey: false, ...init };
}
const ctrl = (code: string) => key({ code, ctrlKey: true });
const ctrlShift = (code: string) => key({ code, ctrlKey: true, shiftKey: true });

describe('기본 단축키 (shortcuts.md §2)', () => {
  it('표의 키가 전과 같은 동작에 걸린다', () => {
    const cases: [KeyLike, string, unknown?][] = [
      [ctrl('KeyK'), 'quickToggle'],
      [ctrlShift('KeyT'), 'quickToggle'],
      [ctrlShift('KeyN'), 'quickToggle'],
      [ctrlShift('KeyP'), 'quickOpen'],
      [key({ code: 'KeyT', ctrlKey: true, altKey: true }), 'newLocal'],
      [ctrlShift('KeyW'), 'closePane'],
      [ctrlShift('KeyD'), 'splitRight'],
      [ctrlShift('KeyE'), 'splitDown'],
      [ctrlShift('Enter'), 'zoom'],
      [ctrlShift('NumpadEnter'), 'zoom'],
      [ctrlShift('KeyF'), 'find'],
      [ctrlShift('KeyS'), 'sftp'],
      [ctrlShift('KeyB'), 'broadcast'],
      [ctrlShift('KeyL'), 'sidebar'],
      [ctrlShift('ArrowLeft'), 'tabLeft'],
      [ctrlShift('ArrowRight'), 'tabRight'],
      [ctrl('Tab'), 'tabNext'],
      [ctrlShift('Tab'), 'tabPrev'],
      [ctrl('Digit1'), 'tabIndex', 0],
      [ctrl('Digit8'), 'tabIndex', 7],
      [ctrl('Numpad3'), 'tabIndex', 2],
      [ctrl('Digit9'), 'tabIndex', 'last'],
      [key({ code: 'ArrowUp', altKey: true }), 'focusMove', 'up'],
      [key({ code: 'ArrowRight', altKey: true }), 'focusMove', 'right'],
      [ctrl('Equal'), 'fontUp'],
      [ctrl('NumpadAdd'), 'fontUp'],
      [ctrl('Minus'), 'fontDown'],
      [ctrl('NumpadSubtract'), 'fontDown'],
      [ctrl('Digit0'), 'fontReset'],
      [key({ code: 'F11' }), 'fullscreen'],
    ];
    for (const [e, action, arg] of cases) {
      const hit = shortcutFor(e);
      expect(hit?.action, comboOf(e)).toBe(action);
      if (arg !== undefined) expect(hit?.arg, comboOf(e)).toBe(arg);
    }
  });

  it('표에 없는 조합은 걸리지 않는다(터미널이 받는다)', () => {
    for (const e of [ctrl('KeyA'), ctrl('KeyC'), ctrl('KeyV'), ctrlShift('KeyK'), ctrlShift('KeyC'), ctrlShift('Digit1'), key({ code: 'KeyK' }), key({ code: 'ArrowUp', altKey: true, shiftKey: true })]) {
      expect(shortcutFor(e), comboOf(e)).toBeNull();
    }
  });

  it('조합 키만 눌렀거나 Win 키가 끼면 조합이 아니다', () => {
    expect(comboOf(key({ code: 'ControlLeft', ctrlKey: true }))).toBe('');
    expect(comboOf(key({ code: 'ShiftRight', shiftKey: true, ctrlKey: true }))).toBe('');
    expect(comboOf(key({ code: 'KeyK', ctrlKey: true, metaKey: true }))).toBe('');
    expect(comboOf(key({ code: 'KeyK', ctrlKey: true, altKey: true, shiftKey: true }))).toBe('Ctrl+Alt+Shift+KeyK');
  });

  it('기본 키는 서로 겹치지 않고 모두 고를 수 있는 조합이다', () => {
    const keys = SHORTCUT_COMMANDS.map((c) => c.keys);
    expect(new Set(keys).size).toBe(keys.length);
    expect(new Set(SHORTCUT_COMMANDS.map((c) => c.id)).size).toBe(SHORTCUT_COMMANDS.length);
    for (const c of SHORTCUT_COMMANDS) expect(comboProblem(c.keys), c.id).toBeNull();
    expect(SHORTCUT_COMMANDS).toHaveLength(34);
  });
});

describe('화면 표기', () => {
  it('조합 글자를 읽기 쉬운 표기로', () => {
    expect(comboText('Ctrl+Shift+KeyT')).toBe('Ctrl+Shift+T');
    expect(comboText('Ctrl+Digit9')).toBe('Ctrl+9');
    expect(comboText('Alt+ArrowUp')).toBe('Alt+↑');
    expect(comboText('Ctrl+Equal')).toBe('Ctrl+=');
    expect(comboText('Ctrl+Shift+Enter')).toBe('Ctrl+Shift+Enter');
    expect(comboText('F11')).toBe('F11');
    expect(comboText('')).toBe('');
  });
});

describe('고를 수 없는 조합 (shortcuts.md §5)', () => {
  it('Ctrl·Alt 없이 누른 보통 키는 안 되고 F 키는 된다', () => {
    expect(comboProblem('KeyA')).toBe('modifier');
    expect(comboProblem('Shift+KeyA')).toBe('modifier');
    expect(comboProblem('Shift+Enter')).toBe('modifier');
    expect(comboProblem('F5')).toBeNull();
    expect(comboProblem('Shift+F12')).toBeNull();
    expect(comboProblem('Alt+KeyX')).toBeNull();
    expect(comboProblem('Ctrl+KeyB')).toBeNull();
  });

  it('터미널의 복사·붙여넣기 키는 안 된다', () => {
    for (const c of ['Ctrl+KeyV', 'Ctrl+Shift+KeyV', 'Ctrl+Shift+KeyC', 'Ctrl+Alt+KeyV', 'Shift+Insert']) expect(comboProblem(c), c).toBe('reserved');
  });
});

describe('키 바꾸기 (shortcuts.md §5)', () => {
  it('바꾸면 새 키로만 동작하고 옛 키는 풀린다', () => {
    const { overrides, displaced } = assignShortcut({}, 'find', 'Ctrl+Alt+KeyF');
    expect(overrides).toEqual({ find: 'Ctrl+Alt+KeyF' });
    expect(displaced).toEqual([]);
    const map = buildKeyMap(true, overrides);
    expect(shortcutFor(key({ code: 'KeyF', ctrlKey: true, altKey: true }), map)?.action).toBe('find');
    expect(shortcutFor(ctrlShift('KeyF'), map)).toBeNull();
    expect(shortcutFor(ctrl('KeyK'), map)?.action).toBe('quickToggle');
  });

  it('다른 명령이 쓰던 키를 고르면 먼저 쓰던 명령을 빈 값으로 저장한다', () => {
    const { overrides, displaced } = assignShortcut({}, 'find', 'Ctrl+KeyK');
    expect(overrides).toEqual({ find: 'Ctrl+KeyK', quick: '' });
    expect(displaced).toEqual(['quick']);
    const keys = effectiveKeys(overrides);
    expect(keys.find).toBe('Ctrl+KeyK');
    expect(keys.quick).toBe('');
    const map = buildKeyMap(true, overrides);
    expect(shortcutFor(ctrl('KeyK'), map)?.id).toBe('find');
  });

  it('이미 바꿔 둔 키끼리 겹쳐도 먼저 쓰던 쪽이 비워진다', () => {
    const a = assignShortcut({}, 'find', 'Ctrl+Alt+KeyX').overrides;
    const b = assignShortcut(a, 'sftp', 'Ctrl+Alt+KeyX');
    expect(b.overrides).toEqual({ find: '', sftp: 'Ctrl+Alt+KeyX' });
    expect(b.displaced).toEqual(['find']);
  });

  it('지우면 빈 값, 기본값으로 되돌리면 설정에서 빠진다', () => {
    const cleared = assignShortcut({}, 'quick', '').overrides;
    expect(cleared).toEqual({ quick: '' });
    expect(shortcutFor(ctrl('KeyK'), buildKeyMap(true, cleared))).toBeNull();
    expect(assignShortcut(cleared, 'quick', 'Ctrl+KeyK').overrides).toEqual({});
  });

  it('기본값으로 되돌릴 때 그 키를 다른 명령이 쓰고 있으면 그 명령이 비워진다', () => {
    const taken = assignShortcut({}, 'find', 'Ctrl+KeyK').overrides; // { find: Ctrl+K, quick: '' }
    const back = assignShortcut(taken, 'quick', 'Ctrl+KeyK');
    expect(back.overrides).toEqual({ find: '' });
    expect(back.displaced).toEqual(['find']);
  });

  it('모르는 명령·글자가 아닌 값은 버리고, 손으로 겹치게 고친 파일은 표의 앞 명령이 이긴다', () => {
    expect(assignShortcut({ nope: 'Ctrl+KeyQ', find: 3 as unknown as string }, 'sftp', 'Ctrl+Alt+KeyS').overrides).toEqual({ sftp: 'Ctrl+Alt+KeyS' });
    const keys = effectiveKeys({ find: 'Ctrl+KeyK' }); // quick 의 기본 키와 겹친다
    expect(keys.quick).toBe('Ctrl+KeyK');
    expect(keys.find).toBe('');
  });
});

describe('전체 끄기 (shortcuts.md §5)', () => {
  it('끄면 어떤 키도 앱 단축키가 아니다', () => {
    const off = buildKeyMap(false, { find: 'Ctrl+Alt+KeyF' });
    expect(off.size).toBe(0);
    for (const c of SHORTCUT_COMMANDS) {
      const parts = c.keys.split('+');
      const code = parts.pop()!;
      const e = key({ code, ctrlKey: parts.includes('Ctrl'), altKey: parts.includes('Alt'), shiftKey: parts.includes('Shift') });
      expect(shortcutFor(e), c.id).not.toBeNull();
      expect(shortcutFor(e, off), c.id).toBeNull();
    }
  });
});
```

### `frontend/src/screens/SettingsShortcuts.test.tsx` — 새 파일

```tsx
// 설정의 "단축키" 구역(D126): 정보에서 떼어 낸 구역, 전체 켜기/끄기, 키 바꾸기(겹치면 먼저 것을 빈 값으로 저장), 기본값으로.
// 근거: spec/ui/settings.md §2a·§8, spec/behavior/shortcuts.md §5.
import { beforeEach, describe, expect, it, vi } from 'vitest';
import { fireEvent, render, screen, waitFor, within } from '@testing-library/react';
import { setBackend, type Backend } from '../ipc/backend';
import { useApp } from '../state/app';
import { useUi } from '../state/ui';
import { DEFAULT_SETTINGS } from '../lib/defaults';
import { keyCapture } from '../lib/keys';
import { startShortcutRows } from '../lib/shortcuts';
import { activeShortcut } from '../state/shortcuts';
import type { Settings as SettingsT, Store } from '../ipc/types';
import { Settings } from './Settings';
import { Welcome } from './Welcome';

const invoke = vi.fn(async () => undefined);

function setStore(settings: Partial<SettingsT> = {}) {
  useApp.setState({ store: { version: 1, groups: [], hosts: [], snippets: [], knownHosts: [], settings: { ...DEFAULT_SETTINGS, ...settings } } as Store });
}

beforeEach(() => {
  invoke.mockClear();
  keyCapture.active = false;
  setBackend({ invoke, termInput: vi.fn(), termResize: vi.fn(), reply: vi.fn(), on: () => () => undefined } as unknown as Backend);
  useApp.setState({ toasts: [] });
  useUi.setState({ settings: { open: false, section: 'appearance' } });
  setStore();
});

function openSection(section: 'shortcuts' | 'about') {
  useUi.setState({ settings: { open: true, section } });
  return render(<Settings />);
}

/** store.patch 로 보낸 설정 값들(보낸 순서대로). */
function patched(key: string): unknown[] {
  return invoke.mock.calls
    .filter((c) => (c as unknown[])[0] === 'store.patch')
    .map((c) => ((c as unknown[])[1] as [{ settings: Record<string, unknown> }])[0].settings[key]);
}

/** 그 명령의 키 칸(이름: 키). */
const keyBox = (name: string, keys: string) => screen.getByRole('button', { name: `${name}: ${keys}` });

describe('설정 > 단축키 (D126)', () => {
  it('구역 목록에 "입력" 다음으로 "단축키"가 있다 (10개)', () => {
    openSection('shortcuts');
    const names = within(screen.getByRole('navigation')).getAllByRole('button').map((b) => b.textContent);
    expect(names).toEqual(['화면', '입력', '단축키', '파일 전송', '연결', '로그', 'git 동기화', '보안', '백업', '정보']);
  });

  it('"정보"에는 단축키 표가 없다', async () => {
    invoke.mockImplementation((async (ch: string) => (ch === 'app.info' ? { version: '1.0.0', dataDir: 'C:\\data' } : undefined)) as never);
    openSection('about');
    await screen.findByText('데이터 폴더');
    expect(screen.queryByText('터미널에서 찾기')).toBeNull();
    expect(screen.queryByText('Ctrl+Shift+F')).toBeNull();
    invoke.mockImplementation(async () => undefined);
  });

  it('명령 34개가 기본 키와 함께 보이고, 터미널 안의 고정 키 3개는 따로 보인다', () => {
    const { container } = openSection('shortcuts');
    expect(container.querySelectorAll('.sc-edit-row')).toHaveLength(34);
    expect(keyBox('빠른 연결 / 명령', 'Ctrl+K')).toBeTruthy();
    expect(keyBox('이전 탭', 'Ctrl+Shift+Tab')).toBeTruthy();
    expect(keyBox('3번째 탭', 'Ctrl+3')).toBeTruthy();
    expect(keyBox('위쪽 창으로', 'Alt+↑')).toBeTruthy();
    expect(keyBox('창 최대화/복원', 'F11')).toBeTruthy();
    expect(screen.getByText('터미널 안에서 쓰는 키')).toBeTruthy();
    expect(screen.getByText('Ctrl+V')).toBeTruthy();
    expect(screen.getByText('Ctrl+Alt+V')).toBeTruthy();
    // 기본 상태에서는 되돌릴 것이 없다.
    expect((screen.getByRole('button', { name: '모두 기본값으로' }) as HTMLButtonElement).disabled).toBe(true);
  });

  it('"단축키 사용"은 기본 켜짐이고, 끄면 shortcutsEnabled: false 로 저장된다', async () => {
    expect(DEFAULT_SETTINGS.shortcutsEnabled).toBe(true);
    openSection('shortcuts');
    const sw = screen.getByRole('switch', { name: '단축키 사용' });
    expect(sw.getAttribute('aria-checked')).toBe('true');
    fireEvent.click(sw);
    await waitFor(() => expect(patched('shortcutsEnabled')).toEqual([false]));
  });

  it('키 칸을 누르고 새 키를 누르면 그 명령만 바뀌어 저장된다', async () => {
    openSection('shortcuts');
    fireEvent.click(keyBox('터미널에서 찾기', 'Ctrl+Shift+F'));
    expect(keyBox('터미널에서 찾기', '키를 누르세요…')).toBeTruthy();
    expect(keyCapture.active).toBe(true);
    // 조합 키만 누른 것은 기다린다.
    fireEvent.keyDown(window, { code: 'ControlLeft', key: 'Control', ctrlKey: true });
    expect(patched('shortcuts')).toEqual([]);
    fireEvent.keyDown(window, { code: 'KeyF', key: 'f', ctrlKey: true, altKey: true });
    await waitFor(() => expect(patched('shortcuts')).toEqual([{ find: 'Ctrl+Alt+KeyF' }]));
    expect(keyCapture.active).toBe(false);
  });

  it('다른 명령이 쓰던 키를 고르면 먼저 쓰던 명령을 빈 값으로 저장하고 알린다', async () => {
    openSection('shortcuts');
    fireEvent.click(keyBox('터미널에서 찾기', 'Ctrl+Shift+F'));
    fireEvent.keyDown(window, { code: 'KeyK', key: 'k', ctrlKey: true });
    await waitFor(() => expect(patched('shortcuts')).toEqual([{ find: 'Ctrl+KeyK', quick: '' }]));
    expect(useApp.getState().toasts.map((x) => x.text)).toContain('Ctrl+K 를 쓰던 "빠른 연결 / 명령"의 단축키를 비웠습니다');
  });

  it('비워진 명령은 "없음"으로 보이고, 기본값으로 누르면 되찾으며 그 키를 쓰던 명령이 비워진다', async () => {
    setStore({ shortcuts: { find: 'Ctrl+KeyK', quick: '' } });
    openSection('shortcuts');
    expect(keyBox('빠른 연결 / 명령', '없음')).toBeTruthy();
    expect(keyBox('터미널에서 찾기', 'Ctrl+K')).toBeTruthy();
    fireEvent.click(screen.getByRole('button', { name: '빠른 연결 / 명령: 기본값으로' }));
    await waitFor(() => expect(patched('shortcuts')).toEqual([{ find: '' }]));
  });

  it('[키 지우기]는 빈 값으로 저장한다', async () => {
    setStore({ shortcuts: { sftp: 'Ctrl+Alt+KeyS' } });
    openSection('shortcuts');
    fireEvent.click(screen.getByRole('button', { name: 'SFTP 창: 키 지우기' }));
    await waitFor(() => expect(patched('shortcuts')).toEqual([{ sftp: '' }]));
  });

  it('[모두 기본값으로]는 빈 표로 저장한다', async () => {
    // (가짜 백엔드는 저장 뒤 설정을 다시 돌려주지 않으므로 따로 연다.)
    setStore({ shortcuts: { sftp: 'Ctrl+Alt+KeyS' } });
    openSection('shortcuts');
    fireEvent.click(screen.getByRole('button', { name: '모두 기본값으로' }));
    await waitFor(() => expect(patched('shortcuts')).toEqual([{}]));
  });

  it('고를 수 없는 키는 저장하지 않고 까닭을 알린 뒤 계속 기다린다', () => {
    openSection('shortcuts');
    fireEvent.click(keyBox('터미널에서 찾기', 'Ctrl+Shift+F'));
    fireEvent.keyDown(window, { code: 'KeyA', key: 'a' });
    fireEvent.keyDown(window, { code: 'KeyV', key: 'v', ctrlKey: true });
    expect(patched('shortcuts')).toEqual([]);
    const texts = useApp.getState().toasts.map((x) => x.text);
    expect(texts).toContain('Ctrl 또는 Alt 와 함께 누르는 키나 F1~F12 만 쓸 수 있습니다');
    expect(texts).toContain('Ctrl+V 는 터미널의 복사·붙여넣기에 쓰는 키라 고를 수 없습니다');
    expect(keyBox('터미널에서 찾기', '키를 누르세요…')).toBeTruthy();
  });

  it('키를 받는 중의 Esc 는 받기만 취소하고 설정 창은 닫지 않는다', () => {
    openSection('shortcuts');
    fireEvent.click(keyBox('터미널에서 찾기', 'Ctrl+Shift+F'));
    fireEvent.keyDown(window, { code: 'Escape', key: 'Escape' });
    expect(useUi.getState().settings.open).toBe(true);
    expect(keyBox('터미널에서 찾기', 'Ctrl+Shift+F')).toBeTruthy();
    expect(patched('shortcuts')).toEqual([]);
    // 받는 중이 아니면 Esc 는 전처럼 설정을 닫는다.
    fireEvent.keyDown(window, { code: 'Escape', key: 'Escape' });
    expect(useUi.getState().settings.open).toBe(false);
  });
});

describe('저장된 단축키 설정이 키 처리에 쓰인다 (D126)', () => {
  const ev = (code: string, mods: { ctrlKey?: boolean; shiftKey?: boolean; altKey?: boolean }) =>
    ({ code, key: '', ctrlKey: false, shiftKey: false, altKey: false, ...mods });

  it('기본 설정이면 기본 키가 앱 단축키다', () => {
    expect(activeShortcut(ev('KeyK', { ctrlKey: true }))?.action).toBe('quickToggle');
  });

  it('전체를 끄면 어떤 키도 앱 단축키가 아니다(터미널이 받는다)', () => {
    setStore({ shortcutsEnabled: false });
    expect(activeShortcut(ev('KeyK', { ctrlKey: true }))).toBeNull();
    expect(activeShortcut(ev('F11', {}))).toBeNull();
    expect(activeShortcut(ev('ArrowUp', { altKey: true }))).toBeNull();
  });

  it('바꾼 키는 새 키로만, 비운 명령은 어떤 키로도 동작하지 않는다', () => {
    setStore({ shortcuts: { find: 'Ctrl+KeyK', quick: '' } });
    expect(activeShortcut(ev('KeyK', { ctrlKey: true }))?.action).toBe('find');
    expect(activeShortcut(ev('KeyF', { ctrlKey: true, shiftKey: true }))).toBeNull();
  });

  it('옛 저장 파일처럼 값이 없으면 켜진 것으로 본다', () => {
    setStore({ shortcutsEnabled: undefined as unknown as boolean, shortcuts: undefined as unknown as Record<string, string> });
    expect(activeShortcut(ev('KeyK', { ctrlKey: true }))?.action).toBe('quickToggle');
  });
});

describe('시작 화면의 단축키 표는 지금 키를 보인다 (D126)', () => {
  it('기본: 12줄, 묶음 줄은 묶음 표기', () => {
    render(<Welcome />);
    const rows = [...document.querySelectorAll('.shortcut-table > div')].map((d) => d.textContent);
    expect(rows).toHaveLength(12);
    expect(rows).toContain('창 이동Alt+방향키');
    expect(rows).toContain('탭 이동Ctrl+Tab / Ctrl+1~9');
    expect(rows).toContain('빠른 연결 / 명령Ctrl+K');
    expect(screen.getAllByText('Ctrl+K').length).toBe(2); // 안내 문장 + 표
  });

  it('바꾼 키를 보이고, 비운 줄은 뺀다', () => {
    setStore({ shortcuts: { quick: 'Ctrl+Alt+KeyK', newLocal: '', focusUp: 'Ctrl+Alt+ArrowUp' } });
    render(<Welcome />);
    const rows = [...document.querySelectorAll('.shortcut-table > div')].map((d) => d.textContent);
    expect(rows).toContain('빠른 연결 / 명령Ctrl+Alt+K');
    expect(rows.some((r) => r?.startsWith('새 로컬 터미널'))).toBe(false);
    expect(rows).toContain('창 이동Ctrl+Alt+↑ …');
  });

  it('전체를 끄면 고정 키 3줄만 남고 안내 문장의 키도 없다', () => {
    expect(startShortcutRows({}).map((r) => r.keys)).toEqual(['Ctrl+V', 'Ctrl+Alt+V', 'Ctrl+Shift+C / 선택']);
    setStore({ shortcutsEnabled: false });
    render(<Welcome />);
    expect(document.querySelectorAll('.shortcut-table > div')).toHaveLength(3);
    expect(screen.queryByText('Ctrl+K')).toBeNull();
  });
});
```

### `frontend/src/screens/SettingsLog.test.tsx` — 수정

```diff
--- a/frontend/src/screens/SettingsLog.test.tsx
+++ b/frontend/src/screens/SettingsLog.test.tsx
@@ -33,10 +33,10 @@
 }
 
 describe('설정 > 로그 (D120)', () => {
-  it('구역 목록에 "연결" 다음으로 "로그"가 있다 (9개)', () => {
+  it('구역 목록에 "연결" 다음으로 "로그"가 있다 (10개)', () => {
     openSection('connection');
     const names = within(screen.getByRole('navigation')).getAllByRole('button').map((b) => b.textContent);
-    expect(names).toEqual(['화면', '입력', '파일 전송', '연결', '로그', 'git 동기화', '보안', '백업', '정보']);
+    expect(names).toEqual(['화면', '입력', '단축키', '파일 전송', '연결', '로그', 'git 동기화', '보안', '백업', '정보']);
   });
 
   it('"연결"에는 로그 항목이 없고 자동 재연결·배치 복원만 남는다', () => {
```


## D. 계약 고정 파일 (백엔드 응답과 프런트 타입을 맞춰 보는 테스트가 읽는다)

### `frontend/src/ipc/contract/store.get.json` — 수정

```diff
--- a/frontend/src/ipc/contract/store.get.json
+++ b/frontend/src/ipc/contract/store.get.json
@@ -46,6 +46,8 @@
     "sessionLogSyncSec": 2,
     "sessionLogTimestamp": false,
     "shiftEnterNewline": true,
+    "shortcuts": {},
+    "shortcutsEnabled": true,
     "sidebarWidth": 260,
     "syncAuto": true,
     "syncRepo": "",
```

### `frontend/src/ipc/contract/store.get.afterNullPatch.json` — 수정

```diff
--- a/frontend/src/ipc/contract/store.get.afterNullPatch.json
+++ b/frontend/src/ipc/contract/store.get.afterNullPatch.json
@@ -46,6 +46,8 @@
     "sessionLogSyncSec": 2,
     "sessionLogTimestamp": false,
     "shiftEnterNewline": true,
+    "shortcuts": {},
+    "shortcutsEnabled": true,
     "sidebarWidth": 260,
     "syncAuto": true,
     "syncRepo": "",
```


## E. 문구

`locales/<언어>.json` 10개 파일에 같은 키를 넣는다(아래 줄의 순서: ko, en, ja, zh, tw, de, fr, es, pt, ru). 한국어·영어 외에는 이 작업을 한 세션이 번역한 것이라 검수가 필요할 수 있다.

넣는 자리: `settings.sec.shortcuts` 는 `settings.sec.input` 바로 다음 줄, 나머지 `sc.*` 30개는 지운 `sc.fontSize` 의 자리(`sc.sidebar` 다음)에 아래 순서대로. 있던 문구는 고치지 않았다.

지운 키(2개, 쓰는 곳이 없어졌다): `sc.fontSize`("글자 크기"), `about.shortcuts`("단축키")

**`sc.quickAlt`** (추가)

```json
  "sc.quickAlt": "빠른 연결 (보조 키)",
  "sc.quickAlt": "Quick connect (alternate key)",
  "sc.quickAlt": "クイック接続（予備キー）",
  "sc.quickAlt": "快速连接（备用键）",
  "sc.quickAlt": "快速連線（備用鍵）",
  "sc.quickAlt": "Schnellverbindung (Zweittaste)",
  "sc.quickAlt": "Connexion rapide (touche secondaire)",
  "sc.quickAlt": "Conexión rápida (tecla alternativa)",
  "sc.quickAlt": "Conexão rápida (tecla alternativa)",
  "sc.quickAlt": "Быстрое подключение (доп. клавиша)",
```

**`sc.quickOpen`** (추가)

```json
  "sc.quickOpen": "빠른 연결 열기 (닫지 않음)",
  "sc.quickOpen": "Open quick connect (never closes)",
  "sc.quickOpen": "クイック接続を開く（閉じない）",
  "sc.quickOpen": "打开快速连接（不关闭）",
  "sc.quickOpen": "開啟快速連線（不關閉）",
  "sc.quickOpen": "Schnellverbindung öffnen (schließt nicht)",
  "sc.quickOpen": "Ouvrir la connexion rapide (sans fermer)",
  "sc.quickOpen": "Abrir conexión rápida (sin cerrar)",
  "sc.quickOpen": "Abrir conexão rápida (sem fechar)",
  "sc.quickOpen": "Открыть быстрое подключение (не закрывает)",
```

**`sc.focusUp`** (추가)

```json
  "sc.focusUp": "위쪽 창으로",
  "sc.focusUp": "Focus pane above",
  "sc.focusUp": "上のペインへ",
  "sc.focusUp": "切换到上方窗格",
  "sc.focusUp": "切換到上方窗格",
  "sc.focusUp": "Zum Bereich darüber",
  "sc.focusUp": "Volet du dessus",
  "sc.focusUp": "Panel de arriba",
  "sc.focusUp": "Painel acima",
  "sc.focusUp": "К панели выше",
```

**`sc.focusDown`** (추가)

```json
  "sc.focusDown": "아래쪽 창으로",
  "sc.focusDown": "Focus pane below",
  "sc.focusDown": "下のペインへ",
  "sc.focusDown": "切换到下方窗格",
  "sc.focusDown": "切換到下方窗格",
  "sc.focusDown": "Zum Bereich darunter",
  "sc.focusDown": "Volet du dessous",
  "sc.focusDown": "Panel de abajo",
  "sc.focusDown": "Painel abaixo",
  "sc.focusDown": "К панели ниже",
```

**`sc.focusLeft`** (추가)

```json
  "sc.focusLeft": "왼쪽 창으로",
  "sc.focusLeft": "Focus pane to the left",
  "sc.focusLeft": "左のペインへ",
  "sc.focusLeft": "切换到左侧窗格",
  "sc.focusLeft": "切換到左側窗格",
  "sc.focusLeft": "Zum Bereich links",
  "sc.focusLeft": "Volet de gauche",
  "sc.focusLeft": "Panel de la izquierda",
  "sc.focusLeft": "Painel à esquerda",
  "sc.focusLeft": "К панели слева",
```

**`sc.focusRight`** (추가)

```json
  "sc.focusRight": "오른쪽 창으로",
  "sc.focusRight": "Focus pane to the right",
  "sc.focusRight": "右のペインへ",
  "sc.focusRight": "切换到右侧窗格",
  "sc.focusRight": "切換到右側窗格",
  "sc.focusRight": "Zum Bereich rechts",
  "sc.focusRight": "Volet de droite",
  "sc.focusRight": "Panel de la derecha",
  "sc.focusRight": "Painel à direita",
  "sc.focusRight": "К панели справа",
```

**`sc.tabNext`** (추가)

```json
  "sc.tabNext": "다음 탭",
  "sc.tabNext": "Next tab",
  "sc.tabNext": "次のタブ",
  "sc.tabNext": "下一个标签页",
  "sc.tabNext": "下一個分頁",
  "sc.tabNext": "Nächster Tab",
  "sc.tabNext": "Onglet suivant",
  "sc.tabNext": "Pestaña siguiente",
  "sc.tabNext": "Próxima aba",
  "sc.tabNext": "Следующая вкладка",
```

**`sc.tabPrev`** (추가)

```json
  "sc.tabPrev": "이전 탭",
  "sc.tabPrev": "Previous tab",
  "sc.tabPrev": "前のタブ",
  "sc.tabPrev": "上一个标签页",
  "sc.tabPrev": "上一個分頁",
  "sc.tabPrev": "Vorheriger Tab",
  "sc.tabPrev": "Onglet précédent",
  "sc.tabPrev": "Pestaña anterior",
  "sc.tabPrev": "Aba anterior",
  "sc.tabPrev": "Предыдущая вкладка",
```

**`sc.tabLeft`** (추가)

```json
  "sc.tabLeft": "탭을 왼쪽으로 옮기기",
  "sc.tabLeft": "Move tab left",
  "sc.tabLeft": "タブを左へ移動",
  "sc.tabLeft": "向左移动标签页",
  "sc.tabLeft": "向左移動分頁",
  "sc.tabLeft": "Tab nach links verschieben",
  "sc.tabLeft": "Déplacer l'onglet à gauche",
  "sc.tabLeft": "Mover pestaña a la izquierda",
  "sc.tabLeft": "Mover aba para a esquerda",
  "sc.tabLeft": "Переместить вкладку влево",
```

**`sc.tabRight`** (추가)

```json
  "sc.tabRight": "탭을 오른쪽으로 옮기기",
  "sc.tabRight": "Move tab right",
  "sc.tabRight": "タブを右へ移動",
  "sc.tabRight": "向右移动标签页",
  "sc.tabRight": "向右移動分頁",
  "sc.tabRight": "Tab nach rechts verschieben",
  "sc.tabRight": "Déplacer l'onglet à droite",
  "sc.tabRight": "Mover pestaña a la derecha",
  "sc.tabRight": "Mover aba para a direita",
  "sc.tabRight": "Переместить вкладку вправо",
```

**`sc.tabN`** (추가)

```json
  "sc.tabN": "{n}번째 탭",
  "sc.tabN": "Tab {n}",
  "sc.tabN": "タブ {n}",
  "sc.tabN": "第 {n} 个标签页",
  "sc.tabN": "第 {n} 個分頁",
  "sc.tabN": "Tab {n}",
  "sc.tabN": "Onglet {n}",
  "sc.tabN": "Pestaña {n}",
  "sc.tabN": "Aba {n}",
  "sc.tabN": "Вкладка {n}",
```

**`sc.tabLast`** (추가)

```json
  "sc.tabLast": "마지막 탭",
  "sc.tabLast": "Last tab",
  "sc.tabLast": "最後のタブ",
  "sc.tabLast": "最后一个标签页",
  "sc.tabLast": "最後一個分頁",
  "sc.tabLast": "Letzter Tab",
  "sc.tabLast": "Dernier onglet",
  "sc.tabLast": "Última pestaña",
  "sc.tabLast": "Última aba",
  "sc.tabLast": "Последняя вкладка",
```

**`sc.fontUp`** (추가)

```json
  "sc.fontUp": "글자 크게",
  "sc.fontUp": "Larger text",
  "sc.fontUp": "文字を大きく",
  "sc.fontUp": "放大文字",
  "sc.fontUp": "放大文字",
  "sc.fontUp": "Schrift größer",
  "sc.fontUp": "Texte plus grand",
  "sc.fontUp": "Texto más grande",
  "sc.fontUp": "Texto maior",
  "sc.fontUp": "Крупнее шрифт",
```

**`sc.fontDown`** (추가)

```json
  "sc.fontDown": "글자 작게",
  "sc.fontDown": "Smaller text",
  "sc.fontDown": "文字を小さく",
  "sc.fontDown": "缩小文字",
  "sc.fontDown": "縮小文字",
  "sc.fontDown": "Schrift kleiner",
  "sc.fontDown": "Texte plus petit",
  "sc.fontDown": "Texto más pequeño",
  "sc.fontDown": "Texto menor",
  "sc.fontDown": "Мельче шрифт",
```

**`sc.fontReset`** (추가)

```json
  "sc.fontReset": "글자 크기 처음으로",
  "sc.fontReset": "Reset text size",
  "sc.fontReset": "文字サイズを元に戻す",
  "sc.fontReset": "重置文字大小",
  "sc.fontReset": "重設文字大小",
  "sc.fontReset": "Schriftgröße zurücksetzen",
  "sc.fontReset": "Réinitialiser la taille du texte",
  "sc.fontReset": "Restablecer tamaño del texto",
  "sc.fontReset": "Redefinir tamanho do texto",
  "sc.fontReset": "Сбросить размер шрифта",
```

**`sc.fullscreen`** (추가)

```json
  "sc.fullscreen": "창 최대화/복원",
  "sc.fullscreen": "Maximize/restore window",
  "sc.fullscreen": "ウィンドウの最大化/元に戻す",
  "sc.fullscreen": "最大化/还原窗口",
  "sc.fullscreen": "最大化/還原視窗",
  "sc.fullscreen": "Fenster maximieren/wiederherstellen",
  "sc.fullscreen": "Agrandir/restaurer la fenêtre",
  "sc.fullscreen": "Maximizar/restaurar ventana",
  "sc.fullscreen": "Maximizar/restaurar janela",
  "sc.fullscreen": "Развернуть/восстановить окно",
```

**`sc.enable`** (추가)

```json
  "sc.enable": "단축키 사용",
  "sc.enable": "Use shortcuts",
  "sc.enable": "ショートカットを使う",
  "sc.enable": "使用快捷键",
  "sc.enable": "使用快速鍵",
  "sc.enable": "Tastenkürzel verwenden",
  "sc.enable": "Utiliser les raccourcis",
  "sc.enable": "Usar atajos",
  "sc.enable": "Usar atalhos",
  "sc.enable": "Использовать сочетания клавиш",
```

**`sc.enableDesc`** (추가)

```json
  "sc.enableDesc": "끄면 아래 앱 단축키가 모두 동작하지 않고, 그 키는 터미널(서버)로 그대로 갑니다. vi·tmux 처럼 같은 키를 쓰는 프로그램과 겹칠 때 끕니다. 하나만 겹치면 그 단축키만 바꾸거나 지우세요",
  "sc.enableDesc": "When off, none of the app shortcuts below work and those keys go straight to the terminal (server). Turn it off when programs such as vi or tmux use the same keys. If only one key clashes, change or clear just that shortcut",
  "sc.enableDesc": "オフにすると下のアプリショートカットはすべて動作せず、そのキーはそのままターミナル（サーバー）に送られます。vi や tmux など同じキーを使うプログラムと重なるときにオフにします。重なるキーが一つだけなら、そのショートカットだけを変更または削除してください",
  "sc.enableDesc": "关闭后，下面的应用快捷键全部不起作用，这些按键会直接发送到终端（服务器）。当 vi、tmux 等程序使用相同按键时可关闭。如果只有一个按键冲突，只需更改或清除该快捷键",
  "sc.enableDesc": "關閉後，下面的應用程式快速鍵全部不起作用，這些按鍵會直接傳送到終端機（伺服器）。當 vi、tmux 等程式使用相同按鍵時可關閉。如果只有一個按鍵衝突，只需變更或清除該快速鍵",
  "sc.enableDesc": "Wenn aus, funktioniert keines der App-Tastenkürzel unten und diese Tasten gehen direkt an das Terminal (den Server). Ausschalten, wenn Programme wie vi oder tmux dieselben Tasten verwenden. Kollidiert nur eine Taste, ändern oder löschen Sie nur dieses Kürzel",
  "sc.enableDesc": "Désactivé, aucun des raccourcis de l'application ci-dessous ne fonctionne et ces touches vont directement au terminal (serveur). Désactivez-le quand des programmes comme vi ou tmux utilisent les mêmes touches. Si une seule touche entre en conflit, modifiez ou effacez seulement ce raccourci",
  "sc.enableDesc": "Desactivado, ninguno de los atajos de la aplicación de abajo funciona y esas teclas van directamente al terminal (servidor). Desactívelo cuando programas como vi o tmux usen las mismas teclas. Si solo choca una tecla, cambie o borre solo ese atajo",
  "sc.enableDesc": "Desativado, nenhum dos atalhos do aplicativo abaixo funciona e essas teclas vão direto para o terminal (servidor). Desative quando programas como vi ou tmux usarem as mesmas teclas. Se apenas uma tecla conflitar, altere ou limpe só esse atalho",
  "sc.enableDesc": "Если выключено, ни одно из сочетаний клавиш приложения ниже не работает, а эти клавиши передаются прямо в терминал (на сервер). Выключайте, когда программы вроде vi или tmux используют те же клавиши. Если мешает только одна клавиша, измените или очистите только это сочетание",
```

**`sc.listTitle`** (추가)

```json
  "sc.listTitle": "앱 단축키",
  "sc.listTitle": "App shortcuts",
  "sc.listTitle": "アプリのショートカット",
  "sc.listTitle": "应用快捷键",
  "sc.listTitle": "應用程式快速鍵",
  "sc.listTitle": "App-Tastenkürzel",
  "sc.listTitle": "Raccourcis de l'application",
  "sc.listTitle": "Atajos de la aplicación",
  "sc.listTitle": "Atalhos do aplicativo",
  "sc.listTitle": "Сочетания клавиш приложения",
```

**`sc.editHint`** (추가)

```json
  "sc.editHint": "키 칸을 누른 뒤 쓸 키를 누르세요. Esc 는 취소입니다. 다른 동작이 쓰던 키를 고르면 그 동작의 키는 비워집니다",
  "sc.editHint": "Click a key box, then press the keys to use. Esc cancels. If you pick keys another action was using, that action is left without a key",
  "sc.editHint": "キーの欄をクリックしてから、使うキーを押してください。Esc でキャンセルします。他の動作が使っていたキーを選ぶと、その動作のキーは空になります",
  "sc.editHint": "点击按键框，然后按下要使用的按键。Esc 取消。如果选择了其他操作正在使用的按键，该操作的按键将被清空",
  "sc.editHint": "點選按鍵框，然後按下要使用的按鍵。Esc 取消。如果選擇了其他動作正在使用的按鍵，該動作的按鍵將被清空",
  "sc.editHint": "Klicken Sie auf ein Tastenfeld und drücken Sie dann die gewünschten Tasten. Esc bricht ab. Wählen Sie Tasten, die eine andere Aktion verwendet hat, bleibt jene Aktion ohne Taste",
  "sc.editHint": "Cliquez sur une case de touche, puis appuyez sur les touches à utiliser. Échap annule. Si vous choisissez des touches utilisées par une autre action, cette action reste sans touche",
  "sc.editHint": "Haga clic en una casilla de tecla y luego pulse las teclas que quiera usar. Esc cancela. Si elige teclas que usaba otra acción, esa acción se queda sin tecla",
  "sc.editHint": "Clique em uma caixa de tecla e pressione as teclas que deseja usar. Esc cancela. Se escolher teclas que outra ação usava, essa ação fica sem tecla",
  "sc.editHint": "Щёлкните поле клавиши, затем нажмите нужные клавиши. Esc — отмена. Если выбрать клавиши, которые использовало другое действие, то действие останется без клавиши",
```

**`sc.none`** (추가)

```json
  "sc.none": "없음",
  "sc.none": "None",
  "sc.none": "なし",
  "sc.none": "无",
  "sc.none": "無",
  "sc.none": "Keine",
  "sc.none": "Aucune",
  "sc.none": "Ninguna",
  "sc.none": "Nenhuma",
  "sc.none": "Нет",
```

**`sc.press`** (추가)

```json
  "sc.press": "키를 누르세요…",
  "sc.press": "Press keys…",
  "sc.press": "キーを押してください…",
  "sc.press": "请按键…",
  "sc.press": "請按鍵…",
  "sc.press": "Tasten drücken…",
  "sc.press": "Appuyez sur les touches…",
  "sc.press": "Pulse las teclas…",
  "sc.press": "Pressione as teclas…",
  "sc.press": "Нажмите клавиши…",
```

**`sc.clear`** (추가)

```json
  "sc.clear": "키 지우기",
  "sc.clear": "Clear key",
  "sc.clear": "キーを削除",
  "sc.clear": "清除按键",
  "sc.clear": "清除按鍵",
  "sc.clear": "Taste löschen",
  "sc.clear": "Effacer la touche",
  "sc.clear": "Borrar tecla",
  "sc.clear": "Limpar tecla",
  "sc.clear": "Очистить клавишу",
```

**`sc.reset`** (추가)

```json
  "sc.reset": "기본값으로",
  "sc.reset": "Reset to default",
  "sc.reset": "既定に戻す",
  "sc.reset": "恢复默认",
  "sc.reset": "恢復預設",
  "sc.reset": "Auf Standard zurücksetzen",
  "sc.reset": "Rétablir par défaut",
  "sc.reset": "Restablecer",
  "sc.reset": "Restaurar padrão",
  "sc.reset": "Сбросить по умолчанию",
```

**`sc.resetAll`** (추가)

```json
  "sc.resetAll": "모두 기본값으로",
  "sc.resetAll": "Reset all to defaults",
  "sc.resetAll": "すべて既定に戻す",
  "sc.resetAll": "全部恢复默认",
  "sc.resetAll": "全部恢復預設",
  "sc.resetAll": "Alle zurücksetzen",
  "sc.resetAll": "Tout rétablir par défaut",
  "sc.resetAll": "Restablecer todo",
  "sc.resetAll": "Restaurar todos",
  "sc.resetAll": "Сбросить все",
```

**`sc.displaced`** (추가)

```json
  "sc.displaced": "{keys} 를 쓰던 \"{name}\"의 단축키를 비웠습니다",
  "sc.displaced": "{keys} was used by \"{name}\", which now has no key",
  "sc.displaced": "{keys} を使っていた「{name}」のショートカットを空にしました",
  "sc.displaced": "{keys} 原先用于“{name}”，该操作的快捷键已清空",
  "sc.displaced": "{keys} 原先用於「{name}」，該動作的快速鍵已清空",
  "sc.displaced": "{keys} wurde von „{name}“ verwendet; diese Aktion hat jetzt keine Taste",
  "sc.displaced": "{keys} était utilisé par « {name} », qui n'a plus de touche",
  "sc.displaced": "{keys} lo usaba «{name}», que ahora no tiene tecla",
  "sc.displaced": "{keys} era usado por \"{name}\", que agora está sem tecla",
  "sc.displaced": "{keys} использовалось для «{name}»; у этого действия больше нет клавиши",
```

**`sc.errModifier`** (추가)

```json
  "sc.errModifier": "Ctrl 또는 Alt 와 함께 누르는 키나 F1~F12 만 쓸 수 있습니다",
  "sc.errModifier": "Use keys together with Ctrl or Alt, or one of F1–F12",
  "sc.errModifier": "Ctrl または Alt と一緒に押すキーか F1〜F12 だけ使えます",
  "sc.errModifier": "只能使用与 Ctrl 或 Alt 组合的按键，或 F1–F12",
  "sc.errModifier": "只能使用與 Ctrl 或 Alt 組合的按鍵，或 F1–F12",
  "sc.errModifier": "Nur Tasten zusammen mit Strg oder Alt oder F1–F12 sind möglich",
  "sc.errModifier": "Utilisez des touches avec Ctrl ou Alt, ou une touche F1–F12",
  "sc.errModifier": "Use teclas junto con Ctrl o Alt, o una de F1–F12",
  "sc.errModifier": "Use teclas junto com Ctrl ou Alt, ou uma de F1–F12",
  "sc.errModifier": "Можно использовать только клавиши вместе с Ctrl или Alt либо F1–F12",
```

**`sc.errReserved`** (추가)

```json
  "sc.errReserved": "{keys} 는 터미널의 복사·붙여넣기에 쓰는 키라 고를 수 없습니다",
  "sc.errReserved": "{keys} is used for copy and paste in the terminal and cannot be chosen",
  "sc.errReserved": "{keys} はターミナルのコピー・貼り付けに使うキーのため選べません",
  "sc.errReserved": "{keys} 用于终端的复制和粘贴，无法选择",
  "sc.errReserved": "{keys} 用於終端機的複製和貼上，無法選擇",
  "sc.errReserved": "{keys} wird im Terminal für Kopieren und Einfügen verwendet und kann nicht gewählt werden",
  "sc.errReserved": "{keys} sert au copier-coller dans le terminal et ne peut pas être choisi",
  "sc.errReserved": "{keys} se usa para copiar y pegar en el terminal y no se puede elegir",
  "sc.errReserved": "{keys} é usado para copiar e colar no terminal e não pode ser escolhido",
  "sc.errReserved": "{keys} используется для копирования и вставки в терминале, выбрать нельзя",
```

**`sc.fixedTitle`** (추가)

```json
  "sc.fixedTitle": "터미널 안에서 쓰는 키",
  "sc.fixedTitle": "Keys inside the terminal",
  "sc.fixedTitle": "ターミナル内で使うキー",
  "sc.fixedTitle": "终端内使用的按键",
  "sc.fixedTitle": "終端機內使用的按鍵",
  "sc.fixedTitle": "Tasten im Terminal",
  "sc.fixedTitle": "Touches dans le terminal",
  "sc.fixedTitle": "Teclas dentro del terminal",
  "sc.fixedTitle": "Teclas dentro do terminal",
  "sc.fixedTitle": "Клавиши в терминале",
```

**`sc.fixedDesc`** (추가)

```json
  "sc.fixedDesc": "바꿀 수 없고, \"단축키 사용\"을 꺼도 동작합니다. Ctrl+V 붙여넣기는 설정 > 입력에서 끌 수 있습니다",
  "sc.fixedDesc": "These cannot be changed and keep working when \"Use shortcuts\" is off. Pasting with Ctrl+V can be turned off in Settings > Input",
  "sc.fixedDesc": "変更できず、「ショートカットを使う」をオフにしても動作します。Ctrl+V での貼り付けは 設定 > 入力 でオフにできます",
  "sc.fixedDesc": "无法更改，关闭“使用快捷键”后仍然有效。Ctrl+V 粘贴可在 设置 > 输入 中关闭",
  "sc.fixedDesc": "無法變更，關閉「使用快速鍵」後仍然有效。Ctrl+V 貼上可在 設定 > 輸入 中關閉",
  "sc.fixedDesc": "Nicht änderbar; funktionieren auch, wenn „Tastenkürzel verwenden“ aus ist. Einfügen mit Strg+V lässt sich unter Einstellungen > Eingabe abschalten",
  "sc.fixedDesc": "Non modifiables ; elles fonctionnent même si « Utiliser les raccourcis » est désactivé. Le collage avec Ctrl+V se désactive dans Paramètres > Saisie",
  "sc.fixedDesc": "No se pueden cambiar y siguen funcionando con «Usar atajos» desactivado. Pegar con Ctrl+V se puede desactivar en Configuración > Entrada",
  "sc.fixedDesc": "Não podem ser alteradas e continuam funcionando com \"Usar atalhos\" desativado. Colar com Ctrl+V pode ser desativado em Configurações > Entrada",
  "sc.fixedDesc": "Изменить нельзя; работают, даже если «Использовать сочетания клавиш» выключено. Вставку по Ctrl+V можно отключить в Настройки > Ввод",
```

**`settings.sec.shortcuts`** (추가)

```json
  "settings.sec.shortcuts": "단축키",
  "settings.sec.shortcuts": "Shortcuts",
  "settings.sec.shortcuts": "ショートカット",
  "settings.sec.shortcuts": "快捷键",
  "settings.sec.shortcuts": "快速鍵",
  "settings.sec.shortcuts": "Tastenkürzel",
  "settings.sec.shortcuts": "Raccourcis",
  "settings.sec.shortcuts": "Atajos",
  "settings.sec.shortcuts": "Atalhos",
  "settings.sec.shortcuts": "Сочетания клавиш",
```


## F. 명세·문서

### `spec/behavior/shortcuts.md` — 수정

```diff
--- a/spec/behavior/shortcuts.md
+++ b/spec/behavior/shortcuts.md
@@ -6,9 +6,10 @@
 
 - 단축키는 **물리 키 기준**으로 인식한다. 문자·숫자 키는 키 위치로 판단하므로 한글 입력기가 켜져 있어도 동작한다.
 - 인식한 단축키가 실제 동작을 했으면 그 키 입력은 터미널로 전달되지 않는다. 동작하지 않았으면(조건 불충족) 터미널이 받는다.
+- 앱 단축키는 설정에서 **전체를 끄거나 명령마다 키를 바꿀 수 있다**(§5, 2026-11-25). §2 의 표는 **기본 키**다.
 - **동작하지 않는 상황**: ① 볼트가 설정되어 있고 잠겨 있을 때 ② 대화상자가 열려 있을 때. 예외: **빠른 접속 대화상자가 열려 있을 때는 모든 단축키가 동작**하고, 전체 화면 전환(F11)은 어떤 대화상자가 열려 있어도 동작한다. (설정·서버 편집 등 다른 대화상자가 열려 있으면 F11 외에는 무시)
 
-## 2. Windows 단축키 표
+## 2. Windows 단축키 표 (기본 키)
 
 | 키 | 동작 | 조건·세부 |
 |---|---|---|
@@ -69,4 +70,59 @@
 | 글자 크기 | Ctrl+= / Ctrl+- / Ctrl+0 |
 | 빠른 접속에서 오른쪽 분할로 열기 | Ctrl+Enter |
 
-시작 화면과 설정의 "단축키" 표는 위 순서의 항목 이름과 라벨을 그대로 쓴다. 시작 화면은 처음 12개만 보여 주고, 설정 화면은 전체를 보여 준다. 표 항목의 이름은 "빠른 연결 / 명령", "새 탭 (빠른 연결)", "새 로컬 터미널", "탭 닫기", "같은 서버를 오른쪽에", "같은 서버를 아래에", "창 이동", "창 최대화", "탭 이동", "붙여넣기 (이미지·파일 자동 업로드)", "클립보드 이미지 → 서버", "복사", "터미널에서 찾기", "SFTP 창", "일괄 입력", "서버 목록 표시/숨김", "글자 크기" 이다.
+시작 화면의 "단축키" 표는 위 순서의 처음 12개를 보여 준다. (2026-11-25, D126) 키는 **지금 설정의 키**로 보인다: 키를 비웠거나 전체를 꺼서 키가 없는 줄은 빼고(붙여넣기·클립보드 이미지·복사 3줄은 늘 남는다), 묶음 줄("패널 이동", "탭 이동")은 묶인 명령이 모두 기본 키면 위 라벨을, 하나라도 바꿨으면 키가 있는 첫 명령의 키 뒤에 " …" 를 붙여 보인다. 설정 화면의 표는 §5 의 명령 목록이다(ui/settings.md §2a). 표 항목의 이름은 "빠른 연결 / 명령", "새 탭 (빠른 연결)", "새 로컬 터미널", "탭 닫기", "같은 서버를 오른쪽에", "같은 서버를 아래에", "창 이동", "창 최대화", "탭 이동", "붙여넣기 (이미지·파일 자동 업로드)", "클립보드 이미지 → 서버", "복사", "터미널에서 찾기", "SFTP 창", "일괄 입력", "서버 목록 표시/숨김", "글자 크기" 이다.
+
+## 5. 단축키 켜기/끄기와 키 바꾸기
+
+> 2026-11-25 추가 (사용자 지시, D126). 기존 앱에는 없던 기능이다.
+
+### 5.1 대상
+- **앱 단축키** 34개가 대상이다(아래 표). 명령마다 키가 **하나**다. 빠른 접속은 기본 키가 여럿이라 명령도 여럿으로 나뉜다.
+- **터미널 안에서 쓰는 키**(Ctrl+V · Ctrl+Shift+V · Shift+Insert 붙여넣기, Ctrl+Shift+C 복사, Ctrl+Alt+V 클립보드 이미지)는 대상이 아니다: 바꿀 수 없고 전체를 꺼도 동작한다. Ctrl+V 는 따로 있는 설정 `ctrlVPaste` 로 끈다. 빠른 접속 안의 Ctrl+Enter 같은 대화상자·목록 안의 키도 대상이 아니다.
+
+| 순서 | 명령 id | 이름 | 기본 키 |
+|---|---|---|---|
+| 1 | quick | 빠른 연결 / 명령 | Ctrl+K |
+| 2 | newTab | 새 탭 (빠른 연결) | Ctrl+Shift+T |
+| 3 | quickAlt | 빠른 연결 (보조 키) | Ctrl+Shift+N |
+| 4 | quickOpen | 빠른 연결 열기 (닫지 않음) | Ctrl+Shift+P |
+| 5 | newLocal | 새 로컬 터미널 | Ctrl+Alt+T |
+| 6 | closePane | 탭 닫기 | Ctrl+Shift+W |
+| 7 | splitRight | 같은 서버를 오른쪽에 | Ctrl+Shift+D |
+| 8 | splitDown | 같은 서버를 아래에 | Ctrl+Shift+E |
+| 9 | zoom | 창 최대화 | Ctrl+Shift+Enter |
+| 10~13 | focusUp · focusDown · focusLeft · focusRight | 위쪽 / 아래쪽 / 왼쪽 / 오른쪽 창으로 | Alt+↑ / ↓ / ← / → |
+| 14 | find | 터미널에서 찾기 | Ctrl+Shift+F |
+| 15 | sftp | SFTP 창 | Ctrl+Shift+S |
+| 16 | broadcast | 일괄 입력 | Ctrl+Shift+B |
+| 17 | sidebar | 서버 목록 표시/숨김 | Ctrl+Shift+L |
+| 18 | tabNext | 다음 탭 | Ctrl+Tab |
+| 19 | tabPrev | 이전 탭 | Ctrl+Shift+Tab |
+| 20 | tabLeft | 탭을 왼쪽으로 옮기기 | Ctrl+Shift+← |
+| 21 | tabRight | 탭을 오른쪽으로 옮기기 | Ctrl+Shift+→ |
+| 22~29 | tab1 ~ tab8 | {n}번째 탭 | Ctrl+1 ~ Ctrl+8 |
+| 30 | tabLast | 마지막 탭 | Ctrl+9 |
+| 31 | fontUp | 글자 크게 | Ctrl+= |
+| 32 | fontDown | 글자 작게 | Ctrl+- |
+| 33 | fontReset | 글자 크기 처음으로 | Ctrl+0 |
+| 34 | fullscreen | 창 최대화/복원 | F11 |
+
+### 5.2 조합의 형식
+- 조합은 `Ctrl+` · `Alt+` · `Shift+` (이 순서, 누른 것만) 뒤에 **물리 키 이름**을 붙인 글자다. 예: `Ctrl+Shift+KeyT`, `Alt+ArrowUp`, `Ctrl+Equal`, `F11`. 물리 키 이름은 웹 표준의 키 위치 이름(KeyA~KeyZ, Digit0~Digit9, ArrowUp, Enter, Tab, Equal, Minus, F1 …)이다.
+- 조합은 **정확히 같을 때만** 맞는다(Ctrl+Shift+K 는 Ctrl+K 가 아니다). 숫자 패드의 Enter · + · - · 0~9 는 본 자판의 Enter · = · - · 0~9 와 같은 키로 본다. Win 키가 함께 눌린 입력은 단축키가 아니다.
+- 화면에는 읽기 쉽게 보인다: KeyT → T, Digit9 → 9, ArrowUp → ↑, Equal → =, Minus → - 등(예: `Ctrl+Shift+T`, `Alt+↑`).
+
+### 5.3 저장
+- `shortcutsEnabled`(기본 true): 앱 단축키 전체를 쓸지.
+- `shortcuts`(기본 빈 표): **기본값에서 바꾼 명령만** 담는다. 키는 명령 id, 값은 조합 글자. **값이 빈 글자면 그 명령은 키가 없다.** 표에 없는 명령은 기본 키다.
+- 읽을 때: 모르는 명령 id 와 글자가 아닌 값은 무시한다. 같은 조합이 두 명령에 걸려 있으면(파일을 손으로 고쳤거나 동기화로 들어온 경우) 5.1 표의 **앞 명령**이 그 키를 갖고 뒤 명령은 키가 없는 것으로 본다.
+
+### 5.4 동작
+- **전체 끄기**: `shortcutsEnabled` 가 false 면 어떤 키 입력도 앱 단축키가 아니다(F11 포함). 그 키들은 터미널이 받아 서버로 간다(터미널이 보낼 것이 없는 조합은 아무 일도 없다). 5.1 의 터미널 안 키는 그대로 동작한다. 메뉴·버튼의 기능 자체는 마우스로 계속 쓸 수 있다.
+- **키 바꾸기**: 바꾼 명령은 새 조합으로만 동작하고 옛 조합은 풀려 터미널이 받는다.
+- **겹침**: 다른 명령이 쓰고 있는 조합을 고르면 새 명령에 그 조합을 주고 **먼저 쓰던 명령은 빈 값으로 저장**한다(키 없음). 기본값으로 되돌릴 때도 같다(그 기본 키를 다른 명령이 쓰고 있으면 그 명령이 비워진다).
+- **지우기**: 빈 값으로 저장한다. **기본값으로**: 기본 키와 같아지면 표에서 그 명령을 뺀다. **모두 기본값으로**: 표를 비운다.
+- **고를 수 없는 조합**: ① Ctrl 도 Alt 도 없는 조합(글자 입력과 겹친다). 단 F1~F24 는 그냥도, Shift 와도 된다. ② 터미널 안 키 5개(Ctrl+V, Ctrl+Shift+V, Ctrl+Shift+C, Ctrl+Alt+V, Shift+Insert). Esc 한 번은 받기 취소라 고를 수 없다.
+- 설정은 바꾸는 즉시 적용된다(열려 있는 터미널 포함). 설정 화면이 키를 받는 동안에는 앱 단축키가 동작하지 않는다.
+- **화면 표기는 지금 키를 따른다**: 시작 화면의 표와 안내 문장의 키(빠른 연결을 여는 키 중 quick → newTab → quickAlt → quickOpen 순서로 처음 있는 것, 하나도 없으면 문장을 보이지 않는다), [로컬 터미널] 버튼의 툴팁, 탭 막대 SFTP 버튼의 툴팁, 칸·탭 메뉴 항목 옆의 키. 키가 없으면(비웠거나 전체를 껐으면) 표기를 뺀다.
+- §1 의 "동작하지 않는 상황"(볼트 잠김·대화상자)은 그대로다.
```

### `spec/ui/settings.md` — 수정

```diff
--- a/spec/ui/settings.md
+++ b/spec/ui/settings.md
@@ -2,21 +2,24 @@
 
 > 작성 방식: 기존 앱 실행 관찰 + 열람. 작성일 2026-11-10. 상태: **draft**. 기본값·키 이름은 data/store.md §4.
 
-크게 보기 모달, 제목 "설정". 왼쪽 **내비게이션**(구역 9개, 선택된 구역 강조) + 오른쪽 **본문**. 모든 변경은 **즉시 적용·저장**된다(저장 버튼 없음). 열 때 구역을 지정할 수 있고 기본은 "화면".
+크게 보기 모달, 제목 "설정". 왼쪽 **내비게이션**(구역 10개, 선택된 구역 강조) + 오른쪽 **본문**. 모든 변경은 **즉시 적용·저장**된다(저장 버튼 없음). 열 때 구역을 지정할 수 있고 기본은 "화면".
 
 | 순서 | 구역 | 아이콘 |
 |---|---|---|
 | 1 | 모양 | 팔레트 |
 | 2 | 입력·붙여넣기 | 키보드 |
-| 3 | 이미지·파일 전송 | 이미지 업로드 |
-| 4 | 연결 | 플러그 |
-| 5 | 로그 | 두루마리 문서 |
-| 6 | 동기화 (git) | 새로고침 |
-| 7 | 보안·계정 | 방패 |
-| 8 | 백업 | 보관함 |
-| 9 | 정보·단축키 | 정보 |
+| 3 | 단축키 | 명령 키(⌘ 모양) |
+| 4 | 이미지·파일 전송 | 이미지 업로드 |
+| 5 | 연결 | 플러그 |
+| 6 | 로그 | 두루마리 문서 |
+| 7 | 동기화 (git) | 새로고침 |
+| 8 | 보안·계정 | 방패 |
+| 9 | 백업 | 보관함 |
+| 10 | 정보 | 정보 |
 
 > 2026-11-23 (D120): "연결"에 있던 세션 로그 항목을 **"로그" 구역**으로 떼어 냈다. 아래 절 번호는 다른 문서·코드가 가리키고 있어 바꾸지 않았다 — 로그 구역은 **§4a** 다(화면의 순서는 위 표).
+>
+> 2026-11-25 (D126): "정보"에 있던 단축키 표를 **"단축키" 구역**으로 떼어 내고, 전체 켜기/끄기와 키 바꾸기를 넣었다. 단축키 구역은 **§2a** 다.
 
 ## 1. 모양
 - **언어**(맨 위, 목록 선택, D75): 도움말 "화면과 안내 문구의 언어입니다. 바꾸면 바로 적용됩니다." 목록은 behavior/i18n.md §1 의 10개(그 언어로 쓴 이름). 지금 화면 언어가 골라져 있다.
@@ -45,6 +48,17 @@
 
 그 아래: 필드 **위험 명령 패턴 (정규식, 한 줄에 하나)** — 7줄 여러 줄 입력. 입력을 줄 단위로 나눠 앞뒤 공백 제거·빈 줄 제거 후 목록으로 저장. 버튼 [기본값으로] → 기본 패턴 목록(data/store.md §4)으로 복원.
 
+## 2a. 단축키
+(2026-11-25, D126. 동작은 behavior/shortcuts.md §5.)
+- 카드 토글 **단축키 사용**(기본 켜짐) — "끄면 아래 앱 단축키가 모두 동작하지 않고, 그 키는 터미널(서버)로 그대로 갑니다. vi·tmux 처럼 같은 키를 쓰는 프로그램과 겹칠 때 끕니다. 하나만 겹치면 그 단축키만 바꾸거나 지우세요" — 값은 `shortcutsEnabled`.
+- 소제목 **앱 단축키** + 오른쪽에 작은 버튼 [모두 기본값으로](바꾼 것이 없으면 누를 수 없다). 아래 도움말 "키 칸을 누른 뒤 쓸 키를 누르세요. Esc 는 취소입니다. 다른 동작이 쓰던 키를 고르면 그 동작의 키는 비워집니다".
+- 목록(테두리 상자, 한 줄에 명령 하나, behavior/shortcuts.md §5 의 34개·그 순서): **이름** · **키 칸**(지금 키, 없으면 흐린 "없음") · [키 지우기](× 아이콘, 키가 없으면 누를 수 없다) · [기본값으로](되돌리기 아이콘, 지금 키가 기본 키면 누를 수 없다, 툴팁에 기본 키).
+  - 키 칸을 누르면 강조색 테두리의 "키를 누르세요…" 로 바뀌고 다음에 누른 조합을 받는다. Ctrl·Shift·Alt 만 누른 것은 기다린다. Esc·다른 곳 누르기·창이 포커스를 잃으면 취소(그동안의 Esc 는 설정 창을 닫지 않는다).
+  - 고를 수 없는 조합이면 오류 토스트를 내고 계속 기다린다: "Ctrl 또는 Alt 와 함께 누르는 키나 F1~F12 만 쓸 수 있습니다" / "{키} 는 터미널의 복사·붙여넣기에 쓰는 키라 고를 수 없습니다".
+  - 다른 명령이 쓰던 조합을 고르면 그대로 저장하고, 먼저 쓰던 명령은 "없음"이 되며 정보 토스트 '{키} 를 쓰던 "{이름}"의 단축키를 비웠습니다'.
+  - "단축키 사용"이 꺼져 있으면 목록의 이름·키가 흐리게 보인다(바꾸는 것은 된다).
+- 소제목 **터미널 안에서 쓰는 키** + 도움말 '바꿀 수 없고, "단축키 사용"을 꺼도 동작합니다. Ctrl+V 붙여넣기는 설정 > 입력에서 끌 수 있습니다' + 표 3줄: 붙여넣기 (이미지·파일 자동 업로드) Ctrl+V · 클립보드 이미지 → 서버 Ctrl+Alt+V · 복사 Ctrl+Shift+C / 선택.
+
 ## 3. 이미지·파일 전송
 - 안내 박스(이미지 아이콘): "SSH 터미널에서 **Ctrl+V**로 이미지(캡처)를 붙여넣거나 파일을 터미널로 끌어다 놓으면, 서버의 업로드 폴더로 전송한 뒤 그 **경로를 터미널에 입력**합니다. 서버에서 실행 중인 **Claude Code** 등 CLI는 이 경로로 이미지를 읽을 수 있습니다. 예: `이 화면의 오류 원인 알려줘` 입력 후 Ctrl+V."
 - **서버 업로드 폴더**(고정폭 문자열, 도움말 "업로드 파일을 둘 서버 경로입니다. ~ 는 홈 디렉터리이며, 없으면 만듭니다.")
@@ -83,7 +97,7 @@
 - 버튼 [백업 내보내기](주 버튼): 저장 위치를 고르는 대화상자 후 저장. 성공하면 토스트 "백업 저장: {경로}"(취소하면 없음), 실패하면 오류 토스트.
 - 버튼 [백업 가져오기]: 확인 대화상자(제목 "백업 가져오기", 본문 "현재 서버 목록과 계정 정보가 백업 파일 내용으로 바뀝니다. 계속할까요?", 확인 "가져오기", 위험) → 파일 선택 → 성공하면 저장 데이터·볼트 상태를 새로 읽고 토스트 "백업을 가져왔습니다. 백업한 PC의 앱 잠금 비밀번호로 잠금을 해제하세요." 실패하면 오류 토스트.
 
-## 8. 정보·단축키
+## 8. 정보
 - 위: 로고(44px, 둥근 모서리), 굵은 "DariTerm {버전}", 흐린 부제 "SSH · Telnet · Serial · 로컬 터미널".
 - 필드 **개발자**: `YangHun Lee (kai)` (여권 영문 이름 + 평소 이름, 2026-11-14 사용자 지시).
 - 필드 **라이선스**: `독점 소프트웨어 · Copyright © 2026 YangHun Lee (kai). All rights reserved.` (D64)
@@ -92,4 +106,4 @@
 - 필드 **참고한 프로젝트** (D63): `BrightTerm v1.1.0 — MIT · Copyright (c) 2026 Dany Kim`, 아래 흐린 글씨 "이 프로그램은 BrightTerm v1.1.0의 기능과 동작을 참고해 새로 작성했습니다. 원본 소스 코드·이름·로고·이미지는 포함하지 않습니다.", 접힌 [라이선스 전문 보기] 안에 MIT 전문(저장소 `THIRD_PARTY_NOTICES.md` 와 같은 내용).
 - 카드: 토글 **새 버전 자동 확인**(설명 "시작할 때와 12시간마다 새 릴리스를 확인합니다. 서버 목록·IP·계정 정보는 보내지 않으며, 폐쇄망 등으로 확인이 안 되면 조용히 건너뜁니다") + 버튼 [업데이트 확인]. 결과 토스트: 새 버전 있음 → 정보 "새 버전 {버전} 이 있습니다", 최신 → 성공 "최신 버전입니다", 배포 저장소 미설정 빌드 → 성공 "배포 저장소가 설정되지 않은 빌드입니다", 오류·응답 없음 → 오류 "확인하지 못했습니다 (네트워크를 확인하세요)".
 - 필드 **데이터 폴더**: 경로(고정폭 박스).
-- 소제목 **단축키** + 전체 단축키 표 (behavior/shortcuts.md §4 의 순서·문구, **시작 화면과 달리 전체 항목**).
+- (2026-11-25, D126) 단축키 표는 여기에 없다 — "단축키" 구역(§2a)으로 옮겼다.
```

### `spec/ui/shell.md` — 수정

```diff
--- a/spec/ui/shell.md
+++ b/spec/ui/shell.md
@@ -85,10 +85,10 @@
 
 가운데 카드 하나:
 - 로고(48px), 제목 "DariTerm"
-- 안내 문장: "서버를 더블클릭하거나 **Ctrl+K** 로 바로 연결하세요." (단축키는 키 모양 표기)
+- 안내 문장: "서버를 더블클릭하거나 **Ctrl+K** 로 바로 연결하세요." (단축키는 키 모양 표기. 키는 지금 설정의 빠른 연결 키이고, 그 키가 하나도 없으면 문장을 보이지 않는다 — behavior/shortcuts.md §5.4)
 - 버튼 4개(가로 나열): **빠른 접속**(주 버튼, 번개 아이콘) · **로컬 터미널**(툴팁 Ctrl+Alt+T) · **새 서버 등록**(서버 편집 대화상자를 최상위 폴더 기준으로 연다) · **PuTTY에서 가져오기**(가져오기 대화상자)
 - **최근 서버 격자**: 서버가 하나라도 있으면 표시. 최근 사용 시각(lastUsedAt) 내림차순, 같으면 이름 오름차순으로 **최대 9개**. 카드마다 왼쪽에 색 막대(서버 색·폴더 색·환경 색, 없으면 테두리색), 프로토콜 아이콘, 굵은 이름, 아래에 대상 문자열. 카드를 누르면 새 탭으로 연결.
-- 단축키 표: 처음 12개 항목 (behavior/shortcuts.md §4)
+- 단축키 표: 처음 12개 항목 (behavior/shortcuts.md §4). 지금 설정의 키로 보이고 키가 없는 줄은 뺀다(§5.4)
 
 ## 5. 토스트(알림)
 
```

### `spec/data/store.md` — 수정

```diff
--- a/spec/data/store.md
+++ b/spec/data/store.md
@@ -103,6 +103,8 @@
 | sessionLogTimestamp | false | 세션 로그의 줄마다 앞에 시각을 붙인다 ([../behavior/session.md](../behavior/session.md) §6, D112) |
 | sessionLogAnsi | false | 세션 로그에 ANSI 시퀀스와 CR 을 지우지 않고 받은 그대로 남긴다 ([../behavior/session.md](../behavior/session.md) §6, D120) |
 | sessionLogSyncSec | 2 | 기록 중인 세션 로그를 디스크까지 내려 쓰는 주기(초), 0~5. 0 이면 출력을 받는 즉시 쓰고 내려 쓴다 ([../behavior/session.md](../behavior/session.md) §6, 2026-11-18) |
+| shortcutsEnabled | true | 앱 단축키 전체 사용. false 면 그 키들은 터미널(서버)로 간다 ([../behavior/shortcuts.md](../behavior/shortcuts.md) §5, D126) |
+| shortcuts | 빈 표(`{}`) | 기본값에서 바꾼 단축키만: 명령 id → 조합 글자(예: `"find": "Ctrl+Alt+KeyF"`). 빈 글자는 키 없음 ([../behavior/shortcuts.md](../behavior/shortcuts.md) §5.2·§5.3, D126) |
 | bell | false | |
 | shiftEnterNewline | true | Shift+Enter 로 줄바꿈 입력 |
 | rememberWindow | true | 창 크기·위치 기억 |
```

### `spec/acceptance/03-panes-terminal.md` — 수정

```diff
--- a/spec/acceptance/03-panes-terminal.md
+++ b/spec/acceptance/03-panes-terminal.md
@@ -50,6 +50,11 @@
 | K5 | 볼트 잠김 | 아무 앱 단축키 | 무시 |
 | K6 | 칸 1개 | Ctrl+Shift+B | 동작 없음 |
 | K7 | 어떤 대화상자든 | F11 | 창이 최대화/복원 토글 |
+| K8 | 설정 > 단축키 "단축키 사용" 끔, 터미널에서 vi 실행 | Ctrl+K, Ctrl+Shift+T, Alt+방향키, F11 | 앱은 반응하지 않고 키가 터미널로 간다. Ctrl+Shift+C 복사·Ctrl+V 붙여넣기는 된다 |
+| K9 | "터미널에서 찾기"의 키를 Ctrl+Alt+F 로 바꿈 | Ctrl+Alt+F / Ctrl+Shift+F | 앞의 것은 찾기 막대, 뒤의 것은 아무 일 없음. 칸 메뉴의 찾기 옆에 Ctrl+Alt+F |
+| K10 | "터미널에서 찾기"의 키로 Ctrl+K 를 누름 | | 찾기 = Ctrl+K, "빠른 연결 / 명령" = 없음(저장 값 빈 글자), 안내 토스트 |
+| K11 | 키를 받는 중 | a (조합 키 없이) / Ctrl+V / Esc | 오류 토스트 후 계속 기다림 / 같음 / 받기 취소(설정 창은 그대로) |
+| K12 | 키를 바꾼 뒤 [모두 기본값으로] | | 모든 명령이 기본 키, 저장 값은 빈 표 |
 
 ## 세션 (behavior/session.md, protocols.md)
 | # | 조건 | 동작 | 기대 결과 |
```

### `spec/acceptance/04-hosts-sftp-settings-sync.md` — 수정

```diff
--- a/spec/acceptance/04-hosts-sftp-settings-sync.md
+++ b/spec/acceptance/04-hosts-sftp-settings-sync.md
@@ -40,7 +40,7 @@
 | G3 | 위험 명령 패턴 편집 | | 줄 단위, 빈 줄 제외로 저장. [기본값으로] 로 복원 |
 | G4 | 앱 잠금 비밀번호 변경, 현재 비밀번호 틀림 | | "현재 비밀번호가 틀립니다" |
 | G5 | 저장된 계정 삭제 | | 그 계정을 쓰던 서버의 계정 연결이 끊어진다 |
-| G6 | 정보 화면 | | 개발자 `YangHun Lee (kai)`, 데이터 폴더 경로, 단축키 전체 표 |
+| G6 | 정보 화면 | | 개발자 `YangHun Lee (kai)`, 데이터 폴더 경로. 단축키 표는 없다("단축키" 구역으로 옮김, D126) |
 | G7 | 업데이트 [업데이트 확인], 배포 저장소 미지정 | | "배포 저장소가 설정되지 않은 빌드입니다" |
 | G8 | [기본 크기로] | | 창이 1440×900 가운데 |
 | G8a | 키워드 강조 규칙 `error`(빨강), `디스크`(노랑, 바탕 칠하기) | 터미널에 `디스크 Error: disk ERROR` 출력 | `Error`·`ERROR` 는 빨간 글자, `디스크` 는 노란 바탕에 흰 글자. 복사하면 원래 글자 그대로. 규칙을 끄거나 지우면 곧바로 원래 색 (D113) |
```

### `docs/scenarios.md` — 수정

```diff
--- a/docs/scenarios.md
+++ b/docs/scenarios.md
@@ -175,19 +175,20 @@
 | 12-4 | tmux 세션 유지 | 서버 고급 "연결 유지 (tmux)": 창·앱을 닫아도 서버의 프로그램이 계속 돌고, 다시 열면 같은 화면에 붙는다. tmux 가 없으면 안내 후 평소처럼. Windows 로컬은 WSL 셸만. |
 | 12-5 | 로컬 셸이 끝남 | "셸이 종료되었습니다" → Enter 로 새 셸. |
 
-## 13. 설정 (톱니바퀴, 구역 9개)
+## 13. 설정 (톱니바퀴, 구역 10개)
 
 | 구역 | 주요 항목 |
 |---|---|
 | 화면 | 언어 · 앱 테마(다크/라이트 2종) · 터미널 색상 6종(DariTerm Dark·Light, PuTTY Classic, Solarized Dark, Monokai, Nord) · 글꼴·크기·줄 간격 · 창 크기·위치 기억 · [기본 크기로] · 커서 모양 · 기록 줄 수 · 위아래 창 최대 개수 · 커서 깜빡임 · 경고음 |
 | 입력 | 드래그하면 바로 복사 · 오른쪽 버튼 붙여넣기 · Ctrl+V 붙여넣기 · Shift+Enter 줄바꿈 · OSC 52 · 여러 줄 붙여넣기 확인 · 운영 서버 위험 명령 경고와 패턴 목록 |
+| 단축키 | 단축키 사용(전체 켜기/끄기, 기본 켜짐) · 앱 단축키 34개의 키 바꾸기·지우기·기본값으로 · 모두 기본값으로 · 터미널 안에서 쓰는 키(바꿀 수 없음) |
 | 파일 전송 | 서버 업로드 폴더 · 자동 정리 일수 · 서버 간 복사 임시 폴더 · 경로 따옴표 |
 | 연결 | 자동 재연결 · 마지막 화면 배치 복원 |
 | 로그 | 터미널 출력 기록 · 로그 줄마다 시각 표시 · ANSI 그대로 저장 · 로그 보관 기간 · 로그 파일 갱신 주기 · 로그 폴더 |
 | git 동기화 | 저장소 주소 · 지금 동기화 · 자동 동기화 |
 | 보안 | 자동 잠금(분) · Windows 로그인으로 열기 · 비밀번호 변경 · 저장된 계정 |
 | 백업 | 내보내기 · 가져오기 |
-| 정보 | 버전 · 개발자 · 라이선스 · **사용 형태(라이선스 키)** · 사용 조건 · 업무용 라이선스 문의 · 참고한 프로젝트 · 오픈소스 라이선스 · 데이터 폴더 경로(표시만) · 단축키 표 (업데이트 확인은 지금 꺼져 있어 보이지 않음, D80) |
+| 정보 | 버전 · 개발자 · 라이선스 · **사용 형태(라이선스 키)** · 사용 조건 · 업무용 라이선스 문의 · 참고한 프로젝트 · 오픈소스 라이선스 · 데이터 폴더 경로(표시만) (업데이트 확인은 지금 꺼져 있어 보이지 않음, D80) |
 
 모든 설정은 바꾸는 즉시 저장된다(저장 버튼 없음).
 
@@ -224,6 +225,8 @@
 
 ### 부록: 단축키
 
+아래는 **기본 키**다. 설정 > 단축키에서 명령마다 키를 바꾸거나 지울 수 있고, "단축키 사용"을 끄면 앱 단축키 전체가 꺼져 그 키들이 터미널(서버)로 간다(vi·tmux 와 키가 겹칠 때). 다른 동작이 쓰던 키를 고르면 그 동작의 키는 비워진다. 붙여넣기·복사 키는 바꿀 수 없고 꺼도 동작한다.
+
 | 키 | 동작 |
 |---|---|
 | Ctrl+K (Ctrl+Shift+T, Ctrl+Shift+N) | 빠른 연결 열기/닫기 |
```

### `docs/scenarios.en.md` — 수정

```diff
--- a/docs/scenarios.en.md
+++ b/docs/scenarios.en.md
@@ -174,19 +174,20 @@
 | 12-4 | Keep session with tmux | Server Advanced "Keep session (tmux)": programs on the server keep running after you close the pane or app, and reopening reattaches to the same screen. Without tmux it says so and works normally. On local Windows, WSL shells only. |
 | 12-5 | Local shell exits | "Shell exited" → Enter starts a new shell. |
 
-## 13. Settings (gear icon, 9 sections)
+## 13. Settings (gear icon, 10 sections)
 
 | Section | Main items |
 |---|---|
 | Appearance | Language · app theme (Dark/Light) · 6 terminal color themes (DariTerm Dark·Light, PuTTY Classic, Solarized Dark, Monokai, Nord) · font, size, line spacing · remember window size and position · [Reset to default size] · cursor style · scrollback lines · maximum vertical panes · blinking cursor · bell sound |
 | Input | Copy on select · paste with right click · paste with Ctrl+V · new line with Shift+Enter · OSC 52 · ask before pasting multiple lines · production dangerous-command warning and pattern list |
+| Shortcuts | Use shortcuts (turn all on/off, on by default) · change, clear or reset the key of each of the 34 app shortcuts · reset all to defaults · keys inside the terminal (cannot be changed) |
 | File transfer | Server upload folder · auto-delete days · temp folder for server-to-server copies · quote paths |
 | Connection | Reconnect when disconnected · restore last layout |
 | Logs | Log terminal output · timestamp each log line · keep ANSI codes · keep session logs (days) · log file update interval · log folder |
 | git sync | Repository address · Sync now · automatic sync |
 | Security | Auto-lock (minutes) · unlock with Windows sign-in · change password · saved accounts |
 | Backup | Export · Import |
-| About | Version · developer · license · **Usage (license key)** · terms of use · business license inquiries · referenced project · open source licenses · data folder path (display only) · shortcut table (update checking is currently turned off and hidden) |
+| About | Version · developer · license · **Usage (license key)** · terms of use · business license inquiries · referenced project · open source licenses · data folder path (display only) (update checking is currently turned off and hidden) |
 
 Every setting is saved as soon as you change it (no Save button).
 
@@ -223,6 +224,8 @@
 
 ### Appendix: Keyboard shortcuts
 
+These are the **default keys**. In Settings > Shortcuts you can change or clear the key of each action, and turning "Use shortcuts" off disables all app shortcuts so those keys go to the terminal (server) — useful when vi or tmux use the same keys. If you pick keys another action was using, that action is left without a key. The paste and copy keys cannot be changed and keep working when shortcuts are off.
+
 | Keys | Action |
 |---|---|
 | Ctrl+K (Ctrl+Shift+T, Ctrl+Shift+N) | Open/close Quick connect |
```

`docs/manual/user-guide.ko.md`·`user-guide.en.md` 의 "단축키 요약" 머리에는 위 시나리오 부록과 같은 뜻의 안내 한 문단을 넣었고, `CHANGELOG.md` 의 미배포 절에 새 기능 한 줄과 수정(Ctrl+Shift+Tab) 한 줄, `docs/decisions.md` 에 D126 을 넣었다.

---

## G. 검증 (2026-11-25 실행)

### G.1 프런트 — 새 파일 [frontend/src/lib/keys.test.ts](../../frontend/src/lib/keys.test.ts) 14개 (순수 규칙)

| 테스트 | 확인하는 것 |
|---|---|
| 표의 키가 전과 같은 동작에 걸린다 | 기본 키 30가지(숫자 패드 5가지 포함)가 전과 같은 동작·인자로 걸린다. Ctrl+Shift+Tab → 이전 탭 |
| 표에 없는 조합은 걸리지 않는다 | Ctrl+A·Ctrl+C·Ctrl+V·Ctrl+Shift+K·Ctrl+Shift+C·Ctrl+Shift+1·그냥 K·Alt+Shift+↑ |
| 조합 키만 눌렀거나 Win 키가 끼면 조합이 아니다 | Ctrl 만, Ctrl+Shift 만, Win+Ctrl+K → 빈 글자. 조합 키 순서는 Ctrl+Alt+Shift |
| 기본 키는 서로 겹치지 않고 모두 고를 수 있는 조합이다 | 명령 34개, id·기본 키 중복 없음, 기본 키가 "고를 수 없는 조합"에 걸리지 않음 |
| 조합 글자를 읽기 쉬운 표기로 | `Ctrl+Shift+KeyT` → `Ctrl+Shift+T`, `Alt+ArrowUp` → `Alt+↑`, `Ctrl+Equal` → `Ctrl+=` |
| Ctrl·Alt 없이 누른 보통 키는 안 되고 F 키는 된다 | `a`·`Shift+a`·`Shift+Enter` 는 안 됨 / `F5`·`Shift+F12`·`Alt+X`·`Ctrl+B` 는 됨 |
| 터미널의 복사·붙여넣기 키는 안 된다 | 5개 조합 |
| 바꾸면 새 키로만 동작하고 옛 키는 풀린다 | 저장 값 `{ find: 'Ctrl+Alt+KeyF' }`, 옛 키는 null, 다른 명령은 그대로 |
| **다른 명령이 쓰던 키를 고르면 먼저 쓰던 명령을 빈 값으로 저장한다** | `{ find: 'Ctrl+KeyK', quick: '' }`, 비워진 명령 id 를 돌려줌, Ctrl+K 는 찾기 |
| 이미 바꿔 둔 키끼리 겹쳐도 먼저 쓰던 쪽이 비워진다 | `{ find: '', sftp: 'Ctrl+Alt+KeyX' }` |
| 지우면 빈 값, 기본값으로 되돌리면 설정에서 빠진다 | `{ quick: '' }` → 되돌리면 `{}` |
| 기본값으로 되돌릴 때 그 키를 다른 명령이 쓰고 있으면 그 명령이 비워진다 | `{ find: 'Ctrl+KeyK', quick: '' }` 에서 quick 을 되돌리면 `{ find: '' }` |
| 모르는 명령·글자가 아닌 값은 버리고, 손으로 겹치게 고친 파일은 표의 앞 명령이 이긴다 | — |
| **끄면 어떤 키도 앱 단축키가 아니다** | 명령 34개의 기본 키 모두: 켜면 걸리고 끄면 null |

### G.2 프런트 — 새 파일 [frontend/src/screens/SettingsShortcuts.test.tsx](../../frontend/src/screens/SettingsShortcuts.test.tsx) 18개 (화면)

| 테스트 | 확인하는 것 |
|---|---|
| 구역 목록에 "입력" 다음으로 "단축키"가 있다 (10개) | 화면·입력·**단축키**·파일 전송·연결·로그·git 동기화·보안·백업·정보 |
| "정보"에는 단축키 표가 없다 | "터미널에서 찾기"·`Ctrl+Shift+F` 가 보이지 않는다 |
| 명령 34개가 기본 키와 함께 보이고, 고정 키 3개는 따로 | 줄 34개, `Ctrl+K`·`Ctrl+Shift+Tab`·`Ctrl+3`·`Alt+↑`·`F11`, [모두 기본값으로]는 누를 수 없음 |
| "단축키 사용"은 기본 켜짐, 끄면 저장 | 기본값 true / 누르면 `store.patch` 로 `shortcutsEnabled: false` |
| 키 칸을 누르고 새 키를 누르면 그 명령만 바뀌어 저장 | "키를 누르세요…" 표시, Ctrl 만 누른 것은 기다림, `{ find: 'Ctrl+Alt+KeyF' }` |
| **다른 명령이 쓰던 키를 고르면 먼저 쓰던 명령을 빈 값으로 저장하고 알린다** | `{ find: 'Ctrl+KeyK', quick: '' }` + 알림 문구 |
| 비워진 명령은 "없음", [기본값으로]로 되찾으면 그 키를 쓰던 명령이 비워진다 | `{ find: '' }` |
| [키 지우기]는 빈 값으로 저장 | `{ sftp: '' }` |
| [모두 기본값으로]는 빈 표로 저장 | `{}` |
| 고를 수 없는 키는 저장하지 않고 까닭을 알린 뒤 계속 기다린다 | `a`, Ctrl+V → 알림 2가지, 저장 없음, 계속 "키를 누르세요…" |
| 키를 받는 중의 Esc 는 받기만 취소 | 설정 창은 열려 있음 / 받는 중이 아니면 Esc 가 전처럼 닫음 |
| 기본 설정이면 기본 키가 앱 단축키다 | 저장된 설정 → 키 판정(`activeShortcut`) |
| 전체를 끄면 어떤 키도 앱 단축키가 아니다 | Ctrl+K·F11·Alt+↑ → null |
| 바꾼 키는 새 키로만, 비운 명령은 어떤 키로도 동작하지 않는다 | — |
| 옛 저장 파일처럼 값이 없으면 켜진 것으로 본다 | `shortcutsEnabled`·`shortcuts` 가 없을 때 |
| 시작 화면 기본: 12줄, 묶음 줄은 묶음 표기 | `Alt+방향키`, `Ctrl+Tab / Ctrl+1~9`, 안내 문장의 `Ctrl+K` |
| 시작 화면: 바꾼 키를 보이고 비운 줄은 뺀다 | `Ctrl+Alt+K`, "새 로컬 터미널" 줄 없음, `Ctrl+Alt+↑ …` |
| 시작 화면: 전체를 끄면 고정 키 3줄만, 안내 문장의 키도 없다 | — |

### G.3 그 밖의 자동 테스트

| 테스트 | 확인하는 것 |
|---|---|
| `SettingsLog.test.tsx` (고침) | 구역 순서 기대값을 10개로 |
| `termHelpers.test.ts` (그대로) | 전부터 있던 단축키 매핑 7가지 — 고치지 않고 통과 |
| 계약 테스트(`internal/app`, `frontend/src/ipc/contract.test.ts`) | `store.get` 의 `shortcuts: {}`·`shortcutsEnabled: true` 가 백엔드 응답과 프런트 고정 파일에서 같다 |
| 문구 검사(`go test ./internal/i18n`, `frontend/src/i18n`) | 10개 언어에 같은 키·같은 자리표시자, 화면 코드에 번역을 거치지 않은 한국어가 없다 |

### G.4 실행한 명령

```bash
go build ./... && go vet ./internal/store/ ./internal/app/      # 통과
go test ./...                          # 아래 한 건을 빼고 전부 통과
(cd frontend && npx tsc --noEmit)      # 통과
(cd frontend && npx vitest run)        # PowerShell 에서. 51개 파일 343개 통과 (앞 문서까지 49개 파일 311개 + 이 문서 32개)
go run github.com/wailsapp/wails/v2/cmd/wails@v2.17.1 build   # build/bin/dariterm.exe
```

`go test ./...` 한 번에서 `internal/clipboard` 의 `TestClipboardInfoIsFastForText` 가 실패했다("텍스트 판정이어야 함: empty"). 이 테스트는 PC 의 실제 클립보드에 글을 넣고 곧바로 다시 읽는데, 그 사이 다른 프로그램이 클립보드를 건드리면 실패한다. 이 문서는 그 패키지를 고치지 않았고, 그 패키지만 두 번 다시 돌렸을 때는 통과했다(`go test -count=1 ./internal/clipboard/`).

자동 테스트가 확인하지 못하는 것:
- 실제 키보드 입력이 앱에서 터미널로 넘어가는지(전체 끄기·비운 키로 vi 가 그 키를 받는지). 테스트는 "앱 단축키가 아니다"라는 판정까지만 본다.
- 실제 창에서 키 받기가 어떻게 보이는지(강조색 테두리, 다른 곳을 눌렀을 때의 취소, Alt 를 누르는 동안의 Windows 메뉴 동작).
- 구역이 10개가 됐을 때 내비게이션이 넘치지 않는지, 34줄 목록의 스크롤.
- `Ctrl+Tab`·`Ctrl+Shift+Tab`·`F11` 처럼 웹뷰나 Windows 가 먼저 쓸 수 있는 키가 받아지는지.

## H. 수동 검증 (남은 일)

| # | 절차 | 기대 결과 |
|---|---|---|
| 1 | 설정을 연다 | 왼쪽에 구역 10개. "입력" 다음에 "단축키"(⌘ 아이콘). 목록이 잘리지 않는다 |
| 2 | "정보"를 누른다 | 단축키 표가 없다. 나머지는 전과 같다 |
| 3 | "단축키"를 누른다 | "단축키 사용"(켜짐), 앱 단축키 34줄(기본 키), [모두 기본값으로](누를 수 없음), 아래에 "터미널 안에서 쓰는 키" 3줄 |
| 4 | 아무것도 바꾸지 않고 평소 쓰던 단축키들을 눌러 본다(Ctrl+K, Ctrl+Shift+D, Alt+방향키, Ctrl+1~9, Ctrl+=, F11 …) | 전과 같다 |
| 5 | 탭 3개에서 Ctrl+Shift+Tab | 이전 탭으로 간다(전에는 동작하지 않았다) |
| 6 | "터미널에서 찾기"의 키 칸 → Ctrl+Alt+F | 칸이 `Ctrl+Alt+F` 로 바뀐다. 터미널에서 Ctrl+Alt+F 는 찾기 막대, Ctrl+Shift+F 는 아무 일 없음. 칸 우클릭 메뉴의 찾기 옆에 `Ctrl+Alt+F` |
| 7 | "터미널에서 찾기"의 키 칸 → **Ctrl+K** | 찾기 = `Ctrl+K`, "빠른 연결 / 명령" = "없음", 알림 'Ctrl+K 를 쓰던 "빠른 연결 / 명령"의 단축키를 비웠습니다'. 시작 화면의 안내 문장과 표가 다른 키(`Ctrl+Shift+T`)로 바뀐다 |
| 8 | 데이터 폴더의 store.json 을 열어 본다 | `"shortcuts": { "find": "Ctrl+KeyK", "quick": "" }` |
| 9 | "빠른 연결 / 명령"의 ↺ | 빠른 연결 = `Ctrl+K`, 찾기 = "없음" |
| 10 | 키 받는 중에 `a` / Ctrl+V / Esc | 오류 알림 후 계속 기다림 / 같음 / 받기만 취소되고 설정 창은 열려 있다 |
| 11 | 키 받는 중에 Ctrl+Shift+D | 창이 나뉘지 않고 그 조합이 그 명령의 키가 된다("같은 서버를 오른쪽에"는 "없음") |
| 12 | [모두 기본값으로] | 모든 줄이 기본 키. store.json 의 `"shortcuts": {}` |
| 13 | **"단축키 사용"을 끄고** 서버에서 `vim` 을 연다. 입력 모드에서 Ctrl+K 다음 `a` `:` | 앱의 빠른 연결이 뜨지 않고 vim 이 `ä` 를 넣는다(이중 글자 입력) |
| 14 | 꺼 둔 채로 Ctrl+Shift+T, Ctrl+Shift+D, Alt+방향키, Ctrl+Tab, Ctrl+1, Ctrl+=, F11 | 앱은 아무 반응이 없다. 화면이 확대되거나 브라우저 기능이 뜨지 않는다 |
| 15 | 꺼 둔 채로 글자를 선택해 Ctrl+Shift+C, 다른 곳에 Ctrl+V, Shift+Insert, Ctrl+Alt+V | 복사·붙여넣기는 된다 |
| 16 | 꺼 둔 채로 시작 화면(탭을 모두 닫음)과 칸 우클릭 메뉴 | 시작 화면 표는 붙여넣기·이미지·복사 3줄만, 안내 문장 없음. 메뉴의 찾기·나누기·SFTP 옆에 키가 없다(복사·붙여넣기 옆에는 있다). 메뉴 항목 자체는 마우스로 동작 |
| 17 | 다시 켠다 | 13~14 의 키들이 곧바로 앱 단축키로 돌아온다(터미널을 다시 열지 않아도) |
| 18 | "위쪽 창으로"만 지우고 창을 위아래로 나눈 뒤 Alt+↑ / Alt+↓ | Alt+↑ 는 터미널로 가고(셸에 `^[[1;3A` 류), Alt+↓ 는 아래 창으로 포커스 이동 |
| 19 | 키를 바꾼 채 앱을 껐다 켠다 | 바꾼 키가 그대로다 |
| 20 | 다른 언어(영어 등)로 바꿔 본다 | 구역 "Shortcuts", "Use shortcuts", 명령 이름·버튼·알림이 그 언어로 나온다 |

## I. 영향 범위·회귀 위험

| 영역 | 위험 | 대비 |
|---|---|---|
| 기본 상태의 단축키 | 키 판정을 if 가지에서 표 찾기로 바꿨다. 가지마다 달랐던 세부(예: Shift 가 눌렸을 때 무시하던 키)가 달라질 수 있다 | 기본 키 30가지 + 걸리지 않아야 하는 8가지 테스트, 전부터 있던 매핑 테스트 통과. 달라진 것은 §2.10 의 두 가지 |
| F11 | Ctrl+F11·Shift+F11·Alt+F11 이 더는 창 최대화가 아니다 | §2.10 에 적었다. 필요하면 그 조합을 명령에 줄 수 있다 |
| Ctrl+Shift+Tab | 전에는 터미널로 갔다(보낼 것이 없는 조합). 이제 이전 탭 | 명세대로다. 싫으면 그 키를 지운다 |
| AltGr 자판(독일어 등) | AltGr 은 Ctrl+Alt 로 들어온다. AltGr+T 가 "새 로컬 터미널"(Ctrl+Alt+T)에 걸리는 것은 전과 같다 | 이제 그 키를 바꾸거나 지울 수 있다 |
| 전체 끄기 | F11 과 빠른 연결 키까지 꺼진다. 마우스로만 열 수 있다 | 토글 설명에 "하나만 겹치면 그 단축키만 바꾸거나 지우세요"라고 적었다. 탭 막대의 +, 시작 화면 버튼, 메뉴는 그대로 동작 |
| 사용자가 꼭 필요한 키를 고름 | Ctrl+C·Ctrl+D·Ctrl+Z 를 앱 단축키로 주면 그 키가 서버로 가지 않는다 | 막지 않았다(§2.5). 줄마다 [기본값으로], 위에 [모두 기본값으로] |
| 키 받기 중 Alt | Alt 만 눌렀다 떼면 Windows 가 창 메뉴 쪽으로 포커스를 옮길 수 있다 | 수동 검증 10·11. 창이 포커스를 잃으면 받기를 취소하게 했다 |
| 웹뷰·Windows 가 먼저 쓰는 키 | Alt+F4, Alt+Space, Win 조합은 앱까지 오지 않는다. 골라도 동작하지 않는다 | Win 조합은 받지 않는다. 나머지는 막지 않았다 |
| 키 받는 중의 Esc | 모달의 Esc 닫기를 `keyCapture.active` 일 때 쉬게 했다. 받기가 끝났는데 값이 남으면 Esc 로 설정이 닫히지 않는다 | 받기를 끝내는 모든 길(성공·Esc·다른 곳 누름·포커스 잃음·구역을 떠남)에서 값을 내린다. 테스트가 "받는 중이 아니면 Esc 가 닫는다"를 본다 |
| 메뉴·시작 화면의 키 표기 | 표기를 지금 키에서 얻는다. 설정이 바뀌면 다시 그린다 | 시작 화면 테스트 3개. 메뉴는 수동 검증 6·16 |
| 설정 화면의 다른 구역 | 구역 목록에 한 줄이 늘었고 "정보"에서 표를 뺐다 | `tsc`(구역 id 를 빠뜨리면 컴파일 오류), 구역 순서 테스트, 정보 화면 테스트 통과 |
| 저장 값 | 옛 저장 파일은 두 키가 없어 기본값으로 채워진다. 이 버전에서 저장한 파일을 옛 버전이 읽으면 두 키를 무시한다 | 계약 테스트, `store` 테스트 |
| 동기화 | `shortcuts` 도 설정과 함께 다른 PC 로 간다. 두 PC 에서 따로 바꾼 것이 합쳐지며 겹칠 수 있다 | 읽을 때 표의 앞 명령이 이긴다(§2.3). 화면에는 뒤 명령이 "없음"으로 보인다 |
| 문구 | 쓰지 않게 된 키 2개(`about.shortcuts`, `sc.fontSize`)를 10개 파일에서 지웠다 | 다른 코드(Go·도구)에서 쓰지 않는 것을 검색으로 확인. 문구 검사 통과 |
| 번역 | 한국어·영어 외 8개 언어의 문구 31개는 이 작업을 한 세션이 번역했다 | 검수가 필요할 수 있다 |
| 설명서 PDF·스토어 스크린샷 | `docs/manual/DariTerm-user-manual.*.html`·`.pdf` 는 다시 만들지 않았다(2026-11-18 것 그대로). 설정 > 정보 화면을 찍은 자료가 있으면 단축키 표가 빠진 모습으로 바뀐다 | 원문(docs/scenarios 한·영, user-guide 한·영)은 고쳤다. `node tools/manual/build.mjs [ko|en]` 로 다시 만든다 |

되돌리기: §B·§C·§D·§F 의 `-` 줄로 되돌리고 새 파일 4개를 지운다(`keys.ts`·`shortcuts.ts` 는 전의 내용으로). §E 의 문구는 넣은 줄을 지우고 지운 줄 2개를 되살린다. 저장 파일에 남은 `shortcutsEnabled`·`shortcuts` 는 읽는 코드가 없으면 무시된다.

## J. 체크리스트

- [x] `lib/keys.ts`: 명령 표 34개, 조합 글자(`comboOf`·`comboText`), 표 찾기(`buildKeyMap`·`shortcutFor`), 겹침 처리(`assignShortcut`), 고를 수 없는 조합(`comboProblem`)
- [x] `state/shortcuts.ts`: 설정 → 키 판정(`activeShortcut`)·표기(`useShortcutLabels`)
- [x] `App.tsx`·`TerminalView.tsx`: 설정을 따르는 판정, 키 받는 중 쉬기. `Modal.tsx`: 키 받는 중의 Esc
- [x] `SettingsShortcuts.tsx` 새로, `Settings.tsx` 구역 목록, `SettingsAbout.tsx` 에서 표 빼기, 구역 id `shortcuts`
- [x] 시작 화면·칸 메뉴·탭 메뉴·툴팁의 키 표기를 지금 키로
- [x] 설정 2개: 프런트 타입·기본값, `store.go` 기본값, 계약 고정 파일 2개
- [x] 문구 31개 추가·2개 삭제(10개 언어)
- [x] 프런트 테스트 32개 추가, 1개 고침. `tsc`·`vitest`(343개)·`go test`(클립보드 한 건은 §G.4)·빌드
- [x] 명세 behavior/shortcuts.md §5·ui/settings.md §2a·ui/shell.md §4·data/store.md·acceptance K8~K12·G6, 시나리오 ko·en, 사용 설명 ko·en, CHANGELOG, decisions D126
- [ ] 수동 검증 1~20
