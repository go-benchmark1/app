package actions

import (
	"errors"
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
	"github.com/sirupsen/logrus"
	"gorm.io/gorm"

	"github.com/mailbadger/app/entities"
	"github.com/mailbadger/app/entities/params"
	"github.com/mailbadger/app/logger"
	"github.com/mailbadger/app/routes/middleware"
	"github.com/mailbadger/app/services/templates"
	"github.com/mailbadger/app/storage"
	"github.com/mailbadger/app/validator"
)

func GetTemplate(svc templates.Service) gin.HandlerFunc {
	return func(c *gin.Context) {
		id, err := strconv.ParseInt(c.Param("id"), 10, 64)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{
				"message": "Id must be an integer",
			})
			return
		}

		u := middleware.GetUser(c)

		template, err := svc.GetTemplate(c, id, u.ID)
		if err != nil {
			switch {
			case errors.Is(err, gorm.ErrRecordNotFound):
				c.JSON(http.StatusNotFound, gin.H{
					"message": "Template not found.",
				})
			case errors.Is(err, templates.ErrHTMLPartNotFound):
				c.JSON(http.StatusNotFound, gin.H{
					"message": "HTML part not found.",
				})
			case errors.Is(err, templates.ErrHTMLPartInvalidState):
				c.JSON(http.StatusNotFound, gin.H{
					"message": "The state of the HTML part is invalid.",
				})
			default:
				logger.From(c).WithFields(logrus.Fields{
					"user_id":     u.ID,
					"template_id": id,
				}).WithError(err).Error("get template: unable to get template")

				c.JSON(http.StatusUnprocessableEntity, gin.H{
					"message": "Unable to get template",
				})
			}
			return
		}

		c.JSON(http.StatusOK, template)
	}
}

func GetTemplates(svc templates.Service) gin.HandlerFunc {
	return func(c *gin.Context) {
		u := middleware.GetUser(c)

		val, ok := c.Get("cursor")
		if !ok {
			logger.From(c).Error("get templates: unable to fetch pagination cursor from context")
			c.AbortWithStatusJSON(http.StatusInternalServerError, gin.H{
				"message": "Unable to fetch templates. Please try again.",
			})
			return
		}

		p, ok := val.(*storage.PaginationCursor)
		if !ok {
			logger.From(c).Error("get templates: unable to cast pagination cursor from context value")
			c.AbortWithStatusJSON(http.StatusInternalServerError, gin.H{
				"message": "Unable to fetch templates. Please try again.",
			})
			return
		}

		scopeMap := c.QueryMap("scopes")

		err := svc.GetTemplates(c, u.ID, p, scopeMap)
		if err != nil {
			logger.From(c).WithFields(logrus.Fields{
				"user_id":   u.ID,
				"scope_map": scopeMap,
			}).WithError(err).Error("get templates: unable to list templates")

			c.AbortWithStatusJSON(http.StatusInternalServerError, gin.H{
				"message": "Unable to fetch templates. Please try again.",
			})
			return
		}

		c.JSON(http.StatusOK, p)
	}
}

