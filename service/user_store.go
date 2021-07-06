package service

type UserStore interface {
	Save(user *User) error
	Find(username string) (*User, error)
}


