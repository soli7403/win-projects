// 테스트 공통 설정: 테스트마다 렌더링 결과를 정리한다(globals 를 쓰지 않으므로 직접 등록).
import { afterEach } from 'vitest';
import { cleanup } from '@testing-library/react';

afterEach(() => {
  cleanup();
});
