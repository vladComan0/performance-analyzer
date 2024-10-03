package entity

type EnvironmentOption func(*Environment)

func WithEnvironmentTokenEndpoint(tokenEndpoint string) EnvironmentOption {
	return func(environment *Environment) {
		environment.TokenEndpoint = tokenEndpoint
	}
}

func WithEnvironmentUsername(username string) EnvironmentOption {
	return func(e *Environment) {
		e.Username = username
	}
}

func WithEnvironmentPassword(password string) EnvironmentOption {
	return func(e *Environment) {
		e.Password = password
	}
}

func WithEnvironmentDisabled(disabled bool) EnvironmentOption {
	return func(e *Environment) {
		e.Disabled = disabled
	}
}
