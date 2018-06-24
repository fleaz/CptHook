FROM alpine
LABEL maintainer mail@fleaz.me

RUN apk add --no-cache ca-certificates
COPY templates/ /templates
COPY webhook-gateway /
EXPOSE 8086
CMD ["/webhook-gateway"]
