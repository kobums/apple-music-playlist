# Apple Music Playlist Manager

이 프로젝트는 Apple Music API를 사용하여 사용자의 플레이리스트를 관리하는 기능을 구현합니다. 사용자는 기존 플레이리스트에 노래를 추가하거나 새 플레이리스트를 생성할 수 있습니다.

## 기능

- 기존 플레이리스트 조회
- 노래 검색 및 플레이리스트에 추가
- 새 플레이리스트 생성

## 설정 방법

프로젝트를 사용하기 전에 다음과 같은 환경 설정이 필요합니다.

### 필수 파일 및 디렉터리 구조

1. **.env 파일**: 프로젝트 루트에 위치해야 하며, 다음 환경 변수를 포함해야 합니다:

   - `TEAM_ID`: Apple Developer 계정의 Team ID
   - `KEY_ID`: Apple Music API Key ID
   - `USER_TOKEN`: Apple Music API를 사용하기 위한 사용자 토큰

   `.env` 파일 예시:

   ```plaintext
   TEAM_ID=your_team_id_here
   KEY_ID=your_key_id_here
   USER_TOKEN=your_user_token_here
   ```

2. **AuthKey 파일 (`AUTHKEY.p8`)**: Apple Music API 키 파일 (Private Key)이며, 프로젝트 루트 디렉터리에 위치해야 합니다.

### playlist.txt 작성법

`playlist.txt` 파일은 플레이리스트와 추가하려는 노래의 정보를 포함해야 합니다. 각 노래는 새 줄에 입력되어야 하며, 다음 형식을 따라야 합니다:

```plaintext
playlistname: My Favorite Songs
노래1 제목 - 아티스트1
노래2 제목 - 아티스트2
```

## 설치 및 실행 방법

1. **의존성 설치**: Go 환경이 설정되어 있어야 합니다. 필요한 모든 의존성을 설치하려면 프로젝트 디렉터리에서 다음 명령을 실행하세요:

   ```bash
   go mod tidy
   ```

2. **프로그램 실행**: 프로젝트 디렉터리에서 다음 명령을 실행하여 프로그램을 실행합니다:
   ```bash
   go run main.go
   ```

## 문의 사항

프로젝트에 대한 추가 문의 사항이 있을 경우 [여기](kobums@naver.com)로 문의해 주세요.
