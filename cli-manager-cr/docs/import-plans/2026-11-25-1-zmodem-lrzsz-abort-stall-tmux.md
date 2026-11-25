# ZMODEM 을 실제 lrzsz 와 맞대어 본 결과와 수정 3가지 (취소 뒤 화면에 쏟아지는 데이터 · 응답 없는 서버 · tmux) — 원인·수정·코드·검증

**작성**: 2026-11-25 · **상태**: 코드 수정·자동 테스트·실제 서버 검증 완료(D123-2·D125). 앱 화면에서의 수동 검증(§H)도 2026-11-25 사용자 확인 완료 · **대상**: cli-manager-cr(DariTerm) · **Known issue**: KI-012 (이 문서로 resolved)
**관련**: 기능을 만든 문서 [2026-11-23-zmodem.md](./2026-11-23-zmodem.md)(§2.3 검증의 한계, §H 수동 검증, §H-1 결과 요약), 동작 명세 [spec/behavior/zmodem.md](../../spec/behavior/zmodem.md) §3·§7·§10, 결정 [docs/decisions.md](../decisions.md) D123-2·D125, 같은 날의 다른 문서 [find-clipboard-ki011](./2026-11-25-2-find-clipboard-ki011.md)

> 이 문서 하나로 **무엇을 확인했는지 → 무엇이 문제였는지 → 왜 그랬는지 → 어떻게 고쳤는지(코드 전체·패치) → 어떻게 검증했는지**를 알 수 있게 쓴다.
> 패치의 "고치기 전"은 2026-11-25 아침의 소스다(2026-11-23 의 ZMODEM 구현 그대로).

---

## 0. 한 줄 요약

2026-11-23 에 만든 ZMODEM 전송(D114)은 **실제 `sz`·`rz` 와 한 번도 맞대어 보지 못한 채**였다(KI-012). 2026-11-25 에 시험 서버에 lrzsz 를 설치하고 맞대어 봤다.

| | 결과 |
|---|---|
| 받기·보내기 | **형식이 맞는다.** 15바이트부터 40MB 이진 파일까지 SHA256 이 같다 |
| 찾은 결함 1 (D123-2) | 받는 중 Ctrl+C 를 누르면 서버가 이미 보낸 데이터(약 1.8MB)가 **화면에 쏟아졌다** → 버리고 프롬프트만 낸다 |
| 남아 있던 한계 2 (D125) | 서버가 응답 없이 멈추면 **전송 상태로 영영 남았다** → 60초 시한, 서버가 조용해도 Ctrl+C 가 곧바로 듣는다 |
| 남아 있던 한계 3 (D125) | tmux 안에서 `sz`·`rz` 를 실행하면 **아무 안내 없이 멈춰 있었다** → 전송할 수 없다는 것과 끝내는 방법을 알린다 (전송 자체는 여전히 안 된다) |

코드는 `internal/zmodem`(엔진)·`internal/session`(세션 연결) 두 패키지와 문구 2개, 그리고 검증 도구 `tools/zmodemcheck`(새로 만듦)다. 프런트엔드는 바뀌지 않았다.

## 1. 배경

### 1.1 사용자 요청

1. "192.168.12.204에 zmodem을 설치하고 검증해줘." → §2 (설치·검증), 그 과정에서 결함 1 발견·수정(D123-2)
2. "KI-011와 KI-012를 수정해주고 오늘 수정한 것을 … import-plans에 추가해줘." → KI-012 의 남은 두 가지 수정(D125), 이 문서. KI-011 은 [다른 문서](./2026-11-25-2-find-clipboard-ki011.md)

### 1.2 KI-012 가 적어 둔 것 (2026-11-23)

| # | 증상(그때는 예상) | 이 문서에서 |
|---|---|---|
| 1 | 자동 테스트는 이 구현의 보내는 쪽·받는 쪽을 서로 맞물린 것뿐이라, 실제 `sz`·`rz` 와는 형식이 어긋날 수 있다 | §2 — 맞는 것을 확인. 맞대는 중에 결함 1 을 찾음(§3.1) |
| 2 | 서버 쪽 프로그램이 응답 없이 멈추면 세션이 전송 상태로 남는다. Ctrl+C 를 눌러야 풀린다 | §3.2 — 고침. 실은 Ctrl+C 로도 곧바로 풀리지 않는 경우가 있었다 |
| 3 | tmux 안에서는 대개 동작하지 않는다 | §3.3 — 실제로 확인, 안내를 냄 |

### 1.3 용어

| 말 | 뜻 |
|---|---|
| `sz` / `rz` | 서버의 lrzsz 프로그램. `sz 파일` = 서버가 보낸다(이 PC 는 받는다), `rz` = 서버가 받는다(이 PC 는 보낸다) |
| 시작 표시 | `sz`·`rz` 가 처음 내보내는 16진 헤더의 앞부분 `*` `*` `0x18` `B` `0` + `0`(sz) 또는 `1`(rz). 세션의 읽기 루프가 출력에서 이것을 찾으면 전송을 시작한다 |
| ZDLE | 값 0x18 인 제어 문자(CAN 과 같은 값). ZMODEM 이 헤더·데이터 조각의 경계와 바꿔 쓰기(이스케이프)에 쓴다 |
| 데이터 조각 | 파일 내용 최대 1024바이트 + `ZDLE` + 끝 표시(`h`·`i`·`j`·`k`) + CRC(2 또는 4바이트). 조각 안에 0x18 이 있으면 `ZDLE` + 다른 글자로 바꿔서 보내므로, **`ZDLE` + `h`~`k` 는 조각의 끝에만 나온다** |
| 중단 신호 | CAN(0x18) 8개 + 백스페이스 8개. 받은 쪽은 CAN 이 5개 이어지면 전송을 그만둔다 |
| 엔진 | `internal/zmodem` 의 `Receive`·`Send`. 연결을 `Stream`(Read·Write) 으로만 본다 |
| 읽기 루프 | `internal/session/session.go` 의 `readLoop`. 연결에서 읽어 화면으로 보내는 고루틴(세션마다 하나) |

---

## 2. 실제 lrzsz 와 맞대어 보기

### 2.1 서버에 lrzsz 설치

- 서버: 시험 서버 192.168.12.204 — Rocky Linux 9.8 (x86_64), 호스트 이름 `kai-test-vm-02`, 계정 `mds`.
- 저장소가 하나도 없고(`dnf repolist` → "No repositories available") 인터넷도 닿지 않아 `dnf install lrzsz` 를 쓸 수 없었다.
- 그래서 이 PC 에서 Rocky 공식 BaseOS 저장소의 패키지를 받아 올렸다.

| 항목 | 값 |
|---|---|
| 패키지 | `lrzsz-0.12.20-58.el9_8.x86_64.rpm` (81,529바이트) |
| 받은 곳 | `https://dl.rockylinux.org/pub/rocky/9/BaseOS/x86_64/os/Packages/l/` |
| SHA256 | `4a57d2fae3c26076f9d1893898d898032cd9072d26cf7d8069398600996b3186` (이 PC 와 서버에서 같음) |
| 서명 확인 | 서버에서 `rpm -K` → `digests signatures OK` (서버에 들어 있는 Rocky 공식 키로 확인) |
| 의존성 | glibc 뿐 |
| 설치 | `sudo -n /bin/sh -c 'rpm -ivh /tmp/lrzsz-0.12.20-58.el9_8.x86_64.rpm'` (계정 `mds` 는 `/bin/sh` 를 비밀번호 없이 sudo 로 쓸 수 있다) |
| 결과 | `/usr/bin/sz`, `/usr/bin/rz` — `sz (lrzsz) 0.12.20` |
| 뒷정리 | 올린 rpm 과 시험 파일(`~/zc`)은 지웠다. **lrzsz 는 설치된 채로 둔다**(다음 검증에 쓴다) |

지우려면: `sudo -n /bin/sh -c 'rpm -e lrzsz'`.

### 2.2 검증 도구 `tools/zmodemcheck`

앱 화면 없이도 앱과 **같은 코드 길**을 실제 서버에 물려 돌리려고 만들었다.

```
        앱                                   zmodemcheck
  프런트(xterm.js)                      화면으로 가는 글을 모아 끝에 찍음
        ▲ session.data                          ▲ session.data
  internal/session  ◀── 같은 코드 ──▶   internal/session
  (읽기 루프 · 저장 · 안내 줄 · Ctrl+C)   (읽기 루프 · 저장 · 안내 줄 · Ctrl+C)
        ▲ Transport                             ▲ Transport
  internal/sshconn 의 PTY 셸            x/crypto/ssh 의 PTY 셸 (같은 PTY 요청)
```

- 세션(`session.Manager`)·읽기 루프·받은 파일 저장(`.dtpart` → 이름 바꾸기, 같은 이름은 ` (2)`)·안내 줄·Ctrl+C 취소·엔진이 모두 앱의 것 그대로다.
- 다른 것: 화면(프런트), 파일 선택 창(`-files` 로 준 목록을 돌려준다), 접속(앱의 `sshconn` 대신 `x/crypto/ssh` 를 직접 — 호스트 키를 확인하지 않는다. 사내 시험 서버용이다).
- 비밀번호는 환경 변수 `ZCHECK_PASS` 로만 받는다(파일·명령줄 인자에 남기지 않는다).

```
go build -o zmodemcheck.exe ./tools/zmodemcheck

# 원격 명령(PTY 없음) · 파일 올리기(검증 준비용, ZMODEM 과 무관)
ZCHECK_PASS=… zmodemcheck -host H -user U exec "sha256sum ~/zc/*"
ZCHECK_PASS=… zmodemcheck -host H -user U put 로컬파일 /tmp/원격파일

# 받기 / 받는 중 Ctrl+C / 보내기 / 보내기 선택 취소
ZCHECK_PASS=… zmodemcheck -host H -user U recv -dir 받을폴더 "sz 파일…"
ZCHECK_PASS=… zmodemcheck -host H -user U recv -dir 받을폴더 -cancelAfter 8000000 "sz 큰파일"
ZCHECK_PASS=… zmodemcheck -host H -user U send -files a,b "rz"
ZCHECK_PASS=… zmodemcheck -host H -user U send "rz"

# 응답 시한을 5초로 줄여 보기 / 끝 안내 뒤에 입력 넣기(tmux 안에서 Ctrl+X 5번)
ZCHECK_PASS=… zmodemcheck -host H -user U -stall 5 recv -dir d "printf '**\030B00000000000000\r\212\021'; sleep 30"
ZCHECK_PASS=… zmodemcheck -host H -user U -then '\x18\x18\x18\x18\x18' recv -dir d "tmux new -s zc 'sz /etc/hostname; echo SZ_ENDED=\$?; exec bash'"
```

Git Bash 에서는 `/tmp/…` 같은 인자가 Windows 경로로 바뀌므로 `MSYS_NO_PATHCONV=1` 을 켜고 로컬 경로는 `cygpath -m` 으로 넘긴다.

도구가 찍는 것: 끝 안내(완료·취소·실패·중단)가 나오기까지 걸린 시간, 화면으로 간 글 전체(제어 문자는 `\x18` 처럼 보이게), **끝 안내 뒤로 화면에 나온 제어 문자 수**(전송 데이터가 화면으로 샜는지), 그리고 전송 뒤 `echo ZC_ALIVE_$((40+2))` 를 쳐서 `ZC_ALIVE_42` 가 찍히는지(셸이 살아 있고 입력이 다시 서버로 가는지).

### 2.3 결과 (2026-11-25, lrzsz 0.12.20)

2026-11-23 문서 §H 의 번호를 따른다. "수정 후"는 이 문서의 수정을 모두 넣은 최종 코드로 다시 돌린 결과다.

| §H # | 한 것 | 결과 |
|---|---|---|
| 1 | `sz /etc/hostname` (15바이트) | 통과 — 진행 줄 → `받음: hostname (15 B) → …\hostname` → `ZMODEM 전송 완료: 파일 1개, 15 B`, SHA256 같음(`83c8cf4e…62321005`), 프롬프트 복귀. 40ms |
| 2 | `sz` 40MB 무작위 이진 파일 | 통과 — SHA256 같음, 1.9초(약 21MB/s, 사내망) |
| 3 | `rz` 로 3개: 5MB 무작위 이진 `up5m.bin`, 이름에 빈칸이 있고 내용에 `\r\n`·0x18·0x11·0x13·0xff·0x00 이 든 24바이트 `up small.txt`, 300KB 0 채움 `zeros.bin` | 통과 — 서버에서 SHA256 세 개 모두 같음. 수정 시각도 전달됨(초 단위) |
| 4 | `rz` → 선택 취소(빈 목록) | 통과 — `ZMODEM 전송을 취소했습니다`, rz 가 끝나고 프롬프트. rz 가 보내는 중단 표시(XON + CAN 10개 + 백스페이스 10개)가 화면으로 지나가지만 보이는 글자는 없다 |
| 5 | 40MB 받는 중 Ctrl+C (6MB·9MB·12MB·18MB 지점) | **고치기 전: 결함**(§3.1) → **수정 후 통과** — `.dtpart` 가 지워지고, 취소 안내 뒤 화면에 제어 문자 0개, 프롬프트 복귀, 서버에 `sz` 프로세스가 남지 않음. 명령부터 취소 안내까지 약 1.0~1.5초 |
| 8 | `sz odd.bin empty.bin mid3m.bin /etc/hostname` — 특수 바이트 14바이트(`a\r\nb\n\0\xff\x18\x11\x13 end`), 빈 파일, 3MB, 그리고 이미 받은 이름 | 통과 — 차례로 받고 SHA256 같음. 빈 파일도 생긴다. 두 번째 `hostname` 은 `hostname (2)` |
| 10 | tmux 3.2a 안에서 `sz /etc/hostname`, `rz` | 전송 안 됨(예상대로). **고치기 전: 아무 안내 없이 멈춤** → **수정 후: 안내가 나옴**(§3.3). Ctrl+C 로는 끝나지 않고 **Ctrl+X 5번**으로 `sz` 가 끝난다(종료 코드 128) |
| — (새로) | 시작 표시만 보내고 멈춘 서버 흉내: `printf '**\030B00000000000000\r\212\021'; sleep 30` (`-stall 5`) | **고치기 전: 끝나지 않음** → **수정 후**: 5.5초 뒤 `서버가 5초 동안 응답하지 않아 ZMODEM 전송을 중단했습니다`, 세션은 그대로(§3.2) |

- `sz` 는 시작 표시 앞에 `rz` + CR 을 먼저 찍는다(받는 쪽 프로그램을 띄우라는 옛 관례). 화면에는 `rz` 가 잠깐 보였다가 CR 뒤에 오는 진행 줄(줄 지우기 + 안내)이 덮는다.
- 받은 파일의 수정 시각은 서버 것으로 맞춰진다(`hostname` 의 mtime = 서버 파일의 mtime).

### 2.4 아직 해 보지 못한 것

- **앱 화면**: 진행 줄이 한 줄에서 고쳐 써지는 모습, `rz` 글자가 덮이는 모습, 파일 선택 창, 받은 파일이 실제 다운로드 폴더에 생기는 것 (§H)
- §H 6(전송 중에 글자를 침 — 자동 테스트는 있음), 7(경유 서버를 거쳐 `ssh` 로 넘어간 셸), 9(시리얼 장비), 11·12(설정 끄기·받는 폴더 바꾸기)
- 느린 회선(WAN)에서의 취소: 뒤처리의 "0.5초 조용" 기준이 사내망에서만 확인됐다(§I)

---

## 3. 원인 분석 (Cause)

### 3.1 결함 1 — 받는 중 취소하면 전송 데이터가 화면에 쏟아진다 (D123-2)

