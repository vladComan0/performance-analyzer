package custom_errors

import "errors"

var ErrNoRecord = errors.New("model: no matching record found")
var ErrInvalidInput = errors.New("model: invalid input")
var ErrEnvironmentDisabled = errors.New("model: environment is disabled")
