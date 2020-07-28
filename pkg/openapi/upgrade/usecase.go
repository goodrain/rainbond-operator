package upgrade

// Usecase represent the upgrade's usecases
type Usecase interface {
	Versions() ([]string, error)
}
