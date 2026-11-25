# Xshell 가져오기에 포트 포워딩 + 세션 파일(`.xsh`) 하나 고르기 — 형식·설계·코드·검증

**작성**: 2026-11-25 · **상태**: 코드·자동 테스트·빌드 완료(D129). 사용자의 실제 세션 파일로 포워딩 43개가 모두 읽히는 것을 확인했다. 앱 화면에서의 수동 검증(§H)도 2026-11-25 사용자 확인 완료 · **대상**: cli-manager-cr(DariTerm) · **Known issue**: 없음
**관련**: 바로 앞 문서 [2026-11-25-5-xshell-import.md](./2026-11-25-5-xshell-import.md)(D128, Xshell 가져오기 — 그때는 포워딩을 옮기지 않았다). 명세 [spec/import.md](../../spec/import.md) §1.6
**적용 순서**: 2026-11-25 의 6번째. **5번을 적용한 뒤에** 적용한다(5번이 만든 `xshell.go`·`xshell_test.go`·문구를 고친다).

> 이 문서 하나로 **무엇을 바꿨는지 → 포워딩이 파일에 어떻게 적혀 있는지 → 왜 그렇게 정했는지 → 코드 → 어떻게 검증하는지**를 알 수 있게 쓴다.

---

## 0. 한 줄 요약

두 가지를 더했다.

1. Xshell 세션의 **포트 포워딩(터널링) 규칙을 옮긴다.** 전에는 "포트 포워딩 n개는 옮기지 않음" 안내만 냈다.
2. 파일 선택 창에서 **세션 파일(`.xsh`) 하나**도 고를 수 있다. 전에는 내보내기 파일(`.xts`)만 됐다.

| | 5번 문서까지 | 이 문서 뒤 |
|---|---|---|
| 포트 포워딩 | 옮기지 않음(개수만 알림) | 옮긴다. 규칙에 붙인 설명은 서버 메모에 적는다 |
| "옮기지 않음" 안내 | 포워딩이 있으면 늘 | 옮기지 못한 규칙이 있을 때만, 그 수 |
| 고를 수 있는 파일 | `.xts` | `.xts`, `.xsh` |

사용자가 준 세션 파일로 확인한 결과: 파일에 적힌 규칙 43개 → 포워딩 43개(모두 로컬), 옮기지 못한 것 0개, 설명 43개가 메모에 들어감.

## 1. 사용자 요청

2026-11-25: "포트포워딩은 임포트 안되는 것인가?" → (형식을 볼 표본이 없어 뺐다고 답함) → 포워딩이 든 세션 파일을 주며 "파일을 참조하라."

5번 문서 때 받은 내보내기 파일은 세션 164개가 모두 `FwdReqCount=0` 이라 포워딩 항목의 모양을 볼 수 없었다. 이번에 받은 세션 파일 하나에 규칙 43개가 있어 형식이 확인됐다.

## 2. 포워딩이 파일에 적힌 모양 (받은 파일을 보고 확인한 것)

받은 파일의 실제 값(주소·설명)은 이 문서와 테스트에 적지 않는다. 아래의 예는 지어낸 값이다.

`[CONNECTION:SSH]` 구역에 규칙 수 `FwdReqCount` 가 있고, 규칙 i(0 부터)는 `FwdReq_{i}_{항목}` 이라는 키 7개로 적힌다:

```ini
[CONNECTION:SSH]
FwdReqCount=2
FwdReq_0_Incoming=0
FwdReq_0_Source=localhost
FwdReq_0_Port=50093
FwdReq_0_Host=10.1.0.93
FwdReq_0_HostPort=22
FwdReq_0_LocalOnly=0
FwdReq_0_Description=DC1-APP01
FwdReq_1_Incoming=0
…
```

| 항목 | 뜻(Xshell 화면의 이름) | 받은 파일(43개)의 값 |
|---|---|---|
| `Incoming` | 방향. `0` = 나가는 것(Local/Outgoing) | 모두 `0` |
| `Source` | 원본 호스트(듣는 주소) | 모두 `localhost` |
| `Port` | 수신 대기 포트(듣는 포트) | 43가지, 겹치는 것 없음 |
| `Host` | 대상 호스트 | 모두 있음 |
| `HostPort` | 대상 포트 | 모두 있음 |
| `LocalOnly` | "로컬 연결만 허용" | `0` 이 20개, `1` 이 23개 |
| `Description` | 설명 | 43개 모두 있음 |

**주의할 점**: 키들이 규칙 순서대로 모여 있지 않고 **구역 안에 뒤섞여** 있다(예: `FwdReq_10_Source` 다음 줄이 `FwdReq_13_Host`). 그래서 줄 순서에 기대지 않고 번호로 찾아 읽는다.

받은 파일에 없어 확인하지 못한 것(표본 미확인): `Incoming=1`(들어오는 것 = 원격 포워딩), 동적(SOCKS) 규칙이 어떻게 적히는지.

## 3. 설계 결정

| 항목 | 정한 것 | 이유 |
|---|---|---|
| 종류 | `Incoming` 0(또는 없음) → 로컬 `L`, 1 → 원격 `R`, 그 밖 → **옮기지 않음** | 0 은 확인했다. 1 은 이름("들어오는 것") 그대로 원격으로 본다(표본 미확인). 모르는 값은 추측하지 않는다 — 동적 규칙이 어떤 값인지 모른다 |
| 듣는 포트·대상 | `Port` → 듣는 포트, `Host`·`HostPort` → 대상 | — |
| 못 쓰는 규칙 | 포트가 1~65535 밖이거나 대상 주소가 비면 옮기지 않는다 | 등록해도 동작하지 않는다 |
| 듣는 주소 | `Source` 가 비었거나 `localhost` 면 `127.0.0.1`, 다른 값은 그대로 | 이 앱의 포워딩은 주소를 숫자로 둔다(다른 가져오기의 기본값과 같다) |
| **`LocalOnly`** | **옮기지 않는다.** 0 이어도 듣는 주소를 넓히지 않는다 | 0 은 "같은 네트워크의 다른 PC 도 이 포트에 접속할 수 있다"는 뜻이다. 받은 파일은 43개 중 20개가 0 이다. 그대로 옮기면 **가져오기만으로 이 PC 의 포트 20개가 네트워크에 열린다.** 모두 `127.0.0.1` 로 두고, 정말 필요한 규칙은 서버 편집에서 듣는 주소를 바꾸게 한다 |
| **설명** | 서버 **메모** 끝에 규칙마다 한 줄: `L 50093 → 10.1.0.93:22  DC1-APP01`. 머리 줄 "포트 포워딩 이름 (Xshell):" | 이 앱의 포워딩 항목에는 이름을 둘 자리가 없다(종류·듣는 주소·포트·대상뿐). 받은 파일은 43개 모두에 설명이 있어, 버리면 어느 포트가 어디로 가는지 알 수 없게 된다 |
| 이름 필드를 새로 만들지 않은 까닭 | 저장 형식(`forwards`)과 서버 편집 화면을 바꾸지 않는다 | 가져오기 하나 때문에 저장 형식·화면·동기화를 건드리게 된다. 필요하면 따로 다룬다(§I) |
| 안내 | 옮기지 못한 규칙이 있을 때만 `포트 포워딩 {n}개는 옮기지 않음` | 문구는 그대로 쓰고 뜻만 "전부"에서 "이 수만큼"으로 바뀐다 |
| 한도 | 세션 하나에 규칙 1000개까지 | 잘못된 파일의 큰 수 대비 |
| 텔넷 세션 | 포워딩을 읽지 않는다 | SSH 의 기능이다 |
| **세션 파일 하나** | 고른 파일의 확장자가 `.xsh` 면 그 세션 하나만 읽는다(폴더 없음). 파일 선택 창의 종류를 `*.xts;*.xsh` 로 | 이번에 받은 파일이 `.xts` 가 아니라 `.xsh` 였다. 세션 하나만 옮기려고 전체를 내보낼 필요가 없다 |

