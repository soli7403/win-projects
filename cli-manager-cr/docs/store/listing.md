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

**What's new in this version** (0.3.0 — 이 버전의 새로운 기능, 1500자 이내. 전체 이력은 [CHANGELOG.md](../../CHANGELOG.md))
- SSH no longer asks for your password again after the screen saver or app lock: a dropped session reconnects with the credentials it signed in with (kept in memory only for that session). If that is not enough, it waits and reconnects as soon as you unlock the app.
- When the server ends the session (idle auto-logout, exit), DariTerm no longer logs back in on its own. Press Enter to reconnect. Network drops still reconnect automatically.
- File panel: sort by Name, Size or Modified by clicking the column header (click again to reverse; folders stay on top). The header stays visible while scrolling.
- Fixed rare out-of-order input when keys were sent in very quick succession (e.g. right after Korean/Japanese/Chinese input composition).

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

**이 버전의 새로운 기능** (0.3.0)
- 화면 보호기·앱 잠금 뒤에 SSH 비밀번호를 다시 묻지 않습니다. 끊긴 세션은 처음 접속에 쓴 계정으로 다시 연결합니다(그 세션의 메모리에만 보관). 그래도 안 되면 기다렸다가 앱 잠금을 풀면 곧바로 다시 연결합니다.
- 서버가 세션을 끝내면(유휴 자동 로그아웃, exit) 저절로 다시 로그인하지 않습니다. Enter 를 누르면 다시 연결합니다. 네트워크 끊김은 지금처럼 자동으로 다시 연결합니다.
- 파일 창: 머리 줄의 이름·크기·수정일을 눌러 정렬(다시 누르면 반대 순서, 폴더는 항상 위). 스크롤해도 머리 줄이 보입니다.
- 아주 빠르게 연달아 보낸 입력(한글 조합 직후의 키 등)의 순서가 드물게 바뀌던 문제를 고쳤습니다.
