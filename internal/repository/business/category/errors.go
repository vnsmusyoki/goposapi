package category

import "errors"

var ErrCategoryAlreadyExists = errors.New("category already exists")
var ErrInvalidCategoryInput = errors.New("invalid category input")
var ErrBusinessNotResolved = errors.New("business not resolved")
var ErrInvalidCategoryImage = errors.New("invalid category image")
var ErrCategoryImageTooLarge = errors.New("category image too large")
var ErrCategoryImageTypeNotAllowed = errors.New("category image type not allowed")
