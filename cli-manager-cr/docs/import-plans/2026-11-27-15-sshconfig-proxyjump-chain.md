# ssh config 가져오기 — ProxyJump 여러 단계 · ProxyCommand

**작성**: 2026-11-27 · **상태**: 코드·자동 테스트·빌드 완료(D153, ProxyCommand 가져오기는 D152). 앱에서의 확인은 수동 검증으로 남음(§H) · **대상**: cli-manager-cr(DariTerm) · **Known issue**: 없음
**관련**: spec/import.md §2 (전에는 "쉼표로 이어진 첫 항목만"), [-14-](./2026-11-27-14-proxy-command.md)

---

## 0. 한 줄 요약

`ProxyJump a,b` 를 대상 → **b** → **a** 의 경유 서버 체인으로 잇는다(대상의 경유 서버는 b, b 의 경유 서버는 a). 전에는 첫 항목(a)만 대상의 경유 서버로 삼아, 실제로는 b 를 거쳐야 하는 서버가 a 에 바로 붙으려다 실패했다. 함께 `ProxyCommand` 도 서버의 프록시 명령으로 옮긴다.

## 1. 결정

| 항목 | 결정 | 까닭 |
|---|---|---|
| 읽는 순서 | 쉼표로 나눈 모든 단계, 바깥(이 PC 가 먼저 연결) → 안 | OpenSSH 의 ProxyJump 와 같은 뜻 |
| 단계 주소 | 같은 파일의 Host 이름이면 그 HostName, 아니면 `[user@]host[:port]` 의 host | 전과 같은 해석을 단계마다 |
| 대상의 경유 서버 | **마지막** 단계(`jumpVia`) | 대상에 가장 가까운 서버 |
| 체인 정보 | 단계가 둘 이상이면 후보 `jumpChain`(바깥부터 전체) | 적용 때 단계끼리 잇는 데 쓴다 |
| 단계끼리 잇기 | 각 단계 서버가 바로 앞 단계를 경유 서버로 | 서버마다 경유 서버 하나(store 형식)로 체인을 표현 |
| 바꾸는 서버 | **이번 가져오기로 새로 만든 서버만**. 이미 있던 서버, 이미 다른 경유 서버가 정해진 새 서버는 그대로 두고 `{이름}: 이미 다른 경유 서버가 정해져 있거나 기존 서버라서 경유 서버를 {주소} 로 바꾸지 않음` | 기존 서버는 다른 서버가 그대로 쓰고 있을 수 있다. 같은 서버가 두 체인의 중간에 있으면 첫 것이 이긴다 |
| 단계를 못 찾으면 | 그 연결만 건너뛰고 `be.imp.jumpMissing` 안내 | 대상 쪽 연결은 이미 같은 안내를 쓴다 |
| ProxyCommand | ProxyJump 가 없을 때만, `proxy = {type: command, command: 원래 값}`. 메모 `ProxyCommand …` | OpenSSH 도 둘 다 있으면 ProxyJump |

## 2. 구현

### 2.1 후보 — `internal/importer/types.go`

```go
type Candidate struct {
	…
	M_szJumpVia string `json:"jumpVia,omitempty"`
	// M_slJumpChain 은 경유 서버가 여러 단계일 때 바깥(이 PC 가 먼저 연결하는 서버)부터의 주소 목록이다.
	// 마지막 항목이 M_szJumpVia 와 같다. 한 단계면 비어 있다 (ssh config 의 ProxyJump a,b, D153).
	M_slJumpChain []string `json:"jumpChain,omitempty"`
	…
}

// HostDraft
	// M_pProxy 는 프록시 경유 설정이다 (지금은 ssh config 의 ProxyCommand 만, D152).
	M_pProxy *ProxyDraft `json:"proxy,omitempty"`

type ProxyDraft struct {
	M_szType    string `json:"type"`              // 지금은 command 만
	M_szCommand string `json:"command,omitempty"` // 프록시 명령 줄 (%h %p %r)
}
```

프런트의 후보 타입(`ipc/c1types.ts`)에도 `jumpChain?: string[]` — 화면은 후보를 그대로 되돌려 보낸다.

### 2.2 읽기 — `internal/importer/sshconfig.go`

```go
	if szProxy := pBlock.mapOpts["proxyjump"]; szProxy != "" && !strings.EqualFold(szProxy, "none") {
		// 쉼표로 이은 여러 단계는 앞(이 PC 가 먼저 연결하는 서버)부터 차례로 거친다 (D153).
		// 대상 서버는 마지막 단계를 경유 서버로 삼고, 각 단계는 바로 앞 단계를 경유 서버로 삼는다.
		slChain := []string{}
		for _, szHop := range strings.Split(szProxy, ",") {
			if szHop = strings.TrimSpace(szHop); szHop == "" {
				continue
			}
			if szResolved, bSame := mapHostNames[szHop]; bSame {
				slChain = append(slChain, szResolved)
			} else {
				slChain = append(slChain, bareHost(szHop))
			}
		}
		cand.M_szNote = "ProxyJump " + szProxy
		host.M_szNotes = cand.M_szNote
		if len(slChain) > 0 {
			cand.M_szJumpVia = slChain[len(slChain)-1]
		}
		if len(slChain) > 1 {
			cand.M_slJumpChain = slChain
		}
	} else if szCommand := pBlock.mapOpts["proxycommand"]; szCommand != "" && !strings.EqualFold(szCommand, "none") {
		// ProxyCommand 는 서버의 프록시 명령으로 옮긴다 (D152). ProxyJump 가 있으면 그것이 이긴다 (OpenSSH 와 같음).
		host.M_pProxy = &ProxyDraft{M_szType: "command", M_szCommand: szCommand}
		cand.M_szNote = "ProxyCommand " + szCommand
	}
```

