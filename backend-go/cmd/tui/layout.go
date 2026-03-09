package tui

import (
	"fmt"

	tcell "github.com/gdamore/tcell/v2"
	"github.com/rivo/tview"
)

func (a *tuiApp) build() tview.Primitive {
	a.footer = newTextWidget(footerText(a.page, a.footerStatus))
	a.footer.SetWrap(false)
	a.pageHolder = tview.NewPages()
	a.tabBar = tview.NewFlex().SetDirection(tview.FlexColumn)
	a.dashboardSummary = readOnlyEditor("")
	a.dashboardEvents = readOnlyEditor("")
	a.logsStatus = newTextWidget(a.t("logs.status.default"))
	a.logsStatus.SetWrap(false)
	a.logsView = readOnlyEditor("")
	a.logsView.SetWrap(false)
	a.logsSearchInput = newInputWidget(a.t("label.search"), nil)
	a.profileDetail = readOnlyEditor(a.t("profiles.detail.empty"))
	a.profileBatchStatus = newTextWidget(a.t("profiles.batch.idle"))
	a.profileEditStatus = newTextWidget(a.t("profiles.editor.idle"))
	a.profileEditName = newInputWidget("name: ", a.profileEditChanged)
	a.profileEditAddress = newInputWidget("address: ", a.profileEditChanged)
	a.profileEditPort = newInputWidget("port: ", a.profileEditChanged)
	a.profileEditNetwork = newInputWidget("network(tcp/ws/grpc): ", a.profileEditChanged)
	a.profileEditTLS = newInputWidget("tls(true/false): ", a.profileEditChanged)
	a.profileEditSNI = newInputWidget("sni: ", a.profileEditChanged)
	a.profileEditFingerprint = newInputWidget("fingerprint: ", a.profileEditChanged)
	a.profileEditALPN = newInputWidget("alpn(csv): ", a.profileEditChanged)
	a.profileEditSkipCert = newInputWidget("skipCertVerify(true/false): ", a.profileEditChanged)
	a.profileEditRealityPK = newInputWidget("realityPublicKey: ", a.profileEditChanged)
	a.profileEditRealitySID = newInputWidget("realityShortId: ", a.profileEditChanged)
	a.profileEditWSPath = newInputWidget("wsPath: ", a.profileEditChanged)
	a.profileEditH2Path = newInputWidget("h2Path(csv): ", a.profileEditChanged)
	a.profileEditH2Host = newInputWidget("h2Host(csv): ", a.profileEditChanged)
	a.profileEditGRPCSvc = newInputWidget("grpcServiceName: ", a.profileEditChanged)
	a.profileEditGRPCMode = newInputWidget("grpcMode(gun|multi): ", a.profileEditChanged)
	a.profileEditVMessUUID = newInputWidget("vmess.uuid: ", a.profileEditChanged)
	a.profileEditVMessAlter = newInputWidget("vmess.alterId: ", a.profileEditChanged)
	a.profileEditVMessSec = newInputWidget("vmess.security: ", a.profileEditChanged)
	a.profileEditVLESSUUID = newInputWidget("vless.uuid: ", a.profileEditChanged)
	a.profileEditVLESSFlow = newInputWidget("vless.flow: ", a.profileEditChanged)
	a.profileEditVLESSEnc = newInputWidget("vless.encryption: ", a.profileEditChanged)
	a.profileEditSSMethod = newInputWidget("ss.method: ", a.profileEditChanged)
	a.profileEditSSPassword = newInputWidget("ss.password: ", a.profileEditChanged)
	a.profileEditSSPlugin = newInputWidget("ss.plugin: ", a.profileEditChanged)
	a.profileEditSSPluginOpt = newInputWidget("ss.pluginOpts: ", a.profileEditChanged)
	a.profileEditTrojanPwd = newInputWidget("trojan.password: ", a.profileEditChanged)
	a.profileEditHy2Pwd = newInputWidget("hy2.password: ", a.profileEditChanged)
	a.profileEditHy2SNI = newInputWidget("hy2.sni: ", a.profileEditChanged)
	a.profileEditHy2Insecure = newInputWidget("hy2.insecure(true/false): ", a.profileEditChanged)
	a.profileEditHy2UpMbps = newInputWidget("hy2.upMbps: ", a.profileEditChanged)
	a.profileEditHy2DownMbps = newInputWidget("hy2.downMbps: ", a.profileEditChanged)
	a.profileEditHy2Obfs = newInputWidget("hy2.obfs: ", a.profileEditChanged)
	a.profileEditHy2ObfsPwd = newInputWidget("hy2.obfsPassword: ", a.profileEditChanged)
	a.profileEditTuicUUID = newInputWidget("tuic.uuid: ", a.profileEditChanged)
	a.profileEditTuicPwd = newInputWidget("tuic.password: ", a.profileEditChanged)
	a.profileEditTuicCC = newInputWidget("tuic.congestionControl: ", a.profileEditChanged)
	a.profileEditTuicSNI = newInputWidget("tuic.sni: ", a.profileEditChanged)
	a.profileEditTuicInsec = newInputWidget("tuic.insecure(true/false): ", a.profileEditChanged)
	a.profileEditTuicALPN = newInputWidget("tuic.alpn(csv): ", a.profileEditChanged)
	a.profileDeleteConfirm = newInputWidget("delete confirm (type DELETE): ", nil)
	a.profilesList = newListWidget()
	a.subscriptionDetail = readOnlyEditor("Select a subscription to inspect.")
	a.subscriptionsList = newListWidget()
	a.networkSummary = readOnlyEditor("")
	a.networkRoutingMode = newInputWidget("targetMode(global|bypass_cn|direct|custom): ", a.networkRoutingChanged)
	a.networkDomainStrategy = newInputWidget("targetDomainStrategy(IPIfNonMatch|IPOnDemand|AsIs): ", a.networkRoutingChanged)
	a.networkLocalBypass = newInputWidget("targetLocalBypass(true/false): ", a.networkRoutingChanged)
	a.networkTestTarget = newInputWidget("target: ", nil)
	a.networkTestPort = newInputWidget("port: ", nil)
	a.networkTestResult = readOnlyEditor("No routing test executed.")
	a.settingsSummary = readOnlyEditor("")
	a.settingsListenAddr = newInputWidget("listenAddr: ", a.settingsChanged)
	a.settingsSocksPort = newInputWidget("socksPort: ", a.settingsChanged)
	a.settingsHTTPPort = newInputWidget("httpPort: ", a.settingsChanged)
	a.settingsTunName = newInputWidget("tunName: ", a.settingsChanged)
	a.settingsTunMode = newInputWidget("tunMode(off|system|mixed|gvisor): ", a.settingsChanged)
	a.settingsTunMtu = newInputWidget("tunMtu: ", a.settingsChanged)
	a.settingsTunAutoRoute = newInputWidget("tunAutoRoute(true/false): ", a.settingsChanged)
	a.settingsTunStrict = newInputWidget("tunStrictRoute(true/false): ", a.settingsChanged)
	a.settingsProxyMode = newInputWidget("proxyMode(forced_change|forced_clear|pac): ", a.settingsChanged)
	a.settingsProxyExcept = newInputWidget("proxyExceptions: ", a.settingsChanged)
	a.settingsCoreEngine = newInputWidget("coreEngine(xray-core): ", a.settingsChanged)
	a.settingsLogLevel = newInputWidget("logLevel(debug|info|warning|error): ", a.settingsChanged)
	a.settingsSkipCert = newInputWidget("skipCertVerify(true/false): ", a.settingsChanged)
	a.settingsDNSMode = newInputWidget("dnsMode: ", a.settingsChanged)
	a.settingsDNSList = newInputWidget("dnsList(csv): ", a.settingsChanged)

	a.profilesList.SetChangedFunc(func(index int, mainText, secondaryText string, shortcut rune) {
		a.onProfileSelectionChanged(index)
	})
	a.profilesList.SetSelectedFunc(func(index int, mainText, secondaryText string, shortcut rune) {
		a.onProfileSelectionChanged(index)
		a.openProfileActionsMenu()
	})
	a.subscriptionsList.SetChangedFunc(func(index int, mainText, secondaryText string, shortcut rune) {
		a.onSubscriptionSelectionChanged(index)
	})

	title := tview.NewTextView()
	title.SetText(a.t("layout.title"))
	title.SetTextColor(tcell.ColorBlack)
	title.SetBackgroundColor(tcell.ColorTeal)

	help := newMutedText(a.t("layout.shortcuts"))

	root := tview.NewFlex().SetDirection(tview.FlexRow)
	root.AddItem(title, 1, 0, false)
	root.AddItem(a.tabBar, 1, 0, false)
	root.AddItem(help, 1, 0, false)
	root.AddItem(a.pageHolder, 0, 1, true)
	root.AddItem(a.footer, 1, 0, false)

	a.syncPages()
	return root
}

