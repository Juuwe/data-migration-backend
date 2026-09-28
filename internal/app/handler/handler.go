package handler

import (
	"errors"
	"net/http"
	"strconv"

	"github.com/Juuwe/data-migration-backend/internal/app/serializer"
	"github.com/Juuwe/data-migration-backend/internal/app/service"
	"github.com/Juuwe/data-migration-backend/internal/ds"
	"github.com/gin-gonic/gin"
)

type MigrationMethodHandler struct {
	s *service.MigrationMethodService
}

func NewMigrationMethodHandler(s *service.MigrationMethodService) *MigrationMethodHandler {
	return &MigrationMethodHandler{s: s}
}

func (h *MigrationMethodHandler) GetGrid(c *gin.Context) {
	ctx := c.Request.Context()
	currentUserID := currentCreatorID()

	minTime, errMin := strconv.ParseFloat(c.Query("min_time"), 64)
	maxTime, errMax := strconv.ParseFloat(c.Query("max_time"), 64)

	var (
		methods []serializer.MigrationMethod
		err     error
	)

	if errMin != nil || errMax != nil || minTime < 0 || maxTime < minTime {
		minTime, maxTime = 0.01, 1.00
		methods, err = h.s.GetPublished(ctx, currentUserID)
	} else {
		methods, err = h.s.GetPublishedByTime(ctx, minTime, maxTime, currentUserID)
	}

	if err != nil {
		c.Status(http.StatusInternalServerError)
		return
	}

	page := serializer.GridResponse{
		Methods: methods,
		MinTime: minTime,
		MaxTime: maxTime,
	}

	c.JSON(http.StatusOK, page)
}

func (h *MigrationMethodHandler) GetFeedItem(c *gin.Context) {
	idStr := c.Param("id")
	ctx := c.Request.Context()
	currentUserID := currentCreatorID()

	var (
		method serializer.MigrationMethod
		err    error
	)

	if idStr == "" {
		var published []serializer.MigrationMethod
		published, err = h.s.GetPublished(ctx, currentUserID)
		if err != nil {
			c.Status(http.StatusInternalServerError)
			return
		}
		if len(published) == 0 {
			c.Status(http.StatusNotFound)
			return
		}
		method = published[0]
	} else {
		var id int64
		id, err = strconv.ParseInt(idStr, 10, 64)
		if err != nil || id <= 0 {
			c.Status(http.StatusBadRequest)
			return
		}

		if c.Query("next") == "true" {
			method, err = h.s.GetNextPublishedAfterID(ctx, id, currentUserID)
		} else {
			method, err = h.s.GetByID(ctx, id, currentUserID)
		}

		if err != nil {
			if errors.Is(err, ds.ErrMigrationMethodNotFound) {
				c.Status(http.StatusNotFound)
			} else {
				c.Status(http.StatusInternalServerError)
			}
			return
		}
	}

	c.JSON(http.StatusOK, serializer.FeedResponse{Method: method})
}

func (h *MigrationMethodHandler) ShowAddMethodPage(c *gin.Context) {
	draft, exists, err := h.s.GetDraft(c.Request.Context(), currentCreatorID())
	if err != nil {
		c.Status(http.StatusInternalServerError)
		return
	}
	if !exists {
		c.Status(http.StatusNotFound)
		return
	}

	page := serializer.DraftResponse{
		Draft: draft,
	}
	c.JSON(http.StatusOK, page)
}