메모가 만들어지는 모양(지어낸 값):

```
게이트웨이                                  ← Xshell 의 세션 설명(있을 때)
                                            ← 빈 줄
포트 포워딩 이름 (Xshell):
L 50093 → 10.1.0.93:22  DC1-APP01
L 50103 → 10.1.0.103:8443  DC1-WEB-VIP
```

### 3.1 영향 범위

- **저장 형식·IPC**: 바뀌지 않았다. 후보의 `host.forwards` 는 PuTTY·SecureCRT 가 이미 쓰는 필드다.
- **5번 문서의 내보내기 파일**: 그 파일의 세션은 포워딩이 없어 결과가 전과 같다(164개 → 164개, 다시 확인).
- **화면**: 버튼은 그대로. 버튼 옆 설명과 파일 선택 창의 종류만 바뀐다.

## A. 바뀐 파일 한눈에

| 파일 | 종류 | 상태 |
|---|---|---|
| `internal/importer/xshell.go` | 코드(Go) | 수정(`ParseXshForwards`, 세션 파일 하나) |
| `internal/app/channels_import.go` | 코드(Go) | 수정(파일 선택 창의 종류) |
| `internal/importer/xshell_test.go` | 테스트 | 수정(3개 추가, 하나는 선택 실행) |
| `locales/*.json` (10개) | 문구 | 키 1개 추가, 1개 고침 |
| `spec/import.md`(§1.6), `spec/ipc.md`, `spec/ui/dialogs.md` | 명세 | 수정 |
| `docs/scenarios.md`, `docs/scenarios.en.md` | 문서 | 수정(10-3a) |
| `CHANGELOG.md`, `docs/decisions.md`(D129), `CLAUDE.md` | 문서 | 수정 |

아래의 `diff` 는 **5번 문서를 적용한 상태**를 기준으로 뽑았다.

## B. 코드

### `internal/importer/xshell.go` — 수정

```diff
--- a/internal/importer/xshell.go
+++ b/internal/importer/xshell.go
@@ -41,6 +41,9 @@
 	M_values   xshValues
 }
 
+// g_ciXshMaxFwd 는 세션 하나에서 읽는 포워딩 수 한도다.
+const g_ciXshMaxFwd = 1000
+
 // xshValues 는 세션 파일의 값이다. 키는 "구역 이름(대문자)\x00키 이름(소문자)".
 type xshValues map[string]string
 
@@ -196,8 +199,18 @@
 			host.M_szAuthType = "key"
 		}
 		host.M_szStartupCommand = values.str(g_cszXshSecSsh, "RemoteCommand")
-		if iForwards := values.num(g_cszXshSecSsh, "FwdReqCount"); iForwards > 0 {
-			slNotes = append(slNotes, i18n.T("be.xsh.forwards", "count", iForwards))
+		slForwards, slNames, iSkipped := ParseXshForwards(values)
+		host.M_slForwards = slForwards
+		if len(slNames) > 0 {
+			// 포워딩 항목에는 이름을 둘 자리가 없어, Xshell 에서 붙인 설명은 서버 메모에 적어 둔다.
+			szNames := i18n.T("be.xsh.forwardNames") + "\n" + strings.Join(slNames, "\n")
+			if host.M_szNotes != "" {
+				szNames = host.M_szNotes + "\n\n" + szNames
+			}
+			host.M_szNotes = szNames
+		}
+		if iSkipped > 0 {
+			slNotes = append(slNotes, i18n.T("be.xsh.forwards", "count", iSkipped))
 		}
 	}
 	if szProxy := values.str(g_cszXshSecProxy, "Proxy"); szProxy != "" {
@@ -212,6 +225,48 @@
 	}, true
 }
 
+// ParseXshForwards 는 [CONNECTION:SSH] 의 포워딩 항목을 읽는다 (import.md §1.6 포트 포워딩).
+// 항목 N(0 ~ FwdReqCount-1)은 키 `FwdReq_{N}_{이름}` 들이다: Incoming(0 나가는 것 = 로컬 L, 1 들어오는 것 = 원격 R),
+// Source·Port(듣는 주소·포트), Host·HostPort(대상 주소·포트), Description(설명), LocalOnly(쓰지 않는다).
+// 돌려주는 것: 포워딩 목록, 설명이 있는 항목의 "{L|R} {듣는 포트} → {대상}:{포트}  {설명}" 줄들, 옮기지 못한 항목 수.
+func ParseXshForwards(values xshValues) ([]Forward, []string, int) {
+	iCount := values.num(g_cszXshSecSsh, "FwdReqCount")
+	if iCount > g_ciXshMaxFwd {
+		iCount = g_ciXshMaxFwd
+	}
+	slOut := []Forward{}
+	slNames := []string{}
+	iSkipped := 0
+	for iIndex := 0; iIndex < iCount; iIndex++ {
+		szPrefix := "FwdReq_" + strconv.Itoa(iIndex) + "_"
+		szType := ""
+		switch values.str(g_cszXshSecSsh, szPrefix+"Incoming") {
+		case "0", "":
+			szType = "L"
+		case "1":
+			szType = "R"
+		}
+		iBindPort := values.num(g_cszXshSecSsh, szPrefix+"Port")
+		iDestPort := values.num(g_cszXshSecSsh, szPrefix+"HostPort")
+		szDestHost := values.str(g_cszXshSecSsh, szPrefix+"Host")
+		if szType == "" || iBindPort < 1 || iBindPort > 65535 || iDestPort < 1 || iDestPort > 65535 || szDestHost == "" {
+			iSkipped++
+			continue
+		}
+		// 듣는 주소: 비었거나 localhost 면 127.0.0.1. LocalOnly=0("다른 PC 의 접속도 받기")은 옮기지 않는다 —
+		// 가져오기만으로 이 PC 의 포트가 네트워크에 열리지 않게 한다.
+		szBindHost := values.str(g_cszXshSecSsh, szPrefix+"Source")
+		if szBindHost == "" || strings.EqualFold(szBindHost, "localhost") {
+			szBindHost = "127.0.0.1"
+		}
+		slOut = append(slOut, Forward{M_szType: szType, M_szBindHost: szBindHost, M_iBindPort: iBindPort, M_szDestHost: szDestHost, M_iDestPort: iDestPort})
+		if szDescription := values.str(g_cszXshSecSsh, szPrefix+"Description"); szDescription != "" {
+			slNames = append(slNames, szType+" "+strconv.Itoa(iBindPort)+" → "+szDestHost+":"+strconv.Itoa(iDestPort)+"  "+szDescription)
+		}
+	}
+	return slOut, slNames, iSkipped
+}
+
 // xshSessionOf 는 세션 폴더 기준 상대 경로(/ 구분)와 내용으로 세션을 만든다. 세션 파일이 아니거나 경로가 이상하면 bOk=false.
 func xshSessionOf(szRel string, bRaw []byte) (XshSession, bool) {
 	slParts := strings.Split(szRel, "/")
@@ -337,7 +392,7 @@
 }
 
 // ScanXshell 은 Xshell 세션을 후보로 바꾼다 (import.md §1.6).
-// szPath 가 비면 기본 세션 폴더(없으면 빈 목록), .xts 파일이면 내보내기 파일, 폴더면 그 폴더를 읽는다.
+// szPath 가 비면 기본 세션 폴더(없으면 빈 목록), 폴더면 그 폴더, .xsh 파일이면 그 세션 하나, 그 밖의 파일이면 내보내기 파일(.xts)로 읽는다.
 // 경로가 없거나 내보내기 파일을 열지 못하면 오류다.
 func ScanXshell(szPath string) ([]Candidate, error) {
 	var slSessions []XshSession
@@ -354,6 +409,18 @@
 		}
 		if pInfo.IsDir() {
 			slSessions = ReadXshDir(ResolveXshDir(szPath))
+		} else if strings.EqualFold(filepath.Ext(szPath), g_cszXshExt) {
+			// 세션 파일 하나: 폴더 없이 그 세션만 (import.md §1.6).
+			if pInfo.Size() > g_ciXshMaxBytes {
+				return []Candidate{}, nil
+			}
+			bRaw, errRead := os.ReadFile(szPath)
+			if errRead != nil {
+				return nil, i18n.Errorf("be.xsh.badFile", errRead)
+			}
+			if session, bOk := xshSessionOf(filepath.Base(szPath), bRaw); bOk {
+				slSessions = []XshSession{session}
+			}
 		} else {
 			slRead, errRead := ReadXtsFile(szPath)
 			if errRead != nil {
```

