package main

import (
	"context"
	"crypto/rand"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"net/url"
	"os"
	"time"

	"github.com/gorilla/sessions"
	"github.com/lestrrat-go/jwx/v2/jwk"
	"github.com/lestrrat-go/jwx/v2/jwt"
)

type Config struct {
	Addr                string
	BaseURL             string // Go自身のURL (callback生成に使用)
	HydraPublicInternal string
	HydraPublicBrowser  string
	ClientID            string
	ClientSecret        string
	RedirectURI         string // http://localhost:8080/api/callback
	JWKSURL             string // http://hydra:4444/.well-known/jwks.json (container内は http://hydra:4444/...)
	Issuer              string // http://localhost:4444/
	ExpectedAud         string // demo-client
	SessionSecret       string
	VueAppURL           string // http://localhost:5173/app
}

func main() {
	cfg := Config{
		Addr:                env("ADDR", ":8080"),
		BaseURL:             env("BASE_URL", "http://localhost:8080"),
		HydraPublicInternal: env("HYDRA_PUBLIC_INTERNAL_URL", "http://hydra:4444"),
		HydraPublicBrowser:  env("HYDRA_PUBLIC_BROWSER_URL", "http://localhost:4444"),
		ClientID:            env("OIDC_CLIENT_ID", "demo-client"),
		ClientSecret:        env("OIDC_CLIENT_SECRET", "demo-secret"),
		RedirectURI:         env("OIDC_REDIRECT_URI", "http://localhost:8080/api/callback"),
		JWKSURL:             env("OIDC_JWKS_URL", "http://hydra:4444/.well-known/jwks.json"),
		Issuer:              env("OIDC_ISSUER", "http://localhost:4444/"),
		ExpectedAud:         env("OIDC_EXPECTED_AUD", "demo-client"),
		SessionSecret:       env("SESSION_SECRET", "please-change-this-32chars-min"),
		VueAppURL:           env("VUE_APP_URL", "http://localhost:5173/app"),
	}

	store := sessions.NewCookieStore([]byte(cfg.SessionSecret))
	store.Options = &sessions.Options{
		Path:     "/",
		MaxAge:   3600,
		HttpOnly: true,
		SameSite: http.SameSiteLaxMode,
		// ローカルHTTPなので Secure=false。HTTPSなら true。
		Secure: false,
	}

	// JWKS キャッシュ（デモ用：起動後に初回取得、以後一定時間で更新）
	var jwksCache jwk.Set
	var jwksFetchedAt time.Time

	getJWKS := func(ctx context.Context) (jwk.Set, error) {
		if jwksCache != nil && time.Since(jwksFetchedAt) < 5*time.Minute {
			return jwksCache, nil
		}
		req, _ := http.NewRequestWithContext(ctx, http.MethodGet, cfg.JWKSURL, nil)
		res, err := http.DefaultClient.Do(req)
		if err != nil {
			return nil, err
		}
		defer res.Body.Close()
		b, _ := io.ReadAll(res.Body)
		set, err := jwk.Parse(b)
		if err != nil {
			return nil, err
		}
		jwksCache, jwksFetchedAt = set, time.Now()
		return set, nil
	}

	mux := http.NewServeMux()

	// CORS（Vue dev server 用：最低限）
	mux.HandleFunc("/api/me", func(w http.ResponseWriter, r *http.Request) {
		allowDevCORS(w, r)

		sess, _ := store.Get(r, "demo_session")
		sub, _ := sess.Values["sub"].(string)
		if sub == "" {
			http.Error(w, "unauthorized", http.StatusUnauthorized)
			return
		}

		writeJSON(w, map[string]any{
			"sub": sub,
		})
	})

	mux.HandleFunc("/api/login", func(w http.ResponseWriter, r *http.Request) {
		sess, _ := store.Get(r, "demo_session")

		state := randomURLSafe(24)
		nonce := randomURLSafe(24)
		sess.Values["state"] = state
		sess.Values["nonce"] = nonce
		if err := sess.Save(r, w); err != nil {
			http.Error(w, "session save failed", http.StatusInternalServerError)
			return
		}

		// Hydra authorize URL
		u, _ := url.Parse(cfg.HydraPublicBrowser + "/oauth2/auth")
		q := u.Query()
		q.Set("client_id", cfg.ClientID)
		q.Set("response_type", "code")
		q.Set("scope", "openid")
		q.Set("redirect_uri", cfg.RedirectURI)
		q.Set("state", state)
		q.Set("nonce", nonce) // OIDC nonce
		u.RawQuery = q.Encode()

		log.Printf("[login] redirect_to=%s", u.String())
		http.Redirect(w, r, u.String(), http.StatusFound)
	})

	mux.HandleFunc("/api/callback", func(w http.ResponseWriter, r *http.Request) {
		code := r.URL.Query().Get("code")
		state := r.URL.Query().Get("state")
		if code == "" || state == "" {
			http.Error(w, "missing code/state", http.StatusBadRequest)
			return
		}

		sess, _ := store.Get(r, "demo_session")
		wantState, _ := sess.Values["state"].(string)
		wantNonce, _ := sess.Values["nonce"].(string)
		if wantState == "" || state != wantState {
			log.Printf("[callback] STATE_MISMATCH want=%s got=%s", wantState, state)
			http.Error(w, "state mismatch", http.StatusUnauthorized)
			return
		}

		// token exchange
		ctx, cancel := context.WithTimeout(r.Context(), 10*time.Second)
		defer cancel()

		tokenRes, err := exchangeCode(ctx, cfg.HydraPublicInternal+"/oauth2/token", cfg.ClientID, cfg.ClientSecret, code, cfg.RedirectURI)
		if err != nil {
			log.Printf("[callback] TOKEN_EXCHANGE_FAILED: %v", err)
			http.Error(w, "token exchange failed", http.StatusUnauthorized)
			return
		}

		idTokenStr, _ := tokenRes["id_token"].(string)
		if idTokenStr == "" {
			http.Error(w, "missing id_token", http.StatusUnauthorized)
			return
		}

		// verify id_token
		jwks, err := getJWKS(ctx)
		if err != nil {
			log.Printf("[callback] JWKS_FETCH_FAILED: %v", err)
			http.Error(w, "jwks fetch failed", http.StatusBadGateway)
			return
		}

		tok, err := jwt.Parse([]byte(idTokenStr), jwt.WithKeySet(jwks), jwt.WithValidate(true))
		if err != nil {
			log.Printf("[callback] JWT_INVALID: %v", err)
			http.Error(w, "invalid token", http.StatusUnauthorized)
			return
		}

		// claims check (iss/aud/nonce)
		iss := tok.Issuer()
		if iss != cfg.Issuer {
			log.Printf("[callback] ISS_MISMATCH want=%s got=%s", cfg.Issuer, iss)
			http.Error(w, "issuer mismatch", http.StatusUnauthorized)
			return
		}
		aud := tok.Audience()
		okAud := false
		for _, a := range aud {
			if a == cfg.ExpectedAud {
				okAud = true
				break
			}
		}
		if !okAud {
			log.Printf("[callback] AUD_MISMATCH want=%s got=%v", cfg.ExpectedAud, aud)
			http.Error(w, "audience mismatch", http.StatusUnauthorized)
			return
		}
		nonce, _ := tok.Get("nonce")
		if wantNonce != "" && nonce != wantNonce {
			log.Printf("[callback] NONCE_MISMATCH want=%s got=%v", wantNonce, nonce)
			http.Error(w, "nonce mismatch", http.StatusUnauthorized)
			return
		}

		sub := tok.Subject()
		sess.Values["sub"] = sub
		delete(sess.Values, "state")
		delete(sess.Values, "nonce")
		if err := sess.Save(r, w); err != nil {
			http.Error(w, "session save failed", http.StatusInternalServerError)
			return
		}

		log.Printf("[callback] success sub=%s", sub)
		http.Redirect(w, r, cfg.VueAppURL, http.StatusFound)
	})

	mux.HandleFunc("/health", func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("ok"))
	})

	log.Printf("go-app listening on %s", cfg.Addr)
	log.Fatal(http.ListenAndServe(cfg.Addr, mux))
}

