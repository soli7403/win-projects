Unicode true

####
## DariTerm Windows 설치 파일 (결정 D76). 만드는 법: 저장소에서 `make installer` 또는 `powershell -File tools/installer.ps1`.
## Wails 가 이 파일은 그대로 두고 wails_tools.nsh 만 wails.json 값으로 다시 만든다.
## 제품 정보(이름·회사·버전·저작권)는 wails.json 의 info 에서 온다.
##
## - 설치 위치: Program Files\DariTerm (관리자 권한). 게시자·저작권 표기는 YangHun Lee (wails.json info, kai 는 넣지 않음). 시작 메뉴·바탕화면 바로가기, 프로그램 추가/제거 등록.
## - WebView2 런타임이 없으면 설치한다.
## - 설치 화면 언어: 앱과 같은 10개 언어 중 Windows 표시 언어에 맞는 것을 자동으로 쓴다(없으면 영어).
## - 사용권 동의 화면: 저장소 LICENSE (tools/installer.ps1 이 resources\license.txt 로 BOM 붙여 복사).
## - 사용자 데이터(%APPDATA%\DariTerm\data: 서버 목록·설정·암호화 계정)는 제거해도 지우지 않는다.
####
!define PRODUCT_EXECUTABLE "dariterm.exe" # wails.json outputfilename
!define UNINST_KEY_NAME "DariTerm" # 프로그램 추가/제거 등록 키(게시자 이름을 붙이지 않는다)

!include "wails_tools.nsh"
!include "LogicLib.nsh"

# The version information for this two must consist of 4 parts
VIProductVersion "${INFO_PRODUCTVERSION}.0"
VIFileVersion    "${INFO_PRODUCTVERSION}.0"

VIAddVersionKey "CompanyName"     "${INFO_COMPANYNAME}"
VIAddVersionKey "FileDescription" "${INFO_PRODUCTNAME} Installer"
VIAddVersionKey "ProductVersion"  "${INFO_PRODUCTVERSION}"
VIAddVersionKey "FileVersion"     "${INFO_PRODUCTVERSION}"
VIAddVersionKey "LegalCopyright"  "${INFO_COPYRIGHT}"
VIAddVersionKey "ProductName"     "${INFO_PRODUCTNAME}"

# Enable HiDPI support. https://nsis.sourceforge.io/Reference/ManifestDPIAware
ManifestDPIAware true

!include "MUI.nsh"

!define MUI_ICON "..\icon.ico"
!define MUI_UNICON "..\icon.ico"
!define MUI_ABORTWARNING # 설치 도중 나가면 확인한다.
!define MUI_FINISHPAGE_RUN "$INSTDIR\${PRODUCT_EXECUTABLE}" # 마침 화면에 [DariTerm 실행] 체크

!insertmacro MUI_PAGE_WELCOME
!insertmacro MUI_PAGE_LICENSE "resources\license.txt"
!insertmacro MUI_PAGE_DIRECTORY
!insertmacro MUI_PAGE_INSTFILES
!insertmacro MUI_PAGE_FINISH

!insertmacro MUI_UNPAGE_CONFIRM
!insertmacro MUI_UNPAGE_INSTFILES

# 설치 화면 언어. 맨 앞(영어)이 맞는 언어가 없을 때 쓰인다. Windows 표시 언어와 맞는 것을 자동으로 고른다.
!insertmacro MUI_LANGUAGE "English"
!insertmacro MUI_LANGUAGE "Korean"
!insertmacro MUI_LANGUAGE "Japanese"
!insertmacro MUI_LANGUAGE "SimpChinese"
!insertmacro MUI_LANGUAGE "TradChinese"
!insertmacro MUI_LANGUAGE "German"
!insertmacro MUI_LANGUAGE "French"
!insertmacro MUI_LANGUAGE "Spanish"
!insertmacro MUI_LANGUAGE "PortugueseBR"
!insertmacro MUI_LANGUAGE "Russian"

