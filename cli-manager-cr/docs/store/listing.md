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

**What's new in this version** (1.1.0 — 이 버전의 새로운 기능, 1500자 이내. 전체 이력은 [CHANGELOG.md](../../CHANGELOG.md))
- Use PuTTY .ppk keys (format 2 and 3) directly, and SSH agent sign-in now also uses Pageant on Windows.
- Connect through an HTTP or SOCKS5 proxy, forward your SSH agent (ssh -A), and open a SOCKS proxy over SSH (dynamic forwarding, ssh -D).
- ZMODEM file transfer: run sz or rz on the server to download or upload files from the terminal (SSH and serial).
- Keyword highlighting: show words such as ERROR or FAIL in a color you choose.
- Serial RTS/CTS and XON/XOFF flow control now take effect. New encodings: EUC-JP, GBK and Big5.
- Session logs can put a timestamp on every line; turning logging on or off applies to open sessions at once.
- Terminal search marks the current match in red and shows matches on the scrollbar. Caps Lock warning on the lock screen.
- After an idle auto-logout on the server, DariTerm reconnects automatically (exit still closes the session).
- Store installs tell you when a new version is available and can open the Windows update dialog.
- Removed "tile all tabs" and "choose where to open", which ignored the split limits.

**What's new in version 1.0.0** (지난 버전 — 참고용, 붙여 넣지 않는다)
DariTerm 1.0 — the first full release.
- File transfers are safer: uploads, downloads, copies and edit-and-save now write to a temporary name and replace the file only when finished. If a transfer drops or is cancelled, the existing file keeps its old content and permissions.
- Servers without their own color or environment now inherit them from their folder. A server in a Production folder gets the red border and the dangerous-command check.
- The selected tab is easy to spot: the color bar on top follows the tab you select (blue if the server has no color).
- The "drop here to split" hint appears only when you drag a tab to the right edge, and no longer stays on screen after a cancelled drag.

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

**이 버전의 새로운 기능** (1.1.0)
- PuTTY 키(.ppk, 형식 2·3)를 변환 없이 쓰고, Windows 의 SSH 에이전트 로그인이 Pageant 도 씁니다.
- HTTP·SOCKS5 프록시를 거쳐 접속하고, SSH 에이전트 전달(ssh -A)과 SSH 를 거치는 SOCKS 프록시(동적 포워딩, ssh -D)를 쓸 수 있습니다.
- ZMODEM 파일 전송: 서버에서 sz·rz 를 실행하면 터미널에서 바로 받고 올립니다(SSH·시리얼).
- 키워드 강조: ERROR·FAIL 같은 글자를 고른 색으로 표시합니다.
- 시리얼 흐름 제어(RTS/CTS·XON/XOFF)가 실제로 동작합니다. 인코딩 EUC-JP·GBK·Big5 추가.
- 세션 로그 줄마다 시각을 붙일 수 있고, 출력 기록을 켜고 끄면 열린 세션에도 곧바로 적용됩니다.
- 터미널 찾기의 현재 항목이 빨강으로 보이고 스크롤 막대에 위치가 표시됩니다. 잠금 화면 Caps Lock 안내.
- 서버의 유휴 자동 로그아웃 뒤 자동으로 다시 연결합니다(exit 는 전처럼 닫힘).
- 스토어 설치본은 새 버전을 알려 주고 Windows 업데이트 확인 창을 띄울 수 있습니다.
- 창 수 한도를 무시하던 "바둑판으로"와 "열 위치 선택"을 없앴습니다.

**1.0.0 의 새로운 기능** (지난 버전 — 참고용, 붙여 넣지 않는다)
DariTerm 1.0 — 첫 정식 버전입니다.
- 파일 전송이 더 안전해졌습니다: 올리기·받기·복사·편집 저장은 임시 이름에 쓰고 끝나야 원래 파일을 바꿉니다. 전송이 끊기거나 취소돼도 기존 파일은 전 내용과 권한 그대로입니다.
- 색·환경을 정하지 않은 서버는 폴더의 색·환경을 물려받습니다. "운영" 폴더 안의 서버는 빨간 테두리와 위험 명령 확인을 받습니다.
- 선택한 탭이 잘 보입니다: 위쪽 색 막대가 선택한 탭을 따라갑니다(서버 색이 없으면 파랑).
- "여기에 놓으면 좌우로 나눕니다" 안내는 탭을 오른쪽 가장자리로 끌 때만 나오고, 끌기를 취소하면 사라집니다.