func exchangeCode(ctx context.Context, tokenURL, clientID, clientSecret, code, redirectURI string) (map[string]any, error) {
	form := url.Values{}
	form.Set("grant_type", "authorization_code")
	form.Set("code", code)
	form.Set("redirect_uri", redirectURI)

	req, _ := http.NewRequestWithContext(ctx, http.MethodPost, tokenURL, stringsReader(form.Encode()))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.SetBasicAuth(clientID, clientSecret)

	res, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer res.Body.Close()

	b, _ := io.ReadAll(res.Body)
	if res.StatusCode < 200 || res.StatusCode >= 300 {
		return nil, fmt.Errorf("status=%d body=%s", res.StatusCode, string(b))
	}

	var out map[string]any
	if err := json.Unmarshal(b, &out); err != nil {
		return nil, err
	}
	return out, nil
}

// small helpers

func allowDevCORS(w http.ResponseWriter, r *http.Request) {
	origin := r.Header.Get("Origin")
	if origin == "http://localhost:5173" {
		w.Header().Set("Access-Control-Allow-Origin", origin)
		w.Header().Set("Access-Control-Allow-Credentials", "true")
	}
	if r.Method == http.MethodOptions {
		w.Header().Set("Access-Control-Allow-Methods", "GET,OPTIONS")
		w.Header().Set("Access-Control-Allow-Headers", "Content-Type")
		w.WriteHeader(http.StatusNoContent)
	}
}

func writeJSON(w http.ResponseWriter, v any) {
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(v)
}

func randomURLSafe(n int) string {
	b := make([]byte, n)
	_, _ = rand.Read(b)
	return base64.RawURLEncoding.EncodeToString(b)
}

func env(k, def string) string {
	if v := os.Getenv(k); v != "" {
		return v
	}
	return def
}

// stringsReader avoids importing strings in a big way; minimal helper.
func stringsReader(s string) io.Reader { return &stringReader{s: s} }

type stringReader struct {
	s string
	i int64
}

func (r *stringReader) Read(p []byte) (int, error) {
	if r.i >= int64(len(r.s)) {
		return 0, io.EOF
	}
	n := copy(p, r.s[r.i:])
	r.i += int64(n)
	return n, nil
}
