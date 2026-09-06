package api

import (
	"net/http"
	"time"

	"github.com/gin-gonic/gin"

	"skyimage/internal/installer"
)

// installerPasswordHeader 携带安装密码的请求头。
const installerPasswordHeader = "X-Install-Password"

func (s *Server) registerInstallerRoutes(r *gin.RouterGroup) {
	group := r.Group("/installer")
	// 状态与默认文案公开：前端需要据此判断是否弹出密码门禁（/api/health 本就暴露安装状态）。
	group.GET("/status", s.getInstallerStatus)
	group.GET("/defaults", s.getInstallerDefaults)
	group.POST("/verify", s.postInstallerVerify)

	// 未安装状态下，安装向导的全部写操作必须通过安装密码验证。
	protected := r.Group("/installer", s.installerPasswordRequired())
	protected.POST("/run", s.postInstallerRun)
	protected.POST("/legacy/test", s.postInstallerLegacyTest)
	protected.POST("/legacy/import", s.postInstallerLegacyImport)
}

// installerPasswordRequired 在未安装状态下强制校验 X-Install-Password；
// 已安装的站点安装向导本身已失效（/run 会拒绝），不再拦截。
func (s *Server) installerPasswordRequired() gin.HandlerFunc {
	return func(c *gin.Context) {
		status, err := s.installer.Status(c.Request.Context())
		if err == nil && status.Installed {
			c.Next()
			return
		}
		if !s.installer.VerifyInstallPassword(c.GetHeader(installerPasswordHeader)) {
			// 只对失败计数限速：封顶爆破尝试，正常用户（校验通过）不受影响
			if ok, retry := s.authLimiter.Allow("installer-gate:fail:"+c.ClientIP(), 20, time.Minute); !ok {
				c.AbortWithStatusJSON(http.StatusTooManyRequests, gin.H{"error": "请求过于频繁，请稍后再试", "retryAfterSeconds": int(retry.Seconds()) + 1})
				return
			}
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{
				"error": "installer password required",
				"code":  "INSTALLER_PASSWORD_REQUIRED",
			})
			return
		}
		c.Next()
	}
}

// postInstallerVerify 供前端密码门禁页校验密码，验证通过后再携带请求头访问向导接口。
func (s *Server) postInstallerVerify(c *gin.Context) {
	if ok, retry := s.authLimiter.Allow("installer-verify:ip:"+c.ClientIP(), 10, time.Minute); !ok {
		c.JSON(http.StatusTooManyRequests, gin.H{"error": "请求过于频繁，请稍后再试", "retryAfterSeconds": int(retry.Seconds()) + 1})
		return
	}
	var input struct {
		Password string `json:"password"`
	}
	if err := c.ShouldBindJSON(&input); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid request"})
		return
	}
	status, err := s.installer.Status(c.Request.Context())
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	if status.Installed {
		c.JSON(http.StatusOK, gin.H{"data": gin.H{"ok": true}})
		return
	}
	if !s.installer.VerifyInstallPassword(input.Password) {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "invalid installer password"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"data": gin.H{"ok": true}})
}

func (s *Server) getInstallerStatus(c *gin.Context) {
	status, err := s.installer.Status(c.Request.Context())
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"data": status})
}

func (s *Server) postInstallerRun(c *gin.Context) {
	var input installer.RunInput
	if err := c.ShouldBindJSON(&input); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	status, err := s.installer.Run(c.Request.Context(), input)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"data": status})
}

func (s *Server) getInstallerDefaults(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{
		"data": gin.H{
			"termsOfService": installer.DefaultTermsOfService,
			"privacyPolicy":  installer.DefaultPrivacyPolicy,
		},
	})
}

func (s *Server) postInstallerLegacyTest(c *gin.Context) {
	var input installer.LegacySourceInput
	if err := c.ShouldBindJSON(&input); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	result, err := s.installer.TestLegacySource(c.Request.Context(), input)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"data": result})
}

func (s *Server) postInstallerLegacyImport(c *gin.Context) {
	var input installer.LegacySourceInput
	if err := c.ShouldBindJSON(&input); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	summary, err := s.installer.RunLegacyImport(c.Request.Context(), input)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"data": summary})
}
