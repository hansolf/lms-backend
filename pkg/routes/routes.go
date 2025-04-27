package routes

import (
	"github.com/gorilla/mux"
	"lms-go/pkg/courses"
	"lms-go/pkg/documents"
	"lms-go/pkg/goauth"
	"lms-go/pkg/lessons"
	python_api "lms-go/pkg/python-api"
	"lms-go/pkg/search"
	"lms-go/pkg/tests"
	"lms-go/pkg/usercour"
)

func SetupAuth(h *mux.Router) {
	reg := goauth.ConstructorReg()
	code := goauth.ConstructorCode()
	h.HandleFunc("/api/send/register", reg.SendEmail).Methods("POST")
	h.HandleFunc("/api/auth/verify", code.SignUp).Methods("POST")
	h.HandleFunc("/api/auth/login", goauth.Login).Methods("POST")
	h.HandleFunc("/api/auth/logout", goauth.Logout).Methods("POST")
	h.HandleFunc("/api/verifyteach/{id}", goauth.VerifyTeacher)
}

func SetupMe(h *mux.Router) {
	verify := goauth.Constructor()
	h.HandleFunc("/me", goauth.Me).Methods("GET")
	h.HandleFunc("/me/update", goauth.UpdateMe).Methods("PUT")
	h.HandleFunc("/me/teacher", verify.SendVerTeach).Methods("POST")
}

func SetupCourses(h *mux.Router) {
	h.HandleFunc("/search", search.SearchCourses).Methods("GET")
	h.HandleFunc("/search/deep", search.SearchCoursesDeep).Methods("GET")
	h.HandleFunc("", courses.GetAll).Methods("GET")
	h.HandleFunc("", courses.Create).Methods("POST")
	h.HandleFunc("/{id}", courses.GetByID).Methods("GET")
	h.HandleFunc("/{id}", courses.Update).Methods("PUT")
	h.HandleFunc("/{id}", courses.Delete).Methods("DELETE")
	h.HandleFunc("/{id}/details", courses.GetCourseDetails).Methods("GET")
	h.HandleFunc("/{courseID}/enroll", usercour.EnrollInCourse).Methods("POST")
	h.HandleFunc("/user", courses.GetUserCourses).Methods("GET")
	h.HandleFunc("/status", courses.UpdateCourseStatus).Methods("PUT")
}

func SetupLessons(h *mux.Router) {
	h.HandleFunc("/search", search.SearchLessons).Methods("GET")
	h.HandleFunc("/search/deep", search.SearchLessonsDeep).Methods("GET")
	h.HandleFunc("/lessons/{lessonID}/upload", documents.UploadDoc).Methods("POST")
	h.HandleFunc("/lessons/{lessonID}/download/{id}", documents.DownloadDoc).Methods("GET")
	h.HandleFunc("/lessons", lessons.CreateLes).Methods("POST")
	h.HandleFunc("/lessons/{id}", lessons.GetLesson).Methods("GET")
	h.HandleFunc("/lessons/{id}", lessons.UpdateLes).Methods("PUT")
	h.HandleFunc("/lessons/{id}", lessons.DeleteLesson).Methods("DELETE")
	h.HandleFunc("/lessons", lessons.GetLessons).Methods("GET")
}

func SetupTests(h *mux.Router) {
	h.HandleFunc("/tests", tests.CreateTest).Methods("POST")
	h.HandleFunc("/tests/{id}", tests.GetTest).Methods("GET")
	h.HandleFunc("/tests/{id}", tests.UpdateTest).Methods("PUT")
	h.HandleFunc("/tests/{id}", tests.DeleteTest).Methods("DELETE")
	h.HandleFunc("/tests", tests.GetTests).Methods("GET")
	h.HandleFunc("/tests/{testID}/questions", tests.CreateQ).Methods("POST")
	h.HandleFunc("/tests/{testID}/questions/{id}", tests.GetQ).Methods("GET")
	h.HandleFunc("/tests/{testID}/questions/{id}", tests.UpdateQ).Methods("PUT")
	h.HandleFunc("/tests/{testID}/questions/{id}", tests.DeleteQ).Methods("DELETE")
	h.HandleFunc("/tests/{testID}/questions", tests.GetQs).Methods("GET")
}

func SetupVideos(h *mux.Router) {
	h.HandleFunc("/videos/{vidID}/stream/", python_api.StreamVid).Methods("GET")
	h.HandleFunc("/videos/upload", python_api.UploadVid).Methods("POST")
	h.HandleFunc("/videos/{vidID}/download/", documents.DownloadDoc).Methods("GET")
	h.HandleFunc("/video/{vidID}/summary", python_api.VideoSummary).Methods("GET")
}

func SetupChat(h *mux.Router) {
	chat := python_api.Constructor()
	h.HandleFunc("/chat", chat.AnswerChatBot).Methods("POST")
}