**증상**(실제 서버에서 재현): 40MB 를 받다가 8MB 지점에서 Ctrl+C. 엔진은 0.4초 만에 `cancelled` 로 끝났는데, 그 뒤로 **1,794,954바이트의 이진 데이터**가 화면으로 나갔다. 끝자락은 이런 모양이었다:

```
…\x18Q\xb3\xeahS\x8d\xc5\xd0F…g*y\x8f\xd0\xb9\xa8\x8f\xe6\x18S\x05\x18i\xf3\xd0\xc1r\x1b]0;mds@kai-test-vm-02:~\x07\x1b[?2004h[mds@kai-test-vm-02 ~]$
                                                              └─ ZDLE 'i' + CRC 4바이트 ─┘└──────────── 셸 프롬프트 ────────────┘
```

**까닭** — 호출 길:

1. 사용자가 Ctrl+C → `Session.Input` → `cancelZmodem`: 취소 표시를 켜고 서버에 중단 신호(CAN 8개)를 보낸다.
2. 엔진의 `readByte`([internal/zmodem/engine.go](../../internal/zmodem/engine.go))는 연결에서 **묶음 하나를 더 읽은 뒤** 취소 표시를 보고 그 묶음만 버리고 `ErrCancelled` 로 돌아간다.
3. `runZmodem` 이 "취소했습니다" 안내를 내고 돌아가고, 읽기 루프는 **보통 때처럼** 연결을 읽어 화면으로 보낸다.
4. 그런데 중단 신호가 서버에 닿아 `sz` 가 멈추기까지, 그리고 그 전에 이미 보낸 것이 TCP·SSH 창·PTY 를 지나 이 PC 에 닿기까지 **데이터가 계속 온다**. 40MB/2초로 흐르던 중이라 약 1.8MB 였다. 3번의 읽기 루프가 이것을 그대로 화면에 낸다.

명세 [spec/behavior/zmodem.md](../../spec/behavior/zmodem.md) §7 은 처음부터 "취소한 뒤 도착한 전송 데이터는 화면에 내보내지 않고 버린다. 그 뒤의 출력(셸 프롬프트)부터 다시 화면에 나온다"였다. 구현은 2번의 **묶음 하나**만 버렸다.

**자동 테스트가 못 잡은 까닭**: `TestZmodemCancelWithCtrlC` 의 서버 흉내는 이 패키지의 `Send` 다. 중단 신호를 읽자마자 멈추고, 그 뒤로 보내는 것이 없다. "이미 보낸 것이 뒤늦게 도착한다"는 실제 통신의 지연이 없었다.

**고칠 때 까다로운 점**: 흘러오는 데이터의 끝과 프롬프트 사이에 **경계가 없다**. 위 끝자락처럼 `sz` 는 마지막 데이터 조각을 다 쓴 뒤 끝나고(끝났다는 표시를 따로 보내지 않는다) 셸이 곧바로 프롬프트를 찍는다. "한동안 조용하면 끝"으로만 정하면 데이터에 붙어 온 프롬프트까지 버리게 되어, 사용자는 Enter 를 쳐야 프롬프트를 본다.

### 3.2 한계 2 — 서버가 응답하지 않으면 전송 상태로 남는다 (D125)

**까닭**: 엔진은 응답을 기다릴 때 `Stream.Read` 에서 멈춰 있고, 세션의 연결(`Transport.Read`)에는 **읽기 시한을 걸 방법이 없다**(SSH 채널·시리얼·텔넷·로컬 PTY 를 한 인터페이스로 묶은 것이라 `SetReadDeadline` 이 없다). 그래서 2026-11-23 에는 시간 제한을 두지 않았다.

**실제로는 Ctrl+C 로도 곧바로 풀리지 않았다**: `readByte` 는 `Read` 를 부르기 **전**과 **후**에만 취소 표시를 본다. `Read` 에 들어가 있는 동안 Ctrl+C 를 누르면 취소 표시는 켜지지만 엔진은 **서버에서 다음 바이트가 올 때까지** 깨어나지 않는다. 대개는 중단 신호를 받은 `sz`·`rz` 가 끝나며 셸이 프롬프트를 찍어 풀리지만, 서버 쪽이 정말로 멈췄다면(멈춘 프로세스, 받지 못하는 장비) 풀리지 않는다. 그동안 그 세션의 입력은 서버로 가지 않는다(`zActive`).

`TestZmodemCancelWithCtrlC` 의 주석이 이 사정을 적고 있었다 — "붙잡아 둔 보내는 쪽을 풀어 준다: 다음 데이터가 오면 앱이 취소를 알아챈다".

### 3.3 한계 3 — tmux 안에서는 안 되고, 까닭을 알 수 없다 (D125)

**실제 서버에서 본 것**(tmux 3.2a 안에서 `sz /etc/hostname`):

```
…\x1b[1;1Hrz\x1b[H**B00000000000000\x1b[Hhostname15 15301430563 100644 0 1 15j\x1b[?7727h…
            └ 시작 표시에서 0x18 이 빠졌다 ┘└── ZFILE 헤더·파일 정보도 제어 문자가 빠진 채 ──┘
```

`rz` 는 `rz waiting to receive.**B0100000023be50`.

**까닭**: tmux·screen 은 안에서 돈 프로그램의 출력을 **터미널 출력으로 해석해 자기 화면에 그린 뒤**, 바깥 터미널에는 그 화면을 다시 그리는 명령을 보낸다. 바이트를 그대로 지나가게 두지 않는다. 0x18(CAN)은 터미널에서 "진행 중인 제어 시퀀스를 그만둔다"는 뜻의 제어 문자라 화면에 남지 않고 버려진다. 시작 표시의 `ZDLE` 이 빠지므로 `Detector` 는 표시를 보지 못하고, 설령 본다 해도 뒤따르는 이진 데이터가 같은 식으로 깨진다. **클라이언트 쪽에서 고칠 수 없다.**

**그래서 사용자가 겪는 것**: 화면에 `**B00000000000000` 같은 글이 찍히고 아무 일도 일어나지 않는다. `sz` 는 터미널을 날것(raw) 방식으로 바꿔 두므로 **Ctrl+C 는 신호가 되지 않고 그냥 바이트로 들어가** 끝나지 않는다(실제로 확인). lrzsz 가 스스로 포기할 때까지(10초 단위 재시도) 기다리거나 Ctrl+X 를 여러 번 눌러야 한다는 것을 알 길이 없다.

고칠 수 있는 것은 "전송이 안 되는 것"이 아니라 **"까닭과 빠져나오는 방법을 모르는 것"**이다.

---

## 4. 수정 방안과 고른 것 (Fix Plan)

### 4.1 결함 1 — 흘러오는 데이터 버리기 (D123-2)

| 안 | 내용 | 판단 |
|---|---|---|
| A | 취소 뒤 일정 시간(예: 2초) 동안 오는 것을 모두 버린다 | 기각 — 느린 회선에서는 모자라고, 빠른 회선에서는 프롬프트와 그 뒤 출력까지 버린다 |
| B | 조용해질 때까지(0.5초 동안 아무것도 안 옴) 버린다 | 끝은 잘 잡지만 **데이터에 붙어 온 프롬프트를 버린다** |
| **C (고름)** | B + 끝자락에서 **마지막 데이터 조각의 끝**을 찾아 그 뒤만 화면에 낸다 | 조각 안의 0x18 은 항상 바꿔서 오므로 `ZDLE` + `h`~`k` 는 조각 끝에만 나온다 → 경계를 정확히 찾을 수 있다 |
| D | 엔진이 취소 뒤에도 조각 단위로 계속 읽어 버린다 | 기각 — 마지막 조각 뒤의 프롬프트를 "다음 조각의 앞부분"으로 붙들고 다음 출력이 올 때까지 내놓지 않는다 |

세부 결정:

| 항목 | 값 | 까닭 |
|---|---|---|
| 조용하다고 보는 시간 | 0.5초 (`G_durZmodemDrainQuiet`) | 사내망에서 데이터는 끊김 없이 온다. 취소 안내가 그만큼 늦어지므로 길게 잡지 않는다 |
| 버리기를 그만두는 시간 | 20초 (`G_durZmodemDrainMax`) | 그때까지 쉬지 않고 온다면 전송 데이터가 아닐 수 있다. 화면을 영영 막지 않는다 |
| 기억해 두는 끝자락 | 4096바이트 | 조각 하나가 1024바이트(바꿔 쓰면 최대 2배)라 마지막 조각 끝이 반드시 들어온다 |
| 화면에 내는 한도 | 256바이트 | 마지막 조각 끝 뒤가 이보다 길면 조각이 중간에서 잘린 것(이진 데이터 + 프롬프트)이라 가를 수 없다 → 버린다. 프롬프트는 대개 100바이트 안쪽 |
| 조각 끝이 하나도 없을 때 | 버린 것이 4096바이트 이하면 모두 화면에 | 데이터가 오기 전에 중단한 경우다(헤더·셸 출력뿐) |
| 언제 하는가 | 받는 쪽에서 취소·실패·상대 중단·응답 없음으로 끝났을 때 | 보내는 쪽은 서버가 보내는 것이 짧은 응답뿐이라 하지 않는다 |
| 그동안의 입력 | 서버로 보내지 않는다(`zActive` 유지) | 뒤처리 중에 친 글자가 죽어 가는 `sz` 에 섞이지 않게 |
| 안내 순서 | 뒤처리 → 취소 안내 → 프롬프트 | 안내가 데이터에 묻히지 않게 |

읽기에 시한을 걸 수 없는 문제는 **읽기를 고루틴에 걸어 두고 결과를 기다리는 것**으로 푼다. 조용해졌을 때 걸려 있는 읽기 하나는 버릴 수 없으므로(그 읽기가 다음 출력을 받는다) **읽기 루프가 이어 받는다**.

### 4.2 한계 2 — 응답 시한과 즉시 취소 (D125)

4.1 의 "고루틴에 건 읽기"를 전송 내내 쓰도록 넓혔다: 엔진은 연결을 바로 읽지 않고 `zmodemLink` 를 거친다.

| 항목 | 값 | 까닭 |
|---|---|---|
| 응답 시한 | 60초 (`G_durZmodemStall`) | lrzsz 자신의 재시도가 10초 단위다. 느린 디스크·NFS 에서 서버가 잠깐 조용한 것을 끊지 않을 만큼 넉넉하게 |
| 무엇을 재는가 | **엔진이 읽기를 기다리는 한 번** 동안 한 바이트도 오지 않은 시간 | 파일 선택 창을 띄워 둔 동안은 엔진이 읽지 않으므로 시한에 들어가지 않는다 |
| 시한에 걸리면 | 서버에 중단 신호 → 전송만 끝냄 → 안내. **연결은 끊지 않는다** | 멈춘 것은 대개 `sz` 하나이고 셸·연결은 살아 있다. 끊으면 자동 재접속이 돌아 작업 중인 셸을 잃는다 |
| 취소를 살피는 간격 | 0.1초 (`g_cdurZmodemPoll`) | Ctrl+C 가 눌린 즉시처럼 느껴지는 값 |
| 취소를 엔진에 알리는 방법 | `Read` 가 `(0, nil)` 로 돌아간다 | 엔진은 `Read` 뒤에 취소 표시를 보므로 엔진 쪽을 고치지 않아도 된다 |

검토하고 버린 안: 시한에 걸리면 연결을 끊는다(위 까닭으로 기각), 엔진에 `context` 를 넣는다(엔진·시험을 크게 고쳐야 하고 얻는 것이 같다).

### 4.3 한계 3 — tmux 안에서는 안내 (D125)

| 안 | 내용 | 판단 |
|---|---|---|
| A | 아무것도 안 한다(명세의 "알려진 한계"로만 둔다) | 기각 — 사용자는 명세를 보지 않는다 |
| **B (고름)** | `ZDLE` 이 빠진 시작 표시를 알아보고 **안내만** 낸다 | 서버로 아무것도 보내지 않으므로 잘못 알아봐도 해가 없다 |
| C | B + 중단 신호(CAN 8개)를 자동으로 보내 `sz` 를 끝낸다 | 기각 — 그 글이 실행 중인 `sz` 의 것이 아닐 수 있다(로그 파일을 `cat`·`vim` 으로 보는 중). 그러면 셸·편집기에 Ctrl+X 를 8번 넣게 된다(vim 에서는 숫자를 줄이는 키다) |

세부 결정:

| 항목 | 값 | 까닭 |
|---|---|---|
| 알아보는 모양 | `**B0` + `0` 또는 `1` + 16진 **소문자** 12자 (17글자) | lrzsz 가 보내는 것은 `**B00000000000000`, `**B0100000023be50`. 우연히 맞기 어려울 만큼 길다. 진짜 표시는 `**` 뒤에 0x18 이 있어 맞지 않는다 |
| 안내 문구 | `ZMODEM 시작 표시가 깨져서 왔습니다 — tmux·screen 안에서는 전송할 수 없습니다. Ctrl+X 를 5번 눌러 끝내고 tmux 밖에서 다시 실행하세요` | 끝내는 방법은 실제 서버에서 확인했다(Ctrl+C ✗, Ctrl+X 5번 ✓) |
| 다시 내지 않는 시간 | 10초 (`G_durZmodemMangledNotice`) | `sz` 가 응답을 못 받으면 표시를 되풀이해 보낸다 |
| 설정이 꺼져 있으면 | 살피지 않는다 | `zmodemEnabled` 를 끈 사용자에게 ZMODEM 안내를 내지 않는다 |
| 서버가 보낸 글 | 그대로 화면에 낸다 | 출력을 바꾸지 않는다. 안내는 그 묶음 뒤에 한 줄 |

### 4.4 영향 범위

- **전송이 잘 끝나는 길은 달라지지 않는다**: 엔진이 연결을 `zmodemLink` 로 읽는 것만 다르고(읽은 바이트는 같다), 실제 서버에서 받기·보내기 해시를 다시 확인했다(§G.3).
- 달라지는 것: 받는 중 취소 안내가 **최대 0.5초 늦게** 나온다(뒤처리). 응답 없는 서버에서 **60초 뒤 전송이 끝난다**. tmux 안에서 **안내 한 줄이 더 나온다**.
- 화면·설정·데이터 형식·IPC 는 그대로다. 문구 2개가 늘었다(10개 언어).

---

## A. 바뀐 파일 한눈에

| 파일 | 종류 | 결정 | 바뀐 것 |
|---|---|---|---|
| `internal/zmodem/engine.go` | 수정 (+6 −1) | D123-2 | 엔진 상태에 `bDataCrc32`·`bDropped`. 취소 뒤 버리는 묶음을 `bDropped` 에 남긴다 |
| `internal/zmodem/receive.go` | 수정 (+7) | D123-2 | `Result.DataCrc32`·`Result.Dropped`. 데이터 프레임의 CRC 종류를 기억 |
| `internal/zmodem/frame.go` | 수정 (+35) | D123-2 | `TrailingOutput` — 끝자락에서 마지막 데이터 조각 뒤의 바이트를 가른다 |
| `internal/zmodem/detect.go` | 수정 (+49) | D125 | `MangledDetector` — `ZDLE` 이 빠진 시작 표시를 찾는다 |
| `internal/session/zmodem.go` | 수정 (+171 −5) | D123-2·D125 | `zmodemLink`(시한·취소를 볼 수 있는 읽기), `drainAborted`·`drainedOutput`(뒤처리), `runZmodem` 이 걸어 둔 읽기를 돌려준다, 응답 없음 처리 |
| `internal/session/session.go` | 수정 (+25 −4) | D123-2·D125 | `readLoop` — 걸어 둔 읽기를 이어 받는다, 깨진 시작 표시 안내 |
| `internal/zmodem/zmodem_test.go` | 수정 (+74) | D123-2·D125 | `TestTrailingOutput`, `TestMangledDetector` |
| `internal/session/zmodem_test.go` | 수정 (+162) | D123-2·D125 | 시험 5개(§C) |
| `tools/zmodemcheck/main.go` | **새 파일** (319줄) | D123-2 | 실제 서버 검증 도구 |
| `locales/{ko,en,de,es,fr,ja,pt,ru,tw,zh}.json` | 수정 (각 +2) | D125 | `be.zm.stalled`, `be.zm.muxed` |
| `spec/behavior/zmodem.md` | 수정 | D123-2·D125 | §3 tmux 안내, §7 뒤처리·응답 시한·즉시 취소, §10 수용 시나리오 Z9~Z12 |
| `docs/decisions.md` | 수정 | | D123-2, D125 |
| `.claude/known-issues.md` | 수정 | | KI-012 → resolved |
| `CHANGELOG.md`, `CLAUDE.md`, `docs/import-plans/2026-11-23-zmodem.md` | 수정 | | 미배포 절의 수정·확인한 것, 인덱스, §H-1 |