### `internal/app/channels_import.go` — 수정

```diff
--- a/internal/app/channels_import.go
+++ b/internal/app/channels_import.go
@@ import.scanXshell @@
-		// 경로(내보내기 파일 .xts 또는 세션 폴더)를 안 주면 내보내기 파일을 고르게 한다. 취소하면 null (spec/ipc.md).
+		// 경로(내보내기 파일 .xts, 세션 파일 .xsh, 세션 폴더)를 안 주면 파일을 고르게 한다. 취소하면 null (spec/ipc.md).
@@
-			szFilter := i18n.T("be.dlg.xshellFilter") + " (*.xts)|*.xts|*.*|*.*"
+			szFilter := i18n.T("be.dlg.xshellFilter") + " (*.xts, *.xsh)|*.xts;*.xsh|*.*|*.*"
```


## C. 테스트 코드

### `internal/importer/xshell_test.go` — 수정

```diff
--- a/internal/importer/xshell_test.go
+++ b/internal/importer/xshell_test.go
@@ -330,3 +330,112 @@
 		t.Fatal("폴더·세션 이름이 깨짐")
 	}
 }
+
+// 포트 포워딩(FwdReq_{번호}_{항목}): 나가는 것은 L, 들어오는 것은 R, 듣는 주소 localhost 는 127.0.0.1,
+// 설명은 서버 메모에 적고, 옮기지 못한 항목은 수를 알린다 (import.md §1.6, D129).
+func TestParseXshForwards(t *testing.T) {
+	useKoreanImport(t)
+	session, _ := xshSessionOf("gw.xsh", xshBytes(
+		"[CONNECTION]", "Host=gw.example", "Port=22", "Protocol=SSH", "Description=게이트웨이",
+		"[CONNECTION:SSH]",
+		// 항목의 줄은 실제 파일처럼 뒤섞여 있다.
+		"FwdReq_1_HostPort=8443", "FwdReq_0_Source=localhost", "FwdReq_2_Incoming=1", "FwdReqCount=6",
+		"FwdReq_0_Incoming=0", "FwdReq_0_Port=50093", "FwdReq_0_Host=10.1.0.93", "FwdReq_0_HostPort=22", "FwdReq_0_LocalOnly=0", "FwdReq_0_Description=DC1-APP01",
+		"FwdReq_1_Incoming=0", "FwdReq_1_Port=50103", "FwdReq_1_Host=10.1.0.103", "FwdReq_1_Source=localhost", "FwdReq_1_LocalOnly=1",
+		"FwdReq_2_Source=0.0.0.0", "FwdReq_2_Port=9000", "FwdReq_2_Host=127.0.0.1", "FwdReq_2_HostPort=3000", "FwdReq_2_Description=웹 미리보기",
+		"FwdReq_3_Incoming=2", "FwdReq_3_Source=localhost", "FwdReq_3_Port=1080", // 모르는 종류(동적 등): 옮기지 않는다
+		"FwdReq_4_Incoming=0", "FwdReq_4_Port=0", "FwdReq_4_Host=10.1.0.1", "FwdReq_4_HostPort=22", // 듣는 포트 없음
+		"FwdReq_5_Incoming=0", "FwdReq_5_Port=50001", "FwdReq_5_Host=", "FwdReq_5_HostPort=22", // 대상 없음
+		"FwdReq_6_Incoming=0", "FwdReq_6_Port=50002", "FwdReq_6_Host=10.1.0.2", "FwdReq_6_HostPort=22", // 개수 밖: 읽지 않는다
+	))
+	cand, bOk := ParseXshSession(session)
+	if !bOk {
+		t.Fatal("후보가 돼야 함")
+	}
+	slWant := []Forward{
+		{M_szType: "L", M_szBindHost: "127.0.0.1", M_iBindPort: 50093, M_szDestHost: "10.1.0.93", M_iDestPort: 22},
+		{M_szType: "L", M_szBindHost: "127.0.0.1", M_iBindPort: 50103, M_szDestHost: "10.1.0.103", M_iDestPort: 8443},
+		{M_szType: "R", M_szBindHost: "0.0.0.0", M_iBindPort: 9000, M_szDestHost: "127.0.0.1", M_iDestPort: 3000},
+	}
+	if len(cand.M_host.M_slForwards) != len(slWant) {
+		t.Fatalf("포워딩 수: %+v", cand.M_host.M_slForwards)
+	}
+	for iIndex, fwdWant := range slWant {
+		if cand.M_host.M_slForwards[iIndex] != fwdWant {
+			t.Fatalf("%d번째: %+v (기대 %+v)", iIndex, cand.M_host.M_slForwards[iIndex], fwdWant)
+		}
+	}
+	const szWantNotes = "게이트웨이\n\n포트 포워딩 이름 (Xshell):\nL 50093 → 10.1.0.93:22  DC1-APP01\nR 9000 → 127.0.0.1:3000  웹 미리보기"
+	if cand.M_host.M_szNotes != szWantNotes {
+		t.Fatalf("메모:\n%q\n기대:\n%q", cand.M_host.M_szNotes, szWantNotes)
+	}
+	if cand.M_szNote != "포트 포워딩 3개는 옮기지 않음" {
+		t.Fatalf("안내: %q", cand.M_szNote)
+	}
+
+	// 설명도 메모도 없으면 메모는 비고, 모두 옮겼으면 안내도 없다. 텔넷 세션의 포워딩은 읽지 않는다.
+	session, _ = xshSessionOf("a.xsh", xshBytes("[CONNECTION]", "Host=h", "Protocol=SSH", "[CONNECTION:SSH]", "FwdReqCount=1", "FwdReq_0_Port=8080", "FwdReq_0_Host=web", "FwdReq_0_HostPort=80"))
+	cand, _ = ParseXshSession(session)
+	if len(cand.M_host.M_slForwards) != 1 || cand.M_host.M_szNotes != "" || cand.M_szNote != "" {
+		t.Fatalf("이름 없는 포워딩: %+v / %q / %q", cand.M_host.M_slForwards, cand.M_host.M_szNotes, cand.M_szNote)
+	}
+	session, _ = xshSessionOf("t.xsh", xshBytes("[CONNECTION]", "Host=h", "Protocol=TELNET", "[CONNECTION:SSH]", "FwdReqCount=1", "FwdReq_0_Port=8080", "FwdReq_0_Host=web", "FwdReq_0_HostPort=80"))
+	cand, _ = ParseXshSession(session)
+	if len(cand.M_host.M_slForwards) != 0 || cand.M_szNote != "" {
+		t.Fatalf("텔넷: %+v", cand)
+	}
+}
+
+// 실제 세션 파일로 맞대어 보기(선택): 환경 변수 DARITERM_XSH 에 .xsh 경로를 주면 포워딩을 읽어 수만 적는다.
+// 파일의 내용(호스트·이름)은 출력하지 않는다. 변수가 없으면 건너뛴다.
+func TestXshellRealSessionForwards(t *testing.T) {
+	szPath := os.Getenv("DARITERM_XSH")
+	if szPath == "" {
+		t.Skip("DARITERM_XSH 가 없음")
+	}
+	bRaw, errRead := os.ReadFile(szPath)
+	if errRead != nil {
+		t.Fatal(errRead)
+	}
+	session, bOk := xshSessionOf(filepath.Base(szPath), bRaw)
+	if !bOk {
+		t.Fatal("세션 파일이 아님")
+	}
+	cand, bOk := ParseXshSession(session)
+	if !bOk {
+		t.Fatal("후보가 돼야 함")
+	}
+	slForwards, slNames, iSkipped := ParseXshForwards(session.M_values)
+	mapType, mapBind, mapPorts := map[string]int{}, map[string]int{}, map[int]int{}
+	for _, fwd := range slForwards {
+		mapType[fwd.M_szType]++
+		mapBind[fwd.M_szBindHost]++
+		mapPorts[fwd.M_iBindPort]++
+		if fwd.M_szDestHost == "" || fwd.M_iDestPort < 1 || fwd.M_iBindPort < 1 {
+			t.Fatal("빈 값이 든 포워딩이 있음")
+		}
+	}
+	t.Logf("적힌 개수 %d → 포워딩 %d개 (종류 %v, 듣는 주소 %v, 듣는 포트 %d가지), 이름 있음 %d, 옮기지 못함 %d",
+		session.M_values.num(g_cszXshSecSsh, "FwdReqCount"), len(slForwards), mapType, mapBind, len(mapPorts), len(slNames), iSkipped)
+	t.Logf("후보: 포워딩 %d개, 메모 %d줄, 안내 있음 %v", len(cand.M_host.M_slForwards), strings.Count(cand.M_host.M_szNotes, "\n")+1, cand.M_szNote != "")
+}
+
+// 세션 파일(.xsh) 하나를 고르면 그 세션만, 폴더 없이 읽는다 (import.md §1.6, D129).
+func TestScanXshellSingleSessionFile(t *testing.T) {
+	szDir := t.TempDir()
+	szPath := filepath.Join(szDir, "gw [LAB].XSH")
+	if errWrite := os.WriteFile(szPath, xshSample("10.0.0.254", "[CONNECTION:SSH]", "FwdReqCount=1", "FwdReq_0_Port=8080", "FwdReq_0_Host=web", "FwdReq_0_HostPort=80"), 0o600); errWrite != nil {
+		t.Fatal(errWrite)
+	}
+	// 같은 폴더의 다른 세션은 읽지 않는다.
+	if errWrite := os.WriteFile(filepath.Join(szDir, "other.xsh"), xshSample("10.0.0.1"), 0o600); errWrite != nil {
+		t.Fatal(errWrite)
+	}
+	slCands, errScan := ScanXshell(szPath)
+	if errScan != nil {
+		t.Fatal(errScan)
+	}
+	if len(slCands) != 1 || slCands[0].M_szName != "gw [LAB]" || slCands[0].M_host.M_szHost != "10.0.0.254" || len(slCands[0].M_slFolderPath) != 0 {
+		t.Fatalf("후보: %+v", slCands)
+	}
+}
```


