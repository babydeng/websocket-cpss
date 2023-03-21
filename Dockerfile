# WITH Go Modules
FROM golang:alpine AS builder
RUN apk update && apk add --no-cache git

RUN mkdir $GOPATH/src/ws_server

ADD ./server.go $GOPATH/src/ws_server

WORKDIR $GOPATH/src/ws_server
RUN go env -w GOPROXY=https://goproxy.cn,direct
RUN go mod init
RUN go mod tidy
RUN go mod download
RUN mkdir /pro
RUN go build -o /pro/ws_server server.go

FROM alpine:latest
RUN mkdir /pro
COPY --from=builder /pro/ws_server /pro/ws_server
EXPOSE 1234
WORKDIR /pro
CMD ["/pro/ws_server"]