프런트엔드(`frontend/`)는 바뀌지 않았다. 문구는 백엔드가 `i18n.T` 로 읽는다.

---

## B. 코드

### `internal/zmodem/engine.go` — 수정 (D123-2)

엔진이 두 가지를 더 기억한다: 상대가 보낸 데이터 프레임의 CRC 종류(끝자락에서 CRC 길이를 알아야 한다), 취소 뒤 읽어서 버린 마지막 묶음(그 끝에 프롬프트가 붙어 왔을 수 있다).

```diff
--- a/internal/zmodem/engine.go
+++ b/internal/zmodem/engine.go
@@ -46,9 +46,12 @@
 	iErrors int
 	// bResync 는 오류 뒤 ZRPOS 를 보내고 상대의 새 ZDATA 를 기다리는 중이라는 뜻이다. 그동안 흘러오는 것은
 	// 이미 버리기로 한 데이터라, 그 안에서 헤더처럼 보이다 만 것들은 오류로 세지 않는다(ZRPOS 를 보낼 때 한 번 셌다).
 	bResync bool
-	arrRead [32 * 1024]byte
+	// bDataCrc32 는 상대가 마지막으로 보낸 데이터 프레임이 CRC-32 였는지다 (중단 뒤 끝자락을 가를 때 쓴다).
+	bDataCrc32 bool
+	bDropped   []byte // 취소한 뒤 읽어서 버린 마지막 묶음
+	arrRead    [32 * 1024]byte
 }
 
 func newEngine(stream Stream, bInitial []byte, opts Options) *engine {
 	return &engine{stream: stream, opts: opts, bIn: append([]byte(nil), bInitial...), bKeep: true}
@@ -84,8 +87,10 @@
 			return 0, &StreamError{Err: errRead}
 		}
 		if pEngine.cancelled() {
 			// 취소한 뒤에 도착한 것은 전송 중이던 데이터다: 화면으로 되돌리지 않고 버린다.
+			// 다만 그 끝에 셸 출력이 붙어 왔을 수 있어, 부른 쪽이 가를 수 있게 끝자락은 남겨 둔다 (Result.Dropped).
+			pEngine.bDropped = append([]byte(nil), pEngine.bIn[pEngine.iIn:]...)
 			pEngine.bIn, pEngine.iIn = pEngine.bIn[:0], 0
 			return 0, ErrCancelled
 		}
 	}
```

### `internal/zmodem/receive.go` — 수정 (D123-2)

```diff
--- a/internal/zmodem/receive.go
+++ b/internal/zmodem/receive.go
@@ -33,8 +33,12 @@
 	Files    int    // 다 주고받은 파일 수
 	Skipped  int    // 건너뛴 파일 수
 	Bytes    int64  // 주고받은 바이트 수
 	Leftover []byte // ZMODEM 이 아닌 것으로 남은 바이트 (화면 출력으로 되돌린다)
+	// DataCrc32 는 받던 데이터 프레임이 CRC-32 였는지다. 받다가 중단했을 때 TrailingOutput 에 넘긴다.
+	DataCrc32 bool
+	// Dropped 는 취소한 뒤 읽어서 버린 마지막 묶음이다 (화면으로 되돌리지 않는다. 끝자락을 가를 때만 쓴다).
+	Dropped []byte
 }
 
 // Receive 는 서버가 보내는 파일들을 받는다. bInitial 은 Detector 가 돌려준 바이트(시작 표시 포함)다.
 // 시작 표시가 실제 헤더가 아니면 ErrNotZmodem 과 함께 읽은 바이트를 Leftover 로 돌려준다.
@@ -51,8 +55,10 @@
 		return result, errFirst
 	}
 	errRun := pEngine.receiveLoop(sink, &result)
 	result.Leftover = pEngine.leftover()
+	result.DataCrc32 = pEngine.bDataCrc32
+	result.Dropped = pEngine.bDropped
 	if errRun != nil {
 		if _, bStream := errRun.(*StreamError); !bStream && errRun != ErrRemoteCancelled {
 			_ = stream.Write(AbortSequence())
 		}
@@ -211,8 +217,9 @@
 				}
 				continue
 			}
 			pEngine.bResync = false
+			pEngine.bDataCrc32 = hdr.bFormat == g_cbZbin32
 			bFrameDone := false
 			for !bFrameDone {
 				bData, bEnd, bOk, errData := pEngine.readSubpacket(hdr.bFormat == g_cbZbin32)
 				if errData != nil {
```

### `internal/zmodem/frame.go` — 수정 (D123-2)

파일 끝에 `TrailingOutput` 을 더했다. 끝자락을 **뒤에서부터** 훑어 마지막 `ZDLE` + 끝 표시를 찾고, 그 뒤의 CRC(2 또는 4바이트, 바꿔서 온 바이트는 두 바이트로 센다)를 건너뛴 나머지를 돌려준다.

```diff
--- a/internal/zmodem/frame.go
+++ b/internal/zmodem/frame.go
@@ -193,4 +193,39 @@
 func AbortSequence() []byte {
 	return []byte{0x18, 0x18, 0x18, 0x18, 0x18, 0x18, 0x18, 0x18, 8, 8, 8, 8, 8, 8, 8, 8}
 }
 
+// TrailingOutput 은 받다가 중단한 전송의 끝자락(bTail)에서, 마지막 데이터 조각 뒤에 붙어 온 바이트를 돌려준다.
+// 중단을 알린 뒤에도 상대가 이미 보낸 데이터가 한동안 흘러오고, 그 끝에 셸 프롬프트 같은 화면 출력이 바로 붙는다.
+// 데이터 조각은 ZDLE + 끝 표시(ZCRCE·ZCRCG·ZCRCQ·ZCRCW) + CRC 로 끝나고, 조각 안의 ZDLE 은 항상 바꿔서 오므로
+// 이 두 바이트는 조각 끝에만 나온다. 끝 표시가 없으면 bFound 가 false 다(끝자락에 데이터 조각이 없었다).
+// CRC 가 끝자락에서 잘렸으면 빈 값을 돌려준다.
+func TrailingOutput(bTail []byte, bCrc32 bool) (bAfter []byte, bFound bool) {
+	iEnd := -1
+	for iIndex := len(bTail) - 2; iIndex >= 0; iIndex-- {
+		if bTail[iIndex] != g_cbZdle {
+			continue
+		}
+		if bNext := bTail[iIndex+1]; bNext >= g_cbZcrce && bNext <= g_cbZcrcw {
+			iEnd = iIndex + 2
+			break
+		}
+	}
+	if iEnd < 0 {
+		return nil, false
+	}
+	iCrc := 2
+	if bCrc32 {
+		iCrc = 4
+	}
+	for ; iCrc > 0; iCrc-- {
+		if iEnd < len(bTail) && bTail[iEnd] == g_cbZdle {
+			iEnd++ // 바꿔서 온 CRC 바이트는 두 바이트다
+		}
+		iEnd++
+		if iEnd > len(bTail) {
+			return nil, true
+		}
+	}
+	return bTail[iEnd:], true
+}
+
```

### `internal/zmodem/detect.go` — 수정 (D125)

파일 끝에 `MangledDetector` 를 더했다. `Detector` 와 같은 방식(맞춘 글자 수를 기억해 묶음 경계에 걸쳐도 찾는다)이고, 서버로 보낼 것이 없으므로 찾았는지만 돌려준다.

```diff
--- a/internal/zmodem/detect.go
+++ b/internal/zmodem/detect.go
@@ -60,4 +60,53 @@
 func (pDetector *Detector) Reset() {
 	pDetector.iMatched = 0
 }
 
+// g_ciMangledLength 는 ZDLE 이 빠진 시작 표시의 길이다: "**B0" + 종류 한 글자 + 16진 12자(값 8 + CRC 4).
+const g_ciMangledLength = 17
+
+// MangledDetector 는 ZDLE(0x18)이 빠진 채 온 시작 표시를 찾는다: `**B00000000000000`(sz), `**B0100000023be50`(rz) 같은 글.
+// tmux·screen 은 안에서 돈 프로그램의 출력을 해석해 다시 그리면서 제어 문자인 ZDLE 을 버린다. 그러면 Detector 는
+// 시작 표시를 보지 못하고 전송도 될 수 없다(뒤따르는 이진 데이터도 같은 식으로 깨진다). 이 모양을 알아보면
+// 사용자에게 까닭을 알려 줄 수 있다. 표시가 출력 묶음 경계에 걸쳐 와도 찾는다.
+type MangledDetector struct {
+	iMatched int // 지금까지 맞춘 글자 수
+}
+
+// Feed 는 출력 묶음 하나를 살펴, ZDLE 이 빠진 시작 표시가 이 묶음에서 끝났으면 true 를 돌려준다.
+func (pDetector *MangledDetector) Feed(bData []byte) bool {
+	bFound := false
+	for _, bOne := range bData {
+		bMatch := false
+		switch iAt := pDetector.iMatched; {
+		case iAt < 2:
+			bMatch = bOne == '*'
+		case iAt == 2:
+			bMatch = bOne == 'B'
+		case iAt == 3:
+			bMatch = bOne == '0'
+		case iAt == 4:
+			bMatch = bOne == '0' || bOne == '1'
+		default:
+			bMatch = bOne >= '0' && bOne <= '9' || bOne >= 'a' && bOne <= 'f'
+		}
+		switch {
+		case bMatch:
+			pDetector.iMatched++
+			if pDetector.iMatched == g_ciMangledLength {
+				bFound = true
+				pDetector.iMatched = 0
+			}
+		case bOne == '*':
+			// "***" 처럼 별이 이어지면 마지막 두 개가 표시의 앞부분일 수 있다.
+			if pDetector.iMatched >= 2 {
+				pDetector.iMatched = 2
+			} else {
+				pDetector.iMatched = 1
+			}
+		default:
+			pDetector.iMatched = 0
+		}
+	}
+	return bFound
+}
+
```

### `internal/session/zmodem.go` — 수정 (D123-2·D125)

이 문서의 중심이다. 바뀐 것 네 가지:

1. `runZmodem` — 엔진에 `conn` 대신 `zmodemLink` 를 넘긴다. 응답 없음(`errZmodemStalled`)이면 중단 신호를 보내고 끊김이 아닌 것으로 바꾼다. 받다가 중단했으면 뒤처리를 한다. 걸어 둔 읽기를 세 번째 값으로 돌려준다.
2. `zmodemLink` — `zmodem.Stream`. 읽기를 고루틴에 걸고(`wait`) 0.1초마다 취소를 살피며 60초 시한을 잰다.
3. `drainAborted` — 조용해질 때까지 읽어 버리고 끝자락을 기억한다.
4. `drainedOutput` — 끝자락에서 화면으로 되돌릴 부분을 고른다.

패치:

