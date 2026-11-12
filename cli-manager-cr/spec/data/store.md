# 데이터 형식 — store (서버·폴더·스니펫·설정·레이아웃)

> 작성 방식: 기존 앱의 저장 데이터 구조 관찰 + 열람(호환에 필요한 형식 사실만 기록). 작성일 2026-11-10. 상태: **draft** (파일 위치·원자 저장·버전 이전 규칙은 다음 라운드에서 보강).
> 이 문서의 JSON 필드 이름과 값은 **데이터 호환을 위한 사실**이다. 구현 구조(타입 이름, 모듈)는 규정하지 않는다.

## 1. 최상위 구조

저장 파일은 JSON 객체 하나이며 다음 키를 가진다. **알 수 없는 키는 읽고 그대로 다시 써서 보존한다.** 배열 키(`groups`·`hosts`·`snippets`·`knownHosts`)는 **null 이 되지 않는다** — 없거나 null 이면 빈 배열로 읽고 저장한다(화면이 `.map` 으로 읽기 때문).

| 키 | 형 | 설명 |
|---|---|---|
| version | 숫자 | 형식 버전 |
| groups | 폴더 배열 | 서버 폴더(중첩 가능) |
| hosts | 서버 배열 | 서버 목록 |
| snippets | 스니펫 배열 | 자주 쓰는 명령 |
| knownHosts | 배열 | 신뢰한 SSH 호스트 키 |
| settings | 객체 | 앱 설정 (§4) |
| layout | 객체(선택) | 저장된 탭·분할 배치 (§5) |

## 2. 폴더 (groups[])

| 필드 | 형 | 설명 |
|---|---|---|
| id | 문자열(UUID v4) | |
| parentId | 문자열 또는 null | 최상위는 null |
| name | 문자열 | |
| color | 문자열(선택) | `#rrggbb` |
| env | 환경 값(선택) | §6 |
| collapsed | 불리언(선택) | 사이드바에서 접힘 |
| sort | 숫자 | 정렬 키(작을수록 앞) |

## 3. 서버 (hosts[])

| 필드 | 형 | 설명 |
|---|---|---|
| id | 문자열(UUID v4) | |
| groupId | 문자열 또는 null | 소속 폴더 |
| alias | 문자열 | 표시 이름 |
| protocol | `ssh` `telnet` `serial` `local` | |
| host | 문자열 | 주소 |
| port | 숫자 | 새 서버 기본 22 |
| username | 문자열 | |
| authType | `password` `key` `agent` `ask` | 새 서버 기본 `password` |
| credentialId | 문자열 또는 null(선택) | 볼트 항목 id |
| jumpHostId | 문자열 또는 null(선택) | 점프 호스트 서버 id. 값이 `direct` 이면 jumpDirect 사용 |
| jumpDirect | 객체(선택) | host, port, username, authType(`password`/`key`), credentialId(선택) |
| color | 문자열(선택) | |
| env | 환경 값(선택) | |
| tags | 문자열 배열 | |
| encoding | `utf-8` `euc-kr` `cp949` `shift_jis` `latin1` | 새 서버 기본 `utf-8` |
| termType | 문자열 | 새 서버 기본 `xterm-256color` |
| startupCommand | 문자열(선택) | 접속 직후 보낼 명령 |
| keepSession | 불리언(선택) | 세션 유지(tmux) |
| tmuxName | 문자열(선택) | 비우면 서버 별칭(로컬은 폴더 이름) |
| keepaliveSec | 숫자 | 새 서버 기본 30 |
| forwards | 배열 | 포트 포워딩: id, type(`L`/`R`), bindHost, bindPort, destHost, destPort |
| serial | 객체(선택) | path, baudRate, dataBits(5/6/7/8), parity(`none` `even` `odd` `mark` `space`), stopBits(1/1.5/2), flowControl(`none` `rtscts` `xonxoff`), enterSends(`CR` `LF` `CRLF`), localEcho |
| local | 객체(선택) | shell, cwd (비우면 기본 셸·홈 폴더) |
| notes | 문자열(선택) | |
| favorite | 불리언(선택) | |
| lastUsedAt | 숫자(선택) | 밀리초 타임스탬프 |
| sort | 숫자 | 정렬 키. 새 서버는 생성 시각(밀리초) |

## 4. 설정 (settings)

저장된 값 위에 **없는 키만 기본값으로 채운다.** 알 수 없는 키는 보존한다.

