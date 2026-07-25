# Apple Music Playlist — 백엔드

Apple Music API로 사용자의 라이브러리 플레이리스트를 관리하는 Go(Fiber) 서버입니다.
같은 이름의 플레이리스트가 있으면 거기에 곡을 추가하고, 없으면 새로 만듭니다.

## API

| 메서드 | 경로 | 설명 |
| --- | --- | --- |
| `GET` | `/api/token` | MusicKit 개발자 토큰 발급 |
| `POST` | `/api/playlist` | 곡 목록을 파싱해 플레이리스트에 추가 |

`POST /api/playlist` 요청 본문:

```json
{
  "title": "My favorite songs",
  "list": "빅뱅 - 붉은 노을\n아이유 - 밤편지",
  "userToken": "<Music User Token>"
}
```

응답:

```json
{
  "code": "ok",
  "result": [{ "song": "빅뱅 - 붉은 노을", "status": true }]
}
```

실패 시에는 상태 코드와 함께 `{ "code": "error", "error": "..." }`를 돌려줍니다.
`developerToken` 필드는 하위 호환을 위해 받기만 하고 무시합니다. 서버가 직접 서명합니다.

## 환경 변수

`.env.example`을 `.env`로 복사해 채웁니다.

| 변수 | 기본값 | 설명 |
| --- | --- | --- |
| `TEAM_ID` | (필수) | Apple Developer 팀 ID |
| `KEY_ID` | (필수) | MusicKit 키 ID |
| `PRIVATE_KEY_PATH` | 자동 탐색 | `.p8` 개인 키 경로. 비우면 실행 디렉터리의 `AuthKey_*.p8` 을 찾습니다 |
| `PORT` | `8002` | 리스닝 포트 |
| `ALLOWED_ORIGINS` | 프로덕션 + `localhost:9002` | CORS 허용 오리진 (쉼표 구분) |

`.p8` 개인 키 파일은 저장소에 커밋하지 않습니다(`.gitignore`에 등록되어 있습니다).

`TEAM_ID`·`KEY_ID`·`.p8` 는 반드시 같은 팀에서 발급된 것끼리 짝이 맞아야 합니다. 하나라도
어긋나면 Apple 이 401 을 돌려줍니다. 키를 교체해서 구 키와 신 키가 함께 있으면 자동 탐색이
동작하지 않으므로(어느 쪽을 쓸지 알 수 없음) `PRIVATE_KEY_PATH` 로 하나를 지정하세요.

## 곡 목록 형식

한 줄에 한 곡씩 적습니다. 아래 세 가지 형태를 인식합니다.

```plaintext
빅뱅 - 붉은 노을
07:19 | 빅뱅 - 붉은 노을
07:19 | 빅뱅 ‘붉은 노을’
```

앞뒤 타임스탬프와 괄호 안 내용은 제거하고 `아티스트 - 곡 제목`으로 정규화합니다.
빈 줄은 무시합니다.

곡 검색은 로그인한 계정의 스토어프론트(`GET /v1/me/storefront`)를 따릅니다.
조회에 실패하면 `kr`로 떨어집니다.

## 실행

```bash
go mod download
make run          # 또는 go run main.go
```

```bash
make vet          # go vet
make server       # bin/apple_music_playlist 빌드
make docker       # 이미지 빌드 (멀티스테이지)
```

> `router/router.go`는 직접 관리합니다. 예전 `make server`는 `buildtool-router`로 이 파일을
> 덮어썼지만, 이제 라우터에 에러 처리와 상태 코드 매핑이 들어 있어 코드 생성 대상이 아닙니다.

## 문의

[kobums@naver.com](mailto:kobums@naver.com)
