package fdump

import (
	"bytes"
	"encoding/gob"
	"fmt"
	"io/ioutil"
	"net"
	"os"
	"sort"
	"strings"
	"sync/atomic"
	"time"

	"github.com/gdamore/tcell"
	runewidth "github.com/mattn/go-runewidth"
	"github.com/rivo/tview"
)

// BriefFunc every decoded record will call this function get the strings to
// show in the brief view.
type BriefFunc func(record *Record) []string

// DetailFunc will called when type `Enter` to get the detail message of
// the record.
type DetailFunc func(record *Record) string

// BriefColumnAttribute the brief column attribute.
type BriefColumnAttribute struct {
	Title    string // Will show in the top line
	MaxWidth int    // the element max width
}

type message struct {
	Seq    int32
	Record *Record
}

const (
	bitFrozen = 1 << iota
	bitDetail
	bitStop
	bitMulti
)

const (
	selectedColor = tcell.ColorGreen
	defaultColor  = tcell.ColorDefault
)

const (
	mainPageName   = "main"
	detailPageName = "detail"
)

var seqColumnAttribute = &BriefColumnAttribute{
	Title:    "Seq",
	MaxWidth: 4,
}

// view controller the message to draw
type view struct {
	app             *tview.Application
	pages           *tview.Pages
	grid            *tview.Grid
	briefView       *tview.Table
	detailView      *tview.TextView
	statusView      *tview.TextView
	promptView      *tview.TextView
	detailPage      *tview.TextView // will use this view to show the detail if too narrow
	capacity        int
	messages        []*message
	briefFunc       BriefFunc
	detailFunc      DetailFunc
	decodeFunc      DecodeFunc
	briefAttributes []*BriefColumnAttribute
	replayHook      ReplayHook
	briefWidth      int

	currentRow int32

	status uint64

	multis map[int]bool // multiple selected rows
}

func newView(
	app *tview.Application,
	capacity int,
	briefFunc BriefFunc,
	detailFunc DetailFunc,
	decodeFunc DecodeFunc,
	replayHook *ReplayHook,
	briefAttributes []*BriefColumnAttribute) *view {
	v := &view{
		app:             app,
		capacity:        capacity,
		briefFunc:       briefFunc,
		detailFunc:      detailFunc,
		decodeFunc:      decodeFunc,
		briefAttributes: briefAttributes,
		multis:          make(map[int]bool),
	}
	v.makeMessages()
	if replayHook != nil {
		v.replayHook.PreReplay = replayHook.PreReplay
		v.replayHook.PreSend = replayHook.PreSend
		v.replayHook.PostSend = replayHook.PostSend
		v.replayHook.PostReplay = replayHook.PostReplay
	}

	v.briefWidth = seqColumnAttribute.MaxWidth + 2
	for _, attribute := range v.briefAttributes {
		v.briefWidth += 1 + attribute.MaxWidth
	}

	return v
}

func (v *view) makeMessages() {
	v.messages = make([]*message, v.capacity)
}

func (v *view) prompt(str string) {
	v.promptView.SetText(str)
}

func (v *view) Init() {
	v.initBriefView()
	v.initDetailView()
	v.initStatusView()
	v.initPrompt()
	v.initGrid()
	v.initPages()
	v.app.SetRoot(v.pages, true)
	v.app.SetInputCapture(func(event *tcell.EventKey) *tcell.EventKey {
		key := event.Key()
		switch key {
		case tcell.KeyCtrlC:
			v.modal("Quit?", func() {
				v.app.Stop()
			})
			return nil
		}
		return event
	})

}

func (v *view) Run() error {
	return v.app.Run()
}

func (v *view) initTitle() {
	for column, attribute := range append([]*BriefColumnAttribute{seqColumnAttribute}, v.briefAttributes...) {
		cell := tview.NewTableCell(attribute.Title).
			SetTextColor(tcell.ColorYellow).
			SetAlign(tview.AlignLeft).
			SetSelectable(false).
			SetMaxWidth(attribute.MaxWidth).
			SetExpansion(1)
		v.briefView.SetCell(0, column, cell)
	}
}

