## build
FROM golang:1.16.2-alpine3.13 as builder

ENV GO111MODULE=on \
    GOPROXY=https://goproxy.cn,direct

#RUN apk add --no-cache gcc musl-dev libxml2-dev libxslt-dev

WORKDIR /hotspot
COPY . .

RUN go get -d -v ./...
RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -ldflags '-w -s' -a -installsuffix cgo -o /go/bin/hotspot

## smallest image
FROM scratch

ENV HOTSPOT_TIMER_DURATION=10 \
    HOTSPOT_HTTP_PORT=80

COPY --from=builder /go/bin/hotspot /go/bin/hotspot

EXPOSE 80
CMD ["/go/bin/hotspot"]