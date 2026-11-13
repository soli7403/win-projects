// notices 는 배포물에 들어가는 외부 구성 요소의 라이선스 고지를 모아
// THIRD_PARTY_NOTICES.md 의 생성 구역과 앱 안 복사본(frontend/src/generated)을 다시 만든다 (결정 D64).
//
// 대상:
//   - Go: 앱(main 패키지)에 실제로 링크되는 모듈. windows·linux·darwin 대상의 합집합.
//   - Go 표준 라이브러리·런타임(GOROOT/LICENSE).
//   - 프런트: frontend/package.json 의 dependencies 와 그 하위 의존성(개발용 devDependencies 제외).
//
// 사용: 저장소 루트에서 `go run ./tools/notices` (의존성을 바꾼 뒤 다시 실행한다).
package main

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strings"
)

// g_cszMarker 아래가 생성 구역이다. 위쪽(참고한 프로젝트 고지)은 손으로 쓴 내용이라 그대로 둔다.
const g_cszMarker = "<!-- 아래는 `go run ./tools/notices` 가 생성한다. 손으로 고치지 않는다. -->"

const g_cszNoticesFile = "THIRD_PARTY_NOTICES.md"

// g_cszAppCopy 는 정보 화면이 읽는 복사본이다(실행 파일만 배포해도 고지가 함께 가도록).
var g_cszAppCopy = filepath.Join("frontend", "src", "generated", "third-party-notices.md")

var g_slTargetOs = []string{"windows", "linux", "darwin"}

// component 는 고지 대상 하나다.
type component struct {
	szKind    string // "Go" 또는 "npm"
	szName    string
	szVersion string
	szLicense string   // 판별한 라이선스 이름
	slTexts   []string // LICENSE·NOTICE 원문
}

func main() {
	slComponents, err := collect()
	if err != nil {
		fmt.Fprintln(os.Stderr, "notices:", err)
		os.Exit(1)
	}
	szBody := render(slComponents)
	if err := writeNotices(szBody); err != nil {
		fmt.Fprintln(os.Stderr, "notices:", err)
		os.Exit(1)
	}
	fmt.Printf("notices: 구성 요소 %d개 → %s, %s\n", len(slComponents), g_cszNoticesFile, g_cszAppCopy)
}

func collect() ([]component, error) {
	slGo, err := goModules()
	if err != nil {
		return nil, err
	}
	pGoStd, err := goStdlib()
	if err != nil {
		return nil, err
	}
	slNpm, err := npmPackages(filepath.Join("frontend"))
	if err != nil {
		return nil, err
	}
	slAll := append([]component{pGoStd}, slGo...)
	slAll = append(slAll, slNpm...)
	return slAll, nil
}

// goModules 는 대상 OS 별로 main 패키지가 링크하는 모듈을 모아 합친다.
func goModules() ([]component, error) {
	mDirByModule := map[string][2]string{} // "path@version" → {path, dir}
	mVersion := map[string]string{}
	for _, szOs := range g_slTargetOs {
		pCmd := exec.Command("go", "list", "-deps", "-f",
			"{{if .Module}}{{if not .Module.Main}}{{.Module.Path}}\t{{.Module.Version}}\t{{.Module.Dir}}{{end}}{{end}}", ".")
		pCmd.Env = append(os.Environ(), "GOOS="+szOs, "CGO_ENABLED=0")
		arrOut, err := pCmd.Output()
		if err != nil {
			return nil, fmt.Errorf("go list (%s): %w", szOs, err)
		}
		for _, szLine := range strings.Split(strings.TrimSpace(string(arrOut)), "\n") {
			slParts := strings.Split(strings.TrimSpace(szLine), "\t")
			if len(slParts) != 3 || slParts[2] == "" {
				continue
			}
			szKey := slParts[0] + "@" + slParts[1]
			mDirByModule[szKey] = [2]string{slParts[0], slParts[2]}
			mVersion[szKey] = slParts[1]
		}
	}
	slOut := make([]component, 0, len(mDirByModule))
	for szKey, arrPathDir := range mDirByModule {
		slTexts, err := licenseTexts(arrPathDir[1])
		if err != nil {
			return nil, fmt.Errorf("%s: %w", szKey, err)
		}
		slOut = append(slOut, component{
			szKind: "Go", szName: arrPathDir[0], szVersion: mVersion[szKey],
			szLicense: classify(slTexts), slTexts: slTexts,
		})
	}
	sortComponents(slOut)
	return slOut, nil
}

