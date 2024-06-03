package custom_errors

import "errors"

var ErrNoRecord = errors.New("data: no matching record found")
var ErrInvalidInput = errors.New("data: invalid input")
var ErrEnvironmentDisabled = errors.New("data: environment is disabled")
