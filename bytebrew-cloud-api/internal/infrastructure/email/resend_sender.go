package email

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"net/http"
)

// ResendSender sends emails via the Resend API.
type ResendSender struct {
	apiKey      string
	fromEmail   string
	frontendURL string
	client      *http.Client
}

// NewResendSender creates a new ResendSender.
func NewResendSender(apiKey, fromEmail, frontendURL string) *ResendSender {
	return &ResendSender{
		apiKey:      apiKey,
		fromEmail:   fromEmail,
		frontendURL: frontendURL,
		client:      &http.Client{},
	}
}

// SendTeamInvite sends a team invitation email.
func (s *ResendSender) SendTeamInvite(ctx context.Context, email, teamName, inviteToken string) error {
	subject := fmt.Sprintf("You've been invited to join %s on ByteBrew", teamName)
	html := fmt.Sprintf(`
<div style="font-family: sans-serif; max-width: 600px; margin: 0 auto;">
  <h2>Team Invitation</h2>
  <p>You've been invited to join <strong>%s</strong> on ByteBrew.</p>
  <p>Use the following token to accept the invitation:</p>
  <p style="background: #f4f4f4; padding: 12px; font-family: monospace; font-size: 14px; border-radius: 4px;">%s</p>
  <p>Or use the CLI command:</p>
  <pre style="background: #f4f4f4; padding: 12px; border-radius: 4px;">bytebrew team accept --token %s</pre>
  <p style="color: #666; font-size: 12px;">This invitation expires in 7 days.</p>
</div>
`, teamName, inviteToken, inviteToken)

	return s.send(ctx, email, subject, html)
}

// SendPasswordReset sends a password reset email with a link containing the token.
func (s *ResendSender) SendPasswordReset(ctx context.Context, email, token string) error {
	subject := "Reset Your Password"
	resetURL := s.frontendURL + "/reset-password?token=" + token
	html := fmt.Sprintf(`
<div style="font-family: sans-serif; max-width: 600px; margin: 0 auto;">
  <h2>Reset Your Password</h2>
  <p>Click the link below to reset your password:</p>
  <a href="%s" style="display: inline-block; padding: 12px 24px; background: #4F46E5; color: white; text-decoration: none; border-radius: 6px;">Reset Password</a>
  <p style="color: #666; font-size: 12px; margin-top: 16px;">This link expires in 1 hour. If you didn't request this, you can safely ignore this email.</p>
</div>
`, resetURL)

	return s.send(ctx, email, subject, html)
}

type resendRequest struct {
	From    string   `json:"from"`
	To      []string `json:"to"`
	Subject string   `json:"subject"`
	HTML    string   `json:"html"`
}

func (s *ResendSender) send(ctx context.Context, to, subject, html string) error {
	body := resendRequest{
		From:    s.fromEmail,
		To:      []string{to},
		Subject: subject,
		HTML:    html,
	}

	payload, err := json.Marshal(body)
	if err != nil {
		return fmt.Errorf("marshal email request: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, "https://api.resend.com/emails", bytes.NewReader(payload))
	if err != nil {
		return fmt.Errorf("create email request: %w", err)
	}

	req.Header.Set("Authorization", "Bearer "+s.apiKey)
	req.Header.Set("Content-Type", "application/json")

	resp, err := s.client.Do(req)
	if err != nil {
		return fmt.Errorf("send email: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode >= 400 {
		respBody, _ := io.ReadAll(resp.Body)
		slog.ErrorContext(ctx, "resend API error", "status", resp.StatusCode, "body", string(respBody))
		return fmt.Errorf("resend API error: status %d", resp.StatusCode)
	}

	return nil
}

// NoopSender is a no-op email sender for development/testing.
type NoopSender struct{}

// NewNoopSender creates a no-op email sender that logs instead of sending.
func NewNoopSender() *NoopSender {
	return &NoopSender{}
}

// SendTeamInvite logs the invite instead of sending an email.
func (s *NoopSender) SendTeamInvite(ctx context.Context, email, teamName, inviteToken string) error {
	slog.InfoContext(ctx, "team invite email (noop)",
		"to", email,
		"team", teamName,
		"token", inviteToken,
	)
	return nil
}

// SendPasswordReset logs the password reset instead of sending an email.
func (s *NoopSender) SendPasswordReset(ctx context.Context, email, token string) error {
	slog.InfoContext(ctx, "password reset email (noop)",
		"to", email,
		"token", token,
	)
	return nil
}
