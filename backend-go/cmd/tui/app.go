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

	page       string
	uiLanguage string

	footerStatus string
	viewportCols int
	viewportRows int

	pageHolder  *tview.Pages
	tabBar      *tview.Flex
	focusables  []tview.Primitive
	focusGroups [][]tview.Primitive
	focusGroup  int

	commandPalette       *tview.List
	commandPaletteInput  *tview.InputField
	profileActionsMenu   *tview.List
	paletteActionsCache  []paletteAction
	palettePreviousFocus tview.Primitive
	profileActionsPrev   tview.Primitive
	profileDeletePrev    tview.Primitive
	profileImportInput   *tview.InputField
	profileImportPrev    tview.Primitive
	profileEditPrev      tview.Primitive
	proxyUserSelectMenu  *tview.List
	proxyUserSelectPrev  tview.Primitive

	status              CoreStatus
	profiles            []ProfileItem
	subscriptions       []SubscriptionItem
	config              map[string]any
	routing             RoutingConfig
	diagnostics         RoutingDiagnostics
	hits                RoutingHitStats
	routingTest         RoutingTestResult
	availability        AvailabilityResult
	systemProxyUsers    []SystemProxyUserCandidate
	proxyUsersStatus    string
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
	profileEditSection  string

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
	profileEditName        *inputWidget
	profileEditAddress     *inputWidget
	profileEditPort        *inputWidget
	profileEditNetwork     *inputWidget
	profileEditTLS         *inputWidget
	profileEditSNI         *inputWidget
	profileEditFingerprint *inputWidget
	profileEditALPN        *inputWidget
	profileEditSkipCert    *inputWidget
	profileEditRealityPK   *inputWidget
	profileEditRealitySID  *inputWidget
	profileEditWSPath      *inputWidget
	profileEditH2Path      *inputWidget
	profileEditH2Host      *inputWidget
	profileEditGRPCSvc     *inputWidget
	profileEditGRPCMode    *inputWidget
	profileEditVMessUUID   *inputWidget
	profileEditVMessAlter  *inputWidget
	profileEditVMessSec    *inputWidget
	profileEditVLESSUUID   *inputWidget
	profileEditVLESSFlow   *inputWidget
	profileEditVLESSEnc    *inputWidget
	profileEditSSMethod    *inputWidget
	profileEditSSPassword  *inputWidget
	profileEditSSPlugin    *inputWidget
	profileEditSSPluginOpt *inputWidget
	profileEditTrojanPwd   *inputWidget
	profileEditHy2Pwd      *inputWidget
	profileEditHy2SNI      *inputWidget
	profileEditHy2Insecure *inputWidget
	profileEditHy2UpMbps   *inputWidget
	profileEditHy2DownMbps *inputWidget
	profileEditHy2Obfs     *inputWidget
	profileEditHy2ObfsPwd  *inputWidget
	profileEditTuicUUID    *inputWidget
	profileEditTuicPwd     *inputWidget
	profileEditTuicCC      *inputWidget
	profileEditTuicSNI     *inputWidget
	profileEditTuicInsec   *inputWidget
	profileEditTuicALPN    *inputWidget
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
	settingsProxyUsers     *inputWidget
	settingsCoreEngine     *inputWidget
	settingsLogLevel       *inputWidget
	settingsSkipCert       *inputWidget
	settingsDNSMode        *inputWidget
	settingsDNSList        *inputWidget

	suspendFieldTracking   atomic.Bool
	suspendListSelection   atomic.Bool
	commandPaletteVisible  atomic.Bool
	profileActionsVisible  atomic.Bool
	profileDeleteVisible   atomic.Bool
	profileImportVisible   atomic.Bool
	profileEditVisible     atomic.Bool
	proxyUserSelectVisible atomic.Bool
}

func newTUI(ctx context.Context, client *apiClient) *tuiApp {
	childCtx, cancel := context.WithCancel(ctx)
	lang := detectDefaultUILanguage()
	setGlobalUILanguage(lang)
	return &tuiApp{
		client:            client,
		ctx:               childCtx,
		cancel:            cancel,
		page:              "dashboard",
		uiLanguage:        lang,
		logs:              make([]string, 0, 256),
		logLines:          make([]LogLine, 0, 256),
		events:            make([]string, 0, 256),
		logLevelFilter:    "all",
		logSourceFilter:   "all",
		logsStreamState:   "idle",
		eventsStreamState: "idle",
		proxyUsersStatus:  "not loaded",
		footerStatus:      tr(lang, "status.ready"),
		viewportCols:      0,
	}
}
