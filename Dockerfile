FROM ubuntu:latest

RUN apt-get update && \
    apt-get -uy upgrade && \
    apt-get install -y ca-certificates software-properties-common gpg && \
    add-apt-repository ppa:longsleep/golang-backports && \
    apt-get update && \
    update-ca-certificates
RUN apt-get -y install ed git golang-go make

ADD . /coredns-pns/
RUN chmod 755 coredns-pns/build.sh && coredns-pns/build.sh

FROM ubuntu:latest

RUN apt-get update && \
    apt-get -uy upgrade && \
    apt-get -y install lsof

COPY --from=0 /etc/ssl/certs /etc/ssl/certs
COPY --from=0 /coredns /coredns

ENTRYPOINT ["/coredns"]
