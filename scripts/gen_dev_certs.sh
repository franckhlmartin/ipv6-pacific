#!/usr/bin/env bash
# Generate a self-signed TLS certificate for local HTTPS (localhost + 127.0.0.1 + ::1).
set -euo pipefail
ROOT="$(cd "$(dirname "$0")/.." && pwd)"
OUT="$ROOT/certs"
mkdir -p "$OUT"

CNF="$OUT/openssl-inline.cnf"
cat > "$CNF" <<'EOF'
[req]
distinguished_name = req_distinguished_name
x509_extensions = v3_req
prompt = no

[req_distinguished_name]
CN = localhost

[v3_req]
basicConstraints = CA:FALSE
keyUsage = digitalSignature, keyEncipherment
extendedKeyUsage = serverAuth
subjectAltName = @alt_names

[alt_names]
DNS.1 = localhost
IP.1 = 127.0.0.1
IP.2 = ::1
EOF

openssl req -x509 -newkey rsa:4096 -sha256 -days 825 -nodes \
  -keyout "$OUT/key.pem" \
  -out "$OUT/cert.pem" \
  -config "$CNF" \
  -extensions v3_req

rm -f "$CNF"
echo "Wrote $OUT/cert.pem and $OUT/key.pem (gitignored)."
