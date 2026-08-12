package domain

type ParsedRelease struct {
	Quality    Quality
	Source     string
	Resolution string
	Codec      string
	Year       *int
	Season     *int
	Seasons    []int
	Episodes   []int
	Group      string
	IsRepack   bool
	Complete   bool
}
