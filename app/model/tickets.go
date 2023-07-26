package model

import "time"

type City struct {
	Id       string `json:"id"`
	Name     string `json:"name"`
	Uf       string `json:"uf"`
	State    string `json:"state"`
	UrlKey   string `json:"urlKey"`
	TimeZone string `json:"timeZone"`
}

type Event struct {
	Id           string   `json:"id"`
	Name         string   `json:"name"`
	CarouselSlug string   `json:"carouselSlug"`
	Priority     int      `json:"priority"`
	Type         string   `json:"type"`
	HasLink      bool     `json:"hasLink"`
	Events       []Events `json:"events"`
	Items        []string `json:"items"`
}
type Events struct {
	Id             string  `json:"id"`
	Title          string  `json:"title"`
	OriginalTitle  string  `json:"originalTitle"`
	MovieIdUrl     string  `json:"movieIdUrl"`
	AncineId       string  `json:"ancineId"`
	CountryOrigin  string  `json:"countryOrigin"`
	Priority       int     `json:"priority"`
	ContentRating  string  `json:"contentRating"`
	Duration       string  `json:"duration"`
	Rating         float64 `json:"rating"`
	Synopsis       string  `json:"synopsis"`
	Cast           string  `json:"cast"`
	Director       string  `json:"director"`
	Distributor    string  `json:"distributor"`
	InPreSale      bool    `json:"inPreSale"`
	IsReexhibition bool    `json:"isReexhibition"`
	UrlKey         string  `json:"urlKey"`
	IsPlaying      bool    `json:"isPlaying"`
	CountIsPlaying int     `json:"countIsPlaying"`
	PremiereDate   struct {
		LocalDate   time.Time `json:"localDate"`
		IsToday     bool      `json:"isToday"`
		DayOfWeek   string    `json:"dayOfWeek"`
		DayAndMonth string    `json:"dayAndMonth"`
		Hour        string    `json:"hour"`
		Year        string    `json:"year"`
	} `json:"premiereDate"`
	CreationDate    time.Time `json:"creationDate"`
	City            string    `json:"city"`
	SiteURL         string    `json:"siteURL"`
	NationalSiteURL string    `json:"nationalSiteURL"`
	Images          []struct {
		Url  string `json:"url"`
		Type string `json:"type"`
	} `json:"images"`
	Genres            []string `json:"genres"`
	RatingDescriptors []string `json:"ratingDescriptors"`
	Trailers          []struct {
		Type        string `json:"type"`
		Url         string `json:"url"`
		EmbeddedUrl string `json:"embeddedUrl"`
	} `json:"trailers"`
	Cities []struct {
		Id       string `json:"id"`
		Name     string `json:"name"`
		Uf       string `json:"uf"`
		State    string `json:"state"`
		UrlKey   string `json:"urlKey"`
		TimeZone string `json:"timeZone"`
	} `json:"cities"`
	BoxOfficeId     string `json:"boxOfficeId"`
	PartnershipType string `json:"partnershipType"`
	RottenTomatoe   *struct {
		Id             string `json:"id"`
		CriticsRating  string `json:"criticsRating"`
		CriticsScore   string `json:"criticsScore"`
		AudienceRating string `json:"audienceRating"`
		AudienceScore  string `json:"audienceScore"`
		OriginalUrl    string `json:"originalUrl"`
	} `json:"rottenTomatoe"`
}

type Session struct {
	Theaters []struct {
		Id                string `json:"id"`
		Name              string `json:"name"`
		Address           string `json:"address"`
		AddressComplement string `json:"addressComplement"`
		Number            string `json:"number"`
		UrlKey            string `json:"urlKey"`
		Neighborhood      string `json:"neighborhood"`
		Properties        struct {
			HasBomboniere              bool `json:"hasBomboniere"`
			HasContactlessWithdrawal   bool `json:"hasContactlessWithdrawal"`
			HasSession                 bool `json:"hasSession"`
			HasSeatDistancePolicy      bool `json:"hasSeatDistancePolicy"`
			HasSeatDistancePolicyArena bool `json:"hasSeatDistancePolicyArena"`
		} `json:"properties"`
		Functionalities struct {
			OperationPolicyEnabled bool `json:"operationPolicyEnabled"`
		} `json:"functionalities"`
		DeliveryType    []string `json:"deliveryType"`
		SiteURL         string   `json:"siteURL"`
		NationalSiteURL string   `json:"nationalSiteURL"`
		Enabled         bool     `json:"enabled"`
		BlockMessage    string   `json:"blockMessage"`
		Rooms           string   `json:"rooms"`
		SessionTypes    []struct {
			Type     []string `json:"type"`
			Sessions []struct {
				Id    string   `json:"id"`
				Price float64  `json:"price"`
				Room  string   `json:"room"`
				Type  []string `json:"type"`
				Types []struct {
					Id      int    `json:"id"`
					Name    string `json:"name"`
					Alias   string `json:"alias"`
					Display bool   `json:"display"`
				} `json:"types"`
				Date struct {
					LocalDate   time.Time `json:"localDate"`
					IsToday     bool      `json:"isToday"`
					DayOfWeek   string    `json:"dayOfWeek"`
					DayAndMonth string    `json:"dayAndMonth"`
					Hour        string    `json:"hour"`
					Year        string    `json:"year"`
				} `json:"date"`
				RealDate struct {
					LocalDate   time.Time `json:"localDate"`
					IsToday     bool      `json:"isToday"`
					DayOfWeek   string    `json:"dayOfWeek"`
					DayAndMonth string    `json:"dayAndMonth"`
					Hour        string    `json:"hour"`
					Year        string    `json:"year"`
				} `json:"realDate"`
				Time             string `json:"time"`
				DefaultSector    string `json:"defaultSector"`
				MidnightMessage  string `json:"midnightMessage"`
				SiteURL          string `json:"siteURL"`
				NationalSiteURL  string `json:"nationalSiteURL"`
				HasSeatSelection bool   `json:"hasSeatSelection"`
				DriveIn          bool   `json:"driveIn"`
				Streaming        bool   `json:"streaming"`
				IsNewCheckout    bool   `json:"isNewCheckout"`
				MaxTicketsPerCar int    `json:"maxTicketsPerCar"`
				Enabled          bool   `json:"enabled"`
				BlockMessage     string `json:"blockMessage"`
			} `json:"sessions"`
		} `json:"sessionTypes"`
		Geolocation struct {
			Lat float64 `json:"lat"`
			Lng float64 `json:"lng"`
		} `json:"geolocation"`
		OperationPolicies []string `json:"operationPolicies"`
	} `json:"theaters"`
	Date          string `json:"date"`
	DateFormatted string `json:"dateFormatted"`
	DayOfWeek     string `json:"dayOfWeek"`
	IsToday       bool   `json:"isToday"`
}
