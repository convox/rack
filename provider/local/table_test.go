package local_test

// func TestTableList(t *testing.T) {
//   p, err := testProvider()
//   assert.NoError(t, err)
//   defer testProviderCleanup(p)

//   zero, err := p.TableList("app")
//   assert.NoError(t, err)
//   assert.Empty(t, zero)

//   expects := []struct {
//     Name    string
//     Indexes []string
//   }{
//     // orderd list by table name
//     {"apples", []string{"bar"}},
//     {"cherries", []string{"foo", "name"}},
//     {"ninjas", []string{"foo", "bar"}},
//   }

//   for _, tab := range expects {
//     err = p.TableCreate("app", tab.Name, structs.TableCreateOptions{Indexes: tab.Indexes})
//     assert.NoError(t, err)
//   }

//   tables, err := p.TableList("app")
//   assert.NoError(t, err)

//   if assert.Len(t, tables, len(expects)) {
//     for i := range tables {
//       assert.Equal(t, expects[i].Name, tables[i].Name)
//       assert.Equal(t, expects[i].Indexes, tables[i].Indexes)
//     }
//   }
// }

// func TestTableTruncate(t *testing.T) {
//   p, err := testProvider()
//   assert.NoError(t, err)
//   defer testProviderCleanup(p)

//   if err := p.TableCreate("app", "table", structs.TableCreateOptions{Indexes: []string{"data"}}); !assert.NoError(t, err) {
//     assert.FailNow(t, "table create failed")
//   }

//   _, err = p.TableRowStore("app", "table", map[string]string{"data": "foo"})
//   assert.NoError(t, err)

//   _, err = p.TableRowStore("app", "table", map[string]string{"data": "foo"})
//   assert.NoError(t, err)

//   _, err = p.TableRowStore("app", "table", map[string]string{"data": "foo"})
//   assert.NoError(t, err)

//   items, err := p.TableRowsGet("app", "table", []string{"foo"}, structs.TableRowGetOptions{Index: "data"})
//   assert.NoError(t, err)
//   assert.Len(t, items, 3)

//   err = p.TableTruncate("app", "table")
//   assert.NoError(t, err)

//   zero, err := p.TableRowsGet("app", "table", []string{"foo"}, structs.TableRowGetOptions{Index: "data"})
//   assert.NoError(t, err)
//   assert.Empty(t, zero)
// }
