package builder_test

import (
	"reflect"
	"testing"

	"github.com/go-courier/x/ptr"
	"github.com/go-courier/x/types"
	. "github.com/kunlun-qilian/sqlx/v3/builder"
	"github.com/onsi/gomega"
)

func TestColumnTypeFromTypeAndTag(t *testing.T) {
	cases := map[string]*ColumnType{
		`,deprecated=f_target_env_id`: &ColumnType{
			Type:              types.FromRType(reflect.TypeOf(1)),
			DeprecatedActions: &DeprecatedActions{RenameTo: "f_target_env_id"},
		},
		`,autoincrement`: &ColumnType{
			Type:          types.FromRType(reflect.TypeOf(1)),
			AutoIncrement: true,
		},
		`,null`: &ColumnType{
			Type: types.FromRType(reflect.TypeOf(float64(1.1))),
			Null: true,
		},
		`,size=2`: &ColumnType{
			Type:   types.FromRType(reflect.TypeOf("")),
			Length: 2,
		},
		`,decimal=1`: &ColumnType{
			Type:    types.FromRType(reflect.TypeOf(float64(1.1))),
			Decimal: 1,
		},
		`,default='1'`: &ColumnType{
			Type:    types.FromRType(reflect.TypeOf("")),
			Default: ptr.String(`'1'`),
		},
	}

	for tagValue, ct := range cases {
		t.Run(tagValue, func(t *testing.T) {
			gomega.NewWithT(t).Expect(ColumnTypeFromTypeAndTag(ct.Type, tagValue)).To(gomega.Equal(ct))
		})
	}
}