func (h *MigrationMethodHandler) CreateDraftMethod(c *gin.Context) {
	userID := currentCreatorID()
	c.Request.Body = http.MaxBytesReader(c.Writer, c.Request.Body, service.MaxUploadBytes)
	form, err := c.MultipartForm()
	if err != nil {
		var sizeError *http.MaxBytesError
		if errors.As(err, &sizeError) {
			c.Status(http.StatusRequestEntityTooLarge)
			return
		}
		c.Status(http.StatusBadRequest)
		return
	}
	defer form.RemoveAll()

	if len(form.Value) != 1 || len(form.Value["title"]) != 1 || len(form.File) != 2 || len(form.File["image"]) != 1 || len(form.File["video"]) != 1 {
		c.Status(http.StatusBadRequest)
		return
	}
	imageHeader := form.File["image"][0]
	videoHeader := form.File["video"][0]
	if imageHeader.Size <= 0 || videoHeader.Size <= 0 {
		c.Status(http.StatusBadRequest)
		return
	}
	if imageHeader.Size > service.MaxImageBytes || videoHeader.Size > service.MaxVideoBytes {
		c.Status(http.StatusRequestEntityTooLarge)
		return
	}
	imageFile, err := imageHeader.Open()
	if err != nil {
		c.Status(http.StatusBadRequest)
		return
	}
	defer imageFile.Close()
	videoFile, err := videoHeader.Open()
	if err != nil {
		c.Status(http.StatusBadRequest)
		return
	}
	defer videoFile.Close()

	draft, err := h.s.CreateDraftMethod(c.Request.Context(), form.Value["title"][0], userID,
		service.MediaInput{Reader: imageFile, Size: imageHeader.Size},
		service.MediaInput{Reader: videoFile, Size: videoHeader.Size})
	if err != nil {
		status := http.StatusInternalServerError
		switch {
		case errors.Is(err, service.ErrDraftTitleRequired), errors.Is(err, service.ErrDraftTitleTooLong),
			errors.Is(err, service.ErrDraftAlreadyExists), errors.Is(err, service.ErrInvalidMedia):
			status = http.StatusBadRequest
		}
		c.Status(status)
		return
	}

	c.Header("Location", c.Request.URL.Path+"/draft")
	c.JSON(http.StatusCreated, serializer.DraftResponse{Draft: draft})
}

func (h *MigrationMethodHandler) PublishDraftMethod(c *gin.Context) {
	userID := currentCreatorID()
	var request struct {
		Description string  `json:"description"`
		TimeInGb    float64 `json:"time_in_gb"`
		Reliability float64 `json:"reliability"`
	}
	if err := bindJSON(c, &request); err != nil {
		c.Status(http.StatusBadRequest)
		return
	}

	method, err := h.s.PublishDraft(c.Request.Context(), userID, request.Description, request.TimeInGb, request.Reliability)
	if err != nil {
		switch {
		case errors.Is(err, service.ErrInvalidDescription), errors.Is(err, service.ErrInvalidTimeInGb),
			errors.Is(err, service.ErrInvalidReliability):
			c.Status(http.StatusBadRequest)
		case errors.Is(err, ds.ErrMigrationMethodNotFound):
			c.Status(http.StatusNotFound)
		default:
			c.Status(http.StatusInternalServerError)
		}
		return
	}

	c.JSON(http.StatusOK, serializer.FeedResponse{Method: method})
}

func (h *MigrationMethodHandler) SoftDeleteMethod(c *gin.Context) {
	id, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil || id <= 0 {
		c.Status(http.StatusBadRequest)
		return
	}

	err = h.s.DeleteMethod(c.Request.Context(), id, currentCreatorID())
	if err != nil {
		status := http.StatusInternalServerError
		if errors.Is(err, ds.ErrMigrationMethodNotFound) {
			status = http.StatusNotFound
		}
		c.Status(status)
		return
	}

	c.Status(http.StatusNoContent)
}

func (h *MigrationMethodHandler) SetLike(c *gin.Context) {
	id, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil || id <= 0 {
		c.Status(http.StatusBadRequest)
		return
	}

	var request struct {
		Like *int `json:"like"`
	}
	if err := bindJSON(c, &request); err != nil {
		c.Status(http.StatusBadRequest)
		return
	}
	if request.Like == nil {
		c.Status(http.StatusBadRequest)
		return
	}

	method, err := h.s.SetLike(c.Request.Context(), id, currentCreatorID(), *request.Like)
	if err != nil {
		status := http.StatusInternalServerError
		switch {
		case errors.Is(err, service.ErrInvalidLike):
			status = http.StatusBadRequest
		case errors.Is(err, ds.ErrMigrationMethodNotFound):
			status = http.StatusNotFound
		}
		c.Status(status)
		return
	}

	c.JSON(http.StatusOK, serializer.LikeResponse{Method: method})
}
