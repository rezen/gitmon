FROM golang:1.12-alpine

RUN apk update && apk upgrade && \
    apk add --no-cache bash git openssh alpine-sdk

ADD https://github.com/golang/dep/releases/download/v0.5.4/dep-linux-amd64 /usr/bin/dep
RUN chmod +x /usr/bin/dep

ENV GOARCH=amd64 
ENV GO_ENABLED=0 
ENV GOOS=linux

RUN mkdir -p $GOPATH/src/github.com/rezen/gitmon
ADD . $GOPATH/src/github.com/rezen/gitmon
WORKDIR $GOPATH/src/github.com/rezen/gitmon
RUN dep ensure --vendor-only

RUN mkdir -p /app
RUN  go build -v -o /app/server ./cmd/gitmon/main.go
RUN /app/server --help
CMD ["/app/server"]