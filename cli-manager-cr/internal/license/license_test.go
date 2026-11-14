package license

import (
	"crypto/ed25519"
	"crypto/rand"
	"encoding/base64"
	"encoding/json"
	"testing"
	"time"
)

func makeKey(t *testing.T, priv ed25519.PrivateKey, mapBody map[string]any) string {
	t.Helper()
	arrBody, _ := json.Marshal(mapBody)
	szSigned := G_cszPrefix + base64.RawURLEncoding.EncodeToString(arrBody)
	return szSigned + "." + base64.RawURLEncoding.EncodeToString(ed25519.Sign(priv, []byte(szSigned)))
}

func TestCheck(t *testing.T) {
	pub, priv, _ := ed25519.GenerateKey(rand.Reader)
	tmNow := time.Date(2026, 11, 14, 12, 0, 0, 0, time.Local)
	base := map[string]any{"v": 1, "id": "DT-2026-0001", "org": "ACME", "seats": 5, "ver": 1, "exp": "", "iat": "2026-11-14"}
	szKey := makeKey(t, priv, base)

	if st := Check(szKey, "1.4.0", tmNow, pub); st.M_szState != G_cszStateValid || st.M_szOrg != "ACME" || st.M_iSeats != 5 {
		t.Fatalf("valid: %+v", st)
	}
	if st := Check(" "+szKey[:20]+"\n"+szKey[20:]+" ", "1.0.0", tmNow, pub); st.M_szState != G_cszStateValid {
		t.Fatalf("붙여 넣은 공백·줄바꿈은 무시: %+v", st)
	}
	if st := Check(szKey, "2.0.0", tmNow, pub); st.M_szState != G_cszStateVersion {
		t.Fatalf("버전 범위 밖: %+v", st)
	}
	if st := Check("", "1.0.0", tmNow, pub); st.M_szState != G_cszStateNone {
		t.Fatalf("키 없음: %+v", st)
	}
	// 내용을 바꾸면 서명이 맞지 않는다.
	otherPub, _, _ := ed25519.GenerateKey(rand.Reader)
	if st := Check(szKey, "1.0.0", tmNow, otherPub); st.M_szState != G_cszStateInvalid {
		t.Fatalf("다른 공개키: %+v", st)
	}
	if st := Check("DT1.abc.def", "1.0.0", tmNow, pub); st.M_szState != G_cszStateInvalid {
		t.Fatalf("형식 오류: %+v", st)
	}
	base["exp"] = "2026-11-14"
	if st := Check(makeKey(t, priv, base), "1.0.0", tmNow, pub); st.M_szState != G_cszStateValid {
		t.Fatalf("만료일 당일은 유효: %+v", st)
	}
	base["exp"] = "2026-11-13"
	if st := Check(makeKey(t, priv, base), "1.0.0", tmNow, pub); st.M_szState != G_cszStateExpired {
		t.Fatalf("만료: %+v", st)
	}
	base["exp"], base["ver"] = "", 0
	if st := Check(makeKey(t, priv, base), "9.0.0", tmNow, pub); st.M_szState != G_cszStateValid {
		t.Fatalf("ver 0 은 모든 버전: %+v", st)
	}
}

func TestEmbeddedPublicKey(t *testing.T) {
	if len(embeddedPublicKey()) != ed25519.PublicKeySize {
		t.Fatal("내장 공개키가 32바이트가 아니다")
	}
}
