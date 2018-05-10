FROM scratch
LABEL maintainer mail@fleaz.me

COPY ca-certificates.crt /etc/ssl/certs/
COPY main /



EXPOSE 8083
CMD ["/main"]
