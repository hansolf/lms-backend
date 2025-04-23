package search

import (
	"bytes"
	"encoding/json"
	"fmt"
	"github.com/elastic/go-elasticsearch/v8"
	"github.com/gorilla/mux"
	"net/http"
	"strconv"
)

var ES *elasticsearch.Client

func IndexCourse(course interface{}, id uint) error {
	data, _ := json.Marshal(course)
	res, err := ES.Index(
		"courses",
		bytes.NewReader(data),
		ES.Index.WithDocumentID(fmt.Sprint(id)),
		ES.Index.WithRefresh("true"),
	)
	if err != nil {
		fmt.Println("Ошибка индексации:", err)
		return err
	}
	defer res.Body.Close()
	if res.IsError() {
		fmt.Println("Ошибка ответа от ES:", res.String())
		return fmt.Errorf("elasticsearch error: %s", res.String())
	}
	return nil
}

func DeleteCourseFromIndex(id uint) error {
	res, err := ES.Delete("courses", fmt.Sprint(id), ES.Delete.WithRefresh("true"))
	if err != nil {
		return err
	}
	defer res.Body.Close()
	return nil
}

func IndexLesson(lesson interface{}, id uint) error {
	data, _ := json.Marshal(lesson)
	res, err := ES.Index(
		"lessons",
		bytes.NewReader(data),
		ES.Index.WithDocumentID(fmt.Sprint(id)),
		ES.Index.WithRefresh("true"),
	)
	if err != nil {
		return err
	}
	defer res.Body.Close()
	return nil
}

func DeleteLessonFromIndex(id uint) error {
	res, err := ES.Delete("lessons", fmt.Sprint(id), ES.Delete.WithRefresh("true"))
	if err != nil {
		return err
	}
	defer res.Body.Close()
	return nil
}
func SrchCourDeep(query string) ([]map[string]interface{}, error) {
	var buf bytes.Buffer
	srchQuery := map[string]interface{}{
		"query": map[string]interface{}{
			"bool": map[string]interface{}{
				"should": []interface{}{
					map[string]interface{}{
						"wildcard": map[string]interface{}{
							"Name": map[string]interface{}{
								"value":            "*" + query + "*",
								"case_insensitive": true,
							},
						},
					},
					map[string]interface{}{
						"wildcard": map[string]interface{}{
							"Description": map[string]interface{}{
								"value":            "*" + query + "*",
								"case_insensitive": true,
							},
						},
					},
				},
			},
		},
	}
	json.NewEncoder(&buf).Encode(srchQuery)
	res, err := ES.Search(ES.Search.WithIndex("courses"), ES.Search.WithBody(&buf), ES.Search.WithTrackTotalHits(true))
	if err != nil {
		return nil, err
	}
	defer res.Body.Close()
	var r map[string]interface{}
	json.NewDecoder(res.Body).Decode(&r)
	var results []map[string]interface{}
	if hits, ok := r["hits"].(map[string]interface{}); ok {
		if hitArr, ok := hits["hits"].([]interface{}); ok {
			for _, h := range hitArr {
				if hit, ok := h.(map[string]interface{}); ok {
					source := hit["_source"].(map[string]interface{})
					results = append(results, source)
				}
			}
		}
	}
	if results == nil {
		results = make([]map[string]interface{}, 0)
	}
	return results, nil

}
func SrchCour(query string) ([]map[string]interface{}, error) {
	var buf bytes.Buffer
	searchQuery := map[string]interface{}{
		"query": map[string]interface{}{
			"bool": map[string]interface{}{
				"should": []interface{}{
					map[string]interface{}{
						"wildcard": map[string]interface{}{
							"Name": map[string]interface{}{
								"value":            "*" + query + "*",
								"case_insensitive": true,
							},
						},
					},
				},
			},
		},
	}
	json.NewEncoder(&buf).Encode(searchQuery)
	res, err := ES.Search(
		ES.Search.WithIndex("courses"),
		ES.Search.WithBody(&buf),
		ES.Search.WithTrackTotalHits(true),
	)
	if err != nil {
		return nil, err
	}
	defer res.Body.Close()
	var r map[string]interface{}
	json.NewDecoder(res.Body).Decode(&r)
	var results []map[string]interface{}
	if hits, ok := r["hits"].(map[string]interface{}); ok {
		if hitArr, ok := hits["hits"].([]interface{}); ok {
			for _, h := range hitArr {
				if hit, ok := h.(map[string]interface{}); ok {
					source := hit["_source"].(map[string]interface{})
					results = append(results, source)
				}
			}
		}
	}
	if results == nil {
		results = make([]map[string]interface{}, 0)
	}
	return results, nil
}

