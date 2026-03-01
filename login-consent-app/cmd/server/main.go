package main

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"log"
	"net/http"
	"os"
	"time"

	"ory-demo/login-consent-app/internal/hydraadmin"
)

// デモ用の “Login/Consent Provider”
// - /login?login_challenge=... を受けて自動 accept → Hydra が返す redirect_to に 302
// - /consent?consent_challenge=... を受けて自動 accept → redirect_to に 302
//
// 注意: 本番ではユーザー認証・同意UI・セッション管理が必須です。
// ここは Hydra とアプリの接続部分の最小限の実装例として、あえて簡略化しています。

func main() {
	addr := env("ADDR", ":3000")
	hydraAdminURL := env("HYDRA_ADMIN_URL", "http://hydra:4445")
	// デモ用: 固定ユーザー（実務では認証結果のユーザーID）
	demoSubject := env("DEMO_SUBJECT", "user-001")

	hydra := hydraadmin.New(hydraAdminURL)

	mux := http.NewServeMux()

	mux.HandleFunc("/health", func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("ok"))
	})

	mux.HandleFunc("/login", func(w http.ResponseWriter, r *http.Request) {
		challenge := r.URL.Query().Get("login_challenge")
		if challenge == "" {
			http.Error(w, "missing login_challenge", http.StatusBadRequest)
			return
		}

		ctx, cancel := context.WithTimeout(r.Context(), 8*time.Second)
		defer cancel()

		// 実務ではここで「ログイン済みか」「認証UIへ誘導」などを行う
		// 今回は自動 accept。
		_, _ = hydra.GetLogin(ctx, challenge) // デモでは内容は使わないが、疎通確認として呼ぶ

		resp, err := hydra.AcceptLogin(ctx, challenge, hydraadmin.AcceptLoginBody{
			Subject:     demoSubject,
			Remember:    true,
			RememberFor: 3600, // 1h
		})
		if err != nil {
			log.Printf("[login] accept error: %v", err)
			http.Error(w, "login accept failed", http.StatusBadGateway)
			return
		}

		log.Printf("[login] accepted subject=%s redirect_to=%s", demoSubject, resp.RedirectTo)
		http.Redirect(w, r, resp.RedirectTo, http.StatusFound)
	})

	mux.HandleFunc("/consent", func(w http.ResponseWriter, r *http.Request) {
		challenge := r.URL.Query().Get("consent_challenge")
		if challenge == "" {
			http.Error(w, "missing consent_challenge", http.StatusBadRequest)
			return
		}

		ctx, cancel := context.WithTimeout(r.Context(), 8*time.Second)
		defer cancel()

		consent, err := hydra.GetConsent(ctx, challenge)
		if err != nil {
			log.Printf("[consent] get error: %v", err)
			http.Error(w, "consent get failed", http.StatusBadGateway)
			return
		}

		// 実務ならrequested_scopeを見てUIで同意を取る
		// デモのためrequested_scopeをそのまま許可
		resp, err := hydra.AcceptConsent(ctx, challenge, hydraadmin.AcceptConsentBody{
			GrantScope: consent.RequestedScope,
			Remember:   true,
		})
		if err != nil {
			log.Printf("[consent] accept error: %v", err)
			http.Error(w, "consent accept failed", http.StatusBadGateway)
			return
		}

		log.Printf("[consent] accepted subject=%s scopes=%v redirect_to=%s", consent.Subject, consent.RequestedScope, resp.RedirectTo)
		http.Redirect(w, r, resp.RedirectTo, http.StatusFound)
	})

	// あると便利なデモ用: stateのダミー生成（後でGoアプリに移す）
	mux.HandleFunc("/", func(w http.ResponseWriter, _ *http.Request) {
		_, _ = w.Write([]byte("login-consent-app: try /health, /login, /consent\nstate example: " + randomHex(8)))
	})

	log.Printf("login-consent-app listening on %s, hydraAdmin=%s", addr, hydraAdminURL)
	if err := http.ListenAndServe(addr, mux); err != nil {
		log.Fatal(err)
	}
}

func env(key, def string) string {
	v := os.Getenv(key)
	if v == "" {
		return def
	}
	return v
}

func randomHex(n int) string {
	b := make([]byte, n)
	_, _ = rand.Read(b)
	return hex.EncodeToString(b)
}
