package controllers

import (
	"fmt"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/gin-gonic/gin/binding"
	"gorm.io/gorm"

	"rua.plus/gymo/models"
	"rua.plus/gymo/utils"
)

type User struct {
	Db *gorm.DB
}

// 查询用户
type UserQuery struct {
	Email string `form:"email" binding:"required,email"`
}

// 通过 email 查询用户
// 仅支持 query
func (user User) GetUser(c *gin.Context) {
	// response
	resp := &utils.BasicRes{}

	var userInfo UserQuery
	if err := c.ShouldBindWith(&userInfo, binding.Query); err != nil {
		utils.FailedAndReturn(
			c,
			resp,
			http.StatusBadRequest,
			err.Error(),
		)
		return
	}

	u := &models.User{}
	res := user.Db.Model(u).Find(u, "email = ?", userInfo.Email)
	if res.Error != nil {
		utils.FailedAndReturn(
			c,
			resp,
			http.StatusInternalServerError,
			res.Error.Error(),
		)
		return
	}
	if res.RowsAffected == 0 {
		utils.FailedAndReturn(
			c,
			resp,
			http.StatusUnprocessableEntity,
			"user not exist",
		)
		return
	}

	resp.Status = "ok"
	resp.Data = u
	c.JSON(http.StatusOK, resp)
}

// 用户注册
type UserJson struct {
	Username    string `json:"username"    binding:"required"`
	Password    string `json:"password"    binding:"required"`
	Email       string `json:"email"       binding:"required,email"`
	Description string `json:"description"`
	Gender      int8   `json:"gender"`
}

// 添加用户
// 仅支持 json body
func (user User) AddUser(c *gin.Context) {
	// response
	resp := &utils.BasicRes{}

	var userInfo UserJson
	if err := c.ShouldBindWith(&userInfo, binding.JSON); err != nil {
		utils.FailedAndReturn(c, resp, http.StatusBadRequest, err.Error())
		return
	}

	u := &models.User{
		Username:    userInfo.Username,
		Password:    userInfo.Password,
		Email:       userInfo.Email,
		Description: userInfo.Description,
		Gender:      userInfo.Gender,
	}

	res := user.Db.Model(u).Where("email = ?", u.Email).FirstOrCreate(&u)
	if res.Error != nil {
		utils.FailedAndReturn(
			c,
			resp,
			http.StatusInternalServerError,
			res.Error.Error(),
		)
		return
	}
	if res.RowsAffected == 0 {
		utils.FailedAndReturn(
			c,
			resp,
			http.StatusConflict,
			"user already exist",
		)
		return
	}

	resp.Status = "ok"
	resp.Data = u
	c.JSON(http.StatusOK, resp)
}

type UserModify struct {
	Username string `json:"username"`
	Password string `json:"password"`
	Email    string `json:"email"    binding:"email"`
}

func (user User) ModifyUser(c *gin.Context) {
	// response
	resp := &utils.BasicRes{}

	u := utils.GetContextUser(c, resp)

	userInfo := &UserModify{}
	if err := c.ShouldBindWith(&userInfo, binding.JSON); err != nil {
		utils.FailedAndReturn(c, resp, http.StatusBadRequest, err.Error())
		return
	}
	if userInfo.Username != "" {
		u.Username = userInfo.Username
	}
	if userInfo.Email != "" {
		u.Email = userInfo.Email
	}
	if userInfo.Password != "" {
		u.Password = userInfo.Password
		u.HashPassword()
	}

	res := user.Db.Save(u)
	if res.Error != nil {
		utils.FailedAndReturn(
			c,
			resp,
			http.StatusInternalServerError,
			res.Error.Error(),
		)
		return
	}

	resp.Status = "ok"
	resp.Data = u
	c.JSON(http.StatusOK, resp)

}

// 用户登录 json
type UserLogin struct {
	Email    string `json:"email"    binding:"required,email"`
	Password string `json:"password" binding:"required"`
}
type LoginResponse struct {
	*models.User
	Token string `json:"token"`
}

// 用户登录
// 仅支持 json
func (user User) Login(c *gin.Context) {
	// response
	resp := &utils.BasicRes{}

	var userInfo UserLogin
	if err := c.ShouldBindWith(&userInfo, binding.JSON); err != nil {
		utils.FailedAndReturn(c, resp, http.StatusBadRequest, err.Error())
		return
	}

	// query the user
	u := &models.User{}
	dbRes := user.Db.Model(&models.User{}).Find(&u, "email = ?", userInfo.Email)
	if dbRes.Error != nil {
		utils.FailedAndReturn(
			c,
			resp,
			http.StatusInternalServerError,
			dbRes.Error.Error(),
		)
		return
	}
	if dbRes.RowsAffected == 0 {
		utils.FailedAndReturn(
			c,
			resp,
			http.StatusUnprocessableEntity,
			"user not exist",
		)
		return
	}

	// check the password
	if err := models.CheckPasswordHash(userInfo.Password, u.Password); err != nil {
		utils.FailedAndReturn(
			c,
			resp,
			http.StatusUnauthorized,
			"password not correct",
		)
		return
	}

	// generate token
	lastLogin := time.Now().Unix()
	token, err := utils.GenerateToken(int(u.ID), lastLogin)
	if err != nil {
		utils.FailedAndReturn(
			c,
			resp,
			http.StatusInternalServerError,
			err.Error(),
		)
		return
	}

	// update last login
	u.LastLogin = lastLogin
	user.Db.Save(u)

	resp.Status = "ok"
	resp.Data = &LoginResponse{
		User:  u,
		Token: token,
	}
	c.JSON(http.StatusOK, resp)
}

// 当前登录的用户信息
// 通过 Token 获取
func (user User) UserSelf(c *gin.Context) {
	// response
	resp := &utils.BasicRes{}
	u := utils.GetContextUser(c, resp)

	resp.Status = "ok"
	resp.Data = u
	c.JSON(http.StatusOK, resp)
	return
}

// 删除当前用户
func (user User) Delete(c *gin.Context) {
	// response
	resp := &utils.BasicRes{}
	u := utils.GetContextUser(c, resp)

	res := user.Db.Model(&models.User{}).Delete(u, "email = ?", u.Email)
	if res.Error != nil {
		utils.FailedAndReturn(
			c,
			resp,
			http.StatusInternalServerError,
			res.Error.Error(),
		)
		return
	}
	msg := fmt.Sprintf("account %s has been deleted", u.Email)

	resp.Status = "ok"
	resp.Message = msg
	c.JSON(http.StatusOK, resp)
}