func (v *view) initBriefView() {
	v.briefView = tview.NewTable().SetFixed(1, 0)
	v.initTitle()

	v.briefView.SetBorder(false)
	v.briefView.SetSelectable(true, false)
	v.briefView.SetSelectedFunc(func(row, column int) {
		v.focusDetail(row)
	})
	v.briefView.SetDoneFunc(func(key tcell.Key) {
		v.focusBrief()
	})

	v.briefView.SetInputCapture(func(event *tcell.EventKey) *tcell.EventKey {
		key := event.Key()
		switch key {
		case tcell.KeyEsc:
			v.prompt("")
		case tcell.KeyRune:
			switch event.Rune() {
			case 'f':
				v.toggle(bitFrozen)
				return nil
			case 's':
				v.toggle(bitStop)
				return nil
			case 'C':
				v.clear()
				return nil
			case 'S':
				v.save()
				return nil
			case 'L':
				v.load()
				return nil
			case 'M':
				v.clearMulti()
				v.toggle(bitMulti)
				return nil
			case 'm':
				v.multiSelect()
				return nil
			case 'r':
				v.revertSelect()
				return nil
			case 'a':
				v.selectAll()
				return nil
			case 'c':
				v.clearSelect()
				return nil
			case 'R':
				v.replay()
			case '?':
				v.help()
				return nil
			}
		}
		return event
	})
}

func (v *view) initDetailView() {
	v.detailView = tview.NewTextView()
	v.detailView.SetBorder(false)
	v.detailView.SetWrap(true)
	v.detailView.SetWordWrap(true)

	v.detailView.SetInputCapture(v.detailEventHandler)
}

func (v *view) detailEventHandler(event *tcell.EventKey) *tcell.EventKey {
	key := event.Key()
	switch key {
	case tcell.KeyEsc:
		v.focusBrief()
		return nil
	case tcell.KeyRune:
		switch event.Rune() {
		case 'q':
			v.focusBrief()
			return nil
		case 'f':
			v.toggle(bitFrozen)
			return nil
		case 's':
			v.toggle(bitStop)
		case '?':
			v.help()
			return nil
		}
	}
	return event
}

func (v *view) initStatusView() {
	v.statusView = tview.NewTextView()
	v.statusView.SetBorder(false)
	v.statusView.SetWrap(false)
	v.statusView.SetWordWrap(false)
	v.statusView.SetRegions(true)
	v.statusView.SetTextAlign(tview.AlignLeft)
	v.statusView.Highlight("a")
	v.redrawStatus()
}

func (v *view) initPrompt() {
	v.promptView = tview.NewTextView()
	v.promptView.SetBorder(false)
	v.promptView.SetWrap(false)
	v.promptView.SetWordWrap(false)
	v.promptView.SetTextAlign(tview.AlignLeft)
	v.promptView.Clear()
	v.prompt(strings.Trim(fmt.Sprintf("%v", os.Args), "[]"))
}

func (v *view) initGrid() {
	separation := tview.NewTextView()
	fmt.Fprintf(separation, "|")
	separation.SetTextColor(tcell.ColorGreen)

	flex := tview.NewFlex()
	flex.AddItem(v.statusView, 4, 1, false).
		AddItem(separation, 1, 1, false).
		AddItem(v.promptView, 0, 1, false)

	v.grid = tview.NewGrid().
		SetRows(-1, 1).
		SetColumns(v.briefWidth, -1).
		SetBorders(true).
		AddItem(v.briefView, 0, 0, 1, 1, 0, 0, true).
		AddItem(v.detailView, 0, 1, 1, 1, 0, 0, false).
		AddItem(flex, 1, 0, 1, 2, 0, 0, false)
}

func (v *view) initPages() {
	v.pages = tview.NewPages()
	v.pages.AddPage(mainPageName, v.grid, true, true)
}

func (v *view) focusBrief() {
	v.detailView.Clear()
	v.app.SetFocus(v.briefView)
	bitClear(&v.status, bitDetail)
	v.redrawStatus()
}

func (v *view) focusDetail(row int) {
	log.Infof("focus detail, row: %d", row)
	rm := v.rowMessage(row)
	if rm == nil {
		log.Errorf("row: %d message is nil", row)
		return
	}

	record := rm.Record
	detail := v.detailFunc(record)

	_, _, width, _ := v.grid.GetRect()
	if width <= 2*v.briefWidth {
		v.focusInDetailPage(detail)
	} else {
		v.focusInDetailView(detail)
	}

	bitSet(&v.status, bitDetail)
	v.redrawStatus()
}

func (v *view) focusInDetailView(detail string) {
	v.detailView.Clear()
	fmt.Fprintf(v.detailView, detail)
	v.app.SetFocus(v.detailView)
}

func (v *view) focusInDetailPage(detail string) {
	if !v.pages.HasPage(detailPageName) {
		v.createDetailPage()
	}
	v.detailPage.Clear()
	v.detailPage.SetText(detail)
	v.pages.SwitchToPage(detailPageName)
	v.pages.ShowPage(detailPageName)
}

func (v *view) createDetailPage() {
	textView := tview.NewTextView()
	textView.SetBorder(true)
	textView.SetTitle("detail")
	textView.SetInputCapture(func(event *tcell.EventKey) *tcell.EventKey {
		key := event.Key()
		if key == tcell.KeyEsc || (key == tcell.KeyRune && event.Rune() == 'q') {
			v.pages.SwitchToPage(mainPageName)
		}
		return v.detailEventHandler(event)
	})
	v.detailPage = textView
	v.pages.AddPage(detailPageName, v.detailPage, true, false)
}

func (v *view) Update(record *Record) {
	if record == nil || isSet(v.status, bitStop) {
		return
	}

	v.updateDraw(record)
}

func (v *view) updateDraw(record *Record) {
	v.app.QueueUpdateDraw(func() {
		if isSet(v.status, bitStop) {
			return
		}

		if v.capacity == int(v.currentRow) {
			v.removeHalf()
		}

		v.drawMessage(record)
	})
}

func (v *view) drawMessage(record *Record) {
	row := atomic.AddInt32(&v.currentRow, 1)

	cell := tview.NewTableCell(fmt.Sprintf("%X", row)).
		SetTextColor(tcell.ColorGreen).
		SetAlign(tview.AlignLeft).
		SetSelectable(true).
		SetMaxWidth(seqColumnAttribute.MaxWidth).
		SetExpansion(1)
	v.briefView.SetCell(int(row), 0, cell)

	items := v.briefFunc(record)
	for column, item := range items {
		cell := tview.NewTableCell(item).
			SetTextColor(tcell.ColorWhite).
			SetAlign(tview.AlignLeft).
			SetSelectable(true).
			SetMaxWidth(v.briefAttributes[column].MaxWidth).
			SetExpansion(1)
		v.briefView.SetCell(int(row), column+1, cell)

		if !isSet(v.status, bitDetail|bitFrozen) {
			v.briefView.Select(int(row), 0)
		}
	}

	v.messages[row-1] = &message{
		Seq:    row,
		Record: record,
	}
}

func (v *view) removeHalf() {
	total := int(v.currentRow)
	messages := v.messages[total/2:]
	v.makeMessages()
	records := make([]*Record, len(messages))
	for i, m := range messages {
		records[i] = m.Record
	}
	v.redraw(records)

	removed := total - len(messages)
	v.prompt(fmt.Sprintf("%d records removed", removed))
}

func (v *view) redraw(records []*Record) {
	v.briefView.Clear()
	v.currentRow = 0
	v.initTitle()
	for _, record := range records {
		v.drawMessage(record)
	}
}

func (v *view) toggle(bit uint64) {
	if isSet(v.status, bit) {
		bitClear(&v.status, bit)
	} else {
		bitSet(&v.status, bit)
	}
	v.redrawStatus()
}

func (v *view) clear() {
	if v.currentRow == 0 {
		return
	}

	v.modal("Clear all?", func() {
		v.briefView.Clear()
		v.initTitle()
		v.currentRow = 0
	})

}

func (v *view) modal(text string, okFunc func()) {
	pageName := "modal"
	modal := tview.NewModal()
	modal.SetBorder(true)
	modal.SetText(text)
	modal.AddButtons([]string{"OK", "Cancel"})
	modal.SetDoneFunc(func(buttonIndex int, buttonLabel string) {
		if buttonIndex == 0 {
			okFunc()
		}
		v.destroyPage(pageName)
	})
	v.pages.AddPage(pageName, modal, true, true)
	v.app.SetFocus(modal)
}

func (v *view) statusString() string {
	result := ""
	if isSet(v.status, bitFrozen) {
		result += `["a"]F[""]`
	} else {
		result += "F"
	}
	if isSet(v.status, bitDetail) {
		result += `["a"]D[""]`
	} else {
		result += "D"
	}
	if isSet(v.status, bitStop) {
		result += `["a"]S[""]`
	} else {
		result += "S"
	}
	if isSet(v.status, bitMulti) {
		result += `["a"]M[""]`
	} else {
		result += "M"
	}

	return result
}

func (v *view) redrawStatus() {
	v.statusView.Clear()
	fmt.Fprintf(v.statusView, v.statusString())
}

func (v *view) save() {
	var messages []*message
	isMulti := isSet(v.status, bitMulti)

	if isMulti {
		// multi mode
		messages = v.selectedMessage()
	} else {
		// all
		messages = v.messages[:int(v.currentRow)]
	}

	if v.currentRow == 0 {
		v.prompt("No message, not need to save.")
		return
	}

	title := ""
	if isMulti {
		title = " Save selected "
	} else {
		title = " Save all "
	}

	v.saveOrLoadModal(title, "Save", func(path string) {
		err := serialize(messages, path)
		if err != nil {
			log.Errorf("serialize failed, err: %+v", err)
			v.prompt(fmt.Sprintf("Save to %s failed, %v", path, err))
		} else {
			v.prompt(fmt.Sprintf("Save to %s success", path))
		}
	})
}

func (v *view) load() {
	v.saveOrLoadModal(" Load records ", "Load", v.loadFile)
}

func (v *view) loadFile(path string) {
	messages, err := v.deserialize(path)
	if err != nil {
		log.Errorf("serialize failed, err: %v", err)
		v.prompt(fmt.Sprintf("Load from %s failed, err: %v", path, err))
	} else {
		v.redraw(messages)
		v.redrawStatus()
		v.prompt(fmt.Sprintf("Load from %s success", path))
	}
}

func (v *view) saveOrLoadModal(title, okButton string, okFunc func(string)) {
	pageName := "modal"

	form := tview.NewForm()
	form.SetTitle(title)
	form.AddInputField("path", "", 50, nil, nil)
	form.SetBorder(true)
	form.SetButtonsAlign(tview.AlignCenter)
	v.pages.AddPage(pageName, nonstandardModal(form, 60, 7), true, true)
	form.AddButton(okButton, func() {
		pathItem := form.GetFormItemByLabel("path")
		input := pathItem.(*tview.InputField)
		path := input.GetText()
		log.Debugf("%s from path: %s", title, path)
		okFunc(path)
		v.destroyPage(pageName)
	})
	form.AddButton("Quit", func() {
		v.destroyPage(pageName)
	})
	v.app.SetFocus(form)
}

func serialize(messages []*message, filename string) error {
	serializations := make([]*serialization, len(messages))
	for i, m := range messages {
		serializations[i] = message2Serialization(m.Record)
		for _, b := range m.Record.Bodies {
			gob.Register(b)
		}
	}

	var buffer bytes.Buffer
	enc := gob.NewEncoder(&buffer)
	err := enc.Encode(serializations)
	if err != nil {
		return err
	}

	f, err := os.OpenFile(filename, os.O_RDWR|os.O_CREATE|os.O_TRUNC, 0644)
	if err != nil {
		return err
	}
	defer f.Close()

	_, err = f.Write(buffer.Bytes())
	if err != nil {
		return err
	}

	return nil
}

func (v *view) deserialize(filename string) ([]*Record, error) {
	f, err := os.Open(filename)
	if err != nil {
		log.Errorf("open file err: %v", err)
		return nil, err
	}

	b, err := ioutil.ReadAll(f)
	if err != nil {
		return nil, err
	}

	defer f.Close()

	buffer := bytes.NewBuffer(b)

	dec := gob.NewDecoder(buffer)
	serializations := make([]*serialization, 0)
	err = dec.Decode(&serializations)
	if err != nil {
		log.Errorf("decode failed, err: %v", err)
		return nil, err
	}

	records := make([]*Record, 0, len(serializations))
	for _, s := range serializations {
		net, err := s.Net()
		if err != nil {
			continue
		}
		transport, err := s.Transport()
		if err != nil {
			continue
		}

		bodies, _, err := v.decodeFunc(net, transport, s.Buffer)
		if err != nil {
			continue
		}

		record := &Record{
			Net:       net,
			Transport: transport,
			Seen:      s.Seen,
			Bodies:    bodies,
			Buffer:    s.Buffer,
		}

		records = append(records, record)
	}

	return records, nil
}

func (v *view) multiSelect() {
	if !isSet(v.status, bitMulti) {
		return
	}
	row, _ := v.briefView.GetSelection()
	var color tcell.Color
	if v.multis[row] {
		// unselect
		delete(v.multis, row)
		color = defaultColor
	} else {
		// select
		v.multis[row] = true
		color = selectedColor
	}

	v.setRowBackgroundColor(row, color)
}

func (v *view) revertSelect() {
	if !isSet(v.status, bitMulti) {
		return
	}

	for i := 1; i <= int(v.currentRow); i++ {
		if v.multis[i] {
			delete(v.multis, i)
			v.setRowBackgroundColor(i, defaultColor)
		} else {
			v.multis[i] = true
			v.setRowBackgroundColor(i, selectedColor)
		}
	}
}

func (v *view) selectAll() {
	if !isSet(v.status, bitMulti) {
		return
	}
	selecte := len(v.multis) != int(v.currentRow)
	if selecte {
		for i := 1; i <= int(v.currentRow); i++ {
			if !v.multis[i] {
				v.multis[i] = true
				v.setRowBackgroundColor(i, selectedColor)
			}
		}
	} else {
		for i := 1; i <= int(v.currentRow); i++ {
			if v.multis[i] {
				delete(v.multis, i)
				v.setRowBackgroundColor(i, defaultColor)
			}
		}
	}
}

func (v *view) clearSelect() {
	if !isSet(v.status, bitMulti) {
		return
	}

	for m := range v.multis {
		delete(v.multis, m)
		v.setRowBackgroundColor(m, defaultColor)
	}
}

func (v *view) selectedMessage() []*message {
	messages := make([]*message, 0, len(v.multis))
	rows := make([]int, 0, len(v.multis))
	for m := range v.multis {
		rows = append(rows, m)
	}
	sort.Ints(rows)

	for _, r := range rows {
		m := v.messages[int(r-1)]
		if m == nil {
			continue
		}
		messages = append(messages, m)
	}

	return messages
}

func (v *view) clearMulti() {
	cellLen := len(v.briefAttributes)
	for m := range v.multis {
		for i := 1; i <= cellLen; i++ {
			v.briefView.GetCell(m, i).SetBackgroundColor(tcell.ColorDefault)
		}
		delete(v.multis, m)
	}
}

func (v *view) destroyPage(name string) {
	v.pages.RemovePage(name)
	v.pages.SwitchToPage(mainPageName)
	v.app.SetFocus(v.briefView)
}

func (v *view) setRowBackgroundColor(row int, color tcell.Color) {
	for i := 1; i <= len(v.briefAttributes); i++ {
		v.briefView.GetCell(row, i).SetBackgroundColor(color)
	}
}

func (v *view) help() {
	title := [3]string{"view", "key", "summary"}
	items := [][3]string{
		[3]string{"all", "f", "toggle frozen scroll"},
		[3]string{"all", "s", "toggle stop capture"},
		[3]string{"all", "h/Left", "left"},
		[3]string{"all", "l/Right", "right"},
		[3]string{"all", "j/Down", "down"},
		[3]string{"all", "k/Up", "up"},
		[3]string{"all", "g/Home", "goto first line"},
		[3]string{"all", "G/End", "goto last line"},
		[3]string{"all", "ctrl-f/PgDn", "page down"},
		[3]string{"all", "ctrl-b/PgUp", "page up"},
		[3]string{"all", "ctrl-c", "exit"},
		[3]string{"all", "?", "help"},
		[3]string{"brief", "enter", "enter detail"},
		[3]string{"brief", "Esc", "clean prompt"},
		[3]string{"brief", "C", "clear"},
		[3]string{"brief", "S", "save selected/all"},
		[3]string{"brief", "L", "load from file"},
		[3]string{"brief", "M", "toggle multiple select mode"},
		[3]string{"brief", "m", "select/unselect row, select mode only"},
		[3]string{"brief", "r", "revert selected, select mode only"},
		[3]string{"brief", "a", "select/unselect all, select mode only"},
		[3]string{"brief", "c", "clear selected, select mode only"},
		[3]string{"brief", "R", "replay current/seleted row"},
		[3]string{"detail", "q/Esc", "exit detail"},
		[3]string{"help", "q/Esc", "exit help"},
	}

	maxWidths := [3]int{}

	table := tview.NewTable()
	table.SetBorder(true)
	table.SetTitle(" help ")
	table.SetFixed(1, 0)
	for column, t := range title {
		cell := tview.NewTableCell(t).
			SetTextColor(tcell.ColorYellow).
			SetSelectable(false)
		table.SetCell(0, column, cell)
	}
	for row, item := range items {
		for column, t := range item {
			currentWidth := runewidth.StringWidth(t)
			if currentWidth > maxWidths[column] {
				maxWidths[column] = currentWidth
			}
			cell := tview.NewTableCell(t).
				SetSelectable(false)
			table.SetCell(row+1, column, cell)
		}
	}

	pageName := "help"

	table.SetInputCapture(func(event *tcell.EventKey) *tcell.EventKey {
		key := event.Key()
		switch key {
		case tcell.KeyEsc:
			v.destroyPage(pageName)
			return nil
		case tcell.KeyRune:
			switch event.Rune() {
			case 'q':
				v.destroyPage(pageName)
				return nil
			}
		}
		return event
	})

	maxWidth := 4 // add 2 space width, 2 border
	for _, width := range maxWidths {
		maxWidth += width
	}

	v.pages.AddPage(pageName, nonstandardModal(table, maxWidth, len(items)+3), true, true)
	v.app.SetFocus(table)
}

func (v *view) replay() {
	// get the replay items
	records := v.replayRecords()
	if len(records) == 0 {
		return
	}

	ip := records[0].Net.Dst().String()
	// INFO(tenfyzhong) 2019-01-14 19:58
	// localhost maybe ::1
	// convert ::1 to 127.0.0.1 which can connect to
	if ip == "::1" {
		ip = "127.0.0.1"
	}
	port := records[0].Transport.Dst().String()

	defaultAddr := fmt.Sprintf("%s:%s", ip, port)
	log.Debugf("replay default addr: %s", defaultAddr)

	pageName := "modal"

	networkType := int(records[0].Type)
	networks := []string{"tcp", "udp"}
	selectedNetwork := networks[networkType]

	// get server address
	form := tview.NewForm()
	form.SetTitle(" replay ")
	form.AddDropDown(
		"network",
		[]string{"tcp", "udp"},
		networkType,
		func(option string, optionIndex int) { selectedNetwork = option })
	form.AddInputField("ip:port", defaultAddr, 20, nil, nil)
	form.SetBorder(true)
	form.SetButtonsAlign(tview.AlignCenter)
	v.pages.AddPage(pageName, nonstandardModal(form, 32, 9), true, true)
	form.AddButton("OK", func() {
		pathItem := form.GetFormItemByLabel("ip:port")
		input := pathItem.(*tview.InputField)
		addr := input.GetText()
		log.Debugf("replay to :%s", addr)
		go v.replaySend(selectedNetwork, addr, records)

		v.destroyPage(pageName)
	})
	form.AddButton("Quit", func() {
		v.destroyPage(pageName)
	})
	v.app.SetFocus(form)

}

func (v *view) replayRecords() []*Record {
	records := make([]*Record, 0)
	if isSet(v.status, bitMulti) {
		messages := v.selectedMessage()
		for _, m := range messages {
			records = append(records, m.Record)
		}
	} else {
		m := v.currentMessage()
		if m != nil {
			records = append(records, m.Record)
		}
	}
	return records
}

func (v *view) replaySend(network, addr string, records []*Record) error {
	if len(records) == 0 {
		return nil
	}

	// make socket
	conn, err := net.DialTimeout(network, addr, 1*time.Second)
	if err != nil {
		v.prompt(fmt.Sprintf("Dial %s failed, err: %v", addr, err))
		return err
	}
	defer conn.Close()

	// prereplay
	if v.replayHook.PreReplay != nil {
		err := v.replayHook.PreReplay(conn, records)
		if err != nil {
			v.prompt(fmt.Sprintf("Prereplay failed, err: %v", err))
			return err
		}
	}

	for i, m := range records {
		log.Debugf("replay message seq: %d", i)
		// presend
		if v.replayHook.PreSend != nil {
			err := v.replayHook.PreSend(conn, m)
			if err != nil {
				continue
			}
		}

		// replay
		err := conn.SetWriteDeadline(time.Now().Add(1 * time.Second))
		if err != nil {
			return err
		}
		writen := 0
		for writen < len(m.Buffer) {
			n, err := conn.Write(m.Buffer[writen:])
			if err != nil {
				continue
			}
			writen += n
		}

		// postsend
		if v.replayHook.PostSend != nil {
			err := v.replayHook.PostSend(conn, m)
			if err != nil {
				continue
			}
		}
	}

	// postreplay
	if v.replayHook.PostReplay != nil {
		err := v.replayHook.PostReplay(conn)
		if err != nil {
			v.prompt(fmt.Sprintf("Postreplay failed, err: %v", err))
			return err
		}
	}

	v.prompt(fmt.Sprintf("Replay finished, addr: %s", addr))

	return nil
}

func (v *view) rowMessage(row int) *message {
	if v.currentRow == 0 {
		return nil
	}

	m := v.messages[int32(row-1)]
	return m
}

func (v *view) currentMessage() *message {
	row, _ := v.briefView.GetSelection()
	return v.rowMessage(row)
}

func bitSet(status *uint64, bit uint64) {
	*status |= bit
}

func bitClear(status *uint64, bit uint64) {
	*status &= ^bit
}

func isSet(status uint64, bit uint64) bool {
	return (status & bit) != 0
}

func nonstandardModal(p tview.Primitive, width, height int) tview.Primitive {
	return tview.NewFlex().
		AddItem(nil, 0, 1, false).
		AddItem(tview.NewFlex().SetDirection(tview.FlexRow).
			AddItem(nil, 0, 1, false).
			AddItem(p, height, 1, false).
			AddItem(nil, 0, 1, false), width, 1, false).
		AddItem(nil, 0, 1, false)
}
