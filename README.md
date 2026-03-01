# ORY Hydra ローカルデモ

Hydra を認可サーバーとして使い、以下の最小構成で OIDC Authorization Code Flow を確認するデモです。

- `hydra`: OAuth2 / OIDC サーバー
- `login-consent-app`: Hydra の Login / Consent / Logout challenge を自動 accept するデモ実装
- `go-app`: ログイン開始、コールバック、IDトークン検証、セッション管理
- `frontend`: Vue 画面（`/` で Login、`/app` で `GET /api/me` 確認）

## 構成

| Service | Port | Role |
| --- | --- | --- |
| `hydra` | `4444` / `4445` | Public / Admin API |
| `postgres` | `5432` | Hydra DB |
| `login-consent` | `3000` | Login/Consent/Logout の accept |
| `go-app` | `8080` | OIDCクライアント + セッションAPI |
| `frontend` | `5173` | Vue UI |

## 前提

- Docker / Docker Compose
- `curl`
- `jq`（`scripts/create-client.sh` 実行時に必要）

## セットアップ

```bash
cp .env.example .env
docker compose up -d --build
docker compose ps
```

起動確認:

```bash
curl -sS http://localhost:4444/health/ready
curl -sS http://localhost:3000/health
curl -sS http://localhost:8080/health
curl -sS http://localhost:4444/.well-known/openid-configuration
curl -sS http://localhost:4444/.well-known/jwks.json
```

## OAuth2 クライアント作成（初回）

```bash
./scripts/create-client.sh
```

デフォルト値:

- `client_id`: `demo-client`
- `client_secret`: `demo-secret`
- `redirect_uri`: `http://localhost:8080/api/callback`
- `post_logout_redirect_uri`: `http://localhost:8080/api/logout/callback`

`client_id` が既に存在する場合は一度削除して再作成します。

```bash
curl -sS -X DELETE http://localhost:4445/admin/clients/demo-client
./scripts/create-client.sh
```

## 動作確認（ブラウザ）

1. `http://localhost:5173` を開く
2. `Login` をクリック
3. Hydra -> login-consent-app -> Hydra -> `go-app /api/callback` とリダイレクト
4. `http://localhost:5173/app` に遷移し、`GET /api/me => 200` と `sub` が表示される
5. `Logout` をクリックしてログアウト確認

## 主要エンドポイント

- Frontend: `http://localhost:5173`
- Go app:
  - `GET /api/login`
  - `GET /api/callback`
  - `GET /api/me`
  - `GET /api/logout`
  - `GET /api/logout/callback`
- Hydra:
  - Discovery: `http://localhost:4444/.well-known/openid-configuration`
  - JWKS: `http://localhost:4444/.well-known/jwks.json`
  - Admin Clients: `http://localhost:4445/admin/clients`

## ログ確認

```bash
docker compose logs -f hydra login-consent go-app frontend
```

## 停止 / クリーンアップ

```bash
docker compose down
```

DBボリュームも消す場合:

```bash
docker compose down -v
```
