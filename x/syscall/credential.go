package xsyscall

import (
	"os/user"
	"strconv"
	"syscall"
)

func UserCredential(u *user.User) (*syscall.Credential, error) {
	uid, err := strconv.ParseUint(u.Uid, 10, 64)
	if err != nil {
		return nil, err
	}

	gid, err := strconv.ParseUint(u.Gid, 10, 64)
	if err != nil {
		return nil, err
	}

	groupIDs, err := u.GroupIds()
	if err != nil {
		return nil, err
	}

	groups := make([]uint32, len(groupIDs))
	for _, groupID := range groupIDs {
		group, err := strconv.ParseUint(groupID, 10, 64)
		if err != nil {
			return nil, err
		}

		groups = append(groups, uint32(group))
	}

	return &syscall.Credential{
		Uid:    uint32(uid),
		Gid:    uint32(gid),
		Groups: groups,
	}, nil
}
