package domain

type Banner struct {
	ID          int64
	Description string
}

type Slot struct {
	ID          int64
	Description string
}

type SocialGroup struct {
	ID          int64
	Description string
}

type Stat struct {
	SlotID   int64
	BannerID int64
	GroupID  int64
	Shows    int64
	Clicks   int64
}
