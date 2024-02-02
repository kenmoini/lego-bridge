# Lego Bridge

This is a microservice to bridge between ACME requests to Step CA and PowerDNS.

## Usage

### Required Input

- `PDNS_API_URL`
- `PDNS_API_KEY`
- `ACME_SERVER_URL`
- `EMAIL_ADDRESS`

### Optional Input

- `DNS_SERVERS` - A list of DNS servers to query for a response, separated by a semi-colon.  Also set `DNS_SERVER_ONE` and `DNS_SERVER_TWO`

```bash
# With Podman
podman run --rm -d --name lego-bridge \
 -p 8080:8080 \
 -e PDNS_API_URL="http://pdns-api.example.com:8081" \
 -e PDNS_API_KEY="somekeyhere" \
 -e ACME_SERVER_URL="https://step-ca.example.com/acme/acme/directory" \
 -e EMAIL_ADDRESS="you@example.com" \
 -e DNS_SERVERS="192.168.42.9,192.168.42.10" \
 -e DNS_SERVER_ONE="192.168.42.9" \
 -e DNS_SERVER_TWO="192.168.42.10" \
 quay.io/kenmoini/lego-bridge:latest

# On Kubernetes - Secret edits needed
kubectl apply -k deployment/
```

Now you should be able to make a cURL to the service:

```
export SERVER_ENDPOINT="http://lego-bridge.apps.k8s.kemo.labs"

./request-cert.sh "test.example.com"
./request-cert.sh "test.example.com;other-test.example.com"
```

Which will save the certificate to `./$FIRST_DOMAIN.crt.pem` and the key to `./$FIRST_DOMAIN.key.pem`