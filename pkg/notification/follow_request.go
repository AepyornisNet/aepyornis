package notification

import (
	"encoding/json"

	"github.com/invopop/ctxi18n/i18n"
	"gorm.io/datatypes"
)

type followRequest struct {
	FollowerName string
}

func NewFollowRequest(name string) *followRequest {
	return &followRequest{
		FollowerName: name,
	}
}

func (*followRequest) GetType() string {
	return "follow_request"
}

func (*followRequest) GetSubject(t *i18n.Locale) string {
	return t.T("notifications.follow_request_subject")
}

func (r *followRequest) GetMessage(t *i18n.Locale) string {
	return t.T("notifications.follow_request_message", r.FollowerName)
}

func (*followRequest) GetMeta() *datatypes.JSON {
	meta := map[string]string{
		"url": "/profile/settings/followers",
	}

	data, err := json.Marshal(meta)
	if err != nil {
		return nil
	}

	jsonData := datatypes.JSON(data)
	return &jsonData
}

func (*followRequest) AllowDB() bool {
	return true
}

func (*followRequest) AllowEmail() bool {
	return true
}

func (*followRequest) AllowWebpush() bool {
	return true
}
