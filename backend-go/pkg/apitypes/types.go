package apitypes

import "v2raye/backend-go/internal/domain"

type CoreStatus = domain.CoreStatus
type ProfileItem = domain.ProfileItem
type TransportConfig = domain.TransportConfig
type DelayTestResult = domain.DelayTestResult
type BatchDelayTestRequest = domain.BatchDelayTestRequest
type ProfileDelayResult = domain.ProfileDelayResult
type BatchDelayTestResult = domain.BatchDelayTestResult
type SubscriptionItem = domain.SubscriptionItem
type SubscriptionUpsertRequest = domain.SubscriptionUpsertRequest
type AvailabilityResult = domain.AvailabilityResult
type SystemProxyApplyRequest = domain.SystemProxyApplyRequest
type RoutingConfig = domain.RoutingConfig
type RoutingRule = domain.RoutingRule
type RoutingDiagnostics = domain.RoutingDiagnostics
type RoutingTestRequest = domain.RoutingTestRequest
type RoutingTestResult = domain.RoutingTestResult
type RoutingOutboundHit = domain.RoutingOutboundHit
type RoutingHitStats = domain.RoutingHitStats
type TunRepairResult = domain.TunRepairResult
type StatsResult = domain.StatsResult
type LogLine = domain.LogLine

type EventMessage struct {
	Event string      `json:"event"`
	TS    string      `json:"ts"`
	Data  interface{} `json:"data"`
}
