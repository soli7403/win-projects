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
