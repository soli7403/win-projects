// Package uid 는 UUID v4 문자열을 만든다 (store·볼트·가져오기 공용).
package uid

import (
	"crypto/rand"
	"fmt"
)

// New 는 UUID v4 문자열을 돌려준다.
func New() (string, error) {
	bUuid := make([]byte, 16)
	if _, errRand := rand.Read(bUuid); errRand != nil {
		return "", errRand
	}
	bUuid[6] = (bUuid[6] & 0x0f) | 0x40
	bUuid[8] = (bUuid[8] & 0x3f) | 0x80
	return fmt.Sprintf("%x-%x-%x-%x-%x", bUuid[0:4], bUuid[4:6], bUuid[6:8], bUuid[8:10], bUuid[10:16]), nil
}
