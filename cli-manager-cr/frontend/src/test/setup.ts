// 테스트 공통 설정: 테스트마다 렌더링 결과를 정리한다(globals 를 쓰지 않으므로 직접 등록).
// 화면 언어는 한국어로 고정한다(테스트는 한국어 원문으로 찾는다, D75).
import { afterEach } from 'vitest';
import { cleanup } from '@testing-library/react';
import { useLang } from '../i18n';

useLang.getState().setLang('ko');

afterEach(() => {
  cleanup();
  useLang.getState().setLang('ko');
});
