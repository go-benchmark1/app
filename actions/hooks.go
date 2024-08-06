package actions

import (
	"encoding/json"
	"io/ioutil"
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
	sns "github.com/robbiet480/go.sns"
	"github.com/sirupsen/logrus"

	"github.com/mailbadger/app/emails"
	"github.com/mailbadger/app/entities"
	"github.com/mailbadger/app/logger"
	"github.com/mailbadger/app/storage"
)

func HandleHook(storage storage.Storage) gin.HandlerFunc {
	return func(c *gin.Context) {
		var payload sns.Payload

		body, err := c.GetRawData()
		if err != nil {
			logger.From(c).WithError(err).Error("handle hook: cannot fetch raw data")
			c.AbortWithStatus(http.StatusBadRequest)
			return
		}

		err = json.Unmarshal(body, &payload)
		if err != nil {
			logger.From(c).WithError(err).Error("handle hook: cannot decode SNS request")
			c.AbortWithStatus(http.StatusBadRequest)
			return
		}

		err = payload.VerifyPayload()
		if err != nil {
			logger.From(c).WithError(err).Error("handle hook: unable to verify SNS payload")
			c.AbortWithStatus(http.StatusBadRequest)
			return
		}

		if payload.Type == emails.SubConfirmationType {
			response, err := http.Get(payload.SubscribeURL)
			if err != nil {
				logger.From(c).WithError(err).Error("handle hook: AWS unable to confirm SNS subscription")
				c.AbortWithStatus(http.StatusBadRequest)
				return
			}

			defer func() {
				err := response.Body.Close()
				if err != nil {
					logger.From(c).WithError(err).Error("handle hook: unable to close response body")
				}
			}()

			if response.StatusCode >= http.StatusBadRequest {
				xml, _ := ioutil.ReadAll(response.Body)
				logger.From(c).WithFields(logrus.Fields{
					"response":    string(xml),
					"status_code": response.StatusCode,
				}).Error("handle hook: AWS error while confirming the subscribe URL")
			}

			c.AbortWithStatus(http.StatusBadRequest)
			return
		}

		var msg entities.SesMessage

		err = json.Unmarshal([]byte(payload.Message), &msg)
		if err != nil {
			logger.From(c).WithError(err).Error("handle hook: cannot unmarshal SNS raw message")

			c.AbortWithStatus(http.StatusBadRequest)
			return
		}

		// fetch the campaign id from tags
		cidTag, ok := msg.Mail.Tags["campaign_id"]
		if !ok || len(cidTag) == 0 {
			logger.From(c).WithFields(logrus.Fields{
				"message_id": msg.Mail.MessageID,
				"source":     msg.Mail.Source,
				"tags":       msg.Mail.Tags,
			}).Error("handle hook: campaign id not found in mail tags")

			c.AbortWithStatus(http.StatusBadRequest)
			return
		}

		cid, err := strconv.ParseInt(cidTag[0], 10, 64)
		if err != nil {
			logger.From(c).WithError(err).Error("handle hook: unable to parse campaign id")

			c.AbortWithStatus(http.StatusBadRequest)
			return
		}

		uuid := c.Param("uuid")
		u, err := storage.GetUserByUUID(uuid)
		if err != nil {
			logger.From(c).WithField("uuid", uuid).WithError(err).Error("handle hook: unable to fetch user by uuid")

			c.AbortWithStatus(http.StatusBadRequest)
			return
		}

		switch msg.NotificationType {
		case emails.BounceType:
			if msg.Bounce == nil {
				logger.From(c).WithField("message", msg).Error("BounceType: bounce is nil")
				c.AbortWithStatus(http.StatusBadRequest)
				return
			}

			for _, recipient := range msg.Bounce.BouncedRecipients {
				err := storage.CreateBounce(&entities.Bounce{
					UserID:         u.ID,
					CampaignID:     cid,
					Recipient:      recipient.EmailAddress,
					Action:         recipient.Action,
					Status:         recipient.Status,
					DiagnosticCode: recipient.DiagnosticCode,
					Type:           msg.Bounce.BounceType,
					SubType:        msg.Bounce.BounceSubType,
					FeedbackID:     msg.Bounce.FeedbackID,
					CreatedAt:      msg.Bounce.Timestamp,
				})
				if err != nil {
					logger.From(c).WithFields(logrus.Fields{
						"message":   msg,
						"recipient": recipient,
					}).WithError(err).Error("Unable to create bounce record")
					c.AbortWithStatus(http.StatusBadRequest)
					return
				}

				if msg.Bounce.BounceType == "Permanent" {
					err = storage.DeactivateSubscriber(u.ID, recipient.EmailAddress)
					if err != nil {
						logger.From(c).WithFields(logrus.Fields{
							"message":   msg,
							"recipient": recipient,
						}).WithError(err).Error("Unable to blacklist bounced recipient")
					}
					c.AbortWithStatus(http.StatusBadRequest)
					return
				}
			}
		case emails.ComplaintType:
			if msg.Complaint == nil {
				logger.From(c).WithField("message", msg).Error("ComplaintType: complaint is nil")
				c.AbortWithStatus(http.StatusBadRequest)
				return
			}

			for _, recipient := range msg.Complaint.ComplainedRecipients {
				err := storage.CreateComplaint(&entities.Complaint{
					UserID:     u.ID,
					CampaignID: cid,
					Recipient:  recipient.EmailAddress,
					Type:       msg.Complaint.ComplaintFeedbackType,
					FeedbackID: msg.Complaint.FeedbackID,
					CreatedAt:  msg.Complaint.Timestamp,
				})
				if err != nil {
					logger.From(c).WithFields(logrus.Fields{
						"user_id":     u.ID,
						"campaign_id": cid,
						"message":     msg,
						"recipient":   recipient,
					}).WithError(err).Error("Unable to create complaint record")
					c.AbortWithStatus(http.StatusBadRequest)
					return
				}
			}
		case emails.DeliveryType:
			if msg.Delivery == nil {
				logger.From(c).WithField("message", msg).Error("DeliveryType: delivery is nil.")
				c.AbortWithStatus(http.StatusBadRequest)
				return
			}

			for _, r := range msg.Delivery.Recipients {
				err := storage.CreateDelivery(&entities.Delivery{
					UserID:               u.ID,
					CampaignID:           cid,
					Recipient:            r,
					ProcessingTimeMillis: msg.Delivery.ProcessingTimeMillis,
					ReportingMTA:         msg.Delivery.ReportingMTA,
					RemoteMtaIP:          msg.Delivery.RemoteMtaIP,
					SMTPResponse:         msg.Delivery.SMTPResponse,
					CreatedAt:            msg.Delivery.Timestamp,
				})
				if err != nil {
					logger.From(c).WithFields(logrus.Fields{
						"user_id":     u.ID,
						"campaign_id": cid,
						"message":     msg,
					}).WithError(err).Error("Unable to create delivery record.")
					c.AbortWithStatus(http.StatusBadRequest)
					return
				}
			}
		case emails.SendType:
			for _, d := range msg.Mail.Destination {
				err := storage.CreateSend(&entities.Send{
					UserID:           u.ID,
					CampaignID:       cid,
					MessageID:        msg.Mail.MessageID,
					Source:           msg.Mail.Source,
					SendingAccountID: msg.Mail.SendingAccountID,
					Destination:      d,
					CreatedAt:        msg.Mail.Timestamp,
				})
				if err != nil {
					logger.From(c).WithFields(logrus.Fields{
						"message":     msg,
						"user_id":     u.ID,
						"campaign_id": cid,
					}).WithError(err).Error("Unable to create send record.")
					c.AbortWithStatus(http.StatusBadRequest)
					return
				}
			}
		case emails.ClickType:
			if msg.Click == nil {
				logger.From(c).WithField("message", msg).Error("ClickType: click is nil.")
				c.AbortWithStatus(http.StatusBadRequest)
				return
			}

			for _, d := range msg.Mail.Destination {
				err := storage.CreateClick(&entities.Click{
					UserID:     u.ID,
					CampaignID: cid,
					Recipient:  d,
					Link:       msg.Click.Link,
					UserAgent:  msg.Click.UserAgent,
					IPAddress:  msg.Click.IPAddress,
					CreatedAt:  msg.Click.Timestamp,
				})
				if err != nil {
					logger.From(c).WithFields(logrus.Fields{
						"user_id":     u.ID,
						"campaign_id": cid,
						"message":     msg,
					}).WithError(err).Error("Unable to create click record.")

					c.AbortWithStatus(http.StatusBadRequest)
					return
				}
			}
		case emails.OpenType:
			if msg.Open == nil {
				logger.From(c).WithField("message", msg).Error("OpenType: open is nil.")
				c.AbortWithStatus(http.StatusBadRequest)
				return
			}

			for _, d := range msg.Mail.Destination {
				err := storage.CreateOpen(&entities.Open{
					UserID:     u.ID,
					CampaignID: cid,
					Recipient:  d,
					UserAgent:  msg.Open.UserAgent,
					IPAddress:  msg.Open.IPAddress,
					CreatedAt:  msg.Open.Timestamp,
				})
				if err != nil {
					logger.From(c).WithFields(logrus.Fields{
						"user_id":     u.ID,
						"campaign_id": cid,
						"message":     msg,
					}).WithError(err).Error("Unable to create open record.")

					c.AbortWithStatus(http.StatusBadRequest)
					return
				}
			}

		case emails.RenderingFailureType:
			logger.From(c).WithFields(logrus.Fields{
				"campaign_id":   cid,
				"user_id":       u.ID,
				"error":         msg.RenderingFailure.ErrorMessage,
				"template_name": msg.RenderingFailure.TemplateName,
			}).Warn("Rendering html template failure.")
		default:
			logger.From(c).WithField("sns", msg).Error("handle hook: Unknown AWS SES message.")
		}
	}
}
