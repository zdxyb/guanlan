package core

import "context"

type Issuer struct {
	ctx context.Context
}

func NewIssuer() *Issuer {
	return &Issuer{}
}

func (i *Issuer) Startup(ctx context.Context) {
	i.ctx = ctx
}

func (i *Issuer) Issue(machine string, days int) *ResponseData {
	code, err := IssueLicense(days, machine)
	if err != nil {
		return &ResponseData{Code: 0, Message: err.Error()}
	}
	return &ResponseData{
		Code:    1,
		Message: "ok",
		Data:    map[string]string{"code": code},
	}
}
