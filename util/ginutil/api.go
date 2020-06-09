package ginutil

import (
	"errors"
	"github.com/gin-gonic/gin"
	"net/http"
)

const (
	//2xx: 成功, 4xx: 客户端错误, 5xx: 服务端错误
	Status_Success           = "20000001" //通用成功
	Status_Server_Err        = "50000001" //服务端异常
	Status_Illegal_Param     = "40000010" //客户端非法参数
	Status_Entity_NotFound   = "40000011" //实体不存在，判为客户端错误
	Status_Api_NotFound      = "40000012" //接口不存在
	Status_Method_NotSupport = "40000013" //方法不支持

	msg_server_err = "服务异常"
)

func ApiResult(c *gin.Context, result interface{}) {
	ApiStatusResult(c, Status_Success, result)
}

func ApiStatusResult(c *gin.Context, status string, result interface{}) {
	ApiCodeResult(c, http.StatusOK, status, result)
}

var emptyResult = gin.H{}

func ApiCodeResult(c *gin.Context, code int, status string, result interface{}) {
	if status == "" {
		panic(errors.New("api result's status cannot be empty"))
	}
	first := status[:1]
	if first != "2" {
		panic(errors.New("api result's status must start with '2'"))
	}
	if len(status) != 8 {
		panic(errors.New("api result's status must be length 8"))
	}
	if result == nil {
		result = emptyResult
	}
	c.JSON(code, gin.H{
		"status": status,
		"result": result,
	})
}

func ApiServerError(c *gin.Context) {
	ApiError(c, Status_Server_Err, msg_server_err)
}

func ApiIllegalParam(c *gin.Context, message string) {
	ApiError(c, Status_Illegal_Param, message)
}

func ApiEntityNotfound(c *gin.Context, message string) {
	ApiError(c, Status_Entity_NotFound, message)
}

func ApiError(c *gin.Context, status string, message string) {
	ApiCodeError(c, http.StatusOK, status, message)
}

func ApiCodeError(c *gin.Context, code int, status string, message string) {
	if status == "" {
		panic(errors.New("api error's status cannot be empty"))
	}
	first := status[:1]
	if first != "4" && first != "5" {
		panic(errors.New("api error's status must start with '4' or '5'"))
	}
	if len(status) != 8 {
		panic(errors.New("api error's status must be length 8"))
	}
	if message == "" {
		message = msg_server_err
	}
	c.JSON(code, gin.H{
		"status":  status,
		"message": message,
	})
}
