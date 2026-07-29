package sync

import "testing"

func TestAttrMethods(t *testing.T) {
	for name, initDestroy := range map[string]func() (int32, int32){
		"mutex": func() (int32, int32) {
			var attr MutexAttr
			return int32(attr.Init()), int32(attr.Destroy())
		},
		"rwlock": func() (int32, int32) {
			var attr RWLockAttr
			return int32(attr.Init()), int32(attr.Destroy())
		},
		"cond": func() (int32, int32) {
			var attr CondAttr
			return int32(attr.Init()), int32(attr.Destroy())
		},
	} {
		initResult, destroyResult := initDestroy()
		if initResult != 0 || destroyResult != 0 {
			t.Errorf("%s attribute lifecycle returned (%d, %d)", name, initResult, destroyResult)
		}
	}
}
