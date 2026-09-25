package restaurant

import "errors"

var (
	ErrNotFound      = errors.New("restaurant not found")
	ErrNotOwner      = errors.New("not the owner of this restaurant")
	ErrImageRequired = errors.New("restaurant must have at least one image")
	ErrImageNotFound = errors.New("image not found")
)
