package repository

import (
	"github.com/jovandeginste/workout-tracker/v2/pkg/model"
	"gorm.io/gorm"
)

type User interface {
	GetAll() ([]*model.User, error)
	GetByID(userID uint64) (*model.User, error)
	GetByUsername(username string) (*model.User, error)
	GetByAPIKey(key string) (*model.User, error)
}

type userRepository struct {
	db *gorm.DB
}

func NewUser(db *gorm.DB) User {
	return &userRepository{db: db}
}

func (r *userRepository) GetAll() ([]*model.User, error) {
	var users []*model.User

	if err := r.db.Find(&users).Error; err != nil {
		return nil, err
	}

	return users, nil
}

func (r *userRepository) GetByID(userID uint64) (*model.User, error) {
	var user model.User

	if err := r.currentUserQuery().First(&user, userID).Error; err != nil {
		return nil, err
	}

	user.SetDB(r.db)

	return &user, nil
}

func (r *userRepository) GetByUsername(username string) (*model.User, error) {
	var user model.User

	if err := r.currentUserQuery().Where(&model.UserData{Username: username}).First(&user).Error; err != nil {
		return nil, err
	}

	if user.ID == user.Profile.UserID {
		user.Profile.User = &user
	}

	user.SetDB(r.db)

	return &user, nil
}

func (r *userRepository) GetByAPIKey(key string) (*model.User, error) {
	var user model.User

	if err := r.currentUserQuery().Where(&model.UserSecrets{APIKey: key}).First(&user).Error; err != nil {
		return nil, err
	}

	user.SetDB(r.db)

	return &user, nil
}

func (r *userRepository) currentUserQuery() *gorm.DB {
	return r.db.Preload("Profile").Preload("Equipment")
}
