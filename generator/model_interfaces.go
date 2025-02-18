package generator

import (
	"sort"

	"github.com/go-courier/codegen"
	"github.com/kunlun-qilian/sqlx/v3/builder"
)

func (m *Model) IndexFieldNames() []string {
	indexedFields := make([]string, 0)

	m.Table.Keys.Range(func(key *builder.Key, idx int) {
		fieldNames := key.Def.FieldNames
		indexedFields = append(indexedFields, fieldNames...)
	})

	indexedFields = stringUniq(indexedFields)

	indexedFields = stringFilter(indexedFields, func(item string, i int) bool {
		if m.HasDeletedAt {
			return item != m.FieldKeyDeletedAt
		}
		return true
	})

	sort.Strings(indexedFields)

	return indexedFields
}

func (m *Model) WriteTableName(file *codegen.File) {
	file.WriteBlock(
		codegen.DeclVar(codegen.Var(codegen.Star(
			codegen.Type(file.Use("github.com/kunlun-qilian/sqlx/v3/builder", "Table"))),
			m.VarTable(),
		)),

		codegen.Func().
			Named("init").
			Do(
				codegen.Expr("? = ?.Register(&?{})",
					codegen.Id(m.VarTable()),
					codegen.Id(m.Database),
					codegen.Id(m.StructName),
				),
			),
	)

	file.WriteBlock(
		codegen.DeclType(
			codegen.Var(codegen.Struct(), string(m.IteratorType().Bytes())),
		),
		codegen.Func().
			Named("New").
			MethodOf(codegen.Var(m.IteratorType())).
			Return(codegen.Var(codegen.Interface())).
			Do(
				codegen.Return(codegen.Expr("&?{}", m.Type())),
			),
		codegen.Func(codegen.Var(codegen.Interface(), "v")).
			Named("Resolve").
			MethodOf(codegen.Var(m.IteratorType())).
			Return(codegen.Var(m.PtrType())).
			Do(
				codegen.Return(codegen.Expr("v.(?)", m.PtrType())),
			),
	)

	file.WriteBlock(
		codegen.Func().
			Named("TableName").
			MethodOf(codegen.Var(m.Type())).
			Return(codegen.Var(codegen.String)).
			Do(
				codegen.Return(file.Val(m.Config.TableName)),
			),
	)
}

func (m *Model) WriteTableInterfaces(file *codegen.File) {
	if m.Description != nil {
		file.WriteBlock(
			codegen.Func().
				Named("TableDescription").
				MethodOf(codegen.Var(m.Type())).
				Return(codegen.Var(codegen.Slice(codegen.String))).
				Do(
					codegen.Return(file.Val(m.Description)),
				),
		)
	}

	file.WriteBlock(
		codegen.Func().
			Named("ColDescriptions").
			MethodOf(codegen.Var(m.Type())).
			Return(codegen.Var(codegen.Map(codegen.String, codegen.Slice(codegen.String)))).
			Do(
				codegen.Return(file.Val(m.GetColDescriptions())),
			),
	)

	m.Columns.Range(func(col *builder.Column, idx int) {
		if col.DeprecatedActions != nil {
			return
		}

		file.WriteBlock(
			codegen.Func().
				Named("FieldKey" + col.FieldName).
				MethodOf(codegen.Var(m.Type())).
				Return(codegen.Var(codegen.String)).
				Do(
					codegen.Return(file.Val(col.FieldName)),
				),
		)

		file.WriteBlock(
			codegen.Func().
				Named("Field" + col.FieldName).
				MethodOf(codegen.Var(m.PtrType(), "m")).
				Return(codegen.Var(codegen.Star(codegen.Type(file.Use("github.com/kunlun-qilian/sqlx/v3/builder", "Column"))))).
				Do(
					codegen.Return(codegen.Expr("?.F(m.FieldKey"+col.FieldName+"())", codegen.Id(m.VarTable()))),
				),
		)
	})

	file.WriteBlock(
		codegen.Func().
			Named("ColRelations").
			MethodOf(codegen.Var(m.Type())).
			Return(codegen.Var(codegen.Map(codegen.String, codegen.Slice(codegen.String)))).
			Do(
				codegen.Return(file.Val(m.GetRelations())),
			),
	)

	file.WriteBlock(
		codegen.Func().
			Named("IndexFieldNames").
			MethodOf(codegen.Var(m.PtrType(), "m")).
			Return(codegen.Var(codegen.Slice(codegen.String))).
			Do(
				codegen.Return(file.Val(m.IndexFieldNames())),
			),
	)

	if !m.Config.WithMethods {
		return
	}

	file.WriteBlock(
		codegen.Func(
			codegen.Var(codegen.Type(file.Use("github.com/kunlun-qilian/sqlx/v3", "DBExecutor")), "db"),
		).
			Named("ConditionByStruct").
			MethodOf(codegen.Var(m.PtrType(), "m")).
			Return(codegen.Var(codegen.Type(file.Use("github.com/kunlun-qilian/sqlx/v3/builder", "SqlCondition")))).
			Do(
				codegen.Expr(`table := db.T(m)`),
				codegen.Expr(`fieldValues := `+file.Use("github.com/kunlun-qilian/sqlx/v3/builder", "FieldValuesFromStructByNonZero")+`(m)

conditions := make([]`+file.Use("github.com/kunlun-qilian/sqlx/v3/builder", "SqlCondition")+`, 0)

for _, fieldName := range m.IndexFieldNames() {
	if v, exists := fieldValues[fieldName]; exists {
		conditions = append(conditions, table.F(fieldName).Eq(v))
		delete(fieldValues, fieldName)
	}
}

if len(conditions) == 0 {
	panic(`+file.Use("fmt", "Errorf")+`("at least one of field for indexes has value"))
}
	
for fieldName, v := range fieldValues {
	conditions = append(conditions, table.F(fieldName).Eq(v))
}
	
condition := `+file.Use("github.com/kunlun-qilian/sqlx/v3/builder", "And")+`(conditions...)
`),

				func() codegen.Snippet {
					if m.HasDeletedAt {
						return codegen.Expr(
							`condition = `+file.Use("github.com/kunlun-qilian/sqlx/v3/builder", "And")+`(condition, table.F(?).Eq(0))`,
							file.Val(m.FieldKeyDeletedAt),
						)
					}
					return codegen.Expr("")
				}(),

				codegen.Return(codegen.Id("condition")),
			),
	)
}

