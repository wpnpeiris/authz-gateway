package authorization

type Request struct {
	Principal Principal
	Resource  Resource
	Action    string
}

type Principal struct {
	ID    string
	Roles []string
}

type Resource struct {
	Kind string
	ID   string
}

type Decision struct {
	Allowed bool
}
