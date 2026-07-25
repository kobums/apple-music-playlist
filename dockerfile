# 1단계: 빌드
FROM golang:1.25-alpine AS builder

WORKDIR /app

# 의존성 레이어를 소스와 분리해 캐시가 살아 있게 한다.
COPY go.mod go.sum ./
RUN go mod download

COPY . .

RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 \
    go build -trimpath -ldflags '-s -w' -o /out/apple_music_playlist .

# 2단계: 실행
FROM alpine:3.22

# Apple Music API 는 HTTPS 이므로 루트 인증서가 필요하고,
# 로그 타임스탬프를 위해 tzdata 를 넣는다.
RUN apk add --no-cache ca-certificates tzdata \
    && adduser -D -u 10001 app

WORKDIR /usr/local/main

COPY --from=builder /out/apple_music_playlist ./main

# MusicKit 개인 키.
# 파일명에 Key ID 가 들어가서 키를 교체할 때마다 바뀌므로 글롭으로 복사하고,
# 실제 경로는 .env 의 PRIVATE_KEY_PATH 로 지정한다.
COPY --chown=app:app AuthKey_*.p8 ./
RUN chmod 400 ./AuthKey_*.p8

# 예전 이미지는 `CMD mkdir -p ...` 을 썼는데, CMD 는 마지막 하나만 적용되므로
# 이 디렉터리는 실제로 만들어지지 않았다.
RUN mkdir -p /usr/local/main/webdata && chown app:app /usr/local/main/webdata

USER app

ENV PORT=8002
EXPOSE 8002

CMD ["./main"]
