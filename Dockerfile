FROM alpine
LABEL maintainer mail@fleaz.me

RUN apk add --no-cache ca-certificates
COPY templates/ /templates
COPY CptHook /
EXPOSE 8086
CMD ["/CptHook"]
