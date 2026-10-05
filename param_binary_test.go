// Copyright 2026 The Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package odbc

import (
	"bytes"
	"fmt"
	"testing"

	"github.com/alexbrainman/odbc/api"
)

func TestBinaryBindingAccess(t *testing.T) {
	descriptions := []struct {
		name  string
		p     Parameter
		limit int
		long  bool
	}{
		{name: "undescribed", limit: 510},
		{name: "fixed-default", p: Parameter{isDescribed: true, SQLType: api.SQL_BINARY}, limit: 510},
		{name: "fixed-255", p: Parameter{isDescribed: true, SQLType: api.SQL_BINARY, Size: 255}, limit: 255},
		{name: "variable-510", p: Parameter{isDescribed: true, SQLType: api.SQL_VARBINARY, Size: 510}, limit: 510},
		{name: "long", p: Parameter{isDescribed: true, SQLType: api.SQL_LONGVARBINARY}, long: true},
	}
	for _, description := range descriptions {
		for _, length := range []int{0, 1, 255, 256, 510, 511, 1069, 7999, 8000, 8193} {
			t.Run(fmt.Sprintf("%s/%d", description.name, length), func(t *testing.T) {
				p := description.p
				value := bytes.Repeat([]byte{0, 255, 128, 39}, (length+3)/4)[:length]
				original := append([]byte(nil), value...)
				typ, size := p.binaryBinding(value, true)
				want := api.SQLSMALLINT(api.SQL_BINARY)
				if description.p.isDescribed {
					want = description.p.SQLType
				}
				if description.long || length > description.limit || (!description.p.isDescribed && length == 0) {
					want = api.SQL_LONGVARBINARY
				}
				if typ != want {
					t.Fatalf("type=%d want=%d", typ, want)
				}
				wantSize := length
				if wantSize == 0 {
					wantSize = 1
				}
				if size != api.SQLULEN(wantSize) || p.StrLen_or_IndPtr != api.SQLLEN(length) {
					t.Fatalf("precision=%d indicator=%d length=%d", size, p.StrLen_or_IndPtr, length)
				}
				buffer := p.Data.([]byte)
				if buffer == nil || !bytes.Equal(buffer, original) || !bytes.Equal(value, original) {
					t.Fatal("binary bytes changed or empty became NULL")
				}
				if length > 0 {
					value[0] ^= 0xff
					if !bytes.Equal(buffer, original) {
						t.Fatal("binding aliases caller's buffer")
					}
				}
			})
		}
	}
}

func TestBinaryBindingNonAccessUnchanged(t *testing.T) {
	for _, description := range []Parameter{{}, {isDescribed: true, SQLType: api.SQL_BINARY, Size: 255}, {isDescribed: true, SQLType: api.SQL_VARBINARY}, {isDescribed: true, SQLType: api.SQL_LONGVARBINARY}} {
		for _, length := range []int{0, 1, 255, 256, 510, 511, 1069, 7999, 8000, 8193} {
			p := description
			typ, size := p.binaryBinding(make([]byte, length), false)
			want := api.SQLSMALLINT(api.SQL_BINARY)
			if description.isDescribed {
				want = description.SQLType
			} else if length == 0 || length >= 8000 {
				want = api.SQL_LONGVARBINARY
			}
			if typ != want || size != api.SQLULEN(length) || p.StrLen_or_IndPtr != api.SQLLEN(length) {
				t.Fatalf("non-Access binding changed: described=%t length=%d", description.isDescribed, length)
			}
		}
	}
}

// Invalid statement handles cannot execute a query or open a connection. The
// test checks the actual BindValue dispatch, including NULL versus empty bytes.
func TestBinaryBindingNullDispatch(t *testing.T) {
	for _, access := range []bool{false, true} {
		for _, value := range []interface{}{nil, []byte{}} {
			p := Parameter{}
			var invalidHandle api.SQLHSTMT
			if err := p.BindValue(invalidHandle, 0, value, &Conn{isMSAccessDriver: access}); err == nil {
				t.Fatal("invalid handle unexpectedly accepted")
			}
			if value == nil {
				if p.Data != nil || p.StrLen_or_IndPtr != api.SQL_NULL_DATA {
					t.Fatal("NULL lost its indicator")
				}
			} else if p.Data == nil || p.StrLen_or_IndPtr != 0 {
				t.Fatal("empty bytes became NULL")
			}
		}
	}
}
