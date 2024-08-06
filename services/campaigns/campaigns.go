package campaigns

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"

	"github.com/aws/aws-sdk-go-v2/service/sqs"
	"github.com/cbroglie/mustache"

	"github.com/mailbadger/app/config"
	"github.com/mailbadger/app/entities"
	awssqs "github.com/mailbadger/app/sqs"
	"github.com/mailbadger/app/storage"
)

type Service interface {
	PrepareSubscriberEmailData(
		s entities.Subscriber,
		msg entities.CampaignerTopicParams,
		campaignID int64,
		html *mustache.Template,
		sub *mustache.Template,
		text *mustache.Template,
	) (*entities.SenderTopicParams, error)
	PublishSubscriberEmailParams(ctx context.Context, params *entities.SenderTopicParams, queueURL *string) error
}

// service implements the Service interface
type service struct {
	db                storage.Storage
	sqsclient         awssqs.SendReceiveMessageAPI
	unsubscribeSecret string
	appURL            string
}

func From(db storage.Storage, sqsclient awssqs.SendReceiveMessageAPI, conf config.Config) Service {
	return New(
		db,
		sqsclient,
		conf.Server.UnsubscribeSecret,
		conf.Server.AppURL,
	)
}

func New(
	db storage.Storage,
	sqsclient awssqs.SendReceiveMessageAPI,
	secret string,
	appURL string,
) Service {
	return &service{
		db:                db,
		sqsclient:         sqsclient,
		unsubscribeSecret: secret,
		appURL:            appURL,
	}
}

func (svc *service) PrepareSubscriberEmailData(
	s entities.Subscriber,
	msg entities.CampaignerTopicParams,
	campaignID int64,
	html *mustache.Template,
	sub *mustache.Template,
	text *mustache.Template,
) (*entities.SenderTopicParams, error) {

	var (
		htmlBuf bytes.Buffer
		subBuf  bytes.Buffer
		textBuf bytes.Buffer
	)

	m, err := s.GetMetadata()
	if err != nil {
		return nil, fmt.Errorf("campaign service: prepare email data: get metadata: %w", err)
	}
	// merge sub metadata with default template metadata
	for k, v := range msg.TemplateData {
		if _, ok := m[k]; !ok {
			m[k] = v
		}
	}

	if s.Name != "" {
		m[entities.TagName] = s.Name
	}

	url, err := s.GetUnsubscribeURL(msg.UserUUID, svc.unsubscribeSecret, svc.appURL)
	if err != nil {
		return nil, fmt.Errorf("campaign service: get unsubscribe url: %w", err)
	}

	m[entities.TagUnsubscribeUrl] = url

	err = html.FRender(&htmlBuf, m)
	if err != nil {
		return nil, fmt.Errorf("campaign service: prepare email data: render html: %w", err)
	}
	err = sub.FRender(&subBuf, m)
	if err != nil {
		return nil, fmt.Errorf("campaign service: prepare email data: render subject: %w", err)
	}
	err = text.FRender(&textBuf, m)
	if err != nil {
		return nil, fmt.Errorf("campaign service: prepare email data: render text: %w", err)
	}

	sender := entities.SenderTopicParams{
		EventID:                msg.EventID,
		SubscriberID:           s.ID,
		SubscriberEmail:        s.Email,
		Source:                 msg.Source,
		ConfigurationSetExists: msg.ConfigurationSetExists,
		CampaignID:             campaignID,
		SesKeys:                msg.SesKeys,
		HTMLPart:               htmlBuf.Bytes(),
		SubjectPart:            subBuf.Bytes(),
		TextPart:               textBuf.Bytes(),
		UserUUID:               msg.UserUUID,
		UserID:                 msg.UserID,
	}

	return &sender, nil

}

func (svc *service) PublishSubscriberEmailParams(ctx context.Context, params *entities.SenderTopicParams, queueURL *string) error {
	senderBytes, err := json.Marshal(params)
	if err != nil {
		return fmt.Errorf("campaign service: publish to sender: marshal params: %w", err)
	}

	body := string(senderBytes)

	_, err = svc.sqsclient.SendMessage(ctx, &sqs.SendMessageInput{
		MessageBody: &body,
		QueueUrl:    queueURL,
	})
	if err != nil {
		return fmt.Errorf("campaign service: publish to sender: %w", err)
	}

	return nil
}
