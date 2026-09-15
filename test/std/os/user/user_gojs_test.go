//go:build llgo && js && wasm && !llgo.wasm.emscripten

package user_test

import (
	"errors"
	"os/user"
	"strings"
	"syscall"
	"testing"
)

func checkLookupUnavailable[T error](t *testing.T, err error) {
	t.Helper()
	if _, ok := err.(T); ok {
		return
	}
	// A browser filesystem reports ENOSYS; Node-backed compatibility runners
	// can inspect passwd/group files and therefore report the typed not-found
	// error instead. Both are official host-capability outcomes.
	if !errors.Is(err, syscall.ENOSYS) {
		t.Fatalf("lookup error = %T %v, want %T or ENOSYS", err, err, *new(T))
	}
}

func TestCurrent(t *testing.T) {
	u, err := user.Current()
	if err == nil || u != nil {
		t.Fatalf("Current = %+v, %v; want nil user and an unavailable-host error", u, err)
	}
	if !strings.Contains(err.Error(), "Current requires cgo") {
		t.Fatalf("Current error = %q, want the official pure-Go unavailable-host contract", err)
	}
}

func TestLookupNonexistent(t *testing.T) {
	_, err := user.Lookup("nonexistent_user_12345")
	checkLookupUnavailable[user.UnknownUserError](t, err)
}

func TestLookupIdNonexistent(t *testing.T) {
	_, err := user.LookupId("99999999")
	checkLookupUnavailable[user.UnknownUserIdError](t, err)
}

func TestLookupGroupNonexistent(t *testing.T) {
	_, err := user.LookupGroup("nonexistent_group_12345")
	checkLookupUnavailable[user.UnknownGroupError](t, err)
}

func TestLookupGroupIdNonexistent(t *testing.T) {
	_, err := user.LookupGroupId("99999999")
	checkLookupUnavailable[user.UnknownGroupIdError](t, err)
}

func TestLookupErrorTypes(t *testing.T) {
	errors := []error{
		user.UnknownUserError("testuser"),
		user.UnknownUserIdError(12345),
		user.UnknownGroupError("testgroup"),
		user.UnknownGroupIdError("12345"),
	}
	for _, err := range errors {
		if err.Error() == "" {
			t.Errorf("%T.Error returned an empty string", err)
		}
	}
}
