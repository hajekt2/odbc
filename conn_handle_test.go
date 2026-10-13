package odbc

import (
	"database/sql/driver"
	"errors"
	"testing"

	"github.com/alexbrainman/odbc/api"
)

// Allocate driver-manager handles only. No SQLDriverConnect, DSN or database.
func TestWithHandleBorrowsSameConnection(t *testing.T) {
	var env, dbc api.SQLHANDLE
	if ret := api.SQLAllocHandle(api.SQL_HANDLE_ENV, api.SQLHANDLE(api.SQL_NULL_HANDLE), &env); IsError(ret) {
		t.Fatalf("allocate env: %d", ret)
	}
	defer api.SQLFreeHandle(api.SQL_HANDLE_ENV, env)
	if ret := api.SQLSetEnvUIntPtrAttr(api.SQLHENV(env), api.SQL_ATTR_ODBC_VERSION, api.SQL_OV_ODBC3, 0); IsError(ret) {
		t.Fatalf("set version: %d", ret)
	}
	if ret := api.SQLAllocHandle(api.SQL_HANDLE_DBC, env, &dbc); IsError(ret) {
		t.Fatalf("allocate dbc: %d", ret)
	}
	defer api.SQLFreeHandle(api.SQL_HANDLE_DBC, dbc)
	conn := &Conn{h: api.SQLHDBC(dbc)}
	sentinel := errors.New("callback error")
	calls := 0
	for _, result := range []error{nil, sentinel} {
		err := conn.WithHandle(func(h api.SQLHDBC) error {
			calls++
			if h != api.SQLHDBC(dbc) {
				t.Fatalf("callback borrowed %v, want %v", h, dbc)
			}
			return result
		})
		if !errors.Is(err, result) {
			t.Fatalf("callback result: %v, want %v", err, result)
		}
		if conn.h != api.SQLHDBC(dbc) || conn.bad {
			t.Fatal("handle ownership or state changed")
		}
	}
	if calls != 2 {
		t.Fatalf("calls=%d", calls)
	}
	if err := conn.WithHandle(nil); err == nil {
		t.Fatal("nil callback accepted")
	}
}

func TestWithHandleRejectsClosedOrBadConnection(t *testing.T) {
	for _, conn := range []*Conn{{}, {bad: true}} {
		if err := conn.WithHandle(func(api.SQLHDBC) error { t.Fatal("callback on unusable connection"); return nil }); !errors.Is(err, driver.ErrBadConn) {
			t.Fatalf("error=%v", err)
		}
	}
}
