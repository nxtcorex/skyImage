package api

import (
	"errors"
	"math"
	"net/http"
	"net/url"
	"strconv"
	"strings"

	"github.com/gin-gonic/gin"

	"skyimage/internal/middleware"
	"skyimage/internal/payment"
	"skyimage/internal/shop"
)

func (s *Server) registerShopRoutes(r *gin.RouterGroup) {
	// Public catalog
	r.GET("/shop/products", middleware.OptionalAuth(s.users, s.session), s.handleShopListProducts)
	r.GET("/shop/providers", middleware.OptionalAuth(s.users, s.session), s.handleShopListProviders)

	// Auth required
	auth := r.Group("")
	auth.Use(s.authMiddleware(), middleware.RequireCSRF())
	auth.POST("/shop/orders", s.handleShopCreateOrder)
	auth.GET("/shop/orders", s.handleShopListOrders)
	auth.GET("/shop/orders/:orderNo", s.handleShopGetOrder)
	auth.GET("/shop/membership", s.handleShopMembership)

	// Payment callbacks — no auth/CSRF
	r.POST("/pay/notify/:provider", s.handlePayNotify)
	r.GET("/pay/notify/:provider", s.handlePayNotify)
	r.GET("/pay/return/:provider", s.handlePayReturn)
}

func (s *Server) handleShopListProducts(c *gin.Context) {
	if s.shop == nil {
		c.JSON(http.StatusServiceUnavailable, gin.H{"error": "shop unavailable"})
		return
	}
	items, err := s.shop.ListProducts(c.Request.Context(), true)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"data": items})
}

func (s *Server) handleShopListProviders(c *gin.Context) {
	if s.shop == nil || s.admin == nil {
		c.JSON(http.StatusOK, gin.H{"data": []string{}})
		return
	}
	settings, err := s.admin.GetSettings(c.Request.Context())
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"data": payment.EnabledProviders(payment.Settings(settings))})
}

func (s *Server) handleShopCreateOrder(c *gin.Context) {
	user, ok := middleware.CurrentUser(c)
	if !ok {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "login required"})
		return
	}
	if s.shop == nil {
		c.JSON(http.StatusServiceUnavailable, gin.H{"error": "shop unavailable"})
		return
	}
	var payload struct {
		ProductID uint   `json:"productId" binding:"required"`
		Provider  string `json:"provider" binding:"required"`
		EpayType  string `json:"epayType"`
		ReturnURL string `json:"returnUrl"`
	}
	if err := c.ShouldBindJSON(&payload); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	base := strings.TrimRight(s.cfg.PublicBaseURL, "/")
	result, err := s.shop.CreateOrder(c.Request.Context(), user, shop.CreateOrderInput{
		ProductID:  payload.ProductID,
		Provider:   payload.Provider,
		EpayType:   payload.EpayType,
		ReturnURL:  payload.ReturnURL,
		ClientIP:   c.ClientIP(),
		PublicBase: base,
	})
	if err != nil {
		status := http.StatusBadRequest
		switch {
		case errors.Is(err, shop.ErrPaymentDisabled), errors.Is(err, shop.ErrProviderDisabled):
			status = http.StatusServiceUnavailable
		case errors.Is(err, shop.ErrProductNotFound):
			status = http.StatusNotFound
		}
		c.JSON(status, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"data": gin.H{
		"order":     result.Order,
		"payUrl":    result.PayURL,
		"qrContent": result.QR,
		"extra":     result.Extra,
	}})
}

func (s *Server) handleShopListOrders(c *gin.Context) {
	user, ok := middleware.CurrentUser(c)
	if !ok {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "login required"})
		return
	}
	limit, _ := strconv.Atoi(c.DefaultQuery("limit", "50"))
	offset, _ := strconv.Atoi(c.DefaultQuery("offset", "0"))
	items, err := s.shop.ListUserOrders(c.Request.Context(), user.ID, limit, offset)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"data": items})
}

func (s *Server) handleShopGetOrder(c *gin.Context) {
	user, ok := middleware.CurrentUser(c)
	if !ok {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "login required"})
		return
	}
	order, err := s.shop.GetUserOrder(c.Request.Context(), user.ID, c.Param("orderNo"))
	if err != nil {
		if errors.Is(err, shop.ErrOrderNotFound) {
			c.JSON(http.StatusNotFound, gin.H{"error": err.Error()})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"data": order})
}

func (s *Server) handleShopMembership(c *gin.Context) {
	user, ok := middleware.CurrentUser(c)
	if !ok {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "login required"})
		return
	}
	info, err := s.shop.GetMembership(c.Request.Context(), user)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"data": info})
}

