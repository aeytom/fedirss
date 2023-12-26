FROM golang:alpine AS builder

# Install git.
# Git is required for fetching the dependencies.
RUN apk update && apk add --no-cache git gcc g++

WORKDIR /go/src/app
COPY . .
RUN go get -d -v
RUN GOOS=linux GOARCH=amd64 CGO_ENABLED=1 go install -ldflags="-w -s"

FROM alpine:latest

ARG APP_USER=fedilpd
ARG APP_GID=1000
ARG APP_UID=1000
ARG APP_DIR=/app

COPY --from=builder /go/bin/fedilpd /bin
ENTRYPOINT ["/bin/fedilpd"]

RUN addgroup -g $APP_GID $APP_USER \
    && adduser -h $APP_DIR -G $APP_USER -u $APP_UID -D $APP_USER  
WORKDIR $APP_DIR
USER $APP_USER
COPY env.sample.yaml $APP_DIR
VOLUME $APP_DIR