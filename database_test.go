package sqlx_test

import (
	"context"
	"database/sql/driver"
	"fmt"
	"os"
	"testing"
	"time"

	"github.com/go-courier/logr"

	"github.com/go-courier/metax"
	_ "github.com/go-sql-driver/mysql"
	"github.com/google/uuid"
	"github.com/kunlun-qilian/sqlx/v3"
	"github.com/kunlun-qilian/sqlx/v3/builder"
	"github.com/kunlun-qilian/sqlx/v3/datatypes"
	"github.com/kunlun-qilian/sqlx/v3/migration"
	"github.com/kunlun-qilian/sqlx/v3/mysqlconnector"
	"github.com/kunlun-qilian/sqlx/v3/postgresqlconnector"
	. "github.com/onsi/gomega"
)

var (
	mysqlConnector = &mysqlconnector.MysqlConnector{
		Host:  "root@tcp(0.0.0.0:3306)",
		Extra: "charset=utf8mb4&parseTime=true&interpolateParams=true&autocommit=true&loc=Local",
	}

	postgresConnector = &postgresqlconnector.PostgreSQLConnector{
		Host:       "postgres://postgres@0.0.0.0:5432",
		Extra:      "sslmode=disable",
		Extensions: []string{"postgis"},
	}
)

func Background() context.Context {
	return logr.WithLogger(context.Background(), logr.StdLogger())
}

type TableOperateTime struct {
	CreatedAt datatypes.MySQLDatetime `db:"f_created_at,default=CURRENT_TIMESTAMP,onupdate=CURRENT_TIMESTAMP"`
	UpdatedAt int64                   `db:"f_updated_at,default='0'"`
}

type Gender int

const (
	GenderMale Gender = iota + 1
	GenderFemale
)

func (Gender) EnumType() string {
	return "Gender"
}

func (Gender) Enums() map[int][]string {
	return map[int][]string{
		int(GenderMale):   {"male", "男"},
		int(GenderFemale): {"female", "女"},
	}
}

func (g Gender) String() string {
	switch g {
	case GenderMale:
		return "male"
	case GenderFemale:
		return "female"
	}
	return ""
}

type User struct {
	ID       uint64 `db:"f_id,autoincrement"`
	Name     string `db:"f_name,size=255,default=''"`
	Nickname string `db:"f_nickname,size=255,default=''"`
	Username string `db:"f_username,default=''"`
	Gender   Gender `db:"f_gender,default='0'"`

	TableOperateTime
}

func (user *User) Comments() map[string]string {
	return map[string]string{
		"Name": "姓名",
	}
}

func (user *User) TableName() string {
	return "t_user"
}

func (user *User) PrimaryKey() []string {
	return []string{"ID"}
}

func (user *User) Indexes() builder.Indexes {
	return builder.Indexes{
		"i_nickname": {"Nickname"},
	}
}

func (user *User) UniqueIndexes() builder.Indexes {
	return builder.Indexes{
		"i_name": {"Name"},
	}
}

type User2 struct {
	ID       uint64 `db:"f_id,autoincrement"`
	Nickname string `db:"f_nickname,size=255,default=''"`
	Gender   Gender `db:"f_gender,default='0'"`
	Name     string `db:"f_name,deprecated=f_real_name"`
	RealName string `db:"f_real_name,size=255,default=''"`
	Age      int32  `db:"f_age,default='0'"`
	Username string `db:"f_username,deprecated"`
}

func (user *User2) TableName() string {
	return "t_user"
}

func (user *User2) PrimaryKey() []string {
	return []string{"ID"}
}

func (user *User2) Indexes() builder.Indexes {
	return builder.Indexes{
		"i_nickname": {"Nickname"},
	}
}

func (user *User2) UniqueIndexes() builder.Indexes {
	return builder.Indexes{
		"i_name": {"RealName"},
	}
}

