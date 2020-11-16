# Build the manager binary
FROM golang:1.13 as builder

WORKDIR /workspace
# Copy the Go Modules manifests
COPY go.mod go.mod
COPY go.sum go.sum
# cache deps before building and copying source so that we don't need to re-download as much
# and so that source changes don't invalidate our downloaded layer
RUN go mod download

# Copy the go source
COPY main.go main.go
COPY api/ api/
COPY controllers/ controllers/

# Build
RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 GO111MODULE=on go build -a -o manager main.go

# Use distroless as minimal base image to package the manager binary
# Refer to https://github.com/GoogleContainerTools/distroless for more details

# FROM gcr.io/distroless/static:nonroot
FROM php:7.4-apache-buster
ENV TZ='Asia/Tokyo'
WORKDIR /
COPY --from=builder /workspace/manager .

RUN apt update && apt install -y git
RUN git clone -b 5.3.0 https://github.com/phpredis/phpredis.git /usr/src/php/ext/redis
RUN docker-php-ext-install redis && docker-php-ext-enable redis

RUN mkdir /template && mkdir /var/www/html/dummy
COPY ./config/template/vhosts.tmpl /template
COPY ./config/dockerConfig/apache2.conf /etc/apache2
COPY ./config/dockerConfig/security.conf /etc/apache2/conf-enabled

ENTRYPOINT ["/manager"]