| 키 | 기본값 | 설명 |
|---|---|---|
| theme | `dark` | `dark` / `light` (앱 외형) |
| terminalTheme | `TermGo Dark` | 터미널 색 테마 이름 (ui/tokens.md) |
| fontFamily | `'Cascadia Mono', 'D2Coding', Consolas, 'Malgun Gothic', monospace` | |
| fontSize | 14 | 허용 8~32 |
| lineHeight | 1.15 | |
| cursorStyle | `block` | `block` `bar` `underline` |
| cursorBlink | true | |
| scrollback | 10000 | 줄 수 |
| copyOnSelect | true | 선택하면 복사 |
| rightClickPaste | true | 우클릭 붙여넣기 |
| ctrlVPaste | true | |
| confirmMultilinePaste | true | 여러 줄 붙여넣기 확인 |
| guardDangerousOnProd | true | 운영 환경 위험 명령 확인 |
| dangerousPatterns | 아래 목록 | 정규식 문자열 배열 |
| autoReconnect | true | |
| autoLockMinutes | 15 | 0 이면 자동 잠금 없음 |
| uploadDir | `~/.termgo/uploads` | 붙여넣기 이미지·파일 업로드 폴더 |
| uploadCleanupDays | 7 | |
| maxVerticalSplits | 3 | 탭 하나의 최대 칸 수(상하 분할 한도), 1~5 ([../ui/panes.md](../ui/panes.md) §2) |
| transferTempDir | `""` | 서버 ↔ 서버 복사 때 거쳐 가는 이 PC 임시 폴더. 빈 값 = OS 임시 폴더 아래 termgo-transfer ([../behavior/file-panel.md](../behavior/file-panel.md) §4) |
| insertPathQuote | true | 경로 삽입 시 따옴표 |
| sessionLog | false | |
| bell | false | |
| shiftEnterNewline | true | Shift+Enter 로 줄바꿈 입력 |
| rememberWindow | true | 창 크기·위치 기억 |
| restoreLayout | true | 탭·분할 복원 |
| updateCheck | true | 새 버전 자동 확인 |
| syncRepo | 빈 문자열 | git 동기화 저장소 주소 |
| syncAuto | true | 자동 동기화 |
| osc52Copy | true | 원격의 OSC 52 클립보드 쓰기 허용 |
| sidebarWidth | 260 | px |
| rightPanelWidth | 340 | px |

dangerousPatterns 기본 목록(정규식 원문, 순서 유지):
`rm\s+-[a-zA-Z]*r[a-zA-Z]*f`, `rm\s+-[a-zA-Z]*f[a-zA-Z]*r`, `\breboot\b`, `\bshutdown\b`, `\bhalt\b`, `\bpoweroff\b`, `mkfs`, `dd\s+if=`, `\bDROP\s+(TABLE|DATABASE)\b`, `\bTRUNCATE\b`, `systemctl\s+(stop|restart)`

호환 규칙: 저장된 `terminalTheme` 이 `BrightTerm ` 으로 시작하면 `TermGo ` 로 시작하도록 접두어를 바꿔 읽는다.

## 5. 저장 레이아웃 (layout)

| 필드 | 설명 |
|---|---|
| tabs | 탭 배열. 각 탭: title(선택), broadcast(선택), group(선택, 오른쪽 그룹이면 1), root(노드) |
| active | 왼쪽(또는 유일한) 그룹의 선택 탭 인덱스 |
| ratio | (선택) 나뉘었을 때 왼쪽 그룹 너비 비율(%). 50 이면 생략 |
| active1 | (선택) 오른쪽 그룹의 선택 탭 인덱스. 그룹 정보가 없는 옛 저장본은 모두 왼쪽 그룹으로 읽는다 |

노드는 둘 중 하나다.
- 칸: `type`=`pane`, hostId(선택), local.cwd(선택), empty(선택, 빈 칸이면 true)
- 분할: `type`=`split`, dir(`row`/`col`), sizes(숫자 배열, 자식 수와 같음), children(노드 배열)

비밀번호나 화면 내용은 저장하지 않는다.

## 6. 환경(env)과 색

| 값 | 라벨 | 색 |
|---|---|---|
| prod | 운영 | `#ef4444` |
| stage | 기능테스트 | `#f59e0b` |
| perf | 성능테스트 | `#a855f7` |
| dev | 개발 | `#22c55e` |
| device | 장비 | `#3b82f6` |
| none | 일반 | `#64748b` |

색 선택 팔레트(14색, 순서 유지): `#ef4444` `#f97316` `#f59e0b` `#eab308` `#22c55e` `#10b981` `#14b8a6` `#06b6d4` `#3b82f6` `#6366f1` `#8b5cf6` `#d946ef` `#ec4899` `#64748b`. 팔레트 맨 앞에 "자동(환경/폴더 색)" 항목이 있으며 선택하면 색 값을 지운다.

## 7. 스니펫 (snippets[])

id, name, body, sendEnter(불리언).

## 8. knownHosts[]

hostPort(`host:port`), keyType, fingerprint(`SHA256:` + base64), addedAt(밀리초).

## 9. 자격 증명 메타(볼트와의 경계)

목록 조회용 정보: id, name, kind(`password`/`key`), username(선택), updatedAt. 비밀(password, privateKey, passphrase)은 볼트에만 있다 (data/vault.md).
