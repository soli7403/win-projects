// 로그인 스크립트 편집용 계산. 근거: spec/ui/host-editor.md §1.4a, spec/behavior/session.md §5.1 (D139).
import type { LoginStep } from '../ipc/types';

/** 최대 단계 수(백엔드 G_ciLoginStepMax 와 같다). */
export const LOGIN_STEP_MAX = 20;
/** 한 단계가 기다리는 시간(초, 백엔드 G_durLoginStepTimeout 과 같다). 도움말 표시용. */
export const LOGIN_STEP_TIMEOUT_SEC = 20;
/** 보낼 글자에 쓰는 자리표시자: 이 서버에 로그인할 때 쓴 비밀번호로 바뀐다. */
export const LOGIN_PASSWORD_TOKEN = '{{password}}';

export function newLoginStep(): LoginStep {
  return { id: `ls-${Date.now().toString(36)}-${Math.random().toString(36).slice(2, 7)}`, expect: '', send: '' };
}

/**
 * 저장할 단계: 기다릴 글자·보낼 글자·보관함 값이 모두 빈 단계는 빼고, 최대 개수까지만. 하나도 없으면 undefined.
 * 보관함 값을 고른 단계는 보낼 글자를 비워 저장한다(D148). 고르지 않은 단계에는 secretId 를 남기지 않는다.
 */
export function cleanLoginScript(steps: readonly LoginStep[] | undefined): LoginStep[] | undefined {
  const out = (steps ?? [])
    .filter((s) => s.expect !== '' || s.send !== '' || !!s.secretId)
    .slice(0, LOGIN_STEP_MAX)
    .map((s) => {
      if (s.secretId) return { id: s.id, expect: s.expect, send: '', secretId: s.secretId };
      const { secretId: _drop, ...rest } = s;
      return rest;
    });
  return out.length ? out : undefined;
}

/** i 번째 단계를 dir(-1 위, +1 아래)로 옮긴 새 배열. 옮길 수 없으면 같은 배열. */
export function moveStep<T>(steps: readonly T[], i: number, dir: -1 | 1): T[] {
  const j = i + dir;
  if (i < 0 || i >= steps.length || j < 0 || j >= steps.length) return steps as T[];
  const next = [...steps];
  [next[i], next[j]] = [next[j], next[i]];
  return next;
}
