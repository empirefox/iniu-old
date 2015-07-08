package bucket

import (
	. "github.com/empirefox/iniu/conf"
	"github.com/empirefox/iniu/form"
	db "github.com/empirefox/iniu/gorm"
	"github.com/go-martini/martini"
	"github.com/martini-contrib/render"
	"time"
)

var bucketForm = &form.Form{
	Title: "Bucket",
	Fields: []form.Field{
		{
			Name: "Id",
			Type: "hidden",
		},
		{
			Name:      "Name",
			Required:  true,
			Maxlength: 63,
		},
		{
			Name:      "Description",
			Type:      "kindeditor",
			Maxlength: 128,
			Ops: map[string]interface{}{
				"height": "300px",
				"bucket": "t1-i3-luck2me",
			},
		},
		{
			Name:      "Ak",
			Required:  true,
			Maxlength: 100,
		},
		{
			Name:      "Sk",
			Required:  true,
			Maxlength: 100,
		},
		{
			Name:      "Uptoken",
			Maxlength: 300,
		},
		{
			Name: "Expires",
			Type: "dateTimeLocal",
			Max:  time.Now().Add(time.Duration(750) * DAY).Format("2006-01-02T15:04"),
		},
		{
			Name:     "Life",
			Type:     "number",
			Required: true,
			Max:      "750",
			Min:      "1",
		},
		{
			Name:     "HasError",
			Type:     "checkbox",
			Readonly: true,
		},
		{
			Name:     "Errors",
			Readonly: true,
		},
		{
			Name:     "CreatedAt",
			Type:     "dateTimeLocal",
			Readonly: true,
		},
		{
			Name:     "UpdatedAt",
			Type:     "dateTimeLocal",
			Readonly: true,
		},
	},
	New: &db.Bucket{Name: "新名称", Life: 380},
}

func Form() martini.Handler {
	return func(r render.Render) {
		r.JSON(200, bucketForm)
	}
}
