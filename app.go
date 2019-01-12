package fdump

import (
	"flag"
	"fmt"
	"os"

	logging "github.com/op/go-logging"
	"github.com/rivo/tview"
)

const (
	// LoggerName the logger name
	LoggerName = "fdump"
)

// AppFlagSet Command line flag set, use this flag set to add flags.
// Warning: Don't use the default flag set.
var AppFlagSet = flag.NewFlagSet("", flag.ExitOnError)
var log = logging.MustGetLogger(LoggerName)

var (
	iface    = ""
	fname    = ""
	filter   = ""
	capacity = 0
	lname    = ""
)

func init() {
	AppFlagSet.StringVar(&iface, "i", "any", "Interface to get packet from")
	AppFlagSet.StringVar(&fname, "r", "", "Filename to read from, overrides -i")
	AppFlagSet.StringVar(&filter, "f", "tcp and host localhost", "BPF filter for pcap")
	AppFlagSet.IntVar(&capacity, "m", 65535, "Max capacity, it will remove halt of records when the size is equal to the max capacity, maximum 65535")
	AppFlagSet.StringVar(&lname, "l", "", "Filename to load record from")

	format := logging.MustStringFormatter(
		`%{time:2006-01-02 15:04:05.000} %{level:.4s} %{shortfile}:%{shortfunc} %{message}`,
	)
	logFile, err := os.OpenFile(os.Args[0]+".log", os.O_WRONLY|os.O_CREATE|os.O_APPEND, 0666)
	if err != nil {
		fmt.Println(err)
		os.Exit(-2)
	}
	backendFile := logging.NewLogBackend(logFile, "", 0)
	backendFileFormatter := logging.NewBackendFormatter(backendFile, format)
	backendFileLeveled := logging.AddModuleLevel(backendFileFormatter)
	logging.SetBackend(backendFileLeveled)
	logging.SetLevel(logging.CRITICAL, "")
}

// Init Init the packet. Call it in main/init function before NewApp
func Init() {
	AppFlagSet.Parse(os.Args[1:])
	if capacity <= 0 || capacity > 65535 {
		capacity = 65535
	}
}

// App the application to run
type App struct {
	ctrl *controller
	view *view
}

// NewApp new an App instance.
func NewApp(
	decodeFunc DecodeFunc,
	sidebarFunc SidebarFunc,
	detailFunc DetailFunc,
	replayHook *ReplayHook,
	sidebarAttributes []*SidebarColumnAttribute) *App {
	if decodeFunc == nil || sidebarFunc == nil || detailFunc == nil || len(sidebarAttributes) == 0 {
		return nil
	}

	if iface == "" {
		iface = "any"
	}

	snaplen := 65535
	tapp := tview.NewApplication()
	a := &App{
		ctrl: newController(iface, fname, snaplen, filter, decodeFunc),
		view: newView(
			tapp,
			capacity,
			sidebarFunc,
			detailFunc,
			decodeFunc,
			replayHook,
			sidebarAttributes),
	}
	return a
}

// Run begin work. It will block the goroutine
func (a *App) Run() {
	a.ctrl.AddUpdateFunc(a.view.Update)

	err := a.ctrl.Init()
	if err != nil {
		panic(err)
	}

	go a.ctrl.Run()

	a.view.Init()
	if lname != "" {
		a.view.toggle(bitStop)
		a.view.loadFile(lname)
	}
	err = a.view.Run()
	if err != nil {
		panic(err)
	}
}
