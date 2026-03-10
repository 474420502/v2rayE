package tui

import (
	"context"
	"sync"
	"sync/atomic"

	"v2raye/backend-go/cmd/tui/components"

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
	layoutMode   string

	pageHolder  *tview.Pages
	tabBar      *tview.Flex
	helpBar     *tview.TextView // 动态帮助/导航栏 (窄屏时切换为 Tab 导航)
	sidebar     *components.Sidebar
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
	dropdownSelectMenu   *tview.List
	dropdownSelectPrev   tview.Primitive
	dropdownSelectTarget *dropdownWidget
	lastDropdownFocus    *tview.DropDown
	dropdownCancelValue  string
	dropdownCancelArmed  bool

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
	dashboardStatus        *textWidget
	dashboardTelemetry     *textWidget
	dashboardConfig        *textWidget
	dashboardEvents        *textWidget
	logsStatus             *textWidget
	logsView               *textWidget
	logsLevelSelect        *dropdownWidget
	logsSourceSelect       *dropdownWidget
	logsSearchInput        *inputWidget
	profilesList           *tview.List
	profileBatchStatus     *textWidget
	profileEditStatus      *textWidget
	profileDetail          *textWidget
	profileEditName        *inputWidget
	profileEditAddress     *inputWidget
	profileEditPort        *inputWidget
	profileEditNetwork     *dropdownWidget
	profileEditTLS         *dropdownWidget
	profileEditSNI         *inputWidget
	profileEditFingerprint *inputWidget
	profileEditALPN        *inputWidget
	profileEditSkipCert    *dropdownWidget
	profileEditRealityPK   *inputWidget
	profileEditRealitySID  *inputWidget
	profileEditWSPath      *inputWidget
	profileEditH2Path      *inputWidget
	profileEditH2Host      *inputWidget
	profileEditGRPCSvc     *inputWidget
	profileEditGRPCMode    *dropdownWidget
	profileEditVMessUUID   *inputWidget
	profileEditVMessAlter  *inputWidget
	profileEditVMessSec    *dropdownWidget
	profileEditVLESSUUID   *inputWidget
	profileEditVLESSFlow   *inputWidget
	profileEditVLESSEnc    *dropdownWidget
	profileEditSSMethod    *inputWidget
	profileEditSSPassword  *inputWidget
	profileEditSSPlugin    *inputWidget
	profileEditSSPluginOpt *inputWidget
	profileEditTrojanPwd   *inputWidget
	profileEditHy2Pwd      *inputWidget
	profileEditHy2SNI      *inputWidget
	profileEditHy2Insecure *dropdownWidget
	profileEditHy2UpMbps   *inputWidget
	profileEditHy2DownMbps *inputWidget
	profileEditHy2Obfs     *inputWidget
	profileEditHy2ObfsPwd  *inputWidget
	profileEditTuicUUID    *inputWidget
	profileEditTuicPwd     *inputWidget
	profileEditTuicCC      *dropdownWidget
	profileEditTuicSNI     *inputWidget
	profileEditTuicInsec   *dropdownWidget
	profileEditTuicALPN    *inputWidget
	profileDeleteConfirm   *inputWidget
	subscriptionsList      *tview.List
	subscriptionDetail     *textWidget
	networkSummary         *textWidget
	networkPresetSelect    *dropdownWidget
	networkRoutingMode     *dropdownWidget
	networkDomainStrategy  *dropdownWidget
	networkLocalBypass     *dropdownWidget
	networkTestTarget      *inputWidget
	networkTestPort        *inputWidget
	networkTestResult      *textWidget
	settingsSummary        *textWidget
	settingsListenAddr     *inputWidget
	settingsSocksPort      *inputWidget
	settingsHTTPPort       *inputWidget
	settingsLanguage       *dropdownWidget
	settingsTunName        *inputWidget
	settingsTunMode        *dropdownWidget
	settingsTunMtu         *inputWidget
	settingsTunAutoRoute   *dropdownWidget
	settingsTunStrict      *dropdownWidget
	settingsProxyMode      *dropdownWidget
	settingsProxyExcept    *inputWidget
	settingsProxyUsers     *inputWidget
	settingsCoreEngine     *dropdownWidget
	settingsLogLevel       *dropdownWidget
	settingsSkipCert       *dropdownWidget
	settingsDNSMode        *dropdownWidget
	settingsDNSList        *inputWidget

	suspendFieldTracking   atomic.Bool
	suspendListSelection   atomic.Bool
	commandPaletteVisible  atomic.Bool
	profileActionsVisible  atomic.Bool
	profileDeleteVisible   atomic.Bool
	profileImportVisible   atomic.Bool
	profileEditVisible     atomic.Bool
	proxyUserSelectVisible atomic.Bool
	dropdownSelectVisible  atomic.Bool
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
		logsStreamState:   tr(lang, "state.idle"),
		eventsStreamState: tr(lang, "state.idle"),
		proxyUsersStatus:  tr(lang, "state.notLoaded"),
		footerStatus:      tr(lang, "status.ready"),
		viewportCols:      0,
	}
}
