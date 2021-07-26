package models

import (
	"encoding/hex"
	uuid "github.com/satori/go.uuid"
	"github.com/tuxcanfly/loggy/auth/database"
	"golang.org/x/crypto/bcrypt"
	"gorm.io/gorm"
	"time"
)

// User defines the user in db
type User struct {
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
	DeletedAt *time.Time `json:"deleted_at"`
	ID		 string `gorm:"type:uuid;primary_key;"`
	Name     string `json:"name"`
	Email    string `json:"email" gorm:"unique"`
	Password string `json:"password"`
}
func (user *User) BeforeCreate(tx *gorm.DB) (err error) {
	u := uuid.NewV4()
	uString := hex.EncodeToString(u.Bytes())
	user.ID = uString
	return
}
// CreateUserRecord creates a user record in the database
func (user *User) CreateUserRecord() error {
	result := database.GlobalDB.Create(&user)
	if result.Error != nil {
		return result.Error
	}

	return nil
}

// HashPassword encrypts user password
func (user *User) HashPassword(password string) error {
	bytes, err := bcrypt.GenerateFromPassword([]byte(password), 14)
	if err != nil {
		return err
	}

	user.Password = string(bytes)

	return nil
}

// CheckPassword checks user password
func (user *User) CheckPassword(providedPassword string) error {
	err := bcrypt.CompareHashAndPassword([]byte(user.Password), []byte(providedPassword))
	if err != nil {
		return err
	}

	return nil
}
