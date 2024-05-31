FROM golang:1.22-alpine3.19 as build

WORKDIR /app

# RUN apk add build-base
RUN apk update --no-cache && apk --no-cache add tzdata

ADD go.mod .
# ADD go.sum .
RUN go mod download

COPY . .

RUN go build -ldflags "-s -w"  -o ./exec metrics_prometheus_grafana/main.go

FROM alpine:3.19

WORKDIR /app

COPY --from=build /usr/share/zoneinfo /usr/share/zoneinfo
ENV TZ=Europe/London

COPY --from=build /app/exec ./