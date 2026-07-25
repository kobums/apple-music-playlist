# Apple Music Playlist — 백엔드

곡 목록 텍스트를 받아 Apple Music 라이브러리 플레이리스트에 넣어 주는 Go(Fiber) 서버입니다.

프론트엔드는 [`apple_music_playlist_front`](../apple_music_playlist_front)에 있습니다.

## 하는 일

붙여넣은 텍스트를 한 줄씩 `아티스트 - 곡 제목`으로 정규화하고, Apple Music 카탈로그에서
후보를 찾아 **채점**한 뒤, 확신되는 것만 자동으로 넣고 애매한 것은 사용자에게 돌려줍니다.

```
텍스트 파싱 → 후보 5개 검색 → 채점 → 3분류 → 중복 제외 → 순서대로 추가
```

핵심 설계 두 가지가 있습니다.

**틀린 곡을 넣지 않습니다.** 예전에는 검색 첫 결과를 검증 없이 담아서, 커버·라이브·동명이인이
들어가도 사용자는 "추가됨"만 봤습니다. 못 찾은 곡은 눈에 띄지만 잘못 들어간 곡은 안 보이므로
더 위험합니다. 이제 신뢰도가 낮으면 넣지 않고 후보를 돌려줍니다.

**적어 준 순서를 지킵니다.** Apple Music API에는 트랙 재정렬 기능이 없고 추가는 항상 뒤에
붙습니다. 그래서 확인이 필요한 곡이 하나라도 있으면 확신되는 곡까지 **전부 보류**했다가, 사용자가
결정한 뒤 한 번에 순서대로 넣습니다.

## API

모든 요청/응답은 JSON입니다. 실패 시 `{"code":"error"|"unauthorized", "error":"..."}` 형태로
내려가며, Apple의 원본 응답은 서버 로그에만 남고 클라이언트로 새지 않습니다.

| 메서드 | 경로 | 설명 |
| --- | --- | --- |
| `GET` | `/api/token` | MusicKit 개발자 토큰 발급 |
| `POST` | `/api/playlists` | 추가 가능한 라이브러리 플레이리스트 목록 |
| `POST` | `/api/playlist` | 곡 목록을 파싱·매칭하고 (가능하면) 추가 |
| `POST` | `/api/search` | 곡 직접 재검색 |
| `POST` | `/api/playlist/tracks` | 사용자가 확정한 곡 추가 |

### `GET /api/token`

```json
{ "code": "ok", "developerToken": "eyJhbGciOiJFUzI1NiIs..." }
```

### `POST /api/playlists`

```json
{ "userToken": "<Music User Token>" }
```

```json
{ "code": "ok", "playlists": [{ "id": "p.xxxx", "name": "드라이브" }] }
```

편집 불가능한 플레이리스트(Apple 큐레이션 등, `canEdit: false`)는 제외하고 **최근 만든 순**으로
돌려줍니다. 고른 뒤 저장 단계에서야 실패하는 일을 막기 위해서입니다.

### `POST /api/playlist`

```json
{
  "title": "여름 밤",
  "playlistId": "p.xxxx",
  "list": "빅뱅 - 붉은 노을\n아이유 - 밤편지",
  "userToken": "<Music User Token>"
}
```

`playlistId`가 있으면 그 플레이리스트에 바로 추가합니다. 없으면 `title`로 찾고, 없으면 만듭니다.
**ID 지정이 권장됩니다** — 이름 매칭은 오타로 새 플레이리스트를 만들거나, 같은 이름이 여러 개일 때
어느 쪽인지 추측하게 됩니다.

`developerToken` 필드는 하위 호환으로 받기만 하고 무시합니다. 서버가 직접 서명합니다.

```json
{
  "code": "ok",
  "playlistId": "p.xxxx",
  "playlistName": "여름 밤",
  "playlistExistingCount": 34,
  "playlistCreated": false,
  "deferred": true,
  "result": [
    {
      "song": "빅뱅 - 붉은 노을",
      "status": "review",
      "score": 0.6,
      "track": { "id": "...", "name": "붉은 노을", "artistName": "BIGBANG", "artworkUrl": "...", "previewUrl": "..." },
      "candidates": [ ... ]
    }
  ]
}
```

`deferred`가 `true`면 **아직 아무것도 저장되지 않았습니다.** 클라이언트가
`/api/playlist/tracks`로 확정해야 합니다.

#### `result[].status`

| 값 | 의미 |
| --- | --- |
| `added` | 플레이리스트에 들어갔습니다 |
| `duplicate` | 이미 그 플레이리스트에 있어 건너뛰었습니다 |
| `ready` | 확신되는 매칭이지만 순서를 지키려고 대기 중입니다 |
| `review` | 후보는 찾았지만 사람이 확인해야 합니다 |
| `missing` | 쓸 만한 후보가 없습니다 |

### `POST /api/search`

```json
{ "userToken": "...", "query": "검정치마 Everything" }
```

