FROM golang:1.16.2-alpine3.13

ENV GO111MODULE=on
ENV GOPROXY=https://goproxy.cn,direct

WORKDIR /hotspot
COPY . .

RUN go get -d -v ./...
RUN go install -v ./...

EXPOSE 80
CMD ["hotspot-online"]