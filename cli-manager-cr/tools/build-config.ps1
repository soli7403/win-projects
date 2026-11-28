# 배포 빌드에 함께 넣는 값 (installer.ps1 이 불러 쓴다, 결정 D135). 스토어 패키지(msix.ps1)는 스토어에 묻으므로 쓰지 않는다.
# $DariTermUpdateRepo: 새 버전을 묻는 공개 릴리스 저장소(owner/name). 스토어 설치본은 스토어에 묻고, 그 밖의 설치본이 이 저장소의 최신 릴리스를 본다.
#   비우면 스토어 밖 설치본의 업데이트 확인이 꺼진다(D80).
$DariTermUpdateRepo = 'soli7403/dariterm-releases'
$DariTermLdflags = "-X dariterm/internal/app.G_cszUpdateRepo=$DariTermUpdateRepo"
