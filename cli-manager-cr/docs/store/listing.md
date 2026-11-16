# 스토어 등록 정보 초안 (Partner Center > 스토어 등록 정보에 붙여 넣기)

> 2026-11-14. 길이 제한: 설명 10,000자, 기능 항목 각 200자·최대 20개, 짧은 설명 270자, 검색어 최대 7개.

## English (en-us)

**Product name**: DariTerm

**Short description**
A terminal for SSH, Telnet, serial and local shells, with SFTP file transfer, split panes and encrypted credential storage.

**Description**
DariTerm keeps all your server connections in one window.

Connect over SSH, Telnet or a serial port, or open a local shell (Command Prompt, PowerShell, WSL). Organize servers into folders, open several sessions side by side in split panes, and type into many sessions at once with broadcast input.

Move files with the built-in SFTP browser: drag and drop between your PC and the server.

Passwords and private keys are stored on your PC, encrypted and locked with an app lock password. Nothing is sent to the developer. Back up and restore your settings as a file, and import servers from PuTTY, SecureCRT, your SSH config or AWS EC2.

The interface is available in 10 languages: English, Korean, Japanese, Chinese (Simplified and Traditional), German, French, Spanish, Portuguese (Brazil) and Russian.

Free for personal use. Business use requires a license — contact soli97403@gmail.com.

**Features**
- SSH, Telnet, serial (COM) and local shell (cmd, PowerShell, WSL) sessions
- SFTP file browser with drag and drop
- Tabs and split panes
- Broadcast input to many sessions at once
- Server folders, search and quick connect (type user@host to connect)
- Passwords and keys stored encrypted, locked with an app lock password
- Backup and restore of all settings
- Import from PuTTY, SecureCRT, SSH config and AWS EC2
- 10 interface languages

**Search terms** (키워드, 하나씩 Enter): ssh, terminal, sftp, telnet, serial, console, ssh client — 다른 제품 이름(PuTTY 등)은 키워드에 넣지 않는다(스토어 정책)

**Copyright and trademark info**: Copyright (c) 2026 YangHun Lee

**Developer**: YangHun Lee

**Additional license terms**
Free for personal, non-commercial use. Use by or for a business, organization or government requires a paid license per user. Contact soli97403@gmail.com. The full license is shown in the app under Settings > About.

**What's new in this version** (0.2.0 — 이 버전의 새로운 기능, 1500자 이내. 전체 이력은 [CHANGELOG.md](../../CHANGELOG.md))
- Faster with heavy output: a tab printing megabytes per second no longer slows down other tabs, and output is never dropped. Terminals now render with WebGL (falls back automatically when unavailable).
- Paste with right-click or Ctrl+V is instant (was delayed by up to a second).
- New: Settings > Connection > "Keep session logs (days)" (default 30). Old session log files are deleted automatically.
- SSH: connections that drop silently (Wi-Fi change, VPN loss) are detected within 4 keep-alive intervals and reconnect automatically.
- Serial: typed characters appear immediately with local echo, even when the device sends nothing.
- Session logs: one file per session (no more mixed logs when the same server is opened twice in a minute), no leftover color codes, safer file names.
- Fixes for closing local terminals and file properties (product name, company, copyright).

## 한국어 (ko-kr)

**제품 이름**: DariTerm

**짧은 설명**
SSH·텔넷·시리얼·로컬 셸을 한 창에서 쓰는 터미널. SFTP 파일 전송, 화면 분할, 접속 정보 암호화 저장을 갖췄습니다.

**설명**
DariTerm 은 여러 서버 접속을 한 창에 모아 줍니다.

SSH, 텔넷, 시리얼 포트로 접속하거나 로컬 셸(명령 프롬프트, PowerShell, WSL)을 엽니다. 서버를 폴더로 묶고, 여러 세션을 화면 분할로 나란히 띄우고, 동시 입력으로 여러 세션에 한 번에 입력할 수 있습니다.

내장 SFTP 창에서 PC 와 서버 사이 파일을 끌어다 놓아 옮깁니다.

비밀번호와 개인 키는 암호화해 PC 안에만 저장하고 앱 잠금 비밀번호로 잠급니다. 개발자에게 아무것도 보내지 않습니다. 설정 전체를 파일로 백업·복원하고, PuTTY·SecureCRT·SSH config·AWS EC2 에서 서버를 가져올 수 있습니다.

화면은 10개 언어(한국어, 영어, 일본어, 중국어 간체·번체, 독일어, 프랑스어, 스페인어, 포르투갈어(브라질), 러시아어)를 지원합니다.

개인 사용은 무료입니다. 업무용은 라이선스가 필요합니다 — soli97403@gmail.com 으로 문의하세요.

**기능**
- SSH, 텔넷, 시리얼(COM), 로컬 셸(cmd, PowerShell, WSL) 세션
- 끌어다 놓기가 되는 SFTP 파일 창
- 탭과 화면 분할
- 여러 세션 동시 입력
- 서버 폴더, 검색, 빠른 접속(user@host 입력으로 바로 접속)
- 비밀번호·키 암호화 저장, 앱 잠금 비밀번호로 잠금
- 설정 전체 백업·복원
- PuTTY·SecureCRT·SSH config·AWS EC2 가져오기
- 10개 언어 화면

**검색어**: ssh, 터미널, sftp, 텔넷, 시리얼, 콘솔, ssh 클라이언트

**저작권 및 상표 정보**: Copyright (c) 2026 YangHun Lee

**개발자**: YangHun Lee

**추가 사용 조건**
개인의 비상업적 사용은 무료입니다. 기업·단체·공공기관에서 또는 이를 위해 사용하려면 사용자별 유료 라이선스가 필요합니다. 문의: soli97403@gmail.com. 전체 라이선스는 앱의 설정 > 정보에서 볼 수 있습니다.

**이 버전의 새로운 기능** (0.2.0)
- 많은 출력에도 빠르게: 한 탭이 초당 수 MB 를 출력해도 다른 탭이 느려지지 않고, 출력은 버리지 않습니다. 터미널을 WebGL 로 그립니다(쓸 수 없으면 자동으로 기존 방식).
- 오른쪽 클릭·Ctrl+V 붙여넣기가 바로 됩니다(최대 1초 가까이 늦던 문제 수정).
- 새 설정: 설정 > 연결 > "로그 보관 기간 (일)"(기본 30일). 오래된 세션 로그를 자동으로 지웁니다.
- SSH: 와이파이 전환·VPN 끊김처럼 조용히 끊긴 연결을 연결 유지 간격 4번 안에 알아채고 자동으로 다시 연결합니다.
- 시리얼: 로컬 에코를 켜면 장치가 아무것도 보내지 않아도 입력한 글자가 바로 보입니다.
- 세션 로그: 세션마다 파일 하나(같은 서버를 같은 분에 두 번 열어도 섞이지 않음), 색 코드 조각 제거, 안전한 파일 이름.
- 로컬 터미널 닫기와 실행 파일 속성(제품 이름·회사·저작권) 표시 수정.
