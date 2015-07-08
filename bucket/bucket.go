package bucket

import (
	"github.com/empirefox/iniu/comm"
	db "github.com/empirefox/iniu/gorm"
	"github.com/go-martini/martini"
	"github.com/golang/glog"
	"github.com/martini-contrib/render"
)

var (
	DB      = db.DB
	names   []string
	buckets []db.Bucket
)

func initNames() {
	names = []string{}
	DB.Model(&db.Bucket{}).Pluck("name", &names)
}

func initBuckets() {
	var bs []db.Bucket
	DB.Find(&bs)
	buckets = bs
}

func Names() []string {
	if names == nil {
		initNames()
	}
	return names
}

func Buckets() []db.Bucket {
	if buckets == nil {
		initBuckets()
	}
	return buckets
}

func NameList() martini.Handler {
	return func(r render.Render) {
		comm.JsonContent(r, Names())
	}
}

func List() martini.Handler {
	return func(r render.Render) {
		comm.JsonContent(r, Buckets())
	}
}

func One() martini.Handler {
	return func(r render.Render, params martini.Params) {
		bucket, err := db.FindByName(params["name"])
		if err != nil {
			comm.JsonErr(r, err)
		} else {
			comm.JsonContent(r, bucket)
		}
	}
}

func Save() martini.Handler {
	return func(data db.Bucket, r render.Render) {
		err := DB.Save(&data).Error
		if err != nil {
			comm.JsonErr(r, err)
		} else {
			initBuckets()
			comm.JsonContent(r, &data)
		}
	}
}

func Remove() martini.Handler {
	return func(data db.Bucket, r render.Render) {
		err := DB.Delete(&data).Error
		if err != nil {
			comm.JsonErr(r, err)
		} else {
			//可综合考虑slice删除方式
			initBuckets()
			comm.JsonOk(r)
		}
	}
}

func Recovery() martini.Handler {
	return func(r render.Render) {
		err := db.Recovery()
		if err != nil {
			glog.Errorln(err)
			comm.JsonErr(r, err)
		} else {
			comm.JsonOk(r)
		}
	}
}

func AutoMigrate() martini.Handler {
	return func(r render.Render) {
		err := db.AutoMigrate()
		if err != nil {
			glog.Errorln(err)
			comm.JsonErr(r, err)
		} else {
			comm.JsonOk(r)
		}
	}
}