func TestMigrate(t *testing.T) {
	os.Setenv("PROJECT_FEATURE", "test1")
	defer func() {
		os.Remove("PROJECT_FEATURE")
	}()

	dbTest := sqlx.NewFeatureDatabase("test_for_migrate")

	for i, connector := range []driver.Connector{
		mysqlConnector,
		postgresConnector,
	} {
		t.Run(fmt.Sprintf("%d", i), func(t *testing.T) {
			for _, schema := range []string{"import", "public", "backup"} {
				dbTest.Tables.Range(func(table *builder.Table, idx int) {
					db := dbTest.OpenDB(connector).WithSchema(schema)
					_, _ = db.ExecExpr(db.Dialect().DropTable(table))
				})

				t.Run("create table", func(t *testing.T) {
					dbTest.Register(&User{})
					db := dbTest.OpenDB(connector).WithSchema(schema)

					t.Run("first migrate", func(t *testing.T) {
						err := migration.Migrate(db, nil)
						NewWithT(t).Expect(err).To(BeNil())
					})

					t.Run("again", func(t *testing.T) {
						_ = migration.Migrate(db, os.Stdout)
						err := migration.Migrate(db, nil)
						NewWithT(t).Expect(err).To(BeNil())
					})
				})

				t.Run("no migrate", func(t *testing.T) {
					dbTest.Register(&User{})
					db := dbTest.OpenDB(connector).WithSchema(schema)
					err := migration.Migrate(db, nil)
					NewWithT(t).Expect(err).To(BeNil())

					t.Run("migrate to user2", func(t *testing.T) {
						dbTest.Register(&User2{})
						db := dbTest.OpenDB(connector).WithSchema(schema)
						err := migration.Migrate(db, nil)
						NewWithT(t).Expect(err).To(BeNil())
					})

					t.Run("migrate to user2 again", func(t *testing.T) {
						dbTest.Register(&User2{})
						db := dbTest.OpenDB(connector).WithSchema(schema)
						err := migration.Migrate(db, nil)
						NewWithT(t).Expect(err).To(BeNil())
					})
				})

				t.Run("migrate to user", func(t *testing.T) {
					db := dbTest.OpenDB(connector).WithSchema(schema)
					err := migration.Migrate(db, os.Stdout)
					NewWithT(t).Expect(err).To(BeNil())
					err = migration.Migrate(db, nil)
					NewWithT(t).Expect(err).To(BeNil())
				})

				dbTest.Tables.Range(func(table *builder.Table, idx int) {
					db := dbTest.OpenDB(connector).WithSchema(schema)
					_, _ = db.ExecExpr(db.Dialect().DropTable(table))
				})
			}
		})
	}
}

func TestMysqlDBNameWithReservedWord(t *testing.T) {
	dbTest := sqlx.NewDatabase("test-name-reserved")
	d := dbTest.OpenDB(mysqlConnector)

	db := d.WithContext(metax.ContextWithMeta(d.Context(), metax.ParseMeta("_id=11111")))
	err := migration.Migrate(db, nil)
	if err != nil {
		t.Fatal(err)
	}

	defer func() {
		dialect := db.Dialect()
		exec := func(expr builder.SqlExpr) error {
			if expr == nil || expr.IsNil() {
				return nil
			}

			_, err := db.ExecExpr(expr)
			return err
		}

		if err := exec(dialect.DropDatabase(d.Name)); err != nil {
			t.Fatal(err)
		}
	}()
}

func TestCRUD(t *testing.T) {
	dbTest := sqlx.NewDatabase("test_crud")

	for _, connector := range []driver.Connector{
		mysqlConnector,
		postgresConnector,
	} {
		t.Run("", func(t *testing.T) {
			d := dbTest.OpenDB(connector)

			db := d.WithContext(metax.ContextWithMeta(d.Context(), metax.ParseMeta("_id=11111")))

			userTable := dbTest.Register(&User{})

			err := migration.Migrate(db, nil)

			NewWithT(t).Expect(err).To(BeNil())

			t.Run("insert single", func(t *testing.T) {
				user := User{
					Name:   uuid.New().String(),
					Gender: GenderMale,
				}

				t.Run("cancel", func(t *testing.T) {
					ctx, cancel := context.WithCancel(Background())
					db2 := db.WithContext(ctx)

					go func() {
						time.Sleep(5 * time.Millisecond)
						cancel()
					}()

					err := sqlx.NewTasks(db2).
						With(
							func(db sqlx.DBExecutor) error {
								_, err := db.ExecExpr(sqlx.InsertToDB(db, &user, nil))
								return err
							},
							func(db sqlx.DBExecutor) error {
								time.Sleep(10 * time.Millisecond)
								return nil
							},
						).
						Do()

					NewWithT(t).Expect(err).NotTo(BeNil())
				})
				_, err := db.ExecExpr(sqlx.InsertToDB(db, &user, nil))
				NewWithT(t).Expect(err).To(BeNil())

				t.Run("update", func(t *testing.T) {
					user.Gender = GenderFemale
					_, err := db.ExecExpr(
						builder.Update(dbTest.T(&user)).
							Set(sqlx.AsAssignments(db, &user)...).
							Where(
								userTable.F("Name").Eq(user.Name),
							),
					)
					NewWithT(t).Expect(err).To(BeNil())
				})
				t.Run("select", func(t *testing.T) {
					userForSelect := User{}
					err := db.QueryExprAndScan(
						builder.Select(nil).From(
							userTable,
							builder.Where(userTable.F("Name").Eq(user.Name)),
							builder.Comment("FindUser"),
						),
						&userForSelect)

					NewWithT(t).Expect(err).To(BeNil())

					NewWithT(t).Expect(user.Name).To(Equal(userForSelect.Name))
					NewWithT(t).Expect(user.Gender).To(Equal(userForSelect.Gender))
				})
				t.Run("conflict", func(t *testing.T) {
					_, err := db.ExecExpr(sqlx.InsertToDB(db, &user, nil))
					NewWithT(t).Expect(sqlx.DBErr(err).IsConflict()).To(BeTrue())
				})
			})
			db.(*sqlx.DB).Tables.Range(func(table *builder.Table, idx int) {
				_, err := db.ExecExpr(db.Dialect().DropTable(table))
				NewWithT(t).Expect(err).To(BeNil())
			})
		})
	}
}

