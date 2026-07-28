// Package service contains application services that compose repositories,
// configuration, and framework integrations.
package service

import "github.com/samber/do/v2"

var Package = do.Package(
	do.Lazy(NewActivityPubRequestService),
	do.Lazy(NewActivityPubActorService),
	do.Lazy(NewActivityPubProfileService),
	do.Lazy(NewNotificationService),
)
