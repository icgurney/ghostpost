ARG GO_VERSION=1.23.4
FROM golang:${GO_VERSION}-alpine as builder

WORKDIR /usr/src/mail
COPY go.mod go.sum ./
RUN go mod download && go mod verify
COPY . .
RUN go build -v -o /run-mail cmd/mail/main.go

FROM alpine:latest

COPY --from=builder /run-mail /usr/local/bin/
CMD ["run-mail"]
