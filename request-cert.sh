#!/bin/bash

DOMAIN_NAMES=$1
FIRST_DOMAIN=""
JSON_QUERY='{"domains":['
SERVER_ENDPOINT="${SERVER_ENDPOINT:-"http://localhost:8080"}"


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

JSON_QUERY="${JSON_QUERY%?}]}"

echo $JSON_QUERY

set -x

REQUEST=$(curl -sSL --max-time 600 --connect-timeout 600 -X POST -H "Content-Type: application/json" -d $JSON_QUERY ${SERVER_ENDPOINT}/get-certificate)

STATUS=$(echo $REQUEST | jq -r '.status')

if [ $STATUS != "success" ]; then
    echo "Error: $REQUEST"
    exit 1
fi

echo $REQUEST | jq -r '.certificate' > ./$FIRST_DOMAIN.crt.pem
echo $REQUEST | jq -r '.key' > ./$FIRST_DOMAIN.key.pem
