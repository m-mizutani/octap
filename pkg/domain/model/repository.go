package model

type Repository struct {
	Owner string
	Name  string
}

func (r Repository) FullName() string {
	return r.Owner + "/" + r.Name
}
