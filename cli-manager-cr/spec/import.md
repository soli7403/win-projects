# 서버 가져오기 — PuTTY · SecureCRT · Xshell · ssh config · AWS · 적용 규칙

> 작성 방식: 기존 앱 동작 열람. 작성일 2026-11-10. 상태: **draft**. 화면은 ui/dialogs.md §5.

후보(candidate) 한 건: `source`(`putty`/`securecrt`/`xshell`/`ssh-config`/`aws`), `name`, 서버 부분 정보(`host`), `keyFile`(선택), `jumpVia`(경유 서버 주소, 선택), `jumpChain`(경유 서버가 여러 단계일 때 바깥부터의 주소 배열, 선택 — 2026-11-27 D153), `note`(선택), `folderPath`(넣을 폴더 아래의 하위 폴더 이름 배열, 선택 — SecureCRT·Xshell 이 씀).

## 1. PuTTY (Windows 레지스트리)
- 읽는 곳: `HKEY_CURRENT_USER\Software\SimonTatham\PuTTY\Sessions` 아래 하위 키 각각이 세션. 값은 문자열(REG_SZ/REG_EXPAND_SZ) 또는 정수(DWORD/QWORD)만 읽는다. 키가 없으면 빈 목록. 비 Windows 는 지원하지 않는다(빈 목록).
- **세션 이름**: 레지스트리 키 이름에서 `%XX`(16진 2자리)를 바이트로 복원한다. 복원한 바이트가 유효한 UTF-8 이면 그대로, 아니면 EUC-KR(CP949)로 해석.
- 건너뛰는 세션: 이름이 "Default Settings" 이고 `HostName` 이 비어 있는 것. SSH·텔넷 세션인데 `HostName` 이 비어 있는 것. (시리얼은 `HostName` 이 없어도 가져옴)
- **프로토콜**: `Protocol` 값이 `telnet` 이면 텔넷, `serial` 이면 시리얼, 그 외(없음 포함)는 SSH.
- **호스트·사용자**: `HostName` 에 `사용자@호스트` 가 들어 있으면 분리(`UserName` 이 이미 있으면 그 값 우선).
- **포트**: `PortNumber`(없으면 22, 텔넷이고 값이 없으면 23).
- **포트 포워딩**: `PortForwardings` 를 쉼표로 나눠 각 항목을 `[4|6]{L|R|D}[바인드:]포트=대상호스트:대상포트` 로 해석. **`D`(동적)는 건너뜀**. 바인드가 없으면 `127.0.0.1`, 대상 호스트가 없으면 `localhost`. 각 항목은 새 id 를 가진 포워딩.
- **인코딩**: `LineCodePage`(없으면 UTF-8) 매핑 — `UTF-8`→utf-8, `CP949`→cp949, `EUC-KR`→euc-kr, `Shift_JIS`→shift_jis, `ISO-8859-1:1998 (Latin-1, West Europe)`→latin1, 그 외는 이름에(대소문자 무시) `949` 또는 `euc-kr` 이 들어 있으면 cp949, `euc-jp` 면 euc-jp, `gbk`·`936`·`gb2312` 면 gbk, `big5`·`950` 이면 big5(뒤의 셋은 2026-11-23 D118), 아니면 utf-8.
- **터미널 종류**: `TerminalType`(비어 있거나 `xterm` 이면 `xterm-256color`).
- **연결 유지**: `PingIntervalSecs`, 없으면 `PingInterval`, 값이 0 이거나 없으면 30.
- **인증**: `PublicKeyFile` 이 있으면 `key`(후보의 keyFile = 그 경로), 아니면 `password`.
- **자동 실행 명령**: `RemoteCommand` 가 있으면 `startupCommand`.
- **시리얼 세션**: `SerialLine`(없으면 COM1)을 장치와 호스트로, 속도 `SerialSpeed`(기본 9600), 데이터 비트 `SerialDataBits`(기본 8), 패리티 `SerialParity`(0=none,1=odd,2=even,3=mark,4=space; 범위 밖은 none), 정지 비트 `SerialStopHalfbits`(3→1.5, 4→2, 그 외→1), 흐름 제어 `SerialFlowControl`(2→rtscts, 1→xonxoff, 그 외→none), Enter 전송값 CR, 로컬 에코는 `LocalEcho` 가 0 일 때만 켬(기본값 2 = 끔).
- 후보 이름 = 세션 이름, 서버 별칭 = 세션 이름.

