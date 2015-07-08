package bucket

import (
	"encoding/json"
	. "github.com/empirefox/iniu/comm"
	. "github.com/empirefox/iniu/conf"
	bucketdb "github.com/empirefox/iniu/gorm"
	"github.com/go-martini/martini"
	"github.com/golang/glog"
	"github.com/martini-contrib/binding"
	"github.com/qiniu/api/auth/digest"
	"github.com/qiniu/api/rs"
	"github.com/qiniu/api/rsf"
	"io"
	"net/http"
	"sort"
	"strconv"
	"strings"
	"time"
)

type KindList struct {
	MoveupDirPath  string      `json:"moveup_dir_path"`
	CurrentDirPath string      `json:"current_dir_path"`
	CurrentUrl     string      `json:"current_url"`
	TotalCount     int         `json:"total_count"`
	FileList       []*KindFile `json:"file_list"`
	Order          string      `json:"-"`
}

type KindFile struct {
	IsDir    bool   `json:"is_dir"`
	HasFile  bool   `json:"has_file"`
	IsPhoto  bool   `json:"is_photo"`
	Filesize int64  `json:"filesize"`
	Filetype string `json:"filetype"`
	Filename string `json:"filename"`
	Datetime string `json:"datetime"`
}

func (list *KindList) Len() int {
	return len(list.FileList)
}

func (list *KindList) Less(i, j int) bool {
	fs := list.FileList
	f1, f2 := fs[i], fs[j]
	switch strings.ToUpper(list.Order) {
	case "SIZE":
		return f1.Filesize > f2.Filesize
	case "TYPE":
		return f1.Filetype < f2.Filetype
	default:
		return f1.Filename < f2.Filename
	}
}

func (list *KindList) Swap(i, j int) {
	fs := list.FileList
	fs[i], fs[j] = fs[j], fs[i]
}

type ListReqData struct {
	Bucket   string `form:"bucket" binding:"required"`
	Dir      string `form:"dir" binding:"required"`
	Path     string `form:"path"`
	Order    string `form:"order"`
	Callback string `form:"callback" binding:"required"`
}

func (data *ListReqData) Validate(errors *binding.Errors, req *http.Request) {
	if strings.ToUpper(data.Dir) != "IMAGE" {
		errors.Add([]string{"Dir"}, "ErrorClass", "Dir错误")
	}
	if data.Path != "" {
		data.Path = strings.TrimSuffix(data.Path, "/")
	}
}

//--------------------------------------
//文件管理逻辑
//
//KindEditor提交参数：
//dir{"image", "flash", "media", "file"},默认"image"
//path，默认""，格式："2014年","2014年1月","201401-<filename>"
//order，默认"name"
//
//返回给KindEditor的参数：
//moveup_dir_path": "",
//"current_dir_path": "",
//"current_url": "/ke4/php/../attached/",
//"total_count": 5,
//"file_list": [
//
//{
//    "is_dir": false,
//    "has_file": false,
//    "filesize": 208736,
//    "is_photo": true,
//    "filetype": "jpg",
//    "filename": "1241601537255682809.jpg",
//    "datetime": "2011-08-02 15:32:43"
//},
//{
//    "is_dir": true,
//    "has_file": (file.listFiles() != null),
//    "filesize": 0L,
//    "is_photo": false,
//    "filetype": "",
//    "filename": file.getName(),
//    "datetime": "2011-08-02 15:32:43"
//},
//--------------------------------------
//martini handler
func ListFilesHandlers() []martini.Handler {
	var bind = binding.Bind(ListReqData{})
	var listHandler = func(data ListReqData, w http.ResponseWriter, r *http.Request) {
		//根据path建立KindList
		list := &KindList{CurrentDirPath: data.Path}
		switch length := len([]rune(data.Path)); length {
		case 0: //""
			listYears(r, list)
		case 5: //"2014年"
			listMonths(r, list)
		case 7, 8: //"2014年4月"
			listFiles(&data, list)
		default:
			return
		}
		io.WriteString(w, data.Callback+"(")
		resJson(w, list)
		io.WriteString(w, ")")
	}
	return []martini.Handler{bind, listHandler}
}

