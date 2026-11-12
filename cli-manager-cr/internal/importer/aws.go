package importer

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"
)

// AwsInstance 은 describe-instances 에서 쓰는 인스턴스 정보다.
type AwsInstance struct {
	M_szRegion    string
	M_szId        string
	M_szName      string
	M_szState     string
	M_szPlatform  string
	M_szKeyName   string
	M_szPublicIp  string
	M_szPrivateIp string
	M_szVpcId     string
	M_szImageId   string
}

// g_reAwsRoleSuffix 는 이름 끝의 역할 접미(app, web, bastion 등 + 숫자)를 찾는다.
var g_reAwsRoleSuffix = regexp.MustCompile(`[-_ ]?(app|api|web|was|server|bastion|jump|배스천)\d*$`)

// ParseAwsRegions 는 describe-regions 출력에서 리전 이름 목록을 뽑는다.
func ParseAwsRegions(bData []byte) ([]string, error) {
	var mapOut struct {
		Regions []struct {
			RegionName string `json:"RegionName"`
		} `json:"Regions"`
	}
	if errJson := json.Unmarshal(bData, &mapOut); errJson != nil {
		return nil, fmt.Errorf("aws CLI 출력 해석 실패")
	}
	slOut := []string{}
	for _, region := range mapOut.Regions {
		if region.RegionName != "" {
			slOut = append(slOut, region.RegionName)
		}
	}
	return slOut, nil
}

// ParseAwsInstances 는 describe-instances 출력을 인스턴스 목록으로 바꾼다 (상태 running·stopped 만 남긴다).
func ParseAwsInstances(bData []byte, szRegion string) ([]AwsInstance, error) {
	var mapOut struct {
		Reservations []struct {
			Instances []struct {
				InstanceId string `json:"InstanceId"`
				ImageId    string `json:"ImageId"`
				KeyName    string `json:"KeyName"`
				Platform   string `json:"Platform"`
				State      struct {
					Name string `json:"Name"`
				} `json:"State"`
				PublicIpAddress  string `json:"PublicIpAddress"`
				PrivateIpAddress string `json:"PrivateIpAddress"`
				VpcId            string `json:"VpcId"`
				Tags             []struct {
					Key   string `json:"Key"`
					Value string `json:"Value"`
				} `json:"Tags"`
			} `json:"Instances"`
		} `json:"Reservations"`
	}
	if errJson := json.Unmarshal(bData, &mapOut); errJson != nil {
		return nil, fmt.Errorf("aws CLI 출력 해석 실패")
	}
	slOut := []AwsInstance{}
	for _, reservation := range mapOut.Reservations {
		for _, raw := range reservation.Instances {
			if raw.State.Name != "running" && raw.State.Name != "stopped" {
				continue
			}
			inst := AwsInstance{
				M_szRegion:    szRegion,
				M_szId:        raw.InstanceId,
				M_szState:     raw.State.Name,
				M_szPlatform:  strings.ToLower(raw.Platform),
				M_szKeyName:   raw.KeyName,
				M_szPublicIp:  raw.PublicIpAddress,
				M_szPrivateIp: raw.PrivateIpAddress,
				M_szVpcId:     raw.VpcId,
				M_szImageId:   raw.ImageId,
			}
			for _, tag := range raw.Tags {
				if tag.Key == "Name" && tag.Value != "" {
					inst.M_szName = tag.Value
				}
			}
			if inst.M_szName == "" {
				inst.M_szName = inst.M_szId
			}
			slOut = append(slOut, inst)
		}
	}
	return slOut, nil
}

// ParseAwsImages 는 describe-images 출력에서 이미지 id → 이름 표를 만든다.
func ParseAwsImages(bData []byte) (map[string]string, error) {
	var mapOut struct {
		Images []struct {
			ImageId string `json:"ImageId"`
			Name    string `json:"Name"`
		} `json:"Images"`
	}
	if errJson := json.Unmarshal(bData, &mapOut); errJson != nil {
		return nil, fmt.Errorf("aws CLI 출력 해석 실패")
	}
	mapOut2 := map[string]string{}
	for _, image := range mapOut.Images {
		mapOut2[image.ImageId] = image.Name
	}
	return mapOut2, nil
}

// AwsLoginUser 는 AMI 이름으로 로그인 사용자를 정한다 (import.md §3, 소문자 포함 검사, 위에서 아래 순서).
func AwsLoginUser(szImageName string) string {
	szLower := strings.ToLower(szImageName)
	slRules := []struct{ szNeedle, szUser string }{
		{"ubuntu", "ubuntu"},
		{"debian", "admin"},
		{"centos", "centos"},
		{"rocky", "rocky"},
		{"fedora", "fedora"},
	}
	for _, rule := range slRules {
		if strings.Contains(szLower, rule.szNeedle) {
			return rule.szUser
		}
	}
	return "ec2-user"
}

