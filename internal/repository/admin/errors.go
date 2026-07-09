package admin

import "errors"

var ErrPackageAlreadyExists = errors.New("package already exists")
var ErrBusinessAlreadyExists = errors.New("business already exists")
var ErrBusinessManagerAlreadyLinked = errors.New("business manager already linked")
var ErrManagerNotFound = errors.New("manager not found")
var ErrManagerAlreadyExists = errors.New("manager already exists")
var ErrInvalidManagerInput = errors.New("invalid manager input")
var ErrInvalidBusinessInput = errors.New("invalid business input")
