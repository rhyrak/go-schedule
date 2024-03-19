all: run

build:
	@go build -o bin/CourseScheduler cmd/CourseScheduler/main.go

run: build
	@bin/CourseScheduler

clean:
	@rm schedule.csv -f && rm bin/CourseScheduler -f