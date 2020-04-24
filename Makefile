prog = sudo ./main -debug
all: build
	sudo ./main 1.1.1.1
build:
	go build main.go

test: build
	$(prog) ethohampton.com
	$(prog)	cloudflare.com
	$(prog) 1.1.1.1
	$(prog) 2601:1c1:8b02:a29:7cd2:e065:d60b:6741
	$(prog) asdfasdf
	$(prog) 1420909.124214.14290421.124