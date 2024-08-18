package controllers

import (
	"net/http"
	"time"

	"github.com/kobums/playlist/global"

	"github.com/CloudyKit/jet/v3"
	"github.com/gofiber/fiber/v2"
)

type Controller struct {
	Context *fiber.Ctx
	Vars    jet.VarMap
	Result  fiber.Map
	Current string
	Code    int

	Date string

	Page     int
	Pagesize int
}

func NewController(ctx *fiber.Ctx) *Controller {
	var ctl Controller
	ctl.Init(ctx)
	return &ctl
}

func (c *Controller) Init(ctx *fiber.Ctx) {
	c.Context = ctx
	c.Vars = make(jet.VarMap)
	c.Result = make(fiber.Map)
	c.Result["code"] = "ok"
	c.Code = http.StatusOK

	c.Date = global.GetDate(time.Now())

	c.Set("_t", time.Now().UnixNano())
}

func (c *Controller) Set(name string, value interface{}) {
	c.Result[name] = value
}

func (c *Controller) Post(name string) string {
	return c.Context.FormValue(name)
}
