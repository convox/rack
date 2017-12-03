package sparta

type contextKey int

const (
	// ContextKeyLogger is the *logrus.Logger instance
	// attached to the request context
	ContextKeyLogger contextKey = iota
	// ContextKeyLambdaContext is the *sparta.LambdaContext
	// pointer in the request
	ContextKeyLambdaContext
)