## E. 문구

`locales/<언어>.json` 10개 파일(아래 줄의 순서: ko, en, ja, zh, tw, de, fr, es, pt, ru). 한국어·영어 외에는 이 작업을 한 세션이 번역한 것이라 검수가 필요할 수 있다.

`be.xsh.forwardNames` 는 `be.xsh.forwards` 바로 다음 줄에 넣는다. `be.xsh.forwards`("포트 포워딩 {count}개는 옮기지 않음")는 글을 고치지 않았다 — 뜻만 "옮기지 못한 수"가 됐다.

**`be.xsh.forwardNames`** (추가)

```json
  "be.xsh.forwardNames": "포트 포워딩 이름 (Xshell):",
  "be.xsh.forwardNames": "Port forwarding names (Xshell):",
  "be.xsh.forwardNames": "ポート転送の名前 (Xshell):",
  "be.xsh.forwardNames": "端口转发名称 (Xshell)：",
  "be.xsh.forwardNames": "連接埠轉送名稱 (Xshell)：",
  "be.xsh.forwardNames": "Namen der Portweiterleitungen (Xshell):",
  "be.xsh.forwardNames": "Noms des redirections de ports (Xshell) :",
  "be.xsh.forwardNames": "Nombres de los reenvíos de puertos (Xshell):",
  "be.xsh.forwardNames": "Nomes dos encaminhamentos de porta (Xshell):",
  "be.xsh.forwardNames": "Имена пробросов портов (Xshell):",
```

