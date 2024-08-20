# 1단계: 빌드 환경 설정
FROM golang:1.22 AS builder

WORKDIR /app

COPY . .

RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -a -ldflags '-s' -o bin/apple_music_playlist.linux main.go

# 2단계: 실행 환경 설정
FROM alpine

# Go 바이너리를 복사합니다.
COPY --from=builder /app/bin/apple_music_playlist.linux /usr/local/main/main

# .p8 파일을 이미지에 포함시키기
COPY AuthKey_GXVS6H2456.p8 /usr/local/main/AuthKey_GXVS6H2456.p8

# .p8 파일의 권한 설정
RUN chmod 600 /usr/local/main/AuthKey_GXVS6H2456.p8

# 설정 파일 및 필요한 디렉토리 생성 (필요에 따라 주석 해제)
#COPY ./config/config.json.docker /usr/local/main/config/config.json
CMD mkdir -p /usr/local/main/webdata

# 정적 파일 및 뷰 파일 복사 (필요에 따라 주석 해제)
#ADD ./assets /usr/local/main/assets
#ADD ./views /usr/local/main/views

WORKDIR /usr/local/main

# 컨테이너 시작 시 실행할 명령어
CMD ./main