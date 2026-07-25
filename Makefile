tag=latest

all: server

# router/router.go 는 이제 직접 관리한다.
# 예전 server 타깃은 buildtool-router 로 router.go 를 덮어썼는데,
# 라우터에 에러 처리와 상태 코드 매핑이 들어가면서 코드 생성으로는 표현할 수 없게 됐다.
server: dummy
	go build -o bin/apple_music_playlist main.go

run:
	go run main.go

test: dummy
	go test ./...

vet:
	go vet ./...

fmt:
	gofmt -l -w .

linux:
	env GOOS=linux GOARCH=amd64 go build -o bin/apple_music_playlist.linux main.go

# 이미지 빌드는 멀티스테이지 Dockerfile 이 처리하므로 호스트 빌드에 의존하지 않는다.
docker:
	docker build --platform linux/amd64 -t kobums/apple_music_playlist:$(tag) .

dockerrun:
	docker run --env-file .env --platform linux/amd64 -d --name="apple_music_playlist" -p 8002:8002 kobums/apple_music_playlist:$(tag)

push: docker
	docker push kobums/apple_music_playlist:$(tag)

clean:
	rm -f bin/apple_music_playlist bin/apple_music_playlist.linux

dummy:
