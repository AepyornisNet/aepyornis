package repository

import (
	"github.com/AepyornisNet/aepyornis/pkg/model"
	"github.com/samber/do/v2"
	"gorm.io/gorm"
)

type User interface {
	GetAll() ([]*model.User, error)
	GetByID(userID uint64) (*model.User, error)
	GetByEmail(email string) (*model.User, error)
	GetByUsername(username string) (*model.User, error)
	GetByAPIKey(key string) (*model.User, error)
}

type userRepository struct {
	db *gorm.DB
}

func NewUser(injector do.Injector) (User, error) {
	return &userRepository{db: do.MustInvoke[*gorm.DB](injector)}, nil
}

func (r *userRepository) GetAll() ([]*model.User, error) {
	var users []*model.User

	if err := r.db.Preload("Profile").Find(&users).Error; err != nil {
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

	if err := r.currentUserQuery().
		Joins("JOIN profiles ON profiles.user_id = users.id").
		Where("profiles.username = ? AND (profiles.domain = '' OR profiles.domain IS NULL)", username).
		First(&user).Error; err != nil {
		return nil, err
	}

	user.Profile.User = &user

	user.SetDB(r.db)

	return &user, nil
}

func (r *userRepository) GetByEmail(email string) (*model.User, error) {
	var user model.User

	if err := r.currentUserQuery().Where(&model.UserSecrets{Email: email}).First(&user).Error; err != nil {
		return nil, err
	}

	user.Profile.User = &user

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
	return r.db.Preload("Profile")
}
