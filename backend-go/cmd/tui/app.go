package tui

import (
	"context"
	"sync"
	"sync/atomic"

	"github.com/rivo/tview"
)

type tuiApp struct {
	client *apiClient
	ctx    context.Context
	cancel context.CancelFunc

	app *tview.Application
	mu  sync.Mutex

	page string

	footerStatus string
	viewportCols int

	pageHolder *tview.Pages
	tabBar     *tview.Flex
	focusables []tview.Primitive

	status              CoreStatus
	profiles            []ProfileItem
	subscriptions       []SubscriptionItem
	config              map[string]any
	routing             RoutingConfig
	diagnostics         RoutingDiagnostics
	hits                RoutingHitStats
	routingTest         RoutingTestResult
	availability        AvailabilityResult
	stats               StatsResult
	logLines            []LogLine
	logs                []string
	events              []string
	batchDelay          BatchDelayTestResult
	batchRunning        bool
	logLevelFilter      string
	logSourceFilter     string
	logSearchQuery      string
	logsStreamState     string
	eventsStreamState   string
	settingsDirty       bool
	settingsFormLoaded  bool
	networkRoutingDirty bool
	profileEditDirty    bool
	profileEditLoaded   bool
	profileEditForID    string
	profileEditMessage  string

	selectedProfileID string
	selectedSubID     string

	footer                 *textWidget
	dashboardSummary       *textWidget
	dashboardEvents        *textWidget
	logsStatus             *textWidget
	logsView               *textWidget
	logsSearchInput        *inputWidget
	profilesList           *tview.List
	profileBatchStatus     *textWidget
	profileEditStatus      *textWidget
	profileDetail          *textWidget
	profileImport          *inputWidget
	profileEditName        *inputWidget
	profileEditAddress     *inputWidget
	profileEditPort        *inputWidget
	profileEditNetwork     *inputWidget
	profileEditTLS         *inputWidget
	profileEditSNI         *inputWidget
	profileEditFingerprint *inputWidget
	profileEditALPN        *inputWidget
	profileEditRealityPK   *inputWidget
	profileEditRealitySID  *inputWidget
	profileEditWSPath      *inputWidget
	profileEditGRPCSvc     *inputWidget
	profileDeleteConfirm   *inputWidget
	subscriptionsList      *tview.List
	subscriptionDetail     *textWidget
	networkSummary         *textWidget
	networkRoutingMode     *inputWidget
	networkDomainStrategy  *inputWidget
	networkLocalBypass     *inputWidget
	networkTestTarget      *inputWidget
	networkTestPort        *inputWidget
	networkTestResult      *textWidget
	settingsSummary        *textWidget
	settingsListenAddr     *inputWidget
	settingsSocksPort      *inputWidget
	settingsHTTPPort       *inputWidget
	settingsTunName        *inputWidget
	settingsTunMode        *inputWidget
	settingsTunMtu         *inputWidget
	settingsTunAutoRoute   *inputWidget
	settingsTunStrict      *inputWidget
	settingsProxyMode      *inputWidget
	settingsProxyExcept    *inputWidget
	settingsCoreEngine     *inputWidget
	settingsLogLevel       *inputWidget
	settingsSkipCert       *inputWidget
	settingsDNSMode        *inputWidget
	settingsDNSList        *inputWidget

	suspendFieldTracking atomic.Bool
	suspendListSelection atomic.Bool
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
		footerStatus:      "Ready",
		viewportCols:      0,
	}
}
