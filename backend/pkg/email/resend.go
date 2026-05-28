package email

import (
    "context"
    "fmt"
    "strings"

    "github.com/resend/resend-go/v2"
)

// Client wraps Resend email sending
type Client struct {
    resend    *resend.Client
    fromEmail string
    fromName  string
}

// Config holds email configuration
type Config struct {
    ResendAPIKey string
    FromEmail    string
    FromName     string
}

// EmailRequest defines an email to send
type EmailRequest struct {
    To      []string
    Subject string
    HTML    string
    Text    string
    ReplyTo string
    Tags    map[string]string
}

// New creates a new email client
func New(cfg Config) *Client {
    return &Client{
        resend:    resend.NewClient(cfg.ResendAPIKey),
        fromEmail: cfg.FromEmail,
        fromName:  cfg.FromName,
    }
}

// Send sends a transactional email via Resend
func (c *Client) Send(
    ctx context.Context,
    req EmailRequest,
) (string, error) {
    from := fmt.Sprintf("%s <%s>", c.fromName, c.fromEmail)
    if req.ReplyTo != "" {
        from = fmt.Sprintf("%s <%s>", c.fromName, c.fromEmail)
    }

    params := &resend.SendEmailRequest{
        From:    from,
        To:      req.To,
        Subject: req.Subject,
        Html:    req.HTML,
        Text:    req.Text,
    }

    resp, err := c.resend.Emails.Send(params)
    if err != nil {
        return "", fmt.Errorf("email.Send: %w", err)
    }

    return resp.Id, nil
}

// SendTemplate sends an email using a template
func (c *Client) SendTemplate(
    ctx context.Context,
    to []string,
    subject string,
    templateName string,
    data map[string]interface{},
) (string, error) {
    html, err := renderTemplate(templateName, data)
    if err != nil {
        return "", fmt.Errorf("email.SendTemplate: render: %w", err)
    }

    return c.Send(ctx, EmailRequest{
        To:      to,
        Subject: subject,
        HTML:    html,
    })
}

// renderTemplate renders an email template
func renderTemplate(name string, data map[string]interface{}) (string, error) {
    templates := map[string]string{
        "welcome": welcomeEmailTemplate,
        "verify":  verifyEmailTemplate,
        "reset":   resetEmailTemplate,
        "notify":  notifyEmailTemplate,
    }

    tmpl, ok := templates[name]
    if !ok {
        return "", fmt.Errorf("template %q not found", name)
    }

    result := tmpl
    for key, val := range data {
        placeholder := fmt.Sprintf("{{%s}}", key)
        result = strings.ReplaceAll(result, placeholder, fmt.Sprintf("%v", val))
    }

    return result, nil
}

const welcomeEmailTemplate = `
<!DOCTYPE html>
<html>
<body style="font-family:sans-serif;max-width:600px;margin:0 auto;padding:40px 20px">
  <h1 style="color:#0052FF">Welcome to ZipDesk</h1>
  <p>Hi {{name}},</p>
  <p>Your account is ready. Start building with AI.</p>
  <a href="{{dashboard_url}}"
     style="background:#0052FF;color:white;padding:12px 24px;
            text-decoration:none;border-radius:8px;display:inline-block">
    Open Dashboard
  </a>
  <p style="color:#888;font-size:12px;margin-top:40px">ZipDesk</p>
</body>
</html>
`

const verifyEmailTemplate = `
<!DOCTYPE html>
<html>
<body style="font-family:sans-serif;max-width:600px;margin:0 auto;padding:40px 20px">
  <h1 style="color:#0052FF">Verify your email</h1>
  <p>Hi {{name}},</p>
  <p>Click below to verify your email address.</p>
  <a href="{{verify_url}}"
     style="background:#0052FF;color:white;padding:12px 24px;
            text-decoration:none;border-radius:8px;display:inline-block">
    Verify Email
  </a>
  <p style="color:#888;font-size:12px">Link expires in 24 hours.</p>
</body>
</html>
`

const resetEmailTemplate = `
<!DOCTYPE html>
<html>
<body style="font-family:sans-serif;max-width:600px;margin:0 auto;padding:40px 20px">
  <h1 style="color:#0052FF">Reset your password</h1>
  <p>Hi {{name}},</p>
  <p>Click below to reset your password.</p>
  <a href="{{reset_url}}"
     style="background:#0052FF;color:white;padding:12px 24px;
            text-decoration:none;border-radius:8px;display:inline-block">
    Reset Password
  </a>
  <p style="color:#888;font-size:12px">Link expires in 1 hour.</p>
</body>
</html>
`

const notifyEmailTemplate = `
<!DOCTYPE html>
<html>
<body style="font-family:sans-serif;max-width:600px;margin:0 auto;padding:40px 20px">
  <h2 style="color:#0052FF">{{title}}</h2>
  <p>{{message}}</p>
  <p style="color:#888;font-size:12px;margin-top:40px">ZipDesk Notifications</p>
</body>
</html>
`
