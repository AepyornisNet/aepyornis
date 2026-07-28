package notification

import (
	"encoding/json"
	"fmt"

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

func (*followRequest) GetSubject() string {
	return "New follow request"
}

func (r *followRequest) GetMessage() string {
	content := "%s wants to follow you"
	return fmt.Sprintf(content, r.FollowerName)
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
