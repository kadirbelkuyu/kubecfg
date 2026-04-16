package group

// Store persists context groups.
type Store interface {
	List() ([]Group, error)
	Get(name string) (Group, error)
	Save(g Group) error
	Delete(name string) error
}
