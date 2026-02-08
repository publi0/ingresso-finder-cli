//go:build darwin && !cgo

package service

import (
	"context"
	"errors"
)

func detectCurrentLocationFromSystem(context.Context) (UserLocation, error) {
	return UserLocation{}, errors.New("system location on darwin requires cgo enabled")
}
