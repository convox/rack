// Convox API Toolkit
//
//   import "github.com/convox/api"
//
//   func main() {
//       // set a logging namespace
//       api.Namespace = "ns=example"
//
//       // activate rollbar
//       api.RollbarToken = os.Getenv("ROLLBAR_TOKEN")
//
//       router := api.NewRouter()
//       router.HandleRedirect("GET", "/", "/users/1")
//       router.HandleText("GET", "/check", "ok")
//       router.HandleApi("GET", "/user/{id}", getUser)
//       router.HandleApi("POST", "/webhook", postWebhook)
//
//       s := api.NewServer()
//       s.UseHandler(router)
//       s.Listen(":5000")
//   }
//
//   func getUser(w http.ResponseWriter, r *http.Request, c api.Context) *api.Error {
//       id := c.Var("id")
//
//       // build up attributes for log lines
//       c.Tagf("id=%q", id)
//
//       user, err := doSomeUserGetting(id)
//
//       // api error type that includes response code
//       if err != nil {
//           return api.ServerError(err)
//       } else if user == nil {
//           return api.Errorf(404, "user not found: %s", id)
//       }
//
//       // built-in logging
//       c.Logf("fn=getUser email=%q", user.Email)
//
//       // built-in handlers that already return appropriate http errors
//       if err := c.WriteJSON(user); err != nil {
//           return err
//       }
//   }
//
//   func postWebhook(w http.ResponseWriter, r *http.Request, c api.Context) *api.Error {
//       var event struct {
//           Foo string
//           Bar string
//       }
//
//       // automatic unmarshalling based on request content type
//       if err := c.UnmarshalBody(&event); err != nil {
//           return err
//       }
//
//       c.Logf("fn=postWebhook foo=%q bar=%q", event.Foo, event.Bar)
//   }
package api

var (
	Namespace    = "ns=api"
	RollbarToken = ""
)