```json
{ "code": "ok", "candidates": [ { "id": "...", "name": "...", "artistName": "..." } ] }
```

### `POST /api/playlist/tracks`

```json
{ "userToken": "...", "playlistId": "p.xxxx", "songIds": ["1440857781", "1443621832"] }
```

```json
{ "code": "ok", "added": ["1440857781"], "duplicate": ["1443621832"] }
```

**보낸 순서대로** 추가합니다. 이미 있는 곡은 `duplicate`로 분리해 돌려주므로, 클라이언트가 행마다
정확한 상태를 표시할 수 있습니다.

## 매칭 방식

카탈로그 후보 5개를 받아 각각 점수를 매깁니다.

```
점수 = 0.6 × 제목 유사도 + 0.4 × 아티스트 유사도 − 변주 감점
```

제목에 더 큰 가중치를 둡니다. 사람은 아티스트명을 더 자주 줄여 쓰거나 틀리고, 카탈로그도 협업곡의
대표 아티스트만 적는 경우가 많기 때문입니다. 유사도는 편집 거리를 써서 `bigbang`과 `big bang`
같은 표기 차이를 흡수합니다.

| 점수 | 결과 |
| --- | --- |
| 0.86 이상 | 자동 추가 |
| 0.45 ~ 0.86 | `review` — 후보를 돌려주고 사용자에게 물음 |
| 0.45 미만 | `missing` |

**변주 감점**: 요청하지 않은 라이브·리믹스·Inst·커버는 0.25점을 깎습니다. 원곡을 원했는데 라이브가
들어가는 것을 막습니다. 반대로 사용자가 `(Live)`를 적었으면 감점하지 않습니다.

**입력 순서 자동 감지**: `아티스트 - 제목`과 `제목 - 아티스트` 양쪽으로 채점해 점수가 높은 쪽을
씁니다. 붙여넣은 목록이 역순이어도 동작합니다.

**동점 처리**: `빅뱅`과 `BIGBANG`처럼 한글↔로마자 표기는 로컬에서 대조가 불가능해 후보들이 모두
같은 점수가 됩니다. 이때만 Apple의 검색 순위를 타이브레이커로 씁니다. 다만 그 가산점은 **선택에만**
쓰이고 신뢰도에는 반영되지 않아, 검증할 수 없는 매칭은 여전히 `review`로 갑니다.

## 중복 방지

대상 플레이리스트의 기존 트랙을 먼저 읽고 걸러냅니다. **다른 플레이리스트나 라이브러리 전체는 보지
않습니다** — 한 곡을 여러 플레이리스트에 넣는 건 정상이니까요.

같은 곡인지는 두 가지로 판단합니다.

1. **카탈로그 ID** — 정확한 일치. 라이브러리 트랙의 `i.xxx` ID는 카탈로그 ID와 다르므로
   `playParams.catalogId`를 꺼내 비교합니다
2. **정규화된 아티스트 + 제목 + 변주** — 카탈로그에 매칭된 적 없어 ID가 비어 있는 항목을 위한 보조

변주가 키에 포함되므로 `세상이 멈출 때까지`와 `세상이 멈출 때까지 (Acoustic Ver.) [Instrumental]`은
**다른 녹음으로 취급**되어 둘 다 들어갑니다.

한 요청 안에 같은 곡이 두 번 있으면 한 번만 넣습니다.

## 곡 목록 형식

한 줄에 한 곡. 아래 세 가지를 인식합니다.

```
빅뱅 - 붉은 노을
07:19 | 빅뱅 - 붉은 노을
07:19 | 빅뱅 ‘붉은 노을’
```

앞뒤 타임스탬프와 괄호 안 내용을 정리해 `아티스트 - 곡 제목`으로 맞춥니다. 빈 줄은 무시하고,
제목에 ` - `가 들어가는 곡(리믹스·부제)도 처리합니다.

곡 검색은 로그인한 계정의 스토어프론트(`GET /v1/me/storefront`)를 따릅니다. 조회 실패 시 `kr`로
떨어집니다. 이 값은 사용자별로 1시간 캐시합니다.

## 환경 변수

`.env.example`을 `.env`로 복사해 채웁니다.

| 변수 | 기본값 | 설명 |
| --- | --- | --- |
| `TEAM_ID` | (필수) | Apple Developer 팀 ID |
| `KEY_ID` | (필수) | MusicKit 키 ID |
| `PRIVATE_KEY_PATH` | 자동 탐색 | `.p8` 경로. 비우면 실행 디렉터리의 `AuthKey_*.p8`을 찾습니다 |
| `PORT` | `8002` | 리스닝 포트 |
| `ALLOWED_ORIGINS` | 프로덕션 + `localhost:9002` | CORS 허용 오리진 (쉼표 구분) |

`TEAM_ID`·`KEY_ID`·`.p8`은 **같은 팀에서 발급된 것끼리** 짝이 맞아야 합니다. 하나라도 어긋나면
Apple이 401을 돌려줍니다. 헤더는 새 키를 가리키는데 서명은 옛 키로 되는 경우가 있어, 토큰이
"정상"처럼 보여도 실제로는 거부될 수 있습니다. 아래 검증 방법으로 확인하세요.

