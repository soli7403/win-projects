# 프런트엔드 ↔ 백엔드 계약 (IPC)

> 작성 방식: 기능 목록을 열람해 **새 이름 체계로 정의**한 계약. 작성일 2026-11-10. 상태: **draft**.
> 이 계약은 프런트(B 단계)와 백엔드(C 단계)를 따로 구현해도 맞물리도록 하는 인터페이스이다. 채널 이름은 이 프로젝트의 것이며 기존 앱과 다르다. 데이터 구조(필드)는 data/*.md 를 따른다.

## 1. 전송 방식 (Wails)

백엔드는 프런트에 다음 4개 메서드를 노출한다(전역 객체로 호출).
| 메서드 | 인자 | 설명 |
|---|---|---|
| `Invoke(channel, argsJson)` | 채널 이름, **JSON 배열 문자열** | 요청-응답. 결과는 JSON 문자열(값이 없으면 빈 문자열 → 프런트는 `undefined`). 오류는 거부(reject)되며 메시지 문자열을 쓴다 |
| `TermInput(sessionId, text)` | | 터미널 입력(UTF-8 문자열). 응답 없음 |
| `TermResize(sessionId, cols, rows)` | | 터미널 크기. 응답 없음 |
| `Reply(requestId, valueJson)` | | 백엔드가 낸 질문(`ui.request`)에 대한 답. 응답 없음 |

**프런트에서 부르는 방법(Wails v2 규약, 고정)**: 위 4개 메서드는 Go 의 `main` 패키지에 있는 구조체 `App` 에 바인딩하며, 프런트에서는 전역 객체 `window.go.main.App.Invoke(...)`, `.TermInput(...)`, `.TermResize(...)`, `.Reply(...)` 로 부른다. 이벤트 구독은 `window.runtime.EventsOn(이벤트이름, 콜백)` 이고 **구독 해제 함수를 돌려준다**. 백엔드는 이벤트를 `EventsEmit(이름, 인자…)` 로 보내며 **값이 여러 개인 이벤트는 인자를 따로따로 전달**한다(객체 하나로 묶지 않는다): `session.data` 는 콜백이 `(세션 id, 텍스트)` 두 인자, `vault.changed` 는 `(불리언)`, 나머지는 §4 표의 값 하나(객체)를 첫 인자로 받는다. 값이 없는 이벤트(`store.changed`)는 인자 없음.

백엔드→프런트는 Wails **이벤트**로 보낸다(§4).

- 알 수 없는 채널: 오류 "구현되지 않은 채널: {채널}". 인자 JSON 해석 실패: "인자 해석 실패({채널}): …". 핸들러 내부에서 패닉이 나면 앱이 멈추지 않고 오류 "내부 오류({채널}): {내용}" 로 바꿔 돌려준다.
- 바이너리 인자는 `{"__b64": "<base64>"}` 객체로 감싼다(스니펫·붙여넣기 버퍼 등).
- 프런트 어댑터는 `Invoke` 거부 시 문자열 또는 오류 메시지를 `Error` 로 바꿔 던진다.

## 2. 요청 채널 (이름 → 인자 → 결과)

### 저장 데이터
| 채널 | 인자 | 결과 |
|---|---|---|
| `store.get` | | 저장 데이터 객체(data/store.md, settings 는 기본값 병합 후) |
| `store.patch` | 객체(최상위 키 → 값) | 없음. 키 단위로 덮어쓰기, 지연 저장 |

### 볼트·계정
| 채널 | 인자 | 결과 |
|---|---|---|
| `vault.status` | | `{initialized, unlocked, osUnlockAvailable, osUnlockEnabled, osUnlockKind?}` |
| `vault.setup` | 비밀번호 | `{recoveryCode}` / 오류 |
| `vault.unlock` | 비밀번호 | true/false |
| `vault.unlockOs` | | true/false |
| `vault.recover` | 복구 코드, 새 비밀번호 | true/false |
| `vault.lock` | | 없음 |
| `vault.changePassword` | 현재, 새 | true/false |
| `vault.setOsUnlock` | 불리언 | 없음 / 오류 |
| `cred.list` | | 계정 메타 배열 |
| `cred.get` | id | 비밀 객체 또는 null / 오류(잠김) |
| `cred.save` | 입력 객체(id?, name, kind, username?, password?, privateKey?, passphrase?; **빠졌거나 null 인 필드는 기존 값 유지, 빈 문자열은 지우기**) | 계정 메타 / 오류 |
| `cred.remove` | id | 없음 |

### 세션
| 채널 | 인자 | 결과 |
|---|---|---|
| `session.open` | `{hostId?, adhoc?, cols?, rows?}` (`adhoc`: host, cwd?, port?, username?, protocol?) | 세션 정보 `{id, hostId|null, title, protocol, target, color?, env?, state, connectedAt?, message?, canSftp, keepSession?}`. 서버 id 도 즉석 정보도 없으면 오류 "서버를 찾을 수 없습니다". **연결은 바로 시작하지 않고 약 30ms 뒤** 시작한다(프런트가 터미널을 등록할 시간). **로컬 터미널**: 저장된 로컬 서버는 `hostId` 로 열고(셸·시작 폴더는 서버 설정의 `local` 값), 즉석 로컬은 `adhoc: {host: "", protocol: "local", cwd?: 시작 폴더}` 로 연다 — 즉석 로컬에는 셸 지정이 없고 기본 셸을 쓴다. 즉석 SSH/텔넷은 `{host, port?, username?, protocol?}` |
| `session.close` | id | 없음. SFTP 캐시도 정리 |
| `session.reconnect` | id | 없음 |
| `local.shells` | | `[{path, label}]` (첫 항목이 기본) |
| `local.tmuxAvailable` | | 불리언 |
| `serial.list` | | `[{path, label}]` |

### 클립보드·대화상자·앱
| 채널 | 인자 | 결과 |
|---|---|---|
| `clip.writeText` / `clip.readText` | 텍스트 / | 없음 / 텍스트 |
| `clip.info` | | `{type: files\|text\|image\|empty, files?, text?}` — 우선순위: 존재하는 파일 목록 → 텍스트 → 이미지 → 비어 있음 |
| `clip.hasImage` | | 불리언 |
| `dialog.readKeyFile` | | `{path, text}` 또는 null(취소). 대화상자 제목 "개인 키 파일 선택", 시작 폴더 `~/.ssh`, 숨김 파일 표시 |
| `dialog.chooseDir` | | 경로 또는 null |
| `backup.export` / `backup.import` | | 경로 또는 false / true 또는 false (data/sync-backup.md §3) |
| `app.info` | | `{version, platform, dataDir}` |
| `app.titlebar` | 배경색, 글자색 | 없음(현재 동작 없음, 창은 기본 프레임) |
| `app.openLogs` | | 없음. 로그 폴더(없으면 생성)를 탐색기로 연다 |
| `app.openExternal` | URL | 없음. **`http://` 또는 `https://` 로 시작하는 주소만** 기본 브라우저로 연다, 그 외는 무시 |
| `app.resetWindow` | | 없음 |
| `app.toggleFullScreen` | | 없음. **창 최대화 ↔ 복원 토글**(진짜 전체 화면 아님) |

### SFTP·붙여넣기
| 채널 | 인자 | 결과 |
|---|---|---|
| `sftp.home` | 세션 id | 서버 홈 경로 |
| `sftp.list` | id, 경로 | `{path, entries}` (data 구조: name, path, isDir, isLink, size, mtime, mode, perm) |
| `sftp.mkdir` | id, 경로 | 없음 |
| `sftp.rename` | id, 이전, 새 | 없음 |
| `sftp.remove` | id, 경로 배열 | 없음 |
| `sftp.chmod` | id, 경로, 모드(숫자) | 없음 |
| `sftp.upload` | id, 로컬 경로 배열, 원격 폴더 | 올린 원격 경로 배열 |
| `sftp.uploadPick` | id, 원격 폴더 | 로컬 파일 선택 대화상자("업로드할 파일 선택", 여러 개) 후 업로드 → 경로 배열(취소하면 빈 배열) |
| `sftp.download` | id, 원격 경로 배열, 로컬 폴더(선택) | 저장한 폴더 경로. 폴더를 안 주면 폴더 선택 대화상자("저장할 폴더 선택", 시작 폴더 다운로드, 새 폴더 만들기 허용) — 취소하면 false |
| `sftp.preview` | id, 경로 | `{kind: image\|text, data}` 또는 null |
| `sftp.edit` | id, 경로 | 임시로 받은 로컬 경로 |
| `sftp.uploadForPrompt` | id, 원본(`{kind: clipboardImage}` / `{kind: files, paths}` / `{kind: buffer, name, data}`) | 올린 원격 경로 배열 |
| `local.saveForPrompt` | 원본 | 로컬 경로 배열 |

### 가져오기 후보 객체 (import.scan / import.scanAws 결과, import.apply 입력)
후보 하나는 다음 JSON 객체다. **서버 정보는 평평하게 펼치지 않고 `host` 객체 안에 둔다.**
```
{ "source": "putty" | "ssh-config" | "aws",
  "name": 문자열,
  "host": { "alias", "protocol", "host", "port", "username"?, "authType", "keepaliveSec"?, "encoding"?, "termType"?,
            "forwards"?, "serial"?, "startupCommand"?, "notes"? },   // data/store.md 서버 필드의 부분 집합
  "keyFile"?: 문자열(키 파일 경로),
  "jumpVia"?: 문자열(점프 호스트 주소),
  "note"?: 문자열 }
```
- 화면은 이 구조를 그대로 읽는다: 종류 열 = `host.protocol`(없으면 `ssh`), 주소 열 = `host.username`·`host.host`·`host.port`(시리얼은 `host.serial.path`·`host.serial.baudRate`), 이름 열 = `name`, 출처 = `source`, 중복 판정 = `host.host`·`host.port`·`host.username` 이 기존 서버와 같은지.
- `import.apply` 에는 사용자가 고른 **후보 객체를 변경 없이 그대로** 보내고, 두 번째 인자로 폴더 이름을 보낸다.
- 일반 원칙: 화면(프런트)은 이 문서와 data/*.md 의 JSON 구조를 **그대로** 쓴다. 필드를 평평하게 바꾸거나 이름을 바꾸는 중간 변환을 임의로 만들지 않는다. 구조가 불분명하면 추측하지 말고 QUESTIONS.md 에 올린다.

### 가져오기·동기화·업데이트
| 채널 | 인자 | 결과 |
|---|---|---|
| `import.scan` | | 후보 배열(PuTTY 레지스트리 + ssh config) — **후보 JSON 구조는 위 "가져오기 후보 객체"** |
| `import.scanAws` | 프로필(선택) | 후보 배열 / 오류(메시지) |
| `import.apply` | 후보 배열, 폴더 이름 | `{hosts, keys, skipped}` |
| `sync.run` | 모드(`""` / `pull-vault` / `overwrite`) | 상태 `{state, message?, lastSync?, changed?}` |
| `sync.status` | | `{configured, gitAvailable, last}` — **`last` 는 항상 객체**이며 동기화한 적이 없으면 `{"state":""}`(나머지 필드 생략). null 이 아니다 |
| `update.check` | | `{status: newer\|latest\|error\|unconfigured, version?, url?}` (수동 확인 — 새 버전이면 안내 이벤트에 "알림 필요" 표시) |
| `update.status` | | 마지막으로 발견한 새 버전 `{status, version, url}` 또는 null |

## 3. 인증·확인 요청 (백엔드가 묻는 방식)
백엔드가 `ui.request` 이벤트로 질문을 보내고 프런트가 `Reply(requestId, 값JSON)` 로 답한다. **요청 객체의 id 필드 이름은 `requestId`** 이며 `kind` 와 아래 표의 필드를 함께 가진다. **응답이 `null` 이거나 10분 안에 없으면 "취소"** 로 본다. 종류와 응답 값:
| kind | 요청 필드 | 응답 |
|---|---|---|
| `hostkey` | sessionTitle, hostPort, keyType, fingerprint, changed, oldFingerprint? | 불리언(신뢰) |
| `password` | sessionTitle, prompt, username, allowSave, failed? | `{password, save}` |
| `keyboard` | sessionTitle, name, instructions, prompts[`{prompt, echo}`] | 문자열 배열 |
| `passphrase` | sessionTitle | 문자열 |
| `username` | sessionTitle | 문자열 |
화면은 ui/dialogs.md §4.

## 4. 이벤트 (백엔드 → 프런트)
| 이벤트 | 값 | 시점 |
|---|---|---|
| `session.data` | 세션 id, 텍스트 | 세션 출력. **약 12ms 단위로 묶어서** 보낸다 |
| `session.state` | 세션 정보. **`state` 값은 정확히 `connecting` · `connected` · `reconnecting` · `closed`(끊김) · `error`(실패)** — behavior/session.md §1 | 상태 변경(접속 중·연결됨·재접속·끊김·오류) |
| `ui.request` | 요청 객체 | §3 |
| `transfer.update` | 전송 정보 `{id, sessionId, name, direction, bytes, total, state, error?, remotePath, localPath?}` | 전송 진행(최소 150ms 간격) |
| `vault.changed` | 불리언(열림 여부) | 볼트 열림/잠김 |
| `store.changed` | | 백엔드가 저장 데이터를 바꿨을 때(가져오기·동기화·비밀번호 저장 등) — 프런트가 다시 읽는다 |
| `toast` | `{kind: ok\|error\|info, text}` | 백엔드가 띄우는 알림 |
| `sync.status` | 동기화 상태 — **`sync.run` 의 결과와 같은 형태** `{state, message?, lastSync?, changed?}` | 동기화 종료마다 |
| `update.available` | `{version, url, announce}` | 새 버전 발견(announce 는 처음 발견했거나 수동 확인일 때 true) |
