#!/usr/bin/env bash
# 生成自签名证书，用于在没有正式证书时先启用 HTTPS。
# 用法: ./scripts/gen-self-signed-cert.sh [域名或IP]
# 产物: certs/fullchain.pem certs/privkey.pem（已被 .gitignore 忽略）
set -euo pipefail

DOMAIN="${1:-localhost}"
DIR="$(cd "$(dirname "$0")/.." && pwd)/certs"
mkdir -p "$DIR"

if [ -f "$DIR/fullchain.pem" ]; then
    echo "证书已存在: $DIR/fullchain.pem（如需重新生成请先删除）"
    exit 0
fi

openssl req -x509 -nodes -newkey rsa:2048 -days 825 \
    -keyout "$DIR/privkey.pem" -out "$DIR/fullchain.pem" \
    -subj "/CN=${DOMAIN}" \
    -addext "subjectAltName=DNS:${DOMAIN},IP:127.0.0.1"

chmod 600 "$DIR/privkey.pem"
echo "已生成自签名证书: $DIR"
echo "浏览器会提示证书不受信任，正式上线请换成 Let's Encrypt 或商业证书。"