**`import.xshNote`** (고침 — 아래가 새 글)

```json
  "import.xshNote": "Xshell 의 [파일 > 내보내기]로 만든 .xts 파일이나 세션 파일(.xsh) 하나에서 세션을 읽습니다. 폴더 구조와 포트 포워딩은 그대로 옮겨집니다. 비밀번호·사용자 키·프록시 설정은 옮기지 않습니다.",
  "import.xshNote": "Reads sessions from an .xts file made with [File > Export] in Xshell, or from a single session file (.xsh). The folder structure and port forwarding are carried over. Passwords, user keys and proxy settings are not imported.",
  "import.xshNote": "Xshell の [ファイル > エクスポート] で作成した .xts ファイル、またはセッションファイル (.xsh) 1 つからセッションを読み込みます。フォルダー構成とポート転送はそのまま移します。パスワード・ユーザーキー・プロキシ設定は移しません。",
  "import.xshNote": "从 Xshell 的 [文件 > 导出] 生成的 .xts 文件或单个会话文件 (.xsh) 读取会话。文件夹结构和端口转发会原样迁移。密码、用户密钥和代理设置不迁移。",
  "import.xshNote": "從 Xshell 的 [檔案 > 匯出] 產生的 .xts 檔或單一工作階段檔 (.xsh) 讀取工作階段。資料夾結構與連接埠轉送會原樣移轉。密碼、使用者金鑰與 Proxy 設定不移轉。",
  "import.xshNote": "Liest Sitzungen aus einer .xts-Datei, die in Xshell mit [Datei > Exportieren] erstellt wurde, oder aus einer einzelnen Sitzungsdatei (.xsh). Ordnerstruktur und Portweiterleitungen werden übernommen. Passwörter, Benutzerschlüssel und Proxy-Einstellungen werden nicht übernommen.",
  "import.xshNote": "Lit les sessions d'un fichier .xts créé avec [Fichier > Exporter] dans Xshell, ou d'un seul fichier de session (.xsh). L'arborescence et les redirections de ports sont reprises. Les mots de passe, clés utilisateur et paramètres de proxy ne sont pas repris.",
  "import.xshNote": "Lee las sesiones de un archivo .xts creado con [Archivo > Exportar] en Xshell, o de un único archivo de sesión (.xsh). Se trasladan la estructura de carpetas y los reenvíos de puertos. No se trasladan contraseñas, claves de usuario ni ajustes de proxy.",
  "import.xshNote": "Lê as sessões de um arquivo .xts criado com [Arquivo > Exportar] no Xshell, ou de um único arquivo de sessão (.xsh). A estrutura de pastas e os encaminhamentos de porta são transferidos. Senhas, chaves de usuário e configurações de proxy não são transferidos.",
  "import.xshNote": "Читает сеансы из файла .xts, созданного в Xshell через [Файл > Экспорт], или из одного файла сеанса (.xsh). Структура папок и проброс портов переносятся. Пароли, пользовательские ключи и настройки прокси не переносятся.",
```

전의 글(한국어): "Xshell 의 [파일 > 내보내기]로 만든 .xts 파일에서 세션을 읽습니다. 폴더 구조는 넣을 폴더 아래에 그대로 만들어집니다. 비밀번호·사용자 키·포트 포워딩·프록시 설정은 옮기지 않습니다."


## F. 명세·문서

### `spec/import.md` — 수정

