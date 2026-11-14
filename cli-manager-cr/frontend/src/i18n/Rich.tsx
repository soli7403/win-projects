// 꾸밈이 들어간 번역 문구(D75): 언어 파일 문구 안의 <b>굵게</b>, <k>키·입력 예시</k> 만 요소로 바꿔 그린다.
// 다른 태그는 글자 그대로 둔다(번역 파일에 HTML 을 넣지 않기 위해).
import { Fragment, type ReactNode } from 'react';

export function Rich({ text }: { text: string }) {
  const parts: ReactNode[] = [];
  const re = /<(b|k)>(.*?)<\/\1>/g;
  let last = 0;
  let m: RegExpExecArray | null;
  while ((m = re.exec(text))) {
    if (m.index > last) parts.push(text.slice(last, m.index));
    parts.push(m[1] === 'b' ? <b key={m.index}>{m[2]}</b> : <span key={m.index} className="kbdish">{m[2]}</span>);
    last = m.index + m[0].length;
  }
  if (last < text.length) parts.push(text.slice(last));
  return <Fragment>{parts}</Fragment>;
}
