# tools/ — skill 보조 스크립트 보관소

`.claude/skills/` 의 절차가 호출하는 실행 스크립트(주로 Python/bash)를 여기에 둔다.

- skill 은 "무엇을 왜 어떻게" 하는 절차(문서), tools 는 "실제 스캔/변환"을 하는 실행부.
- 예: `find_unguarded_log.py`, `audit_naming.py` 등 코드베이스 전체 스캔 스크립트.
- 스크립트는 프로젝트 루트 기준 상대 경로로 동작하도록 작성 (`.claude/tools/foo.py <path>`).
