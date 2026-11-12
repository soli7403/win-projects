# IPC 계약 대조표 (2026-11-11)

> 기준: [spec/ipc.md](../spec/ipc.md), [spec/data/store.md](../spec/data/store.md), [spec/data/vault.md](../spec/data/vault.md), [spec/data/sync-backup.md](../spec/data/sync-backup.md).
> 범위: 가져오기(`import.*`, 가져오기 후보 객체)는 C1 담당이라 제외했다. 백엔드(Go 핸들러의 `json` 태그·반환)와 프런트(`ipc/types.ts`, `state/*`, `screens/*`, 가짜 백엔드)를 채널·이벤트마다 대조했다.
> 고정 증거: `internal/app/contract_test.go` 가 응답을 `frontend/src/ipc/contract/*.json` 에 기록하고, `frontend/src/ipc/contract.test.ts` 가 검증한다.

## 1. 요청 채널

| 채널 | 백엔드 결과 | 프런트 기대 | 판정 | 조치 |
|---|---|---|---|---|
| `store.get` | 최상위 맵, 배열 키 4개, settings 기본값 병합 | `Store` | 배열 키가 `null` 로 저장된 뒤 `null` 이 나올 수 있었음 | **수정**: Go `Patch` 가 배열 키 null 을 `[]` 로 저장. 프런트 `refreshStore` 가 `normalizeStore` 로 배열을 채움 |
| `store.patch` | 결과 없음 | 결과 무시 | 일치 | - |
| `app.info` | `{version, platform, dataDir}` | `{version?, dataDir?}` / `{platform?}` | 일치 | - |
| `vault.status` | `{initialized, unlocked, osUnlockAvailable, osUnlockEnabled, osUnlockKind?}` | `VaultStatus` | 일치. 가짜 백엔드는 `osUnlockKind` 를 생략 | 허용(선택 필드) |
| `vault.setup` | `{recoveryCode}` | `{recoveryCode}` | 일치 | - |
| `vault.unlock` / `unlockOs` / `recover` / `changePassword` | 불리언 | 불리언 | 일치 | - |
| `vault.lock` / `setOsUnlock` | 없음 | 없음 | 일치 | - |
| `cred.list` | `CredMeta[]` (`id,name,kind,username?,updatedAt`) | `CredMeta[]` | 일치 | - |
| `cred.get` | 비밀 객체 또는 `null` | 사용처 없음 | 일치 | - |
| `cred.save` | `CredMeta` | `CredMeta` | 일치. **가짜 백엔드에 채널이 없었음** | **수정**: 실제와 같은 규칙(빠진 필드 유지, 빈 사용자 이름 지우기)으로 추가 |
| `cred.remove` | 없음 | 없음 | 가짜 백엔드에 없음 | **수정**: 추가 |
| `session.open` | `Info`: `id,hostId\|null,title,protocol,target,color?,env?,state,connectedAt?,message?,canSftp,keepSession?` | `SessionInfo` | 구조 일치. 가짜 백엔드의 제목·대상 문자열·`canSftp` 가 실제와 달랐음 | **수정**: 가짜 백엔드가 실제와 같은 제목(`사용자@주소`, 로컬은 `로컬 터미널`), 대상(`사용자@주소:포트`, 로컬은 `로컬 · 폴더`), `canSftp`(ssh 만) 를 낸다 |
| `session.close` / `reconnect` | 없음 | 없음 | 일치 | - |
| `session.state` 값 `state` | 백엔드 상수 `closed` (닫힘) | 프런트가 `disconnected` 로 잘못 가정 | **불일치** | **수정**: 프런트·가짜·테스트·CSS 를 `closed` 로 맞춤 (D48, 명세 session.md §1) |
| `local.shells` | `[{path,label}]` | `{path,label}[]` | 일치 | - |
| `local.tmuxAvailable` | 불리언 | 불리언 | 일치 | - |
| `serial.list` | `[{path,label}]` | `{path,label}[]` | 일치 | - |
| `clip.writeText` / `readText` | 없음 / 텍스트 | 동일 | 일치 | - |
| `clip.info` | `{type, files?, text?}` | `{type, files?, text?}` | 일치 | - |
| `clip.hasImage` | 불리언 | 불리언 | 일치 | - |
| `dialog.readKeyFile` | `{path,text}` 또는 `null` | 같음 | 일치. 가짜 백엔드에 없었음 | **수정**: 가짜에 추가(`null`) |
| `dialog.chooseDir` | 경로 또는 `null` | 같음 | 일치. 가짜에 없었음 | **수정**: 가짜에 추가(`null`) |
| `backup.export` / `import` | 경로 또는 `false` / 불리언 | 같음 | 일치 | - |
| `app.openExternal` / `openLogs` / `resetWindow` / `titlebar` | 없음 | 없음 | 일치. `titlebar`·`toggleFullScreen` 이 가짜에 없었음 | **수정**: 가짜에 추가 |
| `sftp.home` | 문자열 | 문자열 | 일치 | - |
| `sftp.list` | `{path, entries[{name,path,isDir,isLink,size,mtime,mode(숫자),perm(문자열)}]}` | `Listing` | 일치(Q16 의 답은 구현 결정 D26 과 같음). 빈 폴더도 `[]` | - |
| `sftp.mkdir` / `rename` / `remove` / `chmod` | 없음 | 없음 | 일치 | - |
| `sftp.upload` / `uploadPick` | 경로 배열 | `string[]` | 일치 | - |
| `sftp.download` | 저장 폴더 경로 또는 `false` | `string \| false` | 일치 | - |
| `sftp.preview` | `{kind,data}` 또는 `null` | 같음 | 일치 | - |
| `sftp.edit` | 임시 경로 | 사용처는 결과 무시 | 일치 | - |
| `sftp.uploadForPrompt` | 경로 배열 | `string[]` | 일치. **가짜 백엔드에 없었음** | **수정**: 가짜에 추가 |
| `local.saveForPrompt` | 경로 배열 | `string[]` | 일치. **가짜 백엔드에 없었음** | **수정**: 가짜에 추가 |
| `sync.run` | `{state,message?,lastSync,changed?}` | `SyncRunResult` | 일치 | - |
| `sync.status` | `{configured,gitAvailable,last}`; `last` 는 항상 객체, 동기화 전에는 `{"state":""}` (D50) | `last?:` 객체 | 백엔드가 null 을 보내던 것을 객체로 고침 | **수정**: 백엔드 `{"state":""}`. 명세 확정 (QUESTIONS Q26) |
| `update.check` | `{status, version?, url?}` | `UpdateCheckResult` | 일치 | - |
| `update.status` | `{status,version,url}` 또는 `null` | `{version,url}` 또는 null | 일치 | - |

