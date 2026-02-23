package cmd

import (
	"context"
	"fmt"
	"sort"
	"strings"

	"github.com/charmbracelet/bubbles/list"
	"github.com/charmbracelet/bubbles/spinner"
	"github.com/charmbracelet/bubbles/viewport"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/tayloree/publix-deals/internal/api"
	"github.com/tayloree/publix-deals/internal/filter"
)

const (
	minTUIWidth  = 92
	minTUIHeight = 24
)

var (
	tuiHeaderStyle  = lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("86"))
	tuiMetaStyle    = lipgloss.NewStyle().Foreground(lipgloss.Color("245"))
	tuiHintStyle    = lipgloss.NewStyle().Foreground(lipgloss.Color("241"))
	tuiValueStyle   = lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("229"))
	tuiBogoStyle    = lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("205"))
	tuiDealStyle    = lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("229"))
	tuiMutedStyle   = lipgloss.NewStyle().Foreground(lipgloss.Color("244"))
	tuiSectionStyle = lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("81"))
)

type tuiLoadConfig struct {
	ctx         context.Context
	storeNumber string
	zipCode     string
	initialOpts filter.Options
}

type tuiDataLoadedMsg struct {
	storeLabel  string
	allDeals    []api.SavingItem
	initialOpts filter.Options
}

type tuiDataLoadErrMsg struct {
	err error
}

type tuiFocus int

const (
	tuiFocusList tuiFocus = iota
	tuiFocusDetail
)

type tuiGroupItem struct {
	name    string
	count   int
	ordinal int
}

func (g tuiGroupItem) FilterValue() string { return strings.ToLower(g.name) }
func (g tuiGroupItem) Title() string       { return fmt.Sprintf("%d. %s", g.ordinal, g.name) }
func (g tuiGroupItem) Description() string {
	return fmt.Sprintf("Section header • %d deals", g.count)
}

type tuiDealItem struct {
	deal        api.SavingItem
	group       string
	title       string
	description string
	filterValue string
}

func (d tuiDealItem) FilterValue() string { return d.filterValue }
func (d tuiDealItem) Title() string       { return d.title }
func (d tuiDealItem) Description() string { return d.description }

type dealsTUIModel struct {
	loading  bool
	spinner  spinner.Model
	loadCmd  tea.Cmd
	fatalErr error

	storeLabel string
	allDeals   []api.SavingItem

	opts        filter.Options
	initialOpts filter.Options

	sortChoices       []string
	sortIndex         int
	categoryChoices   []string
	categoryIndex     int
	departmentChoices []string
	departmentIndex   int
	limitChoices      []int
	limitIndex        int

	list   list.Model
	detail viewport.Model

	focus      tuiFocus
	showHelp   bool
	selectedID string

	groupStarts  []int
	visibleDeals int

	width, height   int
	bodyHeight      int
	listPaneWidth   int
	detailPaneWidth int
	tooSmall        bool
}

func newLoadingDealsTUIModel(cfg tuiLoadConfig) dealsTUIModel {
	delegate := list.NewDefaultDelegate()
	delegate.SetHeight(2)
	delegate.SetSpacing(1)

	lst := list.New([]list.Item{}, delegate, 0, 0)
	lst.Title = "Deals"
	lst.SetStatusBarItemName("item", "items")
	lst.SetShowStatusBar(true)
	lst.SetFilteringEnabled(true)
	lst.SetShowHelp(false)
	lst.SetShowPagination(true)
	lst.DisableQuitKeybindings()

	detail := viewport.New(0, 0)
	detail.KeyMap.PageDown.SetKeys("f", "pgdown")
	detail.KeyMap.PageUp.SetKeys("b", "pgup")
	detail.KeyMap.HalfPageDown.SetKeys("d")
	detail.KeyMap.HalfPageUp.SetKeys("u")

	spin := spinner.New()
	spin.Spinner = spinner.Dot
	spin.Style = lipgloss.NewStyle().Foreground(lipgloss.Color("86"))

	return dealsTUIModel{
		loading:     true,
		spinner:     spin,
		loadCmd:     loadTUIDataCmd(cfg),
		initialOpts: cfg.initialOpts,
		opts:        cfg.initialOpts,
		list:        lst,
		detail:      detail,
		focus:       tuiFocusList,
	}
}

func loadTUIDataCmd(cfg tuiLoadConfig) tea.Cmd {
	return func() tea.Msg {
		_, storeLabel, allDeals, err := loadTUIData(cfg.ctx, cfg.storeNumber, cfg.zipCode)
		if err != nil {
			return tuiDataLoadErrMsg{err: err}
		}
		return tuiDataLoadedMsg{
			storeLabel:  storeLabel,
			allDeals:    allDeals,
			initialOpts: cfg.initialOpts,
		}
	}
}

func (m dealsTUIModel) Init() tea.Cmd {
	return tea.Batch(m.spinner.Tick, m.loadCmd)
}

func (m dealsTUIModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		m.resize()
		return m, nil

	case tuiDataLoadedMsg:
		m.loading = false
		m.storeLabel = msg.storeLabel
		m.allDeals = msg.allDeals
		m.initialOpts = canonicalizeTUIOptions(msg.initialOpts)
		m.opts = m.initialOpts
		m.initializeInlineChoices()
		m.applyCurrentFilters(true)
		m.resize()
		return m, nil

	case tuiDataLoadErrMsg:
		m.loading = false
		m.fatalErr = msg.err
		return m, tea.Quit

	case spinner.TickMsg:
		if m.loading {
			var cmd tea.Cmd
			m.spinner, cmd = m.spinner.Update(msg)
			return m, cmd
		}
	}

	keyMsg, isKey := msg.(tea.KeyMsg)
	if isKey {
		if keyMsg.String() == "ctrl+c" {
			return m, tea.Quit
		}
		if m.loading {
			if keyMsg.String() == "q" {
				return m, tea.Quit
			}
			return m, nil
		}
	}

	if m.loading {
		return m, nil
	}

	if isKey {
		filtering := m.list.FilterState() == list.Filtering
		key := keyMsg.String()

		switch key {
		case "q":
			if !filtering {
				return m, tea.Quit
			}
		case "tab":
			if !filtering {
				if m.focus == tuiFocusList {
					m.focus = tuiFocusDetail
				} else {
					m.focus = tuiFocusList
				}
				return m, nil
			}
		case "esc":
			if m.focus == tuiFocusDetail && !filtering {
				m.focus = tuiFocusList
				return m, nil
			}
		case "?":
			if !filtering {
				m.showHelp = !m.showHelp
				m.resize()
				return m, nil
			}
		case "s":
			if !filtering {
				m.cycleSortMode()
				return m, nil
			}
		case "g":
			if !filtering {
				m.opts.BOGO = !m.opts.BOGO
				m.applyCurrentFilters(false)
				return m, nil
			}
		case "c":
			if !filtering {
				m.cycleCategory()
				return m, nil
			}
		case "a":
			if !filtering {
				m.cycleDepartment()
				return m, nil
			}
		case "l":
			if !filtering {
				m.cycleLimit()
				return m, nil
			}
		case "r":
			if !filtering {
				m.opts = m.initialOpts
				m.syncChoiceIndexesFromOptions()
				m.applyCurrentFilters(false)
				return m, nil
			}
		case "]":
			if !filtering {
				if m.list.IsFiltered() {
					return m, m.list.NewStatusMessage("Clear fuzzy filter before section jumps.")
				}
				m.jumpSection(1)
				return m, nil
			}
		case "[":
			if !filtering {
				if m.list.IsFiltered() {
					return m, m.list.NewStatusMessage("Clear fuzzy filter before section jumps.")
				}
				m.jumpSection(-1)
				return m, nil
			}
		}

		if !filtering && len(key) == 1 && key[0] >= '1' && key[0] <= '9' {
			if m.list.IsFiltered() {
				return m, m.list.NewStatusMessage("Clear fuzzy filter before section jumps.")
			}
			m.jumpToSection(int(key[0] - '1'))
			return m, nil
		}

		if m.focus == tuiFocusDetail && !filtering {
			var cmd tea.Cmd
			m.detail, cmd = m.detail.Update(msg)
			return m, cmd
		}
	}

	var cmd tea.Cmd
	m.list, cmd = m.list.Update(msg)
	m.refreshDetail(false)
	return m, cmd
}

func (m dealsTUIModel) View() string {
	if m.loading {
		return m.loadingView()
	}
	if m.width == 0 || m.height == 0 {
		return tuiMetaStyle.Render("Loading interface...")
	}
	if m.tooSmall {
		return lipgloss.NewStyle().
			Padding(1, 2).
			Render(
				fmt.Sprintf(
					"Terminal too small (%dx%d).\nResize to at least %dx%d for the two-pane deal explorer.",
					m.width, m.height, minTUIWidth, minTUIHeight,
				),
			)
	}

	return lipgloss.JoinVertical(
		lipgloss.Left,
		m.headerView(),
		m.bodyView(),
		m.footerView(),
	)
}

func (m dealsTUIModel) loadingView() string {
	width := m.width
	if width == 0 {
		width = 80
	}
	skeletonStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color("240"))

	lines := []string{
		tuiHeaderStyle.Render("pubcli tui"),
		tuiMetaStyle.Render("Preparing interactive interface..."),
		"",
		fmt.Sprintf("%s Fetching store and weekly deals", m.spinner.View()),
		tuiHintStyle.Render("Tip: press q to cancel."),
		"",
		skeletonStyle.Render("┌──────────────────────────────┬─────────────────────────────────────────┐"),
		skeletonStyle.Render("│  Loading deal list...        │  Loading detail panel...               │"),
		skeletonStyle.Render("│  • categories                │  • pricing and validity metadata       │"),
		skeletonStyle.Render("│  • sections                  │  • wrapped description text            │"),
		skeletonStyle.Render("│  • filter index              │  • scroll viewport                     │"),
		skeletonStyle.Render("└──────────────────────────────┴─────────────────────────────────────────┘"),
	}

	return lipgloss.NewStyle().
		Width(width).
		Padding(1, 2).
		Render(strings.Join(lines, "\n"))
}

func (m *dealsTUIModel) resize() {
	if m.width == 0 || m.height == 0 {
		return
	}
	if m.loading {
		return
	}

	m.tooSmall = m.width < minTUIWidth || m.height < minTUIHeight
	if m.tooSmall {
		return
	}

	headerH := 3
	footerH := 2
	if m.showHelp {
		footerH = 7
	}
	m.bodyHeight = maxInt(8, m.height-headerH-footerH-1)

	listWidth := maxInt(40, int(float64(m.width)*0.43))
	if listWidth > m.width-42 {
		listWidth = m.width / 2
	}
	detailWidth := m.width - listWidth - 1
	if detailWidth < 36 {
		detailWidth = 36
		listWidth = m.width - detailWidth - 1
	}

	m.listPaneWidth = listWidth
	m.detailPaneWidth = detailWidth

	listInnerWidth := maxInt(24, listWidth-4)
	detailInnerWidth := maxInt(24, detailWidth-4)
	panelInnerHeight := maxInt(6, m.bodyHeight-2)

	m.list.SetSize(listInnerWidth, panelInnerHeight)
	m.detail.Width = detailInnerWidth
	m.detail.Height = panelInnerHeight
	m.refreshDetail(false)
}

func (m dealsTUIModel) headerView() string {
	focus := "list"
	if m.focus == tuiFocusDetail {
		focus = "detail"
	}

	top := fmt.Sprintf("pubcli tui  |  %s", m.storeLabel)
	bottom := fmt.Sprintf(
		"deals: %d visible / %d total  |  filters: %s  |  focus: %s",
		m.visibleDeals, len(m.allDeals), m.activeFilterSummary(), focus,
	)

	return lipgloss.NewStyle().
		Width(m.width).
		Padding(0, 1).
		Render(tuiHeaderStyle.Render(top) + "\n" + tuiMetaStyle.Render(bottom))
}

func (m dealsTUIModel) bodyView() string {
	listBorder := lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(lipgloss.Color("241")).
		Padding(0, 1)
	detailBorder := listBorder

	if m.focus == tuiFocusList {
		listBorder = listBorder.BorderForeground(lipgloss.Color("86"))
	} else {
		detailBorder = detailBorder.BorderForeground(lipgloss.Color("86"))
	}

	left := listBorder.
		Width(m.listPaneWidth).
		Height(m.bodyHeight).
		Render(m.list.View())
	right := detailBorder.
		Width(m.detailPaneWidth).
		Height(m.bodyHeight).
		Render(m.detail.View())

	return lipgloss.JoinHorizontal(lipgloss.Top, left, " ", right)
}

func (m dealsTUIModel) footerView() string {
	base := "Tab switch pane • / fuzzy filter • s sort • g bogo • c category • a department • l limit • r reset • [/] section jump • 1-9 section index • q quit"
	if m.focus == tuiFocusDetail {
		base = "Detail: j/k or ↑/↓ scroll • u/d half-page • b/f page • esc list • ? help • q quit"
	}

	if !m.showHelp {
		return lipgloss.NewStyle().Padding(0, 1).Render(tuiHintStyle.Render(base))
	}

	lines := []string{
		"Key Help",
		"list pane: ↑/↓ or j/k move • / fuzzy filter • c category • a department • g bogo • s sort • l limit",
		"group jumps: ] next section • [ previous section • 1..9 jump to numbered section header",
		"detail pane: j/k or ↑/↓ scroll • u/d half-page • b/f page up/down",
		"global: tab switch pane • esc list • r reset inline options • ? toggle help • q quit • ctrl+c force quit",
	}
	return lipgloss.NewStyle().
		Padding(0, 1).
		Render(tuiHintStyle.Render(strings.Join(lines, "\n")))
}

func (m *dealsTUIModel) initializeInlineChoices() {
	m.opts = canonicalizeTUIOptions(m.opts)

	m.sortChoices = []string{"", "savings", "ending"}
	m.categoryChoices = buildCategoryChoices(m.allDeals, m.opts.Category)
	m.departmentChoices = buildDepartmentChoices(m.allDeals, m.opts.Department)
	m.limitChoices = buildLimitChoices(m.opts.Limit)

	m.syncChoiceIndexesFromOptions()
}

func (m *dealsTUIModel) syncChoiceIndexesFromOptions() {
	m.sortIndex = indexOfString(m.sortChoices, canonicalSortMode(m.opts.Sort))
	if m.sortIndex < 0 {
		m.sortIndex = 0
	}
	m.opts.Sort = m.sortChoices[m.sortIndex]

	m.categoryIndex = indexOfStringFold(m.categoryChoices, m.opts.Category)
	if m.categoryIndex < 0 {
		m.categoryIndex = 0
		m.opts.Category = ""
	} else {
		m.opts.Category = m.categoryChoices[m.categoryIndex]
	}

	m.departmentIndex = indexOfStringFold(m.departmentChoices, m.opts.Department)
	if m.departmentIndex < 0 {
		m.departmentIndex = 0
		m.opts.Department = ""
	} else {
		m.opts.Department = m.departmentChoices[m.departmentIndex]
	}

	m.limitIndex = indexOfInt(m.limitChoices, m.opts.Limit)
	if m.limitIndex < 0 {
		m.limitIndex = 0
		m.opts.Limit = m.limitChoices[m.limitIndex]
	}
}

func (m *dealsTUIModel) cycleSortMode() {
	if len(m.sortChoices) == 0 {
		return
	}
	m.sortIndex = (m.sortIndex + 1) % len(m.sortChoices)
	m.opts.Sort = m.sortChoices[m.sortIndex]
	m.applyCurrentFilters(false)
}

func (m *dealsTUIModel) cycleCategory() {
	if len(m.categoryChoices) == 0 {
		return
	}
	m.categoryIndex = (m.categoryIndex + 1) % len(m.categoryChoices)
	m.opts.Category = m.categoryChoices[m.categoryIndex]
	m.applyCurrentFilters(false)
}

func (m *dealsTUIModel) cycleDepartment() {
	if len(m.departmentChoices) == 0 {
		return
	}
	m.departmentIndex = (m.departmentIndex + 1) % len(m.departmentChoices)
	m.opts.Department = m.departmentChoices[m.departmentIndex]
	m.applyCurrentFilters(false)
}

func (m *dealsTUIModel) cycleLimit() {
	if len(m.limitChoices) == 0 {
		return
	}
	m.limitIndex = (m.limitIndex + 1) % len(m.limitChoices)
	m.opts.Limit = m.limitChoices[m.limitIndex]
	m.applyCurrentFilters(false)
}

func (m dealsTUIModel) activeFilterSummary() string {
	parts := []string{}
	if m.opts.BOGO {
		parts = append(parts, "bogo")
	}
	if m.opts.Category != "" {
		parts = append(parts, "category:"+m.opts.Category)
	}
	if m.opts.Department != "" {
		parts = append(parts, "department:"+m.opts.Department)
	}
	if m.opts.Query != "" {
		parts = append(parts, "query:"+m.opts.Query)
	}
	if m.opts.Sort != "" {
		parts = append(parts, "sort:"+m.opts.Sort)
	}
	if m.opts.Limit > 0 {
		parts = append(parts, fmt.Sprintf("limit:%d", m.opts.Limit))
	}
	if fuzzy := strings.TrimSpace(m.list.FilterValue()); fuzzy != "" {
		parts = append(parts, "fuzzy:"+fuzzy)
	}
	if len(parts) == 0 {
		return "none"
	}
	return strings.Join(parts, ", ")
}

func (m *dealsTUIModel) applyCurrentFilters(resetSelection bool) {
	currentID := m.selectedID
	filtered := filter.Apply(m.allDeals, m.opts)
	m.visibleDeals = len(filtered)

	items, starts := buildGroupedListItems(filtered)
	m.groupStarts = starts

	m.list.Title = fmt.Sprintf("Deals • %d visible", m.visibleDeals)
	m.list.SetItems(items)

	target := -1
	if !resetSelection && currentID != "" {
		target = findItemIndexByID(items, currentID)
	}
	if target < 0 {
		target = firstDealItemIndex(items)
	}
	if target < 0 && len(items) > 0 {
		target = 0
	}
	if target >= 0 {
		m.list.Select(target)
	}

	m.refreshDetail(true)
}

func (m *dealsTUIModel) refreshDetail(resetScroll bool) {
	var content string
	nextID := ""

	if selected := m.list.SelectedItem(); selected != nil {
		switch item := selected.(type) {
		case tuiDealItem:
			content = renderDealDetailContent(item.deal, m.detail.Width)
			nextID = stableIDForDeal(item.deal, item.title)
		case tuiGroupItem:
			content = m.renderGroupDetail(item)
			nextID = stableIDForGroup(item.name)
		}
	}
	if content == "" {
		content = "No deals match the current inline filters.\n\nTry pressing r to reset filters."
	}

	if resetScroll || nextID != m.selectedID {
		m.detail.GotoTop()
	}
	m.selectedID = nextID
	m.detail.SetContent(content)
}

func (m dealsTUIModel) renderGroupDetail(group tuiGroupItem) string {
	preview := m.groupPreviewTitles(group.name, 5)

	lines := []string{
		tuiSectionStyle.Render(fmt.Sprintf("Section %d: %s", group.ordinal, group.name)),
		tuiMetaStyle.Render(fmt.Sprintf("%d deals in this section", group.count)),
		"",
		tuiMetaStyle.Render("Jump keys:"),
		"- `]` next section, `[` previous section",
		"- `1..9` jump directly to section number",
	}
	if len(preview) > 0 {
		lines = append(lines, "")
		lines = append(lines, tuiMetaStyle.Render("Preview:"))
		for _, title := range preview {
			lines = append(lines, "• "+title)
		}
	}

	return strings.Join(lines, "\n")
}

func (m dealsTUIModel) groupPreviewTitles(group string, max int) []string {
	out := make([]string, 0, max)
	for _, item := range m.list.Items() {
		deal, ok := item.(tuiDealItem)
		if !ok || deal.group != group {
			continue
		}
		out = append(out, deal.title)
		if len(out) >= max {
			break
		}
	}
	return out
}

func (m *dealsTUIModel) jumpToSection(index int) {
	if index < 0 || index >= len(m.groupStarts) {
		return
	}

	target := firstDealIndexFrom(m.list.Items(), m.groupStarts[index])
	if target < 0 {
		target = m.groupStarts[index]
	}
	m.list.Select(target)
	m.refreshDetail(true)
}

func (m *dealsTUIModel) jumpSection(delta int) {
	if len(m.groupStarts) == 0 {
		return
	}

	current := m.currentSectionIndex()
	if current < 0 {
		current = 0
	}
	next := current + delta
	if next < 0 {
		next = len(m.groupStarts) - 1
	}
	if next >= len(m.groupStarts) {
		next = 0
	}
	m.jumpToSection(next)
}

func (m dealsTUIModel) currentSectionIndex() int {
	if len(m.groupStarts) == 0 {
		return -1
	}
	cursor := m.list.GlobalIndex()
	current := 0
	for i, start := range m.groupStarts {
		if start <= cursor {
			current = i
			continue
		}
		break
	}
	return current
}

func buildGroupedListItems(deals []api.SavingItem) (items []list.Item, starts []int) {
	if len(deals) == 0 {
		return nil, nil
	}

	groups := map[string][]api.SavingItem{}
	for _, deal := range deals {
		group := dealGroupLabel(deal)
		groups[group] = append(groups[group], deal)
	}

	type groupMeta struct {
		name  string
		count int
	}

	metas := make([]groupMeta, 0, len(groups))
	for name, deals := range groups {
		metas = append(metas, groupMeta{name: name, count: len(deals)})
	}
	sort.Slice(metas, func(i, j int) bool {
		if metas[i].name == "BOGO" && metas[j].name != "BOGO" {
			return true
		}
		if metas[j].name == "BOGO" && metas[i].name != "BOGO" {
			return false
		}
		if metas[i].count != metas[j].count {
			return metas[i].count > metas[j].count
		}
		return metas[i].name < metas[j].name
	})

	items = make([]list.Item, 0, len(deals)+len(metas))
	starts = make([]int, 0, len(metas))
	for idx, meta := range metas {
		starts = append(starts, len(items))

		items = append(items, tuiGroupItem{
			name:    meta.name,
			count:   meta.count,
			ordinal: idx + 1,
		})
		for _, deal := range groups[meta.name] {
			items = append(items, buildTUIDealItem(deal, meta.name))
		}
	}

	return items, starts
}

func dealGroupLabel(item api.SavingItem) string {
	if filter.ContainsIgnoreCase(item.Categories, "bogo") {
		return "BOGO"
	}
	for _, category := range item.Categories {
		clean := strings.TrimSpace(category)
		if clean == "" || strings.EqualFold(clean, "bogo") {
			continue
		}
		return humanizeLabel(clean)
	}
	if dept := strings.TrimSpace(filter.CleanText(filter.Deref(item.Department))); dept != "" {
		return humanizeLabel(dept)
	}
	return "Other"
}

func buildTUIDealItem(item api.SavingItem, group string) tuiDealItem {
	title := topDealTitle(item)
	savings := filter.CleanText(filter.Deref(item.Savings))
	if savings == "" {
		savings = "No savings text"
	}
	dept := filter.CleanText(filter.Deref(item.Department))
	end := strings.TrimSpace(item.EndFormatted)

	descParts := []string{savings}
	if dept != "" {
		descParts = append(descParts, dept)
	}
	if end != "" {
		descParts = append(descParts, "ends "+end)
	}

	filterTokens := []string{
		title,
		savings,
		filter.CleanText(filter.Deref(item.Description)),
		filter.CleanText(filter.Deref(item.Brand)),
		strings.Join(item.Categories, " "),
		dept,
		end,
		group,
	}

	return tuiDealItem{
		deal:        item,
		group:       group,
		title:       title,
		description: strings.Join(descParts, "  •  "),
		filterValue: strings.ToLower(strings.Join(filterTokens, " ")),
	}
}

func renderDealDetailContent(item api.SavingItem, width int) string {
	maxWidth := maxInt(24, width)

	title := topDealTitle(item)
	savings := filter.CleanText(filter.Deref(item.Savings))
	if savings == "" {
		savings = "No savings value provided"
	}

	desc := filter.CleanText(filter.Deref(item.Description))
	if desc == "" {
		desc = "No description provided."
	}

	dept := filter.CleanText(filter.Deref(item.Department))
	brand := filter.CleanText(filter.Deref(item.Brand))
	dealInfo := filter.CleanText(filter.Deref(item.AdditionalDealInfo))
	validity := strings.TrimSpace(item.StartFormatted + " - " + item.EndFormatted)
	imageURL := strings.TrimSpace(filter.Deref(item.ImageURL))

	lines := []string{
		tuiDealStyle.Render(wrapText(title, maxWidth)),
	}

	metaBits := []string{}
	if filter.ContainsIgnoreCase(item.Categories, "bogo") {
		metaBits = append(metaBits, tuiBogoStyle.Render("BOGO"))
	}
	if len(item.Categories) > 0 {
		metaBits = append(metaBits, "categories: "+strings.Join(item.Categories, ", "))
	}
	if len(metaBits) > 0 {
		lines = append(lines, tuiMetaStyle.Render(wrapText(strings.Join(metaBits, "  |  "), maxWidth)))
	}

	lines = append(lines, "")
	lines = append(lines, fmt.Sprintf("%s %s", tuiMetaStyle.Render("Savings:"), tuiValueStyle.Render(savings)))
	if dealInfo != "" {
		lines = append(lines, fmt.Sprintf("%s %s", tuiMetaStyle.Render("Deal info:"), wrapText(dealInfo, maxWidth)))
	}
	lines = append(lines, "")
	lines = append(lines, tuiMetaStyle.Render("Description:"))
	lines = append(lines, wrapText(desc, maxWidth))
	lines = append(lines, "")

	if dept != "" {
		lines = append(lines, fmt.Sprintf("%s %s", tuiMetaStyle.Render("Department:"), dept))
	}
	if brand != "" {
		lines = append(lines, fmt.Sprintf("%s %s", tuiMetaStyle.Render("Brand:"), brand))
	}
	if strings.Trim(validity, " -") != "" {
		lines = append(lines, fmt.Sprintf("%s %s", tuiMetaStyle.Render("Valid:"), strings.Trim(validity, " -")))
	}
	lines = append(lines, fmt.Sprintf("%s %.2f", tuiMetaStyle.Render("Score:"), filter.DealScore(item)))

	if imageURL != "" {
		lines = append(lines, "")
		lines = append(lines, tuiMutedStyle.Render("Image URL:"))
		lines = append(lines, tuiMutedStyle.Render(wrapText(imageURL, maxWidth)))
	}

	return strings.Join(lines, "\n")
}

func wrapText(text string, width int) string {
	words := strings.Fields(text)
	if len(words) == 0 {
		return ""
	}
	if width < 12 {
		width = 12
	}

	line := words[0]
	lines := make([]string, 0, len(words)/6+1)
	for _, w := range words[1:] {
		if len(line)+1+len(w) > width {
			lines = append(lines, line)
			line = w
			continue
		}
		line += " " + w
	}
	lines = append(lines, line)
	return strings.Join(lines, "\n")
}

func canonicalizeTUIOptions(opts filter.Options) filter.Options {
	opts.Sort = canonicalSortMode(opts.Sort)
	if opts.Category != "" {
		opts.Category = strings.TrimSpace(opts.Category)
	}
	if opts.Department != "" {
		opts.Department = strings.TrimSpace(opts.Department)
	}
	if opts.Query != "" {
		opts.Query = strings.TrimSpace(opts.Query)
	}
	return opts
}

func canonicalSortMode(raw string) string {
	switch strings.ToLower(strings.TrimSpace(raw)) {
	case "savings":
		return "savings"
	case "ending", "end", "expiry", "expiration":
		return "ending"
	default:
		return ""
	}
}

func buildCategoryChoices(items []api.SavingItem, current string) []string {
	type bucket struct {
		label string
		count int
	}
	counts := map[string]bucket{}
	for _, item := range items {
		for _, category := range item.Categories {
			clean := strings.ToLower(strings.TrimSpace(category))
			if clean == "" {
				continue
			}
			entry := counts[clean]
			entry.label = clean
			entry.count++
			counts[clean] = entry
		}
	}

	values := make([]string, 0, len(counts))
	for _, value := range counts {
		values = append(values, value.label)
	}
	if current != "" && indexOfStringFold(values, current) < 0 {
		values = append(values, current)
	}
	sort.Strings(values)
	sort.SliceStable(values, func(i, j int) bool {
		left := counts[strings.ToLower(values[i])].count
		right := counts[strings.ToLower(values[j])].count
		if left != right {
			return left > right
		}
		return strings.ToLower(values[i]) < strings.ToLower(values[j])
	})
	return append([]string{""}, values...)
}

func buildDepartmentChoices(items []api.SavingItem, current string) []string {
	type bucket struct {
		label string
		count int
	}
	counts := map[string]bucket{}
	for _, item := range items {
		dept := strings.ToLower(strings.TrimSpace(filter.CleanText(filter.Deref(item.Department))))
		if dept == "" {
			continue
		}
		entry := counts[dept]
		entry.label = dept
		entry.count++
		counts[dept] = entry
	}

	values := make([]string, 0, len(counts))
	for _, value := range counts {
		values = append(values, value.label)
	}
	if current != "" && indexOfStringFold(values, current) < 0 {
		values = append(values, current)
	}
	sort.Strings(values)
	sort.SliceStable(values, func(i, j int) bool {
		left := counts[strings.ToLower(values[i])].count
		right := counts[strings.ToLower(values[j])].count
		if left != right {
			return left > right
		}
		return strings.ToLower(values[i]) < strings.ToLower(values[j])
	})
	return append([]string{""}, values...)
}

func buildLimitChoices(current int) []int {
	values := []int{0, 10, 25, 50, 100}
	if current > 0 && indexOfInt(values, current) < 0 {
		values = append(values, current)
		sort.Ints(values)
	}
	return values
}

func indexOfString(values []string, target string) int {
	for i, value := range values {
		if value == target {
			return i
		}
	}
	return -1
}

func indexOfStringFold(values []string, target string) int {
	for i, value := range values {
		if strings.EqualFold(value, target) {
			return i
		}
	}
	return -1
}

func indexOfInt(values []int, target int) int {
	for i, value := range values {
		if value == target {
			return i
		}
	}
	return -1
}

func findItemIndexByID(items []list.Item, stableID string) int {
	for i, item := range items {
		if stableIDForItem(item) == stableID {
			return i
		}
	}
	return -1
}

func firstDealItemIndex(items []list.Item) int {
	return firstDealIndexFrom(items, 0)
}

func firstDealIndexFrom(items []list.Item, start int) int {
	for i := start; i < len(items); i++ {
		if _, ok := items[i].(tuiDealItem); ok {
			return i
		}
	}
	return -1
}

func stableIDForItem(item list.Item) string {
	switch value := item.(type) {
	case tuiDealItem:
		return stableIDForDeal(value.deal, value.title)
	case tuiGroupItem:
		return stableIDForGroup(value.name)
	default:
		return ""
	}
}

func stableIDForDeal(item api.SavingItem, fallbackTitle string) string {
	if id := strings.TrimSpace(item.ID); id != "" {
		return "deal:" + id
	}
	if fallbackTitle != "" {
		return "deal:title:" + strings.ToLower(strings.TrimSpace(fallbackTitle))
	}
	return "deal:unknown"
}

func stableIDForGroup(group string) string {
	return "group:" + strings.ToLower(strings.TrimSpace(group))
}

func humanizeLabel(raw string) string {
	s := strings.TrimSpace(raw)
	if s == "" {
		return "Other"
	}
	s = strings.ReplaceAll(s, "_", " ")
	s = strings.ReplaceAll(s, "-", " ")
	words := strings.Fields(strings.ToLower(s))
	for i, word := range words {
		if len(word) == 0 {
			continue
		}
		words[i] = strings.ToUpper(word[:1]) + word[1:]
	}
	return strings.Join(words, " ")
}

func maxInt(a, b int) int {
	if a > b {
		return a
	}
	return b
}
