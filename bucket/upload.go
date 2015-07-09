package bucket

import (
	"mime/multipart"
	"net/http"
	"strings"
	"text/template"
	"time"

	"github.com/empirefox/iniu-old/comm"
	. "github.com/empirefox/iniu-old/conf"
	bucketdb "github.com/empirefox/iniu-old/gorm"
	"github.com/go-martini/martini"
	"github.com/golang/glog"
	"github.com/martini-contrib/binding"
	qio "github.com/qiniu/api.v6/io"
)

var (
	IframeHtml = `<html>
<head>
    <meta charset="utf-8">
    	<script src="[[.MessengerJs]]"></script>
</head>
<body>
<script>
    var messenger = new Messenger('iframe');
    messenger.addTarget(window.parent, 'parent');
    messenger.targets['parent'].send('[[.UpTime]][[.UpJson | json]]');
    console.log("发送了json");
    location.href='about:blank';
</script>
</body>
</html>`
	MessengerJs = "/lib/messenger.js"
	MsgJsonpTpl *template.Template
)

func init() {
	MsgJsonpTpl, _ = template.New("MsgJsonpTpl").Funcs(template.FuncMap{
		"json": comm.ToJsonFunc,
	}).Delims("[[", "]]").Parse(IframeHtml)
}

type UploadData struct {
	UpTime   string                `form:"up_time"  binding:"required"`
	Dir      string                `form:"dir" 	  binding:"required"`
	Bucket   string                `form:"bucket"   binding:"required"`
	LocalUrl string                `form:"localUrl" binding:"required"`
	ImgFile  *multipart.FileHeader `form:"imgFile"  binding:"required"`
}

func (data *UploadData) Validate(errors *binding.Errors, r *http.Request) {
	data.UpTime = r.FormValue("up_time")
	data.Dir = r.FormValue("dir")
	data.Bucket = r.FormValue("bucket")
	if data.Dir != "IMAGE" {
		errors.Add([]string{"Dir"}, "ErrorClass", "Dir错误")
	}
	//修复无法解析查询问题
	if data.UpTime == "" {
		errors.Add([]string{"up_time"}, "ErrorClass", "required")
	}
	if data.Bucket == "" {
		errors.Add([]string{"bucket"}, "ErrorClass", "required")
	}
}

func (data *UploadData) ImgName() string {
	imgName := data.LocalUrl
	if strings.ContainsAny(imgName, "/\\:") {
		i := strings.LastIndexAny(imgName, "/\\:")
		runes := []rune(imgName)
		imgName = string(runes[i+1:])
	}
	return time.Now().Format(IMG_PRE_FMT) + imgName
}

type UpJson struct {
	Error   int    `json:"error"`
	Url     string `json:"url,omitempty"`
	Message string `json:"message,omitempty"`
}

type UploadRetJsonp struct {
	MessengerJs string
	UpTime      string
	UpJson
}

func resUpJson(w http.ResponseWriter, ret interface{}) {
	err := MsgJsonpTpl.Execute(w, ret)
	if err != nil {
		glog.Infoln("模板调用错误:", err)
	}
}

func (ret *UploadRetJsonp) Respose(w http.ResponseWriter, url string) {
	ret.Url = url
	resUpJson(w, ret)
}

func (ret *UploadRetJsonp) Err(w http.ResponseWriter, err error) {
	ret.Error = 1
	ret.Message = err.Error()
	resUpJson(w, ret)
}

//martini Handler
func UploadHandlers() []martini.Handler {
	var bind martini.Handler = binding.MultipartForm(UploadData{})
	var upload = func(data UploadData, w http.ResponseWriter) {
		retJsonp := &UploadRetJsonp{MessengerJs: MessengerJs, UpTime: data.UpTime}

		//取得bucket
		bucket, err := bucketdb.FindByName(data.Bucket)
		if err != nil {
			retJsonp.Err(w, err)
			return
		}

		imgFile, err := data.ImgFile.Open()
		if err != nil {
			retJsonp.Err(w, err)
			return
		}

		//上传内容到Qiniu
		var ret qio.PutRet
		// ret       	变量用于存取返回的信息，详情见 qio.PutRet
		// uptoken   	为业务服务器端生成的上传口令
		// key:imgName	为文件存储的标识
		// r:imgFile   	为io.Reader类型，用于从其读取数据
		// extra     	为上传文件的额外信息,可为空， 详情见 qio.PutExtra, 可选
		err = qio.Put(nil, &ret, bucket.Uptoken, data.ImgName(), imgFile, nil)
		if err != nil {
			bucket.LogErr()
			retJsonp.Err(w, err)
			return
		}

		//上传成功，返回给KindEditor
		//w.Header().Set("Content-type", "application/json")
		retJsonp.Respose(w, bucket.ImgUrl(ret.Key))
	}
	return []martini.Handler{bind, upload}
}
