package hydraadmin

import (
	"context"
	"net/url"

	"ory-demo/login-consent-app/internal/httpx"
)

type Client struct {
	c *httpx.Client
}

func New(adminBaseURL string) *Client {
	return &Client{c: httpx.New(adminBaseURL)}
}

// Login

type GetLoginRequest struct {
	Challenge string
}

type LoginRequest struct {
	Subject string `json:"subject"`
}

func (c *Client) GetLogin(ctx context.Context, challenge string) (*LoginRequest, error) {
	q := url.Values{}
	q.Set("login_challenge", challenge)

	var out LoginRequest
	if err := c.c.GetJSON(ctx, "/admin/oauth2/auth/requests/login?"+q.Encode(), &out); err != nil {
		return nil, err
	}
	return &out, nil
}

type AcceptLoginBody struct {
	// subject: ログインしたユーザーID扱い
	Subject     string `json:"subject"`
	Remember    bool   `json:"remember"`
	RememberFor int    `json:"remember_for"`
	// ACR などはデモのため省略
}

type AcceptLoginResponse struct {
	RedirectTo string `json:"redirect_to"`
}

func (c *Client) AcceptLogin(ctx context.Context, challenge string, body AcceptLoginBody) (*AcceptLoginResponse, error) {
	q := url.Values{}
	q.Set("login_challenge", challenge)

	var out AcceptLoginResponse
	if err := c.c.PutJSON(ctx, "/admin/oauth2/auth/requests/login/accept?"+q.Encode(), body, &out); err != nil {
		return nil, err
	}
	return &out, nil
}

// Consent

type ConsentRequest struct {
	// リクエストされたスコープなど（実務ならここで同意判断）
	RequestedScope []string `json:"requested_scope"`
	Subject        string   `json:"subject"`
}

func (c *Client) GetConsent(ctx context.Context, challenge string) (*ConsentRequest, error) {
	q := url.Values{}
	q.Set("consent_challenge", challenge)

	var out ConsentRequest
	if err := c.c.GetJSON(ctx, "/admin/oauth2/auth/requests/consent?"+q.Encode(), &out); err != nil {
		return nil, err
	}
	return &out, nil
}

type AcceptConsentBody struct {
	GrantScope []string `json:"grant_scope"`
	Remember   bool     `json:"remember"`
	// 付与クレームはデモのため省略
}

type AcceptConsentResponse struct {
	RedirectTo string `json:"redirect_to"`
}

func (c *Client) AcceptConsent(ctx context.Context, challenge string, body AcceptConsentBody) (*AcceptConsentResponse, error) {
	q := url.Values{}
	q.Set("consent_challenge", challenge)

	var out AcceptConsentResponse
	if err := c.c.PutJSON(ctx, "/admin/oauth2/auth/requests/consent/accept?"+q.Encode(), body, &out); err != nil {
		return nil, err
	}
	return &out, nil
}

// Logout

type LogoutRequest struct {
	Subject string `json:"subject"`
}

func (c *Client) GetLogout(ctx context.Context, challenge string) (*LogoutRequest, error) {
	q := url.Values{}
	q.Set("logout_challenge", challenge)

	var out LogoutRequest
	if err := c.c.GetJSON(ctx, "/admin/oauth2/auth/requests/logout?"+q.Encode(), &out); err != nil {
		return nil, err
	}
	return &out, nil
}

type AcceptLogoutResponse struct {
	RedirectTo string `json:"redirect_to"`
}

func (c *Client) AcceptLogout(ctx context.Context, challenge string) (*AcceptLogoutResponse, error) {
	q := url.Values{}
	q.Set("logout_challenge", challenge)

	var out AcceptLogoutResponse
	if err := c.c.PutJSON(ctx, "/admin/oauth2/auth/requests/logout/accept?"+q.Encode(), map[string]any{}, &out); err != nil {
		return nil, err
	}
	return &out, nil
}
