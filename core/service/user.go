package service

import (
	"common/biz"
	"common/logs"
	"common/utils"
	"connector/models/request"
	"context"
	"core/dao"
	"core/models/entity"
	"core/repo"
	"fmt"
	games "framework/game"
	"framework/msError"
	hall "hall/models/request"
	"time"
)

type UserService struct {
	userDao *dao.UserDao
}

func (s *UserService) FindAndSaveUserByUid(ctx context.Context, uid string, info request.UserInfo) (*entity.User, error) {
	//查询mongo 有就返回，没有就新增
	user, err := s.userDao.FindUserByUid(ctx, uid)
	if err != nil {
		return nil, err
	}
	if user == nil {
		user = &entity.User{}
		user.Uid = uid
		user.Gold = int64(games.Conf.GameConfig["startGold"]["value"].(float64))
		user.Avatar = utils.Default(info.Avatar, "Common/head_icon_default")
		user.Nickname = utils.Default(info.Nickname, fmt.Sprintf("%s%s", "赌怪", uid))
		user.Sex = info.Sex //0 男 1 女
		user.CreateTime = time.Now().UnixMilli()
		user.LastLoginTime = time.Now().UnixMilli()
		err = s.userDao.Insert(context.TODO(), user)
		if err != nil {
			logs.Error("[UserService] FindUserByUid insert user err:%v", err)
			return nil, err
		}
	}
	return user, nil
}
func NewUserService(r *repo.Manager) *UserService {
	return &UserService{
		userDao: dao.NewUserDao(r),
	}
}

func (s *UserService) UpdateUserAddressByUid(uid string, req hall.UpdateUserAddressReq) error {
	user := &entity.User{
		Uid:      uid,
		Address:  req.Address,
		Location: req.Location,
	}
	err := s.userDao.UpdateUserAddressByUid(context.TODO(), user)
	if err != nil {
		logs.Error("userDao.UpdateUserAddressByUid err:%v", err)
		return err
	}
	return nil
}
func (s *UserService) FindUserByUid(ctx context.Context, uid string) (*entity.User, *msError.Error) {
	user, err := s.userDao.FindUserByUid(ctx, uid)
	if err != nil {
		logs.Error("FindUserByUid error : %v", err)
		return nil, biz.SqlError
	}
	return user, nil
}
