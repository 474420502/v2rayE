package tui

import (
	"context"
	"sync"

	"github.com/gcla/gowid"
	"github.com/gcla/gowid/widgets/edit"
	"github.com/gcla/gowid/widgets/holder"
	"github.com/gcla/gowid/widgets/text"
)

type tuiApp struct {
	client *apiClient
	ctx    context.Context
	cancel context.CancelFunc

	app *gowid.App
	mu  sync.Mutex

	page string

	status             CoreStatus
	profiles           []ProfileItem
	subscriptions      []SubscriptionItem
	config             map[string]any
	routing            RoutingConfig
	diagnostics        RoutingDiagnostics
	hits               RoutingHitStats
	routingTest        RoutingTestResult
	availability       AvailabilityResult
	stats              StatsResult
	logLines           []LogLine
	logs               []string
	events             []string
	batchDelay         BatchDelayTestResult
	batchRunning       bool
	logLevelFilter     string
	logSourceFilter    string
	logSearchQuery     string
	logsStreamState    string
	eventsStreamState  string
	settingsDirty      bool
	settingsFormLoaded bool

	selectedProfileID string
	selectedSubID     string

	pageHolder          *holder.Widget
	footer              *text.Widget
	dashboardSummary    *edit.Widget
	dashboardEvents     *edit.Widget
	logsStatus          *text.Widget
	logsView            *edit.Widget
	logsSearchInput     *edit.Widget
	profilesListHolder  *holder.Widget
	profileBatchStatus  *text.Widget
	profileDetail       *edit.Widget
	profileImport       *edit.Widget
	subscriptionsHolder *holder.Widget
	subscriptionDetail  *edit.Widget
	networkSummary      *edit.Widget
	networkTestTarget   *edit.Widget
	networkTestPort     *edit.Widget
	networkTestResult   *edit.Widget
	settingsSummary     *edit.Widget
	settingsListenAddr  *edit.Widget
	settingsSocksPort   *edit.Widget
	settingsHTTPPort    *edit.Widget
	settingsTunName     *edit.Widget
	settingsProxyMode   *edit.Widget
	settingsProxyExcept *edit.Widget
}

func newTUI(ctx context.Context, client *apiClient) *tuiApp {
	childCtx, cancel := context.WithCancel(ctx)
	return &tuiApp{
		client:            client,
		ctx:               childCtx,
		cancel:            cancel,
		page:              "dashboard",
		logs:              make([]string, 0, 256),
		logLines:          make([]LogLine, 0, 256),
		events:            make([]string, 0, 256),
		logLevelFilter:    "all",
		logSourceFilter:   "all",
		logsStreamState:   "idle",
		eventsStreamState: "idle",
	}
}
