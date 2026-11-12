#!/usr/bin/env bash
# commands-history.md 기록 훅 (프로젝트/.claude 변경 시만, 무엇이 어떻게 바뀌었는지까지 기록)
#
# 동작:
#   pre  (UserPromptSubmit): 프롬프트에서 하네스 주입 블록(ide_opened_file /
#         task-notification / system-reminder 등)을 제거해 세션별 임시파일에 stash +
#         현재 git 상태(HEAD + dirty/untracked 파일 목록과 각 mtime) 스냅샷 저장.
#   post (Stop): 턴 종료 시 스냅샷을 재비교. HEAD 변경(commit) 또는 working tree
#         변경(파일 추가/수정/삭제)이 있으면, 명령 + "변경 파일 목록"을 history 에 append.
#
# 사용: cmdhist.sh pre    (stdin = UserPromptSubmit JSON)
#       cmdhist.sh post   (stdin = Stop JSON)
#
# 이식성: REPO 는 $CLAUDE_PROJECT_DIR → git toplevel → pwd 순으로 자동 탐지.

set -u

MODE="${1:-}"
REPO="${CLAUDE_PROJECT_DIR:-$(git rev-parse --show-toplevel 2>/dev/null || pwd)}"
HIST="$REPO/docs/commands-history.md"
STATE="${TMPDIR:-/tmp}/claude-cmdhist"
MAX_FILES=40   # 변경 파일 목록 최대 표시 수 (초과 시 "... (+N more)")
mkdir -p "$STATE" 2>/dev/null || true

IN="$(cat)"
SID="$(printf '%s' "$IN" | jq -r '.session_id // "nosess"')"

# HEAD 커밋 해시 (없으면 빈 값)
git_head() { git -C "$REPO" rev-parse HEAD 2>/dev/null || true; }

# working tree 스냅샷: "<code>\t<path>\t<mtime>" 라인들.
# dirty(tracked 변경) + untracked(-uall) 를 모두 포함하되 ignored 는 제외.
# 자기 자신(history 파일)은 제외해 self-reference 노이즈 방지.
git_worktree() {
  git -C "$REPO" -c core.quotepath=false status --porcelain -uall 2>/dev/null \
  | while IFS= read -r line; do
      code="${line:0:2}"
      path="${line:3}"
      case "$path" in *" -> "*) path="${path##* -> }";; esac   # rename → new 경로
      [ "$path" = "docs/commands-history.md" ] && continue
      if [ -e "$REPO/$path" ]; then
        mt="$(stat -c '%Y' "$REPO/$path" 2>/dev/null || stat -f '%m' "$REPO/$path" 2>/dev/null || echo 0)"
      else
        mt="0"
      fi
      printf '%s\t%s\t%s\n' "$code" "$path" "$mt"
    done
}

