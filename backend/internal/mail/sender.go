package mail

import (
	"context"
	"fmt"
	"os"
	"strings"

	"github.com/resend/resend-go/v2"
	"go.uber.org/zap"
)

// Sender handles email sending
type Sender struct {
	client    *resend.Client
	fromEmail string
	fromName  string
	log       *zap.Logger
}

// NewSender creates a new email sender
func NewSender(log *zap.Logger) *Sender {
	apiKey := os.Getenv("RESEND_API_KEY")
	return &Sender{
		client:    resend.NewClient(apiKey),
		fromEmail: os.Getenv("EMAIL_FROM"),
		fromName:  os.Getenv("EMAIL_FROM_NAME"),
		log:       log,
	}
}

// SendTransactional sends a single email
func (s *Sender) SendTransactional(
	ctx context.Context,
	to string,
	subject string,
	html string,
) (string, error) {
	if s.client == nil {
		s.log.Warn("email sender not configured")
		return "", nil
	}

	from := fmt.Sprintf(
		"%s <%s>", s.fromName, s.fromEmail,
	)

	params := &resend.SendEmailRequest{
		From:    from,
		To:      []string{to},
		Subject: subject,
		Html:    html,
	}

	resp, err := s.client.Emails.Send(params)
	if err != nil {
		return "", fmt.Errorf(
			"mail.Sender.SendTransactional: %w", err,
		)
	}

	s.log.Info("email sent",
		zap.String("to", to),
		zap.String("subject", subject),
		zap.String("id", resp.Id),
	)

	return resp.Id, nil
}

// SendWelcome sends welcome email to new contact
func (s *Sender) SendWelcome(
	ctx context.Context,
	to string,
	name string,
	workspaceName string,
) error {
	subject := fmt.Sprintf(
		"Welcome to %s", workspaceName,
	)

	displayName := name
	if displayName == "" {
		displayName = "there"
	}

	html := fmt.Sprintf(`
<!DOCTYPE html>
<html>
<body style="font-family:sans-serif;
             max-width:600px;
             margin:0 auto;
             padding:40px 20px">
  <h2>Welcome!</h2>
  <p>Hi %s,</p>
  <p>Thanks for signing up. We are excited
     to have you on board.</p>
  <p>If you have any questions, just reply
     to this email.</p>
  <p>Best,<br>%s</p>
</body>
</html>`, displayName, workspaceName)

	_, err := s.SendTransactional(
		ctx, to, subject, html,
	)
	return err
}

// BuildHTML builds simple HTML email
func BuildHTML(
	title string,
	body string,
	ctaText string,
	ctaURL string,
) string {
	cta := ""
	if ctaText != "" && ctaURL != "" {
		cta = fmt.Sprintf(`
<a href="%s" style="background:#0052FF;
   color:white;padding:12px 24px;
   text-decoration:none;
   border-radius:8px;
   display:inline-block;
   margin-top:16px">%s</a>`,
			ctaURL, ctaText,
		)
	}

	return fmt.Sprintf(`
<!DOCTYPE html>
<html>
<body style="font-family:sans-serif;
             max-width:600px;
             margin:0 auto;
             padding:40px 20px">
  <h2 style="color:#0052FF">%s</h2>
  <div>%s</div>
  %s
  <p style="color:#888;
            font-size:12px;
            margin-top:40px">
    Sent via ZipDesk
  </p>
</body>
</html>`,
		title,
		strings.ReplaceAll(body, "\n", "<br>"),
		cta,
	)
}