## 1.5 SecureCRT (세션 폴더)

> 작성 방식: **원본 앱에 없는 신규 기능**(사용자 요청, D66). 형식은 공개 자료와 사용자 PC 의 SecureCRT 설정 관찰(2026-11-14, 세션 190개, 모두 SSH2)로 정했다. "(표본 미확인)" 표시는 실제 파일로 확인하지 못한 최선 추정이다.

- **읽는 곳**: 설정 폴더 아래 `Sessions` 폴더. 설정 폴더는 Windows 에서 `HKEY_CURRENT_USER\Software\VanDyke\SecureCRT` 의 문자열 값 `Config Path`, 없으면 `%APPDATA%\VanDyke\Config`. macOS 는 `~/Library/Application Support/VanDyke/SecureCRT/Config`, 그 밖은 `~/.vandyke/SecureCRT/Config`. `Sessions` 가 없으면 빈 목록.
- 사용자가 폴더를 고른 경우: 고른 폴더 안에 `Sessions` 폴더가 있으면 그것을, 없으면 고른 폴더 자체를 세션 폴더로 본다. 설정 폴더는 세션 폴더의 부모.
- **폴더 구조**: 세션 폴더 아래 하위 폴더를 모두 따라 내려간다. 확장자 `.ini`(대소문자 무시) 파일 하나가 세션 하나. 세션 이름 = 파일 이름에서 `.ini` 를 뗀 것. `folderPath` = 세션 폴더에서 그 파일까지의 폴더 이름들(최상위 세션은 빈 배열). 건너뛰는 파일: 어느 폴더에서든 `__FolderData__.ini`, 세션 폴더 바로 아래의 `Default.ini`. 순서는 폴더 경로·이름의 사전순.
- **파일 형식**: 앞의 UTF-8 BOM 은 버린다. 유효한 UTF-8 이 아니면 EUC-KR(CP949)로 해석. 줄 끝 CR 무시. 한 줄은 `{형}:"{키}"={값}`.
  - `S` 문자열, `D` 정수(8자리 16진, 예 `00000016`=22), `Z` 여러 줄 문자열 — 값은 16진 줄 수 N 이고 다음 N 줄(맨 앞 공백 한 칸을 뗌)이 내용, `B` 이진값 — 다음의 공백으로 시작하는 줄들과 함께 무시. 다른 형이거나 해석 못 하는 줄은 무시.
