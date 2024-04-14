package api

import (
	"common"
	"common/biz"
	"common/config"
	"common/jwts"
	"common/logs"
	"common/rpc"
	"context"
	"framework/msError"
	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt/v5"
	"time"
	"user/pb"
)

type UserHandler struct {
}

func NewUserHandler() *UserHandler {
	return &UserHandler{}
}
func (u *UserHandler) Register(ctx *gin.Context) {
	var req pb.RegisterParams
	err1 := ctx.BindJSON(&req)
	if err1 != nil {
		logs.Error("user handler register err:%v", err1)
		common.ReturnFail(ctx, biz.RequestDataError)
		return
	}
	response, err := rpc.UserClient.Register(context.TODO(), &req)
	if err != nil {
		logs.Error("user handler register err:%v", err)
		common.ReturnFail(ctx, msError.ToError(err))
		return
	}
	uid := response.Uid
	if len(uid) == 0 {
		logs.Error("uid is empty")
		common.ReturnFail(ctx, biz.Fail)
	}
	logs.Info("uid%s", uid)
	//get token by uid token 由A.B.C部分组成，A是头（定义加墨算法）B部分（存储数据） C部分签名
	claim := &jwts.CustomClaims{
		Uid: uid,
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(7 * 24 * time.Hour)),
		},
	}
	token, err := jwts.GenToken(claim, config.Conf.Jwt.Secret)
	if err != nil {
		logs.Error("uid :%v register get token error : %v", uid, err)
		common.ReturnFail(ctx, biz.Fail)
		return
	}
	result := map[string]any{
		"token": token,
		"serverInfo": map[string]any{
			"host": config.Conf.Services["connector"].ClientHost,
			"port": config.Conf.Services["connector"].ClientPort,
		},
	}
	common.ReturnSuccess(ctx, result)
}