```diff
--- a/spec/import.md
+++ b/spec/import.md
@@ -47,10 +47,11 @@
 
 > 작성 방식: **원본 앱에 없는 신규 기능**(사용자 요청, 2026-11-25, D128). 형식은 사용자가 준 Xshell 내보내기 파일(`.xts`, 세션 파일 164개 — 판 8.1 이 110개·7.1 이 54개, 모두 SSH·비밀번호 인증·UTF-8) 관찰로 정했다. "(표본 미확인)" 표시는 그 파일에 해당하는 값이 없어 실제로 확인하지 못한 최선 추정이다.
 
-- **읽는 곳**(셋 중 하나):
+- **읽는 곳**(넷 중 하나):
   - **내보내기 파일**(`.xts`, Xshell 의 [파일 > 내보내기]로 만든 것): zip 이다. 그 안에서 **최상위 폴더 이름이 `Xshell`**(대소문자 무시)인 항목 아래만 읽는다(`Xftp`·`xsl`·`com` 등 다른 폴더는 보지 않는다). 디스크에 풀지 않고 읽는다. zip 으로 열 수 없으면 오류 "Xshell 내보내기 파일을 열지 못했습니다: {원인}".
     - 항목 이름은 UTF-8 표시가 없는 경우가 있다(내보낸 PC 의 코드 페이지 바이트): **유효한 UTF-8 이 아니면 EUC-KR(CP949)로 해석**한다. 구분자 `\` 는 `/` 로 본다.
     - 한도: 세션 파일 하나 1MB(넘으면 그 파일은 건너뜀), 세션 20000개, 읽은 내용 합계 64MB(넘으면 거기서 멈춤).
+  - **세션 파일 하나**(`.xsh`, 2026-11-25 D129): 그 세션 하나만 읽는다. `folderPath` 는 빈 배열. 1MB 를 넘으면 빈 목록.
   - **세션 폴더**: 고른 폴더 안에 `Sessions` 폴더가 있으면 그것을, 없으면 고른 폴더 자체를 세션 폴더로 본다.
   - **이 PC 의 기본 세션 폴더**(자동 검색용, 표본 미확인): "문서" 폴더 아래 `NetSarang Computer\8\Xshell\Sessions`, 없으면 `…\7\…`, `…\6\…`, 그다음 옛 판의 `NetSarang\Xshell\Sessions` 순으로 처음 있는 것. "문서" 폴더는 Windows 에서 `HKEY_CURRENT_USER\Software\Microsoft\Windows\CurrentVersion\Explorer\User Shell Folders` 의 `Personal`(환경 변수를 펼침), 그다음 `%USERPROFILE%\Documents`. 비 Windows 는 기본 폴더가 없다(빈 목록). 없으면 빈 목록.
 - **폴더 구조**: 세션 폴더(내보내기 파일에서는 `Xshell/`) 아래 하위 폴더를 모두 따라 내려간다. 확장자 `.xsh`(대소문자 무시) 파일 하나가 세션 하나. 세션 이름 = 파일 이름에서 `.xsh` 를 뗀 것. `folderPath` = 그 파일까지의 폴더 이름들(최상위 세션은 빈 배열). 폴더 이름이 빈 것·`.`·`..` 인 항목은 버린다. 다른 확장자(`default.xshf.xfs` 등)는 읽지 않는다. 순서는 폴더 경로·이름의 사전순.
@@ -65,7 +66,14 @@
 - **인증**: 기본 `password`. ssh 세션에서 `[CONNECTION:AUTHENTICATION] UserKey` 가 비어 있지 않으면 `key`(keyFile 은 없음 — 적용 규칙 §4 에 따라 서버는 `agent` 인증이 된다)이고 후보 `note` 에 `Xshell 사용자 키 "{이름}" 은 옮기지 않음 (키를 내보내 서버 편집에서 지정)`. 사용자 키가 없고 `[CONNECTION:SSH] UseAuthAgent` 가 0 이 아니면 `key`(안내 없음). (둘 다 표본 미확인) **저장된 비밀번호(`Password`)·암호문(`Passphrase`)은 읽지 않는다.**
   - 사용자 키를 옮기지 않는 까닭: Xshell 은 키를 세션 파일이 아니라 자기 키 보관소에 두고 세션에는 이름만 적는다.
 - **자동 실행 명령**: ssh 세션의 `[CONNECTION:SSH] RemoteCommand` 가 비어 있지 않으면 `startupCommand`(표본 미확인).
-- **옮기지 않고 알리는 것**(후보 `note`, 여러 개면 ` · ` 로 잇는다): 위의 사용자 키, ssh 세션의 `[CONNECTION:SSH] FwdReqCount` 가 양수면 `포트 포워딩 {개수}개는 옮기지 않음`(포워딩 항목의 형식을 확인할 표본이 없다), `[CONNECTION:PROXY] Proxy` 가 비어 있지 않으면 `프록시 설정 "{이름}" 은 옮기지 않음`.
+- **포트 포워딩**(2026-11-25, D129 — 사용자가 준 세션 파일 하나(포워딩 43개, 모두 나가는 것)로 형식을 확인했다. 전에는 옮기지 않고 개수만 알렸다): ssh 세션의 `[CONNECTION:SSH] FwdReqCount` 가 항목 수 N 이고(1000 을 넘으면 1000), 항목 i(0 ~ N-1)는 같은 구역의 키 `FwdReq_{i}_{항목}` 들이다. 키들은 파일 안에서 순서 없이 흩어져 있다.
+  - `Incoming`: `0`(또는 없음) = 나가는 것 → 로컬 `L`. `1` = 들어오는 것 → 원격 `R`(표본 미확인). 그 밖의 값(동적 등, 표본 미확인)은 옮기지 않는다.
+  - `Port` = 듣는 포트(bindPort), `Host`·`HostPort` = 대상 주소·포트(destHost·destPort). 포트가 1~65535 밖이거나 대상 주소가 비면 그 항목은 옮기지 않는다.
+  - `Source` = 듣는 주소(bindHost). 비었거나 `localhost`(대소문자 무시)면 `127.0.0.1`, 다른 값은 그대로.
+  - `LocalOnly`("로컬 연결만 허용", 0·1)는 **쓰지 않는다**: 0 이어도 듣는 주소를 넓히지 않는다(가져오기만으로 이 PC 의 포트가 네트워크에 열리지 않게 한다. 필요하면 가져온 뒤 서버 편집에서 듣는 주소를 바꾼다).
+  - `Description`(항목의 설명): 포워딩에는 이름을 둘 자리가 없으므로, 설명이 있는 항목마다 `{L|R} {듣는 포트} → {대상 주소}:{대상 포트}  {설명}` 한 줄을 만들어 서버 `notes` 끝에 붙인다 — 머리 줄 `포트 포워딩 이름 (Xshell):` 뒤에 줄바꿈으로 잇고, 앞에 메모(`[CONNECTION] Description`)가 있으면 빈 줄 하나를 사이에 둔다.
+  - 옮기지 못한 항목이 있으면 후보 `note` 에 `포트 포워딩 {개수}개는 옮기지 않음`.
+- **옮기지 않고 알리는 것**(후보 `note`, 여러 개면 ` · ` 로 잇는다): 위의 사용자 키, 위의 옮기지 못한 포워딩 수, `[CONNECTION:PROXY] Proxy` 가 비어 있지 않으면 `프록시 설정 "{이름}" 은 옮기지 않음`.
 - 그 밖의 값(글꼴·색 구성·탭 색·로그·X11 전달·암호화 알고리즘 목록 등)은 읽지 않는다.
 - 후보 이름·서버 별칭 = 세션 이름.
 
```

### `spec/ui/dialogs.md` — 수정

```diff
--- a/spec/ui/dialogs.md
+++ b/spec/ui/dialogs.md
@@ -104,7 +104,7 @@
 1. **안내 박스**: "PuTTY 저장 세션(레지스트리), SecureCRT 세션 폴더, ~/.ssh/config를 읽어 옵니다. 호스트·포트·사용자·키 파일·포트 포워딩·시리얼 설정·인코딩이 옮겨집니다. 비밀번호는 옮기지 않으므로 처음 연결할 때 한 번 입력하면 저장됩니다." 볼트가 잠겨 있으면 문장 끝에 " (앱이 잠겨 있으면 키 파일은 저장되지 않습니다)" 추가.
 2. **AWS EC2 불러오기** 버튼(구름 아이콘, 진행 중 "AWS 전 리전 조회 중...") + 설명 "aws CLI 자격 증명으로 인스턴스를 읽습니다. 사설 IP 서버는 같은 VPC 의 경유 서버를 경유 서버로 잇고, 키는 ~/.ssh/<키 이름>.pem 에서 찾습니다."
    - **SecureCRT 폴더 선택** 버튼(폴더 아이콘, 진행 중 "SecureCRT 폴더 읽는 중...") + 설명 "다른 위치(다른 PC에서 복사한 Config 폴더 등)의 SecureCRT 세션을 읽습니다. 폴더 구조는 넣을 폴더 아래에 그대로 만들어집니다." (신규 기능, D66)
-   - **Xshell 내보내기 파일 선택** 버튼(압축 파일 아이콘, 진행 중 "Xshell 파일 읽는 중...") + 설명 "Xshell 의 [파일 > 내보내기]로 만든 .xts 파일에서 세션을 읽습니다. 폴더 구조는 넣을 폴더 아래에 그대로 만들어집니다. 비밀번호·사용자 키·포트 포워딩·프록시 설정은 옮기지 않습니다." (신규 기능, 2026-11-25, D128)
+   - **Xshell 내보내기 파일 선택** 버튼(압축 파일 아이콘, 진행 중 "Xshell 파일 읽는 중...") + 설명 "Xshell 의 [파일 > 내보내기]로 만든 .xts 파일이나 세션 파일(.xsh) 하나에서 세션을 읽습니다. 폴더 구조와 포트 포워딩은 그대로 옮겨집니다. 비밀번호·사용자 키·프록시 설정은 옮기지 않습니다." (신규 기능, 2026-11-25, D128. 포트 포워딩·세션 파일 하나는 D129)
 3. 후보 검색 중이면 "검색 중...". 후보가 없으면 "가져올 세션을 찾지 못했습니다." 줄바꿈 "PuTTY·SecureCRT에 저장된 세션이 없거나 ~/.ssh/config가 없습니다."
 4. 후보가 있으면:
    - 필드 "넣을 폴더 이름"(기본값 "PuTTY 가져오기", 단 자동 검색 결과에 PuTTY 후보가 없고 SecureCRT 후보가 있으면 "SecureCRT", PuTTY·SecureCRT 후보가 없고 Xshell 후보가 있으면 "Xshell"), 도움말 "이름이 prod-/stg-/dev- 로 시작하면 운영/검증/개발 환경이 자동 지정됩니다. 비우면 최상위에 넣습니다. SecureCRT 폴더 구조는 이 폴더 아래에 만들어집니다."
```

