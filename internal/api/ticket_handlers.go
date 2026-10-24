package api

import (
	"errors"
	"fmt"
	"io"
	"math"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"

	"skyimage/internal/captcha"
	"skyimage/internal/data"
	"skyimage/internal/middleware"
	"skyimage/internal/tickets"
	"skyimage/internal/users"
)

const ticketWriteCooldown = 10 * time.Second

type ticketCaptchaPayload struct {
	CaptchaToken    string            `json:"captchaToken"`
	CaptchaData     map[string]string `json:"captchaData"`
	CaptchaProvider string            `json:"captchaProvider"`
}

func (s *Server) enforceTicketWriteRateLimit(c *gin.Context, userID uint, action string) bool {
	key := fmt.Sprintf("ticket:%s:user:%d", action, userID)
	ok, retry := s.authLimiter.AllowInterval(key, ticketWriteCooldown)
	if ok {
		return true
	}
	secs := int(retry.Seconds() + 0.999)
	if secs < 1 {
		secs = 1
	}
	c.Header("Retry-After", strconv.Itoa(secs))
	c.JSON(http.StatusTooManyRequests, gin.H{
		"error":      "操作过于频繁，请稍后再试",
		"retryAfter": secs,
	})
	return false
}

func (s *Server) requireTicketCaptcha(c *gin.Context, token, providerRaw string, data map[string]string) bool {
	cfg, err := s.captcha.GetConfig(c.Request.Context(), "ticket")
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "系统错误"})
		return false
	}
	if !cfg.Enabled {
		return true
	}
	provider := captcha.Provider(providerRaw)
	if provider == "" {
		provider = cfg.Provider
	}
	clientIP := getClientIP(c, s.isCDNEnabled(c.Request.Context()))
	if err := s.verifyCaptcha(c.Request.Context(), provider, token, clientIP, data); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "人机验证失败，请重试"})
		return false
	}
	return true
}

func (s *Server) registerTicketRoutes(r *gin.RouterGroup) {
	account := r.Group("/account")
	account.Use(s.authMiddleware())
	account.GET("/tickets", s.handleListMyTickets)
	account.GET("/tickets/attachment-strategy", s.handleTicketAttachmentStrategy)
	account.GET("/tickets/:id", s.handleGetMyTicket)

	accountWrite := account.Group("")
	accountWrite.Use(middleware.RequireCSRF())
	accountWrite.POST("/tickets", s.handleCreateTicket)
	accountWrite.POST("/tickets/:id/replies", s.handleReplyMyTicket)
	accountWrite.POST("/tickets/:id/close", s.handleCloseMyTicket)
	accountWrite.POST("/tickets/:id/attachments", s.handleUploadMyTicketAttachment)
}

func (s *Server) registerAdminTicketRoutes(adminGroup *gin.RouterGroup) {
	adminGroup.GET("/tickets", s.handleAdminListTickets)
	adminGroup.GET("/tickets/:id", s.handleAdminGetTicket)
	adminGroup.PATCH("/tickets/:id", s.handleAdminUpdateTicket)
	adminGroup.POST("/tickets/:id/replies", s.handleAdminReplyTicket)
	adminGroup.POST("/tickets/:id/attachments", s.handleAdminUploadTicketAttachment)
}

type ticketUserDTO struct {
	ID    uint   `json:"id,string"`
	Name  string `json:"name"`
	Email string `json:"email"`
}

type ticketAttachmentDTO struct {
	ID        uint      `json:"id"`
	TicketID  uint      `json:"ticketId"`
	MessageID *uint     `json:"messageId,omitempty"`
	Name      string    `json:"name"`
	Size      int64     `json:"size"`
	MimeType  string    `json:"mimeType"`
	URL       string    `json:"url"`
	CreatedAt time.Time `json:"createdAt"`
}

type ticketMessageDTO struct {
	ID          uint                   `json:"id"`
	TicketID    uint                   `json:"ticketId"`
	UserID      uint                   `json:"userId,string"`
	Body        string                 `json:"body"`
	IsStaff     bool                   `json:"isStaff"`
	CreatedAt   time.Time              `json:"createdAt"`
	User        *ticketUserDTO         `json:"user,omitempty"`
	Attachments []ticketAttachmentDTO  `json:"attachments,omitempty"`
}

