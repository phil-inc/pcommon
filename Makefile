generate:
	go generate ./...

install:
	go install go.uber.org/mock/mockgen@latest
	go get -u github.com/golang/mock/mockgen/model
