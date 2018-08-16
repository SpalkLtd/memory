FROM golang:1.10-alpine3.8

RUN mkdir -p /go/src/github.com/SpalkLtd/synchroniser/memory
RUN apk add git
ADD . /go/src/github.com/SpalkLtd/synchroniser/memory
RUN cd /go/src/github.com/SpalkLtd/synchroniser/memory && go get -t -v ./... && go test