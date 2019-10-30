package target

import (
	"container/list"
	"fmt"
	"regexp"
	"sync"

	basecfg "github.com/studease/common/utils/config"
)

// Server masks
const (
	BACKUP = "backup"
	DOWN   = "down"
)

var (
	srvRe, _ = regexp.Compile("\\${SERVER.([-\\.[:word:]]+)}")
	mtx      sync.RWMutex
	groups   = make(map[string]*Group)
)

// Group of target server
type Group struct {
	mtx      sync.RWMutex
	name     string
	servers  list.List
	backups  list.List
	ptr      *list.Element
	weighted int
}

// Init this class
func (me *Group) Init(name string) *Group {
	me.servers.Init()
	me.backups.Init()
	me.name = name
	me.ptr = nil
	me.weighted = 0
	return me
}

// Add a server into the group
func (me *Group) Add(srv *basecfg.Server) {
	if srv.Name == "" {
		panic("name not presented")
	}
	if srv.HostPort == "" {
		panic("hostport not presented")
	}
	if srv.Weight == 0 {
		srv.Weight = 1
	}
	if srv.Timeout == 0 {
		srv.Timeout = 10
	}

	me.mtx.Lock()
	defer me.mtx.Unlock()

	switch srv.Mask {
	case "":
		me.servers.PushBack(srv)
	case BACKUP:
		me.backups.PushBack(srv)
	}
}

// Remove the server from the group
func (me *Group) Remove(srv *basecfg.Server) {
	me.mtx.Lock()
	defer me.mtx.Unlock()

	for e := me.servers.Front(); e != nil; e = e.Next() {
		if e.Value == srv {
			me.servers.Remove(e)
			return
		}
	}

	for e := me.backups.Front(); e != nil; e = e.Next() {
		if e.Value == srv {
			me.backups.Remove(e)
			return
		}
	}
}

// Get the next useable server
func (me *Group) Get() *basecfg.Server {
	me.mtx.RLock()
	defer me.mtx.RUnlock()

	if me.ptr == nil {
		me.ptr = me.servers.Front()
		if me.ptr == nil {
			me.ptr = me.backups.Front()
			if me.ptr == nil {
				return nil
			}
		}
	}

	srv := me.ptr.Value.(*basecfg.Server)
	me.weighted++

	if me.weighted >= srv.Weight {
		me.ptr = me.ptr.Next()
		me.weighted = 0
	}

	return srv
}

// Add a server into the group
func Add(srv *basecfg.Server) {
	if srv.Name == "" || srv.HostPort == "" {
		return
	}

	mtx.Lock()
	defer mtx.Unlock()

	g, ok := groups[srv.Name]
	if ok == false {
		g = new(Group).Init(srv.Name)
		groups[srv.Name] = g
	}

	g.Add(srv)
}

// Remove the server from the group
func Remove(srv *basecfg.Server) {
	mtx.Lock()
	defer mtx.Unlock()

	g, ok := groups[srv.Name]
	if ok {
		g.Remove(srv)
	}
}

// Get the next useable server
func Get(name string) *basecfg.Server {
	mtx.RLock()
	defer mtx.RUnlock()

	g, ok := groups[name]
	if ok == false {
		return nil
	}

	return g.Get()
}

// Parse returns a copy of rawurl, replacing matches of the Regexp with the HostPort
func Parse(rawurl string) (string, error) {
	var (
		srv *basecfg.Server
	)

	tmp := rawurl

	arr := srvRe.FindStringSubmatch(tmp)
	if arr != nil {
		srv = Get(arr[1])
		if srv == nil {
			return "", fmt.Errorf("unrecognized target server %s", arr[1])
		}

		tmp = srvRe.ReplaceAllString(tmp, srv.HostPort)
	}

	return tmp, nil
}

// Replace returns a copy of rawurl, replacing matches of the Regexp with the HostPort
func Replace(rawurl string, hostport string) string {
	return srvRe.ReplaceAllString(rawurl, hostport)
}
