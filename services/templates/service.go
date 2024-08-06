package templates

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"io/ioutil"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/awserr"
	"github.com/aws/aws-sdk-go/service/s3"
	"github.com/aws/aws-sdk-go/service/s3/s3iface"
	"github.com/cbroglie/mustache"

	"github.com/mailbadger/app/config"
	"github.com/mailbadger/app/entities"
	"github.com/mailbadger/app/storage"
)

var (
	ErrHTMLPartNotFound     = errors.New("HTML part not found")
	ErrHTMLPartInvalidState = errors.New("HTML part is in invalid state")

	ErrParseHTMLPart    = errors.New("failed to parse HTMLPart")
	ErrParseTextPart    = errors.New("failed to parse TextPart")
	ErrParseSubjectPart = errors.New("failed to parse SubjectPart")
)

type Service interface {
	AddTemplate(c context.Context, input *entities.Template) error
	UpdateTemplate(c context.Context, input *entities.Template) error
	GetTemplates(c context.Context, userID int64, p *storage.PaginationCursor, scopeMap map[string]string) error
	DeleteTemplate(c context.Context, templateID, userID int64) error
	GetTemplate(c context.Context, templateID int64, userID int64) (*entities.Template, error)
	ParseTemplate(c context.Context, templateID int64, userID int64) (*entities.CampaignTemplateData, error)
}

// service implements the Service interface
type service struct {
	db              storage.Storage
	s3              s3iface.S3API
	templatesBucket string
}

func From(db storage.Storage, s3 s3iface.S3API, conf config.Config) Service {
	return New(db, s3, conf.Storage.S3.TemplatesBucket)
}

func New(db storage.Storage, s3 s3iface.S3API, bucket string) Service {
	return &service{
		db:              db,
		s3:              s3,
		templatesBucket: bucket,
	}
}

func (s service) AddTemplate(c context.Context, template *entities.Template) error {
	// parse string to validate template params
	_, err := mustache.ParseString(template.HTMLPart)
	if err != nil {
		return ErrParseHTMLPart
	}
	// parse string to validate template params
	_, err = mustache.ParseString(template.TextPart)
	if err != nil {
		return ErrParseTextPart
	}
	// parse string to validate template params
	_, err = mustache.ParseString(template.SubjectPart)
	if err != nil {
		return ErrParseSubjectPart
	}

	err = s.db.CreateTemplate(template)
	if err != nil {
		return fmt.Errorf("create template: %w", err)
	}

	s3Input := &s3.PutObjectInput{
		Bucket: aws.String(s.templatesBucket),
		Key:    aws.String(templateKey(template.UserID, template.ID)),
		Body:   bytes.NewReader([]byte(template.HTMLPart)),
	}

	_, err = s.s3.PutObject(s3Input)
	if err != nil {
		return fmt.Errorf("upload template: put s3 object: %w", err)
	}

	return nil
}

func (s service) UpdateTemplate(c context.Context, template *entities.Template) error {
	// parse string to validate template params
	_, err := mustache.ParseString(template.HTMLPart)
	if err != nil {
		return ErrParseHTMLPart
	}
	// parse string to validate template params
	_, err = mustache.ParseString(template.TextPart)
	if err != nil {
		return ErrParseTextPart
	}
	// parse string to validate template params
	_, err = mustache.ParseString(template.SubjectPart)
	if err != nil {
		return ErrParseSubjectPart
	}

	s3Input := &s3.PutObjectInput{
		Bucket: aws.String(s.templatesBucket),
		Key:    aws.String(templateKey(template.UserID, template.ID)),
		Body:   bytes.NewReader([]byte(template.HTMLPart)),
	}

	_, err = s.s3.PutObject(s3Input)
	if err != nil {
		return fmt.Errorf("upload template: put s3 object: %w", err)
	}

	err = s.db.UpdateTemplate(template)
	if err != nil {
		return fmt.Errorf("update template: %w", err)
	}

	return nil
}

// GetTemplates populates a pagination object with a collection of
// templates by the specified user id.
func (s service) GetTemplates(c context.Context, userID int64, p *storage.PaginationCursor, scopeMap map[string]string) error {
	return s.db.GetTemplates(userID, p, scopeMap)
}

// DeleteTemplate deletes the given template
func (s *service) DeleteTemplate(c context.Context, templateID, userID int64) error {
	_, err := s.s3.DeleteObject(&s3.DeleteObjectInput{
		Bucket: aws.String(s.templatesBucket),
		Key:    aws.String(templateKey(userID, templateID)),
	})
	if err != nil {
		return fmt.Errorf("delete object: %w", err)
	}

	err = s.db.DeleteTemplate(templateID, userID)
	if err != nil {
		return fmt.Errorf("delete template: %w", err)
	}

	return nil
}

// GetTemplate returns the template with given template id and user id
func (s service) GetTemplate(c context.Context, templateID int64, userID int64) (template *entities.Template, err error) {
	template, err = s.db.GetTemplate(templateID, userID)
	if err != nil {
		return nil, fmt.Errorf("get template: %w", err)
	}

	resp, err := s.s3.GetObject(&s3.GetObjectInput{
		Bucket: aws.String(s.templatesBucket),
		Key:    aws.String(templateKey(template.UserID, template.ID)),
	})
	if err != nil {
		if aerr, ok := err.(awserr.Error); ok {
			switch aerr.Code() {
			case s3.ErrCodeNoSuchKey:
				return nil, ErrHTMLPartNotFound
			case s3.ErrCodeInvalidObjectState:
				return nil, ErrHTMLPartInvalidState
			default:
				return nil, fmt.Errorf("get object: %w", aerr)
			}
		}
		return nil, fmt.Errorf("get object: %w", err)
	}

	defer func() {
		if cerr := resp.Body.Close(); cerr != nil {
			err = cerr
		}
	}()

	htmlBytes, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("read: %w", err)
	}

	template.HTMLPart = string(htmlBytes)

	return
}

func (s *service) ParseTemplate(c context.Context, templateID int64, userID int64) (*entities.CampaignTemplateData, error) {
	template, err := s.GetTemplate(c, templateID, userID)
	if err != nil {
		return nil, fmt.Errorf("campaign service: get template: %w", err)
	}

	html, err := mustache.ParseString(template.HTMLPart)
	if err != nil {
		return nil, fmt.Errorf("campaign service: parse html part: %w", err)
	}
	text, err := mustache.ParseString(template.TextPart)
	if err != nil {
		return nil, fmt.Errorf("campaign service: parse text part: %w", err)
	}
	sub, err := mustache.ParseString(template.SubjectPart)
	if err != nil {
		return nil, fmt.Errorf("campaign service: parse subject part: %w", err)
	}
	return &entities.CampaignTemplateData{
		Template:    template,
		HTMLPart:    html,
		SubjectPart: sub,
		TextPart:    text,
	}, nil
}

// templateKey generates template key
func templateKey(userID, templateID int64) string {
	return fmt.Sprintf("templates/%d/%d", userID, templateID)
}