### `docs/scenarios.md` — 수정

```diff
--- a/docs/scenarios.md
+++ b/docs/scenarios.md
@@ -152,7 +152,7 @@
 | 10-1 | 가져오기 열기 | 사이드바 [가져오기], 빠른 연결 `> 가져오기`, 시작 화면 | PuTTY 저장 세션(레지스트리), SecureCRT 세션 폴더, `~/.ssh/config` 를 자동으로 찾아 목록으로 보인다. |
 | 10-2 | 고르기 | 체크, 전체 선택 | 이름·주소·종류·출처, 키 파일 표시. "넣을 폴더 이름"(기본 "PuTTY 가져오기", prod-/stg-/dev- 로 시작하면 환경 자동). |
 | 10-3 | SecureCRT 다른 위치 | [SecureCRT 폴더 선택] | 다른 PC 에서 복사한 Config 폴더도 읽는다. 폴더 구조와 포트 포워딩을 그대로. |
-| 10-3a | Xshell | [Xshell 내보내기 파일 선택] | Xshell 의 [파일 > 내보내기]로 만든 `.xts` 파일에서 세션(이름·폴더 구조·호스트·포트·사용자·메모·인코딩)을 읽는다. 이 PC 에 Xshell 이 있으면 그 세션은 자동으로 목록에 나온다. 비밀번호·사용자 키·포트 포워딩·프록시·시리얼 세션은 옮기지 않는다. |
+| 10-3a | Xshell | [Xshell 내보내기 파일 선택] | Xshell 의 [파일 > 내보내기]로 만든 `.xts` 파일에서 세션(이름·폴더 구조·호스트·포트·사용자·메모·인코딩·포트 포워딩)을 읽는다. 세션 파일(`.xsh`) 하나만 골라도 된다. 포워딩에 붙인 이름은 서버 메모에 적힌다. 이 PC 에 Xshell 이 있으면 그 세션은 자동으로 목록에 나온다. 비밀번호·사용자 키·프록시·시리얼 세션은 옮기지 않는다. |
 | 10-4 | AWS | [AWS EC2 불러오기] | aws CLI 자격 증명으로 모든 리전 인스턴스. 사설 IP 는 같은 VPC 배스천을 경유 서버로, 키는 `~/.ssh/<키 이름>.pem`. |
 | 10-5 | 가져오기 | [n개 가져오기] | "n개 서버를 가져왔습니다 (키 m개 암호화해 저장)". 비밀번호는 옮기지 않는다(처음 연결 때 한 번 입력). |
 
```

### `docs/scenarios.en.md` — 수정

```diff
--- a/docs/scenarios.en.md
+++ b/docs/scenarios.en.md
@@ -151,7 +151,7 @@
 | 10-1 | Open Import | Sidebar [Import], Quick connect `> Import`, start screen | Finds PuTTY saved sessions (registry), SecureCRT session folders and `~/.ssh/config` and lists them. |
 | 10-2 | Choose | Tick items, select all | Name, address, type, source, key file. "Target folder name" (default "PuTTY import"; names starting with prod-/stg-/dev- get that environment). |
 | 10-3 | SecureCRT elsewhere | [Choose SecureCRT folder] | Reads a Config folder copied from another PC too, keeping folder structure and port forwarding. |
-| 10-3a | Xshell | [Choose Xshell export file] | Reads sessions (name, folder structure, host, port, user, notes, encoding) from an `.xts` file made with [File > Export] in Xshell. If Xshell is installed on this PC its sessions are listed automatically. Passwords, user keys, port forwarding, proxies and serial sessions are not imported. |
+| 10-3a | Xshell | [Choose Xshell export file] | Reads sessions (name, folder structure, host, port, user, notes, encoding, port forwarding) from an `.xts` file made with [File > Export] in Xshell, or from a single session file (`.xsh`). Names given to forwarding rules go into the server notes. If Xshell is installed on this PC its sessions are listed automatically. Passwords, user keys, proxies and serial sessions are not imported. |
 | 10-4 | AWS | [Load AWS EC2] | All regions via your aws CLI credentials. Private IPs go through a bastion in the same VPC as the jump server; keys from `~/.ssh/<key name>.pem`. |
 | 10-5 | Import | [Import N] | "Imported N servers (M keys saved encrypted)". Passwords are not imported (enter once on first connection). |
 
```

`spec/ipc.md` 의 `import.scanXshell` 줄은 인자 설명에 "세션 파일(`.xsh`)"을, 파일 종류를 `*.xts;*.xsh` 로 고쳤다. `CHANGELOG.md` 의 Xshell 줄을 고쳤고(포워딩·세션 파일 하나), `docs/decisions.md` 에 D129 를 넣었다.

---

## G. 검증 (2026-11-25 실행)

### G.1 [internal/importer/xshell_test.go](../../internal/importer/xshell_test.go) 에 3개 추가

테스트의 값은 모두 지어낸 것이다.

| 테스트 | 확인하는 것 |
|---|---|
| `TestParseXshForwards` | 키가 뒤섞여 있어도 번호로 읽는다 / `Incoming` 0 → `L`, 1 → `R` / `localhost` → `127.0.0.1`, 다른 듣는 주소는 그대로 / `LocalOnly=0` 이어도 `127.0.0.1` / 모르는 종류·듣는 포트 0·대상 없음은 옮기지 않고 안내 "포트 포워딩 3개는 옮기지 않음" / `FwdReqCount` 밖의 번호는 읽지 않는다 / 설명이 있는 규칙만 메모에 한 줄씩, 세션 설명 뒤에 빈 줄을 두고 붙는다 / 설명도 메모도 없으면 메모는 비고, 모두 옮겼으면 안내가 없다 / 텔넷 세션의 포워딩은 읽지 않는다 |
| `TestScanXshellSingleSessionFile` | `.xsh` 파일 하나를 고르면 그 세션만, 폴더 없이 읽는다(확장자 대소문자 무시, 같은 폴더의 다른 세션은 읽지 않는다, 포워딩도 읽힌다) |
| `TestXshellRealSessionForwards` | (선택 실행) 환경 변수 `DARITERM_XSH` 에 실제 `.xsh` 경로를 주면 포워딩을 읽어 **수만** 적는다. 변수가 없으면 건너뛴다 |

5번 문서의 `TestParseXshSessionVariants` 는 고치지 않고 통과한다(`FwdReqCount=2` 인데 규칙 내용이 없는 세션 → 둘 다 옮기지 못함 → 같은 안내 문구).

### G.2 실제 파일로 확인

사용자가 준 세션 파일로 `TestXshellRealSessionForwards` 를 돌린 결과(값은 출력하지 않고 수만 센다):

