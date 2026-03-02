# proman

AI 코딩 도구들(Claude Code, Codex CLI, Gemini CLI, OpenCode)의 프롬프트/스킬/커맨드 파일을 한 곳에서 동기화하고 백업/복구하는 CLI입니다. (`v0.0.1`)

기준(source) 도구를 하나 선택하면, 해당 설정을 다른 도구들(target)로 반영합니다. 동기화/복구 전에는 항상 백업을 생성합니다.

## 기능

- 설치된 도구 자동 감지 (`detect`)
- 기준 도구 -> 대상 도구 동기화 (`sync`)
  - 동기화 전 대상 자산 자동 백업
  - 메인 지침 파일은 `PROMAN-SYNC-START/END` 블록 삽입/교체 방식
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

### 메인 지침 파일 동기화(sync block)

대상 메인 지침 파일에 아래 블록이 없으면 파일 끝에 추가하고, 이미 있으면 `PROMAN-SYNC-START/END` 사이 내용만 교체합니다.

```markdown
<!-- PROMAN-SYNC-START source=/absolute/path/to/source.md -->
[source 파일의 실제 내용이 여기에 삽입됨]
<!-- PROMAN-SYNC-END -->
```

### 디렉터리 동기화(커맨드/스킬/에이전트)

- `commands/`(또는 Codex의 `prompts/`): 대상 디렉터리를 삭제 후 재생성한 다음 미러링합니다.
  - Gemini 대상으로 동기화 시 `.md <-> .toml` 변환이 들어갑니다.
- `skills/`: 대상 디렉터리를 삭제 후 재생성한 다음 미러링합니다.
- `agents/`: 양쪽 도구에 `agents/` 디렉터리가 있을 때만 삭제 후 재생성한 다음 미러링합니다.

주의: `sync`는 대상 디렉터리를 교체(remove+recreate)합니다. 실행 전 자동 백업을 만들지만, 대상 디렉터리에 로컬 변경이 있다면 의도대로인지 확인하세요.

## 지원 경로 매핑

- `~/.claude/commands` == `~/.codex/prompts` == `~/.gemini/commands` == `~/.config/opencode/commands`
- `~/.claude/skills` == `~/.codex/skills` == `~/.gemini/skills` == `~/.config/opencode/skills`
- `~/.claude/CLAUDE.md` == `~/.codex/AGENTS.md` == `~/.gemini/GEMINI.md` == `~/.config/opencode/AGENTS.md`

## 백업 정책

- 기본 위치: `~/.proman/backups/`
- 백업 단위: 도구별 스냅샷 1개(`main + commands + skills + agents`)를 단일 압축 파일(`.tar.gz`)로 저장
- 파일명: `{tool}_snapshot_{YYYYMMDD_HHmmss}.tar.gz` (+ 해시 메타데이터 `{same}.sha256`)
- 보관: `{tool, asset, 확장자}`별 최근 10개 유지(초과분 자동 삭제)
- 중복 방지: 소스 해시가 최신 스냅샷 해시와 같으면 새 스냅샷 생성 생략
- 복구 안전장치: `restore` 실행 전 현재 상태를 `_pre_restore` 스냅샷으로 한 번 더 백업

## 빌드

```bash
go build -o proman .
```

## 사용법

설치 감지:

```bash
./proman detect
```

동기화(플래그 모드):

```bash
./proman sync --source claude --target gemini,opencode
```

동기화(인터랙티브 모드):

```bash
./proman sync
```

기본 런처(인터랙티브 모드 선택: `sync` / `restore` / `view`):

```bash
./proman
```

백업 조회(view 모드 직접 실행):

```bash
./proman view --tool gemini
```

백업 생성:

```bash
./proman backup --tool gemini
```

백업 이력 조회:

```bash
./proman backup --tool gemini --list
```

백업 삭제(인터랙티브 선택):

```bash
./proman backup --tool gemini --delete
```

복구(백업명 직접 지정):

```bash
./proman restore --tool gemini --backup gemini_snapshot_20260302_143000.tar.gz
```

복구(인터랙티브 백업 선택):

```bash
./proman restore --tool gemini
```

## 테스트

```bash
GOCACHE=/tmp/proman-go-build go test ./... -timeout=120s
```