```diff
--- a/internal/session/zmodem.go
+++ b/internal/session/zmodem.go
@@ -54,22 +54,25 @@
 }
 
 // runZmodem 은 전송 하나를 끝까지 돌린다. 읽기 루프 고루틴에서 부른다.
 // 돌려주는 바이트는 ZMODEM 이 아닌 것으로 남은 출력(화면으로 보낸다), 오류는 연결이 끊겼을 때만 nil 이 아니다.
-func (pSession *Session) runZmodem(conn Transport, kind zmodem.Kind, bEngine []byte) ([]byte, error) {
+// 채널은 nil 이 아니면 이미 걸어 둔 읽기 하나다: 읽기 루프가 conn.Read 대신 그 결과를 먼저 받아야 한다.
+func (pSession *Session) runZmodem(conn Transport, kind zmodem.Kind, bEngine []byte) ([]byte, chan readResult, error) {
 	pSession.zCancel.Store(false)
 	pSession.zActive.Store(true)
 	defer pSession.zActive.Store(false)
 
 	pHooks := pSession.cfg.Zmodem
 	bSending := kind == zmodem.KindRemoteAccepts
 	pView := &zmodemView{pSession: pSession, bSending: bSending}
 	opts := zmodem.Options{Cancelled: pSession.zCancel.Load, Progress: pView.onProgress}
+	// 엔진은 연결을 바로 읽지 않고 zmodemLink 를 거친다: 서버가 응답 없이 멈춰도 시한이 지나면 끝나고, Ctrl+C 가 곧바로 듣는다.
+	pLink := &zmodemLink{conn: conn, fnCancel: pSession.zCancel.Load}
 
 	var result zmodem.Result
 	var errRun error
 	if bSending {
-		result, errRun = zmodem.Send(conn, bEngine, func() ([]zmodem.FileSource, error) {
+		result, errRun = zmodem.Send(pLink, bEngine, func() ([]zmodem.FileSource, error) {
 			pSession.writeOutput(noticeLine(i18n.T("be.zm.pick")))
 			if pHooks.PickFiles == nil {
 				return nil, nil
 			}
@@ -84,19 +87,42 @@
 		if pHooks.DownloadDir != nil {
 			szDir = pHooks.DownloadDir()
 		}
 		pView.szDir = szDir
-		result, errRun = zmodem.Receive(conn, bEngine, &zmodemSink{szDir: szDir, pView: pView}, opts)
-	}
-
+		result, errRun = zmodem.Receive(pLink, bEngine, &zmodemSink{szDir: szDir, pView: pView}, opts)
+	}
+
+	if len(pLink.bRest) > 0 {
+		// 연결에서 읽었지만 엔진에 넘기지 못한 바이트: 엔진이 남긴 것 뒤에 잇는다.
+		result.Leftover = append(result.Leftover, pLink.bRest...)
+	}
+	if errors.Is(errRun, errZmodemStalled) {
+		// 서버가 시한 동안 아무것도 보내지 않았다: 연결은 살아 있으므로 끊김으로 보지 않고, 중단을 알린 뒤 전송만 끝낸다.
+		_ = conn.Write(zmodem.AbortSequence())
+		errRun = errZmodemStalled
+	}
 	var errStream *zmodem.StreamError
+	// 받다가 중단했으면(취소·실패) 상대가 이미 보낸 데이터가 한동안 더 흘러온다: 화면에 내보내지 않고 버린다 (zmodem.md §7).
+	var bTail []byte
+	if !bSending && errRun != nil && !errors.Is(errRun, zmodem.ErrNotZmodem) && !errors.As(errRun, &errStream) {
+		// 엔진이 읽어 두고 쓰지 않은 것(Leftover)과 취소 뒤 버린 묶음(Dropped)도 전송 데이터다: 끝자락에 넣어 함께 가른다.
+		bSeen := append(append([]byte(nil), result.Dropped...), result.Leftover...)
+		result.Leftover = nil
+		var errDrain error
+		if bTail, errDrain = drainAborted(pLink, bSeen, result.DataCrc32); errDrain != nil {
+			errRun = &zmodem.StreamError{Err: errDrain}
+		}
+	}
 	switch {
 	case errRun == nil:
 		pSession.writeOutput(noticeLine(i18n.T("be.zm.done", "n", result.Files, "size", formatBytes(result.Bytes))))
 	case errors.Is(errRun, zmodem.ErrNotZmodem):
 		// 시작 표시처럼 보였을 뿐이다: 아무 안내 없이 읽은 바이트를 화면으로 되돌린다.
 	case errors.As(errRun, &errStream):
-		return result.Leftover, errStream.Err
+		return result.Leftover, nil, errStream.Err
+	case errors.Is(errRun, errZmodemStalled):
+		pView.endLine()
+		pSession.writeOutput(noticeLine(i18n.T("be.zm.stalled", "sec", int(G_durZmodemStall/time.Second))))
 	case errors.Is(errRun, zmodem.ErrCancelled):
 		pView.endLine()
 		pSession.writeOutput(noticeLine(i18n.T("be.zm.cancelled")))
 	case errors.Is(errRun, zmodem.ErrRemoteCancelled):
@@ -105,9 +131,149 @@
 	default:
 		pView.endLine()
 		pSession.writeOutput(noticeLine(i18n.T("be.zm.failed", "error", errRun)))
 	}
-	return result.Leftover, nil
+	// 걸어 둔 읽기가 남아 있으면(취소·시한으로 끝났을 때) 읽기 루프가 이어 받는다.
+	return append(result.Leftover, bTail...), pLink.chPending, nil
+}
+
+// readResult 는 따로 걸어 둔 읽기 하나의 결과다.
+type readResult struct {
+	bData []byte
+	err   error
+}
+
+// ZMODEM 전송의 시간 값. 시험에서 줄일 수 있게 변수다.
+var (
+	// G_durZmodemStall 동안 서버에서 아무것도 오지 않으면(엔진이 응답을 기다리는 중에) 전송을 중단한다 (zmodem.md §7).
+	G_durZmodemStall = 60 * time.Second
+	// G_durZmodemMangledNotice 안에는 "시작 표시가 깨져서 왔다"는 안내를 다시 내지 않는다 (sz·rz 가 표시를 되풀이해 보낸다).
+	G_durZmodemMangledNotice = 10 * time.Second
+	// G_durZmodemDrainQuiet 동안 아무것도 오지 않으면 흘러오던 데이터가 끝난 것으로 본다.
+	G_durZmodemDrainQuiet = 500 * time.Millisecond
+	// G_durZmodemDrainMax 를 넘도록 계속 오면 버리기를 그만둔다 (그 뒤의 출력은 화면에 나온다).
+	G_durZmodemDrainMax = 20 * time.Second
+)
+
+const (
+	g_cdurZmodemPoll    = 100 * time.Millisecond // 읽기를 기다리는 동안 취소를 살피는 간격
+	g_ciZmodemDrainTail = 4096                   // 끝자락으로 기억해 두는 바이트 수
+	g_ciZmodemDrainShow = 256                    // 마지막 데이터 조각 뒤에 붙어 온 것이 이보다 길면 화면 출력으로 보지 않는다
+)
+
+// errZmodemStalled 는 서버가 G_durZmodemStall 동안 응답하지 않았다는 오류다.
+var errZmodemStalled = errors.New("zmodem: remote not responding")
+
+// zmodemLink 는 전송 동안 엔진이 쓰는 연결(zmodem.Stream)이다.
+// 연결의 읽기(Transport.Read)에는 시한을 걸 수 없으므로 읽기를 고루틴에 걸어 두고 결과를 기다린다.
+// 기다리는 동안 취소를 살피고, 너무 오래 조용하면 errZmodemStalled 로 끝낸다. 전송이 끝났을 때 걸려 있는
+// 읽기(chPending)는 읽기 루프가 이어 받는다 — 그 읽기가 받는 것은 전송 뒤의 화면 출력이다.
+// 읽기 루프 고루틴 하나만 쓴다(잠금 없음).
+type zmodemLink struct {
+	conn      Transport
+	fnCancel  func() bool
+	chPending chan readResult // 걸어 둔 읽기. 없으면 nil
+	bRest     []byte          // 읽었지만 아직 넘기지 않은 바이트
+}
+
+func (pLink *zmodemLink) Write(bData []byte) error { return pLink.conn.Write(bData) }
+
+// wait 는 읽기 하나의 결과를 durWait 동안 기다린다. 시간 안에 오지 않으면 false 이고 읽기는 걸린 채 남는다.
+func (pLink *zmodemLink) wait(durWait time.Duration) (readResult, bool) {
+	if pLink.chPending == nil {
+		chRead := make(chan readResult, 1)
+		conn := pLink.conn
+		go func() {
+			bBuf := make([]byte, 32*1024)
+			nRead, errRead := conn.Read(bBuf)
+			chRead <- readResult{bData: bBuf[:nRead], err: errRead}
+		}()
+		pLink.chPending = chRead
+	}
+	timer := time.NewTimer(durWait)
+	defer timer.Stop()
+	select {
+	case got := <-pLink.chPending:
+		pLink.chPending = nil
+		return got, true
+	case <-timer.C:
+		return readResult{}, false
+	}
+}
+
+// Read 는 서버에서 온 바이트를 돌려준다. 취소했으면 (0, nil) 로 곧바로 돌아가 엔진이 취소를 알아채게 한다.
+func (pLink *zmodemLink) Read(bBuf []byte) (int, error) {
+	if len(pLink.bRest) > 0 {
+		nCopy := copy(bBuf, pLink.bRest)
+		pLink.bRest = pLink.bRest[nCopy:]
+		return nCopy, nil
+	}
+	tmStall := time.Now().Add(G_durZmodemStall)
+	for {
+		if pLink.fnCancel != nil && pLink.fnCancel() {
+			return 0, nil
+		}
+		got, bGot := pLink.wait(g_cdurZmodemPoll)
+		if bGot {
+			if len(got.bData) == 0 {
+				return 0, got.err
+			}
+			// 데이터와 오류가 함께 왔으면 데이터부터 넘긴다(오류는 다음 읽기에서 다시 난다).
+			nCopy := copy(bBuf, got.bData)
+			pLink.bRest = got.bData[nCopy:]
+			return nCopy, nil
+		}
+		if time.Now().After(tmStall) {
+			return 0, errZmodemStalled
+		}
+	}
+}
+
+// drainAborted 는 받기를 중단한 뒤 흘러오는 전송 데이터를 읽어 버린다.
+// G_durZmodemDrainQuiet 동안 조용해질 때까지 읽는다. 조용해졌을 때 걸려 있는 읽기는 pLink.chPending 에 남아
+// 읽기 루프가 이어 받는다(그 읽기가 받는 것은 중단 뒤의 새 출력이다).
+// 돌려주는 바이트는 마지막 데이터 조각 뒤에 바로 붙어 온 화면 출력(셸 프롬프트)이다.
+// bSeen 은 엔진이 이미 읽어 버린 마지막 부분이다(끝자락의 앞부분이 된다).
+func drainAborted(pLink *zmodemLink, bSeen []byte, bCrc32 bool) ([]byte, error) {
+	tmGiveUp := time.Now().Add(G_durZmodemDrainMax)
+	bTail := bSeen
+	if len(bTail) > g_ciZmodemDrainTail {
+		bTail = bTail[len(bTail)-g_ciZmodemDrainTail:]
+	}
+	bTail = append([]byte(nil), bTail...)
+	iTotal := len(bSeen)
+	for {
+		got, bGot := pLink.wait(G_durZmodemDrainQuiet)
+		if !bGot {
+			return drainedOutput(bTail, iTotal, bCrc32), nil
+		}
+		if got.err != nil && len(got.bData) == 0 {
+			return nil, got.err
+		}
+		iTotal += len(got.bData)
+		bTail = append(bTail, got.bData...)
+		if len(bTail) > g_ciZmodemDrainTail {
+			bTail = append([]byte(nil), bTail[len(bTail)-g_ciZmodemDrainTail:]...)
+		}
+		if time.Now().After(tmGiveUp) {
+			return nil, nil
+		}
+	}
+}
+
+// drainedOutput 은 버린 바이트의 끝자락에서 화면으로 되돌릴 부분을 고른다.
+func drainedOutput(bTail []byte, iTotal int, bCrc32 bool) []byte {
+	bAfter, bFound := zmodem.TrailingOutput(bTail, bCrc32)
+	if bFound {
+		if len(bAfter) > g_ciZmodemDrainShow {
+			return nil // 데이터 조각이 중간에서 잘렸다: 화면 출력과 가를 수 없다
+		}
+		return bAfter
+	}
+	if iTotal > len(bTail) {
+		return nil // 끝자락이 통째로 조각 하나의 안쪽이다
+	}
+	return bTail // 데이터 조각이 하나도 없었다: 헤더·셸 출력뿐이다
 }
 
 // noticeLine 은 터미널 안내 한 줄(청록 ●)을 만든다.
 func noticeLine(szText string) string {
```

고친 뒤의 해당 부분 전체(`runZmodem` 부터 `drainedOutput` 까지):

```go
// runZmodem 은 전송 하나를 끝까지 돌린다. 읽기 루프 고루틴에서 부른다.
// 돌려주는 바이트는 ZMODEM 이 아닌 것으로 남은 출력(화면으로 보낸다), 오류는 연결이 끊겼을 때만 nil 이 아니다.
// 채널은 nil 이 아니면 이미 걸어 둔 읽기 하나다: 읽기 루프가 conn.Read 대신 그 결과를 먼저 받아야 한다.
func (pSession *Session) runZmodem(conn Transport, kind zmodem.Kind, bEngine []byte) ([]byte, chan readResult, error) {
	pSession.zCancel.Store(false)
	pSession.zActive.Store(true)
	defer pSession.zActive.Store(false)

	pHooks := pSession.cfg.Zmodem
	bSending := kind == zmodem.KindRemoteAccepts
	pView := &zmodemView{pSession: pSession, bSending: bSending}
	opts := zmodem.Options{Cancelled: pSession.zCancel.Load, Progress: pView.onProgress}
	// 엔진은 연결을 바로 읽지 않고 zmodemLink 를 거친다: 서버가 응답 없이 멈춰도 시한이 지나면 끝나고, Ctrl+C 가 곧바로 듣는다.
	pLink := &zmodemLink{conn: conn, fnCancel: pSession.zCancel.Load}

	var result zmodem.Result
	var errRun error
	if bSending {
		result, errRun = zmodem.Send(pLink, bEngine, func() ([]zmodem.FileSource, error) {
			pSession.writeOutput(noticeLine(i18n.T("be.zm.pick")))
			if pHooks.PickFiles == nil {
				return nil, nil
			}
			slPaths, errPick := pHooks.PickFiles()
			if errPick != nil {
				return nil, errPick
			}
			return zmodemSources(slPaths), nil
		}, opts)
	} else {
		szDir := ""
		if pHooks.DownloadDir != nil {
			szDir = pHooks.DownloadDir()
		}
		pView.szDir = szDir
		result, errRun = zmodem.Receive(pLink, bEngine, &zmodemSink{szDir: szDir, pView: pView}, opts)
	}

	if len(pLink.bRest) > 0 {
		// 연결에서 읽었지만 엔진에 넘기지 못한 바이트: 엔진이 남긴 것 뒤에 잇는다.
		result.Leftover = append(result.Leftover, pLink.bRest...)
	}
	if errors.Is(errRun, errZmodemStalled) {
		// 서버가 시한 동안 아무것도 보내지 않았다: 연결은 살아 있으므로 끊김으로 보지 않고, 중단을 알린 뒤 전송만 끝낸다.
		_ = conn.Write(zmodem.AbortSequence())
		errRun = errZmodemStalled
	}
	var errStream *zmodem.StreamError
	// 받다가 중단했으면(취소·실패) 상대가 이미 보낸 데이터가 한동안 더 흘러온다: 화면에 내보내지 않고 버린다 (zmodem.md §7).
	var bTail []byte
	if !bSending && errRun != nil && !errors.Is(errRun, zmodem.ErrNotZmodem) && !errors.As(errRun, &errStream) {
		// 엔진이 읽어 두고 쓰지 않은 것(Leftover)과 취소 뒤 버린 묶음(Dropped)도 전송 데이터다: 끝자락에 넣어 함께 가른다.
		bSeen := append(append([]byte(nil), result.Dropped...), result.Leftover...)
		result.Leftover = nil
		var errDrain error
		if bTail, errDrain = drainAborted(pLink, bSeen, result.DataCrc32); errDrain != nil {
			errRun = &zmodem.StreamError{Err: errDrain}
		}
	}
	switch {
	case errRun == nil:
		pSession.writeOutput(noticeLine(i18n.T("be.zm.done", "n", result.Files, "size", formatBytes(result.Bytes))))
	case errors.Is(errRun, zmodem.ErrNotZmodem):
		// 시작 표시처럼 보였을 뿐이다: 아무 안내 없이 읽은 바이트를 화면으로 되돌린다.
	case errors.As(errRun, &errStream):
		return result.Leftover, nil, errStream.Err
	case errors.Is(errRun, errZmodemStalled):
		pView.endLine()
		pSession.writeOutput(noticeLine(i18n.T("be.zm.stalled", "sec", int(G_durZmodemStall/time.Second))))
	case errors.Is(errRun, zmodem.ErrCancelled):
		pView.endLine()
		pSession.writeOutput(noticeLine(i18n.T("be.zm.cancelled")))
	case errors.Is(errRun, zmodem.ErrRemoteCancelled):
		pView.endLine()
		pSession.writeOutput(noticeLine(i18n.T("be.zm.remoteCancelled")))
	default:
		pView.endLine()
		pSession.writeOutput(noticeLine(i18n.T("be.zm.failed", "error", errRun)))
	}
	// 걸어 둔 읽기가 남아 있으면(취소·시한으로 끝났을 때) 읽기 루프가 이어 받는다.
	return append(result.Leftover, bTail...), pLink.chPending, nil
}

// readResult 는 따로 걸어 둔 읽기 하나의 결과다.
type readResult struct {
	bData []byte
	err   error
}

// ZMODEM 전송의 시간 값. 시험에서 줄일 수 있게 변수다.
var (
	// G_durZmodemStall 동안 서버에서 아무것도 오지 않으면(엔진이 응답을 기다리는 중에) 전송을 중단한다 (zmodem.md §7).
	G_durZmodemStall = 60 * time.Second
	// G_durZmodemMangledNotice 안에는 "시작 표시가 깨져서 왔다"는 안내를 다시 내지 않는다 (sz·rz 가 표시를 되풀이해 보낸다).
	G_durZmodemMangledNotice = 10 * time.Second
	// G_durZmodemDrainQuiet 동안 아무것도 오지 않으면 흘러오던 데이터가 끝난 것으로 본다.
	G_durZmodemDrainQuiet = 500 * time.Millisecond
	// G_durZmodemDrainMax 를 넘도록 계속 오면 버리기를 그만둔다 (그 뒤의 출력은 화면에 나온다).
	G_durZmodemDrainMax = 20 * time.Second
)

const (
	g_cdurZmodemPoll    = 100 * time.Millisecond // 읽기를 기다리는 동안 취소를 살피는 간격
	g_ciZmodemDrainTail = 4096                   // 끝자락으로 기억해 두는 바이트 수
	g_ciZmodemDrainShow = 256                    // 마지막 데이터 조각 뒤에 붙어 온 것이 이보다 길면 화면 출력으로 보지 않는다
)

// errZmodemStalled 는 서버가 G_durZmodemStall 동안 응답하지 않았다는 오류다.
var errZmodemStalled = errors.New("zmodem: remote not responding")

// zmodemLink 는 전송 동안 엔진이 쓰는 연결(zmodem.Stream)이다.
// 연결의 읽기(Transport.Read)에는 시한을 걸 수 없으므로 읽기를 고루틴에 걸어 두고 결과를 기다린다.
// 기다리는 동안 취소를 살피고, 너무 오래 조용하면 errZmodemStalled 로 끝낸다. 전송이 끝났을 때 걸려 있는
// 읽기(chPending)는 읽기 루프가 이어 받는다 — 그 읽기가 받는 것은 전송 뒤의 화면 출력이다.
// 읽기 루프 고루틴 하나만 쓴다(잠금 없음).
type zmodemLink struct {
	conn      Transport
	fnCancel  func() bool
	chPending chan readResult // 걸어 둔 읽기. 없으면 nil
	bRest     []byte          // 읽었지만 아직 넘기지 않은 바이트
}

func (pLink *zmodemLink) Write(bData []byte) error { return pLink.conn.Write(bData) }

// wait 는 읽기 하나의 결과를 durWait 동안 기다린다. 시간 안에 오지 않으면 false 이고 읽기는 걸린 채 남는다.
func (pLink *zmodemLink) wait(durWait time.Duration) (readResult, bool) {
	if pLink.chPending == nil {
		chRead := make(chan readResult, 1)
		conn := pLink.conn
		go func() {
			bBuf := make([]byte, 32*1024)
			nRead, errRead := conn.Read(bBuf)
			chRead <- readResult{bData: bBuf[:nRead], err: errRead}
		}()
		pLink.chPending = chRead
	}
	timer := time.NewTimer(durWait)
	defer timer.Stop()
	select {
	case got := <-pLink.chPending:
		pLink.chPending = nil
		return got, true
	case <-timer.C:
		return readResult{}, false
	}
}

// Read 는 서버에서 온 바이트를 돌려준다. 취소했으면 (0, nil) 로 곧바로 돌아가 엔진이 취소를 알아채게 한다.
func (pLink *zmodemLink) Read(bBuf []byte) (int, error) {
	if len(pLink.bRest) > 0 {
		nCopy := copy(bBuf, pLink.bRest)
		pLink.bRest = pLink.bRest[nCopy:]
		return nCopy, nil
	}
	tmStall := time.Now().Add(G_durZmodemStall)
	for {
		if pLink.fnCancel != nil && pLink.fnCancel() {
			return 0, nil
		}
		got, bGot := pLink.wait(g_cdurZmodemPoll)
		if bGot {
			if len(got.bData) == 0 {
				return 0, got.err
			}
			// 데이터와 오류가 함께 왔으면 데이터부터 넘긴다(오류는 다음 읽기에서 다시 난다).
			nCopy := copy(bBuf, got.bData)
			pLink.bRest = got.bData[nCopy:]
			return nCopy, nil
		}
		if time.Now().After(tmStall) {
			return 0, errZmodemStalled
		}
	}
}

// drainAborted 는 받기를 중단한 뒤 흘러오는 전송 데이터를 읽어 버린다.
// G_durZmodemDrainQuiet 동안 조용해질 때까지 읽는다. 조용해졌을 때 걸려 있는 읽기는 pLink.chPending 에 남아
// 읽기 루프가 이어 받는다(그 읽기가 받는 것은 중단 뒤의 새 출력이다).
// 돌려주는 바이트는 마지막 데이터 조각 뒤에 바로 붙어 온 화면 출력(셸 프롬프트)이다.
// bSeen 은 엔진이 이미 읽어 버린 마지막 부분이다(끝자락의 앞부분이 된다).
func drainAborted(pLink *zmodemLink, bSeen []byte, bCrc32 bool) ([]byte, error) {
	tmGiveUp := time.Now().Add(G_durZmodemDrainMax)
	bTail := bSeen
	if len(bTail) > g_ciZmodemDrainTail {
		bTail = bTail[len(bTail)-g_ciZmodemDrainTail:]
	}
	bTail = append([]byte(nil), bTail...)
	iTotal := len(bSeen)
	for {
		got, bGot := pLink.wait(G_durZmodemDrainQuiet)
		if !bGot {
			return drainedOutput(bTail, iTotal, bCrc32), nil
		}
		if got.err != nil && len(got.bData) == 0 {
			return nil, got.err
		}
		iTotal += len(got.bData)
		bTail = append(bTail, got.bData...)
		if len(bTail) > g_ciZmodemDrainTail {
			bTail = append([]byte(nil), bTail[len(bTail)-g_ciZmodemDrainTail:]...)
		}
		if time.Now().After(tmGiveUp) {
			return nil, nil
		}
	}
}

// drainedOutput 은 버린 바이트의 끝자락에서 화면으로 되돌릴 부분을 고른다.
func drainedOutput(bTail []byte, iTotal int, bCrc32 bool) []byte {
	bAfter, bFound := zmodem.TrailingOutput(bTail, bCrc32)
	if bFound {
		if len(bAfter) > g_ciZmodemDrainShow {
			return nil // 데이터 조각이 중간에서 잘렸다: 화면 출력과 가를 수 없다
		}
		return bAfter
	}
	if iTotal > len(bTail) {
		return nil // 끝자락이 통째로 조각 하나의 안쪽이다
	}
	return bTail // 데이터 조각이 하나도 없었다: 헤더·셸 출력뿐이다
}
```

