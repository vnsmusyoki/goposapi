package expensesubcategory

import "errors"

var ErrExpenseSubCategoryAlreadyExists = errors.New("expense sub category already exists")
var ErrExpenseSubCategoryNotFound = errors.New("expense sub category not found")
var ErrInvalidExpenseSubCategoryInput = errors.New("invalid expense sub category input")
var ErrBusinessNotResolved = errors.New("business not resolved")
