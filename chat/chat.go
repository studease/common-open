package chat

import (
	"database/sql"
	"time"

	"github.com/studease/common/chat/message"
	"github.com/studease/common/log"
	"github.com/studease/common/utils"
	basecfg "github.com/studease/common/utils/config"
)

// Keys
const (
	KEY_COMMAND = "cmd"
	KEY_SN      = "sn"
	KEY_DATA    = "data"
	KEY_MODE    = "mode"
	KEY_CHANNEL = "channel"
	KEY_GROUP   = "group"
	KEY_USER    = "user"
	KEY_TARGET  = "target"

	KEY_ID         = "id"
	KEY_NAME       = "name"
	KEY_ICON       = "icon"
	KEY_ROLE       = "role"
	KEY_ATTRIBUTES = "attributes"
	KEY_TOTAL      = "total"
	KEY_STAT       = "stat"
	KEY_STATUS     = "status"
	KEY_CODE       = "code"

	KEY_OPTION = "opt"
)

// Commands
const (
	CMD_INFO   = "info"
	CMD_TEXT   = "text"
	CMD_USER   = "user"
	CMD_JOIN   = "join"
	CMD_LEFT   = "left"
	CMD_CTRL   = "ctrl"
	CMD_ATTR   = "attr"
	CMD_EXTERN = "extern"
	CMD_RESULT = "result"
	CMD_ERROR  = "error"
	CMD_PING   = "ping"
	CMD_PONG   = "pong"
)

// Modes
const (
	MODE_UNI       = 0x00
	MODE_GROUP     = 0x01
	MODE_CHANNEL   = 0x02
	MODE_BROADCAST = 0x03
	MODE_HISTORY   = 0x04
)

// Options
const (
	OPT_MUTE   = "mute"
	OPT_FORBID = "forbid"
)

var (
	r = utils.NewRegister()
)

// IQuery responds to an HTTP request
type IQuery interface {
	Init(cfg *basecfg.Query, logger log.ILogger) IQuery
	NewChannel(id string) error
	Attach(user *User) error
	History(channel string, user string, limit int, offset int, callback func(*sql.Rows) error) error
	Insert(channel string, tar string, mgr *message.User, opt string, d time.Duration) error
	Delete(channel string, tar string, opt string) error
	DB() *sql.DB
}

// Register an IQuery with the given name
func Register(name string, q interface{}) {
	r.Add(name, q)
}

// NewQuery creates a registered IQuery by the name
func NewQuery(cfg *basecfg.Query, factory log.ILoggerFactory) IQuery {
	if q := r.New(cfg.Name); q != nil {
		return q.(IQuery).Init(cfg, factory.NewLogger(cfg.Name))
	}

	return nil
}
