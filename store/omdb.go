package store

import (
	"time"
)

const omdbCacheTTL = 30 * 24 * time.Hour // 30 dias de cache, pois metadados de filmes mudam raramente

// OMDbRating representa os dados essenciais que queremos salvar do filme
type OMDbRating struct {
	ImdbRating string `json:"imdbRating"`
	Metascore  string `json:"metascore"`
	Rotten     string `json:"rotten"`
	Genre      string `json:"genre"`
	Director   string `json:"director"`
	Plot       string `json:"plot"`
	NotFound   bool   `json:"not_found"`
}

type omdbCacheMap map[string]OMDbRating

// LoadMovieRating busca as informações do filme no cache local
func LoadMovieRating(title string) (OMDbRating, bool) {
	path, err := cachePath("omdb_ratings.json")
	if err != nil {
		return OMDbRating{}, false
	}

	cache, err := loadCache[omdbCacheMap](path)
	if err != nil || cache.Data == nil {
		return OMDbRating{}, false
	}

	// Como notas não expiram do dia para a noite, podemos usar um TTL longo
	if time.Since(cache.UpdatedAt) > omdbCacheTTL {
		return OMDbRating{}, false
	}

	rating, exists := cache.Data[title]
	return rating, exists
}

// SaveMovieRating atualiza o dicionário local com os dados recém-buscados
func SaveMovieRating(title string, rating OMDbRating) error {
	path, err := cachePath("omdb_ratings.json")
	if err != nil {
		return err
	}

	cache, err := loadCache[omdbCacheMap](path)
	var data omdbCacheMap

	if err == nil && cache.Data != nil {
		// Se o cache já passou do TTL, nós limpamos e começamos um novo para evitar lixo eterno
		if time.Since(cache.UpdatedAt) > omdbCacheTTL {
			data = make(omdbCacheMap)
		} else {
			data = cache.Data
		}
	} else {
		data = make(omdbCacheMap)
	}

	data[title] = rating

	return saveCache(path, data)
}