func (m *Model) WriteTableKeyInterfaces(file *codegen.File) {
	if len(m.Keys.Primary) > 0 {
		file.WriteBlock(
			codegen.Func().
				Named("PrimaryKey").
				MethodOf(codegen.Var(m.Type())).
				Return(codegen.Var(codegen.Slice(codegen.String))).
				Do(
					codegen.Return(file.Val(m.Keys.Primary)),
				),
		)
	}

	// 增加分区表支持
	if len(m.Keys.Partition) > 0 {
		file.WriteBlock(
			codegen.Func().
				Named("Partition").
				MethodOf(codegen.Var(m.Type())).
				Return(codegen.Var(codegen.Slice(codegen.String))).
				Do(
					codegen.Return(file.Val(m.Keys.Partition)),
				),
		)
	}

	if len(m.Keys.Indexes) > 0 {

		file.WriteBlock(
			codegen.Func().
				Named("Indexes").
				MethodOf(codegen.Var(m.Type())).
				Return(codegen.Var(codegen.Type(file.Use("github.com/kunlun-qilian/sqlx/v3/builder", "Indexes")))).
				Do(
					codegen.Return(file.Val(m.Keys.Indexes)),
				),
		)
	}

	if len(m.Keys.UniqueIndexes) > 0 {
		indexKeys := make([]string, 0)

		for k := range m.Keys.UniqueIndexes {
			indexKeys = append(indexKeys, k)
		}

		sort.Strings(indexKeys)

		for _, k := range indexKeys {
			file.WriteBlock(
				codegen.Func().
					Named("UniqueIndex" + codegen.UpperCamelCase(k)).
					MethodOf(codegen.Var(m.Type())).
					Return(codegen.Var(codegen.String)).
					Do(
						codegen.Return(file.Val(k)),
					),
			)
		}

		file.WriteBlock(
			codegen.Func().
				Named("UniqueIndexes").
				MethodOf(codegen.Var(m.Type())).
				Return(codegen.Var(codegen.Type(file.Use("github.com/kunlun-qilian/sqlx/v3/builder", "Indexes")))).
				Do(
					codegen.Return(file.Val(m.Keys.UniqueIndexes)),
				),
		)
	}

	if m.WithComments {
		file.WriteBlock(
			codegen.Func().
				Named("Comments").
				MethodOf(codegen.Var(m.Type())).
				Return(codegen.Var(codegen.Map(codegen.String, codegen.String))).
				Do(
					codegen.Return(file.Val(m.GetComments())),
				),
		)
	}
}

func (m *Model) GetRelations() map[string][]string {
	rels := map[string][]string{}
	m.Columns.Range(func(col *builder.Column, idx int) {
		if len(col.Relation) == 2 {
			rels[col.FieldName] = col.Relation
		}
	})
	return rels
}

func (m *Model) GetComments() map[string]string {
	comments := map[string]string{}
	m.Columns.Range(func(col *builder.Column, idx int) {
		if col.Comment != "" {
			comments[col.FieldName] = col.Comment
		}
	})
	return comments
}

func (m *Model) GetColDescriptions() map[string][]string {
	descriptions := map[string][]string{}
	m.Columns.Range(func(col *builder.Column, idx int) {
		if col.Description != nil {
			descriptions[col.FieldName] = col.Description
		}
	})
	return descriptions
}
