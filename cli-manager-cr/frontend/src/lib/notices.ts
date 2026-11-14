// 참고한 프로젝트 고지(D63). 저장소 루트 THIRD_PARTY_NOTICES.md 와 같은 내용을 정보 화면에 보인다.
// 실행 파일만 배포해도 고지가 함께 가도록 앱 안에 넣어 둔다(MIT 조건: 저작권 고지·허가 문구 포함).

import type { MsgId } from '../i18n';

export const REFERENCE_NOTICE = {
  name: 'BrightTerm v1.1.0',
  summary: 'MIT · Copyright (c) 2026 Dany Kim',
  note: 'about.referenceNote' as MsgId, // 문구 키(D75)
  license: `MIT License

Copyright (c) 2026 Dany Kim

Permission is hereby granted, free of charge, to any person obtaining a copy
of this software and associated documentation files (the "Software"), to deal
in the Software without restriction, including without limitation the rights
to use, copy, modify, merge, publish, distribute, sublicense, and/or sell
copies of the Software, and to permit persons to whom the Software is
furnished to do so, subject to the following conditions:

The above copyright notice and this permission notice shall be included in all
copies or substantial portions of the Software.

THE SOFTWARE IS PROVIDED "AS IS", WITHOUT WARRANTY OF ANY KIND, EXPRESS OR
IMPLIED, INCLUDING BUT NOT LIMITED TO THE WARRANTIES OF MERCHANTABILITY,
FITNESS FOR A PARTICULAR PURPOSE AND NONINFRINGEMENT. IN NO EVENT SHALL THE
AUTHORS OR COPYRIGHT HOLDERS BE LIABLE FOR ANY CLAIM, DAMAGES OR OTHER
LIABILITY, WHETHER IN AN ACTION OF CONTRACT, TORT OR OTHERWISE, ARISING FROM,
OUT OF OR IN CONNECTION WITH THE SOFTWARE OR THE USE OR OTHER DEALINGS IN THE
SOFTWARE.`,
};
