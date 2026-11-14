package api

import (
	"math"
	"strconv"
	"strings"

	"github.com/gin-gonic/gin"
)

// parseUintValue 将十进制字符串转换为平台原生 uint，空串/非数字/0 均视为非法。
// strconv.ParseUint 以 64 位解析后再收窄到 uint：64 位平台上无损；32 位平台上
// uint 为 32 位，超过 math.MaxUint 的值会被拒绝，避免静默截断回绕。
// 路由参数与表单/设置值的整型解析统一走本函数。
func parseUintValue(raw string) (uint, bool) {
	value, err := strconv.ParseUint(strings.TrimSpace(raw), 10, 64)
	if err != nil || value == 0 || value > math.MaxUint {
		return 0, false
	}
	return uint(value), true
}

// parseIDParam 解析路由中的 :id 参数，非法或为 0 时返回 false。
func parseIDParam(c *gin.Context) (uint, bool) {
	return parseUintValue(c.Param("id"))
}