키가 여러 개 있으면 자동 탐색이 동작하지 않으므로(어느 쪽인지 알 수 없음) `PRIVATE_KEY_PATH`로
하나를 지정해야 합니다.

## 개인 키 다루기

`.p8` 파일은 **저장소에도 도커 이미지에도 넣지 않습니다.** 이 이미지는 공개 레지스트리에 올라가므로
키를 구우면 누구나 `docker pull`로 꺼낼 수 있습니다. 파일 권한 400은 컨테이너 안에서만 의미가 있고
레이어를 받는 것을 막지 못합니다.

서버에서는 런타임에 마운트합니다.

```bash
sudo mkdir -p /data/apple_music_playlist
sudo cp AuthKey_XXXXXXXXXX.p8 /data/apple_music_playlist/
# 컨테이너는 uid 10001(app)로 실행되므로 소유자를 맞춰야 읽힙니다.
sudo chown 10001:10001 /data/apple_music_playlist/AuthKey_*.p8
sudo chmod 400 /data/apple_music_playlist/AuthKey_*.p8
```

`.env`에 절대 경로를 넣습니다.

```
PRIVATE_KEY_PATH=/data/apple_music_playlist/AuthKey_XXXXXXXXXX.p8
```

마운트는 `docker-compose.yml`에 정의되어 있습니다.

## 실행

```bash
go mod download
make run          # 또는 go run main.go
```

| 명령 | 설명 |
| --- | --- |
| `make run` | 로컬 실행 |
| `make server` | `bin/apple_music_playlist` 빌드 |
| `make test` | `go test ./...` |
| `make vet` | `go vet ./...` |
| `make docker` | 이미지 빌드 (멀티스테이지) |
| `make push` | 레지스트리 푸시 |

> `router/router.go`는 직접 관리합니다. 예전 `make server`는 `buildtool-router`로 이 파일을
> 덮어썼지만, 라우터에 에러 처리와 상태 코드 매핑이 들어 있어 코드 생성 대상이 아닙니다.

## 키 검증

배포 전에 개발자 토큰이 실제로 Apple에 통하는지 확인하세요. 사용자 토큰 없이 됩니다.

```bash
TOKEN=$(curl -s localhost:8002/api/token | python3 -c 'import sys,json;print(json.load(sys.stdin)["developerToken"])')

# 헤더/페이로드 확인 — kid 가 KEY_ID, iss 가 TEAM_ID 와 같아야 합니다
echo "$TOKEN" | cut -d. -f1 | base64 -d 2>/dev/null; echo
echo "$TOKEN" | cut -d. -f2 | base64 -d 2>/dev/null; echo

# Apple 이 실제로 받아 주는지
curl -s -o /dev/null -w "%{http_code}\n" \
  -H "Authorization: Bearer $TOKEN" \
  "https://api.music.apple.com/v1/catalog/kr/search?term=iu&types=songs&limit=1"
```

| 응답 | 의미 |
| --- | --- |
| 200 | 정상 |
| 401 | `TEAM_ID` / `KEY_ID` / `.p8` 조합이 안 맞음 |
| 403 | 키에 MusicKit이 활성화되지 않음 (Media ID 연결 확인) |

## 문제 해결

로그에 판단 근거가 남습니다.

```
playlist "여름 밤" → p.xxxx (existing, 34 tracks)     ← 어느 플레이리스트에 몇 곡이 있었는지
playlist "여름 밤" matches 2 playlists (...)          ← 같은 이름이 여러 개일 때
duplicate by catalog-id: "..." → ... (1443621832)     ← 정확한 ID 일치. 오탐 아님
duplicate by name: "..."                              ← 이름 매칭. 오탐 가능성 있음
playlist "...": holding 12 confident matches ...      ← 순서 유지를 위해 보류 중
```

"빈 플레이리스트인데 이미 있음이 뜬다"면 `(existing, N tracks)`의 N을 보세요. N이 0이 아니면
앱이 찾은 플레이리스트와 보고 있는 플레이리스트가 다른 것입니다.

## 구조

```
main.go                     설정, 미들웨어, CORS
router/router.go            라우팅, 에러 → 상태 코드 매핑
models/playlist.go          요청 본문
controllers/rest/
  auth.go                   개발자 토큰 서명·캐싱, 키 탐색
  applemusic.go             Apple Music API 클라이언트
  matching.go               정규화, 채점, 중복 판정
  playlist.go               곡 목록 파싱, 저장 흐름
```

테스트는 `controllers/rest/*_test.go`에 42개 있습니다. 매칭 채점, 중복 판정, 변주 구분, 키 탐색,
스토어프론트 캐시, 401/403 매핑처럼 조용히 망가지기 쉬운 부분을 고정해 둔 것들입니다.

## 문의

[kobums@naver.com](mailto:kobums@naver.com)
