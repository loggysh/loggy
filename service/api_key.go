package service

import (
	"github.com/tomogoma/go-api-guard"
	"github.com/tomogoma/go-typed-errors"
)

type APIKeyMock struct {
	Val []byte
}

func (k APIKeyMock) Value() []byte {
	return k.Val
}

type DBMock struct {
	errors.NotFoundErrCheck

	ExpInsAPIKErr        error
	ExpAPIKBUsrIDVal     api.Key
	ExpAPIKsBUsrIDValErr error
	RecInsAPIKUsrID      string
}

func (db *DBMock) APIKeyByUserIDVal(userID string, key []byte) (api.Key, error) {
	if db.ExpAPIKsBUsrIDValErr != nil {
		return nil, db.ExpAPIKsBUsrIDValErr
	}
	if db.ExpAPIKBUsrIDVal == nil {
		return nil, errors.NewNotFound("not found")
	}
	return db.ExpAPIKBUsrIDVal, db.ExpAPIKsBUsrIDValErr
}

func (db *DBMock) InsertAPIKey(userID string, key []byte) (api.Key, error) {
	if db.ExpInsAPIKErr != nil {
		return nil, db.ExpInsAPIKErr
	}
	db.RecInsAPIKUsrID = userID
	return APIKeyMock{Val: key}, db.ExpInsAPIKErr
}

type KeyGenMock struct {
	ExpSRBsErr error
	ExpSRBs    []byte
}

func (kg *KeyGenMock) SecureRandomBytes(length int) ([]byte, error) {
	if kg.ExpSRBsErr != nil {
		return nil, kg.ExpSRBsErr
	}
	return kg.ExpSRBs, kg.ExpSRBsErr
}


func GenerateKey(userid string) (string, error){
	db := &DBMock{}


	g, _ := api.NewGuard(
		db,
	)
	// Generate API key
	APIKey, _ := g.NewAPIKey(userid)

	x := string(APIKey.Value())
	return x, nil
}

func ValidateKey(key string) (string, error) {

	db := &DBMock{}

	g, _ := api.NewGuard(
		db,
	)
	// Validate API Key
	keyString := []byte(key)
	userID, _ := g.APIKeyValid(keyString)

	return userID, nil

}