func SrchLessonsDeep(courseID uint, query string) ([]map[string]interface{}, error) {
	var buf bytes.Buffer
	srchQuery := map[string]interface{}{
		"query": map[string]interface{}{
			"bool": map[string]interface{}{
				"must": []interface{}{
					map[string]interface{}{
						"term": map[string]interface{}{
							"CourseID": courseID,
						},
					},
					map[string]interface{}{
						"bool": map[string]interface{}{
							"should": []interface{}{
								map[string]interface{}{
									"wildcard": map[string]interface{}{
										"Title": map[string]interface{}{
											"value":            "*" + query + "*",
											"case_insensitive": true,
										},
									},
								},
								map[string]interface{}{
									"wildcard": map[string]interface{}{
										"Description": map[string]interface{}{
											"value":            "*" + query + "*",
											"case_insensitive": true,
										},
									},
								},
							},
						},
					},
				},
			},
		},
	}
	json.NewEncoder(&buf).Encode(srchQuery)
	res, err := ES.Search(
		ES.Search.WithIndex("lessons"),
		ES.Search.WithBody(&buf),
		ES.Search.WithTrackTotalHits(true),
	)
	if err != nil {
		return nil, err
	}
	defer res.Body.Close()
	var r map[string]interface{}
	json.NewDecoder(res.Body).Decode(&r)
	var results []map[string]interface{}
	if hits, ok := r["hits"].(map[string]interface{}); ok {
		if hitArr, ok := hits["hits"].([]interface{}); ok {
			for _, h := range hitArr {
				if hit, ok := h.(map[string]interface{}); ok {
					source := hit["_source"].(map[string]interface{})
					results = append(results, source)
				}
			}
		}
	}
	return results, nil

}
func SrchLessons(courseID uint, query string) ([]map[string]interface{}, error) {
	var buf bytes.Buffer
	searchQuery := map[string]interface{}{
		"query": map[string]interface{}{
			"bool": map[string]interface{}{
				"must": []interface{}{
					map[string]interface{}{
						"term": map[string]interface{}{
							"CourseID": courseID,
						},
					},
					map[string]interface{}{
						"bool": map[string]interface{}{
							"should": []interface{}{
								map[string]interface{}{
									"wildcard": map[string]interface{}{
										"Title": map[string]interface{}{
											"value":            "*" + query + "*",
											"case_insensitive": true,
										},
									},
								},
							},
						},
					},
				},
			},
		},
	}
	json.NewEncoder(&buf).Encode(searchQuery)
	res, err := ES.Search(
		ES.Search.WithIndex("lessons"),
		ES.Search.WithBody(&buf),
		ES.Search.WithTrackTotalHits(true),
	)
	if err != nil {
		return nil, err
	}
	defer res.Body.Close()
	var r map[string]interface{}
	json.NewDecoder(res.Body).Decode(&r)
	var results []map[string]interface{}
	if hits, ok := r["hits"].(map[string]interface{}); ok {
		if hitArr, ok := hits["hits"].([]interface{}); ok {
			for _, h := range hitArr {
				if hit, ok := h.(map[string]interface{}); ok {
					source := hit["_source"].(map[string]interface{})
					results = append(results, source)
				}
			}
		}
	}
	return results, nil
}

func SearchCourses(w http.ResponseWriter, r *http.Request) {
	q := r.URL.Query().Get("q")
	results, err := SrchCour(q)
	if err != nil {
		http.Error(w, "Ошибка поиска", http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(results)
}

func SearchCoursesDeep(w http.ResponseWriter, r *http.Request) {
	q := r.URL.Query().Get("q")
	result, err := SrchCourDeep(q)
	if err != nil {
		http.Error(w, "Ошибка поиска", http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(result)
}

func SearchLessons(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	course := vars["courseID"]
	courseID := parseUint(course)
	q := r.URL.Query().Get("q")
	results, err := SrchLessons(courseID, q)
	if err != nil {
		http.Error(w, "Ошибка поиска", http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(results)
}

func SearchLessonsDeep(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	course := vars["courseID"]
	courseID := parseUint(course)
	q := r.URL.Query().Get("q")
	results, err := SrchLessonsDeep(courseID, q)
	if err != nil {
		http.Error(w, "Ошибка поиска"+err.Error(), http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(results)
}

func parseUint(s string) uint {
	u, err := strconv.ParseUint(s, 10, 64)
	if err != nil {
		return 0
	}
	return uint(u)
}