type ticketDTO struct {
	ID          uint       `json:"id"`
	TicketNo    string     `json:"ticketNo"`
	UserID      uint       `json:"userId,string"`
	Subject     string     `json:"subject"`
	Status      string     `json:"status"`
	Priority    string     `json:"priority"`
	LastReplyAt *time.Time `json:"lastReplyAt,omitempty"`
	ClosedAt    *time.Time `json:"closedAt,omitempty"`
	CreatedAt   time.Time  `json:"createdAt"`
	UpdatedAt   time.Time  `json:"updatedAt"`
	User        *ticketUserDTO `json:"user,omitempty"`
}

type ticketDetailDTO struct {
	Ticket      ticketDTO             `json:"ticket"`
	Messages    []ticketMessageDTO    `json:"messages"`
	Attachments []ticketAttachmentDTO `json:"attachments"`
}

func buildTicketUserDTO(u data.User) *ticketUserDTO {
	if u.ID == 0 {
		return nil
	}
	return &ticketUserDTO{ID: u.ID, Name: u.Name, Email: u.Email}
}

func buildTicketMessageUserDTO(m data.TicketMessage) *ticketUserDTO {
	if m.User.ID == 0 {
		return nil
	}
	name := m.User.Name
	if m.IsStaff {
		name = users.TicketStaffDisplayName(m.User)
	}
	return &ticketUserDTO{ID: m.User.ID, Name: name, Email: m.User.Email}
}

func (s *Server) buildTicketDTO(t data.Ticket) ticketDTO {
	return ticketDTO{
		ID:          t.ID,
		TicketNo:    t.TicketNo,
		UserID:      t.UserID,
		Subject:     t.Subject,
		Status:      t.Status,
		Priority:    t.Priority,
		LastReplyAt: t.LastReplyAt,
		ClosedAt:    t.ClosedAt,
		CreatedAt:   t.CreatedAt,
		UpdatedAt:   t.UpdatedAt,
		User:        buildTicketUserDTO(t.User),
	}
}

func (s *Server) buildTicketMessageDTO(m data.TicketMessage, attachments []ticketAttachmentDTO) ticketMessageDTO {
	return ticketMessageDTO{
		ID:          m.ID,
		TicketID:    m.TicketID,
		UserID:      m.UserID,
		Body:        m.Body,
		IsStaff:     m.IsStaff,
		CreatedAt:   m.CreatedAt,
		User:        buildTicketMessageUserDTO(m),
		Attachments: attachments,
	}
}

func (s *Server) buildTicketAttachmentDTO(c *gin.Context, a data.TicketAttachment) ticketAttachmentDTO {
	url := ""
	if s.tickets != nil {
		url = s.tickets.AttachmentPublicURL(c.Request.Context(), a)
	}
	return ticketAttachmentDTO{
		ID:        a.ID,
		TicketID:  a.TicketID,
		MessageID: a.MessageID,
		Name:      a.Name,
		Size:      a.Size,
		MimeType:  a.MimeType,
		URL:       url,
		CreatedAt: a.CreatedAt,
	}
}

func (s *Server) buildTicketDetailDTO(c *gin.Context, detail tickets.TicketDetail, viewer *data.User) ticketDetailDTO {
	visible := tickets.FilterAttachmentsForViewer(detail.Attachments, viewer)
	byMessage := map[uint][]ticketAttachmentDTO{}
	orphan := make([]ticketAttachmentDTO, 0)
	for _, a := range visible {
		dto := s.buildTicketAttachmentDTO(c, a)
		if a.MessageID != nil && *a.MessageID > 0 {
			byMessage[*a.MessageID] = append(byMessage[*a.MessageID], dto)
		} else {
			orphan = append(orphan, dto)
		}
	}
	messages := make([]ticketMessageDTO, 0, len(detail.Messages))
	for _, m := range detail.Messages {
		messages = append(messages, s.buildTicketMessageDTO(m, byMessage[m.ID]))
	}
	return ticketDetailDTO{
		Ticket:      s.buildTicketDTO(detail.Ticket),
		Messages:    messages,
		Attachments: orphan,
	}
}

