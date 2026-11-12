# 기준 화면 (기존 앱 관찰)

> 촬영일 2026-11-10 · 창 1440×900 · 기존 앱의 화면을 **백엔드 없이 가짜 데이터로** 띄워 찍은 것(서버 4개, 폴더 2개, 스니펫 1개, 계정 1개). 터미널 본문은 비어 있다(실제 출력 없음). 글꼴은 이 PC 의 설치 글꼴에 따라 다를 수 있다.
> 구현자는 이 이미지와 `spec/ui/*.md` 의 수치를 함께 기준으로 삼는다. 이미지에서 읽어 낸 정보는 모두 명세 본문에 글로 있다(이미지가 우선하지 않는다).

| 파일 | 화면 | 관련 문서 |
|---|---|---|
| 01-lock.png | 잠금 해제 | ui/lock.md §1 |
| 02-lock-recover.png | 복구 코드로 재설정 | ui/lock.md §4 |
| 03-welcome.png | 시작 화면 | ui/shell.md §4 |
| 04-session.png | 세션 1개(운영 서버, 색 있음) | ui/shell.md, ui/panes.md |
| 05-split.png | 오른쪽 분할(칸 2개) | ui/panes.md |
| 06-quick.png | 빠른 접속 | ui/dialogs.md §2 |
| 07-host-menu.png | 서버 우클릭 메뉴 | ui/sidebar.md §7 |
| 08-sftp.png | 우측 패널 SFTP | ui/sftp-snippets.md §2 |
| 09-snippets.png | 우측 패널 스니펫 | ui/sftp-snippets.md §4 |
| 10-host-basic.png ~ 13-host-fw.png | 서버 편집 기본·인증·고급·포트 포워딩 | ui/host-editor.md |
| 14-settings-1.png ~ 21-settings-8.png | 설정 8개 구역(모양 … 정보·단축키) | ui/settings.md |
| 22-import.png | 서버 가져오기 | ui/dialogs.md §5 |
| 23-light-session.png | 라이트 테마 세션 | ui/tokens.md §1 |

## 측정으로 확인된 치수 (명세 ui/tokens.md 와 일치)
제목 표시줄 높이 38 · 탭 높이 32(최소 폭 120) · 사이드바 폭 260 · 사이드바 행 높이 30 · 사이드바 버튼 높이 28 · 검색 입력 높이 32 · 칸 머리글 높이 28 · 상태 표시줄 높이 26 · 잠금 카드 폭 400 · 빠른 접속 폭 620 · 모달 기본 폭 560, 크게 820 · 설정 내비 폭 170 · 우측 패널 폭 340 · SFTP 행 높이 26 · 컨텍스트 메뉴 폭 200.
