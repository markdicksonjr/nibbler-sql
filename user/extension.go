package sql

import (
	"github.com/jinzhu/gorm"
	"github.com/markdicksonjr/nibbler"
	sql "github.com/markdicksonjr/nibbler-sql"
)

type Extension struct {
	nibbler.NoOpExtension
	SqlExtension *sql.Extension
}

func (s *Extension) Init(app *nibbler.Application) error {
	// sql extension AutoMigrates models
	return nil
}

func (s *Extension) GetName() string {
	return "sql-user"
}

func (s *Extension) GetUserById(id string) (*nibbler.User, error) {
	userValue := nibbler.User{}
	err := s.SqlExtension.Db.First(&userValue, id).Error

	if gorm.IsRecordNotFoundError(err) {
		return nil, nil
	}

	return &userValue, err
}

func (s *Extension) GetUserByEmail(email string) (*nibbler.User, error) {
	userValue := nibbler.User{}
	err := s.SqlExtension.Db.First(&userValue, "email = ?", email).Error

	if gorm.IsRecordNotFoundError(err) {
		return nil, nil
	}

	return &userValue, err
}

func (s *Extension) GetUserByUsername(username string) (*nibbler.User, error) {
	userValue := nibbler.User{}
	err := s.SqlExtension.Db.First(&userValue, "username = ?", username).Error

	if gorm.IsRecordNotFoundError(err) {
		return nil, nil
	}

	return &userValue, err
}

func (s *Extension) GetUserByPasswordResetToken(token string) (*nibbler.User, error) {
	s.SqlExtension.Db.Error = nil

	userValue := nibbler.User{}
	err := s.SqlExtension.Db.First(&userValue, "password_reset_token = ?", token).Error

	if gorm.IsRecordNotFoundError(err) {
		return nil, nil
	}

	return &userValue, err
}

func (s *Extension) GetUserByEmailValidationToken(token string) (*nibbler.User, error) {
	s.SqlExtension.Db.Error = nil

	userValue := nibbler.User{}
	err := s.SqlExtension.Db.First(&userValue, "email_validation_token = ?", token).Error

	if gorm.IsRecordNotFoundError(err) {
		return nil, nil
	}

	return &userValue, err
}

func (s *Extension) Create(user *nibbler.User) (*nibbler.User, error) {
	err := s.SqlExtension.Db.Create(user).Error
	return user, err
}

func (s *Extension) Update(userValue *nibbler.User) error {
	// Update will not save nil values, but Save will
	return s.SqlExtension.Db.Model(userValue).Updates(*userValue).Error
}

func (s *Extension) Save(userValue *nibbler.User) error {
	// Update will not save nil values, but Save will
	return s.SqlExtension.Db.Model(userValue).Save(*userValue).Error
}

func (s *Extension) UpdatePassword(userValue *nibbler.User) error {
	if err := s.SqlExtension.Db.Model(userValue).Updates(nibbler.User{
		ID:       userValue.ID,
		Password: userValue.Password,
	}).Error; err != nil {
		return err
	}

	if err := sql.NullifyField(s.SqlExtension.Db, "password_reset_token").Error; err != nil {
		return err
	}

	if err := sql.NullifyField(s.SqlExtension.Db, "password_reset_token_expiration").Error; err != nil {
		return err
	}

	return nil
}