읽기 하나가 누구 손에 있는지(고루틴은 한 번에 하나만 `conn.Read` 를 부른다):

```
readLoop: conn.Read ──▶ 시작 표시 발견 ──▶ runZmodem
                                              │ 엔진 ⇄ zmodemLink.Read ⇄ wait() ─▶ [고루틴: conn.Read] ─▶ chPending
                                              │        (0.1초마다 취소 확인, 60초 시한)
                                              │ 엔진 끝 ─▶ (받다가 중단했으면) drainAborted ⇄ wait(0.5초) …
                                              ▼
                              return (화면으로 낼 바이트, pLink.chPending, 오류)
readLoop: chAhead != nil 이면 conn.Read 대신 <-chAhead  ──▶ 그 뒤로는 다시 conn.Read
```

- 전송이 잘 끝나면 `chPending` 은 nil 이다(엔진이 마지막으로 부른 `Read` 가 데이터를 받아 돌아왔다).
- 취소·시한·뒤처리로 끝나면 읽기 하나가 걸린 채 남는다. 그 고루틴이 받는 것은 **전송 뒤의 화면 출력**이므로 읽기 루프가 받아 화면에 낸다.
- 채널은 버퍼 1칸이라, 세션이 닫혀 읽기 루프가 끝난 뒤에 읽기가 돌아와도 고루틴이 막히지 않는다.

### `internal/session/session.go` — 수정 (D123-2·D125)

`readLoop` 만 바뀌었다: 걸어 둔 읽기가 있으면 `conn.Read` 대신 그 결과를 받고, 전송이 아닌 묶음에서는 깨진 시작 표시를 살핀다.

```diff
--- a/internal/session/session.go
+++ b/internal/session/session.go
@@ -330,30 +330,51 @@
 
 // readLoop 는 연결의 출력을 읽어 화면과 로그로 보낸다. 끝나면 끊김 처리한다.
 func (pSession *Session) readLoop(conn Transport, iGen int) {
 	bBuf := make([]byte, 32*1024)
-	var zDetector zmodem.Detector // 이 연결의 출력에서 ZMODEM 시작 표시를 찾는다 (zmodem.md)
+	var zDetector zmodem.Detector       // 이 연결의 출력에서 ZMODEM 시작 표시를 찾는다 (zmodem.md)
+	var chAhead chan readResult         // 끝난 ZMODEM 전송이 걸어 둔 읽기 하나 (zmodemLink)
+	var zMangled zmodem.MangledDetector // tmux·screen 이 ZDLE 을 버린 시작 표시를 찾는다 (zmodem.md §3)
+	var tmMangledNotice time.Time       // 그 안내를 마지막으로 낸 때
 	for {
 		// 프런트가 아직 처리하지 않은 출력이 상한을 넘으면 여기서 멈춘다 (session.md §8 출력 흐름 제어).
 		// 읽기를 멈추면 TCP 윈도우·PTY 파이프가 차서 상대 프로세스가 스스로 느려진다.
 		pSession.credit.Wait()
-		nRead, errRead := conn.Read(bBuf)
+		var nRead int
+		var errRead error
+		if chAhead != nil {
+			ahead := <-chAhead
+			chAhead = nil
+			nRead, errRead = copy(bBuf, ahead.bData), ahead.err
+		} else {
+			nRead, errRead = conn.Read(bBuf)
+		}
 		if nRead > 0 && pSession.isCurrent(iGen) { // 닫힌 뒤 읽힌 조각은 버린다
 			bScreen := bBuf[:nRead]
+			bMangled := false
 			if pSession.zmodemOn() {
 				bBefore, kind, bEngine := zDetector.Feed(bScreen)
+				if kind == zmodem.KindNone {
+					bMangled = zMangled.Feed(bScreen)
+				}
 				if kind != zmodem.KindNone {
 					// 서버에서 sz·rz 가 시작됐다: 표시 앞의 출력만 화면에 보내고, 전송이 끝날 때까지 연결을 엔진에 넘긴다.
 					pSession.writeOutput(pSession.decoder.Decode(bBefore))
-					bLeft, errStream := pSession.runZmodem(conn, kind, bEngine)
-					bScreen = bLeft
+					bLeft, chNext, errStream := pSession.runZmodem(conn, kind, bEngine)
+					bScreen, chAhead = bLeft, chNext
 					if errStream != nil {
 						errRead = errStream // 전송 중에 연결이 끊겼다: 아래에서 끊김으로 처리한다
 					}
 				}
 			}
 			if len(bScreen) > 0 {
 				pSession.writeOutput(pSession.decoder.Decode(bScreen))
+			}
+			if bMangled && time.Since(tmMangledNotice) >= G_durZmodemMangledNotice {
+				// 시작 표시가 깨져서 왔다(tmux·screen 안): 전송은 될 수 없으니 까닭과 끝내는 방법을 알린다.
+				// 서버로는 아무것도 보내지 않는다 — 이 글이 실행 중인 sz·rz 의 것이 아닐 수도 있다(파일 내용 등).
+				tmMangledNotice = time.Now()
+				pSession.writeOutput(noticeLine(i18n.T("be.zm.muxed")))
 			}
 		}
 		if errRead != nil {
 			if errors.Is(errRead, transport.ErrRemoteExited) {
```

### `tools/zmodemcheck/main.go` — 새 파일 (전체)

