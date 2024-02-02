#!/bin/bash

DOMAIN_NAMES=$1
FIRST_DOMAIN=""
JSON_QUERY='{"domains":['
SERVER_ENDPOINT="${SERVER_ENDPOINT:-"http://lego-bridge.apps.k8s.kemo.labs"}"

if [ -z $DOMAIN_NAMES ]; then
    echo "Usage: $0 <domain-name>[;<domain-name>...]"
    exit 1
fi

IFS=';' read -ra DOMAINS <<< "$DOMAIN_NAMES"
for DOMAIN in "${DOMAINS[@]}"; do
    if [ -z $FIRST_DOMAIN ]; then
        FIRST_DOMAIN=$DOMAIN
    fi
    JSON_QUERY="${JSON_QUERY}\"$DOMAIN\","
done

# Cockpit
CERT_PATH="${CERT_PATH:-"/etc/cockpit/ws-certs.d/999-${FIRST_DOMAIN}.cert"}"
KEY_PATH="${KEY_PATH:-"/etc/cockpit/ws-certs.d/999-${FIRST_DOMAIN}.key"}"
RELOAD_CMD="${RELOAD_CMD:-"systemctl restart cockpit.socket"}"

# IDM
#CERT_PATH="${CERT_PATH:-"/etc/cockpit/ws-certs.d/999-${FIRST_DOMAIN}.cert"}"
#KEY_PATH="${KEY_PATH:-"/etc/cockpit/ws-certs.d/999-${FIRST_DOMAIN}.key"}"
#RELOAD_CMD="${RELOAD_CMD:-"systemctl restart cockpit.socket"}"

JSON_QUERY="${JSON_QUERY%?}]}"

# Request certificate
REQUEST=$(curl -sSL --max-time 600 --connect-timeout 600 -X POST -H "Content-Type: application/json" -d $JSON_QUERY ${SERVER_ENDPOINT}/get-certificate)

STATUS=$(echo $REQUEST | jq -r '.status')

if [ $STATUS != "success" ]; then
    echo "Error: $REQUEST"
    exit 1
fi

echo $REQUEST | jq -r '.certificate' > ${CERT_PATH}
echo $REQUEST | jq -r '.key' > ${KEY_PATH}

# Reload service
$(${RELOAD_CMD})
