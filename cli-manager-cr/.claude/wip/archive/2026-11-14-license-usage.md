# 사용 용도 선택 · 회사 PC 안내 · 라이선스 키 적용(검증만)

**Status**: done
**Started**: 2026-11-14 13:20
**Last update**: 2026-11-14 13:20
**Active agent**: main
**Branch**: main

---

## Context

- 사용자 요청: "2(첫 실행 용도 선택), 3(회사 PC 추정 안내) 적용. 라이선스 키는 발급 없이 적용하는 코드만. 발급은 나중에 서버에서. 키는 회사 이름·번호·사용 가능 버전 정도 — 뭐가 더 있나?"
- 결정(D78, 권장안):
  - settings.usage: '' | 'personal' | 'business', settings.usageSince(ms, 업무용 고른 시각), settings.licenseKey, settings.nagShownAt(ms), settings.domainNoticeDone(bool)
  - 첫 실행: 언어 선택 다음 화면 "어떻게 사용하시나요?" [개인용] [업무용(30일 평가)]. 유효한 키가 있으면 건너뜀.
  - 업무용·키 없음·30일 지남 → 하루 한 번 안내(라이선스 입력/구매 문의/나중에).
  - 개인용인데 도메인(AD/Azure AD) 가입 PC → 한 번 안내([업무용으로 바꾸기]/[개인용이 맞습니다]). 판단은 로컬, 아무것도 보내지 않음.
  - 키: `DT1.<base64url(JSON)>.<base64url(Ed25519 서명)>`, JSON {v:1,id,org,seats,ver(최대 주 버전, 0=전부),exp("YYYY-MM-DD" 또는 ""),iat,email}. 공개키는 앱에, 비밀키는 저장소 밖(%USERPROFILE%\.dariterm-signing\license-ed25519.key).
  - 백엔드 internal/license (Parse/Verify), IPC license.status / license.apply / license.remove, app.info.domainJoined.
  - 개발용 발급 도구 tools/licensekey (keygen/issue) — 시험용, 서버는 나중.

## Plan

- [x] 1. 키 쌍 생성 도구 + 비밀키 저장소 밖 보관 + 공개키 상수
- [x] 2. internal/license (Parse, Verify, Status) + 테스트
- [x] 3. app: license.* 채널, settings 기본값, domainJoined(Windows NetGetJoinInformation + AAD), 계약 고정 파일
- [x] 4. 프런트: 용도 선택 화면, 안내(30일·회사 PC), 정보 화면 라이선스 상태·입력·제거, 언어 파일 10개
- [x] 5. 테스트·빌드·설치 파일, 명세(behavior/license.md), D78, host-settings(비밀키 위치 기록, 값 없음)

## Files touched

## Resume hint

1단계부터.