func (s *Server) handlePayNotify(c *gin.Context) {
	if s.shop == nil || s.admin == nil {
		c.String(http.StatusServiceUnavailable, "unavailable")
		return
	}
	providerName := payment.NormalizeProvider(c.Param("provider"))
	prov, err := payment.Get(providerName)
	if err != nil {
		c.String(http.StatusBadRequest, "bad provider")
		return
	}
	settingsMap, err := s.admin.GetSettings(c.Request.Context())
	if err != nil {
		c.String(http.StatusInternalServerError, "error")
		return
	}
	settings := payment.Settings(settingsMap)

	// Peek order amount from form/query when available (epay/alipay/wechat).
	orderNo := firstQueryOrForm(c, "out_trade_no", "order_no")
	var expected int64
	if orderNo != "" {
		if order, err := s.shop.GetOrderByNo(c.Request.Context(), orderNo); err == nil {
			expected = order.PriceCents
		}
	}

	result, err := prov.HandleNotify(c.Request, settings, expected)
	if err != nil {
		c.String(http.StatusBadRequest, "fail")
		return
	}
	// FulfillFromNotify re-loads the order and enforces amount == snapshot.
	if result.Paid {
		if err := s.shop.FulfillFromNotify(c.Request.Context(), providerName, result); err != nil {
			if errors.Is(err, shop.ErrAmountMismatch) {
				c.String(http.StatusBadRequest, "fail")
				return
			}
			if !errors.Is(err, shop.ErrOrderNotPending) {
				c.String(http.StatusInternalServerError, "fail")
				return
			}
		}
	}
	if result.RespondOK != "" {
		c.Data(http.StatusOK, "text/plain; charset=utf-8", []byte(result.RespondOK))
		return
	}
	c.String(http.StatusOK, "success")
}

func (s *Server) handlePayReturn(c *gin.Context) {
	// Browser return: always land on same-origin orders page (no open redirect).
	orderNo := firstQueryOrForm(c, "out_trade_no", "order_no")
	base := strings.TrimRight(s.cfg.PublicBaseURL, "/")
	target := base + "/dashboard/orders"
	if orderNo != "" {
		target += "?order_no=" + url.QueryEscape(orderNo)
	}
	c.Redirect(http.StatusFound, target)
}

func firstQueryOrForm(c *gin.Context, keys ...string) string {
	for _, k := range keys {
		if v := strings.TrimSpace(c.Query(k)); v != "" {
			return v
		}
		if v := strings.TrimSpace(c.PostForm(k)); v != "" {
			return v
		}
	}
	return ""
}

// --- Admin shop ---

func (s *Server) registerAdminShopRoutes(adminGroup *gin.RouterGroup) {
	adminGroup.GET("/shop/products", s.handleAdminListShopProducts)
	adminGroup.POST("/shop/products", s.handleAdminCreateShopProduct)
	adminGroup.PUT("/shop/products/:id", s.handleAdminUpdateShopProduct)
	adminGroup.DELETE("/shop/products/:id", s.handleAdminDeleteShopProduct)
	adminGroup.GET("/shop/orders", s.handleAdminListShopOrders)
	adminGroup.GET("/system/payment", s.handleAdminPaymentSettings)
	adminGroup.PUT("/system/payment", s.handleAdminUpdatePaymentSettings)
}

func (s *Server) handleAdminListShopProducts(c *gin.Context) {
	items, err := s.shop.ListProducts(c.Request.Context(), false)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"data": items})
}

func (s *Server) handleAdminCreateShopProduct(c *gin.Context) {
	var input shop.ProductInput
	if err := c.ShouldBindJSON(&input); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	item, err := s.shop.CreateProduct(c.Request.Context(), input)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"data": item})
}

func (s *Server) handleAdminUpdateShopProduct(c *gin.Context) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil || id > math.MaxUint {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid id"})
		return
	}
	var input shop.ProductInput
	if err := c.ShouldBindJSON(&input); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	item, err := s.shop.UpdateProduct(c.Request.Context(), uint(id), input)
	if err != nil {
		status := http.StatusBadRequest
		if errors.Is(err, shop.ErrProductNotFound) {
			status = http.StatusNotFound
		}
		c.JSON(status, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"data": item})
}

func (s *Server) handleAdminDeleteShopProduct(c *gin.Context) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil || id > math.MaxUint {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid id"})
		return
	}
	if err := s.shop.DeleteProduct(c.Request.Context(), uint(id)); err != nil {
		status := http.StatusBadRequest
		if errors.Is(err, shop.ErrProductNotFound) {
			status = http.StatusNotFound
		}
		c.JSON(status, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"data": true})
}

func (s *Server) handleAdminListShopOrders(c *gin.Context) {
	limit, _ := strconv.Atoi(c.DefaultQuery("limit", "50"))
	offset, _ := strconv.Atoi(c.DefaultQuery("offset", "0"))
	status := c.Query("status")
	items, err := s.shop.ListAllOrders(c.Request.Context(), status, limit, offset)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"data": items})
}

type paymentSettingsPayload struct {
	Enabled bool                    `json:"enabled"`
	Epay    paymentProviderEpay     `json:"epay"`
	Alipay  paymentProviderAlipay   `json:"alipay"`
	Wechat  paymentProviderWechat   `json:"wechat"`
	Stripe  paymentProviderStripe   `json:"stripe"`
}

