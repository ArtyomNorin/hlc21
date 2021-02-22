FROM golang:alpine

RUN mkdir -p /tmp/hlc/app

RUN apk add git zip

COPY . /tmp/hlc/app

WORKDIR /tmp/hlc/app

RUN chmod +x start.sh

RUN go get && go build

ENTRYPOINT /tmp/hlc/app/start.sh