package service

import (
	"errors"

	"v2raye/backend-go/internal/domain"
)

var (
	ErrNotFound               = errors.New("not found")
	ErrInvalidMode            = errors.New("invalid mode")
	ErrSystemProxyUnsupported = errors.New("system proxy unsupported")
)

// BackendService is the full interface implemented by the native service.
type BackendService interface {
	// Core lifecycle
	CoreStatus() domain.CoreStatus
	StartCore() domain.CoreStatus
	StopCore() domain.CoreStatus
	RestartCore() domain.CoreStatus
	ClearCoreError() domain.CoreStatus

	// Profile management
	ListProfiles() []domain.ProfileItem
	GetProfile(id string) (domain.ProfileItem, error)
	CreateProfile(input domain.ProfileItem) (domain.ProfileItem, error)
	UpdateProfile(id string, input domain.ProfileItem) (domain.ProfileItem, error)
	DeleteProfile(id string) error
	DeleteProfiles(ids []string) error
	SelectProfile(id string) error
	TestProfileDelay(id string) domain.DelayTestResult
	ImportProfileFromURI(uri string) (domain.ProfileItem, error)

	// Subscription management
	ListSubscriptions() []domain.SubscriptionItem
	CreateSubscription(input domain.SubscriptionUpsertRequest) (domain.SubscriptionItem, error)
	UpdateSubscription(id string, input domain.SubscriptionUpsertRequest) (domain.SubscriptionItem, error)
	DeleteSubscription(id string) error
	UpdateSubscriptions() int
	UpdateSubscriptionByID(id string) error

	// Network & system proxy
	NetworkAvailability() domain.AvailabilityResult
	ApplySystemProxy(mode, exceptions string) (map[string]interface{}, error)

	// App configuration
	GetConfig() map[string]interface{}
	UpdateConfig(next map[string]interface{}) map[string]interface{}

	// Routing
	GetRoutingConfig() domain.RoutingConfig
	UpdateRoutingConfig(rc domain.RoutingConfig) domain.RoutingConfig

	// Bandwidth statistics
	GetStats() domain.StatsResult

	// Core log streaming — returns a channel and a cancel func.
	SubscribeCoreLogs() (<-chan domain.LogLine, func())
}
