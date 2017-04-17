FROM scratch

MAINTAINER roman@cryptopay.me

ADD hooker /

CMD ["/hooker", "--help"]
