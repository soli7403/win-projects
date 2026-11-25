// `npm test` 진입점: 작업 경로의 Windows 드라이브 문자를 대문자로 맞춘 뒤 vitest 를 실행한다.
// Vitest 5 는 작업 경로가 소문자 드라이브(e:\ — Git Bash 의 /e/... 가 이렇게 넘어온다)이면 vitest 모듈을
// 두 벌 불러와 setup 파일의 afterEach 가 "failed to find the current suite" 로 실패한다.
// 다른 OS 와 대문자 드라이브에서는 바뀌는 것이 없다. 인자는 그대로 vitest 에 넘긴다.
import path from 'node:path';
import { pathToFileURL } from 'node:url';

const cwd = process.cwd();
if (/^[a-z]:/.test(cwd)) {
  process.chdir(cwd[0].toUpperCase() + cwd.slice(1));
}

await import(pathToFileURL(path.join(process.cwd(), 'node_modules', 'vitest', 'vitest.mjs')).href);