// goStdlib 는 실행 파일에 들어가는 Go 런타임·표준 라이브러리의 고지다.
func goStdlib() (component, error) {
	arrRoot, err := exec.Command("go", "env", "GOROOT").Output()
	if err != nil {
		return component{}, fmt.Errorf("go env GOROOT: %w", err)
	}
	arrVer, err := exec.Command("go", "env", "GOVERSION").Output()
	if err != nil {
		return component{}, fmt.Errorf("go env GOVERSION: %w", err)
	}
	slTexts, err := licenseTexts(strings.TrimSpace(string(arrRoot)))
	if err != nil {
		return component{}, fmt.Errorf("GOROOT: %w", err)
	}
	return component{
		szKind: "Go", szName: "Go 표준 라이브러리·런타임", szVersion: strings.TrimSpace(string(arrVer)),
		szLicense: classify(slTexts), slTexts: slTexts,
	}, nil
}

// packageJson 은 package.json 에서 쓰는 필드만 읽는다.
type packageJson struct {
	M_szName            string            `json:"name"`
	M_szVersion         string            `json:"version"`
	M_mDependencies     map[string]string `json:"dependencies"`
	M_mPeerDependencies map[string]string `json:"peerDependencies"`
}

// npmPackages 는 frontend 의 실행용 의존성을 하위까지 따라가며 모은다(devDependencies 제외).
func npmPackages(szFrontendDir string) ([]component, error) {
	pRoot, err := readPackageJson(filepath.Join(szFrontendDir, "package.json"))
	if err != nil {
		return nil, err
	}
	szModules := filepath.Join(szFrontendDir, "node_modules")
	mSeen := map[string]bool{}
	slQueue := make([]string, 0, len(pRoot.M_mDependencies))
	for szName := range pRoot.M_mDependencies {
		slQueue = append(slQueue, szName)
	}
	var slOut []component
	for len(slQueue) > 0 {
		szName := slQueue[0]
		slQueue = slQueue[1:]
		// 타입 정의(@types/*)는 컴파일에만 쓰이고 번들에 들어가지 않는다.
		if mSeen[szName] || strings.HasPrefix(szName, "@types/") {
			continue
		}
		mSeen[szName] = true
		szDir := filepath.Join(szModules, filepath.FromSlash(szName))
		pPkg, err := readPackageJson(filepath.Join(szDir, "package.json"))
		if err != nil {
			return nil, fmt.Errorf("npm %s: %w (먼저 frontend 에서 npm install)", szName, err)
		}
		slTexts, err := licenseTexts(szDir)
		if err != nil {
			return nil, fmt.Errorf("npm %s: %w", szName, err)
		}
		slOut = append(slOut, component{
			szKind: "npm", szName: szName, szVersion: pPkg.M_szVersion,
			szLicense: classify(slTexts), slTexts: slTexts,
		})
		for szDep := range pPkg.M_mDependencies {
			slQueue = append(slQueue, szDep)
		}
		// react-dom 이 react 를 쓰듯 peer 의존성도 번들에 들어간다(설치되어 있을 때만).
		for szDep := range pPkg.M_mPeerDependencies {
			if _, errStat := os.Stat(filepath.Join(szModules, filepath.FromSlash(szDep), "package.json")); errStat == nil {
				slQueue = append(slQueue, szDep)
			}
		}
	}
	sortComponents(slOut)
	return slOut, nil
}

func readPackageJson(szPath string) (*packageJson, error) {
	arrData, err := os.ReadFile(szPath)
	if err != nil {
		return nil, err
	}
	var pPkg packageJson
	if err := json.Unmarshal(arrData, &pPkg); err != nil {
		return nil, fmt.Errorf("%s: %w", szPath, err)
	}
	return &pPkg, nil
}

