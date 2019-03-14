package k8s_test

// func testProvider(t *testing.T, fn func(*k8s.Provider, *fakek8s.Clientset)) {
//   c := fakek8s.NewSimpleClientset()

//   p, err := k8s.FromEnv()
//   require.NoError(t, err)

//   p.Cluster = c
//   p.Rack = "test"

//   fn(p, c)
// }

// func testProviderManual(t *testing.T, fn func(*k8s.Provider, *fakek8s.Clientset)) {
//   c := &fakek8s.Clientset{}

//   p := &k8s.Provider{
//     Cluster: c,
//     Rack:    "test",
//   }

//   fn(p, c)
// }
