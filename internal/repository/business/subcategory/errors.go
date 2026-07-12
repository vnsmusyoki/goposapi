package subcategory

import "errors"

var ErrSubCategoryAlreadyExists = errors.New("sub category already exists")
var ErrSubCategoryNotFound = errors.New("sub category not found")
var ErrInvalidSubCategoryInput = errors.New("invalid sub category input")
var ErrBusinessNotResolved = errors.New("business not resolved")