type paymentProviderEpay struct {
	Enabled     bool   `json:"enabled"`
	APIURL      string `json:"apiUrl"`
	PID         string `json:"pid"`
	Key         string `json:"key"`
	DefaultType string `json:"defaultType"`
}

type paymentProviderAlipay struct {
	Enabled         bool   `json:"enabled"`
	AppID           string `json:"appId"`
	PrivateKey      string `json:"privateKey"`
	AlipayPublicKey string `json:"alipayPublicKey"`
	Gateway         string `json:"gateway"`
}

type paymentProviderWechat struct {
	Enabled bool   `json:"enabled"`
	AppID   string `json:"appId"`
	MchID   string `json:"mchId"`
	APIKey  string `json:"apiKey"`
}

type paymentProviderStripe struct {
	Enabled       bool   `json:"enabled"`
	SecretKey     string `json:"secretKey"`
	WebhookSecret string `json:"webhookSecret"`
}

func (s *Server) handleAdminPaymentSettings(c *gin.Context) {
	if !requireSuperAdmin(c) {
		return
	}
	settings, err := s.admin.GetSettings(c.Request.Context())
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	payload := paymentSettingsPayload{
		Enabled: settings["pay.enabled"] == "true",
		Epay: paymentProviderEpay{
			Enabled:     settings["pay.epay.enabled"] == "true",
			APIURL:      settings["pay.epay.api_url"],
			PID:         settings["pay.epay.pid"],
			Key:         redactSecret(settings["pay.epay.key"]),
			DefaultType: settings["pay.epay.default_type"],
		},
		Alipay: paymentProviderAlipay{
			Enabled:         settings["pay.alipay.enabled"] == "true",
			AppID:           settings["pay.alipay.app_id"],
			PrivateKey:      redactSecret(settings["pay.alipay.private_key"]),
			AlipayPublicKey: settings["pay.alipay.alipay_public_key"],
			Gateway:         settings["pay.alipay.gateway"],
		},
		Wechat: paymentProviderWechat{
			Enabled: settings["pay.wechat.enabled"] == "true",
			AppID:   settings["pay.wechat.app_id"],
			MchID:   settings["pay.wechat.mch_id"],
			APIKey:  redactSecret(settings["pay.wechat.api_key"]),
		},
		Stripe: paymentProviderStripe{
			Enabled:       settings["pay.stripe.enabled"] == "true",
			SecretKey:     redactSecret(settings["pay.stripe.secret_key"]),
			WebhookSecret: redactSecret(settings["pay.stripe.webhook_secret"]),
		},
	}
	c.JSON(http.StatusOK, gin.H{"data": payload})
}

func (s *Server) handleAdminUpdatePaymentSettings(c *gin.Context) {
	if !requireSuperAdmin(c) {
		return
	}
	var payload paymentSettingsPayload
	if err := c.ShouldBindJSON(&payload); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	settings, _ := s.admin.GetSettings(c.Request.Context())
	keep := func(incoming, key string) string {
		v := strings.TrimSpace(incoming)
		if v == "" || v == "***" {
			return settings[key]
		}
		return v
	}
	updates := map[string]string{
		"pay.enabled":                  strconv.FormatBool(payload.Enabled),
		"pay.epay.enabled":             strconv.FormatBool(payload.Epay.Enabled),
		"pay.epay.api_url":             strings.TrimSpace(payload.Epay.APIURL),
		"pay.epay.pid":                 strings.TrimSpace(payload.Epay.PID),
		"pay.epay.key":                 keep(payload.Epay.Key, "pay.epay.key"),
		"pay.epay.default_type":        strings.TrimSpace(payload.Epay.DefaultType),
		"pay.alipay.enabled":           strconv.FormatBool(payload.Alipay.Enabled),
		"pay.alipay.app_id":            strings.TrimSpace(payload.Alipay.AppID),
		"pay.alipay.private_key":       keep(payload.Alipay.PrivateKey, "pay.alipay.private_key"),
		"pay.alipay.alipay_public_key": strings.TrimSpace(payload.Alipay.AlipayPublicKey),
		"pay.alipay.gateway":           strings.TrimSpace(payload.Alipay.Gateway),
		"pay.wechat.enabled":           strconv.FormatBool(payload.Wechat.Enabled),
		"pay.wechat.app_id":            strings.TrimSpace(payload.Wechat.AppID),
		"pay.wechat.mch_id":            strings.TrimSpace(payload.Wechat.MchID),
		"pay.wechat.api_key":           keep(payload.Wechat.APIKey, "pay.wechat.api_key"),
		"pay.stripe.enabled":           strconv.FormatBool(payload.Stripe.Enabled),
		"pay.stripe.secret_key":        keep(payload.Stripe.SecretKey, "pay.stripe.secret_key"),
		"pay.stripe.webhook_secret":    keep(payload.Stripe.WebhookSecret, "pay.stripe.webhook_secret"),
	}
	if err := s.admin.UpdateSettings(c.Request.Context(), updates); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"data": true})
}

