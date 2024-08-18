tag=latest

all: server

server: dummy
	buildtool-model ./ 
	buildtool-router ./ > ./router/router.go
	go build -o bin/apple_music_playlist main.go

fswatch:
	fswatch -0 controllers | xargs -0 -n1 build/notify.sh

run:
	gin --port 8002 -a 8002 --bin bin/apple_music_playlist run main.go

allrun:
	fswatch -0 controllers | xargs -0 -n1 build/notify.sh &
	gin --port 8002 -a 8002 --bin bin/apple_music_playlist run main.go

test: dummy
	go test -v ./...

linux:
	env GOOS=linux GOARCH=amd64 go build -o bin/apple_music_playlist.linux main.go

dockerbuild:
	env CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -a -ldflags '-s' -o bin/apple_music_playlist.linux main.go

docker: dockerbuild
	docker build --platform linux/amd64 -t kobums/apple_music_playlist:$(tag) .

dockerrun:
	docker run --platform -d --name="apple_music_playlist" -p 8002:8002 kobums/apple_music_playlist

push: docker
	docker push kobums/apple_music_playlist:$(tag)

clean:
	rm -f bin/apple_music_playlist

dummy: