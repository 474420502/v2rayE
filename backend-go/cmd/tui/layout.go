package tui

import (
	"fmt"
	"strings"
	"v2raye/backend-go/cmd/tui/components"

	tcell "github.com/gdamore/tcell/v2"
	"github.com/rivo/tview"
)

func (a *tuiApp) build() tview.Primitive {
	a.footer = newTextWidget(footerText(a.page, a.footerStatus))
	a.footer.SetWrap(false)
	a.pageHolder = tview.NewPages()
	a.sidebar = components.NewSidebar(nil, func(page string) {
		a.navigateToPage(page)
	}, nil)

	// 设置侧边栏的初始状态
	a.updateSidebarMode()
	a.dashboardStatus = readOnlyEditor("")
	a.dashboardTelemetry = readOnlyEditor("")
	a.dashboardConfig = readOnlyEditor("")
	a.dashboardEvents = readOnlyEditor("")
	a.logsStatus = newTextWidget(a.t("logs.status.default"))
	a.logsStatus.SetWrap(false)
	a.logsView = readOnlyEditor("")
	a.logsView.SetWrap(false)
	a.logsLevelSelect = newDropdownWidget("", []selectOption{{Label: "all", Value: "all"}, {Label: "error", Value: "error"}, {Label: "warning", Value: "warning"}, {Label: "info", Value: "info"}, {Label: "debug", Value: "debug"}}, func(value string) {
		if a.fieldTrackingSuspended() {
			return
		}
		a.storeLogLevelFilter(value)
		a.refreshLogsWidget()
	})
	a.logsSourceSelect = newDropdownWidget("", []selectOption{{Label: "all", Value: "all"}, {Label: "app", Value: "app"}, {Label: "xray-core", Value: "xray-core"}}, func(value string) {
		if a.fieldTrackingSuspended() {
			return
		}
		a.storeLogSourceFilter(value)
		a.refreshLogsWidget()
	})
	a.logsSearchInput = newInputWidget(a.t("label.search"), nil)
	a.profileDetail = readOnlyEditor(a.t("profiles.detail.empty"))
	a.profileBatchStatus = newTextWidget(a.t("profiles.batch.idle"))
	a.profileEditStatus = newTextWidget(a.t("profiles.editor.idle"))
	a.profileEditName = newInputWidget("name: ", a.profileEditChanged)
	a.profileEditAddress = newInputWidget("address: ", a.profileEditChanged)
	a.profileEditPort = newInputWidget("port: ", a.profileEditChanged)
	a.profileEditNetwork = newDropdownWidget("network: ", prependEmptyOption([]selectOption{{Label: "tcp", Value: "tcp"}, {Label: "ws", Value: "ws"}, {Label: "grpc", Value: "grpc"}, {Label: "h2", Value: "h2"}, {Label: "kcp", Value: "kcp"}, {Label: "quic", Value: "quic"}, {Label: "xhttp", Value: "xhttp"}}), func(string) {
		if a.fieldTrackingSuspended() {
			return
		}
		a.markProfileEditDirty()
	})
	a.profileEditTLS = newDropdownWidget("tls: ", boolSelectOptions(), func(string) {
		if a.fieldTrackingSuspended() {
			return
		}
		a.markProfileEditDirty()
	})
	a.profileEditSNI = newInputWidget("sni: ", a.profileEditChanged)
	a.profileEditFingerprint = newInputWidget("fingerprint: ", a.profileEditChanged)
	a.profileEditALPN = newInputWidget("alpn(csv): ", a.profileEditChanged)
	a.profileEditSkipCert = newDropdownWidget("skipCertVerify: ", boolSelectOptions(), func(string) {
		if a.fieldTrackingSuspended() {
			return
		}
		a.markProfileEditDirty()
	})
	a.profileEditRealityPK = newInputWidget("realityPublicKey: ", a.profileEditChanged)
	a.profileEditRealitySID = newInputWidget("realityShortId: ", a.profileEditChanged)
	a.profileEditWSPath = newInputWidget("wsPath: ", a.profileEditChanged)
	a.profileEditH2Path = newInputWidget("h2Path(csv): ", a.profileEditChanged)
	a.profileEditH2Host = newInputWidget("h2Host(csv): ", a.profileEditChanged)
	a.profileEditGRPCSvc = newInputWidget("grpcServiceName: ", a.profileEditChanged)
	a.profileEditGRPCMode = newDropdownWidget("grpcMode: ", prependEmptyOption([]selectOption{{Label: "gun", Value: "gun"}, {Label: "multi", Value: "multi"}}), func(string) {
		if a.fieldTrackingSuspended() {
			return
		}
		a.markProfileEditDirty()
	})
	a.profileEditVMessUUID = newInputWidget("vmess.uuid: ", a.profileEditChanged)
	a.profileEditVMessAlter = newInputWidget("vmess.alterId: ", a.profileEditChanged)
	a.profileEditVMessSec = newDropdownWidget("vmess.security: ", prependEmptyOption([]selectOption{{Label: "none", Value: "none"}, {Label: "auto", Value: "auto"}, {Label: "aes-128-gcm", Value: "aes-128-gcm"}, {Label: "chacha20-poly1305", Value: "chacha20-poly1305"}}), func(string) {
		if a.fieldTrackingSuspended() {
			return
		}
		a.markProfileEditDirty()
	})
	a.profileEditVLESSUUID = newInputWidget("vless.uuid: ", a.profileEditChanged)
	a.profileEditVLESSFlow = newInputWidget("vless.flow: ", a.profileEditChanged)
	a.profileEditVLESSEnc = newDropdownWidget("vless.encryption: ", prependEmptyOption([]selectOption{{Label: "none", Value: "none"}}), func(string) {
		if a.fieldTrackingSuspended() {
			return
		}
		a.markProfileEditDirty()
	})
	a.profileEditSSMethod = newInputWidget("ss.method: ", a.profileEditChanged)
	a.profileEditSSPassword = newInputWidget("ss.password: ", a.profileEditChanged)
	a.profileEditSSPlugin = newInputWidget("ss.plugin: ", a.profileEditChanged)
	a.profileEditSSPluginOpt = newInputWidget("ss.pluginOpts: ", a.profileEditChanged)
	a.profileEditTrojanPwd = newInputWidget("trojan.password: ", a.profileEditChanged)
	a.profileEditHy2Pwd = newInputWidget("hy2.password: ", a.profileEditChanged)
	a.profileEditHy2SNI = newInputWidget("hy2.sni: ", a.profileEditChanged)
	a.profileEditHy2Insecure = newDropdownWidget("hy2.insecure: ", boolSelectOptions(), func(string) {
		if a.fieldTrackingSuspended() {
			return
		}
		a.markProfileEditDirty()
	})
	a.profileEditHy2UpMbps = newInputWidget("hy2.upMbps: ", a.profileEditChanged)
	a.profileEditHy2DownMbps = newInputWidget("hy2.downMbps: ", a.profileEditChanged)
	a.profileEditHy2Obfs = newInputWidget("hy2.obfs: ", a.profileEditChanged)
	a.profileEditHy2ObfsPwd = newInputWidget("hy2.obfsPassword: ", a.profileEditChanged)
	a.profileEditTuicUUID = newInputWidget("tuic.uuid: ", a.profileEditChanged)
	a.profileEditTuicPwd = newInputWidget("tuic.password: ", a.profileEditChanged)
	a.profileEditTuicCC = newDropdownWidget("tuic.congestionControl: ", prependEmptyOption([]selectOption{{Label: "bbr", Value: "bbr"}, {Label: "cubic", Value: "cubic"}, {Label: "new_reno", Value: "new_reno"}}), func(string) {
		if a.fieldTrackingSuspended() {
			return
		}
		a.markProfileEditDirty()
	})
	a.profileEditTuicSNI = newInputWidget("tuic.sni: ", a.profileEditChanged)
	a.profileEditTuicInsec = newDropdownWidget("tuic.insecure: ", boolSelectOptions(), func(string) {
		if a.fieldTrackingSuspended() {
			return
		}
		a.markProfileEditDirty()
	})
	a.profileEditTuicALPN = newInputWidget("tuic.alpn(csv): ", a.profileEditChanged)
	a.profileDeleteConfirm = newInputWidget("delete confirm (type DELETE): ", nil)
	a.profilesList = newListWidget()
	a.subscriptionDetail = readOnlyEditor("Select a subscription to inspect.")
	a.subscriptionsList = newListWidget()
	a.networkSummary = readOnlyEditor("")
	a.networkPresetSelect = newDropdownWidget("", []selectOption{{Label: "(empty)", Value: ""}, {Label: "global", Value: "global"}, {Label: "bypass_cn", Value: "bypass_cn"}, {Label: "direct", Value: "direct"}}, func(value string) {
		if a.fieldTrackingSuspended() {
			return
		}
		if strings.TrimSpace(value) == "" {
			a.resetNetworkRoutingForm()
			return
		}
		a.applyRoutingPresetToForm(value)
	})
	a.networkRoutingMode = newDropdownWidget("", []selectOption{{Label: "global", Value: "global"}, {Label: "bypass_cn", Value: "bypass_cn"}, {Label: "direct", Value: "direct"}, {Label: "custom", Value: "custom"}}, func(string) {
		if a.fieldTrackingSuspended() {
			return
		}
		a.markNetworkRoutingDirtyFromManualEdit()
	})
	a.networkDomainStrategy = newDropdownWidget("", []selectOption{{Label: "IPIfNonMatch", Value: "IPIfNonMatch"}, {Label: "IPOnDemand", Value: "IPOnDemand"}, {Label: "AsIs", Value: "AsIs"}}, func(string) {
		if a.fieldTrackingSuspended() {
			return
		}
		a.markNetworkRoutingDirtyFromManualEdit()
	})
	a.networkLocalBypass = newDropdownWidget("", []selectOption{{Label: "true", Value: "true"}, {Label: "false", Value: "false"}}, func(string) {
		if a.fieldTrackingSuspended() {
			return
		}
		a.markNetworkRoutingDirtyFromManualEdit()
	})
	a.networkTestTarget = newInputWidget("target: ", nil)
	a.networkTestPort = newInputWidget("port: ", nil)
	a.networkTestResult = readOnlyEditor("No routing test executed.")
	a.settingsSummary = readOnlyEditor("")
	a.settingsListenAddr = newInputWidget("listenAddr: ", a.settingsChanged)
	a.settingsSocksPort = newInputWidget("socksPort: ", a.settingsChanged)
	a.settingsHTTPPort = newInputWidget("httpPort: ", a.settingsChanged)
	a.settingsLanguage = newDropdownWidget("", []selectOption{{Label: "English", Value: uiLangEN}, {Label: "中文", Value: uiLangZH}}, func(value string) {
		if a.fieldTrackingSuspended() {
			return
		}
		a.setUILanguage(value)
	})
	a.settingsTunName = newInputWidget("tunName: ", a.settingsChanged)
	a.settingsTunMode = newDropdownWidget("tunMode: ", []selectOption{{Label: "off", Value: "off"}, {Label: "system", Value: "system"}, {Label: "mixed", Value: "mixed"}, {Label: "gvisor", Value: "gvisor"}}, func(string) {
		if a.fieldTrackingSuspended() {
			return
		}
		a.markSettingsDirty()
	})
	a.settingsTunMtu = newInputWidget("tunMtu: ", a.settingsChanged)
	a.settingsTunAutoRoute = newDropdownWidget("tunAutoRoute: ", boolSelectOptions(), func(string) {
		if a.fieldTrackingSuspended() {
			return
		}
		a.markSettingsDirty()
	})
	a.settingsTunStrict = newDropdownWidget("tunStrictRoute: ", boolSelectOptions(), func(string) {
		if a.fieldTrackingSuspended() {
			return
		}
		a.markSettingsDirty()
	})
	a.settingsProxyMode = newDropdownWidget("proxyMode: ", []selectOption{{Label: "forced_change", Value: "forced_change"}, {Label: "forced_clear", Value: "forced_clear"}, {Label: "pac", Value: "pac"}}, func(string) {
		if a.fieldTrackingSuspended() {
			return
		}
		a.markSettingsDirty()
	})
	a.settingsLocalProxyMode = newDropdownWidget("localProxyMode: ", []selectOption{{Label: "follow-routing", Value: "follow-routing"}, {Label: "force-proxy", Value: "force-proxy"}}, func(string) {
		if a.fieldTrackingSuspended() {
			return
		}
		a.markSettingsDirty()
	})
	a.settingsProxyExcept = newInputWidget("proxyExceptions: ", a.settingsChanged)
	a.settingsProxyUsers = newInputWidget("proxyUsers(csv): ", a.settingsChanged)
	a.settingsCoreEngine = newDropdownWidget("coreEngine: ", []selectOption{{Label: "xray-core", Value: "xray-core"}}, func(string) {
		if a.fieldTrackingSuspended() {
			return
		}
		a.markSettingsDirty()
	})
	a.settingsLogLevel = newDropdownWidget("logLevel: ", []selectOption{{Label: "debug", Value: "debug"}, {Label: "info", Value: "info"}, {Label: "warning", Value: "warning"}, {Label: "error", Value: "error"}}, func(string) {
		if a.fieldTrackingSuspended() {
			return
		}
		a.markSettingsDirty()
	})
	a.settingsSkipCert = newDropdownWidget("skipCertVerify: ", boolSelectOptions(), func(string) {
		if a.fieldTrackingSuspended() {
			return
		}
		a.markSettingsDirty()
	})
	a.settingsDNSMode = newDropdownWidget("dnsMode: ", []selectOption{{Label: "UseSystemDNS", Value: "UseSystemDNS"}, {Label: "UseDNSList", Value: "UseDNSList"}, {Label: "Direct", Value: "Direct"}}, func(string) {
		if a.fieldTrackingSuspended() {
			return
		}
		a.markSettingsDirty()
	})
	a.settingsDNSList = newInputWidget("dnsList(csv): ", a.settingsChanged)
	a.refreshDropdownLabels()
	a.refreshFieldLabels()

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
	a.helpBar = help
	a.contentLayout = tview.NewFlex().SetDirection(tview.FlexColumn)
	a.sidebarSpacer = tview.NewBox().SetBackgroundColor(tcell.ColorBlack)
	content := a.contentLayout

	// 声明 root 变量
	var root *tview.Flex

	content.AddItem(a.sidebar, 0, 0, false)
	content.AddItem(a.sidebarSpacer, 0, 0, false)
	content.AddItem(a.pageHolder, 0, 1, true)

	root = tview.NewFlex().SetDirection(tview.FlexRow)
	root.AddItem(title, 1, 0, false)
	root.AddItem(help, 1, 0, false)
	root.AddItem(content, 0, 1, true)
	root.AddItem(a.footer, 1, 0, false)

	a.rootPages = tview.NewPages()
	a.rootPages.AddPage("main", root, true, true)

	a.updateSidebarMode()
	a.syncPages()
	return a.rootPages
}

func (a *tuiApp) fieldLabel(key string) string {
	return a.t(key) + ": "
}

func (a *tuiApp) fieldLabelWithChoices(labelKey, choicesKey string) string {
	return fmt.Sprintf("%s (%s): ", a.t(labelKey), a.t(choicesKey))
}

func (a *tuiApp) fieldLabelWithChoicesAdaptive(labelKey, choicesKey string) string {
	if a.useShortFieldLabels() {
		return a.fieldLabel(labelKey)
	}
	return a.fieldLabelWithChoices(labelKey, choicesKey)
}

func (a *tuiApp) refreshFieldLabels() {
	if a.logsLevelSelect != nil {
		a.logsLevelSelect.SetLabel(a.fieldLabel("field.logs.level"))
	}
	if a.logsSourceSelect != nil {
		a.logsSourceSelect.SetLabel(a.fieldLabel("field.logs.source"))
	}
	if a.logsSearchInput != nil {
		a.logsSearchInput.SetLabel(a.t("label.search"))
	}
	if a.networkRoutingMode != nil {
		a.networkRoutingMode.SetLabel(a.fieldLabelWithChoicesAdaptive("field.network.mode", "field.choices.network.mode"))
	}
	if a.networkPresetSelect != nil {
		a.networkPresetSelect.SetLabel(a.fieldLabel("field.network.preset"))
	}
	if a.networkDomainStrategy != nil {
		a.networkDomainStrategy.SetLabel(a.fieldLabelWithChoicesAdaptive("field.network.domainStrategy", "field.choices.network.domainStrategy"))
	}
	if a.networkLocalBypass != nil {
		a.networkLocalBypass.SetLabel(a.fieldLabelWithChoicesAdaptive("field.network.localBypass", "field.choices.network.localBypass"))
	}
	if a.networkTestTarget != nil {
		a.networkTestTarget.SetLabel(a.fieldLabel("field.network.target"))
	}
	if a.networkTestPort != nil {
		a.networkTestPort.SetLabel(a.fieldLabel("field.network.port"))
	}
	if a.settingsLanguage != nil {
		a.settingsLanguage.SetLabel(a.fieldLabelWithChoicesAdaptive("field.settings.language", "field.choices.settings.language"))
	}
	if a.settingsListenAddr != nil {
		a.settingsListenAddr.SetLabel(a.fieldLabel("field.settings.listenAddr"))
	}
	if a.settingsSocksPort != nil {
		a.settingsSocksPort.SetLabel(a.fieldLabel("field.settings.socksPort"))
	}
	if a.settingsHTTPPort != nil {
		a.settingsHTTPPort.SetLabel(a.fieldLabel("field.settings.httpPort"))
	}
	if a.settingsTunName != nil {
		a.settingsTunName.SetLabel(a.fieldLabel("field.settings.tunName"))
	}
	if a.settingsTunMode != nil {
		a.settingsTunMode.SetLabel(a.fieldLabelWithChoicesAdaptive("field.settings.tunMode", "field.choices.settings.tunMode"))
	}
	if a.settingsTunMtu != nil {
		a.settingsTunMtu.SetLabel(a.fieldLabel("field.settings.tunMtu"))
	}
	if a.settingsTunAutoRoute != nil {
		a.settingsTunAutoRoute.SetLabel(a.fieldLabel("field.settings.tunAutoRoute"))
	}
	if a.settingsTunStrict != nil {
		a.settingsTunStrict.SetLabel(a.fieldLabel("field.settings.tunStrictRoute"))
	}
	if a.settingsProxyMode != nil {
		a.settingsProxyMode.SetLabel(a.fieldLabelWithChoicesAdaptive("field.settings.proxyMode", "field.choices.settings.proxyMode"))
	}
	if a.settingsLocalProxyMode != nil {
		a.settingsLocalProxyMode.SetLabel(a.fieldLabelWithChoicesAdaptive("field.settings.localProxyMode", "field.choices.settings.localProxyMode"))
	}
	if a.settingsProxyExcept != nil {
		a.settingsProxyExcept.SetLabel(a.fieldLabel("field.settings.proxyExceptions"))
	}
	if a.settingsProxyUsers != nil {
		a.settingsProxyUsers.SetLabel(a.fieldLabel("field.settings.proxyUsers"))
	}
	if a.settingsCoreEngine != nil {
		a.settingsCoreEngine.SetLabel(a.fieldLabel("field.settings.coreEngine"))
	}
	if a.settingsLogLevel != nil {
		// Keep this label short to avoid compressing the dropdown field.
		a.settingsLogLevel.SetLabel(a.fieldLabel("field.settings.logLevel"))
	}
	if a.settingsSkipCert != nil {
		a.settingsSkipCert.SetLabel(a.fieldLabel("field.settings.skipCertVerify"))
	}
	if a.settingsDNSMode != nil {
		a.settingsDNSMode.SetLabel(a.fieldLabelWithChoicesAdaptive("field.settings.dnsMode", "field.choices.settings.dnsMode"))
	}
	if a.settingsDNSList != nil {
		a.settingsDNSList.SetLabel(a.fieldLabel("field.settings.dnsList"))
	}
}

func (a *tuiApp) refreshDropdownLabels() {
	if a.logsLevelSelect != nil {
		a.logsLevelSelect.ReplaceOptions([]selectOption{{Label: a.t("dropdown.logs.level.all"), Value: "all"}, {Label: a.t("dropdown.logs.level.error"), Value: "error"}, {Label: a.t("dropdown.logs.level.warning"), Value: "warning"}, {Label: a.t("dropdown.logs.level.info"), Value: "info"}, {Label: a.t("dropdown.logs.level.debug"), Value: "debug"}})
	}
	if a.logsSourceSelect != nil {
		a.logsSourceSelect.ReplaceOptions([]selectOption{{Label: a.t("dropdown.logs.source.all"), Value: "all"}, {Label: a.t("dropdown.logs.source.app"), Value: "app"}, {Label: a.t("dropdown.logs.source.core"), Value: "xray-core"}})
	}
	if a.profileEditNetwork != nil {
		a.profileEditNetwork.ReplaceOptions(a.prependLocalizedEmptyOption([]selectOption{{Label: "TCP", Value: "tcp"}, {Label: "WebSocket", Value: "ws"}, {Label: "gRPC", Value: "grpc"}, {Label: "HTTP/2", Value: "h2"}, {Label: "KCP", Value: "kcp"}, {Label: "QUIC", Value: "quic"}, {Label: "XHTTP", Value: "xhttp"}}))
	}
	if a.profileEditTLS != nil {
		a.profileEditTLS.ReplaceOptions(a.localizedBoolSelectOptions())
	}
	if a.profileEditSkipCert != nil {
		a.profileEditSkipCert.ReplaceOptions(a.localizedBoolSelectOptions())
	}
	if a.profileEditGRPCMode != nil {
		a.profileEditGRPCMode.ReplaceOptions(a.prependLocalizedEmptyOption([]selectOption{{Label: "Gun", Value: "gun"}, {Label: "Multi", Value: "multi"}}))
	}
	if a.profileEditVMessSec != nil {
		a.profileEditVMessSec.ReplaceOptions(a.prependLocalizedEmptyOption([]selectOption{{Label: "None", Value: "none"}, {Label: "Auto", Value: "auto"}, {Label: "AES-128-GCM", Value: "aes-128-gcm"}, {Label: "ChaCha20-Poly1305", Value: "chacha20-poly1305"}}))
	}
	if a.profileEditVLESSEnc != nil {
		a.profileEditVLESSEnc.ReplaceOptions(a.prependLocalizedEmptyOption([]selectOption{{Label: "None", Value: "none"}}))
	}
	if a.profileEditHy2Insecure != nil {
		a.profileEditHy2Insecure.ReplaceOptions(a.localizedBoolSelectOptions())
	}
	if a.profileEditTuicCC != nil {
		a.profileEditTuicCC.ReplaceOptions(a.prependLocalizedEmptyOption([]selectOption{{Label: "BBR", Value: "bbr"}, {Label: "Cubic", Value: "cubic"}, {Label: "New Reno", Value: "new_reno"}}))
	}
	if a.profileEditTuicInsec != nil {
		a.profileEditTuicInsec.ReplaceOptions(a.localizedBoolSelectOptions())
	}
	if a.networkRoutingMode != nil {
		a.networkRoutingMode.ReplaceOptions([]selectOption{{Label: a.t("dropdown.routing.mode.global"), Value: "global"}, {Label: a.t("dropdown.routing.mode.bypassCN"), Value: "bypass_cn"}, {Label: a.t("dropdown.routing.mode.direct"), Value: "direct"}, {Label: a.t("dropdown.routing.mode.custom"), Value: "custom"}})
	}
	if a.networkPresetSelect != nil {
		a.networkPresetSelect.ReplaceOptions([]selectOption{{Label: a.t("dropdown.routing.preset.empty"), Value: ""}, {Label: a.t("dropdown.routing.preset.global"), Value: "global"}, {Label: a.t("dropdown.routing.preset.bypassCN"), Value: "bypass_cn"}, {Label: a.t("dropdown.routing.preset.direct"), Value: "direct"}})
	}
	if a.networkDomainStrategy != nil {
		a.networkDomainStrategy.ReplaceOptions([]selectOption{{Label: a.t("dropdown.routing.strategy.ifNonMatch"), Value: "IPIfNonMatch"}, {Label: a.t("dropdown.routing.strategy.onDemand"), Value: "IPOnDemand"}, {Label: a.t("dropdown.routing.strategy.asIs"), Value: "AsIs"}})
	}
	if a.networkLocalBypass != nil {
		a.networkLocalBypass.ReplaceOptions(a.localizedBoolSelectOptions())
	}
	if a.settingsLanguage != nil {
		a.settingsLanguage.ReplaceOptions([]selectOption{{Label: a.t("dropdown.language.english"), Value: uiLangEN}, {Label: a.t("dropdown.language.chinese"), Value: uiLangZH}})
	}
	if a.settingsTunMode != nil {
		a.settingsTunMode.ReplaceOptions([]selectOption{{Label: a.t("dropdown.tun.off"), Value: "off"}, {Label: a.t("dropdown.tun.system"), Value: "system"}, {Label: a.t("dropdown.tun.mixed"), Value: "mixed"}, {Label: a.t("dropdown.tun.gvisor"), Value: "gvisor"}})
	}
	if a.settingsTunAutoRoute != nil {
		a.settingsTunAutoRoute.ReplaceOptions(a.localizedBoolSelectOptions())
	}
	if a.settingsTunStrict != nil {
		a.settingsTunStrict.ReplaceOptions(a.localizedBoolSelectOptions())
	}
	if a.settingsProxyMode != nil {
		a.settingsProxyMode.ReplaceOptions([]selectOption{{Label: a.t("dropdown.proxy.forceEnable"), Value: "forced_change"}, {Label: a.t("dropdown.proxy.forceDisable"), Value: "forced_clear"}, {Label: a.t("dropdown.proxy.pac"), Value: "pac"}})
	}
	if a.settingsLocalProxyMode != nil {
		a.settingsLocalProxyMode.ReplaceOptions([]selectOption{{Label: a.t("dropdown.proxyTraffic.followRouting"), Value: "follow-routing"}, {Label: a.t("dropdown.proxyTraffic.forceProxy"), Value: "force-proxy"}})
	}
	if a.settingsCoreEngine != nil {
		a.settingsCoreEngine.ReplaceOptions([]selectOption{{Label: a.t("dropdown.engine.xray"), Value: "xray-core"}})
	}
	if a.settingsLogLevel != nil {
		a.settingsLogLevel.ReplaceOptions([]selectOption{{Label: a.t("dropdown.log.debug"), Value: "debug"}, {Label: a.t("dropdown.log.info"), Value: "info"}, {Label: a.t("dropdown.log.warning"), Value: "warning"}, {Label: a.t("dropdown.log.error"), Value: "error"}})
	}
	if a.settingsSkipCert != nil {
		a.settingsSkipCert.ReplaceOptions(a.localizedBoolSelectOptions())
	}
	if a.settingsDNSMode != nil {
		a.settingsDNSMode.ReplaceOptions([]selectOption{{Label: a.t("dropdown.dns.system"), Value: "UseSystemDNS"}, {Label: a.t("dropdown.dns.list"), Value: "UseDNSList"}, {Label: a.t("dropdown.dns.direct"), Value: "Direct"}})
	}
}

func (a *tuiApp) syncSidebar() {
	if a.sidebar == nil {
		return
	}
	items := make([]components.NavItem, 0, len(tuiPageTabs()))
	for _, tab := range tuiPageTabs() {
		items = append(items, components.NavItem{
			Key:      tab.key,
			Label:    pageDisplayName(tab.key),
			Shortcut: tab.shortcut,
		})
	}
	a.sidebar.SetItems(items)
	a.sidebar.SetSelectedKey(a.page)
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

	// 同步侧边栏
	a.syncSidebar()

	prevFocusGroup := a.focusGroup
	a.focusables = append([]tview.Primitive{}, page.focusables...)
	a.focusGroups = nil
	for _, group := range page.focusGroups {
		if len(group) == 0 {
			continue
		}
		a.focusGroups = append(a.focusGroups, group)
	}
	// 只有当侧边栏可见时才添加其焦点元素
	if a.sidebar != nil && a.sidebar.IsVisible() {
		sidebarFocusables := a.sidebar.GetFocusables()
		if len(sidebarFocusables) > 0 {
			a.focusables = append(a.focusables, sidebarFocusables...)
			a.focusGroups = append(a.focusGroups, sidebarFocusables)
		}
	}
	if len(a.focusGroups) == 0 && len(a.focusables) > 0 {
		a.focusGroups = [][]tview.Primitive{a.focusables}
	}
	if len(a.focusGroups) == 0 {
		a.focusGroup = -1
	} else {
		if prevFocusGroup < 0 {
			prevFocusGroup = 0
		}
		if prevFocusGroup >= len(a.focusGroups) {
			prevFocusGroup = len(a.focusGroups) - 1
		}
		a.focusGroup = prevFocusGroup
	}
	a.pageHolder.RemovePage("current")
	a.pageHolder.AddAndSwitchToPage("current", page.root, true)
	if a.app != nil {
		if len(a.focusGroups) > 0 && a.focusGroup >= 0 && a.focusGroup < len(a.focusGroups) && len(a.focusGroups[a.focusGroup]) > 0 {
			a.app.SetFocus(a.focusGroups[a.focusGroup][0])
		} else if len(a.focusables) > 0 {
			a.app.SetFocus(a.focusables[0])
		}
		a.refreshFooter()
		a.refreshHelpBar()
	}
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
	subscriptions := a.subscriptionsSnapshot()
	if index < 0 || index >= len(subscriptions) {
		return
	}
	selected := subscriptions[index]
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
