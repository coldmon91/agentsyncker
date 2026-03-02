# cli 프로그램 프롬프트 통합 관리자

프롬프트와 스킬들이 통일된 위치에 있지 않아서 관리가 어려운 문제가 있어.
통합 프롬프트 관리자 프로그램을 만들자. (cli 프로그램으로 만들면 좋을듯)

## 버전 
- v0.0.1

## 대상 프로그램 
- Claude Code, Codex Cli, Gemini Cli, OpenCode

## 기능
- skills/, prompts/, agents/, AGENTS.md, GEMINI.md 등등 프롬프트 파일들을 통합 관리
- 대상 프로그램의 프롬프트를 하나 지정하면 그 프롬프트에 필요한 스킬과 프롬프트들을 다른 프로그램들에 동기화 한다.
- 생성된 백업 이력을 조회하고 특정 백업 시점으로 복구(restore)할 수 있어야 한다.

### 흐름
- 프로그램이 시작되면 어떤 프로그램들이 설치되어 있는지 자동으로 감지한다. (예: `~/.claude/`, `~/.codex/`, `~/.gemini/`, `~/.config/opencode/` 디렉터리 존재 여부 확인)
- 기준이 될 프로그램을 하나 지정할 수 있다. (예: Claude Code)
- 기준 프로그램이 선택되면 동기화 할 프로그램을 체크박스 형식으로 선택할 수 있다. (예: Gemini Cli, OpenCode)
- 동기화 작업 전에 백업 필수.
- 사용자는 백업 목록에서 시점을 선택해 파일/디렉터리를 복구할 수 있어야 한다.

### 시나리오
예를들어 Claude Code의 설정을 Gemini cli 와 동기화 한다면,
~/.claude/skills/, ~/.claude/commands/, ~/.claude/CLAUDE.md 
~/.gemini/GEMINI.md 의 내용에 
```markdown
<!-- PROMAN-SYNC-START source=~/.claude/CLAUDE.md -->
[~/.claude/CLAUDE.md의 실제 내용을 여기에 복사해서 삽입한다]
<!-- PROMAN-SYNC-END -->
```
와 같은 동기화 블록을 추가한다.

규칙:
1. 동기화 전에 대상 파일을 반드시 백업한다.
2. `@파일경로` 같은 참조 문법은 설정 파일에서 include로 동작하지 않으므로 사용하지 않는다.
3. 소스 파일의 실제 내용을 대상 파일에 직접 삽입한다.
4. 재동기화 시 `PROMAN-SYNC-START/END` 사이 구간만 교체한다.
5. 모든 도구 간 변환은 단방향이 아니라 상호(왕복) 변환 가능해야 하며, 재변환 시 의미 손실이 없어야 한다.
6. 복구(restore) 실행 전 현재 상태를 한 번 더 백업하고, 복구 실패 시 즉시 중단한다.

## 파일 매핑
- `~/.claude/commands/` == `~/.codex/prompts/` == `~/.gemini/commands/` == `~/.config/opencode/commands/` (단, 형식 변환 필요: Gemini=`.toml`)
- `~/.claude/skills/` == `~/.codex/skills/` == `~/.gemini/skills/` == `~/.config/opencode/skills/`
- `~/.claude/CLAUDE.md` == `~/.codex/AGENTS.md` == `~/.gemini/GEMINI.md` == `~/.config/opencode/AGENTS.md`

## 도구별 형식 요구사항
- Claude Code: 메인 지침 파일은 `~/.claude/CLAUDE.md`, 보조 자산은 `commands/`, `skills/` 디렉터리.
- Codex Cli: 메인 지침 파일은 `~/.codex/AGENTS.md`, 보조 자산은 `prompts/`, `skills/` 디렉터리.
- Gemini Cli: 메인 지침 파일은 `~/.gemini/GEMINI.md`. 보조 자산은 `~/.gemini/commands/`(**.toml** 형식), `~/.gemini/skills/` 디렉터리. 커맨드 동기화 시 Claude `.md` → Gemini `.toml` 형식 변환 필요.
- OpenCode: 전역 설정은 `~/.config/opencode/opencode.json`(JSON/JSONC). 보조 자산은 `~/.config/opencode/{agents,commands,skills}/` 또는 프로젝트 `.opencode/{agents,commands,skills}/` 사용.
- OpenCode 참고: `opencode.json`에서는 `{file:...}` 변수 치환이 가능하지만, 본 관리자 동기화 정책은 소스 내용을 대상 파일/디렉터리에 직접 반영하는 것을 기본으로 한다.

## 커맨드 형식 변환 규칙 (Claude `.md` ↔ Gemini `.toml`)

Claude 커맨드 파일 (`~/.claude/commands/test.md`):
```markdown
---
description: Run tests with coverage
---
Run the full test suite with coverage report and show any failures.
Focus on the failing tests and suggest fixes.
```

↓ 변환 ↓

Gemini 커맨드 파일 (`~/.gemini/commands/test.toml`):
```toml
description = "Run tests with coverage"
prompt = """
Run the full test suite with coverage report and show any failures.
Focus on the failing tests and suggest fixes.
"""
```

변환 규칙:
1. `.md` frontmatter의 `description` → `.toml`의 `description` 필드.
2. `.md` body(frontmatter 이후 본문) → `.toml`의 `prompt` 필드 (여러 줄일 경우 `"""` 사용).
3. 파일명은 그대로 유지하되 확장자만 `.md` ↔ `.toml`로 변경.
4. 하위 디렉터리 구조도 유지 (예: `commands/git/commit.md` → `commands/git/commit.toml`). Gemini에서는 `/` 구분이 `:`으로 변환되어 `/git:commit` 명령이 됨.

## 백업 정책
- 백업 위치: `~/.agentsyncker/backups/`
- 파일명 규칙: `{프로그램명}_{원본파일명}_{YYYYMMDD_HHmmss}.bak` (예: `gemini_GEMINI.md_20260302_143000.bak`)
- 디렉터리 백업: 대상 디렉터리 전체를 tar.gz로 압축하여 저장 (예: `gemini_commands_20260302_143000.tar.gz`)
- 보관: 최근 5회분 백업 유지, 초과 시 가장 오래된 백업 자동 삭제.
- 복구 단위: 파일 백업(`.bak`)과 디렉터리 백업(`.tar.gz`) 모두 지원.
- 복구 명령 예시: `agentsyncker restore --tool gemini --backup gemini_GEMINI.md_20260302_143000.bak`
- 복구 시 안전장치: 원복 전에 현재 상태를 사전 백업(pre-restore backup)으로 저장.
