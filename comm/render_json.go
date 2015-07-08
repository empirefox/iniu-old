package comm

import (
	"github.com/martini-contrib/render"
	"net/http"
)

func JsonOk(r render.Render) {
	Json(r, 0, "")
}

func JsonContent(r render.Render, content interface{}) {
	Json(r, 0, content)
}

func JsonErr(r render.Render, err error) {
	Json(r, 1, err)
}

func JsonErrState(r render.Render) {
	Json(r, 1, "")
}

func Json(r render.Render, state int, content interface{}) {
	r.JSON(http.StatusOK, map[string]interface{}{"error": state, "content": content})
}
