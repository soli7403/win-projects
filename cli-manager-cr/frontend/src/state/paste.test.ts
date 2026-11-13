// 경로 입력 형식 테스트. 근거: spec/behavior/terminal.md §5.
import { describe, expect, it } from 'vitest';
import { quotePath } from './paste';

describe('경로 입력 따옴표 (terminal.md §5)', () => {
  it('특수문자나 공백이 있을 때만 작은따옴표, 안의 작은따옴표는 이스케이프', () => {
    expect(quotePath('/tmp/a.png', true)).toBe('/tmp/a.png');
    expect(quotePath('/tmp/my file.png', true)).toBe("'/tmp/my file.png'");
    expect(quotePath("it's.png", true)).toBe("'it'\\''s.png'");
    expect(quotePath('/tmp/my file.png', false)).toBe('/tmp/my file.png');
  });
});

// T9: 클립보드 이미지를 SSH 세션에 붙여넣으면 "클립보드 이미지 서버로 전송 중..." → 서버 경로 입력 (acceptance 03 T9).
describe('이미지 붙여넣기 업로드 (terminal.md §5, T9)', () => {
  it('알림 라벨은 세션 이름이 아니라 "클립보드 이미지"이고, 올린 경로를 붙여넣기 방식으로 입력한다', async () => {
    const { setBackend } = await import('../ipc/backend');
    const { useApp } = await import('./app');
    const { useWorkspace } = await import('./workspace');
    const { uploadAndInsert } = await import('./paste');
    const typed: string[] = [];
    setBackend({
      async invoke(channel: string): Promise<any> {
        if (channel === 'sftp.uploadForPrompt') return ['/home/ops/.dariterm/uploads/tg-20261114-093015-a3f1.png'];
        return undefined;
      },
      termInput: (_sid: string, data: string) => typed.push(data),
      termResize: () => undefined,
      reply: () => undefined,
      on: () => () => undefined,
    });
    useApp.setState({ toasts: [] });
    useWorkspace.setState({
      sessions: { s1: { info: { id: 's1', title: 'mac-mini', protocol: 'ssh', state: 'connected', canSftp: true } } },
    } as never);
    await uploadAndInsert('s1', ['s1'], { kind: 'clipboardImage' });
    const texts = useApp.getState().toasts.map((t) => t.text);
    expect(texts[0]).toBe('클립보드 이미지 서버로 전송 중...');
    expect(texts[1]).toBe('클립보드 이미지 전송 완료 → /home/ops/.dariterm/uploads/tg-20261114-093015-a3f1.png');
    expect(typed).toEqual(['\x1b[200~/home/ops/.dariterm/uploads/tg-20261114-093015-a3f1.png \x1b[201~']);
  });
});
