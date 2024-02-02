FROM quay.io/fedora/fedora:39 as builder

WORKDIR /opt/app

COPY go.mod .
COPY go.sum .
COPY main.go .

RUN dnf install -y go \
 && go mod download \
 && go build -o /tmp/lego-bridge main.go

FROM quay.io/fedora/fedora:39

RUN dnf install -y bind-utils curl jq \
 && dnf clean all \
 && rm -rf /var/cache/dnf

WORKDIR /opt/app

COPY dns-ping.sh .
COPY request-cert.sh .
COPY --from=builder /tmp/lego-bridge .
ADD https://raw.githubusercontent.com/kenmoini/homelab/main/pki/root-authorities/klstep-ca.pem /etc/pki/ca-trust/source/anchors/klstep-ca.pem

RUN chmod +x /opt/app/lego-bridge \
 && chmod +x /opt/app/dns-ping.sh \
 && chmod +x /opt/app/request-cert.sh \
 && update-ca-trust

USER 1001
EXPOSE 8080

CMD ["/opt/app/lego-bridge"]
