#!/usr/bin/env bash
# Issues Let's Encrypt certificates for the Payminto hostnames and switches
# nginx to HTTPS with an 80 -> 443 redirect. Safe to re-run.
#
# HTTP-01 validation requires every name to already resolve to this host.
set -euo pipefail

EXPECT_IP=${EXPECT_IP:-54.234.10.215}
EMAIL=${CERTBOT_EMAIL:-sagarblockchaindev@gmail.com}
NAMES=(payminto.io www.payminto.io app.payminto.io checkout.payminto.io)
AUTH_NS=${AUTH_NS:-$(dig +short NS payminto.io | sort | head -1)}
[ -n "$AUTH_NS" ] || { echo "cannot resolve authoritative nameserver for payminto.io"; exit 1; }
echo "checking against authoritative nameserver $AUTH_NS"

missing=()
for name in "${NAMES[@]}"; do
  # Ask the zone's authoritative nameserver, not a public resolver: Let's Encrypt
  # validates against authoritative data, while a resolver can hold a negative
  # cache for a freshly added name for up to the SOA minimum (3600s here).
  got=$(dig +short "$name" A @"$AUTH_NS" | tr '\n' ' ')
  case " $got " in
    *" $EXPECT_IP "*) printf '  ok      %-24s -> %s\n' "$name" "$got" ;;
    *) printf '  PENDING %-24s -> %s\n' "$name" "${got:-<no record>}"; missing+=("$name") ;;
  esac
done

if (( ${#missing[@]} )); then
  echo
  echo "Not issuing yet: ${#missing[@]} name(s) do not resolve to $EXPECT_IP."
  echo "Add/propagate the A records, then re-run this script."
  exit 2
fi

args=()
for name in "${NAMES[@]}"; do args+=(-d "$name"); done

sudo certbot --nginx --non-interactive --agree-tos --email "$EMAIL" \
  --redirect --keep-until-expiring "${args[@]}"

sudo nginx -t && sudo systemctl reload nginx
echo "TLS issued and nginx reloaded."
