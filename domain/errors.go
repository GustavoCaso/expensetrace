package domain

type NotFoundError struct{}

func (e *NotFoundError) Error() string {
	return "record not found"
}

// AlreadyExistsError is returned when a record would violate a uniqueness
// constraint.
type AlreadyExistsError struct{}

func (e *AlreadyExistsError) Error() string {
	return "record already exists"
}
