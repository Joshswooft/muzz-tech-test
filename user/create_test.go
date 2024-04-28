package user

import (
	"database/sql"
	"encoding/json"
	"muzz/store"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"
	"time"

	_ "github.com/mattn/go-sqlite3"
)

func TestCreateUserHandler(t *testing.T) {
	db, err := sql.Open("sqlite3", "./test.db")
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	defer os.Remove("./test.db")

	if _, err := db.Exec(store.SchemaSQL); err != nil {
		t.Fatal(err)
	}

	req, err := http.NewRequest("GET", "/user/create", nil)
	if err != nil {
		t.Fatal(err)
	}

	rr := httptest.NewRecorder()
	handler := http.HandlerFunc(CreateUserHandler(CreateUserHandlerDeps{DB: db}))

	handler.ServeHTTP(rr, req)
	wantStatus := http.StatusCreated

	if status := rr.Code; status != wantStatus {
		t.Errorf("handler returned wrong status code: got %v want %v",
			status, wantStatus)
	}

	expectedContentType := "application/json"
	if contentType := rr.Header().Get("Content-Type"); contentType != expectedContentType {
		t.Errorf("handler returned unexpected content type: got %v want %v",
			contentType, expectedContentType)
	}

	var response CreateUserResponse

	if err := json.Unmarshal(rr.Body.Bytes(), &response); err != nil {
		t.Errorf("error parsing response body: %v", err)
	}

	if response.Result.Email == "" ||
		response.Result.Password == "" ||
		response.Result.Name == "" ||
		response.Result.Gender == "" ||
		response.Result.Age == 0 {
		t.Errorf("handler returned incomplete user data: %v", response)
	}
}

func TestCalculateAge(t *testing.T) {
	type args struct {
		dob string
		now time.Time
	}
	tests := []struct {
		name    string
		args    args
		want    int
		wantErr bool
	}{
		{
			name: "calculate 2000-01-01 as 24",
			args: args{
				dob: "2000-01-01",
				now: time.Date(2024, 06, 20, 0, 0, 0, 0, time.UTC),
			},
			want:    24,
			wantErr: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := CalculateAge(tt.args.dob, tt.args.now)
			if (err != nil) != tt.wantErr {
				t.Errorf("CalculateAge() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if got != tt.want {
				t.Errorf("CalculateAge() = %v, want %v", got, tt.want)
			}
		})
	}
}
