package importer

import (
	"context"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strings"
	"sync"
	"time"

	"dariterm/internal/procutil"
)

// G_cdurAwsCallTimeout 은 aws 호출 한 번의 제한 시간이다 (import.md §3).
const G_cdurAwsCallTimeout = 60 * time.Second

// AwsRunner 는 aws 인자로 호출하고 표준 출력을 돌려준다. 실패하면 표준 오류의 마지막 줄을 오류로 돌려준다.
type AwsRunner func(slArgs ...string) ([]byte, error)

// PuttySessionReader 는 PuTTY 세션 목록(키 이름 → 값 맵)을 돌려준다.
type PuttySessionReader func() (map[string]map[string]any, error)

// ScanPutty 는 PuTTY 세션을 후보로 바꾼다. 레지스트리가 없으면 빈 목록이다.
func ScanPutty(fnRead PuttySessionReader) []Candidate {
	slOut := []Candidate{}
	mapSessions, errRead := fnRead()
	if errRead != nil {
		return slOut
	}
	slNames := make([]string, 0, len(mapSessions))
	for szName := range mapSessions {
		slNames = append(slNames, szName)
	}
	sort.Strings(slNames)
	for _, szName := range slNames {
		if cand, bOk := ParsePuttySession(szName, mapSessions[szName]); bOk {
			slOut = append(slOut, cand)
		}
	}
	return slOut
}

// ScanSshConfig 은 홈의 .ssh/config 를 읽어 후보로 바꾼다. 없으면 빈 목록이다.
func ScanSshConfig(szHome string) []Candidate {
	bText, errRead := os.ReadFile(filepath.Join(szHome, ".ssh", "config"))
	if errRead != nil {
		return []Candidate{}
	}
	return ParseSSHConfig(string(bText), szHome)
}

// ScanAws 는 aws CLI 로 EC2 인스턴스를 찾아 후보로 바꾼다 (import.md §3 절차).
// 리전 목록 실패는 오류, 리전별 인스턴스 조회 실패는 그 리전만 건너뛴다.
func ScanAws(fnRun AwsRunner, szHome string) ([]Candidate, error) {
	bRegions, errRegions := fnRun("ec2", "describe-regions", "--output", "json")
	if errRegions != nil {
		return nil, errRegions
	}
	slRegions, errParse := ParseAwsRegions(bRegions)
	if errParse != nil {
		return nil, errParse
	}

	type regionResult struct {
		slInstances []AwsInstance
		mapImages   map[string]string
	}
	slResults := make([]regionResult, len(slRegions))
	var wg sync.WaitGroup
	for iIndex, szRegion := range slRegions {
		wg.Add(1)
		go func(iIndex int, szRegion string) {
			defer wg.Done()
			bInst, errInst := fnRun("ec2", "describe-instances", "--region", szRegion, "--output", "json")
			if errInst != nil {
				return
			}
			slInst, errParseInst := ParseAwsInstances(bInst, szRegion)
			if errParseInst != nil || len(slInst) == 0 {
				return
			}
			slResults[iIndex].slInstances = slInst
			slResults[iIndex].mapImages = lookupAwsImages(fnRun, szRegion, slInst)
		}(iIndex, szRegion)
	}
	wg.Wait()

	slAll := []AwsInstance{}
	mapImageNames := map[string]string{}
	for _, result := range slResults {
		slAll = append(slAll, result.slInstances...)
		for szId, szName := range result.mapImages {
			mapImageNames[szId] = szName
		}
	}
	return BuildAwsCandidates(slAll, mapImageNames, szHome), nil
}

// lookupAwsImages 는 인스턴스가 쓰는 이미지 이름을 묶음으로 조회하고, 묶음이 실패하면 하나씩 다시 조회한다.
func lookupAwsImages(fnRun AwsRunner, szRegion string, slInst []AwsInstance) map[string]string {
	mapOut := map[string]string{}
	slIds := []string{}
	mapSeen := map[string]bool{}
	for _, inst := range slInst {
		if inst.M_szImageId != "" && !mapSeen[inst.M_szImageId] {
			mapSeen[inst.M_szImageId] = true
			slIds = append(slIds, inst.M_szImageId)
		}
	}
	if len(slIds) == 0 {
		return mapOut
	}
	slArgs := append([]string{"ec2", "describe-images", "--region", szRegion, "--image-ids"}, slIds...)
	bImages, errImages := fnRun(append(slArgs, "--output", "json")...)
	if errImages == nil {
		if mapParsed, errParse := ParseAwsImages(bImages); errParse == nil {
			return mapParsed
		}
	}
	for _, szId := range slIds {
		bOne, errOne := fnRun("ec2", "describe-images", "--region", szRegion, "--image-ids", szId, "--output", "json")
		if errOne != nil {
			continue
		}
		if mapParsed, errParse := ParseAwsImages(bOne); errParse == nil {
			for szKey, szName := range mapParsed {
				mapOut[szKey] = szName
			}
		}
	}
	return mapOut
}

// RunAwsCli 는 실제 aws 실행 파일을 부르는 AwsRunner 를 만든다. szProfile 이 있으면 --profile 을 붙인다.
func RunAwsCli(szProfile string) AwsRunner {
	return func(slArgs ...string) ([]byte, error) {
		slFull := append([]string{}, slArgs...)
		if szProfile != "" {
			slFull = append(slFull, "--profile", szProfile)
		}
		ctxCall, fnCancel := context.WithTimeout(context.Background(), G_cdurAwsCallTimeout)
		defer fnCancel()
		pCmd := exec.CommandContext(ctxCall, "aws", slFull...)
		procutil.HideConsole(pCmd)
		var bStderr strings.Builder
		pCmd.Stderr = &bStderr
		bOut, errRun := pCmd.Output()
		if errRun != nil {
			return nil, lastLineError(bStderr.String(), errRun)
		}
		return bOut, nil
	}
}

// lastLineError 는 표준 오류의 마지막 줄을 오류 메시지로 쓴다 (없으면 실행 오류 문구).
func lastLineError(szStderr string, errRun error) error {
	slLines := strings.Split(strings.TrimSpace(szStderr), "\n")
	szLast := strings.TrimSpace(slLines[len(slLines)-1])
	if szLast == "" {
		return errRun
	}
	return &awsError{szMessage: szLast}
}

// awsError 는 aws CLI 가 낸 마지막 오류 줄이다.
type awsError struct {
	szMessage string
}

func (pErr *awsError) Error() string {
	return pErr.szMessage
}
