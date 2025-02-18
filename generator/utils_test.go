package generator

import (
	"os"
	"testing"

	"github.com/kunlun-qilian/sqlx/v3/builder"
	"github.com/onsi/gomega"
)

func init() {
	_ = os.Chdir("./test")
}

func TestParseIndexesFromDoc(t *testing.T) {
	t.Run("parse primary", func(t *testing.T) {
		keys, other := parseKeysFromDoc(`
@def primary ID
summary
desc
`)
		gomega.NewWithT(t).Expect(keys).To(gomega.Equal(&Keys{
			Primary: []string{"ID"},
		}))

		gomega.NewWithT(t).Expect(other).To(gomega.Equal([]string{
			"summary",
			"desc",
		}))
	})
	t.Run("parse index", func(t *testing.T) {
		keys, _ := parseKeysFromDoc(`
@def index I_name   Name
@def index I_nickname/HASH Nickname Name
`)
		gomega.NewWithT(t).Expect(keys).To(gomega.Equal(&Keys{
			Indexes: builder.Indexes{
				"i_name":          []string{"Name"},
				"i_nickname/HASH": []string{"Nickname", "Name"},
			},
		}))
	})
	t.Run("parse all", func(t *testing.T) {
		keys, _ := parseKeysFromDoc(`
@def primary ID
@def index I_nickname/BTREE Nickname
@def index I_username Username
@def index I_geom/SPATIAL Geom
@def unique_index I_name Name
`)
		gomega.NewWithT(t).Expect(keys).To(gomega.Equal(&Keys{
			Primary: []string{"ID"},
			Indexes: builder.Indexes{
				"i_nickname/BTREE": []string{"Nickname"},
				"i_username":       []string{"Username"},
				"i_geom/SPATIAL":   []string{"Geom"},
			},
			UniqueIndexes: builder.Indexes{
				"i_name": []string{"Name"},
			},
		}))
	})
}

func TestParseColRel(t *testing.T) {
	t.Run("rel", func(t *testing.T) {
		rel, others := parseColRelFromComment(`
@rel Account.AccountID

summary

desc
`)
		gomega.NewWithT(t).Expect(rel).To(gomega.Equal("Account.AccountID"))
		gomega.NewWithT(t).Expect(others).To(gomega.Equal([]string{
			"summary",
			"desc",
		}))
	})
}
