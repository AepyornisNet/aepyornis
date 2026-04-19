package model

type APStatusLike struct {
	Model

	StatusID uint64    `gorm:"index:idx_ap_status_like_status_user,unique;index:idx_ap_status_like_status_actor,unique;not null" json:"status_id"`
	Status   *APStatus `gorm:"foreignKey:StatusID;references:ID;constraint:OnDelete:CASCADE" json:"-"`

	UserID *uint64 `gorm:"index:idx_ap_status_like_status_user,unique" json:"user_id,omitempty"`
	User   *User   `gorm:"constraint:OnDelete:CASCADE" json:"-"`

	ActorIRI *string `gorm:"type:text;index:idx_ap_status_like_status_actor,unique" json:"actor_iri,omitempty"`
}

func (APStatusLike) TableName() string {
	return "ap_status_likes"
}
