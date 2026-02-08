//go:build !darwin

package service

import (
	"context"
	"errors"
)

func detectCurrentLocationFromSystem(context.Context) (UserLocation, error) {
	return UserLocation{}, errors.New("system location is not supported on this OS")
}
