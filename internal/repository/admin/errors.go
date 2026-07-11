package admin

import "errors"

var ErrPackageAlreadyExists = errors.New("package already exists")
var ErrPackageNotFound = errors.New("package not found")
var ErrBillingIntervalNotFound = errors.New("billing interval not found")
var ErrBusinessAlreadyExists = errors.New("business already exists")
var ErrBusinessManagerAlreadyLinked = errors.New("business manager already linked")
var ErrManagerNotFound = errors.New("manager not found")
var ErrManagerAlreadyExists = errors.New("manager already exists")
var ErrInvalidManagerInput = errors.New("invalid manager input")
var ErrInvalidBusinessInput = errors.New("invalid business input")
var ErrModuleAlreadyExists = errors.New("module already exists")
var ErrSubmoduleAlreadyExists = errors.New("submodule already exists")
var ErrModuleNotFound = errors.New("module not found")
var ErrSubmoduleNotFound = errors.New("submodule not found")
var ErrRoleNotFound = errors.New("role not found")
