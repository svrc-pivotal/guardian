package runrunc

import (
	"path/filepath"

	"github.com/opencontainers/runc/libcontainer/user"
)

const (
	DefaultUID int = 0
	DefaultGID int = 0
)

func LookupUser(rootFsPath, userName string) (*user.ExecUser, error) {
	defaultUser := &user.ExecUser{Uid: DefaultUID, Gid: DefaultGID}
	passwdPath := filepath.Join(rootFsPath, "etc", "passwd")
	groupPath := filepath.Join(rootFsPath, "etc", "group")

	return user.GetExecUserPath(userName, defaultUser, passwdPath, groupPath)
}
