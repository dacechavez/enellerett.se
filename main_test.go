package main

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func ok(t *testing.T, code int) {
	want := 200

	if code != want {
		t.Errorf("got: [%q], want: [%q]", code, want)
	}
}

func TestGETRootAsCurl(t *testing.T) {
	t.Run("returns helpful hint when no useragent and no path is given", func(t *testing.T) {
		request, _ := http.NewRequest(http.MethodGet, "/", nil)
		response := httptest.NewRecorder()

		handler(response, request)

		got := response.Body.String()
		want := "No input given"

		if !strings.Contains(got, want) {
			t.Errorf("got: [%q]\nit did not contain: [%q]", got, want)
		}

		ok(t, response.Code)

	})
}

func TestGETRootAsBrowser(t *testing.T) {
	t.Run("returns some html when browser requests root path /", func(t *testing.T) {
		request, _ := http.NewRequest(http.MethodGet, "/", nil)
		response := httptest.NewRecorder()

		request.Header.Set("User-Agent", "Mozilla/5.0 (X11; Linux x86_64; rv:134.0) Gecko/20100101 Firefox/134.0")

		handler(response, request)
		got := response.Body.String()
		want := "<body>"

		if !strings.Contains(got, want) {
			t.Errorf("got: [%q]\nit did not contain: [%q]", got, want)
		}

		ok(t, response.Code)
	})
}

func TestGETNounCAPS(t *testing.T) {
	t.Run("query a noun with all CAPS", func(t *testing.T) {
		request, _ := http.NewRequest(http.MethodGet, "/STOL", nil)
		response := httptest.NewRecorder()

		handler(response, request)

		got := response.Body.String()
		want := "En stol"

		if !strings.Contains(got, want) {
			t.Errorf("got: [%q]\nit did not contain: [%q]", got, want)
		}

		ok(t, response.Code)
	})
}

func TestGETSwedishCharacters(t *testing.T) {
	t.Run("handles Swedish characters in path", func(t *testing.T) {
		request, _ := http.NewRequest(http.MethodGet, "/öäå", nil)
		response := httptest.NewRecorder()

		handler(response, request)

		got := response.Body.String()
		want := "öäå"

		if !strings.Contains(got, want) {
			t.Errorf("got: [%q]\nit did not contain: [%q]", got, want)
		}

		ok(t, response.Code)
	})
}

func TestPOSTRequest(t *testing.T) {
	t.Run("handles POST request", func(t *testing.T) {
		body := strings.NewReader("name=test")
		request, _ := http.NewRequest(http.MethodPost, "/stol", body)
		response := httptest.NewRecorder()

		handler(response, request)

		got := response.Body.String()
		want := "En stol"

		if !strings.Contains(got, want) {
			t.Errorf("got: [%q]\nit did not contain: [%q]", got, want)
		}

		ok(t, response.Code)
	})
}

func TestPUTRequest(t *testing.T) {
	t.Run("handles PUT request", func(t *testing.T) {
		body := strings.NewReader("update=data")
		request, _ := http.NewRequest(http.MethodPut, "/stol", body)
		response := httptest.NewRecorder()

		handler(response, request)

		got := response.Body.String()
		want := "En stol"

		if !strings.Contains(got, want) {
			t.Errorf("got: [%q]\nit did not contain: [%q]", got, want)
		}

		ok(t, response.Code)
	})
}

func TestWeirdQueryParams(t *testing.T) {
	t.Run("handles weird query params", func(t *testing.T) {
		request, _ := http.NewRequest(http.MethodGet, "/stol?q=☃️&z=😈", nil)
		response := httptest.NewRecorder()

		handler(response, request)

		got := response.Body.String()
		want := "En stol"

		if !strings.Contains(got, want) {
			t.Errorf("got: [%q]\nit did not contain: [%q]", got, want)
		}

		ok(t, response.Code)
	})
}

func TestPathTraversal(t *testing.T) {
	t.Run("handles common path traversal attacks", func(t *testing.T) {
		paths := []string{
			"/../../../etc/passwd",
			"/..//..//..//..//windows/system32",
			"/../.../.../../",
		}

		for _, path := range paths {
			t.Run(path, func(t *testing.T) {
				request, _ := http.NewRequest(http.MethodGet, path, nil)
				response := httptest.NewRecorder()

				handler(response, request)

				got := response.Body.String()
				want := "Kunde inte hitta"

				if !strings.Contains(got, want) {
					t.Errorf("got: [%q]\nit did not contain: [%q]", got, want)
				}

				ok(t, response.Code)
			})
		}
	})
}

func BenchmarkHandler(b *testing.B) {
	request, _ := http.NewRequest(http.MethodGet, "/", nil)
	response := httptest.NewRecorder()

	for i := 0; i < b.N; i++ {
		handler(response, request)
	}
}

func TestInputOutputTable(t *testing.T) {
	t.Run("loops through input-output table", func(t *testing.T) {
		tests := []struct {
			input  string
			output string
		}{
			{"/stol", "En stol"},
			{"/STOL", "En stol"},
			{"/stoL", "En stol"},
			{"/stol ", "En stol"},
			{"/ stol", "En stol"},
			{"/Stol", "En stol"},
			{"/bok", "En bok"},
			{"/penna", "En penna"},
			{"/öl", "En eller ett öl beroende på kontext"},
			{"/bord", "Ett bord"},
			{"/äpple", "Ett äpple"},
		}

		for _, test := range tests {
			request, _ := http.NewRequest(http.MethodGet, test.input, nil)
			response := httptest.NewRecorder()

			handler(response, request)

			got := response.Body.String()

			if !strings.Contains(got, test.output) {
				t.Errorf("for input [%q], got: [%q]\nit did not contain: [%q]", test.input, got, test.output)
			}

			ok(t, response.Code)
		}
	})
}