```
적힌 개수 43 → 포워딩 43개 (종류 map[L:43], 듣는 주소 map[127.0.0.1:43], 듣는 포트 43가지), 이름 있음 43, 옮기지 못함 0
후보: 포워딩 43개, 메모 46줄, 안내 있음 false
```

메모 46줄 = 세션 설명 1줄 + 빈 줄 1 + 머리 줄 1 + 규칙 43줄. 5번 문서의 내보내기 파일도 다시 돌렸다: 세션 164개 → 후보 164개, 결과가 전과 같다.

### G.3 실행한 명령

```bash
go build ./... && go vet ./internal/importer/ ./internal/app/     # 통과
go test -count=1 ./...                 # 전부 통과 (이 PC 는 cgo 가 꺼져 있어 -race 는 돌리지 못했다)
DARITERM_XSH="<받은 세션 파일 경로>" go test -count=1 -run TestXshellRealSessionForwards -v ./internal/importer/   # §G.2
(cd frontend && npx tsc --noEmit)      # 통과
(cd frontend && npx vitest run)        # PowerShell 에서. 51개 파일 348개 통과 (프런트 코드는 바꾸지 않았다, 문구 검사 포함)
go run github.com/wailsapp/wails/v2/cmd/wails@v2.17.1 build   # build/bin/dariterm.exe
```

자동 테스트가 확인하지 못하는 것:
- 실제 앱에서 가져온 서버의 포워딩이 접속했을 때 열리는지(가져오기는 저장까지만 한다).
- 파일 선택 창에 `.xsh` 가 보이는지.
- `Incoming=1`(원격)과 동적 규칙이 든 파일(표본이 없다).

## H. 수동 검증 (남은 일)

| # | 절차 | 기대 결과 |
|---|---|---|
| 1 | [가져오기] → [Xshell 내보내기 파일 선택] | 파일 선택 창에 `.xts` 와 `.xsh` 가 보인다. 버튼 옆 설명에 "폴더 구조와 포트 포워딩은 그대로 옮겨집니다" |
| 2 | 받은 `.xsh` 파일을 고른다 | 목록에 그 세션 한 줄(폴더 경로 없음), 출처 "Xshell". 툴팁에 "옮기지 않음" 안내가 없다 |
| 3 | 가져온 뒤 그 서버를 편집으로 연다 | 포트 포워딩 43개가 모두 로컬, 듣는 주소 `127.0.0.1`, 포트·대상이 Xshell 과 같다 |
| 4 | 그 서버의 메모 | 세션 설명 아래에 "포트 포워딩 이름 (Xshell):" 과 규칙마다 `L 포트 → 대상:포트  이름` 한 줄 |
| 5 | 그 서버에 접속한 뒤 포워딩 포트 하나로 접속해 본다(예: `ssh -p <듣는 포트> localhost`) | 대상 서버로 이어진다 |
| 6 | 같은 네트워크의 다른 PC 에서 이 PC 의 그 포트로 접속 | 되지 않는다(`LocalOnly` 를 옮기지 않았다). 필요하면 서버 편집에서 그 규칙의 듣는 주소를 `0.0.0.0` 으로 |
| 7 | Xshell 에서 들어오는(원격) 규칙·동적 규칙을 하나씩 넣은 세션을 가져온다 | 원격은 종류 R 로, 동적은 옮기지 않고 "포트 포워딩 1개는 옮기지 않음" 안내 — 다르게 나오면 그 파일로 다시 맞춘다 |

## I. 영향 범위·회귀 위험

| 영역 | 위험 | 대비 |
|---|---|---|
| 세션 설명이 메모로 | 5번 문서부터 Xshell 의 세션 설명(`Description`)을 서버 메모로 옮긴다. **받은 세션 파일의 설명에는 계정·비밀번호로 보이는 글이 적혀 있다.** 서버 메모는 `store.json` 에 암호화 없이 저장되고 git 동기화를 켜면 저장소로도 간다 | 가져온 뒤 그 서버의 메모를 확인해 지우는 것을 권한다. 비밀번호는 접속할 때 한 번 입력하면 앱이 암호화해 저장한다 |
| 듣는 주소 | Xshell 에서 다른 PC 도 쓰게 열어 둔 규칙(`LocalOnly=0`)이 이 PC 에서만 되게 바뀐다 | 일부러 그렇게 했다(§3). 수동 검증 6 |
| 포트가 겹침 | 여러 서버를 가져와 듣는 포트가 겹치면, 나중에 접속한 쪽의 그 포워딩이 열리지 않는다 | Xshell 에서도 같다. 가져오기는 고치지 않는다 |
| 포워딩이 많은 서버 | 접속할 때 43개를 한꺼번에 연다 | 이 앱의 포워딩 기능 그대로다. 수동 검증 5 |
| 메모가 길어짐 | 규칙마다 한 줄이 붙는다 | 설명이 있는 규칙만. 지워도 포워딩에는 영향이 없다 |
| `Incoming` 의 다른 값 | 1 을 원격으로 본 것은 이름에 기댄 추정이다. 동적 규칙은 옮기지 않는다 | 옮기지 못한 수를 알린다. 수동 검증 7 |
| 포워딩 이름 | 서버 편집의 포워딩 목록에는 이름이 보이지 않는다(메모에만 있다) | 포워딩 항목에 이름 필드를 두려면 저장 형식·서버 편집 화면·명세를 함께 고쳐야 한다. 필요하면 따로 다룬다 |
| 번역 | 한국어·영어 외 8개 언어의 문구(새 1개, 고친 1개)는 이 작업을 한 세션이 번역했다 | 검수가 필요할 수 있다 |
| 받은 파일 | 저장소 폴더 바로 위(`c:\work\win-projects-main`)에 둔 세션 파일(`.xsh`)에는 실제 서버 주소와 계정 정보가 들어 있다 | 테스트·문서에는 그 값을 넣지 않았다. 가져오기를 마쳤으면 지우는 것이 낫다 |

되돌리기: §B·§C 의 `-` 줄로 되돌리고, 문구 `be.xsh.forwardNames` 를 지우고 `import.xshNote` 를 5번 문서의 글로 되돌린다. 이미 가져온 서버의 포워딩과 메모는 그대로 남는다.

## J. 체크리스트

- [x] `xshell.go`: `ParseXshForwards`(종류·주소·포트·설명·옮기지 못한 수), 메모에 설명 붙이기, `.xsh` 하나 읽기
- [x] `channels_import.go`: 파일 선택 창의 종류 `*.xts;*.xsh`
- [x] 문구: `be.xsh.forwardNames` 추가, `import.xshNote` 고침(10개 언어)
- [x] Go 테스트 3개 추가. `go test ./...`·`tsc`·`vitest`(348개)·빌드
- [x] 실제 세션 파일로 확인(43개 → 43개), 5번 문서의 내보내기 파일 다시 확인
- [x] 명세 import.md §1.6·ipc.md·ui/dialogs.md §5, 시나리오 ko·en, CHANGELOG, decisions D129, CLAUDE.md
- [x] 수동 검증 1~7 — 2026-11-25 사용자가 앱에서 확인함
