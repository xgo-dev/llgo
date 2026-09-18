//go:build !llgo || !js || !wasm || llgo.wasm.emscripten

package user_test

import (
	"errors"
	"os/user"
	"runtime"
	"slices"
	"syscall"
	"testing"
)

func currentUser(t *testing.T) *user.User {
	t.Helper()
	u, err := user.Current()
	if err != nil {
		t.Fatalf("Current error: %v", err)
	}
	if u == nil {
		t.Fatal("Current returned nil user")
	}
	return u
}

func compareUsers(t *testing.T, got, want *user.User) {
	t.Helper()
	if *got != *want {
		t.Errorf("user = %+v, want %+v", got, want)
	}
}

func currentGroup(t *testing.T) *user.Group {
	t.Helper()
	u := currentUser(t)
	g, err := user.LookupGroupId(u.Gid)
	if err == nil {
		return g
	}

	gids, groupIDsErr := u.GroupIds()
	if groupIDsErr != nil {
		t.Fatalf("LookupGroupId(%q): %v; GroupIds: %v", u.Gid, err, groupIDsErr)
	}
	for _, gid := range gids {
		if g, lookupErr := user.LookupGroupId(gid); lookupErr == nil {
			return g
		}
	}
	t.Fatalf("no group ID for current user could be resolved: primary %q: %v; groups: %v", u.Gid, err, gids)
	return nil
}

func checkLookupError[T error](t *testing.T, err error) {
	t.Helper()
	if err == nil {
		t.Fatal("lookup unexpectedly succeeded")
	}
	if runtime.GOOS == "windows" {
		// Windows account APIs return the underlying Win32 lookup error. This is
		// also the behavior of the official Go os/user implementation.
		var errno syscall.Errno
		if !errors.As(err, &errno) {
			t.Errorf("error type = %T, want syscall.Errno", err)
		}
		return
	}
	if _, ok := err.(T); !ok {
		t.Errorf("error type = %T, want %T", err, *new(T))
	}
}

func TestCurrent(t *testing.T) {
	u := currentUser(t)

	if u.Uid == "" {
		t.Error("User Uid is empty")
	}

	if u.Username == "" {
		t.Error("User Username is empty")
	}
}

func TestLookup(t *testing.T) {
	want := currentUser(t)
	u, err := user.Lookup(want.Username)
	if err != nil {
		t.Fatalf("Lookup(%q) error: %v", want.Username, err)
	}
	compareUsers(t, u, want)
}

func TestLookupNonexistent(t *testing.T) {
	_, err := user.Lookup("nonexistent_user_12345")
	checkLookupError[user.UnknownUserError](t, err)
}

func TestLookupId(t *testing.T) {
	want := currentUser(t)
	u, err := user.LookupId(want.Uid)
	if err != nil {
		t.Fatalf("LookupId(%q) error: %v", want.Uid, err)
	}
	compareUsers(t, u, want)
}

func TestLookupIdNonexistent(t *testing.T) {
	id := "99999999"
	if runtime.GOOS == "windows" {
		id = "S-1-5-21-0-0-0-4294967294"
	}
	_, err := user.LookupId(id)
	checkLookupError[user.UnknownUserIdError](t, err)
}

func TestUserGroupIds(t *testing.T) {
	u := currentUser(t)
	gids, err := u.GroupIds()
	if err != nil {
		t.Fatalf("GroupIds error: %v", err)
	}
	if !slices.Contains(gids, u.Gid) {
		t.Errorf("GroupIds = %v, want primary group %q", gids, u.Gid)
	}
}

func TestLookupGroup(t *testing.T) {
	want := currentGroup(t)
	g, err := user.LookupGroup(want.Name)
	if err != nil {
		t.Fatalf("LookupGroup(%q) error: %v", want.Name, err)
	}
	if *g != *want {
		t.Errorf("group = %+v, want %+v", g, want)
	}
}

func TestLookupGroupNonexistent(t *testing.T) {
	_, err := user.LookupGroup("nonexistent_group_12345")
	checkLookupError[user.UnknownGroupError](t, err)
}

func TestLookupGroupId(t *testing.T) {
	want := currentGroup(t)
	g, err := user.LookupGroupId(want.Gid)
	if err != nil {
		t.Fatalf("LookupGroupId(%q) error: %v", want.Gid, err)
	}
	if *g != *want {
		t.Errorf("group = %+v, want %+v", g, want)
	}
}

func TestLookupGroupIdNonexistent(t *testing.T) {
	id := "99999999"
	if runtime.GOOS == "windows" {
		id = "S-1-5-21-0-0-0-4294967294"
	}
	_, err := user.LookupGroupId(id)
	checkLookupError[user.UnknownGroupIdError](t, err)
}

func TestUserFields(t *testing.T) {
	u := currentUser(t)

	if u.Uid == "" {
		t.Error("User.Uid is empty")
	}
	if u.Gid == "" {
		t.Error("User.Gid is empty")
	}
	if u.Username == "" {
		t.Error("User.Username is empty")
	}
}

func TestGroupFields(t *testing.T) {
	g := currentGroup(t)

	if g.Gid == "" {
		t.Error("Group.Gid is empty")
	}
	if g.Name == "" {
		t.Error("Group.Name is empty")
	}

	var _ *user.Group = g
}

func TestUnknownUserError(t *testing.T) {
	err := user.UnknownUserError("testuser")
	errStr := err.Error()
	if errStr == "" {
		t.Error("UnknownUserError.Error() returned empty string")
	}
}

func TestUnknownUserIdError(t *testing.T) {
	err := user.UnknownUserIdError(12345)
	errStr := err.Error()
	if errStr == "" {
		t.Error("UnknownUserIdError.Error() returned empty string")
	}
}

func TestUnknownGroupError(t *testing.T) {
	err := user.UnknownGroupError("testgroup")
	errStr := err.Error()
	if errStr == "" {
		t.Error("UnknownGroupError.Error() returned empty string")
	}
}

func TestUnknownGroupIdError(t *testing.T) {
	err := user.UnknownGroupIdError("12345")
	errStr := err.Error()
	if errStr == "" {
		t.Error("UnknownGroupIdError.Error() returned empty string")
	}
}