```go
// zmodemcheck 는 internal/zmodem 엔진을 실제 서버의 lrzsz(sz·rz)와 맞대어 보는 검증 도구다 (KI-012).
// 앱의 백엔드와 같은 코드(internal/session 의 세션 — 읽기 루프·저장·안내 줄·취소)를 실제 SSH PTY 셸에 물려 돌린다.
// 화면(프런트)만 빠져 있다: 세션이 화면으로 보내는 글을 모아 끝에 찍는다.
//
// 사용:
//
//	ZCHECK_PASS=... go run ./tools/zmodemcheck -host H -user U exec "명령"
//	ZCHECK_PASS=... go run ./tools/zmodemcheck -host H -user U recv -dir 받을폴더 [-cancelAfter N] "sz 파일…"
//	ZCHECK_PASS=... go run ./tools/zmodemcheck -host H -user U send [-files a,b] "rz"
//
// 비밀번호는 환경 변수 ZCHECK_PASS 로만 받는다. 호스트 키는 확인하지 않는다(사내망 검증용 — 앱의 접속 경로가 아니다).
package main

import (
	"flag"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"sync"
	"time"

	"golang.org/x/crypto/ssh"

	"dariterm/internal/i18n"
	"dariterm/internal/session"
)

func main() {
	szHost := flag.String("host", "", "서버 주소")
	iPort := flag.Int("port", 22, "포트")
	szUser := flag.String("user", "", "계정")
	iTimeout := flag.Int("timeout", 300, "전체 시한(초)")
	iStall := flag.Int("stall", 0, "서버 응답 없음 시한(초). 0 이면 앱의 기본값")
	szThen := flag.String("then", "", "끝 안내가 나온 뒤 칠 입력 (Go 문자열 표기, 예: \"\\x18\\x18\\x18\\x18\\x18\")")
	flag.Parse()
	slArgs := flag.Args()
	if *iStall > 0 {
		session.G_durZmodemStall = time.Duration(*iStall) * time.Second
	}
	if *szHost == "" || *szUser == "" || len(slArgs) < 2 {
		fmt.Fprintln(os.Stderr, "usage: zmodemcheck -host H -user U exec|recv|send [옵션] \"명령\"")
		os.Exit(2)
	}
	cfg := &ssh.ClientConfig{
		User:            *szUser,
		Auth:            []ssh.AuthMethod{ssh.Password(os.Getenv("ZCHECK_PASS"))},
		HostKeyCallback: ssh.InsecureIgnoreHostKey(),
		Timeout:         10 * time.Second,
	}
	client, errDial := ssh.Dial("tcp", fmt.Sprintf("%s:%d", *szHost, *iPort), cfg)
	if errDial != nil {
		fail("접속 실패: %v", errDial)
	}
	defer client.Close()
	// 끝 안내가 나오지 않는 경우(시작 표시가 오지 않음 등)를 대비한 도구 자체의 시한이다.
	timer := time.AfterFunc(time.Duration(*iTimeout)*time.Second, func() {
		fmt.Fprintln(os.Stderr, "\n[zmodemcheck] 시한 초과 — 연결을 닫고 끝낸다")
		if g_pScreen != nil {
			fmt.Printf("[zmodemcheck] 그때까지의 화면 출력:\n%s\n", visible(g_pScreen.text()))
		}
		_ = client.Close()
		os.Exit(3)
	})
	defer timer.Stop()

	switch slArgs[0] {
	case "exec":
		runExec(client, slArgs[1])
	case "put":
		// put 로컬파일 원격경로 — 검증 준비물(패키지 등)을 올린다. ZMODEM 과 무관한 경로(셸의 cat)다.
		if len(slArgs) < 3 {
			fail("usage: put 로컬파일 원격경로")
		}
		runPut(client, slArgs[1], slArgs[2])
	case "recv":
		fs := flag.NewFlagSet("recv", flag.ExitOnError)
		szDir := fs.String("dir", ".", "받을 폴더")
		i64Cancel := fs.Int64("cancelAfter", 0, "받는 파일이 이만큼 커지면 Ctrl+C 로 취소(0 = 취소 안 함)")
		_ = fs.Parse(slArgs[1:])
		runTransfer(client, fs.Arg(0), *szDir, nil, *i64Cancel, unquote(*szThen))
	case "send":
		fs := flag.NewFlagSet("send", flag.ExitOnError)
		szFiles := fs.String("files", "", "보낼 파일(쉼표로 구분). 비우면 선택 취소")
		_ = fs.Parse(slArgs[1:])
		var slFiles []string
		if *szFiles != "" {
			slFiles = strings.Split(*szFiles, ",")
		}
		runTransfer(client, fs.Arg(0), "", slFiles, 0, unquote(*szThen))
	default:
		fail("모르는 동작: %s", slArgs[0])
	}
}

// unquote 는 Go 문자열 표기(\\x18 등)를 실제 글자로 바꾼다.
func unquote(szText string) string {
	if szText == "" {
		return ""
	}
	szOut, errUnquote := strconv.Unquote(`"` + szText + `"`)
	if errUnquote != nil {
		fail("-then 값이 잘못됨: %v", errUnquote)
	}
	return szOut
}

func fail(szFormat string, args ...any) {
	fmt.Fprintf(os.Stderr, "[zmodemcheck] "+szFormat+"\n", args...)
	os.Exit(1)
}

// runExec 는 PTY 없이 명령 하나를 돌리고 출력을 그대로 낸다.
func runExec(client *ssh.Client, szCommand string) {
	session, errNew := client.NewSession()
	if errNew != nil {
		fail("세션 실패: %v", errNew)
	}
	defer session.Close()
	bOut, errRun := session.CombinedOutput(szCommand)
	_, _ = os.Stdout.Write(bOut)
	if errRun != nil {
		fail("명령 실패: %v", errRun)
	}
}

// runPut 은 로컬 파일을 서버의 경로에 쓴다 (PTY 없는 세션의 표준 입력 → cat).
func runPut(client *ssh.Client, szLocal string, szRemote string) {
	pFile, errOpen := os.Open(szLocal)
	if errOpen != nil {
		fail("열기 실패: %v", errOpen)
	}
	defer pFile.Close()
	session, errNew := client.NewSession()
	if errNew != nil {
		fail("세션 실패: %v", errNew)
	}
	defer session.Close()
	session.Stdin = pFile
	szQuoted := "'" + strings.ReplaceAll(szRemote, "'", `'\''`) + "'"
	if bOut, errRun := session.CombinedOutput("cat > " + szQuoted); errRun != nil {
		fail("올리기 실패: %v %s", errRun, bOut)
	}
}

// shellTransport 는 SSH PTY 셸을 세션의 Transport 로 본다.
type shellTransport struct {
	session *ssh.Session
	stdin   io.Writer
	stdout  io.Reader
}

func (pShell *shellTransport) Read(bBuf []byte) (int, error) { return pShell.stdout.Read(bBuf) }
func (pShell *shellTransport) Write(bData []byte) error {
	_, errWrite := pShell.stdin.Write(bData)
	return errWrite
}
func (pShell *shellTransport) Resize(iCols int, iRows int) error {
	return pShell.session.WindowChange(iRows, iCols)
}
func (pShell *shellTransport) Close() error { return pShell.session.Close() }

// g_pScreen 은 지금 돌고 있는 전송의 화면 기록이다 (시한 초과 때 찍는다).
var g_pScreen *screenLog

// screenLog 는 세션이 화면으로 보낸 글을 모은다.
type screenLog struct {
	mtxText sync.Mutex
	sbText  strings.Builder
}

func (pLog *screenLog) emit(szEvent string, anyData ...any) {
	if szEvent != "session.data" || len(anyData) != 2 {
		return
	}
	pLog.mtxText.Lock()
	pLog.sbText.WriteString(anyData[1].(string))
	pLog.mtxText.Unlock()
}

func (pLog *screenLog) text() string {
	pLog.mtxText.Lock()
	defer pLog.mtxText.Unlock()
	return pLog.sbText.String()
}

// runTransfer 는 앱처럼 PTY 셸을 세션으로 열어 명령을 치고, 전송이 끝난 뒤 셸이 살아 있는지 echo 로 확인한다.
func runTransfer(client *ssh.Client, szCommand string, szDir string, slFiles []string, i64CancelAfter int64, szThen string) {
	i18n.SetLang("ko")
	shell, errNew := client.NewSession()
	if errNew != nil {
		fail("세션 실패: %v", errNew)
	}
	// 앱(internal/sshconn/shell.go)과 같은 PTY 요청.
	modes := ssh.TerminalModes{ssh.ECHO: 1, ssh.TTY_OP_ISPEED: 14400, ssh.TTY_OP_OSPEED: 14400}
	if errPty := shell.RequestPty("xterm-256color", 24, 80, modes); errPty != nil {
		fail("PTY 실패: %v", errPty)
	}
	stdin, _ := shell.StdinPipe()
	stdout, _ := shell.StdoutPipe()
	if errShell := shell.Shell(); errShell != nil {
		fail("셸 실패: %v", errShell)
	}
	transport := &shellTransport{session: shell, stdin: stdin, stdout: stdout}

	log := &screenLog{}
	g_pScreen = log
	pManager := session.NewManager(log.emit)
	info, errOpen := pManager.Open(session.OpenConfig{
		Info:     session.Info{M_szTitle: "zmodemcheck", M_szProtocol: "ssh", M_szTarget: client.RemoteAddr().String()},
		Encoding: "utf-8",
		Dialer:   func(func(string)) (session.Transport, error) { return transport, nil },
		Zmodem: &session.ZmodemHooks{
			DownloadDir: func() string { return szDir },
			PickFiles:   func() ([]string, error) { return slFiles, nil },
		},
	})
	if errOpen != nil {
		fail("세션 열기 실패: %v", errOpen)
	}
	pSession, _ := pManager.Get(info.M_szId)
	go func() { // 프런트 대신 출력 확인(ack)을 해 준다 (출력 흐름 제어에 막히지 않게)
		for {
			time.Sleep(5 * time.Millisecond)
			pSession.AckOutput(1000)
		}
	}()
	time.Sleep(1500 * time.Millisecond) // 로그인 배너·프롬프트
	tmStart := time.Now()
	pSession.Input(szCommand + "\r")

	slEnds := []string{"ZMODEM 전송 완료", i18n.T("be.zm.cancelled"), i18n.T("be.zm.remoteCancelled"), "ZMODEM 전송 실패", "ZMODEM 전송을 중단했습니다", "ZMODEM 시작 표시가 깨져서"}
	szEnd := ""
	bCancelled := false
	for szEnd == "" {
		time.Sleep(20 * time.Millisecond)
		if i64CancelAfter > 0 && !bCancelled && partSize(szDir) >= i64CancelAfter {
			fmt.Printf("[zmodemcheck] 받는 파일이 %d바이트를 넘어 Ctrl+C 를 보냄\n", i64CancelAfter)
			pSession.Input("\x03")
			bCancelled = true
		}
		szScreen := log.text()
		for _, szOne := range slEnds {
			if strings.Contains(szScreen, szOne) {
				szEnd = szOne
			}
		}
	}
	fmt.Printf("[zmodemcheck] 끝 안내: %q (명령 뒤 %s)\n", szEnd, time.Since(tmStart).Round(time.Millisecond))
	iEndAt := strings.Index(log.text(), szEnd)

	if szThen != "" {
		time.Sleep(500 * time.Millisecond)
		fmt.Printf("[zmodemcheck] 끝 안내 뒤 입력: %q\n", szThen)
		pSession.Input(szThen)
		time.Sleep(time.Second)
	}
	// 셸이 살아 있는지: 계산 결과가 찍혀야 한다(친 글자의 에코와 구분된다). 전송 상태가 풀릴 때까지 몇 번 다시 친다.
	bAlive := false
	for iTry := 0; iTry < 10 && !bAlive; iTry++ {
		pSession.Input("echo ZC_ALIVE_$((40+2))\r")
		for iWait := 0; iWait < 50 && !bAlive; iWait++ {
			time.Sleep(20 * time.Millisecond)
			bAlive = strings.Contains(log.text(), "ZC_ALIVE_42")
		}
	}
	szScreen := log.text()
	// 끝 안내 뒤로 화면에 나온 것 중 글자가 아닌 바이트 수: 전송 데이터가 화면으로 샜는지 본다.
	iJunk := 0
	for _, bOne := range []byte(szScreen[iEndAt:]) {
		if bOne < 0x20 && bOne != '\r' && bOne != '\n' && bOne != 0x1b && bOne != 0x07 && bOne != 0x08 || bOne == 0x7f {
			iJunk++
		}
	}
	szShow := szScreen
	if len(szShow) > 6000 {
		szShow = szShow[:2500] + "\n…(가운데 생략)…\n" + szShow[len(szShow)-2500:]
	}
	fmt.Printf("[zmodemcheck] 화면 출력(%d바이트, 끝 안내 뒤 %d바이트 중 제어 문자 %d개):\n%s\n", len(szScreen), len(szScreen)-iEndAt, iJunk, visible(szShow))
	if bAlive {
		fmt.Println("[zmodemcheck] 셸 확인: 전송 뒤 프롬프트가 돌아와 명령이 실행됨")
	} else {
		fmt.Println("[zmodemcheck] 셸 확인 실패: 전송 뒤 명령이 실행되지 않음")
	}
	pSession.Input("exit\r")
	time.Sleep(200 * time.Millisecond)
	if !bAlive {
		os.Exit(1)
	}
}

// partSize 는 폴더에서 받는 중인 파일(.dtpart)의 크기다. 없으면 0.
func partSize(szDir string) int64 {
	slParts, _ := filepath.Glob(filepath.Join(szDir, "*.dtpart"))
	var i64Max int64
	for _, szPart := range slParts {
		if pInfo, errStat := os.Stat(szPart); errStat == nil && pInfo.Size() > i64Max {
			i64Max = pInfo.Size()
		}
	}
	return i64Max
}

// visible 은 제어 문자를 보이는 글자로 바꾼다(화면에 이진 쓰레기가 섞였는지 눈으로 볼 수 있게).
func visible(szText string) string {
	var sb strings.Builder
	for _, bOne := range []byte(szText) {
		switch {
		case bOne == '\n' || bOne == '\t' || bOne >= 0x20 && bOne != 0x7f:
			sb.WriteByte(bOne)
		case bOne == '\r':
		default:
			fmt.Fprintf(&sb, "\\x%02x", bOne)
		}
	}
	return sb.String()
}
```

---

## C. 테스트 코드

### `internal/zmodem/zmodem_test.go` — 더한 시험 2개

import 에 `"strings"` 를 더했다.

```go
// 받다가 중단한 전송의 끝자락에서 마지막 데이터 조각 뒤의 화면 출력을 가른다 (실제 lrzsz 에서 본 모양: 조각 끝 + 프롬프트).
func TestTrailingOutput(t *testing.T) {
	bData := bytes.Repeat([]byte{0x18, 0x11, 'a', 0xff}, 64) // 바꿔서 가야 하는 바이트가 섞인 내용
	const cszPrompt = "\x1b]0;user@host:~\x07[user@host ~]$ "
	for _, bCrc32 := range []bool{true, false} {
		bStream := appendSubpacket(nil, bData, g_cbZcrcg, bCrc32)
		bStream = appendSubpacket(bStream, bData, g_cbZcrcg, bCrc32)
		bAfter, bFound := TrailingOutput(append(append([]byte(nil), bStream...), cszPrompt...), bCrc32)
		if !bFound || string(bAfter) != cszPrompt {
			t.Fatalf("crc32=%v: 조각 뒤의 출력 = %q (found=%v)", bCrc32, bAfter, bFound)
		}
		// 조각 끝에서 딱 끝났으면 뒤에 붙은 것이 없다.
		if bAfter, bFound = TrailingOutput(bStream, bCrc32); !bFound || len(bAfter) != 0 {
			t.Fatalf("crc32=%v: 조각 끝에서 끝난 흐름 = %q (found=%v)", bCrc32, bAfter, bFound)
		}
		// CRC 가 잘린 채 끝났으면 빈 값이다.
		if bAfter, bFound = TrailingOutput(bStream[:len(bStream)-1], bCrc32); !bFound || len(bAfter) != 0 {
			t.Fatalf("crc32=%v: CRC 가 잘린 흐름 = %q (found=%v)", bCrc32, bAfter, bFound)
		}
	}
	// CRC 바이트가 바꿔서(ZDLE + 값) 온 경우: 두 바이트를 한 바이트로 센다.
	bEscaped := []byte{'x', g_cbZdle, g_cbZcrcg, g_cbZdle, 0x58, 'A', g_cbZdle, 0x51, 'B', '$', ' '}
	if bAfter, bFound := TrailingOutput(bEscaped, true); !bFound || string(bAfter) != "$ " {
		t.Fatalf("바꿔서 온 CRC: %q (found=%v)", bAfter, bFound)
	}
	// 데이터 조각이 없으면 찾지 못한 것으로 알린다.
	if _, bFound := TrailingOutput([]byte("plain shell output\r\n$ "), true); bFound {
		t.Fatal("조각 끝 표시가 없으면 found=false 여야 함")
	}
	if _, bFound := TrailingOutput(nil, true); bFound {
		t.Fatal("빈 입력은 found=false 여야 함")
	}
}

// tmux·screen 이 ZDLE 을 버린 시작 표시("**B0" + 종류 + 16진 12자)를 찾는다. 진짜 시작 표시(ZDLE 포함)는 찾지 않는다.
func TestMangledDetector(t *testing.T) {
	for _, tc := range []struct {
		szName string
		slIn   []string
		iWant  int
	}{
		{"sz 의 표시", []string{"rz\x1b[H**B00000000000000\x1b[H"}, 1},
		{"rz 의 표시", []string{"rz waiting to receive.**B0100000023be50"}, 1},
		{"묶음 경계에 걸침", []string{"abc*", "*B0", "0000000", "0000000 xyz"}, 1},
		{"한 글자씩", strings.Split("**B0100000023be50", ""), 1},
		{"별이 더 붙음", []string{"****B00000000000000"}, 1},
		{"두 번", []string{"**B00000000000000 ... **B0100000023be50"}, 2},
		{"진짜 시작 표시(ZDLE 있음)", []string{"rz\r**\x18B00000000000000\r\x8a\x11"}, 0},
		{"16진이 모자람", []string{"**B0000000000000 "}, 0},
		{"대문자 16진", []string{"**B0100000023BE50"}, 0},
		{"다른 종류(ZFIN)", []string{"**B0800000000022d"}, 0},
		{"보통 글", []string{"ls -la **/*.go\r\nB0 0 1\r\n"}, 0},
	} {
		var detector MangledDetector
		iGot := 0
		for _, szPart := range tc.slIn {
			if detector.Feed([]byte(szPart)) {
				iGot++
			}
		}
		// "두 번"은 한 묶음 안에 둘이 들어 있어 Feed 한 번이 true 다: 묶음 수로 센다.
		if tc.szName == "두 번" {
			if iGot != 1 {
				t.Errorf("%s: 묶음 하나에 여러 번 있어도 true 한 번이어야 함, got %d", tc.szName, iGot)
			}
			continue
		}
		if iGot != tc.iWant {
			t.Errorf("%s: 찾은 횟수 = %d, 기대 %d", tc.szName, iGot, tc.iWant)
		}
	}
}
```

### `internal/session/zmodem_test.go` — 더한 시험 5개

기존 도우미(`linkTransport`·`remoteEnd`·`openZmodemSession`·`holdReader`)를 그대로 쓴다.

| 시험 | 본 것 |
|---|---|
| `TestZmodemCancelDropsDataInFlight` | 받는 중 Ctrl+C → 그 뒤 300개 조각(약 300KB) + `PROMPT$ ` 가 와도 화면에는 `PROMPT$ ` 만. 받던 파일이 지워짐. 그 뒤의 출력(`NEXT-OUTPUT`)이 순서대로 나옴(걸어 둔 읽기 이어받기) |
| `TestZmodemStalledRemoteTimesOut` | 시작 표시만 보내고 멈춘 서버 → 시한(시험에서는 0.3초) 뒤 전송 상태가 풀리고 안내가 나옴. 서버에 CAN 5개 이상이 감. 세션은 연결된 채, 입력이 서버로 가고 출력이 화면에 나옴 |
| `TestZmodemCancelWhileRemoteSilent` | 서버가 조용한 동안 Ctrl+C → 1초 안에 전송 상태가 풀림. 취소 안내, 그 뒤 출력이 화면에 나옴 |
| `TestZmodemMangledStartShowsNotice` | `ZDLE` 이 빠진 시작 표시(묶음 경계에 걸침) → 전송을 시작하지 않고 안내 한 번. 서버가 보낸 글은 그대로. 10초 안의 두 번째 표시에는 안내를 내지 않음. 서버로 아무것도 보내지 않음 |
| `TestZmodemMangledStartIgnoredWhenDisabled` | `zmodemEnabled` 가 꺼져 있으면 안내를 내지 않음 |

