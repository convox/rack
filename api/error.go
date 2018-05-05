package api

type ApiErrorer interface {
	ApiError(error) error
}
