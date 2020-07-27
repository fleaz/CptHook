FROM golang:1.14 AS builder
WORKDIR /go/src/cpthook
COPY . /go/src/cpthook
RUN CGO_ENABLED=0 GOOS=linux go build

FROM alpine:latest  
LABEL maintainer mail@fleaz.me
RUN apk --no-cache add ca-certificates
COPY --from=builder /go/src/cpthook/CptHook /
CMD ["./CptHook"]