func (s *Server) handleListMyTickets(c *gin.Context) {
	user, ok := middleware.CurrentUser(c)
	if !ok {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "missing user"})
		return
	}
	limit, offset := parsePagination(c, 20, 100)
	items, err := s.tickets.List(c.Request.Context(), tickets.ListFilter{
		UserID:   user.ID,
		Status:   c.Query("status"),
		Priority: c.Query("priority"),
		Limit:    limit,
		Offset:   offset,
	})
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	out := make([]ticketDTO, 0, len(items))
	for _, item := range items {
		out = append(out, s.buildTicketDTO(item))
	}
	c.JSON(http.StatusOK, gin.H{"data": out})
}

func (s *Server) handleTicketAttachmentStrategy(c *gin.Context) {
	id, err := s.tickets.AttachmentStrategyID(c.Request.Context())
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"data": gin.H{
		"strategyId": id,
		"enabled":    id > 0,
	}})
}

func (s *Server) handleGetMyTicket(c *gin.Context) {
	user, ok := middleware.CurrentUser(c)
	if !ok {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "missing user"})
		return
	}
	id, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil || id == 0 || id > math.MaxUint {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid id"})
		return
	}
	detail, err := s.tickets.Get(c.Request.Context(), uint(id))
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			c.JSON(http.StatusNotFound, gin.H{"error": "ticket not found"})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	if detail.Ticket.UserID != user.ID {
		c.JSON(http.StatusNotFound, gin.H{"error": "ticket not found"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"data": s.buildTicketDetailDTO(c, detail, &user)})
}

func (s *Server) handleCreateTicket(c *gin.Context) {
	user, ok := middleware.CurrentUser(c)
	if !ok {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "missing user"})
		return
	}
	if !s.enforceTicketWriteRateLimit(c, user.ID, "create") {
		return
	}
	var payload struct {
		Subject  string `json:"subject"`
		Body     string `json:"body"`
		Priority string `json:"priority"`
		ticketCaptchaPayload
	}
	if err := c.ShouldBindJSON(&payload); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	if !s.requireTicketCaptcha(c, payload.CaptchaToken, payload.CaptchaProvider, payload.CaptchaData) {
		return
	}
	detail, err := s.tickets.Create(c.Request.Context(), tickets.CreateInput{
		UserID:   user.ID,
		Subject:  payload.Subject,
		Body:     payload.Body,
		Priority: payload.Priority,
	})
	if err != nil {
		writeTicketError(c, err)
		return
	}
	c.JSON(http.StatusCreated, gin.H{"data": s.buildTicketDetailDTO(c, detail, &user)})
}

func (s *Server) handleReplyMyTicket(c *gin.Context) {
	user, ok := middleware.CurrentUser(c)
	if !ok {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "missing user"})
		return
	}
	if !s.enforceTicketWriteRateLimit(c, user.ID, "reply") {
		return
	}
	id, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil || id == 0 || id > math.MaxUint {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid id"})
		return
	}
	var payload struct {
		Body string `json:"body"`
		ticketCaptchaPayload
	}
	if err := c.ShouldBindJSON(&payload); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	if !s.requireTicketCaptcha(c, payload.CaptchaToken, payload.CaptchaProvider, payload.CaptchaData) {
		return
	}
	msg, err := s.tickets.Reply(c.Request.Context(), tickets.ReplyInput{
		TicketID: uint(id),
		UserID:   user.ID,
		Body:     payload.Body,
		IsStaff:  false,
	})
	if err != nil {
		writeTicketError(c, err)
		return
	}
	c.JSON(http.StatusCreated, gin.H{"data": s.buildTicketMessageDTO(msg, nil)})
}

