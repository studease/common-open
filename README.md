# common
This is a go common lib, includes log, event driven, for media streaming, DFS and websocket chat.

-------
## Note

Extended event packages, such as MediaEvent and NetStatusEvent, could be removed if you don't need them. 
So is av, http and rtmp packages.

Package target is used for load balencing while requesting an upstream server.


----------
## Example

```go
import (
	"time"
	
	"github.com/studease/common/events"
	Event "github.com/studease/common/events/event"
	TimerEvent "github.com/studease/common/events/timerevent"
	"github.com/studease/common/log"
	"github.com/studease/common/utils/timer"
)

var (
	factory := log.NewDefaultLoggerFactory("logs/2006-01-02 15-04-05.000.log", "debug")
	logger := factory.NewLogger("CORE")
)

type Object struct {
	events.EventDispatcher
	
	logger        log.ILogger
	factory       log.ILoggerFactory
	timerListener *events.EventListener
	timer         timer.Timer
}

func (me *Object) Init(logger log.ILogger, factory log.ILoggerFactory) *Object {
	me.logger = logger
	me.factory = factory
	me.timerListener = events.NewListener(onTimer, 0)
	
	me.timer.Init(5*time.Second, 0, me.logger)
	me.timer.AddEventListener(TimerEvent.TIMER, me.timerListener)
	me.timer.Start()
	
	return me
}

func (me *Object) onTimer(e *TimerEvent.TimerEvent) {
	// t will be me.timer, you can use me.timer instead
	t := e.Target.(*timer.Timer)
	t.Stop()
	
	me.logger.Debugf(0, "Connecting...")
	me.connect()
}

func (me *Object) connect() {
	// While connected, dispatch the event
	me.DispatchEvent(Event.New(Event.CONNECT, me))
}

func main() {
	// Auto remove listener after triggered "1" time[s]
	listener := events.NewListener(onConnect, 1)
	
	obj := new(Object).Init(factory.NewLogger("OBJECT"))
	obj.AddEventListener(Event.CONNECT, listener)
	
	c := make(chan int)
	<-c
}

func onConnect(e *Event.Event) {
	logger.Debugf(0, "Connected!")
	
	obj := e.Target.(*Object)
	obj.Anything()
}
```