type UserSet map[string]*User

func (UserSet) New() interface{} {
	return &User{}
}

func (u UserSet) Next(v interface{}) error {
	user := v.(*User)
	u[user.Name] = user
	time.Sleep(500 * time.Microsecond)
	return nil
}

func TestSelect(t *testing.T) {
	dbTest := sqlx.NewDatabase("test_for_s")

	for _, connector := range []driver.Connector{
		mysqlConnector,
		postgresConnector,
	} {
		t.Run("", func(t *testing.T) {
			db := dbTest.OpenDB(connector)
			table := dbTest.Register(&User{})

			db.Tables.Range(func(t *builder.Table, idx int) {
				_, _ = db.ExecExpr(db.Dialect().DropTable(t))
			})

			err := migration.Migrate(db, nil)
			NewWithT(t).Expect(err).To(BeNil())

			{
				columns := table.MustFields("Name", "Gender")
				values := make([]interface{}, 0)

				for i := 0; i < 1000; i++ {
					values = append(values, uuid.New().String(), GenderMale)
				}

				_, err := db.ExecExpr(builder.Insert().Into(table).Values(columns, values...))
				NewWithT(t).Expect(err).To(BeNil())
			}

			t.Run("select to slice", func(t *testing.T) {
				users := make([]User, 0)
				err := db.QueryExprAndScan(
					builder.Select(nil).From(table, builder.Where(table.F("Gender").Eq(GenderMale))),
					&users,
				)
				NewWithT(t).Expect(err).To(BeNil())
				NewWithT(t).Expect(users).To(HaveLen(1000))
			})

			t.Run("select to set", func(t *testing.T) {
				userSet := UserSet{}
				err := db.QueryExprAndScan(
					builder.Select(nil).From(table, builder.Where(table.F("Gender").Eq(GenderMale))),
					userSet,
				)
				NewWithT(t).Expect(err).To(BeNil())
				NewWithT(t).Expect(userSet).To(HaveLen(1000))
			})

			t.Run("not found", func(t *testing.T) {
				user := User{}
				err := db.QueryExprAndScan(
					builder.Select(nil).From(
						table,
						builder.Where(table.F("ID").Eq(1001)),
					),
					&user,
				)
				NewWithT(t).Expect(sqlx.DBErr(err).IsNotFound()).To(BeTrue())
			})

			t.Run("count", func(t *testing.T) {
				count := 0
				err := db.QueryExprAndScan(
					builder.Select(builder.Count()).From(table),
					&count,
				)
				NewWithT(t).Expect(err).To(BeNil())
				NewWithT(t).Expect(count).To(Equal(1000))
			})

			t.Run("canceled", func(t *testing.T) {
				ctx, cancel := context.WithCancel(Background())
				db2 := db.WithContext(ctx)

				go func() {
					time.Sleep(3 * time.Millisecond)
					cancel()
				}()

				userSet := UserSet{}
				err := db2.QueryExprAndScan(
					builder.Select(nil).From(table, builder.Where(table.F("Gender").Eq(GenderMale))),
					userSet,
				)
				NewWithT(t).Expect(err).NotTo(BeNil())
			})

			db.Tables.Range(func(tab *builder.Table, idx int) {
				_, _ = db.ExecExpr(db.Dialect().DropTable(tab))
			})
		})
	}
}