case "$MODE" in
  pre)
    PROMPT="$(printf '%s' "$IN" | jq -r '.prompt // ""')"

    # 하네스가 주입하는 블록 제거 후, 앞뒤 공백/빈 줄 정리
    CLEAN="$(printf '%s' "$PROMPT" | perl -0777 -pe '
      s{<ide_opened_file>.*?</ide_opened_file>}{}gs;
      s{<ide_selection>.*?</ide_selection>}{}gs;
      s{<task-notification>.*?</task-notification>}{}gs;
      s{<system-reminder>.*?</system-reminder>}{}gs;
      s{<local-command-[^>]*>.*?</local-command-[^>]*>}{}gs;
      s{<command-[^>]*>.*?</command-[^>]*>}{}gs;
      s{\A\s+}{}s;
      s{\s+\z}{}s;
    ')"

    # 순수 알림/빈 프롬프트면 이번 턴은 기록 대상 아님 → stash 정리 후 종료
    if [ -z "$CLEAN" ]; then
      rm -f "$STATE/$SID".{prompt,ts,head,tree} 2>/dev/null || true
      exit 0
    fi

    printf '%s' "$CLEAN"          > "$STATE/$SID.prompt"
    date '+%Y-%m-%d %H:%M:%S'     > "$STATE/$SID.ts"
    git_head                      > "$STATE/$SID.head"
    git_worktree                  > "$STATE/$SID.tree"
    exit 0
    ;;

  post)
    [ -f "$STATE/$SID.prompt" ] || exit 0

    BEFORE_HEAD="$(cat "$STATE/$SID.head" 2>/dev/null)"
    AFTER_HEAD="$(git_head)"
    AFTER_TREE="$(git_worktree)"
    BEFORE_TREE_FILE="$STATE/$SID.tree"

    HEAD_CHANGED=0
    [ -n "$AFTER_HEAD" ] && [ "$BEFORE_HEAD" != "$AFTER_HEAD" ] && HEAD_CHANGED=1

    # ---- working tree 델타 계산 (mtime 기준) ----
    # 결과를 CHANGES 에 "<mark> <path>" 라인으로 누적.
    # HEAD 변경(commit) 턴에는 "cleaned"(dirty→사라짐) 라인을 억제 — commit diff 가 이미 그 파일들을 나열하므로 중복 방지.
    CHANGES="$(
      awk -F'\t' -v beforef="$BEFORE_TREE_FILE" -v hchg="$HEAD_CHANGED" '
        BEGIN {
          while ((getline l < beforef) > 0) {
            n = split(l, a, "\t"); if (n>=3) { bmt[a[2]] = a[3]; bseen[a[2]] = 1 }
          }
        }
        {
          code=$1; path=$2; mt=$3; aseen[path]=1; acode[path]=code; amt[path]=mt
        }
        END {
          for (p in aseen) {
            c = acode[p]; gsub(/ /,"",c)
            if (!(p in bseen)) {
              # 이전 스냅샷에 없던 경로 = 이번 턴에 새로 dirty/untracked 가 됨
              if (c == "??") mark="+ new "
              else if (c ~ /D/) mark="D del "
              else mark="M edit"
              print mark "\t" p
            } else if (amt[p] != bmt[p]) {
              # 이전에도 dirty 였으나 이번 턴에 다시 수정됨(mtime 변화)
              if (c ~ /D/) mark="D del "
              else mark="M edit"
              print mark "\t" p
            }
          }
          if (hchg != "1") {
            for (p in bseen) {
              if (!(p in aseen)) print "- clr \t" p   # dirty 였다가 사라짐(revert/clean). commit 턴은 commit diff 가 담당하므로 생략
            }
          }
        }
      ' <<< "$AFTER_TREE"
    )"

    # ---- 기록 게이트: HEAD 변경 또는 working tree 변경이 있을 때만 ----
    if [ "$HEAD_CHANGED" -eq 0 ] && [ -z "$CHANGES" ]; then
      rm -f "$STATE/$SID".{prompt,ts,head,tree} 2>/dev/null || true
      exit 0
    fi

    {
      printf '\n### %s\n\n' "$(cat "$STATE/$SID.ts" 2>/dev/null)"
      cat "$STATE/$SID.prompt"
      printf '\n\n변경:\n'

      if [ "$HEAD_CHANGED" -eq 1 ]; then
        printf '  commit %s\n' "$(git -C "$REPO" log -1 --format='%h %s' "$AFTER_HEAD" 2>/dev/null)"
        git -C "$REPO" diff --name-status "$BEFORE_HEAD" "$AFTER_HEAD" 2>/dev/null \
          | awk -F'\t' '{printf "    %s %s\n", $1, $2}' | head -n "$MAX_FILES"
      fi

      if [ -n "$CHANGES" ]; then
        TOTAL="$(printf '%s\n' "$CHANGES" | grep -c .)"
        printf '%s\n' "$CHANGES" | head -n "$MAX_FILES" \
          | awk -F'\t' '{printf "  %s %s\n", $1, $2}'
        if [ "$TOTAL" -gt "$MAX_FILES" ]; then
          printf '  ... (+%d more)\n' "$((TOTAL - MAX_FILES))"
        fi
      fi
    } >> "$HIST"

    rm -f "$STATE/$SID".{prompt,ts,head,tree} 2>/dev/null || true
    exit 0
    ;;

  *)
    echo "usage: cmdhist.sh {pre|post}" >&2
    exit 0
    ;;
esac
