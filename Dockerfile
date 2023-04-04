# WITH Go Modules
FROM golang:alpine AS builder
RUN apk update && apk add --no-cache git

RUN mkdir $GOPATH/src/send_face_event

ADD ./perceptionUnit.go $GOPATH/src/send_face_event

WORKDIR $GOPATH/src/send_face_event
RUN go env -w GOPROXY=https://goproxy.cn,direct
RUN go mod init
RUN go mod tidy
RUN go mod download
RUN mkdir /pro
RUN go build -o /pro/send_face_event perceptionUnit.go

FROM alpine:latest
RUN mkdir /pro
COPY --from=builder /pro/send_face_event /pro/send_face_event
EXPOSE 1234
WORKDIR /pro
CMD ["/pro/send_face_event"]