func (s *Server) handleCloseMyTicket(c *gin.Context) {
	user, ok := middleware.CurrentUser(c)
	if !ok {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "missing user"})
		return
	}
	id, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil || id == 0 || id > math.MaxUint {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid id"})
		return
	}
	ticket, err := s.tickets.CloseByOwner(c.Request.Context(), uint(id), user.ID)
	if err != nil {
		writeTicketError(c, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"data": s.buildTicketDTO(ticket)})
}

func (s *Server) handleUploadMyTicketAttachment(c *gin.Context) {
	user, ok := middleware.CurrentUser(c)
	if !ok {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "missing user"})
		return
	}
	id, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil || id == 0 || id > math.MaxUint {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid id"})
		return
	}
	detail, err := s.tickets.Get(c.Request.Context(), uint(id))
	if err != nil {
		writeTicketError(c, err)
		return
	}
	if detail.Ticket.UserID != user.ID {
		c.JSON(http.StatusNotFound, gin.H{"error": "ticket not found"})
		return
	}
	// Soft limit for multi-file uploads after create/reply (5 files / 10s).
	if ok, retry := s.authLimiter.Allow(fmt.Sprintf("ticket:attach:user:%d", user.ID), 5, ticketWriteCooldown); !ok {
		secs := int(retry.Seconds() + 0.999)
		if secs < 1 {
			secs = 1
		}
		c.Header("Retry-After", strconv.Itoa(secs))
		c.JSON(http.StatusTooManyRequests, gin.H{"error": "操作过于频繁，请稍后再试", "retryAfter": secs})
		return
	}
	att, err := s.readAndStoreTicketAttachment(c, uint(id), user.ID)
	if err != nil {
		writeTicketError(c, err)
		return
	}
	c.JSON(http.StatusCreated, gin.H{"data": s.buildTicketAttachmentDTO(c, att)})
}

func (s *Server) handleAdminListTickets(c *gin.Context) {
	limit, offset := parsePagination(c, 20, 100)
	items, err := s.tickets.List(c.Request.Context(), tickets.ListFilter{
		Status:   c.Query("status"),
		Priority: c.Query("priority"),
		Limit:    limit,
		Offset:   offset,
	})
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	out := make([]ticketDTO, 0, len(items))
	for _, item := range items {
		out = append(out, s.buildTicketDTO(item))
	}
	c.JSON(http.StatusOK, gin.H{"data": out})
}

func (s *Server) handleAdminGetTicket(c *gin.Context) {
	user, ok := middleware.CurrentUser(c)
	if !ok {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "missing user"})
		return
	}
	id, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil || id == 0 || id > math.MaxUint {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid id"})
		return
	}
	detail, err := s.tickets.Get(c.Request.Context(), uint(id))
	if err != nil {
		writeTicketError(c, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"data": s.buildTicketDetailDTO(c, detail, &user)})
}

func (s *Server) handleAdminUpdateTicket(c *gin.Context) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil || id == 0 || id > math.MaxUint {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid id"})
		return
	}
	var payload struct {
		Status   *string `json:"status"`
		Priority *string `json:"priority"`
	}
	if err := c.ShouldBindJSON(&payload); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	ticket, err := s.tickets.Update(c.Request.Context(), uint(id), tickets.UpdateInput{
		Status:   payload.Status,
		Priority: payload.Priority,
	})
	if err != nil {
		writeTicketError(c, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"data": s.buildTicketDTO(ticket)})
}

func (s *Server) handleAdminReplyTicket(c *gin.Context) {
	user, ok := middleware.CurrentUser(c)
	if !ok {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "missing user"})
		return
	}
	if !s.enforceTicketWriteRateLimit(c, user.ID, "admin-reply") {
		return
	}
	id, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil || id == 0 || id > math.MaxUint {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid id"})
		return
	}
	var payload struct {
		Body string `json:"body"`
	}
	if err := c.ShouldBindJSON(&payload); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	msg, err := s.tickets.Reply(c.Request.Context(), tickets.ReplyInput{
		TicketID: uint(id),
		UserID:   user.ID,
		Body:     payload.Body,
		IsStaff:  true,
	})
	if err != nil {
		writeTicketError(c, err)
		return
	}
	c.JSON(http.StatusCreated, gin.H{"data": s.buildTicketMessageDTO(msg, nil)})
}

