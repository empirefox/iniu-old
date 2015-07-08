//	m.Use(oauth.Prepare("/", func validate(name string) bool {
//	  return googleNames.Contains(name)
//  }))
//
//  oauth.CheckPrev()
package oauth

import (
	"github.com/bradrydzewski/go.auth"
	"github.com/dchest/uniuri"
	"github.com/go-martini/martini"
	"github.com/golang/glog"
	"net/http"
	"net/url"
	"reflect"
)

func init() {

}

var (
	PathLogout = "/logout"
	OpenId     = auth.OpenId(auth.GoogleOpenIdEndpoint)
)

type ValidateType interface{}

type ValidateFunc func(string) bool

func getValidateFunc(vType ValidateType) ValidateFunc {
	if fn, ok := vType.(ValidateFunc); ok {
		return fn
	}
	if fn, ok := vType.(func(string) bool); ok {
		return fn
	}
	glog.Infoln(reflect.ValueOf(vType))
	panic("ValidateType must be a callable ValidateFunc")
}

func NilValidate(name string) bool {
	return false
}

var Prepare = func(okPath string, v ValidateFunc) martini.Handler {
	if v == nil {
		v = NilValidate
	}
	auth.Config.CookieSecret = []byte(uniuri.New())
	auth.Config.LoginSuccessRedirect = okPath
	auth.Config.CookieSecure = martini.Env == martini.Prod

	return func(c martini.Context, w http.ResponseWriter, r *http.Request) {
		if r.Method == "GET" {
			switch r.URL.Path {
			case auth.Config.LoginRedirect:
				OpenId.ServeHTTP(w, r)
			case PathLogout:
				Logout(w, r)
			}
		}
		c.MapTo(v, (*ValidateType)(nil))
	}
}

var PrepareNoValidate = func(okPath string) martini.Handler {
	return Prepare(okPath, nil)
}

func Logout(w http.ResponseWriter, r *http.Request) {
	auth.DeleteUserCookie(w, r)
	http.Redirect(w, r, "/", http.StatusSeeOther)
}

//只用当前ValidateFunc验证，不验证Prepare
var OnlyIn = func(v ValidateFunc) martini.Handler {
	return func(c martini.Context, w http.ResponseWriter, r *http.Request) {
		innerCheck(c, v, nil, w, r)
	}
}

//只用Prepare中的ValidateFunc验证
var CheckPrev = func() martini.Handler {
	return func(c martini.Context, vType ValidateType, w http.ResponseWriter, r *http.Request) {
		innerCheck(c, nil, vType, w, r)
	}
}

//Prepare中和当前的ValidateFunc验证
var CheckAll = func(v ValidateFunc) martini.Handler {
	return func(c martini.Context, vType ValidateType, w http.ResponseWriter, r *http.Request) {
		innerCheck(c, v, vType, w, r)
	}
}

func innerCheck(c martini.Context, v ValidateFunc, vType ValidateType, w http.ResponseWriter, r *http.Request) {
	user, err := auth.GetUserCookie(r)

	if err != nil || user.Id() == "" {
		http.Redirect(w, r, auth.Config.LoginRedirect, http.StatusFound)
		return
	}

	username := user.Id()
	//一方验证通过即放行
	if (v != nil && v(username)) || (vType != nil && getValidateFunc(vType)(username)) {
		r.URL.User = url.User(user.Id())
		return
	}
	Logout(w, r)
	return
}
