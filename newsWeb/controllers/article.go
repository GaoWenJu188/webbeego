package controllers

import (
	"github.com/astaxie/beego"
	"path"
	"time"
	"github.com/astaxie/beego/orm"
	"newsWeb/models"
	"math"
	"strconv"

	"github.com/gomodule/redigo/redis"
	"bytes"
	"encoding/gob"
)

type ArticleController struct {
	beego.Controller
}

func (this *ArticleController) ShowIndex() {
	userName := this.GetSession("userName")
	if userName == nil {
		this.Redirect("/login", 302)
		return
	}
	this.Data["userName"] = userName.(string)
	o := orm.NewOrm()
	qs := o.QueryTable("Article")
	var articles []models.Article
	//qs.All(&articles)
	//this.Data["articles"]=articles
	//获取总个数
	//关联表或获取相对应的数据
	typeName := this.GetString("select")
	var count int64
	if typeName == "" {
		count, _ = qs.RelatedSel("ArticleType").Count()
	} else {
		count, _ = qs.RelatedSel("ArticleType").Filter("ArticleType__TypeName", typeName).Count()
	}
	pageIndex := 2
	pageCount := math.Ceil(float64(count) / float64(pageIndex))
	pageNum, err := this.GetInt("pageNum")
	if err != nil {
		pageNum = 1
	}
	if typeName == "" {
		qs.Limit(pageIndex, pageIndex*(pageNum-1)).RelatedSel("ArticleType").All(&articles)
	} else {
		qs.Limit(pageIndex, pageIndex*(pageNum-1)).RelatedSel("ArticleType").Filter("ArticleType__TypeName", typeName).All(&articles)
	}
	var articleTypes []models.ArticleType
	//查找的时候先从redis中查找，没有就是第一次，要创建新的
	conn, err := redis.Dial("tcp", ":6379")
	if err != nil {
		beego.Error("redis没有连接上", err)
		return
	}
	defer conn.Close()
	resp, err := conn.Do("get", "articleType")
	result, _ := redis.Bytes(resp, err)
	if len(result) == 0 {
		o.QueryTable("ArticleType").All(&articleTypes)
		//把数据存储到redis中
		//因为是对象，要编码
		//首先创建容器存储
		var buffer bytes.Buffer
		enc:= gob.NewEncoder(&buffer)
		enc.Encode(articleTypes)
		conn.Do("set","articleType",buffer.Bytes())
		beego.Info("将mysql中获取数据存储到redis中")
	}else {
		//解码
		dec:= gob.NewDecoder(bytes.NewReader(result))
		dec.Decode(&articleTypes)
		beego.Info(articleTypes)
		beego.Info("从redsi中获取数据")
	}
	this.Data["articleTypes"] = articleTypes
	this.Data["articles"] = articles
	this.Data["count"] = count
	this.Data["pageCount"] = pageCount
	this.Data["pageNum"] = pageNum
	this.Data["TypeName"] = typeName
	this.TplName = "index.html"
}
func (this *ArticleController) ShowAddArtical() {
	o := orm.NewOrm()
	var articlesTypes []models.ArticleType
	o.QueryTable("ArticleType").All(&articlesTypes)
	this.Data["articlesTypes"] = articlesTypes
	this.Layout = "layout.html"
	this.TplName = "add.html"
}
func (this *ArticleController) HandlerAddArtical() {
	userName := this.GetSession("userName")
	if userName == nil {
		this.Redirect("/login", 302)
		return
	}
	this.Data["userName"] = userName.(string) //断言
	articleName := this.GetString("articleName")
	content := this.GetString("content")
	typeName := this.GetString("select")
	if articleName == "" || content == "" || typeName == "" {
		beego.Error("获取数据错误")
		this.Data["errmsg"] = "获取数据错误"
		this.Layout = "layout.html"
		this.TplName = "add.html"
		return
	}
	file, head, err := this.GetFile("uploadname")
	if err != nil {
		beego.Error("获取图片错误")
		this.Data["errmsg"] = "获取图片错误"
		this.Layout = "layout.html"
		this.TplName = "add.html"
		return
	}
	defer file.Close()
	if head.Size > 5000000 {
		beego.Error("图片过大")
		this.Data["errmsg"] = "图片过大"
		this.Layout = "layout.html"
		this.TplName = "add.html"
		return
	}
	ext := path.Ext(head.Filename)
	if ext != ".jpg" && ext != ".png" && ext != ".jpeg" {
		beego.Error("图片格式错误")
		this.Data["errmsg"] = "图片格式错误"
		this.Layout = "layout.html"
		this.TplName = "add.html"
		return
	}
	fileName := time.Now().Format("200601021504052222")
	this.SaveToFile("uploadname", "./static/img/"+fileName+ext)
	//处理数据
	o := orm.NewOrm()
	var artical models.Article
	artical.Title = articleName
	artical.Content = content
	//因为artical.ArticleType 的类型是一个对象，所以要农一个对象出来复制
	var articleType models.ArticleType
	articleType.TypeName = typeName
	//奥的是对象，所以要先检查是否存在
	o.Read(&articleType, "TypeName")
	artical.ArticleType = &articleType

	artical.Img = "/static/img/" + fileName + ext

	_, err = o.Insert(&artical)
	if err != nil {
		beego.Error("数据错误", err)
		this.Data["errmsg"] = "数据插入失败"
		this.Layout = "layout.html"
		this.TplName = "add.html"
		return
	}
	this.Redirect("/article/index", 302)
}

