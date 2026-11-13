# 구현자 → 명세 작성자 질문 큐

형식: `Q<번호> · 상태(open/answered) · 관련 문서 · 질문`. 답은 이 파일이 아니라 **해당 명세 본문**에 반영한 뒤 상태만 `answered` 로 바꾼다.

| # | 상태 | 문서 | 질문 |
|---|---|---|---|
| Q1 | answered | data/vault.md §6 | 복구·비밀번호 변경에서 새 비밀번호 최소 길이를 설정과 같은 8자로 검사해도 되는가? 명세는 설정(Setup)에만 8자 규칙을 적었다. 현재 구현은 8자 검사를 적용하고 `decisions.md` D12 에 기록. |
| Q2 | answered | data/vault.md §6 (저장) | 계정 저장에서 빈 사용자 이름·비밀번호는 "지정 안 함"으로 기존 값을 유지한다. 기존 사용자 이름을 **지우는** 방법이 명세에 없다. 현재 구현은 지울 수 없다(빈 값 = 유지). |
| Q3 | answered | data/vault.md §6 (저장) | 항목 메타의 사용자 이름은 평문 필드와 비밀 객체 안 양쪽에 둔다(구현). 명세는 "사용자 이름 (평문, 비면 생략)" 과 "비밀 객체의 username" 을 둘 다 적었다. 두 값이 다를 때 어느 쪽이 우선인지 명세에 없다. |
| Q4 | answered | data/vault.md §6 (잠금) | 이미 잠긴 볼트에 잠금 명령을 보내면 "잠김" 알림을 보내는가? 구현은 전이 때만 보낸다(D13). |
| Q5 | answered | behavior/lifecycle.md §1, sync-backup.md §1 | store.json 이 깨져 있고 `.bak` 도 깨졌을 때: 명세는 "앱이 오류로 종료"(lifecycle §1)와 "둘 다 없으면 없음"(sync-backup §1)이 있다. 깨짐과 없음을 같은 것으로 보고 빈 저장소로 시작해야 하는가, 종료해야 하는가? 현재 구현은 깨진 store 를 종료 오류로 본다. vault 는 깨짐 오류를 종료로 본다. |
| Q6 | answered | spec/README.md 알려진 공백 (Windows 클립보드·SSH 에이전트·tmux 감지) | 본 라운드(골격·저장·볼트·IPC)에는 해당 채널이 없어 미정. 세션 단계에서 필요. |
| Q7 | answered | ipc.md §2 (vault.status) | `osUnlockKind` 는 OS 잠금 해제가 가능할 때만 `windows` 로 채운다(구현). 가능 여부 자체가 false 인 환경의 값 표기 규칙이 명세에 없다. |
| Q8 | answered | data/sync-backup.md §1 | 데이터 폴더 기본 위치를 `os.UserConfigDir()`(Windows: `%APPDATA%`)로 구현했다. 다른 OS 의 기본 위치 규칙은 명세에 없다(1차 범위 밖). |
| Q9 | answered | ipc.md §1·§4 | Wails 전역 객체 이름과 이벤트 구독 함수가 명세에 없다. 구현은 Wails v2 관례(`window.go.main.App`, `window.runtime.EventsOn`)를 가정했다. 또 `session.data` 처럼 값이 둘인 이벤트의 전달 방식(인자 여러 개인지 객체 하나인지)을 정해 달라. |
| Q10 | answered | ui/shell.md §3·§4, ui/dialogs.md §2.1 | 로컬 터미널·시리얼 세션의 "대상 문자열" 표기가 없다. 구현은 로컬을 빈 문자열, 시리얼을 주소만으로 둔다. |
| Q11 | answered | ui/lock.md §1, ipc.md §2 (vault.status) | "Windows 로그인으로 열기" 버튼의 표시 조건이 `osUnlockEnabled` 인지, `osUnlockAvailable` 도 함께 봐야 하는지 명세에 없다. 구현은 `osUnlockEnabled` 만 본다. |
| Q12 | answered | ui/lock.md §1, ipc.md §2 (vault.unlock) | `vault.unlock` 이 거짓(false)이 아니라 오류를 던질 때(예: 볼트 손상) 표시할 문구가 없다. 구현은 오류 메시지를 그대로 보인다. |
| Q13 | answered | ui/shell.md §2, ui/tokens.md §3 | 제목 표시줄 오른쪽 "창 컨트롤용 여백 약 140px" 과 "운영체제 기본 프레임" 의 관계가 불명확하다(기본 프레임이면 창 버튼이 제목 표시줄 밖에 있는데 왜 여백이 필요한지). 구현은 명세대로 여백을 둔다. |
| Q14 | answered | ui/lock.md §4 | 복구 코드 입력 중 하이픈을 자동으로 넣는지 여부가 없다(명세는 대문자 변환만). 구현은 입력을 그대로 두고 제출 시에만 하이픈·공백을 뺀다. |
| Q15 | answered | ipc.md §2 (session.open), behavior/session.md | 로컬 터미널을 여는 요청 형식이 명세에 없다. 구현은 `session.open` 의 adhoc 에 `{host: "", protocol: "local"}` 을 보낸다. 로컬 셸·시작 폴더를 함께 보내는 형식을 정해 달라. |
| Q16 · open · protocols.md §3 · sftp.list 항목의 "권한(하위 9비트 숫자)"과 "권한 문자열" 중 어느 쪽을 `mode`, `perm` 필드로 주는가? 현재 구현은 mode=문자열, perm=숫자로 가정했다 (decisions D26). |
| Q17 · open · ipc.md §4 (transfer.update) · 전송 `state` 의 값 목록이 없다. 구현은 queued/running/done/error 로 가정한다 (D27). 취소 상태 이름도 알려 달라. |
| Q18 · open · ui/sftp-snippets.md §2.4 · 여러 항목을 선택한 상태의 우클릭 "열기/미리보기" 동작이 명세에 없다. 현재는 비활성으로 두었다 (D29). |
| Q19 · open · ui/sftp-snippets.md §4 · 스니펫 검색 결과가 0개일 때(스니펫은 있음) 보일 문구가 없다. 현재는 빈 목록만 보인다. |
| Q20 · open · ui/settings.md §1 · 글자 크기·줄 간격의 범위 밖 값: 본문은 "범위 밖이거나 비면 14", 수용 G2 는 100 입력 시 32 로 제한. 구현은 경계 제한(초과·미만), 빈 값·숫자 아님은 기본값으로 함. 맞는지 확인 요청. |
| Q21 · open · ipc.md §4 · `sync.status` 이벤트의 값 형태(상태·메시지·마지막 동기화 시각·변경됨)가 `sync.run` 결과와 같은지 명세에 없다. 구현은 같은 형태로 읽는다. |
| Q22 | answered | data/sync-backup.md §5.4 · behavior/lifecycle.md §2 | 새 PC(볼트 미설정)에서 "저장소의 보관함 받기"를 하려면 볼트가 열려 있어야 하지만, §5.4 사전 검사 3은 잠긴 상태를 `locked` 로 먼저 막는다. 받기 전에 빈 보관함을 만들도록 안내하는 절차(또는 받기 전용 예외)를 명세에 적어 달라. 현재 구현은 명세대로 `locked` 로 막는다(임시). |
| Q23 | answered | behavior/lifecycle.md §4 | 시작 창 위치 판정 기준이 없다. Wails 화면 목록에는 화면 원점이 없어 보조 모니터 위치를 알 수 없다. 현재 구현은 주 화면 크기 안에 걸치는지만 본다(D40). 이 기준으로 충분한지, 다른 기준이 필요한지 정해 달라. |
| Q24 | answered | ipc.md §3 (ui.request) | `ui.request` 요청 객체에 요청 id 를 담는 키 이름이 명세에 없다. 구현은 `requestId` 로 통일했다(D45). 명세에 이 이름을 적어 달라. |
| Q25 | answered | ipc.md §2 (session.open)·§4 (session.state) | 세션 상태의 토큰 이름(연결 중·연결됨·재접속·끊김·오류)이 명세에 없다. 구현은 `connecting/connected/reconnecting/disconnected/error` 를 쓴다(D46). 닫힌 세션의 끊김 상태 이름을 정해 달라. |
| Q26 | answered | ipc.md §2 (sync.status) | 동기화를 한 번도 하지 않았을 때 `sync.status` 의 `last` 값이 없음(`null`)인지, 아니면 필드를 빼는지 명세에 없다. 구현은 `null` 을 보낸다. |

> Q10 답: 대상 문자열은 이미 `spec/behavior/session.md` §1 에 있다(시리얼 `{장치} @ {속도}`, 로컬 `로컬 · {시작 폴더, 비면 ~}`). 세션 정보의 `target` 은 **백엔드가 만들어 내려주므로** 프런트는 그대로 표시하면 된다.
