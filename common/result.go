package common

import (
	"common/biz"
	"framework/msError"
	"github.com/gin-gonic/gin"
	"net/http"
)

type Result struct {
	Code int `json:"code"`
	Msg  any `json:"msg"`
}

func ReturnSuccess(ctx *gin.Context, data any) {
	json := &Result{Code: biz.Ok, Msg: data}
	ctx.JSON(200, json)
}

// ReturnFail Fail err 最后自己封装一个
func ReturnFail(ctx *gin.Context, err *msError.Error) {
	ctx.JSON(http.StatusOK, Result{
		Code: err.Code,
		Msg:  err.Err.Error(),
	})
}
func F(err *msError.Error) Result {
	return Result{
		Code: err.Code,
	}

}
func S(data any) Result {
	return Result{
		Code: biz.Ok,
		Msg:  data,
	}
}