func (s *Server) handleAdminUploadTicketAttachment(c *gin.Context) {
	user, ok := middleware.CurrentUser(c)
	if !ok {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "missing user"})
		return
	}
	if ok, retry := s.authLimiter.Allow(fmt.Sprintf("ticket:admin-attach:user:%d", user.ID), 5, ticketWriteCooldown); !ok {
		secs := int(retry.Seconds() + 0.999)
		if secs < 1 {
			secs = 1
		}
		c.Header("Retry-After", strconv.Itoa(secs))
		c.JSON(http.StatusTooManyRequests, gin.H{"error": "操作过于频繁，请稍后再试", "retryAfter": secs})
		return
	}
	id, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil || id == 0 || id > math.MaxUint {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid id"})
		return
	}
	if _, err := s.tickets.Get(c.Request.Context(), uint(id)); err != nil {
		writeTicketError(c, err)
		return
	}
	att, err := s.readAndStoreTicketAttachment(c, uint(id), user.ID)
	if err != nil {
		writeTicketError(c, err)
		return
	}
	c.JSON(http.StatusCreated, gin.H{"data": s.buildTicketAttachmentDTO(c, att)})
}

func (s *Server) readAndStoreTicketAttachment(c *gin.Context, ticketID, userID uint) (data.TicketAttachment, error) {
	fileHeader, err := c.FormFile("file")
	if err != nil {
		return data.TicketAttachment{}, err
	}
	if fileHeader.Size > tickets.MaxAttachmentBytes {
		return data.TicketAttachment{}, tickets.ErrAttachmentTooLarge
	}
	f, err := fileHeader.Open()
	if err != nil {
		return data.TicketAttachment{}, err
	}
	defer f.Close()
	payload, err := io.ReadAll(io.LimitReader(f, tickets.MaxAttachmentBytes+1))
	if err != nil {
		return data.TicketAttachment{}, err
	}
	if len(payload) > tickets.MaxAttachmentBytes {
		return data.TicketAttachment{}, tickets.ErrAttachmentTooLarge
	}
	mimeType := fileHeader.Header.Get("Content-Type")
	if mimeType == "" {
		mimeType = http.DetectContentType(payload)
	}
	var messageID *uint
	if raw := strings.TrimSpace(c.PostForm("messageId")); raw != "" {
		if parsed, err := strconv.ParseUint(raw, 10, 64); err == nil && parsed > 0 && parsed <= math.MaxUint {
			id := uint(parsed)
			messageID = &id
		}
	}
	return s.tickets.AddAttachment(c.Request.Context(), tickets.AttachmentInput{
		TicketID:  ticketID,
		MessageID: messageID,
		UserID:    userID,
		Filename:  fileHeader.Filename,
		MimeType:  mimeType,
		Data:      payload,
	})
}

func writeTicketError(c *gin.Context, err error) {
	switch {
	case errors.Is(err, gorm.ErrRecordNotFound):
		c.JSON(http.StatusNotFound, gin.H{"error": "ticket not found"})
	case errors.Is(err, tickets.ErrForbidden):
		c.JSON(http.StatusForbidden, gin.H{"error": err.Error()})
	case errors.Is(err, tickets.ErrTicketClosed),
		errors.Is(err, tickets.ErrEmptyBody),
		errors.Is(err, tickets.ErrEmptySubject),
		errors.Is(err, tickets.ErrSubjectTooLong),
		errors.Is(err, tickets.ErrBodyTooLong),
		errors.Is(err, tickets.ErrInvalidPriority),
		errors.Is(err, tickets.ErrInvalidStatus),
		errors.Is(err, tickets.ErrAttachmentStrategyNotSet),
		errors.Is(err, tickets.ErrAttachmentTooLarge),
		errors.Is(err, tickets.ErrTooManyAttachments),
		errors.Is(err, tickets.ErrUnsupportedAttachmentType):
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
	default:
		if strings.Contains(err.Error(), "not found") {
			c.JSON(http.StatusNotFound, gin.H{"error": err.Error()})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
	}
}
