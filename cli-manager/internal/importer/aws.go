package importer

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strings"
	"sync"
	"time"
)

// Ec2 는 describe-instances 결과 중 쓰는 필드.
type Ec2 struct {
	InstanceId       string `json:"InstanceId"`
	ImageId          string `json:"ImageId"`
	State            struct{ Name string }
	PublicIpAddress  string
	PrivateIpAddress string
	KeyName          string
	VpcId            string
	Platform         string
	Tags             []struct{ Key, Value string }
}

func (e Ec2) name() string {
	for _, t := range e.Tags {
		if t.Key == "Name" && t.Value != "" {
			return t.Value
		}
	}
	return e.InstanceId
}

// UserForImage 는 AMI 이름으로 기본 로그인 계정을 고른다.
func UserForImage(szName string) string {
	n := strings.ToLower(szName)
	for _, p := range [][2]string{{"ubuntu", "ubuntu"}, {"debian", "admin"}, {"centos", "centos"}, {"rocky", "rocky"}, {"fedora", "fedora"}} {
		if strings.Contains(n, p[0]) {
			return p[1]
		}
	}
	return "ec2-user"
}

// FindKeyFile 은 키 이름의 .pem 을 흔한 폴더에서 찾는다.
func FindKeyFile(szKeyName string) string {
	if szKeyName == "" {
		return ""
	}
	szHome, _ := os.UserHomeDir()
	for _, d := range []string{filepath.Join(szHome, ".ssh"), filepath.Join(szHome, "Downloads"), filepath.Join(szHome, ".aws")} {
		for _, ext := range []string{".pem", ""} {
			p := filepath.Join(d, szKeyName+ext)
			if st, err := os.Stat(p); err == nil && !st.IsDir() {
				return p
			}
		}
	}
	return ""
}

var reStem = regexp.MustCompile(`[-_ ]?(app|api|web|was|server|bastion|jump|배스천)\d*$`)
var reBastion = regexp.MustCompile(`(?i)bastion|jump|배스천`)

// Ec2ToCandidates 는 인스턴스를 후보로 바꾼다. 공인 IP 가 없으면 같은 VPC 의 베스천을 점프 호스트로 잇는다.
func Ec2ToCandidates(szRegion string, all []Ec2, mImage map[string]string) []Candidate {
	live := []Ec2{}
	for _, i := range all {
		if i.State.Name == "running" || i.State.Name == "stopped" {
			live = append(live, i)
		}
	}
	stem := func(n string) string { return reStem.ReplaceAllString(strings.ToLower(n), "") }
	bastionOf := func(t Ec2) *Ec2 {
		var pub []Ec2
		for _, i := range live {
			if i.VpcId == t.VpcId && i.PublicIpAddress != "" && i.Platform != "windows" {
				pub = append(pub, i)
			}
		}
		pick := func(f func(Ec2) bool) *Ec2 {
			for k := range pub {
				if f(pub[k]) {
					return &pub[k]
				}
			}
			return nil
		}
		same := func(i Ec2) bool { return stem(i.name()) == stem(t.name()) }
		isB := func(i Ec2) bool { return reBastion.MatchString(i.name()) }
		if b := pick(func(i Ec2) bool { return same(i) && isB(i) }); b != nil {
			return b
		}
		if b := pick(same); b != nil {
			return b
		}
		if b := pick(isB); b != nil {
			return b
		}
		if len(pub) > 0 {
			return &pub[0]
		}
		return nil
	}
	out := []Candidate{}
	for _, i := range live {
		if i.Platform == "windows" {
			continue
		}
		szJump := ""
		if i.PublicIpAddress == "" {
			if b := bastionOf(i); b != nil {
				szJump = b.PublicIpAddress
			}
		}
		szAddr := i.PublicIpAddress
		if szAddr == "" {
			szAddr = i.PrivateIpAddress
		}
		if szAddr == "" {
			continue
		}
		szKey := FindKeyFile(i.KeyName)
		notes := []string{szRegion}
		if i.State.Name == "stopped" {
			notes = append(notes, "중지됨")
		}
		if i.KeyName != "" && szKey == "" {
			notes = append(notes, "키 파일 없음: "+i.KeyName+".pem")
		}
		szNotes := fmt.Sprintf("AWS %s %s", szRegion, i.InstanceId)
		if i.KeyName != "" {
			szNotes += fmt.Sprintf(" (키: %s)", i.KeyName)
		}
		out = append(out, Candidate{
			Source: "aws", Name: i.name(), KeyFile: szKey, JumpVia: szJump, Note: strings.Join(notes, " · "),
			Host: map[string]any{
				"alias": i.name(), "protocol": "ssh", "host": szAddr, "port": 22, "username": UserForImage(mImage[i.ImageId]),
				"authType": "key", "keepaliveSec": 30, "notes": szNotes,
			},
		})
	}
	return out
}

func awsBin() string {
	if p, err := exec.LookPath("aws"); err == nil {
		return p
	}
	return "aws"
}

func awsRun(szProfile string, args ...string) ([]byte, error) {
	a := append(append([]string{}, args...), "--output", "json")
	if szProfile != "" {
		a = append(a, "--profile", szProfile)
	}
	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()
	cmd := exec.CommandContext(ctx, awsBin(), a...)
	hideWindow(cmd)
	var so, se bytes.Buffer
	cmd.Stdout, cmd.Stderr = &so, &se
	if err := cmd.Run(); err != nil {
		msg := strings.TrimSpace(se.String())
		if msg == "" {
			msg = err.Error()
		}
		if i := strings.LastIndex(msg, "\n"); i >= 0 {
			msg = msg[i+1:]
		}
		return nil, errors.New(msg)
	}
	return so.Bytes(), nil
}

// ScanAws 는 aws CLI 로 모든 리전의 EC2 인스턴스를 읽는다.
func ScanAws(szProfile string) ([]Candidate, error) {
	arr, err := awsRun(szProfile, "ec2", "describe-regions", "--query", "Regions[].RegionName")
	if err != nil {
		return nil, err
	}
	var regions []string
	if err := json.Unmarshal(arr, &regions); err != nil {
		return nil, errors.New("aws CLI 출력 해석 실패")
	}
	var mu sync.Mutex
	var wg sync.WaitGroup
	out := []Candidate{}
	for _, region := range regions {
		wg.Add(1)
		go func(szRegion string) {
			defer wg.Done()
			arr, err := awsRun(szProfile, "ec2", "describe-instances", "--region", szRegion, "--query", "Reservations[].Instances[]")
			if err != nil {
				return // 비활성 리전 등
			}
			var inst []Ec2
			if json.Unmarshal(arr, &inst) != nil || len(inst) == 0 {
				return
			}
			mImg := map[string]string{}
			ids := map[string]bool{}
			for _, i := range inst {
				ids[i.ImageId] = true
			}
			list := []string{}
			for id := range ids {
				list = append(list, id)
			}
			fill := func(ids []string) error {
				a, err := awsRun(szProfile, append([]string{"ec2", "describe-images", "--region", szRegion, "--image-ids"}, append(ids, "--query", "Images[].[ImageId,Name]")...)...)
				if err != nil {
					return err
				}
				var imgs [][]string
				_ = json.Unmarshal(a, &imgs)
				for _, p := range imgs {
					if len(p) >= 2 {
						mImg[p[0]] = p[1]
					}
				}
				return nil
			}
			if fill(list) != nil {
				for _, id := range list {
					_ = fill([]string{id})
				}
			}
			c := Ec2ToCandidates(szRegion, inst, mImg)
			mu.Lock()
			out = append(out, c...)
			mu.Unlock()
		}(region)
	}
	wg.Wait()
	return out, nil
}
