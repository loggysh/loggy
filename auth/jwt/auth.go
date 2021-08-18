package jwt

import (
	"errors"
	"time"

	jwt "github.com/golang-jwt/jwt"
)

// Wrapper wraps the signing key and the issuer
type Wrapper struct {
	SecretKey       string
	Issuer          string
	ExpirationHours int64
}

// Claim adds email as a claim to the token
type Claim struct {
	Email string
	jwt.StandardClaims
}

// GenerateToken generates a jwt token
func (j *Wrapper) GenerateToken(email string) (signedToken string, err error) {
	claims := &Claim{
		Email: email,
		StandardClaims: jwt.StandardClaims{
			ExpiresAt: time.Now().Local().Add(time.Hour * time.Duration(j.ExpirationHours)).Unix(),
			Issuer:    j.Issuer,
		},
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)

	signedToken, err = token.SignedString([]byte(j.SecretKey))
	if err != nil {
		return
	}

	return
}

//ValidateToken validates the jwt token
func (j *Wrapper) ValidateToken(signedToken string) (claims *Claim, err error) {
	token, err := jwt.ParseWithClaims(
		signedToken,
		&Claim{},
		func(token *jwt.Token) (interface{}, error) {
			return []byte(j.SecretKey), nil
		},
	)

	if err != nil {
		return
	}

	claims, ok := token.Claims.(*Claim)
	if !ok {
		err = errors.New("Couldn't parse claims ")
		return
	}

	if claims.ExpiresAt < time.Now().Local().Unix() {
		err = errors.New("JWT is expired")
		return
	}

	return

}
