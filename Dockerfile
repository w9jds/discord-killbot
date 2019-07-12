FROM golang:latest as builder

LABEL version="v2.4"
LABEL repository="https://github.com/w9jds/discord-killbot"
LABEL homepage="https://github.com/w9jds/discord-killbot"
LABEL maintainer="Jeremy Shore <w9jds@github.com>"

WORKDIR /go/src/killbot/cmd

COPY . /go/src/killbot

RUN go get -d ./...
RUN CGO_ENABLED=0 GOOS=linux go build -ldflags '-w -s' -a -installsuffix cgo -o killbot
RUN curl -o ca-certificates.crt https://raw.githubusercontent.com/bagder/ca-bundle/master/ca-bundle.crt

FROM scratch

WORKDIR /go/src/killbot/cmd

COPY --from=builder /go/src/killbot/cmd/ca-certificates.crt /etc/ssl/certs/
COPY --from=builder /go/src/killbot/pkg /go/src/killbot/pkg
COPY --from=builder /go/src/killbot/cmd/killbot /go/src/killbot/cmd

CMD ["./killbot"]
