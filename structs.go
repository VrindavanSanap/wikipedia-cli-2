package main

type wikiPage struct {
	ID          int    `json:"id"`
	Key         string `json:"key"`
	Title       string `json:"title"`
	Description string `json:"description"`
	Excerpt     string `json:"excerpt"`
}

type wikiSearchResponse struct {
	Pages []wikiPage `json:"pages"`
}
