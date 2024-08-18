FROM golang:1.22 AS builder

WORKDIR /app

COPY . .

RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -a -ldflags '-s' -o bin/apple_music_playlist.linux main.go

FROM        alpine

COPY --from=builder /app/bin/apple_music_playlist.linux /usr/local/main/main
# COPY ./config/config.json.docker /usr/local/main/config/config.json
CMD mkdir -p /usr/local/main/webdata
#ADD ./assets /usr/local/main/assets
#ADD ./views /usr/local/main/views

WORKDIR /usr/local/main
CMD ./main