package nibbler_user_group

import (
	nibbler_user_group "github.com/markdicksonjr/nibbler/user/group"
	"testing"
)

func TestSqlPersistenceExtension_ImplementsExtension(t *testing.T) {
	var base nibbler_user_group.PersistenceExtension
	b := SqlPersistenceExtension{}
	base = &b
	if base == nil {
		t.Fatal(base)
	}
}
