package models

type ReqPaginator struct {
	PageNum  int    `default:"1" query:"PageNum" validate:"gte=0"`
	PageSize int    `default:"10" query:"PageSize" validate:"required,min=1,max=100000"`
	Sort     string `default:"id desc" query:"Sort" validate:"omitempty"`
}

func (p *ReqPaginator) FillDefaults() {
	if p.PageNum == 0 {
		p.PageNum = 1
	}
	if p.PageSize == 0 {
		p.PageSize = 10
	}
	if p.Sort == "" {
		p.Sort = "id desc"
	}
}

type RspPaginator struct {
	PageNum  int
	PageSize int
	Total    int64
	List     any
}
