#!/bin/bash
#set -x

DOMAIN_NAME=$1
MAX_SECONDS=600
SECONDS=0
SUCCESS_LIMIT=20
DNS_SERVER_ONE="${DNS_SERVER_ONE:-"192.168.42.9"}"
DNS_SERVER_TWO="${DNS_SERVER_TWO:-"192.168.42.10"}"

if [ -z $DOMAIN_NAME ]; then
    echo "Usage: $0 <domain-name>"
    exit 1
fi

function digDomain() {
    echo "[INFO] [${DOMAIN_NAME#\*.}] Checking DNS record"
    dnsPingONE=$(dig @${DNS_SERVER_ONE} -t txt +short _acme-challenge.${DOMAIN_NAME#\*.})
    dnsPingTWO=$(dig @${DNS_SERVER_TWO} -t txt +short _acme-challenge.${DOMAIN_NAME#\*.})
    if [ -z $dnsPingONE ] || [ -z $dnsPingTWO ] ; then
        echo "[WARN] [${DOMAIN_NAME}] DNS record not found"
        sleep 5
        if [ $SECONDS -gt $MAX_SECONDS ]; then
            echo "[ERROR] [${DOMAIN_NAME}] DNS record not found after $MAX_SECONDS seconds"
            exit 1
        fi
        SECONDS=$(($SECONDS+5))
        digDomain
    else
        if [ $SUCCESS_LIMIT -lt 0 ]; then
            echo "[INFO] [${DOMAIN_NAME}] DNS record found - stopping after $((SECONDS-5)) seconds"
            exit 0
        fi
        echo "[INFO] [${DOMAIN_NAME}] DNS record found"
        sleep 5
        SECONDS=$(($SECONDS+5))
        SUCCESS_LIMIT=$(($SUCCESS_LIMIT-1))
        digDomain
    fi
}

digDomain