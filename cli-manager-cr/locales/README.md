# 언어 파일 (locales)

DariTerm 화면과 안내 문구의 언어별 파일입니다(결정 D75, 명세 `spec/behavior/i18n.md`). **문구를 고칠 때는 이 폴더의 json 만 고칩니다.** 코드에는 키만 있습니다.

| 파일 | 언어 |
|---|---|
| `ko.json` | 한국어 — **기준 파일**(키 목록이 여기서 정해짐) |
| `en.json` | English — 맞는 언어가 없을 때 쓰는 언어 |
| `ja.json` `zh.json` `tw.json` `de.json` `fr.json` `es.json` `pt.json` `ru.json` | 日本語, 简体中文, 繁體中文, Deutsch, Français, Español, Português (Brasil), Русский |

## 규칙

- 형식: `"키": "문구"`. 키 이름은 `영역.이름`(예: `sb.addServer`), 백엔드 문구는 `be.` 로 시작.
- `{이름}` 은 자리표시자다. 언어마다 같은 이름을 그대로 둔다(순서는 바꿔도 된다).
- 꾸밈은 `<b>굵게</b>`, `<k>키·입력 예시</k>` 만 쓴다.
- 새 문구: `ko.json` 에 키를 추가하고 **10개 파일 모두에** 같은 키를 넣는다. 빠지거나 자리표시자가 다르면 테스트가 실패한다
  (`frontend: npx vitest run src/i18n`, `go test ./internal/i18n`).
- 고친 뒤 `wails build` 하면 실행 파일에 들어간다.
