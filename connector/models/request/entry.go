package request

type EntryReq struct {
	Token    string `json:"token"`
	UserInfo struct {
		Nickname string `json:"nickname"`
		Avatar   string `json:"avatar"`
		Sex      int    `json:"sex"`
	} `json:"userInfo"`
}
type UserInfo struct {
	Nickname string `json:"nickname"`
	Avatar   string `json:"avatar"`
	Sex      int    `json:"sex"`
}
