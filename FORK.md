# Consumer patches

This branch is based on upstream commit
`49e6b2bc0cf07623f6d1f24a5160eee24ee77e68`.
The upstream README and BSD licence remain unchanged. The module/import path
remains `github.com/alexbrainman/odbc`; consumers pin this fork with a Go `replace`.

The patch changes only Microsoft Access byte-slice parameter bindings:

- Preserve an explicitly described `SQL_LONGVARBINARY` type.
- Use `SQL_LONGVARBINARY` above a described short-binary size, or above the
  conservative 510-byte Access short-binary default when none is available.
- Give an empty binary parameter nonzero precision, without changing its
  zero-byte buffer/indicator or converting it into SQL NULL.
- Preserve every non-Access type/size decision and all supplied bytes.

Microsoft documents OLE/LONGBINARY as `SQL_LONGVARBINARY` and the 510-byte default:
https://learn.microsoft.com/en-us/sql/odbc/microsoft/microsoft-access-data-types

Related open upstream question (not an exact report of this precision failure):
https://github.com/alexbrainman/odbc/issues/187
The closed MEMO/string precision issue is a separate precedent, not this bug:
https://github.com/alexbrainman/odbc/issues/106
No matching open precision-bug PR was identified. No upstream issue, comment or
pull request is created by this fork.

`TestBinaryBindingAccess`, `TestBinaryBindingNonAccessUnchanged`, and
`TestBinaryBindingNullDispatch` are offline unit tests. The last uses only an
invalid statement handle, never a database connection. A real Access driver
accepting the repaired binding still requires an independently authorized test.

## Borrowed connection handle

`(*Conn).WithHandle(func(api.SQLHDBC) error)` permits native catalog calls such
as SQLTables on an existing connection. Use it inside `database/sql.Conn.Raw`.
The callback must not retain, disconnect or free the borrowed handle, change
transaction state, or leave native statements/cursors open. The API allocates
no connection and propagates the callback error without changing ownership.

`TestWithHandleBorrowsSameConnection` allocates only driver-manager handles,
never calls SQLDriverConnect, and verifies the exact borrowed handle and error
propagation. `TestWithHandleRejectsClosedOrBadConnection` verifies admission.
Native catalog results require a separate driver/database integration test.

Remove the consumer replacement after pinning an upstream revision that includes
equivalent binary-binding and borrowed-handle APIs and passes these regressions.