## 2. 이벤트 (백엔드 → 프런트)

| 이벤트 | 백엔드 | 프런트 | 판정 | 조치 |
|---|---|---|---|---|
| `session.data` | 인자 2개 `(id, 텍스트)` | `(id, text)` | 일치 | - |
| `session.state` | `Info` 1개 | `SessionInfo` | `state` 값 불일치(위 `closed`) | 수정 D46 |
| `ui.request` | 키 `id` 로 발행 | 키 `requestId` 사용 (`requests.ts`, `RequestDialogs.tsx`, Reply 인자명) | **불일치**: 프런트는 `requestId` 로 답해야 하는데 키가 `id` 라 응답 큐가 깨짐 | **수정**: 백엔드 키를 `requestId` 로 (D45). Go 테스트(`router_test.go`, `integration_test.go`) 도 같이 고침. 명세 확정 (ipc.md §3, QUESTIONS Q24) |
| `ui.request` 필드 | hostkey `hostPort,keyType,fingerprint,changed,oldFingerprint?`; password `prompt,username,allowSave,failed?`; keyboard `name,instructions,prompts[{prompt,echo}]` | `UiRequest` 유니온과 같음 | 일치 | - |
| `transfer.update` | `Transfer` 1개 `{id,sessionId,name,direction,bytes,total,state,error?,remotePath,localPath?}` | `TransferInfo` | 일치. 상태 값은 백엔드 `queued/running/done/error` (D49), 프런트는 여러 값을 허용 | 명세 미정(Q17) |
| `vault.changed` | 불리언 1개 | 불리언 | 일치 | - |
| `store.changed` | 인자 없음 | 인자 없음 | 일치 | - |
| `toast` | `{kind,text}` | `ToastPayload` | 일치 | - |
| `sync.status` | `gitsync.Result` (`state,message?,lastSync,changed?`) | `SyncRunResult` | 일치 | - |
| `update.available` | `{version,url,announce}` | `UpdateInfo` | 일치 | - |

## 3. 수정 요약

- 백엔드: `ui.request` 키 `requestId`(`internal/ipc/ask.go`), 세션 상태 `disconnected`(`internal/session/messages.go`), 배열 키 null 방어(`internal/store/store.go`), 테스트 3곳 갱신.
- 프런트: `normalizeStore`(`state/app.ts`), `SyncStatus.last` 타입(`ipc/types.ts`), 가짜 백엔드 채널 추가·정정(`ipc/fakeBackend.ts`, `ipc/fakeSftp.ts`).
- 검증: Go 계약 테스트(`internal/app/contract_test.go`, 고정 파일 18개), 프런트 계약 테스트(`src/ipc/contract.test.ts`).

## 4. 남은 것 (못 고친 것)

- 전송 상태 값(`queued`/`waiting`/`running`/`done`/`error`/취소)과 세션 상태 토큰 이름은 명세에 없다(QUESTIONS Q17, Q25). 현재 구현에 맞춰 두었다.
- `sync.status.last` 가 `null` 인 경우의 명세 의미(QUESTIONS Q26).
- 가져오기(`import.*`)와 가져오기 후보 객체는 C1 담당이라 손대지 않았다.