func (this *ArticleController) ShowContent() {
	userName := this.GetSession("userName")
	if userName == nil {
		this.Redirect("/login", 302)
		return
	}
	this.Data["userName"] = userName.(string) //断言
	id, err := this.GetInt("id")
	if err != nil {
		beego.Error("获取文章id出错")
		this.Redirect("/article/index", 302)
		return
	}
	o := orm.NewOrm()
	var article models.Article
	article.Id = id
	o.Read(&article)

	var users []models.User
	o.QueryTable("User").Filter("Articles__Article__Id", id).Distinct().All(&users)
	this.Data["users"] = users

	article.ReadCount += 1
	o.Update(&article)
	this.Data["article"] = article

	userName = this.GetSession("userName")
	var user models.User
	user.Name = userName.(string)
	o.Read(&user, "Name")

	m2m := o.QueryM2M(&article, "Users")
	m2m.Add(user)

	var articleTypes models.ArticleType
	o.QueryTable("ArticleType").All(&articleTypes)
	this.Data["articleTypes"] = articleTypes

	this.Layout = "layout.html"
	this.TplName = "content.html"
}

func (this *ArticleController) ShowUpdata() {
	id, err := this.GetInt("id")
	if err != nil {
		beego.Error("编辑文章id获取失败")
		this.Redirect("/article/index", 302)
		return
	}
	o := orm.NewOrm()
	var article models.Article
	article.Id = id
	o.Read(&article)
	this.Data["article"] = article
	this.TplName = "update.html"

}
func UploadFile(this *ArticleController, filePath string, errHtml string) string {
	//获取图片
	//返回值 文件二进制流  文件头    错误信息
	file, head, err := this.GetFile(filePath)
	if err != nil {
		beego.Error("获取数据错误")
		this.Data["errmsg"] = "图片上传失败"
		this.TplName = errHtml
		return " "
	}
	defer file.Close()
	//校验文件大小
	if head.Size > 5000000 {
		beego.Error("获取数据错误")
		this.Data["errmsg"] = "图片数据过大"
		this.TplName = errHtml
		return " "
	}

	//校验格式 获取文件后缀
	ext := path.Ext(head.Filename)
	if ext != ".jpg" && ext != ".png" && ext != ".jpeg" {
		beego.Error("获取数据错误")
		this.Data["errmsg"] = "上传文件格式错误"
		this.TplName = errHtml
		return " "
	}

	//防止重名
	fileName := time.Now().Format("200601021504052222")

	//jianhuangcaozuo

	//把上传的文件存储到项目文件夹
	this.SaveToFile(filePath, "./static/img/"+fileName+ext)
	return "/static/img/" + fileName + ext
}
func (this *ArticleController) HandleUpdate() {
	articleName := this.GetString("articleName")
	content := this.GetString("content")
	savePath := UploadFile(this, "uploadname", "update.html")
	id, _ := this.GetInt("id")
	if articleName == " " || content == " " || savePath == " " {
		beego.Error("获取数据失败")
		this.Redirect("/article/update?id="+strconv.Itoa(id), 302)
		return
	}
	o := orm.NewOrm()
	var article models.Article
	article.Id = id
	//一定要先查询
	o.Read(&article)
	article.Title = articleName
	article.Content = content
	article.Img = savePath
	o.Update(&article)
	//返回数据
	this.Redirect("/article/index", 302)
}

func (this *ArticleController) HandleDelete() {
	id, err := this.GetInt("id")
	if err != nil {
		beego.Error("文件id获取失败")
		this.Redirect("/article/index", 302)
		return
	}
	o := orm.NewOrm()
	var article models.Article
	article.Id = id
	o.Delete(&article, "Id")
	this.Redirect("/article/index", 302)
}

func (this *ArticleController) ShowAddType() {
	o := orm.NewOrm()
	var articleTypes []models.ArticleType
	o.QueryTable("ArticleType").All(&articleTypes)
	this.Data["articleTypes"] = articleTypes
	this.Layout = "layout.html"
	this.TplName = "addType.html"
}

//处理添加类型请求
func (this *ArticleController) HandleAddType() {
	typeName := this.GetString("typeName")
	if typeName == "" {
		beego.Error("获取名称传输失败")
		this.Redirect("/article/addType", 302)
		return
	}
	o := orm.NewOrm()
	var articleType models.ArticleType
	articleType.TypeName = typeName
	o.Insert(&articleType)

	this.Redirect("/article/addType", 302)
}

func (this *ArticleController) DeleteType() {
	id, err := this.GetInt("id")
	if err != nil {
		beego.Error("获取type id 失败", err)
		this.Redirect("/article/addType", 302)
		return
	}
	o := orm.NewOrm()
	var articleType models.ArticleType
	articleType.Id = id
	o.Delete(&articleType)
	this.Redirect("/article/addType", 302)

}