- **프로토콜**: `Protocol Name` 이 `SSH2`·`SSH1`(또는 비어 있음)이면 ssh, `Telnet` 이면 telnet, `Serial` 이면 serial(표본 미확인). 그 밖(RLogin, Raw, TAPI, Local Shell 등)은 건너뜀.
- 건너뛰는 세션: ssh·telnet 인데 `Hostname` 이 빈 것.
- **호스트·사용자**: `Hostname`, `Username`.
- **포트**: SSH2 는 `[SSH2] Port`, SSH1 은 `[SSH1] Port`, 텔넷은 `Port`. 없거나 0 이면 22(텔넷 23).
- **인코딩**: `Output Transformer Name` — `UTF-8`(또는 없음·`Default`)→utf-8, 이름에 `949`·`euc-kr`·`korean`(대소문자 무시)이 있으면 cp949, `euc-jp`·`eucjp` 또는 (`japanese` 와 `euc` 가 함께) 있으면 euc-jp, 그 밖에 `shift`·`japanese` 가 있으면 shift_jis, `gbk`·`936`·`gb2312` 가 있으면 gbk, `big5`·`950` 이 있으면 big5(euc-jp·gbk·big5 는 2026-11-23 D118), `8859-1`·`latin` 이 있으면 latin1, 그 외 utf-8.
- **터미널 종류**: 옮기지 않는다(적용 기본값 xterm-256color). SecureCRT 기본 에뮬레이션(VT100)을 그대로 쓰면 원격 프로그램의 색이 줄기 때문.
- **연결 유지**: `Idle NO-OP Check` 가 0 이 아니고 `Idle NO-OP Timeout` 이 양수면 그 초, 아니면 30.
- **인증·키 파일**: `Use Global Public Key` 가 0 이면 세션의 `Identity Filename V2`, 아니면(없음 포함) 설정 폴더 `SSH2.ini` 의 `Identity Filename V2`. 값이 비어 있지 않으면 인증 `key`·keyFile 은 그 경로(값에 `::` 가 있으면 그 앞까지, `%이름%`·`${이름}` 은 환경 변수로 바꾸고 `${VDS_CONFIG_PATH}` 는 설정 폴더). 비어 있으면 `password`. **저장된 비밀번호(`Password V2` 등)는 읽지 않는다.**
- **포트 포워딩**: `Port Forward Table V2`(로컬 L), `Reverse Forward Table V2`(원격 R, 표본 미확인)의 각 줄 = `|` 로 나눈 `이름|로컬|플래그|대상 호스트|대상 포트|…`. 로컬은 `포트` 또는 `주소:포트`(주소 없으면 `127.0.0.1`). 대상 호스트가 비면 `localhost`. 포트가 숫자가 아니면 그 줄은 무시. 이름·플래그·나머지 칸은 쓰지 않는다.
- **메모**: `Description` 의 줄들을 줄바꿈으로 이어 앞뒤 공백을 뗀 것이 비어 있지 않으면 `notes`.
- **경유 서버**(표본 미확인): `Firewall Name` 이 `Session:{경로}` 이면 이번에 읽은 세션 중 `{폴더/…/이름}`(구분자 `/` 또는 `\`, 대소문자 무시)이 같은 세션의 `Hostname` 을 `jumpVia` 로 둔다. 못 찾거나 `None`·빈 값이 아닌 다른 값(전역 방화벽 이름)이면 경유 없음이고 후보 `note` 에 `방화벽 설정 "{값}" 은 옮기지 않음`(다른 안내가 있으면 ` · ` 로 이어 붙인다).
- **로그인 스크립트(로그온 동작)**(2026-11-27, D150): 세션 파일의 저장 형식을 확인하지 못해 **옮기지 않는다**. `Use Login Script` 가 0 이 아니고 `Login Script V4`(없으면 V3·V2)에 빈칸이 아닌 줄이 있으면 후보 `note` 에 `로그인 스크립트(로그온 동작)는 옮기지 않음 — 서버 편집 > 자동화에서 다시 넣으세요` 를 덧붙인다. 단계는 만들지 않는다. Xshell 의 로그인 스크립트도 옮기지 않는다(형식 미확인). PuTTY 에는 이 기능이 없다.
- **시리얼 세션**(표본 미확인): `Com Port`(없으면 COM1)를 장치와 호스트로, `Baud Rate`(기본 9600), `Data Bits`(기본 8), `Parity`(0=none,1=odd,2=even,3=mark,4=space; 범위 밖 none), `Stop Bits`(0→1, 1→1.5, 2→2), 흐름 제어 none, Enter 전송값 CR, 로컬 에코 끔.
- 후보 이름·서버 별칭 = 세션 이름.

## 1.6 Xshell (내보내기 파일 · 세션 폴더)

> 작성 방식: **원본 앱에 없는 신규 기능**(사용자 요청, 2026-11-25, D128). 형식은 사용자가 준 Xshell 내보내기 파일(`.xts`, 세션 파일 164개 — 판 8.1 이 110개·7.1 이 54개, 모두 SSH·비밀번호 인증·UTF-8) 관찰로 정했다. "(표본 미확인)" 표시는 그 파일에 해당하는 값이 없어 실제로 확인하지 못한 최선 추정이다.

- **읽는 곳**(넷 중 하나):
  - **내보내기 파일**(`.xts`, Xshell 의 [파일 > 내보내기]로 만든 것): zip 이다. 그 안에서 **최상위 폴더 이름이 `Xshell`**(대소문자 무시)인 항목 아래만 읽는다(`Xftp`·`xsl`·`com` 등 다른 폴더는 보지 않는다). 디스크에 풀지 않고 읽는다. zip 으로 열 수 없으면 오류 "Xshell 내보내기 파일을 열지 못했습니다: {원인}".
    - 항목 이름은 UTF-8 표시가 없는 경우가 있다(내보낸 PC 의 코드 페이지 바이트): **유효한 UTF-8 이 아니면 EUC-KR(CP949)로 해석**한다. 구분자 `\` 는 `/` 로 본다.
    - 한도: 세션 파일 하나 1MB(넘으면 그 파일은 건너뜀), 세션 20000개, 읽은 내용 합계 64MB(넘으면 거기서 멈춤).
  - **세션 파일 하나**(`.xsh`, 2026-11-25 D129): 그 세션 하나만 읽는다. `folderPath` 는 빈 배열. 1MB 를 넘으면 빈 목록.
  - **세션 폴더**: 고른 폴더 안에 `Sessions` 폴더가 있으면 그것을, 없으면 고른 폴더 자체를 세션 폴더로 본다.
  - **이 PC 의 기본 세션 폴더**(자동 검색용, 표본 미확인): "문서" 폴더 아래 `NetSarang Computer\8\Xshell\Sessions`, 없으면 `…\7\…`, `…\6\…`, 그다음 옛 판의 `NetSarang\Xshell\Sessions` 순으로 처음 있는 것. "문서" 폴더는 Windows 에서 `HKEY_CURRENT_USER\Software\Microsoft\Windows\CurrentVersion\Explorer\User Shell Folders` 의 `Personal`(환경 변수를 펼침), 그다음 `%USERPROFILE%\Documents`. 비 Windows 는 기본 폴더가 없다(빈 목록). 없으면 빈 목록.
- **폴더 구조**: 세션 폴더(내보내기 파일에서는 `Xshell/`) 아래 하위 폴더를 모두 따라 내려간다. 확장자 `.xsh`(대소문자 무시) 파일 하나가 세션 하나. 세션 이름 = 파일 이름에서 `.xsh` 를 뗀 것. `folderPath` = 그 파일까지의 폴더 이름들(최상위 세션은 빈 배열). 폴더 이름이 빈 것·`.`·`..` 인 항목은 버린다. 다른 확장자(`default.xshf.xfs` 등)는 읽지 않는다. 순서는 폴더 경로·이름의 사전순.
- **파일 형식**: 맨 앞이 UTF-16 BOM(`FF FE` 또는 `FE FF`)이면 UTF-16 으로 읽는다(표본은 모두 `FF FE`). UTF-8 BOM 은 버리고, 그 밖에는 유효한 UTF-8 이면 그대로 아니면 EUC-KR. 줄 끝 CR 무시. INI 형식: `[구역]` 줄과 `키=값` 줄(첫 `=` 에서 나눔, 키·값의 앞뒤 공백을 뗌). 구역·키 이름은 대소문자를 가리지 않는다. 같은 구역의 같은 키가 두 번 나오면 처음 것. 그 밖의 줄은 무시.
- **프로토콜**: `[CONNECTION] Protocol` 이 `SSH`(또는 없음)이면 ssh, `TELNET` 이면 telnet(표본 미확인). 그 밖(`SERIAL`·`RLOGIN`·`SFTP`·`FTP`·`LOCAL`·`RDP` 등)은 **건너뜀**. 시리얼을 건너뛰는 까닭: 시리얼 값(`[CONNECTION:SERIAL]` 의 `BaudRate=12`·`DataBits=3`·`ComPort=0` 등)이 목록의 번호로 적혀 있는데 번호의 뜻을 확인할 표본이 없다.
- 건너뛰는 세션: `[CONNECTION] Host` 가 빈 것.
- **호스트·포트·사용자**: `[CONNECTION] Host`, `[CONNECTION] Port`(없거나 1~65535 밖이면 22, 텔넷 23), `[CONNECTION:AUTHENTICATION] UserName`.
- **메모**: `[CONNECTION] Description` 이 비어 있지 않으면 `notes`.
- **인코딩**: `[TERMINAL] CodePage`(Windows 코드 페이지 번호) — 65001→utf-8, 949→cp949, 51949→euc-kr, 932→shift_jis, 51932·20932→euc-jp, 936·54936→gbk, 950→big5, 1252·28591→latin1, 그 밖·없음→utf-8. (표본은 65001 뿐)
- **터미널 종류**: `[TERMINAL] Type` 이 비어 있거나 `xterm` 이면 옮기지 않는다(적용 기본값 xterm-256color). 다른 값은 그대로(표본 미확인).
- **연결 유지**: `[CONNECTION:KEEPALIVE] KeepAlive` 가 0 이 아니고 `KeepAliveInterval` 이 양수면 그 초, 아니면 30.
- **인증**: 기본 `password`. ssh 세션에서 `[CONNECTION:AUTHENTICATION] UserKey` 가 비어 있지 않으면 `key`(keyFile 은 없음 — 적용 규칙 §4 에 따라 서버는 `agent` 인증이 된다)이고 후보 `note` 에 `Xshell 사용자 키 "{이름}" 은 옮기지 않음 (키를 내보내 서버 편집에서 지정)`. 사용자 키가 없고 `[CONNECTION:SSH] UseAuthAgent` 가 0 이 아니면 `key`(안내 없음). (둘 다 표본 미확인) **저장된 비밀번호(`Password`)·암호문(`Passphrase`)은 읽지 않는다.**
  - 사용자 키를 옮기지 않는 까닭: Xshell 은 키를 세션 파일이 아니라 자기 키 보관소에 두고 세션에는 이름만 적는다.
- **자동 실행 명령**: ssh 세션의 `[CONNECTION:SSH] RemoteCommand` 가 비어 있지 않으면 `startupCommand`(표본 미확인).
- **포트 포워딩**(2026-11-25, D129 — 사용자가 준 세션 파일 하나(포워딩 43개, 모두 나가는 것)로 형식을 확인했다. 전에는 옮기지 않고 개수만 알렸다): ssh 세션의 `[CONNECTION:SSH] FwdReqCount` 가 항목 수 N 이고(1000 을 넘으면 1000), 항목 i(0 ~ N-1)는 같은 구역의 키 `FwdReq_{i}_{항목}` 들이다. 키들은 파일 안에서 순서 없이 흩어져 있다.
  - `Incoming`: `0`(또는 없음) = 나가는 것 → 로컬 `L`. `1` = 들어오는 것 → 원격 `R`(표본 미확인). 그 밖의 값(동적 등, 표본 미확인)은 옮기지 않는다.
  - `Port` = 듣는 포트(bindPort), `Host`·`HostPort` = 대상 주소·포트(destHost·destPort). 포트가 1~65535 밖이거나 대상 주소가 비면 그 항목은 옮기지 않는다.
  - `Source` = 듣는 주소(bindHost). 비었거나 `localhost`(대소문자 무시)면 `127.0.0.1`, 다른 값은 그대로.
  - `LocalOnly`("로컬 연결만 허용", 0·1)는 **쓰지 않는다**: 0 이어도 듣는 주소를 넓히지 않는다(가져오기만으로 이 PC 의 포트가 네트워크에 열리지 않게 한다. 필요하면 가져온 뒤 서버 편집에서 듣는 주소를 바꾼다).
  - `Description`(항목의 설명): 포워딩에는 이름을 둘 자리가 없으므로, 설명이 있는 항목마다 `{L|R} {듣는 포트} → {대상 주소}:{대상 포트}  {설명}` 한 줄을 만들어 서버 `notes` 끝에 붙인다 — 머리 줄 `포트 포워딩 이름 (Xshell):` 뒤에 줄바꿈으로 잇고, 앞에 메모(`[CONNECTION] Description`)가 있으면 빈 줄 하나를 사이에 둔다.
  - 옮기지 못한 항목이 있으면 후보 `note` 에 `포트 포워딩 {개수}개는 옮기지 않음`.
- **옮기지 않고 알리는 것**(후보 `note`, 여러 개면 ` · ` 로 잇는다): 위의 사용자 키, 위의 옮기지 못한 포워딩 수, `[CONNECTION:PROXY] Proxy` 가 비어 있지 않으면 `프록시 설정 "{이름}" 은 옮기지 않음`.
- 그 밖의 값(글꼴·색 구성·탭 색·로그·X11 전달·암호화 알고리즘 목록 등)은 읽지 않는다.
- 후보 이름·서버 별칭 = 세션 이름.

## 2. ssh config (`~/.ssh/config`, 없으면 빈 목록)
- 줄 단위로 읽는다. `#` 뒤는 주석. 빈 줄 무시. `키 값` 또는 `키=값` 형식, 키는 대소문자 무시, 값의 양끝 큰따옴표 제거.
- `Host` 줄이 새 블록을 시작한다(여러 이름은 공백으로 구분 → 이름마다 후보 하나). **`*` `?` `!` 가 들어간 이름(와일드카드·부정)은 건너뜀.** `Match` 줄은 블록을 끝내고 그 뒤 값은 어느 블록에도 속하지 않는다.
- 블록 안 옵션은 **같은 키가 처음 나온 값만** 쓴다. `LocalForward`/`RemoteForward` 는 여러 개를 모두 모으되 `[바인드:]포트 호스트:포트` 형식(IPv6 는 대괄호)으로 해석, 해석 못 하면 무시. 바인드 없으면 `127.0.0.1`.
- 후보 필드: 별칭=Host 이름, 호스트=`HostName`(없으면 Host 이름), 포트=`Port`(없으면 22), 사용자=`User`, 키 파일=`IdentityFile`(`~` 는 홈으로), 인증=키 파일이 있으면 `key` 아니면 `password`, 연결 유지=`ServerAliveInterval`(양수일 때), 메모=`ProxyJump` 가 있으면 `ProxyJump {값}`.
- **ProxyJump**(`none` 제외): 쉼표로 나눈 **모든 단계**를 바깥(이 PC 가 먼저 연결하는 서버)부터 읽는다(2026-11-27 D153, 전에는 첫 항목만). 단계마다 같은 파일의 다른 Host 이름과 같으면 그 후보의 `HostName`, 아니면 `[사용자@]호스트[:포트]` 에서 호스트만 취한다. **마지막** 단계를 후보의 경유 서버 주소(`jumpVia`)로 두고, 단계가 둘 이상이면 전체를 `jumpChain` 으로 둔다. 메모 `ProxyJump {원래 값}`.
- **ProxyCommand**(`none` 제외, 2026-11-27 D152): ProxyJump 가 **없을 때만** 후보의 `host.proxy = {type: "command", command: 원래 값}` 으로 옮긴다(둘 다 있으면 ProxyJump — OpenSSH 와 같다). 메모 `ProxyCommand {원래 값}`.

## 3. AWS EC2 (aws CLI 필요)
- 실행: `aws` 실행 파일(PATH 에서 찾음). 프로필이 있으면 `--profile`. 모든 호출은 `--output json`, **제한시간 60초**, Windows 에서 콘솔 창을 띄우지 않는다. 오류 메시지는 표준 오류의 **마지막 줄**.
- 절차: `ec2 describe-regions` 로 리전 목록(실패하면 오류 전달; 해석 실패 시 "aws CLI 출력 해석 실패") → 리전마다 **병렬로** `ec2 describe-instances`(실패한 리전은 조용히 건너뜀) → 인스턴스가 쓰는 이미지(AMI)의 이름을 `ec2 describe-images` 로 조회(묶음 조회가 실패하면 하나씩 재시도).
- **대상 인스턴스**: 상태가 `running` 또는 `stopped` 인 것. **Windows 플랫폼은 제외**.
- **이름**: 태그 `Name`(비어 있지 않으면), 없으면 인스턴스 id.
- **주소**: 공인 IP, 없으면 사설 IP, 둘 다 없으면 제외.
- **로그인 사용자**: AMI 이름에 `ubuntu`→ubuntu, `debian`→admin, `centos`→centos, `rocky`→rocky, `fedora`→fedora(소문자 비교, 위에서 아래 순서), 그 외 `ec2-user`.
- **키 파일 찾기**: 키 이름으로 `~/.ssh`, `~/Downloads`, `~/.aws` 폴더에서 `{키이름}.pem` → `{키이름}`(확장자 없음) 순으로 찾아 처음 존재하는 **파일**.
- **베스천 추정**(공인 IP 가 없는 인스턴스만): 같은 VPC 에서 공인 IP 가 있는 비-Windows 인스턴스들 중 우선순위로 고른다 — ① 이름의 "줄기"가 같고 이름에 bastion/jump/배스천(대소문자 무시)이 든 것 ② 줄기가 같은 것 ③ 이름에 bastion/jump/배스천이 든 것 ④ 첫 번째. 줄기 = 소문자 이름에서 끝의 `[-_ ]?(app|api|web|was|server|bastion|jump|경유 서버)\d*` 를 뗀 것. 고른 베스천의 **공인 IP 를 점프 호스트 주소**로 둔다. 없으면 점프 없음.
- 후보 필드: 별칭=이름, 프로토콜 ssh, 포트 22, 인증 `key`, 연결 유지 30, 메모 `AWS {리전} {인스턴스id}`(+ 키 이름이 있으면 ` (키: {키이름})`). 후보의 `note`(화면 툴팁): `{리전}` + (중지됨이면 ` · 중지됨`) + (키 이름이 있는데 파일을 못 찾으면 ` · 키 파일 없음: {키이름}.pem`).

## 4. 적용 규칙 (가져오기 버튼)
입력: 선택된 후보 목록, 폴더 이름(비어 있을 수 있음).
1. **폴더**: 이름이 있으면 **같은 이름의 최상위 폴더**를 찾아 쓰고, 없으면 새 최상위 폴더(환경 `none`, 정렬 키=현재 폴더 수)를 만든다. 이름이 비면 폴더 없음(최상위).
   - **하위 폴더(`folderPath`)**: 후보에 `folderPath` 가 있으면 위 폴더(없으면 최상위)에서 시작해 이름마다 **같은 부모 아래 같은 이름의 폴더**를 찾아 쓰고, 없으면 새로 만든다(환경 `none`, 정렬 키=그때의 폴더 수). 같은 적용 안에서 만든 폴더도 다시 찾는다. 빈 이름은 건너뛴다. 서버는 마지막 폴더에 넣는다.
2. 후보마다 서버를 만든다(새 UUID, 정렬 키=현재 서버 수에서 이어서).
   - **키 파일**: `keyFile` 이 있으면 파일을 읽어 — 읽지 못하면 건너뛰기 목록에 `{이름}: 키 파일을 읽지 못함 ({경로})` 추가. 읽었고 **볼트가 열려 있으면** 볼트에 항목(이름 `{후보 이름} 키`, 종류 key, 사용자 이름=서버 사용자)으로 저장하고 서버에 연결(키 개수 +1). 볼트가 잠겨 있으면 복사하지 않는다.
   - **환경**: 후보 이름(소문자)이 `prod`/`prd`/`live`/`운영` 으로 **시작**하면 prod, `stg`/`stage`/`staging`/`func`/`기능` 이면 stage(기능테스트), `perf`/`성능` 이면 perf(성능테스트), `dev`/`test`/`개발` 이면 dev, 그 외 프로토콜이 시리얼이면 device, 아니면 none.
   - **기본값**: 프로토콜 없으면 ssh, 포트 없으면 22, **인증**: 키 파일을 볼트에 저장했으면 `key`, 아니고 후보의 인증이 `key` 였으면 `agent`, 그 외 `password`. 별칭 없으면 후보 이름. 인코딩 없으면 utf-8, 터미널 종류 없으면 xterm-256color, 연결 유지 값이 없거나 0 이면 30, 포워딩 없으면 빈 목록.
   - **태그**: 출처 한 개(`putty`/`securecrt`/`xshell`/`ssh-config`/`aws`). **자동 실행 명령·시리얼·메모는 비어 있지 않을 때만** 복사. 후보에 로그인 스크립트 단계(`host.loginScript`, 각 `{expect, send}`)가 있으면 로컬이 아닌 서버에 `loginScript` 로 옮기고 단계마다 새 id 를 준다(2026-11-27, D150 — 지금은 이 단계를 채우는 가져오기가 없다).
3. **점프 호스트 연결**: 점프 주소(`jumpVia`)가 있는 후보마다, 이번에 새로 만든 서버 중 **호스트가 같은 것**을 먼저 찾고 없으면 기존 서버 전체에서 찾는다. 찾았고 자기 자신이 아니면 `jumpHostId` 로 연결. 못 찾으면 건너뛰기 목록에 `{이름}: 점프 호스트 {주소} 를 찾지 못해 직접 접속으로 등록` 추가.
   - **여러 단계**(`jumpChain`, D153): 그다음, 체인의 각 단계 서버(안쪽부터)가 바로 앞 단계 서버를 경유 서버로 삼게 잇는다(새 서버 → 기존 서버 순으로 주소가 같은 것). **이번에 새로 만든 서버만** 바꾼다 — 기존 서버이거나 이미 다른 경유 서버가 정해진 새 서버는 그대로 두고 건너뛰기 목록에 `{이름}: 이미 다른 경유 서버가 정해져 있거나 기존 서버라서 경유 서버를 {주소} 로 바꾸지 않음`. 앞 단계를 찾지 못하면 그 연결만 건너뛰고 위와 같은 "찾지 못해" 안내. 이미 앞 단계로 이어져 있으면 그대로.
   - 후보에 `host.proxy`(명령, D152)가 있으면 SSH 서버에 `proxy = {type: "command", command, host: "", port: 0, username: "", credentialId: null}` 로 쓴다.
4. 폴더·서버 목록을 저장한다. 결과: `{hosts: 후보 수, keys: 볼트에 저장한 키 수, skipped: 건너뛰기 메시지 배열}`.
- **중복 판정은 화면에서만** 한다(초기 선택 해제용, ui/dialogs.md §5). 적용 단계는 중복을 검사하지 않고 선택된 것을 모두 추가한다.
- `후보 검색`(import.scan)은 PuTTY 후보, 기본 위치의 SecureCRT 후보, 기본 위치의 Xshell 후보(§1.6), ssh config 후보 순으로 이어 붙인 목록을 돌려준다.
