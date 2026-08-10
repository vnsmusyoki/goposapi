package expensecategory

import "errors"

var ErrExpenseCategoryAlreadyExists = errors.New("expense category already exists")
var ErrExpenseCategoryNotFound = errors.New("expense category not found")
var ErrInvalidExpenseCategoryInput = errors.New("invalid expense category input")
var ErrBusinessNotResolved = errors.New("business not resolved")
