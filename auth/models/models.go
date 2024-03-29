package models

import (
	"encoding/hex"
	"time"

	uuid "github.com/satori/go.uuid"
	"golang.org/x/crypto/bcrypt"
	"gorm.io/gorm"
)

// User defines the user in db
type User struct {
	CreatedAt time.Time  `json:"created_at"`
	UpdatedAt time.Time  `json:"updated_at"`
	DeletedAt *time.Time `json:"deleted_at"`
	ID        string     `gorm:"type:uuid;primary_key;"`
	Name      string     `json:"name"`
	Email     string     `json:"email" gorm:"unique"`
	Password  string     `json:"password"`
	APIKey    string     `json:"api_key"`
}

func (user *User) BeforeCreate(tx *gorm.DB) (err error) {
	uString := hex.EncodeToString(uuid.NewV4().Bytes())
	user.ID = uString
	apiKey := hex.EncodeToString(uuid.NewV4().Bytes())
	user.APIKey = apiKey
	return
}

// CreateUserRecord creates a user record in the database
func (user *User) CreateUserRecord(db *gorm.DB) error {
	result := db.Create(&user)
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
