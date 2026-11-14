package app

import (
	"encoding/json"
	"time"

	"dariterm/internal/license"
)

// registerLicense 는 업무용 라이선스 키 확인 채널이다 (D78, spec/behavior/license.md).
// license.check [키] → {state, id, org, seats, maxMajor, exp}. 키 저장은 화면이 settings.licenseKey 로 한다.
// 확인은 이 PC 안에서만 한다(서버로 보내지 않음).
func (pService *Service) registerLicense() {
	pService.pRouter.Handle("license.check", func(slArgs []json.RawMessage) (any, error) {
		szKey, errArg := argString(slArgs, 0)
		if errArg != nil {
			return nil, errArg
		}
		return license.Check(szKey, G_cszAppVersion, time.Now(), nil), nil
	})
}