```go
// 받는 중에 취소하면, 서버가 이미 보낸 전송 데이터가 뒤이어 흘러와도 화면에 내보내지 않고 버린다.
// 그 끝에 붙어 온 셸 프롬프트와 그 뒤의 출력은 화면에 나온다 (zmodem.md §7 — 2026-11-25 실제 lrzsz 로 확인한 흐름).
func TestZmodemCancelDropsDataInFlight(t *testing.T) {
	durOld := G_durZmodemDrainQuiet
	G_durZmodemDrainQuiet = 200 * time.Millisecond
	t.Cleanup(func() { G_durZmodemDrainQuiet = durOld })

	szDir := t.TempDir()
	pLink := newLinkTransport()
	pSession, rec := openZmodemSession(t, pLink, &ZmodemHooks{DownloadDir: func() string { return szDir }})
	pEnd := &remoteEnd{pLink: pLink}
	chHold := make(chan struct{}) // 끝까지 닫지 않는다: 서버 흉내의 보내는 쪽은 64KB 를 보낸 뒤 멈춰 있다
	go func() {
		_ = pEnd.Write([]byte(g_cszTestZrqinit))
		src := zmodem.FileSource{Name: "flood.bin", Size: 10 * 1024 * 1024, ModTime: time.Now(),
			Open: func() (io.ReadSeekCloser, error) { return &holdReader{chHold: chHold, iBefore: 64 * 1024}, nil }}
		var detector zmodem.Detector
		bBuf := make([]byte, 4096)
		for {
			nRead, errRead := pEnd.Read(bBuf)
			if errRead != nil {
				return
			}
			if _, kind, bEngine := detector.Feed(bBuf[:nRead]); kind == zmodem.KindRemoteAccepts {
				_, _ = zmodem.Send(pEnd, bEngine, func() ([]zmodem.FileSource, error) { return []zmodem.FileSource{src}, nil }, zmodem.Options{})
				return
			}
		}
	}()
	szTemp := filepath.Join(szDir, "flood.bin"+g_cszZmodemTempSuffix)
	waitUntil(t, func() bool {
		pInfo, errStat := os.Stat(szTemp)
		return errStat == nil && pInfo.Size() > 0
	}, 5*time.Second)
	pSession.Input("\x03")

	// 취소 뒤에도 흘러오는 데이터: 1024바이트 조각 300개(끝 표시 ZDLE 'i' + CRC 4바이트), 마지막 조각 뒤에 프롬프트가 바로 붙는다.
	bPiece := append(bytes.Repeat([]byte("G"), 1024), 0x18, 'i', 'c', 'r', 'c', '4')
	var bFlood []byte
	for iPiece := 0; iPiece < 300; iPiece++ {
		bFlood = append(bFlood, bPiece...)
	}
	_ = pEnd.Write(append(bFlood, "PROMPT$ "...))

	waitUntil(t, func() bool { return strings.Contains(rec.screen(), "PROMPT$ ") }, 5*time.Second)
	waitUntil(t, func() bool { return !pSession.zActive.Load() }, time.Second)
	if strings.Contains(rec.screen(), "GGGG") {
		t.Fatal("취소 뒤에 온 전송 데이터가 화면에 나오면 안 됨")
	}
	if _, errStat := os.Stat(szTemp); !os.IsNotExist(errStat) {
		t.Fatal("취소하면 받던 파일이 남지 않아야 함")
	}
	// 조용해진 뒤의 출력은 그대로 화면에 나온다 (뒤처리가 걸어 둔 읽기를 읽기 루프가 이어 받는다).
	_ = pEnd.Write([]byte("NEXT-OUTPUT"))
	waitUntil(t, func() bool { return strings.Contains(rec.screen(), "NEXT-OUTPUT") }, 2*time.Second)
	iPrompt := strings.Index(rec.screen(), "PROMPT$ ")
	if iNext := strings.Index(rec.screen(), "NEXT-OUTPUT"); iNext < iPrompt {
		t.Fatalf("출력 순서가 바뀜: 프롬프트 %d, 다음 출력 %d", iPrompt, iNext)
	}
}

// 서버가 시작 표시만 보내고 아무 응답도 하지 않으면, 시한이 지난 뒤 전송을 중단하고 세션은 그대로 쓴다 (zmodem.md §7, KI-012).
func TestZmodemStalledRemoteTimesOut(t *testing.T) {
	durOld, durQuiet := G_durZmodemStall, G_durZmodemDrainQuiet
	G_durZmodemStall, G_durZmodemDrainQuiet = 300*time.Millisecond, 100*time.Millisecond
	t.Cleanup(func() { G_durZmodemStall, G_durZmodemDrainQuiet = durOld, durQuiet })

	pLink := newLinkTransport()
	pSession, rec := openZmodemSession(t, pLink, &ZmodemHooks{DownloadDir: func() string { return t.TempDir() }})
	pEnd := &remoteEnd{pLink: pLink}
	_ = pEnd.Write([]byte(g_cszTestZrqinit)) // sz 가 시작만 하고 멈췄다

	waitUntil(t, func() bool { return pSession.zActive.Load() }, time.Second)
	waitUntil(t, func() bool { return !pSession.zActive.Load() }, 3*time.Second)
	waitUntil(t, func() bool { return strings.Contains(rec.screen(), "응답하지 않아") }, 2*time.Second) // 응답 없음 안내
	// 서버에는 중단 신호(CAN 5개 이상)가 갔다.
	var bSent []byte
	for len(pLink.chToRemote) > 0 {
		bSent = append(bSent, <-pLink.chToRemote...)
	}
	if !bytes.Contains(bSent, []byte{0x18, 0x18, 0x18, 0x18, 0x18}) {
		t.Fatalf("시한이 지나면 서버에 중단을 알려야 함: %q", bSent)
	}
	// 연결은 끊기지 않았다: 입력이 서버로 가고, 서버의 출력이 화면에 나온다.
	if pSession.Snapshot().M_szState != G_cszStateConnected {
		t.Fatalf("세션 상태: %s", pSession.Snapshot().M_szState)
	}
	pSession.Input("after")
	select {
	case bGot := <-pLink.chToRemote:
		if string(bGot) != "after" {
			t.Fatalf("시한 뒤 입력: %q", string(bGot))
		}
	case <-time.After(time.Second):
		t.Fatal("시한 뒤에는 입력이 서버로 가야 함")
	}
	_ = pEnd.Write([]byte("PROMPT$ "))
	waitUntil(t, func() bool { return strings.Contains(rec.screen(), "PROMPT$ ") }, 2*time.Second)
}

// 서버가 조용한 동안에도 Ctrl+C 는 곧바로 전송을 끝낸다(전에는 서버에서 다음 바이트가 와야 끝났다).
func TestZmodemCancelWhileRemoteSilent(t *testing.T) {
	durQuiet := G_durZmodemDrainQuiet
	G_durZmodemDrainQuiet = 100 * time.Millisecond
	t.Cleanup(func() { G_durZmodemDrainQuiet = durQuiet })

	pLink := newLinkTransport()
	pSession, rec := openZmodemSession(t, pLink, &ZmodemHooks{DownloadDir: func() string { return t.TempDir() }})
	pEnd := &remoteEnd{pLink: pLink}
	_ = pEnd.Write([]byte(g_cszTestZrqinit))
	waitUntil(t, func() bool { return pSession.zActive.Load() }, time.Second)

	tmStart := time.Now()
	pSession.Input("\x03") // 서버는 그 뒤로도 아무것도 보내지 않는다
	waitUntil(t, func() bool { return !pSession.zActive.Load() }, 2*time.Second)
	if durTook := time.Since(tmStart); durTook > time.Second {
		t.Fatalf("조용한 서버에서 취소가 너무 늦음: %s", durTook)
	}
	waitUntil(t, func() bool { return strings.Contains(rec.screen(), "취소했습니다") }, 2*time.Second) // 취소 안내
	// 그 뒤의 출력은 화면에 나온다 (걸려 있던 읽기를 읽기 루프가 이어 받는다).
	_ = pEnd.Write([]byte("PROMPT$ "))
	waitUntil(t, func() bool { return strings.Contains(rec.screen(), "PROMPT$ ") }, 2*time.Second)
}

// tmux·screen 안에서 sz·rz 를 실행하면 ZDLE 이 빠진 시작 표시가 글자로 온다: 전송을 시작하지 않고 까닭을 알린다.
// 서버로는 아무것도 보내지 않는다 (2026-11-25 tmux 3.2a 에서 본 모양).
func TestZmodemMangledStartShowsNotice(t *testing.T) {
	pLink := newLinkTransport()
	pSession, rec := openZmodemSession(t, pLink, &ZmodemHooks{DownloadDir: func() string { return t.TempDir() }})
	pEnd := &remoteEnd{pLink: pLink}
	_ = pEnd.Write([]byte("rz\x1b[H**B0000000"))
	_ = pEnd.Write([]byte("0000000\x1b[H")) // 묶음 경계에 걸쳐 와도 찾는다
	waitUntil(t, func() bool { return strings.Contains(rec.screen(), "tmux") }, 2*time.Second)
	if pSession.zActive.Load() {
		t.Fatal("깨진 시작 표시로는 전송을 시작하면 안 됨")
	}
	if !strings.Contains(rec.screen(), "**B00000000000000") {
		t.Fatal("서버가 보낸 글은 그대로 화면에 나와야 함")
	}
	// 같은 표시가 곧바로 다시 와도 안내는 한 번만 낸다.
	_ = pEnd.Write([]byte("**B0100000023be50"))
	waitUntil(t, func() bool { return strings.Contains(rec.screen(), "**B0100000023be50") }, 2*time.Second)
	if iCount := strings.Count(rec.screen(), "깨져서"); iCount != 1 {
		t.Fatalf("안내 횟수 = %d, 기대 1", iCount)
	}
	if len(pLink.chToRemote) != 0 {
		t.Fatal("깨진 시작 표시를 봤다고 서버로 무엇을 보내면 안 됨")
	}
}

// 설정으로 ZMODEM 을 끄면 깨진 시작 표시도 살피지 않는다.
func TestZmodemMangledStartIgnoredWhenDisabled(t *testing.T) {
	pLink := newLinkTransport()
	_, rec := openZmodemSession(t, pLink, &ZmodemHooks{Enabled: func() bool { return false }})
	pEnd := &remoteEnd{pLink: pLink}
	_ = pEnd.Write([]byte("**B00000000000000 END"))
	waitUntil(t, func() bool { return strings.Contains(rec.screen(), " END") }, 2*time.Second)
	if strings.Contains(rec.screen(), "tmux") {
		t.Fatal("ZMODEM 이 꺼져 있으면 안내를 내면 안 됨")
	}
}
```

---

## E. 문구

키 2개를 `be.zm.failed` 다음에 더했다. 백엔드가 읽는 문구이고 프런트의 타입·계약 파일과는 관계없다.

| 언어 | `be.zm.stalled` | `be.zm.muxed` |
|---|---|---|
| ko | 서버가 {sec}초 동안 응답하지 않아 ZMODEM 전송을 중단했습니다 | ZMODEM 시작 표시가 깨져서 왔습니다 — tmux·screen 안에서는 전송할 수 없습니다. Ctrl+X 를 5번 눌러 끝내고 tmux 밖에서 다시 실행하세요 |
| en | ZMODEM transfer stopped: no response from the server for {sec} seconds | The ZMODEM start marker arrived damaged — transfers do not work inside tmux or screen. Press Ctrl+X five times to stop it, then run it again outside tmux |
| de | ZMODEM-Übertragung abgebrochen: seit {sec} Sekunden keine Antwort vom Server | Die ZMODEM-Startmarke kam beschädigt an — in tmux oder screen ist keine Übertragung möglich. Fünfmal Strg+X drücken, um abzubrechen, und außerhalb von tmux erneut ausführen |
| es | Transferencia ZMODEM detenida: el servidor no respondió durante {sec} segundos | La marca de inicio de ZMODEM llegó dañada: no se puede transferir dentro de tmux o screen. Pulse Ctrl+X cinco veces para detenerlo y ejecútelo de nuevo fuera de tmux |
| fr | Transfert ZMODEM arrêté : aucune réponse du serveur depuis {sec} secondes | Le marqueur de début ZMODEM est arrivé endommagé — le transfert ne fonctionne pas dans tmux ou screen. Appuyez cinq fois sur Ctrl+X pour l'arrêter, puis relancez hors de tmux |
| ja | サーバーが {sec} 秒間応答しないため、ZMODEM 転送を中止しました | ZMODEM の開始マーカーが壊れて届きました — tmux・screen の中では転送できません。Ctrl+X を 5 回押して終了し、tmux の外で実行し直してください |
| pt | Transferência ZMODEM interrompida: o servidor não respondeu durante {sec} segundos | O marcador de início do ZMODEM chegou danificado — não é possível transferir dentro do tmux ou screen. Prima Ctrl+X cinco vezes para parar e execute novamente fora do tmux |
| ru | Передача ZMODEM остановлена: сервер не отвечает {sec} с | Начальная метка ZMODEM пришла повреждённой — внутри tmux или screen передача не работает. Нажмите Ctrl+X пять раз, чтобы остановить, и запустите снова вне tmux |
| tw | 伺服器 {sec} 秒沒有回應，已中止 ZMODEM 傳輸 | ZMODEM 起始標記已損壞 — 在 tmux、screen 內無法傳輸。請按 5 次 Ctrl+X 結束，並在 tmux 外重新執行 |
| zh | 服务器 {sec} 秒没有响应，已中止 ZMODEM 传输 | ZMODEM 起始标记已损坏 — 在 tmux、screen 内无法传输。请按 5 次 Ctrl+X 结束，并在 tmux 外重新执行 |

### `locales/ko.json` — 수정 (다른 9개 언어도 같은 자리)

```diff
--- a/locales/ko.json
+++ b/locales/ko.json
@@ -777,8 +777,10 @@
   "be.zm.done": "ZMODEM 전송 완료: 파일 {n}개, {size}",
   "be.zm.cancelled": "ZMODEM 전송을 취소했습니다",
   "be.zm.remoteCancelled": "서버 쪽에서 ZMODEM 전송을 중단했습니다",
   "be.zm.failed": "ZMODEM 전송 실패: {error}",
+  "be.zm.stalled": "서버가 {sec}초 동안 응답하지 않아 ZMODEM 전송을 중단했습니다",
+  "be.zm.muxed": "ZMODEM 시작 표시가 깨져서 왔습니다 — tmux·screen 안에서는 전송할 수 없습니다. Ctrl+X 를 5번 눌러 끝내고 tmux 밖에서 다시 실행하세요",
   "be.zm.noDir": "받은 파일을 저장할 폴더를 정할 수 없습니다",
   "be.zm.noName": "같은 이름의 파일이 너무 많아 저장할 이름을 정할 수 없습니다",
   "be.dlg.pickSaveDir": "저장할 폴더 선택",
   "be.sftp.mkUploadDir": "업로드 폴더를 만들지 못했습니다 ({dir})",
```

---

## F. 명세

### `spec/behavior/zmodem.md` — 수정

