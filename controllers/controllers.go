package controllers

import (
	"net/http"
	"time"

	"github.com/gofiber/fiber/v2"
	"github.com/kobums/playlist/global"
)

// Controller carries the shared response scaffolding for the REST handlers.
//
// The `Vars jet.VarMap` field was dropped: this service renders no templates,
// so it only pulled in the CloudyKit/jet dependency for a map nothing read.
type Controller struct {
	Context *fiber.Ctx
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
	c.Result = make(fiber.Map)
	c.Result["code"] = "ok"
	c.Code = http.StatusOK

	c.Date = global.GetDate(time.Now())

	c.Set("_t", time.Now().UnixNano())
}

func (c *Controller) Set(name string, value any) {
	c.Result[name] = value
}

func (c *Controller) Post(name string) string {
	return c.Context.FormValue(name)
}
