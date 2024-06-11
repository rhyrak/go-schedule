all: run

build_sa:
	go build -o bin/sa_backend cmd/sa/main.go cmd/sa/handlers.go

build:
	@go build -o bin/CourseScheduler cmd/CourseScheduler/main.go

run: build
	@bin/CourseScheduler

clean:
	@rm schedule.csv -f && rm bin/CourseScheduler -f