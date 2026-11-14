# notices: 의존성을 바꾼 뒤 오픈소스 고지(THIRD_PARTY_NOTICES.md + 앱 복사본)를 다시 만든다 (D64).
.PHONY: build test vet tidy notices

build:
	go build ./...

test:
	go test -race ./...

vet:
	go vet ./...

tidy:
	go mod tidy

notices:
	go run ./tools/notices

# installer: Windows 설치 파일 build/bin/DariTerm-amd64-installer.exe 를 만든다 (D76, NSIS 필요 — docs/host-settings/01_installer-build.md).
.PHONY: installer
installer:
	powershell -NoProfile -ExecutionPolicy Bypass -File tools/installer.ps1

# msix: Microsoft Store 업로드용 패키지 build/bin/DariTerm-<버전>.0.msix 를 만든다 (D81, Windows SDK 필요 — docs/host-settings/03_msix.md).
#   make msix VERSION=0.1.1   → wails.json 버전을 올린 뒤 만든다(D82). 결과 파일을 Partner Center > 제출 업데이트 > 패키지에 올린다.
# 이 PC 에서 시험 설치할 자체 서명본까지: powershell -File tools/msix.ps1 -TestSign
.PHONY: msix
msix:
	powershell -NoProfile -ExecutionPolicy Bypass -File tools/msix.ps1 $(if $(VERSION),-Version $(VERSION),)

# store-publish: 스토어 업데이트 자동 제출 — 버전 올리기 → MSIX → 업로드 → 인증 요청 (D82, msstore CLI·자격 증명 필요 — docs/store/README.md §3).
#   make store-publish VERSION=0.1.1     (VERSION 을 비우면 지금 버전 그대로 다시 올린다)
.PHONY: store-publish
store-publish:
	powershell -NoProfile -ExecutionPolicy Bypass -File tools/store-publish.ps1 $(if $(VERSION),-Version $(VERSION),)
