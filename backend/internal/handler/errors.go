package handler

import "errors"

// errNoFieldCode means a binding rule failed that has no entry in
// validationTagCodes: a bug in a request type, answered as internal.
var errNoFieldCode = errors.New("binding rule has no field code")
