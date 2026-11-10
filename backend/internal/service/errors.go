package service

import "errors"

// errNotRational reports a float64's decimal form that big.Rat can't parse.
// It can't happen for a finite float64; it exists so the impossible case is
// still an error, wrapped as internal, rather than a panic.
var errNotRational = errors.New("not a rational number")
