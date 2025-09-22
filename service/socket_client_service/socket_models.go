package socket_client_service

// PrivateChatItem Private chat record item
type PrivateChatItem struct {
	From string `json:"from"` // Sender MetaId
	// FromUserInfo *UserInfo   `json:"fromUserInfo"`
	To string `json:"to"` // Receiver MetaId
	// ToUserInfo   *UserInfo   `json:"toUserInfo"`
	TxId    string `json:"txId"`
	PinId   string `json:"pinId"`
	MetaId  string `json:"metaId"`  // Message creator MetaId
	Address string `json:"address"` // Message creator address
	// UserInfo     *UserInfo   `json:"userInfo"` // User info
	NickName    string      `json:"nickName"`
	Protocol    string      `json:"protocol"`
	Content     string      `json:"content"`
	ContentType string      `json:"contentType"`
	Encryption  string      `json:"encryption"`
	Version     string      `json:"version"`  // Version
	ChatType    int64       `json:"chatType"` // 0-msg, 1-red, 2-img
	Data        interface{} `json:"data"`
	ReplyPin    string      `json:"replyPin"`
	// ReplyInfo   *ReplyInfo  `json:"replyInfo"`
	ReplyMetaId string `json:"replyMetaId"`
	Timestamp   int64  `json:"timestamp"`   // Chat record timestamp
	Params      string `json:"params"`      // General field for future parameter additions
	Chain       string `json:"chain"`       // Chain type
	BlockHeight int64  `json:"blockHeight"` // Block height
	Index       int64  `json:"index"`       //Index default -1
}

type GroupChatItem struct {
	GroupId   string `json:"groupId"`   //Room ID, unique
	ChannelId string `json:"channelId"` //Channel ID, unique
	MetanetId string `json:"metanetId"` //
	TxId      string `json:"txId"`
	PinId     string `json:"pinId"`
	MetaId    string `json:"metaId"`
	Address   string `json:"address"`
	// UserInfo    *UserInfo       `json:"userInfo"`
	NickName    string `json:"nickName"`
	Protocol    string `json:"protocol"`
	Content     string `json:"content"`
	ContentType string `json:"contentType"`
	Encryption  string `json:"encryption"`
	Version     string `json:"version"` // Version
	// ChatType    models.ChatType `json:"chatType"` //0-msg, 1-red, 2-img
	Data     interface{} `json:"data"`
	ReplyPin string      `json:"replyPin"`
	// ReplyInfo   *ReplyInfo      `json:"replyInfo"`
	ReplyMetaId string `json:"replyMetaId"`
	Timestamp   int64  `json:"timestamp"`   //Chat record timestamp
	Params      string `json:"params"`      //General field for future parameter additions
	Chain       string `json:"chain"`       //Chain type
	BlockHeight int64  `json:"blockHeight"` //Block height
	Index       int64  `json:"index"`       //Index default -1
}
