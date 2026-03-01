# ORY Hydra demo

## Setup

```bash
cp .env.example .env
docker compose up -d
docker compose ps
```

## Verify

curl -sS http://localhost:4444/health/ready
curl -sS http://localhost:4444/.well-known/openid-configuration
curl -sS http://localhost:4444/.well-known/jwks.json
