# agentsyncker

`agentsyncker`는 Claude Code, Codex CLI, Gemini CLI, OpenCode에 흩어져 있는 프롬프트/스킬/커맨드 자산을 **하나의 기준(source) 설정으로 동기화**하고, 변경 전 상태를 **자동 백업/복구**할 수 있게 해주는 CLI 도구입니다. (`v0.0.1`)

## 프로그램 소개

AI CLI 도구를 여러 개 함께 쓰다 보면 아래와 같은 문제가 자주 생깁니다.

- 도구마다 메인 지침 파일 이름이 다름 (`CLAUDE.md`, `AGENTS.md`, `GEMINI.md` 등)
- 커맨드/프롬프트 저장 경로가 서로 다름 (`commands/`, `prompts/`)
- Gemini처럼 별도 포맷(`.toml`) 변환이 필요한 도구가 있음
- 수동 복사 과정에서 누락/충돌/덮어쓰기 위험이 큼

`agentsyncker`는 이 문제를 해결하기 위해,

1. 설치된 도구를 자동 감지하고,
2. 기준 도구를 선택해 다른 도구로 동기화하며,
3. 동기화/복구 전에 항상 백업을 생성하고,
4. 필요 시 원하는 시점으로 안전하게 복구할 수 있게 설계되었습니다.

---

기준(source) 도구를 하나 선택하면, 해당 설정을 다른 도구들(target)로 반영합니다. 동기화/복구 전에는 항상 백업을 생성합니다.

## 기능

- 설치된 도구 자동 감지 (`detect`)
- 기준 도구 -> 대상 도구 동기화 (`sync`)
  - 동기화 전 대상 자산 자동 백업
  - 메인 지침 파일은 소스 파일 내용으로 전체 교체
  - 커맨드 디렉터리 동기화 시 `Markdown <-> TOML` 변환 지원(Gemini)
  - `skills/` 디렉터리 미러링(그대로 복사)
  - 양쪽 도구가 `agents/`를 지원하면 `agents/`도 미러링
- 백업 생성/조회/삭제 (`backup`, `backup --list`, `backup --delete`)
  - 기본 보관 정책: `{tool, asset, 확장자}` 기준 최근 10개 유지
  - 소스 해시(sha256)가 최신 백업과 동일하면 새 백업 생성 건너뜀
  - 인터랙티브 삭제 시 백업 생성 일시(YYYY-MM-DD HH:mm:ss) 표시
- 복구 (`restore`)
  - 복구 전 pre-restore 백업 자동 생성

## 동작 방식

### 메인 지침 파일 동기화(전체 교체)

대상 메인 지침 파일은 기존 내용을 유지하지 않고, 소스 메인 지침 파일의 내용으로 완전히 덮어씁니다.

### 디렉터리 동기화(커맨드/스킬/에이전트)

- `commands/`(또는 Codex의 `prompts/`): 대상 디렉터리를 삭제 후 재생성한 다음 미러링합니다.
  - Gemini 대상으로 동기화 시 `.md <-> .toml` 변환이 들어갑니다.
- `skills/`: 대상 디렉터리를 삭제 후 재생성한 다음 미러링합니다.
- `agents/`: 양쪽 도구에 `agents/` 디렉터리가 있을 때만 삭제 후 재생성한 다음 미러링합니다.

주의: `sync`는 대상 메인 파일/디렉터리를 교체합니다. 실행 전 자동 백업을 만들지만, 대상에 로컬 변경이 있다면 의도대로인지 확인하세요.

## 지원 경로 매핑

- `~/.claude/commands` == `~/.codex/prompts` == `~/.gemini/commands` == `~/.config/opencode/commands`
- `~/.claude/skills` == `~/.codex/skills` == `~/.gemini/skills` == `~/.config/opencode/skills`
- `~/.claude/CLAUDE.md` == `~/.codex/AGENTS.md` == `~/.gemini/GEMINI.md` == `~/.config/opencode/AGENTS.md`

## 백업 정책

- 기본 위치: `~/.agentsyncker/backups/`
- 백업 단위: 도구별 스냅샷 1개(`main + commands + skills + agents`)를 단일 압축 파일(`.tar.gz`)로 저장
- 파일명: `{tool}_snapshot_{YYYYMMDD_HHmmss}.tar.gz` (+ 해시 메타데이터 `{same}.sha256`)
- 보관: `{tool, asset, 확장자}`별 최근 10개 유지(초과분 자동 삭제)
- 중복 방지: 소스 해시가 최신 스냅샷 해시와 같으면 새 스냅샷 생성 생략
- 복구 안전장치: `restore` 실행 전 현재 상태를 `_pre_restore` 스냅샷으로 한 번 더 백업

## 빌드

```bash
go build -o agentsyncker .
```

## 사용법

설치 감지:

```bash
./agentsyncker detect
```

동기화(플래그 모드):

```bash
./agentsyncker sync --source claude --target gemini,opencode
```

동기화(인터랙티브 모드):

```bash
./agentsyncker sync
```

기본 런처(인터랙티브 모드 선택: `sync` / `restore` / `view`):

```bash
./agentsyncker
```

백업 조회(view 모드 직접 실행):

```bash
./agentsyncker view --tool gemini
```

백업 생성:

```bash
./agentsyncker backup --tool gemini
```

백업 이력 조회:

```bash
./agentsyncker backup --tool gemini --list
```

백업 삭제(인터랙티브 선택):

```bash
./agentsyncker backup --tool gemini --delete
```

복구(백업명 직접 지정):

```bash
./agentsyncker restore --tool gemini --backup gemini_snapshot_20260302_143000.tar.gz
```

복구(인터랙티브 백업 선택):

```bash
./agentsyncker restore --tool gemini
```

## 테스트

```bash
GOCACHE=/tmp/agentsyncker-go-build go test ./... -timeout=120s
```