func (a *tuiApp) buildTabs() {
	if a.tabBar == nil {
		return
	}
	a.tabBar.Clear()
	for idx, tab := range tuiPageTabs() {
		tab := tab
		label := fmt.Sprintf("%c %s", tab.shortcut, pageDisplayName(tab.key))
		btn := tview.NewButton(label)
		btn.SetSelectedFunc(func() {
			a.setActivePage(tab.key)
		})
		btn.SetLabelColor(tcell.ColorWhite)
		btn.SetLabelColorActivated(tcell.ColorBlack)
		if tab.key == a.page {
			btn.SetBackgroundColor(tcell.ColorGreen)
			btn.SetBackgroundColorActivated(tcell.ColorYellow)
		} else {
			btn.SetBackgroundColor(tcell.ColorDarkBlue)
			btn.SetBackgroundColorActivated(tcell.ColorYellow)
		}
		a.tabBar.AddItem(btn, buttonWidth(label), 0, false)
		if idx != len(tuiPageTabs())-1 {
			a.tabBar.AddItem(horizontalSpacer(1), 1, 0, false)
		}
	}
}

func (a *tuiApp) syncPages() {
	if a.pageHolder == nil {
		return
	}

	var page builtPage
	switch a.page {
	case pageProfiles:
		page = a.buildProfilesPage()
	case pageSubscriptions:
		page = a.buildSubscriptionsPage()
	case pageNetwork:
		page = a.buildNetworkPage()
	case pageSettings:
		page = a.buildSettingsPage()
	case pageLogs:
		page = a.buildLogsPage()
	default:
		page = a.buildDashboardPage()
	}

	a.focusables = page.focusables
	a.focusGroups = nil
	for _, group := range page.focusGroups {
		if len(group) == 0 {
			continue
		}
		a.focusGroups = append(a.focusGroups, group)
	}
	if len(a.focusGroups) == 0 && len(a.focusables) > 0 {
		a.focusGroups = [][]tview.Primitive{a.focusables}
	}
	a.focusGroup = 0
	a.buildTabs()
	a.pageHolder.RemovePage("current")
	a.pageHolder.AddAndSwitchToPage("current", page.root, true)
	if a.app != nil {
		if len(a.focusGroups) > 0 && len(a.focusGroups[0]) > 0 {
			a.app.SetFocus(a.focusGroups[0][0])
		} else if len(a.focusables) > 0 {
			a.app.SetFocus(a.focusables[0])
		}
	}
	a.refreshFooter()
}

