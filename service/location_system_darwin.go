//go:build darwin && cgo

package service

/*
#cgo CFLAGS: -x objective-c -fobjc-arc
#cgo LDFLAGS: -framework Foundation -framework CoreLocation

#import <CoreFoundation/CoreFoundation.h>
#import <CoreLocation/CoreLocation.h>
#import <Foundation/Foundation.h>
#import <stdlib.h>
#import <string.h>

static BOOL ingresso_status_is_authorized(CLAuthorizationStatus status) {
	return status == kCLAuthorizationStatusAuthorizedAlways || status == kCLAuthorizationStatusAuthorized;
}

static CLAuthorizationStatus ingresso_current_auth_status(CLLocationManager *manager) {
	if ([manager respondsToSelector:@selector(authorizationStatus)]) {
		return manager.authorizationStatus;
	}
#pragma clang diagnostic push
#pragma clang diagnostic ignored "-Wdeprecated-declarations"
	return [CLLocationManager authorizationStatus];
#pragma clang diagnostic pop
}

@interface IngressoLocationDelegate : NSObject<CLLocationManagerDelegate>
@property(nonatomic, strong) CLLocation *location;
@property(nonatomic, strong) NSError *error;
@property(nonatomic, assign) BOOL finished;
@end

@implementation IngressoLocationDelegate

- (void)finishIfNeeded {
	if (!self.finished) {
		self.finished = YES;
		CFRunLoopStop(CFRunLoopGetCurrent());
	}
}

- (void)locationManager:(CLLocationManager *)manager didUpdateLocations:(NSArray<CLLocation *> *)locations {
	self.location = [locations lastObject];
	if (self.location == nil) {
		self.error = [NSError errorWithDomain:@"ingresso.location" code:3 userInfo:@{NSLocalizedDescriptionKey: @"empty location update"}];
	}
	[self finishIfNeeded];
}

- (void)locationManager:(CLLocationManager *)manager didFailWithError:(NSError *)error {
	self.error = error;
	[self finishIfNeeded];
}

- (void)locationManagerDidChangeAuthorization:(CLLocationManager *)manager {
	CLAuthorizationStatus status = ingresso_current_auth_status(manager);
	if (ingresso_status_is_authorized(status)) {
		[manager requestLocation];
		return;
	}
	if (status == kCLAuthorizationStatusDenied || status == kCLAuthorizationStatusRestricted) {
		self.error = [NSError errorWithDomain:@"ingresso.location" code:1 userInfo:@{NSLocalizedDescriptionKey: @"location permission denied"}];
		[self finishIfNeeded];
	}
}

- (void)locationManager:(CLLocationManager *)manager didChangeAuthorizationStatus:(CLAuthorizationStatus)status {
	if (ingresso_status_is_authorized(status)) {
		[manager requestLocation];
		return;
	}
	if (status == kCLAuthorizationStatusDenied || status == kCLAuthorizationStatusRestricted) {
		self.error = [NSError errorWithDomain:@"ingresso.location" code:1 userInfo:@{NSLocalizedDescriptionKey: @"location permission denied"}];
		[self finishIfNeeded];
	}
}

@end

static int ingresso_detect_system_location(double *lat, double *lng, char **err_out) {
	@autoreleasepool {
		if (lat == NULL || lng == NULL) {
			if (err_out != NULL) {
				*err_out = strdup("invalid output pointers");
			}
			return 1;
		}

		if (![CLLocationManager locationServicesEnabled]) {
			if (err_out != NULL) {
				*err_out = strdup("location services are disabled");
			}
			return 1;
		}

		CLLocationManager *manager = [[CLLocationManager alloc] init];
		IngressoLocationDelegate *delegate = [[IngressoLocationDelegate alloc] init];
		manager.delegate = delegate;
		manager.desiredAccuracy = kCLLocationAccuracyNearestTenMeters;

		CLAuthorizationStatus status = ingresso_current_auth_status(manager);
		if (ingresso_status_is_authorized(status)) {
			[manager requestLocation];
		} else if (status == kCLAuthorizationStatusNotDetermined) {
			[manager requestWhenInUseAuthorization];
			[manager requestLocation];
		} else if (status == kCLAuthorizationStatusDenied || status == kCLAuthorizationStatusRestricted) {
			if (err_out != NULL) {
				*err_out = strdup("location permission denied");
			}
			return 1;
		}

		NSTimeInterval remaining = 12.0;
		while (!delegate.finished && remaining > 0) {
			CFRunLoopRunInMode(kCFRunLoopDefaultMode, 0.25, false);
			remaining -= 0.25;
		}

		if (!delegate.finished) {
			if (err_out != NULL) {
				*err_out = strdup("timed out waiting for system location");
			}
			return 1;
		}

		if (delegate.error != nil) {
			const char *msg = [[delegate.error localizedDescription] UTF8String];
			if (err_out != NULL) {
				if (msg != NULL && strlen(msg) > 0) {
					*err_out = strdup(msg);
				} else {
					*err_out = strdup("system location request failed");
				}
			}
			return 1;
		}

		if (delegate.location == nil) {
			if (err_out != NULL) {
				*err_out = strdup("system location request returned no coordinates");
			}
			return 1;
		}

		*lat = delegate.location.coordinate.latitude;
		*lng = delegate.location.coordinate.longitude;
		return 0;
	}
}
*/
import "C"

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"unsafe"
)

func detectCurrentLocationFromSystem(ctx context.Context) (UserLocation, error) {
	if err := ctx.Err(); err != nil {
		return UserLocation{}, err
	}
	location, err := detectCurrentLocationFromSystemBlocking()
	if err != nil {
		if ctxErr := ctx.Err(); ctxErr != nil {
			return UserLocation{}, ctxErr
		}
		return UserLocation{}, err
	}
	if err := ctx.Err(); err != nil {
		return UserLocation{}, err
	}
	return location, nil
}

func detectCurrentLocationFromSystemBlocking() (UserLocation, error) {
	var lat C.double
	var lng C.double
	var errMsg *C.char

	status := C.ingresso_detect_system_location(&lat, &lng, &errMsg)
	if errMsg != nil {
		defer C.free(unsafe.Pointer(errMsg))
	}
	if status != 0 {
		message := ""
		if errMsg != nil {
			message = strings.TrimSpace(C.GoString(errMsg))
		}
		if message == "" {
			message = "unknown error"
		}
		return UserLocation{}, fmt.Errorf("macos corelocation failed: %s", compactProviderErrorSnippet(message))
	}

	location := UserLocation{
		Latitude:  float64(lat),
		Longitude: float64(lng),
		Source:    "system",
	}
	if location.Latitude == 0 && location.Longitude == 0 {
		return UserLocation{}, errors.New("system location returned empty coordinates")
	}
	return location, nil
}
