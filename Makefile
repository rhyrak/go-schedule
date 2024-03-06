all: run

build:
	@go build -o bin/CourseScheduler.exe cmd/CourseScheduler/main.go

run: build
	@bin/CourseScheduler.exe