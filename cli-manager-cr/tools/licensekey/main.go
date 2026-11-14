// licensekey 는 업무용 라이선스 키의 서명 키 쌍을 만들고, 시험용으로 키를 발급한다 (결정 D78).
// 실제 판매용 발급은 나중에 서버에서 같은 형식(internal/license 머리말 설명)으로 한다.
//
// 사용(저장소 루트):
//
//	go run ./tools/licensekey keygen  <비밀키 파일>          # 키 쌍 생성. 공개키를 화면에 출력(앱에 넣는다)
//	go run ./tools/licensekey issue   <비밀키 파일> -id DT-2026-0001 -org "회사" -seats 5 [-ver 1] [-exp 2027-12-31] [-email a@b]
//
// 비밀키 파일은 절대 저장소에 넣지 않는다. 잃어버리면 이미 판 키는 계속 쓸 수 있지만 새 키를 발급할 수 없다.
package main

import (
	"crypto/ed25519"
	"crypto/rand"
	"encoding/base64"
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"time"
)

func main() {
	if len(os.Args) < 3 {
		fmt.Fprintln(os.Stderr, "usage: licensekey keygen <keyfile> | issue <keyfile> -id ... -org ...")
		os.Exit(2)
	}
	switch os.Args[1] {
	case "keygen":
		keygen(os.Args[2])
	case "issue":
		issue(os.Args[2], os.Args[3:])
	default:
		fmt.Fprintln(os.Stderr, "unknown command:", os.Args[1])
		os.Exit(2)
	}
}

func keygen(szFile string) {
	if _, errStat := os.Stat(szFile); errStat == nil {
		fmt.Fprintln(os.Stderr, "already exists (refusing to overwrite):", szFile)
		os.Exit(1)
	}
	pub, priv, errGen := ed25519.GenerateKey(rand.Reader)
	if errGen != nil {
		panic(errGen)
	}
	if errMk := os.MkdirAll(filepath.Dir(szFile), 0o700); errMk != nil {
		panic(errMk)
	}
	if errWrite := os.WriteFile(szFile, []byte(base64.StdEncoding.EncodeToString(priv.Seed())+"\n"), 0o600); errWrite != nil {
		panic(errWrite)
	}
	fmt.Println("public key:", base64.StdEncoding.EncodeToString(pub))
}

func issue(szFile string, slArgs []string) {
	pFlags := flag.NewFlagSet("issue", flag.ExitOnError)
	szId := pFlags.String("id", "", "license id, e.g. DT-2026-0001")
	szOrg := pFlags.String("org", "", "licensee (company) name")
	iSeats := pFlags.Int("seats", 1, "number of users")
	iVer := pFlags.Int("ver", 0, "max major version (0 = all)")
	szExp := pFlags.String("exp", "", "expiry date YYYY-MM-DD (empty = perpetual)")
	szEmail := pFlags.String("email", "", "contact email")
	_ = pFlags.Parse(slArgs)
	if *szId == "" || *szOrg == "" {
		fmt.Fprintln(os.Stderr, "-id and -org are required")
		os.Exit(2)
	}
	arrSeed, errRead := os.ReadFile(szFile)
	if errRead != nil {
		panic(errRead)
	}
	arrSeedRaw, errDec := base64.StdEncoding.DecodeString(string(trimNewline(arrSeed)))
	if errDec != nil || len(arrSeedRaw) != ed25519.SeedSize {
		panic("bad key file")
	}
	priv := ed25519.NewKeyFromSeed(arrSeedRaw)
	mapBody := map[string]any{"v": 1, "id": *szId, "org": *szOrg, "seats": *iSeats, "ver": *iVer, "exp": *szExp, "iat": time.Now().Format("2006-01-02"), "email": *szEmail}
	arrBody, _ := json.Marshal(mapBody)
	szSigned := "DT1." + base64.RawURLEncoding.EncodeToString(arrBody)
	fmt.Println(szSigned + "." + base64.RawURLEncoding.EncodeToString(ed25519.Sign(priv, []byte(szSigned))))
}

func trimNewline(arrData []byte) []byte {
	for len(arrData) > 0 && (arrData[len(arrData)-1] == '\n' || arrData[len(arrData)-1] == '\r') {
		arrData = arrData[:len(arrData)-1]
	}
	return arrData
}
