# 11. Go 코딩 표준 (헝가리안 명명 규칙)

> **APP_VERSION = `<x.y.z>` 기준 / 작성: YYYY-MM-DD**
>
> 본 문서는 프로젝트 Go 코드의 명명 규칙을 명문화한 것입니다. **신규 코드는 본 표준을 100% 준수**해야 하며, 기존 코드 수정 시는 §11 [충돌 및 마이그레이션 정책](#11-충돌-및-마이그레이션-정책)을 따릅니다.
>
> <!-- 헝가리안 규칙을 안 쓰는 프로젝트라면 이 문서를 프로젝트 컨벤션으로 교체하고 CLAUDE.md §2 링크만 유지 -->

---

## 1. 목적과 적용 범위

### 1.1 목적
- 코드베이스 전반의 **일관된 명명**으로 가독성·유지보수성 확보
- 신규 작성자(개발자 / Claude / 신규 입사자)가 즉시 따를 수 있는 단일 참조 지점
- 변수 이름만 보고 **타입·범위(scope)·export 여부**를 즉시 파악 가능하게 함
- 식별자 이름만 보고 **무엇을 담는지 / 무엇을 하는지** 즉시 이해 가능하게 함 (§3.4)

### 1.2 적용 대상
- **포함**: Go 전부. 펌웨어 C++ 도 같은 접두어. 셸/PowerShell 스크립트 전역·로컬도 같은 방식 (`g_szDeployDir`, `szArg`)
- **제외**: 생성 코드(sqlc·protobuf), 벤더링 코드, Swift (Swift 관례를 따름)

### 1.3 적용 강도
- **신규 파일/함수/구조체**: 본 표준 **강제 준수**
- **기존 코드 수정**: §11 정책 (주변 일관성 우선)

---

## 2. 빠른 요약 (Cheat Sheet)

| 종류 | 범위 / 의미 | 접두어 | 예시 |
|---|---|---|---|
| 전역 변수 | 같은 모듈 내부만 사용 | `g_` + 타입 | `g_szProcessName`, `g_iMaxLogQueueSize` |
| 전역 변수 | 타 모듈에서 참조 (export) | `G_` + 타입 | `G_iServerType`, `G_bEncryptedTextContent` |
| 구조체 멤버 | export 필드 | `M_` + 타입 | `M_ui64UserUid`, `M_szUserId` |
| 구조체 멤버 | unexport 필드 | 헝가리안만 | `bDbDetailLog`, `szProcessName` |
| 상수 | export | `G_c` + 타입 | `G_cszTimeLayoutMicro`, `G_ciServerType_Core` |
| 상수 | unexport | `g_c` + 타입 | `g_ciEncryptedContentSize` |
| 로컬 변수 | 함수 안 | 헝가리안만 | `szExePath`, `iLevel`, `bOk` |
| 함수 | export | UpperCamelCase, 헝가리안 X | `CheckIdValid`, `IsMainLoopStop` |
| 함수 | unexport | lowerCamelCase, 헝가리안 X | `parseCmd`, `getLogConfig` |
| 함수 | goroutine 진입 (unexport) | `go_` + 동사 | `go_runLoop` (선택) |
| 함수 | goroutine 진입 (export) | `GO_` + 동사 | `GO_CheckNFS` (선택) |
| 리시버 | 포인터 | `pstr<TypeName>` | `pstrLogger *CustomLogger` |
| 패키지 | 소문자 한 단어 (여러 단어가 꼭 필요할 때만 snake_case) | — | `config`, `store`, `api` |
| 파일 | snake_case 또는 hyphen-case | — | `msd_def.go`, `main-proc.go` |

**공통 원칙**:
- **이름은 읽는 사람이 즉시 의미를 알 수 있게** 짓는다. 약어 남발 / 의미 없는 한두 글자 금지.
- 길이가 길어도 의미가 명확하면 그쪽이 낫다 (`g_iMaxLogQueueSize` > `g_iMQS`).

**핵심 예외 (헝가리안 미적용)**:
- `err error` — 표준 Go 관례
- 표준 라이브러리 인터페이스 시그니처(예: `io.Reader.Read(p []byte)`)
- `context.Context` — `ctx` 또는 의미 있는 이름
- 리플렉션(XML/JSON 마샬링) 필드의 `XMLName` 등 라이브러리 요구 이름

---

## 3. 헝가리안 타입 접두어 매핑

| 접두어 | 타입 | 예시 |
|---|---|---|
| `sz` | string | `szUserId` |
| `i` / `i64` | int / int64 | `iLevel`, `i64Offset` |
| `ui` / `ui64` | uint / uint64 | `ui64UserUid` |
| `b` | bool | `bOk` |
| `f` / `d` | float32 / float64 | `fRate` |
| `p` | pointer | `pstrLogger` |
| `arr` / `sl` | array / slice | `slUserIds` |
| `m` | map | `mUserById` |
| `ch` | channel | `chDone` |

> 실사용 예: `dLat`, `slItems`, `mByKey`, `pstrService`, `arrBytes`.

### 3.4 이름의 의미 명확성 원칙
- 사람이 읽고 **즉시 의미를 알 수 있는** 이름을 쓴다. 타입 접두어는 보조일 뿐, 의미 전달이 우선.

---

## 11. 충돌 및 마이그레이션 정책

- **기존 파일 수정 시**: 그 파일/함수의 **주변 관행에 일관**되게 맞춘다 (표준과 달라도 부분적 혼용 금지).
- **새 파일/패키지**: 본 표준 강제.
- 대규모 리네이밍은 별도 작업으로 분리 (기능 변경과 섞지 않는다).
