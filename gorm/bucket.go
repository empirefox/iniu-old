package gorm

import (
	"errors"
	"fmt"
	. "github.com/empirefox/iniu/conf"
	"github.com/golang/glog"
	"github.com/jinzhu/gorm"
	_ "github.com/lib/pq"
	"github.com/martini-contrib/binding"
	"github.com/qiniu/api/auth/digest"
	"github.com/qiniu/api/rs"
	"net/http"
	"os"
	"time"
)

var DB gorm.DB

func init() {
	var err error
	DbUrl := os.Getenv("DB_URL")
	if DbUrl == "" {
		panic("数据库环境变量没有正确设置")
	}
	glog.Infoln(DbUrl)
	DB, err = gorm.Open("postgres", DbUrl)
	if err != nil {
		panic(fmt.Sprintf("链接数据库错误: '%v'", err))
	}
	DB.DB().SetMaxIdleConns(5)
	//go1.2
	//DB.DB().SetMaxOpenConns(10)
}

//Bucket:七牛bucket的相关信息
type Bucket struct {
	Id          int64     `json:",omitempty"`
	Name        string    `json:",omitempty" binding:"required" sql:"not null;type:varchar(63);unique"`
	Description string    `json:",omitempty"                    sql:"type:varchar(128)"`
	Ak          string    `json:",omitempty" binding:"required" sql:"not null;type:varchar(100)"`
	Sk          string    `json:",omitempty" binding:"required" sql:"not null;type:varchar(100)"`
	Uptoken     string    `json:",omitempty"                    sql:"not null;type:varchar(300)"`
	Expires     time.Time `json:",omitempty"                    sql:"not null"`
	Life        int64     `json:",omitempty" binding:"required"`
	HasError    bool      `json:",omitempty"`
	Errors      int       `json:",omitempty"`
	CreatedAt   time.Time `json:",omitempty"`
	UpdatedAt   time.Time `json:",omitempty"`
}

//martini binding 包绑定时验证
func (this *Bucket) Validate(errors *binding.Errors, req *http.Request) {
	glog.Infoln(this)
}

//内存中new一个uptoken,没有持久化的,有效期为从现在开始的第X天
func (this *Bucket) NewUptoken() error {
	if this.Name == "" || this.Ak == "" || this.Sk == "" {
		return errors.New("Bucket的Name/Ak/Sk为空，无法生成Uptoken")
	}
	if this.Life == 0 {
		this.Life = 380
	}
	this.Expires = time.Now().Add(time.Duration(this.Life) * DAY)
	putPolicy := rs.PutPolicy{
		Scope:   this.Name,
		Expires: uint32(this.Expires.Unix()),
		//CallbackUrl: callbackUrl,
		//CallbackBody:callbackBody,
		//ReturnUrl:   returnUrl,
		//ReturnBody:  returnBody,
		//AsyncOps:    asyncOps,
		//EndUser:     endUser,
	}
	this.Uptoken = putPolicy.Token(&digest.Mac{this.Ak, []byte(this.Sk)})
	this.HasError = false
	return nil
}

//恢复uptoken
func recUptoken(old string) func(this *Bucket) {
	return func(this *Bucket) {
		if err := recover(); err != nil {
			this.Uptoken = old
		}
	}
}

//更新uptoken,去除Err标志
func (this *Bucket) ReUptoken() {
	defer recUptoken(this.Uptoken)(this)

	err := this.NewUptoken()
	if err != nil {
		panic(err)
	}

	err = this.Save()
	if err != nil {
		panic(err)
	}

	this.NoErr()
}

func Recovery() error {
	DB.DropTable(Bucket{})
	return DB.CreateTable(Bucket{}).Error
}

func AutoMigrate() error {
	return DB.AutoMigrate(Bucket{}).Error
}

func All() (bs []Bucket) {
	DB.Find(&bs)
	return bs
}

func Names() (names []string) {
	DB.Model(&Bucket{}).Pluck("name", &names)
	return names
}

func FindByName(name string) (*Bucket, error) {
	if name == "" {
		return nil, errors.New("Bucket is null")
	}
	bucket := &Bucket{Name: name}
	return bucket, bucket.Find()
}

func Delete(id int64) error {
	return DB.Delete(Bucket{Id: id}).Error
}

//生成img的url
func (this *Bucket) ImgUrl(key string) string {
	return this.ImgBaseUrl() + key
}

//生成img的url前缀
func (this *Bucket) ImgBaseUrl() string {
	return fmt.Sprintf("http://%v.qiniudn.com/", this.Name)
}

//保存
func (this *Bucket) Save() error {
	if this.Uptoken == "" {
		this.NewUptoken()
	}
	return DB.Save(this).Error
}

func (this *Bucket) Find() error {
	return DB.Find(this, this).Error
}

func (this *Bucket) Del() error {
	return DB.Delete(this).Error
}

func (this *Bucket) LogErr() {
	DB.Model(this).UpdateColumns(Bucket{Errors: this.Errors + 1, HasError: true})
}

func (this *Bucket) NoErr() {
	DB.Model(this).UpdateColumns(Bucket{HasError: false})
}
