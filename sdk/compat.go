package sdk

// func (c *Client) BuildCreateMultipart(app string, source []byte, opts structs.BuildCreateOptions) (*structs.Build, error) {
//   ro, err := stdsdk.MarshalOptions(opts)
//   if err != nil {
//     return nil, err
//   }

//   ro.Files = stdsdk.Files{
//     "source": source,
//   }

//   var b *structs.Build

//   if err := c.Post(fmt.Sprintf("/apps/%s/builds", app), ro, &b); err != nil {
//     return nil, err
//   }

//   return b, nil
// }
