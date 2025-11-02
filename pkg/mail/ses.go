package mail

import (
	"context"
	"fmt"
	"strings"

	"github.com/aws/aws-sdk-go-v2/aws"
	awsconfig "github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/credentials"
	"github.com/aws/aws-sdk-go-v2/service/sesv2"
	sestypes "github.com/aws/aws-sdk-go-v2/service/sesv2/types"
)

// SESMailer adalah implementasi Mailer menggunakan AWS SES v2.
type SESMailer struct {
	client           *sesv2.Client
	defaultFrom      string
	configurationSet string
}

// NewSESMailer membuat Mailer SES baru menggunakan AWS SDK v2 (SESv2).
// region dan from wajib. Jika accessKeyID/secretAccessKey kosong, akan memakai default provider chain.
func NewSESMailer(region, accessKeyID, secretAccessKey, defaultFrom, configurationSet string) (Mailer, error) {
	region = strings.TrimSpace(region)

	opts := []func(*awsconfig.LoadOptions) error{}
	if region != "" {
		opts = append(opts, awsconfig.WithRegion(region))
	}
	if strings.TrimSpace(accessKeyID) != "" {
		creds := credentials.NewStaticCredentialsProvider(accessKeyID, secretAccessKey, "")
		opts = append(opts, awsconfig.WithCredentialsProvider(creds))
	}

	cfg, err := awsconfig.LoadDefaultConfig(context.Background(), opts...)
	if err != nil {
		return nil, fmt.Errorf("ses: load config: %w", err)
	}

	client := sesv2.NewFromConfig(cfg)

	return &SESMailer{
		client:           client,
		defaultFrom:      strings.TrimSpace(defaultFrom),
		configurationSet: strings.TrimSpace(configurationSet),
	}, nil
}

// Send mengirim email menggunakan AWS SES v2 dengan konten MIME mentah.
// To/Cc/Bcc akan diisi di Destination (envelope). Header kustom, body, dan attachments dibangun oleh BuilMime.
func (m *SESMailer) Send(ctx context.Context, msg *Message) error {
	if msg == nil {
		return fmt.Errorf("ses: message is nil")
	}

	from := strings.TrimSpace(msg.From)
	if from == "" {
		from = m.defaultFrom
	}
	if from == "" {
		return fmt.Errorf("ses: from is empty")
	}

	// Bangun raw MIME (headers + body + attachments)
	raw, err := BuildMime(from, msg)
	if err != nil {
		return fmt.Errorf("ses: build mime: %w", err)
	}

	to := NormalizeEmails(msg.To)
	cc := NormalizeEmails(msg.Cc)
	bcc := NormalizeEmails(msg.Bcc)

	input := &sesv2.SendEmailInput{
		FromEmailAddress: aws.String(from),
		Content: &sestypes.EmailContent{
			Raw: &sestypes.RawMessage{
				Data: raw,
			},
		},
	}

	// Destination (envelope); opsional untuk raw email, namun lebih eksplisit.
	if len(to) > 0 || len(cc) > 0 || len(bcc) > 0 {
		input.Destination = &sestypes.Destination{
			ToAddresses:  to,
			CcAddresses:  cc,
			BccAddresses: bcc,
		}
	}

	// Configuration set opsional
	if m.configurationSet != "" {
		input.ConfigurationSetName = aws.String(m.configurationSet)
	}
	// Email tags (SES MessageTag)
	if len(msg.Tags) > 0 {
		for k, v := range msg.Tags {
			input.EmailTags = append(input.EmailTags, sestypes.MessageTag{
				Name:  aws.String(k),
				Value: aws.String(v),
			})
		}
	}

	if _, err := m.client.SendEmail(ctx, input); err != nil {
		return fmt.Errorf("ses: send email: %w", err)
	}

	return nil
}
