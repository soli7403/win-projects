// 명령 모음 변수 (spec/behavior/terminal.md §7.1, D138).
import { beforeEach, describe, expect, it } from 'vitest';
import { fillVars, initialValues, parseVars, parseVarsAll, rememberValues, resetRememberedValues } from './snippetVars';

beforeEach(() => resetRememberedValues());

describe('parseVars', () => {
  it('나온 순서대로, 같은 이름은 한 번만', () => {
    expect(parseVars('ssh {{user}}@{{host}} -p {{port:22}}; echo {{host}}')).toEqual([
      { name: 'user', def: '' },
      { name: 'host', def: '' },
      { name: 'port', def: '22' },
    ]);
  });

  it('한글 이름·빈칸·하이픈을 받고 앞뒤 빈칸은 뺀다', () => {
    expect(parseVars('systemctl restart {{ 서비스 이름 }} {{log-level:info}}')).toEqual([
      { name: '서비스 이름', def: '' },
      { name: 'log-level', def: 'info' },
    ]);
  });

  it('점으로 시작하는 출력 형식과 \\{{ 는 변수가 아니다', () => {
    expect(parseVars('docker ps --format "{{.Names}} {{ .Status }}"')).toEqual([]);
    expect(parseVars('echo \\{{name}}')).toEqual([]);
  });

  it('기본값은 처음 적힌 것을 쓴다', () => {
    expect(parseVars('{{a}} {{a:1}} {{a:2}}')).toEqual([{ name: 'a', def: '1' }]);
  });

  it('여러 본문을 합친다', () => {
    expect(parseVarsAll(['cd {{dir}}', 'tail -f {{file:app.log}}', 'ls {{dir}}']).map((v) => v.name)).toEqual(['dir', 'file']);
  });
});

describe('fillVars', () => {
  it('값으로 바꾸고, 값이 없으면 기본값', () => {
    expect(fillVars('ssh {{user}}@{{host}} -p {{port:22}}', { user: 'root', host: '10.0.0.1' })).toBe('ssh root@10.0.0.1 -p 22');
  });

  it('빈 값을 넣으면 기본값이 아니라 빈 글자', () => {
    expect(fillVars('a{{x:1}}b', { x: '' })).toBe('ab');
  });

  it('변수가 아닌 것은 그대로, \\{{ 는 {{ 로', () => {
    expect(fillVars('docker ps --format "{{.Names}}" \\{{name}}', { name: 'x' })).toBe('docker ps --format "{{.Names}}" {{name}}');
  });

  it('값 안의 {{ 를 다시 풀지 않는다', () => {
    expect(fillVars('{{a}}', { a: '{{b}}', b: 'no' })).toBe('{{b}}');
  });
});

describe('마지막 값 기억', () => {
  it('지난번 값을 먼저, 없으면 기본값', () => {
    const vars = parseVars('{{host}} {{port:22}}');
    expect(initialValues('s1', vars)).toEqual({ host: '', port: '22' });
    rememberValues('s1', { host: 'web-01', port: '2222' });
    expect(initialValues('s1', vars)).toEqual({ host: 'web-01', port: '2222' });
    expect(initialValues('s2', vars)).toEqual({ host: '', port: '22' });
  });
});