### 2.3 적용 — `internal/importer/apply.go`

대상의 경유 서버를 잇는 기존 단계 뒤에:

```go
	// 여러 단계(ProxyJump a,b): 각 단계 서버가 바로 앞 단계를 경유 서버로 삼게 잇는다 (D153).
	for _, mapHost := range slNewHosts {
		szSelf, _ := mapHost["id"].(string)
		if slChain, bChain := mapChainPending[szSelf]; bChain {
			linkJumpChain(slChain, slNewHosts, out.Hosts, szSelf, &out.Result)
		}
	}

// linkJumpChain 은 경유 서버 체인(바깥부터의 주소)의 각 단계를 바로 앞 단계로 잇는다.
// 이번에 새로 만든 서버만 바꾼다: 이미 있던 서버는 건드리지 않고, 다른 경유 서버가 정해져 있으면 그대로 둔다.
// 잇지 못한 단계는 건너뛰기 목록에 남긴다.
func linkJumpChain(slChain []string, slNewHosts []map[string]any, slOldHosts []map[string]any, szSelf string, pResult *ApplyResult) {
	… findAny: 새 서버 → 기존 서버 순으로 주소가 같은 서버(자기 자신 제외)
	for iHop := len(slChain) - 1; iHop >= 1; iHop-- {
		mapHop, bHop := findAny(slChain[iHop])
		mapPrev, bPrev := findAny(slChain[iHop-1])
		if !bHop {
			continue // 대상 서버를 이을 때 이미 "찾지 못함"을 남겼거나, 중간 단계가 목록에 없다
		}
		if !bPrev {
			→ be.imp.jumpMissing({단계 이름}, {앞 단계 주소})
			continue
		}
		switch {
		case 이미 앞 단계로 이어져 있음:   // 그대로
		case 기존 서버:                    → be.imp.jumpChainKept
		case 다른 경유 서버가 정해져 있음:  → be.imp.jumpChainKept
		default:                          mapHop["jumpHostId"] = 앞 단계 id
		}
	}
}
```

`buildHost` 는 `host.M_pProxy`(command, SSH 만)를 `proxy = {type: command, command, host: "", port: 0, username: "", credentialId: nil}` 로 쓴다.

### 2.4 문구 (10개 언어)

`be.imp.jumpChainKept` — "{name}: 이미 다른 경유 서버가 정해져 있거나 기존 서버라서 경유 서버를 {jump} 로 바꾸지 않음"

## 3. 명세

spec/import.md §2(ProxyJump·ProxyCommand), §2 끝 "적용", spec/ipc.md 후보의 `jumpChain`.

## 4. 테스트 — `internal/importer/jumpchain_test.go`

시험 설정:

```
Host a          → 10.0.0.1
Host b          → 10.0.0.2
Host target     → 10.0.0.3, ProxyJump a, b
Host pc         → 10.0.0.4, ProxyCommand ssh -W %h:%p a
Host both       → 10.0.0.5, ProxyJump a + ProxyCommand nc %h %p
```

| 테스트 | 확인 |
|---|---|
| TestSSHConfigJumpChainAndProxyCommand | target: jumpVia 10.0.0.2, jumpChain [10.0.0.1, 10.0.0.2]. pc: 프록시 명령. both: ProxyJump 만 |
| TestApplyJumpChainLinksEachHop | target → b → a, a 는 바로 연결, pc 의 proxy 가 command, 건너뛴 것 없음 |
| TestApplyJumpChainKeepsExistingHosts | b 가 기존 서버면 target 은 기존 b 로 잇고 기존 b 는 바꾸지 않으며 안내 하나 |
| TestApplyJumpChainMissingHop | a 가 없으면 b 는 바로 연결, "10.0.0.1 을 찾지 못함" 안내 |

기존 가져오기 시험(한 단계 ProxyJump 포함) 모두 통과.

## 5. 검증 결과 (2026-11-27)

`go test -race ./internal/importer/` 통과, 전체 검증은 다른 계획서와 같다.

## H. 수동 검증

| # | 하는 일 | 기대 | 결과 |
|---|---|---|---|
| 1 | `~/.ssh/config` 에 §4 와 같은 a·b·target 을 두고 가져오기에서 셋 모두 선택 | 가져온 target 편집 > 경유 서버 = b, b 의 경유 서버 = a | |
| 2 | target 연결 | "경유 서버 a 경유 중..." → "b 경유 중..." 뒤 접속 | |
| 3 | target 만 선택해 가져오기(a·b 는 이미 등록) | target → 기존 b, 기존 b 가 경유 서버 없으면 안내 | |
| 4 | ProxyCommand 가 있는 Host 가져오기 | 서버 편집 > 고급 > 프록시 명령에 그 줄 | |

## 체크리스트

- [x] 후보 · [x] 읽기 · [x] 적용·체인 잇기 · [x] ProxyCommand · [x] 문구 · [x] 명세 · [x] 테스트 · [ ] §H