func PostTemplate(
	svc templates.Service,
	storage storage.Storage,
) gin.HandlerFunc {
	return func(c *gin.Context) {
		u := middleware.GetUser(c)

		body := &params.PostTemplate{}
		if err := c.ShouldBindJSON(body); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{
				"message": "Invalid parameters, please try again",
			})
			return
		}

		if err := validator.Validate(body); err != nil {
			c.JSON(http.StatusBadRequest, err)
			return
		}

		template := &entities.Template{
			BaseTemplate: entities.BaseTemplate{
				UserID:      u.ID,
				Name:        body.Name,
				SubjectPart: body.SubjectPart,
			},
			HTMLPart: body.HTMLPart,
			TextPart: body.TextPart,
		}

		_, err := storage.GetTemplateByName(template.Name, u.ID)
		if err == nil {
			c.JSON(http.StatusUnprocessableEntity, gin.H{
				"message": "Template with that name already exists",
			})
			return
		}

		err = svc.AddTemplate(c, template)
		if err != nil {
			switch {
			case errors.Is(err, templates.ErrParseHTMLPart):
				c.JSON(http.StatusBadRequest, gin.H{
					"message": "Unable to create template, failed to parse html_part",
				})
			case errors.Is(err, templates.ErrParseTextPart):
				c.JSON(http.StatusBadRequest, gin.H{
					"message": "Unable to create template, failed to parse text_part",
				})
			case errors.Is(err, templates.ErrParseSubjectPart):
				c.JSON(http.StatusBadRequest, gin.H{
					"message": "Unable to create template, failed to parse subject_part",
				})
			default:
				logger.From(c).WithFields(logrus.Fields{
					"template": template,
					"user_id":  u.ID,
				}).WithError(err).Error("Unable to create template")
				c.JSON(http.StatusBadRequest, gin.H{
					"message": "Unable to create template, please try again.",
				})
			}
			return
		}

		c.JSON(http.StatusCreated, template)
	}
}

func PutTemplate(
	svc templates.Service,
	storage storage.Storage,
) gin.HandlerFunc {
	return func(c *gin.Context) {
		u := middleware.GetUser(c)

		id, err := strconv.ParseInt(c.Param("id"), 10, 64)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{
				"message": "Id must be an integer",
			})
			return
		}

		template, err := storage.GetTemplate(id, u.ID)
		if err != nil {
			c.JSON(http.StatusNotFound, gin.H{
				"message": "Template not found",
			})
			return
		}

		body := &params.PutTemplate{}
		if err := c.ShouldBindJSON(body); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{
				"message": "Invalid parameters, please try again",
			})
			return
		}

		if err := validator.Validate(body); err != nil {
			c.JSON(http.StatusBadRequest, err)
			return
		}

		template2, err := storage.GetTemplateByName(body.Name, u.ID)
		if err == nil && template.ID != template2.ID {
			c.JSON(http.StatusUnprocessableEntity, gin.H{
				"message": "Template with that name already exists",
			})
			return
		}
		template.Name = body.Name
		template.HTMLPart = body.HTMLPart
		template.TextPart = body.TextPart
		template.SubjectPart = body.SubjectPart

		err = svc.UpdateTemplate(c, template)
		if err != nil {
			switch {
			case errors.Is(err, templates.ErrParseHTMLPart):
				c.JSON(http.StatusBadRequest, gin.H{
					"message": "Unable to update template, failed to parse html_part",
				})
			case errors.Is(err, templates.ErrParseTextPart):
				c.JSON(http.StatusBadRequest, gin.H{
					"message": "Unable to update template, failed to parse text_part",
				})
			case errors.Is(err, templates.ErrParseSubjectPart):
				c.JSON(http.StatusBadRequest, gin.H{
					"message": "Unable to update template, failed to parse subject_part",
				})
			default:
				logger.From(c).WithFields(logrus.Fields{
					"template": template,
					"user_id":  u.ID,
				}).WithError(err).Error("unable to update template")
				c.JSON(http.StatusBadRequest, gin.H{
					"message": "Unable to update template, please try again.",
				})
			}
			return
		}

		c.JSON(http.StatusOK, template)
	}
}

func DeleteTemplate(svc templates.Service) gin.HandlerFunc {
	return func(c *gin.Context) {
		id, err := strconv.ParseInt(c.Param("id"), 10, 64)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{
				"message": "Id must be an integer",
			})
			return
		}

		u := middleware.GetUser(c)

		err = svc.DeleteTemplate(c, id, u.ID)
		if err != nil {
			logger.From(c).WithFields(logrus.Fields{
				"user_id":     u.ID,
				"template_id": id,
			}).WithError(err).Error("unable to delete template")

			c.JSON(http.StatusUnprocessableEntity, gin.H{
				"message": "Unable to delete template.",
			})
			return
		}

		c.Status(http.StatusNoContent)
	}
}