# 앱이 켜져 있을 때 안내(설치·제거 모두). 영어 외에는 한국어를 함께 둔다(나머지 언어는 영어 문구).
LangString MsgAppRunning ${LANG_ENGLISH} "DariTerm is running. Close it and click Retry."
LangString MsgAppRunning ${LANG_KOREAN} "DariTerm 이 실행 중입니다. 닫은 뒤 [다시 시도]를 누르세요."
LangString MsgAppRunning ${LANG_JAPANESE} "DariTerm が実行中です。終了してから［再試行］を押してください。"
LangString MsgAppRunning ${LANG_SIMPCHINESE} "DariTerm 正在运行。请关闭后点击“重试”。"
LangString MsgAppRunning ${LANG_TRADCHINESE} "DariTerm 正在執行。請關閉後按「重試」。"
LangString MsgAppRunning ${LANG_GERMAN} "DariTerm läuft noch. Bitte schließen und auf Wiederholen klicken."
LangString MsgAppRunning ${LANG_FRENCH} "DariTerm est en cours d’exécution. Fermez-le puis cliquez sur Réessayer."
LangString MsgAppRunning ${LANG_SPANISH} "DariTerm se está ejecutando. Ciérrelo y pulse Reintentar."
LangString MsgAppRunning ${LANG_PORTUGUESEBR} "O DariTerm está em execução. Feche-o e clique em Repetir."
LangString MsgAppRunning ${LANG_RUSSIAN} "DariTerm запущен. Закройте его и нажмите «Повтор»."

## The following two statements can be used to sign the installer and the uninstaller. The path to the binaries are provided in %1
#!uninstfinalize 'signtool --file "%1"'
#!finalize 'signtool --file "%1"'

Name "${INFO_PRODUCTNAME}"
OutFile "..\..\bin\${INFO_PROJECTNAME}-${ARCH}-installer.exe" # Name of the installer's file.
InstallDir "$PROGRAMFILES64\${INFO_PRODUCTNAME}" # C:\Program Files\DariTerm
ShowInstDetails show # This will always show the installation details.

# 실행 중인 dariterm.exe 가 있으면 닫을 때까지 [다시 시도]/[취소]를 묻는다.
!macro CheckAppRunning
    retryAppRunning:
    # find 는 찾으면 0 을 돌려준다.
    nsExec::Exec 'cmd /c tasklist /FI "IMAGENAME eq ${PRODUCT_EXECUTABLE}" /NH | find /I "${PRODUCT_EXECUTABLE}"'
    Pop $0
    ${If} $0 == 0
        # 조용히 설치(/S)할 때는 묻지 않고 취소한다(/SD).
        MessageBox MB_RETRYCANCEL|MB_ICONEXCLAMATION "$(MsgAppRunning)" /SD IDCANCEL IDRETRY retryAppRunning
        Abort
    ${EndIf}
!macroend

Function .onInit
   !insertmacro wails.checkArchitecture
FunctionEnd

Function un.onInit
FunctionEnd

Section
    !insertmacro CheckAppRunning

    !insertmacro wails.setShellContext

    !insertmacro wails.webview2runtime

    # 첫 설치 파일(2026-11-14)의 옛 위치·등록을 지운다: Program Files\<옛 회사명>\DariTerm 과 그 등록 키.
    ${If} "$INSTDIR" != "$PROGRAMFILES64\kai\DariTerm"
        RMDir /r "$PROGRAMFILES64\kai\DariTerm"
        RMDir "$PROGRAMFILES64\kai"
    ${EndIf}
    SetRegView 64
    DeleteRegKey HKLM "Software\Microsoft\Windows\CurrentVersion\Uninstall\kaiDariTerm"

    SetOutPath $INSTDIR

    !insertmacro wails.files

    CreateShortcut "$SMPROGRAMS\${INFO_PRODUCTNAME}.lnk" "$INSTDIR\${PRODUCT_EXECUTABLE}"
    CreateShortCut "$DESKTOP\${INFO_PRODUCTNAME}.lnk" "$INSTDIR\${PRODUCT_EXECUTABLE}"

    !insertmacro wails.associateFiles
    !insertmacro wails.associateCustomProtocols

    !insertmacro wails.writeUninstaller
SectionEnd

Section "uninstall"
    !insertmacro CheckAppRunning

    !insertmacro wails.setShellContext

    RMDir /r "$AppData\${PRODUCT_EXECUTABLE}" # WebView2 캐시(%APPDATA%\dariterm.exe). 사용자 데이터 %APPDATA%\DariTerm 은 남긴다.

    RMDir /r $INSTDIR

    Delete "$SMPROGRAMS\${INFO_PRODUCTNAME}.lnk"
    Delete "$DESKTOP\${INFO_PRODUCTNAME}.lnk"

    !insertmacro wails.unassociateFiles
    !insertmacro wails.unassociateCustomProtocols

    !insertmacro wails.deleteUninstaller
SectionEnd
