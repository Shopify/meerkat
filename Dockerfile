FROM golang:1.12-alpine AS builder

LABEL maintainer "bshafiee@uwaterloo.ca"
# Install tools required to build the project (gcc and build-base are required for statd cgo)
RUN apk add --no-cache git openssh-client build-base gcc &&\
  go get github.com/golang/dep/cmd/dep &&\
  go get github.com/Shopify/ejson/cmd/ejson

COPY . /go/src/github.com/meerkat
WORKDIR /go/src/github.com/meerkat
# build the go binary fully static with all the crap linked staticly
RUN dep ensure -vendor-only && go build

# Start with a clean layer again and just copy binaries
FROM alpine:latest
#update ca root for TLS handshakes
RUN apk --no-cache --update add ca-certificates git

COPY --from=builder /go/src/github.com/meerkat/meerkat /go/bin/ejson \
  /apps/production/

#entrypoint
WORKDIR /apps/production
ENTRYPOINT ["/apps/production/entrypoint.sh"]
