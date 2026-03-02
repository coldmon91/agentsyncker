# agentsyncker 작업계획서

> **버전**: v0.0.1  
> **언어**: Go (1.22+)  
> **작성일**: 2026-03-02  

---

## 1. 프로젝트 개요

AI 코딩 도구들(Claude Code, Codex CLI, Gemini CLI, OpenCode)의 프롬프트·스킬·커맨드를 통합 관리하는 CLI 프로그램.  
하나의 기준 도구를 선택하면 나머지 도구들에 설정을 동기화한다.

---

## 2. 기술 스택

| 항목 | 선택 | 사유 |
|------|------|------|
| 언어 | Go 1.24.1+ | 요구사항 |
| CLI 프레임워크 | [cobra](https://github.com/spf13/cobra) | Go CLI 표준, 서브커맨드 구조 |
| 인터랙티브 UI | [bubbletea](https://github.com/charmbracelet/bubbletea) + [huh](https://github.com/charmbracelet/huh) | 체크박스·셀렉트 등 TUI |
| TOML 처리 | [pelletier/go-toml/v2](https://github.com/pelletier/go-toml) | Gemini 커맨드 변환 |
| YAML frontmatter | [adrg/frontmatter](https://github.com/adrg/frontmatter) | Claude 커맨드 파싱 |
| 테스트 | 표준 `testing` + [testify](https://github.com/stretchr/testify) | assertion 편의 |
| 압축 | 표준 `archive/tar`, `compress/gzip` | 디렉터리 백업 |

---

## 3. 디렉터리 구조

```
agentsyncker/
├── AGENTS.md                 # 요구사항 문서
├── PLAN.md                   # 작업계획서 (본 문서)
├── go.mod
├── go.sum
├── main.go                   # 엔트리포인트
├── cmd/                      # cobra 커맨드 정의
│   ├── root.go               # 루트 커맨드 (버전, 도움말)
│   ├── detect.go             # detect 서브커맨드
│   ├── sync.go               # sync 서브커맨드
│   ├── backup.go             # backup 서브커맨드
│   └── restore.go            # restore 서브커맨드
├── internal/
│   ├── detector/             # 도구 설치 감지
│   │   ├── detector.go
│   │   └── detector_test.go
│   ├── config/               # 도구별 설정 정의 (경로, 매핑)
│   │   ├── tools.go
│   │   └── tools_test.go
│   ├── sync/                 # 동기화 엔진
│   │   ├── engine.go         # 동기화 오케스트레이터
│   │   ├── engine_test.go
│   │   ├── mainfile.go       # 메인 지침 파일 동기화 (CLAUDE.md → GEMINI.md 등)
│   │   ├── mainfile_test.go
│   │   ├── commands.go       # 커맨드/프롬프트 디렉터리 동기화
│   │   ├── commands_test.go
│   │   ├── skills.go         # 스킬 디렉터리 동기화
│   │   └── skills_test.go
│   ├── converter/            # 형식 변환기
│   │   ├── md_toml.go        # Claude .md ↔ Gemini .toml
│   │   └── md_toml_test.go
│   ├── backup/               # 백업 관리
│   │   ├── backup.go
│   │   └── backup_test.go
│   └── syncblock/            # PROMAN-SYNC 블록 삽입/교체
│       ├── block.go
│       └── block_test.go
└── testdata/                 # 테스트 픽스처
    ├── claude_command.md
    ├── gemini_command.toml
    └── sample_claude.md
```

---

## 4. 핵심 모듈 설계

### 4.1 도구 정의 (`internal/config/tools.go`)

```go
type Tool struct {
    Name       string   // "claude", "codex", "gemini", "opencode"
    DisplayName string  // "Claude Code", "Codex CLI", ...
    HomeDir    string   // "~/.claude", "~/.codex", ...
    MainFile   string   // "CLAUDE.md", "AGENTS.md", "GEMINI.md", ...
    CommandDir string   // "commands", "prompts", "commands", ...
    SkillDir   string   // "skills"
    CmdFormat  string   // "md", "md", "toml", "md"
}
```

### 4.2 도구 감지 (`internal/detector/`)

- 각 도구의 홈 디렉터리 존재 여부 확인
- `[]Tool` (감지된 도구 목록) 반환

### 4.3 동기화 블록 (`internal/syncblock/`)

```
<!-- PROMAN-SYNC-START source={소스경로} -->
{내용}
<!-- PROMAN-SYNC-END -->
```

- **Insert**: 대상 파일에 블록이 없으면 파일 끝에 추가
- **Update**: 블록이 이미 있으면 `START/END` 사이만 교체
- **Extract**: 블록에서 소스 경로와 내용 파싱 (역방향 변환용)

### 4.4 커맨드 형식 변환 (`internal/converter/`)

| 방향 | 입력 | 출력 |
|------|------|------|
| MD → TOML | frontmatter `description` + body | `description = "..."` + `prompt = """..."""` |
| TOML → MD | `description`, `prompt` 필드 | frontmatter + body |

- 하위 디렉터리 구조 유지
- 확장자만 변경 (`.md` ↔ `.toml`)

### 4.5 백업 관리 (`internal/backup/`)

- 백업 위치: `~/.agentsyncker/backups/`
- 파일 백업: `{tool}_{filename}_{YYYYMMdd_HHmmss}.bak`
- 디렉터리 백업: `{tool}_{dirname}_{YYYYMMdd_HHmmss}.tar.gz`
- 최근 5회분만 유지, 초과분 자동 삭제
- 백업 목록 조회: 도구별/자산별 백업 이력 조회 지원
- 복구(restore): 선택한 `.bak` 또는 `.tar.gz`를 원위치로 복원
- 복구 안전장치: 복구 직전 현재 상태를 pre-restore 백업으로 저장

### 4.6 동기화 엔진 (`internal/sync/`)

동기화 순서:
1. 소스/대상 도구 검증
2. 대상 파일·디렉터리 백업
3. 메인 지침 파일 동기화 (sync block 방식)
4. 커맨드 디렉터리 동기화 (형식 변환 포함)
5. 스킬 디렉터리 동기화 (그대로 복사)
6. 결과 리포트 출력

---

## 5. CLI 인터페이스

### 5.1 인터랙티브 모드 (기본)

```
$ agentsyncker sync

🔍 설치된 도구 감지 중...
  ✓ Claude Code (~/.claude/)
  ✓ Gemini CLI  (~/.gemini/)
  ✗ Codex CLI   (~/.codex/) — 미설치
  ✓ OpenCode    (~/.config/opencode/)

? 기준 도구를 선택하세요:
  > Claude Code
    Gemini CLI
    OpenCode

? 동기화 대상을 선택하세요: (Space로 선택, Enter로 확인)
  [x] Gemini CLI
  [ ] OpenCode

📦 백업 생성 중...
  ✓ gemini_GEMINI.md_20260302_143000.bak
  ✓ gemini_commands_20260302_143000.tar.gz

🔄 동기화 중...
  ✓ CLAUDE.md → GEMINI.md (sync block 삽입)
  ✓ commands/ → commands/ (3 파일, md→toml 변환)
  ✓ skills/ → skills/ (2 파일 복사)

✅ 동기화 완료!
```

### 5.2 비인터랙티브 모드 (플래그)

```bash
agentsyncker sync --source claude --target gemini,opencode
agentsyncker detect                        # 설치된 도구 목록 출력
agentsyncker backup --tool gemini          # 수동 백업
agentsyncker restore --tool gemini --backup gemini_GEMINI.md_20260302_143000.bak
agentsyncker sync --source gemini --target claude  # 역방향 동기화
```

---

## 6. 작업 단계 (마일스톤)

### Phase 1: 프로젝트 초기화 & 기반 구조
| # | 작업 | 예상 |
|---|------|------|
| 1-1 | Go 모듈 초기화, 의존성 설치 | 10분 |
| 1-2 | `internal/config/tools.go` — 도구 정의 및 경로 매핑 | 20분 |
| 1-3 | `internal/detector/` — 도구 설치 감지 + 테스트 | 20분 |
| 1-4 | `cmd/root.go`, `cmd/detect.go` — 기본 CLI 골격 | 15분 |

### Phase 2: 백업 시스템
| # | 작업 | 예상 |
|---|------|------|
| 2-1 | `internal/backup/` — 파일 백업 (복사 + 타임스탬프) | 25분 |
| 2-2 | `internal/backup/` — 디렉터리 백업 (tar.gz) | 20분 |
| 2-3 | 백업 보관 정책 (최근 5회분, 자동 삭제) | 15분 |
| 2-4 | `internal/backup/` — 백업 이력 조회 + restore 복구 로직 | 25분 |
| 2-5 | `cmd/backup.go`, `cmd/restore.go` — backup/restore 서브커맨드 | 15분 |

### Phase 3: 동기화 블록 & 메인 파일 동기화
| # | 작업 | 예상 |
|---|------|------|
| 3-1 | `internal/syncblock/` — sync block 삽입/교체/파싱 + 테스트 | 30분 |
| 3-2 | `internal/sync/mainfile.go` — 메인 지침 파일 동기화 | 20분 |

### Phase 4: 커맨드 형식 변환
| # | 작업 | 예상 |
|---|------|------|
| 4-1 | `internal/converter/md_toml.go` — MD → TOML 변환 + 테스트 | 25분 |
| 4-2 | `internal/converter/md_toml.go` — TOML → MD 역변환 + 테스트 | 20분 |
| 4-3 | 왕복 변환 무손실 검증 테스트 | 15분 |

### Phase 5: 디렉터리 동기화
| # | 작업 | 예상 |
|---|------|------|
| 5-1 | `internal/sync/commands.go` — 커맨드 디렉터리 동기화 (변환 포함) | 25분 |
| 5-2 | `internal/sync/skills.go` — 스킬 디렉터리 동기화 (복사) | 15분 |
| 5-3 | `internal/sync/engine.go` — 동기화 오케스트레이터 | 20분 |

### Phase 6: CLI 완성 & 인터랙티브 UI
| # | 작업 | 예상 | 상태 |
|---|------|------|------|
| 6-1 | `cmd/sync.go` — sync 서브커맨드 (플래그 모드) | 20분 | ✓ |
| 6-2 | 인터랙티브 UI (bubbletea/huh) — 도구 선택, 체크박스 | 30분 | ✓ |
| 6-3 | 동기화 진행 상황 표시 & 결과 리포트 | 15분 | ✓ |

### Phase 7: 테스트 & 마무리
| # | 작업 | 예상 |
|---|------|------|
| 7-1 | 통합 테스트 (임시 디렉터리 기반 E2E) | 30분 |
| 7-2 | 역방향 동기화 테스트 (Gemini → Claude 등) | 20분 |
| 7-3 | README.md 작성 | 15분 |
| 7-4 | goreleaser 또는 `go install` 빌드 설정 | 10분 |

---

## 7. 데이터 흐름도

```
┌─────────────┐     detect      ┌──────────────┐
│   CLI (cmd/) │ ──────────────> │  detector/   │
│              │                 │  (감지 결과)  │
│  sync 실행   │                 └──────────────┘
│              │     backup      ┌──────────────┐
│              │ ──────────────> │  backup/     │
│              │                 │  (~/.agentsyncker)│
│              │     sync        └──────────────┘
│              │ ──────────────> ┌──────────────┐
└─────────────┘                 │  sync/engine │
                                │              │
                    ┌───────────┤  mainfile    │
                    │           │  commands    │
                    │           │  skills      │
                    │           └──────┬───────┘
                    │                  │
                    v                  v
            ┌──────────────┐   ┌──────────────┐
            │  syncblock/  │   │  converter/  │
            │  (블록 삽입)  │   │  (md ↔ toml) │
            └──────────────┘   └──────────────┘
```

---

## 8. 주요 제약사항 & 설계 결정

1. **직접 삽입 정책**: `@파일경로` 참조 문법 사용 금지, 소스 내용을 대상 파일에 직접 복사
2. **왕복 무손실**: 모든 변환은 A→B→A 시 의미 손실 없어야 함
3. **sync block 교체**: 재동기화 시 `PROMAN-SYNC-START/END` 사이만 교체
4. **백업 필수**: 동기화 전 반드시 백업, 백업 실패 시 동기화 중단
5. **복구 안전성**: 복구 전 pre-restore 백업 생성, 복구 실패 시 기존 상태 유지
6. **에러 처리**: Go 관례에 따라 `Result`/`Option` 대신 `error` 반환, `panic` 사용 금지
7. **타임아웃**: 모든 테스트에 타임아웃 설정 (`-timeout` 플래그)

---

## 9. 작업 우선순위

```
Phase 1 (기반) → Phase 2 (백업) → Phase 3 (sync block)
    → Phase 4 (변환기) → Phase 5 (동기화) → Phase 6 (CLI)
        → Phase 7 (테스트/마무리)
```

Phase 1 ~ 3 이 핵심 의존성이므로 우선 구현하고, Phase 4 ~ 6 은 병렬로 진행 가능.