```diff
--- a/spec/behavior/zmodem.md
+++ b/spec/behavior/zmodem.md
@@ -17,8 +17,9 @@
 
 - 세션 출력에 **시작 표시**가 나타나면 전송을 시작한다. 시작 표시는 16진 헤더의 앞부분 `* * 0x18 B 0` 다음에 `0`(서버가 보내려 함, `sz`) 또는 `1`(서버가 받으려 함, `rz`)이 오는 것이다. 표시가 출력 묶음 경계에 걸쳐 와도 찾는다.
 - 표시 **앞의 출력은 그대로 화면에 보낸다**(`sz` 가 먼저 찍는 `rz` 글자 등).
 - 표시처럼 보였지만 뒤가 올바른 헤더가 아니면(검사값 불일치) 전송하지 않고 읽은 바이트를 화면 출력으로 되돌린다.
+- **tmux·screen 안에서는 전송할 수 없다**(한계, 2026-11-25 D125): 다중화기가 안에서 돈 프로그램의 출력을 해석해 다시 그리면서 제어 문자 0x18 을 버리므로 시작 표시도 전송 데이터도 깨져서 온다. 0x18 이 빠진 시작 표시(`**B0` + `0` 또는 `1` + 16진 소문자 12자 — 예: `**B00000000000000`, `**B0100000023be50`)가 출력에 나타나면 전송은 시작하지 않고, 그 출력은 그대로 화면에 낸 뒤 안내 한 줄을 낸다: `ZMODEM 시작 표시가 깨져서 왔습니다 — tmux·screen 안에서는 전송할 수 없습니다. Ctrl+X 를 5번 눌러 끝내고 tmux 밖에서 다시 실행하세요`. 같은 안내는 **10초 안에 다시 내지 않는다**. 서버로는 아무것도 보내지 않는다(그 글이 실행 중인 `sz`·`rz` 의 것이 아닐 수 있다). `zmodemEnabled` 가 꺼져 있으면 살피지 않는다. (서버의 `sz`·`rz` 는 터미널을 날것 방식으로 바꿔 두어 Ctrl+C 로는 끝나지 않고, Ctrl+X 5번이 중단 신호다.)
 - 전송이 시작되면 끝날 때까지 **그 연결의 입출력을 전송에만 쓴다**: 서버가 보낸 바이트는 화면에 나오지 않고, **터미널 입력은 서버로 보내지 않는다**(전송 데이터에 섞이지 않게). 일괄 입력으로 들어온 입력도 같다.
 
 ## 4. 받기 (서버에서 `sz`)
 
@@ -58,10 +59,12 @@
 ## 7. 취소
 
 - 전송 중에 터미널에서 **Ctrl+C** 를 누르면 취소한다: 서버에 중단 신호(CAN 8개)를 보내고, 받던 파일은 지운다.
 - 취소한 뒤 도착한 전송 데이터는 화면에 내보내지 않고 버린다. 그 뒤의 출력(셸 프롬프트)부터 다시 화면에 나온다.
+  - (2026-11-25, D123-2) 받는 중에 취소·실패하면 서버가 이미 보낸 데이터가 한동안 더 온다. **0.5초 동안 아무것도 오지 않을 때까지** 읽어 버리고(최대 20초 — 넘으면 버리기를 그만둔다), 마지막 데이터 조각 바로 뒤에 붙어 온 256바이트 이하의 출력(셸 프롬프트)만 화면에 낸다. 그동안의 입력은 서버로 보내지 않는다. 취소 안내 줄은 이 뒤처리가 끝난 뒤에 나온다.
 - 세션을 닫거나 연결이 끊기면 전송도 끝난다(끊김 처리는 보통 세션과 같다 — 자동 재접속 규칙 적용).
-- **시간 제한은 없다**: 서버 쪽 프로그램이 응답 없이 멈추면 전송 상태로 남으므로 Ctrl+C 로 풀어야 한다(알려진 한계).
+- **응답 시한 60초** (2026-11-25, D125 — 전에는 시간 제한이 없었다): 앱이 서버의 응답을 기다리는 동안 **60초** 동안 한 바이트도 오지 않으면 서버에 중단 신호를 보내고 전송을 끝낸다. 연결은 끊지 않는다(세션은 그대로 쓴다). 안내: `서버가 60초 동안 응답하지 않아 ZMODEM 전송을 중단했습니다`. 받던 파일은 지운다. 보낼 파일을 고르는 동안(파일 선택 창)은 시한에 넣지 않는다.
+- **Ctrl+C 는 서버가 조용해도 곧바로 듣는다**(0.1초 안, D125): 전에는 취소한 뒤 서버에서 다음 바이트가 와야 전송 상태가 풀렸다.
 
 ## 8. 오류 복구와 한도
 
 - 받은 데이터의 검사값이 맞지 않으면 그 위치부터 다시 보내 달라고 요청해 이어 받는다. 전송 하나에서 이런 오류가 **10번**을 넘으면 실패로 끝낸다.
@@ -91,5 +94,9 @@
 | Z5 | 큰 파일을 `sz` 로 받는 중 | Ctrl+C | 전송이 멈추고 `.dtpart` 파일이 지워진다. 그 뒤 입력이 정상으로 간다 |
 | Z6 | 큰 파일을 받는 중 | 글자를 친다 | 서버로 가지 않는다(전송이 깨지지 않는다) |
 | Z7 | 설정에서 ZMODEM 을 끔 | `sz 파일` | 전송하지 않고 서버가 보낸 글자가 화면에 그대로 나온다 |
 | Z8 | 서버가 `../../x` 같은 이름을 알림 | 받기 | 받는 폴더 안에 `x` 로 저장된다(폴더 밖에 쓰지 않는다) |
+| Z9 | 큰 파일(수십 MB)을 `sz` 로 받는 중 | Ctrl+C | `ZMODEM 전송을 취소했습니다` 뒤에 프롬프트가 나온다. 화면에 깨진 글자(전송 데이터)가 나오지 않는다 (D123-2) |
+| Z10 | 서버에서 `printf '**\030B00000000000000\r\212\021'; sleep 300` (시작만 하고 멈춘 서버 흉내) | 60초 기다림 | `서버가 60초 동안 응답하지 않아 ZMODEM 전송을 중단했습니다`. 세션은 연결된 채이고 입력이 다시 서버로 간다 (D125) |
+| Z11 | Z10 과 같은 명령 | 곧바로 Ctrl+C | 1초 안에 `ZMODEM 전송을 취소했습니다`. 입력이 다시 서버로 간다 (D125) |
+| Z12 | tmux 안에서 `sz /etc/hostname` | — | 전송은 시작되지 않고 `**B00000000000000` 같은 글 뒤에 "tmux·screen 안에서는 전송할 수 없습니다…" 안내가 한 번 나온다. Ctrl+X 를 5번 누르면 `sz` 가 끝난다 (D125) |
 
```

---

## G. 검증 (2026-11-25 실행)

환경: Windows 11, go1.27.1 windows/amd64. 서버 Rocky Linux 9.8, lrzsz 0.12.20, tmux 3.2a.

### G.1 자동 테스트

```
go build ./...                     통과
go vet ./...                       통과
go test ./... -count=1             전부 ok (19개 패키지)
  dariterm/internal/zmodem         ok   (시험 22개 — 2개 추가)
  dariterm/internal/session        ok   (ZMODEM 시험 14개 — 5개 추가)
gofmt -l internal/zmodem internal/session/zmodem.go internal/session/session.go tools/zmodemcheck    (없음)
```

`-race` 는 이 PC 에 cgo 컴파일러가 없어 돌리지 않았다. `zmodemLink` 는 읽기 루프 고루틴 하나만 쓰고, 읽기 고루틴과는 버퍼 1칸 채널로만 주고받는다.

### G.2 시험이 결함을 잡는지 확인

뒤처리를 부르는 조건 앞에 `false &&` 를 넣어 수정을 끈 채 돌렸다:

```
--- FAIL: TestZmodemCancelDropsDataInFlight (0.03s)
    zmodem_test.go:524: 취소 뒤에 온 전송 데이터가 화면에 나오면 안 됨
```

되돌린 뒤 통과.

### G.3 실제 서버 (최종 코드)

| # | 한 것 | 결과 |
|---|---|---|
| A1 | 시작 표시만 보내고 멈춘 서버(`printf '**\030B00000000000000\r\212\021'; sleep 30`), `-stall 5` | 5.5초 뒤 `서버가 5초 동안 응답하지 않아 ZMODEM 전송을 중단했습니다`. 안내 뒤 제어 문자 0개. (도구의 셸 확인은 실패로 찍힌다 — `sleep 30` 이 아직 앞에서 돌고 있어 `echo` 가 실행되지 않았을 뿐이다) |
| A3 | `sz big40m.bin mid3m.bin /etc/hostname` | `ZMODEM 전송 완료: 파일 3개, 43.0 MB`, 2.1초. SHA256 세 개 모두 같음 |
| A4 | `rz` 로 3개 | `ZMODEM 전송 완료: 파일 3개, 5.3 MB`. 서버에서 SHA256 세 개 모두 같음 |
| A5 | `rz` 선택 취소 | `ZMODEM 전송을 취소했습니다`, 프롬프트 복귀 |
| A6 | 40MB 받는 중 9MB 에서 Ctrl+C | `ZMODEM 전송을 취소했습니다`(명령 뒤 1.0초), 안내 뒤 제어 문자 0개, 프롬프트 복귀, 받는 폴더에 남은 파일 0개 |
| T1 | tmux 안에서 `sz /etc/hostname` → 안내 뒤 Ctrl+C | 안내가 나옴. `sz` 는 끝나지 않음 |
| T2 | tmux 안에서 `sz /etc/hostname` → 안내 뒤 Ctrl+X 5번 | 안내가 나옴. `SZ_ENDED=128`, 셸에서 명령이 실행됨 |

SHA256(서버 = 이 PC):

```
7ea3ef1eaf815ae9755bc9c1b3cb96c77d45fb767c100eaffb1f10f9d11611c3  big40m.bin     (40MB, 받기)
d06120be2342eb1240229e71f1d32a3c9f20c112b43fc264ef4c42265ae32f13  mid3m.bin      (3MB, 받기)
83c8cf4e5d96434ee2f2ad25209a9b5fca2b9e42c0a2975b0d2b2bab62321005  hostname       (15바이트, 받기)
1be6093750d2ccca597894617ab3f6c082fd703a5e535780d2d6fd446c8e0cf4  up5m.bin       (5MB, 보내기)
8d90c75f77843f3c2fc939e2cabe3ff322040da2d2250bb97757e2e516d10046  up small.txt   (24바이트, 보내기)
886715e4051e827f4fe215df3053af3f85ad0d352db2c829c7487af6d78efe30  zeros.bin      (300KB, 보내기)
```

### G.4 앱 빌드

`build/bin/dariterm.exe` 는 **다시 빌드하지 않았다**. 앱에서 확인하려면 `go run github.com/wailsapp/wails/v2/cmd/wails@v2.17.1 build` (이 PC 에는 wails CLI 가 없다).

---

## H. 수동 검증 (남은 일)

앱을 다시 빌드한 뒤, lrzsz 가 있는 서버(192.168.12.204)에 SSH 로 붙어서 한다.

| # | 절차 | 기대 결과 |
|---|---|---|
| 1 | `sz /etc/hostname` | 진행 줄 → `받음: hostname (15 B) → …\Downloads\hostname` → `ZMODEM 전송 완료`. 앞에 찍힌 `rz` 글자가 남아 보이지 않는다. 다운로드 폴더에 파일이 있다 |
| 2 | `head -c 40M /dev/urandom > big.bin; sz big.bin` → 받는 중 **Ctrl+C** | 진행 줄이 멈추고 0.5초쯤 뒤 `ZMODEM 전송을 취소했습니다`, 바로 아래에 프롬프트. **화면에 깨진 글자가 나오지 않는다.** 다운로드 폴더에 `big.bin.dtpart` 가 없다. 그 뒤 입력이 정상 |
| 3 | 2번을 세 번 되풀이(취소하는 때를 바꿔 가며) | 매번 같다. 프롬프트가 안 나온 채 멈추는 일이 없다 |
| 4 | `rz` → 파일 선택 창에서 파일 2개 | 서버 현재 폴더에 생기고 `sha256sum` 이 같다 |
| 5 | `rz` → 선택 창 취소 | `ZMODEM 전송을 취소했습니다`, 프롬프트 |
| 6 | `printf '**\030B00000000000000\r\212\021'; sleep 300` → 60초 기다림 | 60초 뒤 `서버가 60초 동안 응답하지 않아 ZMODEM 전송을 중단했습니다`. 탭은 연결된 채다(재접속하지 않는다). Ctrl+C 로 `sleep` 을 끝내면 프롬프트 |
| 7 | 6번 명령을 다시 → 곧바로 **Ctrl+C** | 1초 안에 `ZMODEM 전송을 취소했습니다` |
| 8 | `tmux` 안에서 `sz /etc/hostname` | `**B00000000000000…` 글 뒤에 "tmux·screen 안에서는 전송할 수 없습니다…" 안내가 **한 번**. Ctrl+X 를 5번 누르면 `sz` 가 끝나고 프롬프트 |
| 9 | 설정 > 파일 전송에서 ZMODEM 을 끄고 8번 | 안내가 나오지 않는다 |
| 10 | (가능하면) 느린 회선·VPN 너머 서버에서 2번 | 깨진 글자가 나오지 않는다. 나온다면 `G_durZmodemDrainQuiet`(0.5초)가 모자란 것 — §I |
| 11 | 2026-11-23 문서 §H 의 6·7·9·11·12 | 그 문서의 기대 결과 |

## I. 영향 범위·회귀 위험

| 위험 | 정도 | 대비 |
|---|---|---|
| 느린·끊기는 회선에서 0.5초 넘게 데이터가 쉬었다가 다시 온다 → 나머지가 화면에 나온다 | 낮음 | 고치기 전보다 나빠지지 않는다(전에는 전부 나왔다). 문제가 보이면 `G_durZmodemDrainQuiet` 를 늘린다 |
| 마지막 데이터 조각이 중간에서 잘려 온다(서버가 터미널 출력 버퍼를 비우며 끝남) → 256바이트 이하면 깨진 글자 조금 + 프롬프트가 나오고, 넘으면 프롬프트까지 버린다 | 낮음 | 실제 lrzsz 는 조각 끝에서 끝났다(4번 모두). 프롬프트가 안 보이면 Enter |
| 취소 직후 0.5초 안에 서버가 낸 **다른** 출력이 버려진다 | 낮음 | 취소 직후에는 죽어 가는 `sz` 의 데이터뿐이다. 그동안 입력은 서버로 가지 않으므로 새 명령의 출력일 수 없다 |
| 60초 넘게 조용한 정상 전송이 끊긴다(아주 느린 저장 장치에서 `rz` 가 파일을 닫는 중 등) | 낮음 | lrzsz 자신의 시한(10초 단위)이 먼저 걸린다. 문제가 보이면 `G_durZmodemStall` 을 늘린다 |
| 보통 출력에 `**B0` + 16진 13자가 나와 tmux 안내가 잘못 나온다 | 매우 낮음 | 안내 한 줄뿐이고 서버로 아무것도 보내지 않는다. 설정에서 ZMODEM 을 끄면 살피지 않는다 |
| 전송 중 세션을 닫는다 → 걸어 둔 읽기 고루틴 | 없음 | 연결이 닫히면 읽기가 오류로 돌아오고 고루틴이 끝난다(채널 버퍼 1칸) |
| 전송이 잘 끝나는 길 | 없음 | 읽는 바이트는 같다. 실제 서버에서 해시 재확인(§G.3) |

되돌리기: 이 문서의 패치를 거꾸로 적용한다. 설정으로 피하려면 설정 > 파일 전송 > "ZMODEM 파일 전송"을 끈다(끄면 2026-11-23 전과 같은 동작).

## J. 체크리스트

- [x] 서버에 lrzsz 설치(공식 rpm, 서명 확인), 시험 파일 정리
- [x] 검증 도구 `tools/zmodemcheck`
- [x] 받기·보내기 SHA256 같음(실제 lrzsz)
- [x] D123-2: 받다가 중단하면 흘러오는 데이터를 버린다 — 코드, 시험 2개, 실제 서버
- [x] D125: 응답 시한 60초, 조용한 서버에서 즉시 취소 — 코드, 시험 2개, 실제 서버(A1)
- [x] D125: tmux 안내 — 코드, 시험 3개, 문구 10개 언어, 실제 서버(T1·T2)
- [x] `go build`·`go vet`·`go test ./...`
- [x] 명세 §3·§7·§10, decisions D123-2·D125, known-issues KI-012 resolved, CHANGELOG, CLAUDE.md
- [x] 앱 빌드 후 수동 검증(§H) — 2026-11-25 사용자가 앱에서 확인함
- [ ] 커밋(사용자 요청 시)
