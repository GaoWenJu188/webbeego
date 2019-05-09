package controllers

import (
	"github.com/astaxie/beego"
	"github.com/astaxie/beego/orm"
	"newsWeb/models"
	"encoding/base64"
)

type UserController struct {
	beego.Controller
}

func (this*UserController) ShowRegister(){
	this.TplName="register.html"
}
func (this*UserController) HandlerRegister(){
	userName:= this.GetString("userName")
	pwd:= this.GetString("password")
	o:=orm.NewOrm()
	var user models.User
	user.Name=userName
	user.Pwd=pwd
	p,err:= o.Insert(&user)
	beego.Error(err)
	beego.Info(p)
	this.TplName="login.html"
}
func (this*UserController)ShowLogin(){
	userName:= this.Ctx.GetCookie("userName")
	dec,_:= base64.StdEncoding.DecodeString(userName)

	if userName!=""{
		this.Data["userName"]=string(dec)
		this.Data["checked"]="checked"
	}else {
		this.Data["userName"]=""
		this.Data["checked"]=""
	}

	this.TplName="login.html"
}
func(this*UserController)HandleLogin(){
	userName:=this.GetString("userName")
	pwd	:= this.GetString("password")
	if userName==""|| pwd==""{
		beego.Error("数据信息不完整")
		this.TplName="login.html"
		return
	}
	o:=orm.NewOrm()
	var user models.User
	user.Name=userName
	err:= o.Read(&user,"Name")
	if err!=nil{
		beego.Error("用户名不正确")
		this.TplName="login.html"
		return
	}
	if pwd!=user.Pwd{
		beego.Error("密码不正确")
		this.TplName="login.html"
		return
	}

	remember:= this.GetString("remember")

	bec:= base64.StdEncoding.EncodeToString([]byte(userName))
	if remember=="on"{
		this.Ctx.SetCookie("userName",bec,60)

	}else {
		this.Ctx.SetCookie("userName",userName,-1)
	}

	this.SetSession("userName",userName)
	this.Redirect("/article/index",302)

}
func (this*UserController)Logout(){
	this.DelSession("userName")
	this.Redirect("/login",302)
}