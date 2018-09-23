FROM golang:latest as build

RUN mkdir -p /go/src/github.com/aspacca/keyvaluestorage/
WORKDIR /go/src/github.com/aspacca/keyvaluestorage/

RUN go get -d -v github.com/gorilla/mux && \
	go get -d -v github.com/PuerkitoBio/ghost/handlers && \
	go get -d -v github.com/sirupsen/logrus && \
	go get -d -v github.com/minio/cli

ADD . .

RUN CGO_ENABLED=0 go build -v -a -o /usr/bin/server

FROM alpine:latest

COPY --from=build /usr/bin/server /root/

EXPOSE 8080
WORKDIR /root/

CMD ["./server", "--basedir", "./", "--provider", "fs"]