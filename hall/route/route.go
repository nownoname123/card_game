package route

import (
	"common/biz"
	"core/repo"
	"framework/node"
	"framework/remote"
	"hall/handler"
	"hall/models/response"
)

func Register(r *repo.Manager) node.LogicHandler {
	handlers := make(node.LogicHandler)
	userHandler := handler.NewUserHandler(r)
	handlers["userHandler.updateUserAddress"] = userHandler.UpdateUserAddress
	handlers["unionHandler.getUserUnionList"] = returnNil
	// unionHandler.getGameRecord
	handlers["unionHandler.getGameRecord"] = returnNil
	return handlers
}

func returnNil(session *remote.Session, msg []byte) any {
	res := response.UpdateUserAddressRes{}
	res.Code = biz.Ok
	return res
}
