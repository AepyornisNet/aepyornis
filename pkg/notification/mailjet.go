package notification

import (
	"context"
	"fmt"

	"github.com/mailjet/mailjet-apiv3-go/v4"
)

// MailjetOption describes a functional parameter for the Mailgun constructor.
type MailjetOption func(*Mailjet)

// Mailjet struct holds necessary data to communicate with the Mailjet API.
type Mailjet struct {
	client            *mailjet.Client
	senderAddress     string
	senderName        string
	receiverAddresses []string
}

func NewMailjet(apiKeyPublic, apiKeyPrivate, senderAddress, senderName string, opts ...MailjetOption) *Mailjet {
	m := &Mailjet{
		client:            mailjet.NewMailjetClient(apiKeyPublic, apiKeyPrivate),
		senderAddress:     senderAddress,
		senderName:        senderName,
		receiverAddresses: []string{},
	}

	for _, opt := range opts {
		opt(m)
	}

	return m
}

// AddReceivers takes email addresses and adds them to the internal address list. The Send method will send
// a given message to all those addresses.
func (m *Mailjet) AddReceivers(addresses ...string) {
	m.receiverAddresses = append(m.receiverAddresses, addresses...)
}

// Send takes a message subject and a message body and sends them to all previously set chats. Message body supports
// html as markup language.
func (m Mailjet) Send(ctx context.Context, subject, message string) error {
	if len(m.receiverAddresses) < 1 {
		return nil
	}

	messagesInfo := []mailjet.InfoMessagesV31{
		{
			From: &mailjet.RecipientV31{
				Email: m.senderAddress,
				Name:  m.senderName,
			},
			To: &mailjet.RecipientsV31{
				mailjet.RecipientV31{
					Email: m.receiverAddresses[0],
				},
			},
			Subject:  subject,
			TextPart: message,
			// HTMLPart: "",
		},
	}
	messages := mailjet.MessagesV31{Info: messagesInfo}

	_, err := m.client.SendMailV31(&messages)
	if err != nil {
		return fmt.Errorf("send message: %w", err)
	}

	return nil
}
