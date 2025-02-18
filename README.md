## Sqlx

[![GoDoc Widget](https://godoc.org/github.com/kunlun-qilian/sqlx/v3?status.svg)](https://godoc.org/github.com/kunlun-qilian/sqlx/v3)
[![codecov](https://codecov.io/gh/go-courier/sqlx/branch/master/graph/badge.svg)](https://codecov.io/gh/go-courier/sqlx)
[![Go Report Card](https://goreportcard.com/badge/github.com/kunlun-qilian/sqlx/v3)](https://goreportcard.com/report/github.com/kunlun-qilian/sqlx/v3)


Sql helpers just for mysql(5.7+)/postgres(11+) and mysql/postgres-compatibility db.


```go
// @def primary ID CreatedAt
// @def partition range CreatedAt
// @def index I_nickname/BTREE Nickname
// @def index I_username Username
// @def index I_geom/SPATIAL Geom
// @def unique_index I_name Name
type User struct {
	ID uint64 `db:"f_id,autoincrement"`
	// 姓名
	NameNeedToDrop   string                   `db:"f_name_need_to_drop,deprecated"`
	OldName   string                          `db:"f_old_name,deprecated=f_name"`
	Name      string                          `db:"f_name,default=''"`
	Username  string                          `db:"f_username,default=''"`
	Nickname  string                          `db:"f_nickname,default=''"`
	Gender    Gender                          `db:"f_gender,default='0'"`
	Boolean   bool                            `db:"f_boolean,default=false"`
	Geom      GeomString                      `db:"f_geom"`
	CreatedAt datatypes.Timestamp             `db:"f_created_at,default='0'"`
	UpdatedAt datatypes.Timestamp             `db:"f_updated_at,default='0'"`
	Enabled   datatypes.Bool                  `db:"f_enabled,default='0'"`
}

type GeomString struct {
	V string
}

func (g GeomString) Value() (driver.Value, error) {
	return g.V, nil
}

func (g *GeomString) Scan(src interface{}) error {
	return nil
}

func (GeomString) DataType(driverName string) string {
	if driverName == "mysql" {
		return "geometry"
	}
	return "geometry(Point)"
}

func (GeomString) ValueEx() string {
	return "ST_GeomFromText(?)"
}
```
