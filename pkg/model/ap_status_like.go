package model

type APStatusLike struct {
	Model

	StatusID uint64    `gorm:"index:idx_ap_status_like_status_profile,unique;not null" json:"status_id"`
	Status   *APStatus `gorm:"foreignKey:StatusID;references:ID;constraint:OnDelete:CASCADE" json:"-"`

	ProfileID *uint64  `gorm:"index:idx_ap_status_like_status_profile,unique" json:"profile_id,omitempty"`
	Profile   *Profile `gorm:"foreignKey:ProfileID;constraint:OnDelete:CASCADE" json:"-"`
}

func (APStatusLike) TableName() string {
	return "ap_status_likes"
}