func (a *tuiApp) settingsChanged(string) {
	if a.fieldTrackingSuspended() {
		return
	}
	a.markSettingsDirty()
}

func (a *tuiApp) networkRoutingChanged(string) {
	if a.fieldTrackingSuspended() {
		return
	}
	a.markNetworkRoutingDirty()
}

func (a *tuiApp) profileEditChanged(string) {
	if a.fieldTrackingSuspended() {
		return
	}
	a.markProfileEditDirty()
}

func (a *tuiApp) onProfileSelectionChanged(index int) {
	if a.suspendListSelection.Load() {
		return
	}
	profiles := a.sortedProfilesForDisplay()
	if index < 0 || index >= len(profiles) {
		return
	}
	selected := profiles[index]
	a.mu.Lock()
	if a.selectedProfileID == selected.ID {
		a.mu.Unlock()
		return
	}
	a.selectedProfileID = selected.ID
	a.mu.Unlock()
	a.refreshWidgets()
	a.setFooter(a.tf("footer.selectedProfile", emptyFallback(selected.Name, selected.ID)))
}

func (a *tuiApp) onSubscriptionSelectionChanged(index int) {
	if a.suspendListSelection.Load() {
		return
	}
	if index < 0 || index >= len(a.subscriptions) {
		return
	}
	selected := a.subscriptions[index]
	a.mu.Lock()
	if a.selectedSubID == selected.ID {
		a.mu.Unlock()
		return
	}
	a.selectedSubID = selected.ID
	a.mu.Unlock()
	a.refreshWidgets()
	a.setFooter(a.tf("footer.selectedSubscription", emptyFallback(selected.Remarks, selected.ID)))
}
