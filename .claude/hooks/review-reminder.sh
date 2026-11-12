#!/usr/bin/env bash
# PostToolUse(Edit|Write|MultiEdit) hook.
# 방금 편집/생성된 파일이 Go 또는 C/C++ 소스면, code-defect-reviewer 서브에이전트로
# 결함(nil deref / race / 누수 / 무시된 에러 / cgo 메모리 / 락)을 검토하라는
# REMINDER 컨텍스트를 모델에 주입한다. 그 외 파일은 아무것도 출력하지 않는다.
#
# 입력: PostToolUse hook JSON (stdin). .tool_input.file_path 사용.
# 출력: {"hookSpecificOutput":{"hookEventName":"PostToolUse","additionalContext":"..."}}

input="$(cat)"
fpath="$(printf '%s' "$input" | jq -r '.tool_input.file_path // ""' 2>/dev/null)"

if [ -z "$fpath" ]; then
  exit 0
fi

case "$fpath" in
  *_test.go)
    # 테스트 파일은 결함 리뷰 대상에서 제외. 동작은 실행+ -race 로 검증.
    exit 0 ;;
  *.go)
    lang="Go" ;;
  *.cpp|*.cc|*.cxx|*.hpp|*.hh|*.h)
    lang="C/C++ (cgo 경계 포함)" ;;
  *)
    exit 0 ;;
esac

ctx="REMINDER: 방금 ${lang} 파일을 수정/생성했다 (${fpath}).
이 파일을 'code-defect-reviewer' 서브에이전트(Agent tool, subagent_type: code-defect-reviewer)로 즉시 검토하라. 프롬프트에 위 파일 경로를 넘겨라.
검토 카테고리: (1) nil/null 역참조 및 성공경로 nil-error 포맷(%v/%s/.Error(), 특히 defer 안에서 성공 시 nil 인 err 포맷 → \"<nil>\"); (2) 맵/슬라이스/공유변수 동시접근 race; (3) goroutine·파일·DB세션·소켓 핸들 누수 및 defer/Close 누락; (4) 무시된 에러(_ = / 반환 에러 미확인); (5) cgo 메모리(C.CString 후 C.free 누락, cgo.Handle Delete 누락, C++ new/delete·delete[] 누수·이중해제, 버퍼 경계); (6) 락 미해제·이중락·복사된 Mutex.
직전 턴에 여러 소스 파일을 수정했다면, 파일당 1개씩 에이전트를 한 메시지에서 병렬로 호출하라.
발견 시 프로젝트 Fix Protocol(원인→방안→승인→수정)에 따라 보고/조치하라."

jq -cn --arg ctx "$ctx" '{hookSpecificOutput:{hookEventName:"PostToolUse",additionalContext:$ctx}}'
