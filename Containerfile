FROM quay.io/fedora/fedora:39 as builder

WORKDIR /opt/app

COPY go.mod .
COPY go.sum .
COPY main.go .

RUN dnf install -y go \
 && go mod download \
 && go build -o /tmp/lego-bridge main.go

FROM quay.io/fedora/fedora:39

RUN dnf install -y bind-utils jq \
 && dnf clean all \
 && rm -rf /var/cache/dnf

WORKDIR /opt/app

COPY dns-ping.sh .
COPY request-cert.sh .
COPY --from=builder /tmp/lego-bridge .

RUN chmod +x /opt/app/lego-bridge \
 && chmod +x /opt/app/dns-ping.sh \
 && chmod +x /opt/app/request-cert.sh

USER 1001
EXPOSE 8080

CMD ["/opt/app/lego-bridge"]
