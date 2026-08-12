package domain

type Role string

const (
	RoleAdmin Role = "admin"
	RoleUser  Role = "user"
)

type User struct {
	Id           ID
	Name         string
	Email        string
	PasswordHash string
	Role         Role
	CreatedAt    string
}

func (u *User) Validate() error {
	if u.Name == "" {
		return ErrEmptyName
	}
	if u.Email == "" {
		return ErrEmptyEmail
	}
	if u.PasswordHash == "" {
		return ErrPasswordRequired
	}
	if u.Role != RoleAdmin && u.Role != RoleUser {
		return ErrInvalidRole
	}
	return nil
}

func (u *User) IsAdmin() bool {
	return u != nil && u.Role == RoleAdmin
}

type UserRepository interface {
	Add(u *User) error
	GetByID(id ID) (*User, error)
	GetByEmail(email string) (*User, error)
	List(page int, limit int) ([]User, error)
	Update(u *User) error
	Remove(id ID) error
	Count() (int, error)
}