// awsStem 은 소문자 이름에서 끝의 역할 접미를 뗀 줄기다.
func awsStem(szName string) string {
	return g_reAwsRoleSuffix.ReplaceAllString(strings.ToLower(szName), "")
}

// awsIsBastionName 은 이름에 bastion/jump/배스천 이 있는지 본다 (대소문자 무시).
func awsIsBastionName(szName string) bool {
	szLower := strings.ToLower(szName)
	return strings.Contains(szLower, "bastion") || strings.Contains(szLower, "jump") || strings.Contains(szLower, "배스천")
}

// PickAwsBastion 은 공인 IP 없는 대상의 같은 VPC 베스천을 고른다 (import.md §3 우선순위). 없으면 nil.
func PickAwsBastion(target AwsInstance, slAll []AwsInstance) *AwsInstance {
	if target.M_szVpcId == "" {
		return nil
	}
	slPool := []AwsInstance{}
	for _, inst := range slAll {
		if inst.M_szVpcId == target.M_szVpcId && inst.M_szPublicIp != "" && inst.M_szPlatform != "windows" && inst.M_szId != target.M_szId {
			slPool = append(slPool, inst)
		}
	}
	if len(slPool) == 0 {
		return nil
	}
	szStem := awsStem(target.M_szName)
	fnPick := func(fnMatch func(AwsInstance) bool) *AwsInstance {
		for iIndex := range slPool {
			if fnMatch(slPool[iIndex]) {
				return &slPool[iIndex]
			}
		}
		return nil
	}
	if pPick := fnPick(func(inst AwsInstance) bool {
		return awsStem(inst.M_szName) == szStem && awsIsBastionName(inst.M_szName)
	}); pPick != nil {
		return pPick
	}
	if pPick := fnPick(func(inst AwsInstance) bool { return awsStem(inst.M_szName) == szStem }); pPick != nil {
		return pPick
	}
	if pPick := fnPick(func(inst AwsInstance) bool { return awsIsBastionName(inst.M_szName) }); pPick != nil {
		return pPick
	}
	return &slPool[0]
}

// FindAwsKeyFile 은 키 이름으로 ~/.ssh, ~/Downloads, ~/.aws 에서 {키}.pem 과 {키} 를 차례로 찾는다 (처음 있는 파일).
func FindAwsKeyFile(szKeyName string, szHome string) string {
	if szKeyName == "" {
		return ""
	}
	for _, szDir := range []string{".ssh", "Downloads", ".aws"} {
		for _, szName := range []string{szKeyName + ".pem", szKeyName} {
			szPath := filepath.Join(szHome, szDir, szName)
			if info, errStat := os.Stat(szPath); errStat == nil && !info.IsDir() {
				return szPath
			}
		}
	}
	return ""
}

// BuildAwsCandidates 는 인스턴스 목록으로 후보 목록을 만든다 (import.md §3). szHome 은 키 파일 찾기 기준이다.
func BuildAwsCandidates(slInstances []AwsInstance, mapImageNames map[string]string, szHome string) []Candidate {
	slOut := []Candidate{}
	for _, inst := range slInstances {
		szAddress := inst.M_szPublicIp
		szJump := ""
		if szAddress == "" {
			szAddress = inst.M_szPrivateIp
			if pBastion := PickAwsBastion(inst, slInstances); pBastion != nil {
				szJump = pBastion.M_szPublicIp
			}
		}
		if szAddress == "" {
			continue
		}
		szKeyFile := FindAwsKeyFile(inst.M_szKeyName, szHome)
		szNote := inst.M_szRegion
		if inst.M_szState == "stopped" {
			szNote += " · 중지됨"
		}
		if inst.M_szKeyName != "" && szKeyFile == "" {
			szNote += " · 키 파일 없음: " + inst.M_szKeyName + ".pem"
		}
		szMemo := "AWS " + inst.M_szRegion + " " + inst.M_szId
		if inst.M_szKeyName != "" {
			szMemo += " (키: " + inst.M_szKeyName + ")"
		}
		cand := Candidate{
			M_szSource:  G_cszSourceAws,
			M_szName:    inst.M_szName,
			M_szKeyFile: szKeyFile,
			M_szJumpVia: szJump,
			M_szNote:    szNote,
			M_host: HostDraft{
				M_szAlias:       inst.M_szName,
				M_szProtocol:    "ssh",
				M_szHost:        szAddress,
				M_iPort:         22,
				M_szUsername:    AwsLoginUser(mapImageNames[inst.M_szImageId]),
				M_szAuthType:    "key",
				M_iKeepaliveSec: 30,
				M_szNotes:       szMemo,
			},
		}
		slOut = append(slOut, cand)
	}
	return slOut
}
