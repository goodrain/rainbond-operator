package user

// Usecase represent the user's usecases
type Usecase interface {
	Login(username, password string) (string, error)
}