// licenseTexts 는 폴더 바로 아래의 LICENSE·LICENCE·COPYING·NOTICE 파일 원문을 이름순으로 읽는다.
func licenseTexts(szDir string) ([]string, error) {
	slEntries, err := os.ReadDir(szDir)
	if err != nil {
		return nil, err
	}
	var slNames []string
	for _, pEntry := range slEntries {
		if pEntry.IsDir() {
			continue
		}
		szUpper := strings.ToUpper(pEntry.Name())
		for _, szPrefix := range []string{"LICENSE", "LICENCE", "COPYING", "NOTICE"} {
			if strings.HasPrefix(szUpper, szPrefix) {
				slNames = append(slNames, pEntry.Name())
				break
			}
		}
	}
	if len(slNames) == 0 {
		return nil, errors.New("라이선스 파일 없음 — 직접 확인해 생성 구역 밖에 적는다")
	}
	sort.Strings(slNames)
	slTexts := make([]string, 0, len(slNames))
	for _, szName := range slNames {
		arrData, err := os.ReadFile(filepath.Join(szDir, szName))
		if err != nil {
			return nil, err
		}
		slTexts = append(slTexts, strings.TrimSpace(strings.ReplaceAll(string(arrData), "\r\n", "\n")))
	}
	return slTexts, nil
}

// classify 는 원문에서 라이선스 종류를 어림한다(목록 표시용. 원문이 기준이다).
func classify(slTexts []string) string {
	szAll := strings.Join(slTexts, "\n")
	switch {
	case strings.Contains(szAll, "GNU LESSER GENERAL PUBLIC"):
		return "LGPL"
	case strings.Contains(szAll, "GNU GENERAL PUBLIC"):
		return "GPL"
	case strings.Contains(szAll, "Mozilla Public License"):
		return "MPL"
	case strings.Contains(szAll, "Apache License"):
		return "Apache-2.0"
	case strings.Contains(szAll, "Permission is hereby granted, free of charge"):
		return "MIT"
	case strings.Contains(szAll, "Permission to use, copy, modify, and/or distribute"):
		return "ISC"
	case strings.Contains(szAll, "Redistribution and use in source and binary forms"):
		return "BSD"
	}
	return "기타(원문 참고)"
}

func sortComponents(slItems []component) {
	sort.Slice(slItems, func(i, j int) bool { return slItems[i].szName < slItems[j].szName })
}

func render(slComponents []component) string {
	var pB strings.Builder
	pB.WriteString("## 오픈소스 구성 요소 / Open Source Components\n\n")
	pB.WriteString("DariTerm 실행 파일에는 아래 오픈소스 구성 요소가 들어 있습니다. 각 구성 요소는 아래 원문의 라이선스를 따릅니다.\n\n")
	pB.WriteString("| 종류 | 이름 | 버전 | 라이선스 |\n|---|---|---|---|\n")
	for _, pC := range slComponents {
		fmt.Fprintf(&pB, "| %s | %s | %s | %s |\n", pC.szKind, pC.szName, pC.szVersion, pC.szLicense)
	}
	for _, pC := range slComponents {
		fmt.Fprintf(&pB, "\n### %s %s\n\n", pC.szName, pC.szVersion)
		for _, szText := range pC.slTexts {
			// 원문에 ``` 가 있어도 깨지지 않도록 ~~~~ 울타리를 쓴다.
			pB.WriteString("~~~~\n" + szText + "\n~~~~\n")
		}
	}
	return pB.String()
}

// writeNotices 는 THIRD_PARTY_NOTICES.md 의 표식 아래를 바꾸고, 같은 전체 내용을 앱 복사본에 쓴다.
func writeNotices(szBody string) error {
	arrOld, err := os.ReadFile(g_cszNoticesFile)
	if err != nil {
		return fmt.Errorf("%s 를 저장소 루트에서 찾지 못함 (루트에서 실행): %w", g_cszNoticesFile, err)
	}
	szOld := strings.ReplaceAll(string(arrOld), "\r\n", "\n")
	szHead := szOld
	if iPos := strings.Index(szOld, g_cszMarker); iPos >= 0 {
		szHead = szOld[:iPos]
	}
	szFull := strings.TrimRight(szHead, "\n") + "\n\n" + g_cszMarker + "\n\n" + szBody
	if err := os.WriteFile(g_cszNoticesFile, []byte(szFull), 0o644); err != nil {
		return err
	}
	if err := os.MkdirAll(filepath.Dir(g_cszAppCopy), 0o755); err != nil {
		return err
	}
	return os.WriteFile(g_cszAppCopy, []byte(szFull), 0o644)
}