//列出图片
func listFiles(data *ListReqData, list *KindList) {
	currPath := list.CurrentDirPath
	t, err := time.Parse("2006年1月", currPath)
	if err != nil {
		glog.Infoln("解析时间错误:", err)
		return
	}
	list.MoveupDirPath = t.Format("2006年")
	prefix := t.Format(IMG_PRE_FMT)

	//取得bucket
	bucket, _ := bucketdb.FindByName(data.Bucket)
	list.CurrentUrl = bucket.ImgBaseUrl()

	//取得图片列表
	list.FileList = make([]*KindFile, 0, 10)
	client := rsf.New(&digest.Mac{bucket.Ak, []byte(bucket.Sk)})
	marker := ""
	limit := 1000

	//    "is_dir": false,
	//    "has_file": false,
	//    "filesize": 208736,
	//    "is_photo": true,
	//    "filetype": "jpg",
	//    "filename": "1241601537255682809.jpg",
	//    "datetime": "2011-08-02 15:32:43"
	var es []rsf.ListItem
	for err == nil {
		es, marker, err = client.ListPrefix(nil, bucket.Name, prefix, marker, limit)
		for _, item := range es {
			f := &KindFile{
				IsDir:    false,
				HasFile:  false,
				IsPhoto:  true,
				Filesize: item.Fsize,
				Filetype: item.MimeType,
				Filename: item.Key,
				Datetime: time.Unix(item.PutTime, 0).Format("2006-01-02 15:04:05"),
			}
			list.FileList = append(list.FileList, f)
		}
	}
	if err != io.EOF {
		//非预期的错误
		glog.Infoln("listAll failed:", err)
	}
	list.TotalCount = len(list.FileList)
	list.Order = data.Order
	sort.Sort(list)
}

//列出月份
func listMonths(r *http.Request, list *KindList) {
	year, err := strconv.Atoi(strings.TrimSuffix(list.CurrentDirPath, "年"))
	if err != nil {
		return
	}

	list.MoveupDirPath = ""
	list.CurrentUrl = r.Host + r.URL.Path
	if year == CurrYear() {
		list.TotalCount = CurrMonth()
	} else {
		list.TotalCount = MONTH_COUNT
	}

	list.FileList = make([]*KindFile, list.TotalCount)
	for i := range list.FileList {
		list.FileList[list.TotalCount-1-i] = &KindFile{
			IsDir:    true,
			HasFile:  true,
			IsPhoto:  false,
			Filesize: int64(0),
			Filetype: "",
			Filename: strconv.Itoa(i+1) + "月",
			Datetime: "",
		}
	}
}

//列出年份
func listYears(r *http.Request, list *KindList) {
	yearFrom := CurrYear()
	list.MoveupDirPath = ""
	list.CurrentUrl = r.Host + r.URL.Path
	list.TotalCount = yearFrom - 2013 + 1

	list.FileList = make([]*KindFile, list.TotalCount)
	for i := range list.FileList {
		list.FileList[i] = &KindFile{
			IsDir:    true,
			HasFile:  true,
			IsPhoto:  false,
			Filesize: int64(0),
			Filetype: "",
			Filename: strconv.Itoa(yearFrom-i) + "年",
			Datetime: "",
		}
	}
}

func resJson(w http.ResponseWriter, v interface{}) {
	r, _ := json.Marshal(v)
	w.Write(r)
}

//----------delete--------------
type DeleteReqData struct {
	Url      string `form:"url" binding:"required"`
	Bucket   string `form:"bucket" binding:"required"`
	Callback string `form:"callback" binding:"required"`
}

func DeleteHandlers() []martini.Handler {
	var bind = binding.Bind(DeleteReqData{})
	var deleteHandler = func(data DeleteReqData, w http.ResponseWriter) {
		index := strings.LastIndex(data.Url, "/")
		runes := []rune(data.Url)
		key := string(runes[index+1:])
		bucket, err := bucketdb.FindByName(data.Bucket)
		rsCli := rs.New(&digest.Mac{bucket.Ak, []byte(bucket.Sk)})
		err = rsCli.Delete(nil, bucket.Name, key)
		if err != nil {
			//产生错误
			glog.Infoln("删除图片错误:", err)
			io.WriteString(w, data.Callback+"({error:1})")
			return
		}
		io.WriteString(w, data.Callback+"({error:0})")
	}
	return []martini.Handler{bind, deleteHandler}
}
